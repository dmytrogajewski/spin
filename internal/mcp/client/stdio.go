package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dmytrogajewski/spin/internal/mcp/types"
)

// StdioClient implements Client using stdio transport with JSON-RPC 2.0.
type StdioClient struct {
	cfg    Config
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	// JSON-RPC state
	nextID      atomic.Int64
	pendingMu   sync.RWMutex
	pending     map[int64]chan *jsonrpcResponse
	initialized bool

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	closeOnce sync.Once
	err       error
}

// jsonrpcRequest represents a JSON-RPC 2.0 request.
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcResponse represents a JSON-RPC 2.0 response.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

// jsonrpcError represents a JSON-RPC 2.0 error.
type jsonrpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// NewStdioClient creates a new MCP client using stdio transport.
func NewStdioClient(cfg Config) (*StdioClient, error) {
	if err := cfg.Validate(); err != nil {
		return nil, &Error{Op: "NewStdioClient.Validate", Err: err}
	}

	// Create command
	cmd := exec.Command(cfg.Command, cfg.Args...)

	// Set environment variables
	if len(cfg.Env) > 0 {
		env := make([]string, 0, len(cfg.Env))
		for k, v := range cfg.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = append(cmd.Env, env...)
	}

	// Get stdin/stdout pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, &Error{Op: "NewStdioClient.StdinPipe", Err: err}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, &Error{Op: "NewStdioClient.StdoutPipe", Err: err}
	}

	// Start process
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, &Error{Op: "NewStdioClient.Start", Err: ErrSpawnFailed}
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &StdioClient{
		cfg:     cfg,
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		pending: make(map[int64]chan *jsonrpcResponse),
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}

	// Start message pump
	go client.readLoop()

	return client, nil
}

// Initialize initializes the MCP connection.
func (c *StdioClient) Initialize(ctx context.Context, req types.InitializeRequest) (*types.InitializeResponse, error) {
	params, err := json.Marshal(req)
	if err != nil {
		return nil, &Error{Op: "Initialize.Marshal", Err: err}
	}

	result, err := c.call(ctx, "initialize", params)
	if err != nil {
		return nil, &Error{Op: "Initialize.Call", Err: err}
	}

	var resp types.InitializeResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, &Error{Op: "Initialize.Unmarshal", Err: err}
	}

	c.initialized = true
	return &resp, nil
}

// ListTools lists available tools from the server.
func (c *StdioClient) ListTools(ctx context.Context) (*types.ListToolsResponse, error) {
	if !c.initialized {
		return nil, &Error{Op: "ListTools", Err: fmt.Errorf("client not initialized")}
	}

	params, err := json.Marshal(types.ListToolsRequest{})
	if err != nil {
		return nil, &Error{Op: "ListTools.Marshal", Err: err}
	}

	result, err := c.call(ctx, "tools/list", params)
	if err != nil {
		return nil, &Error{Op: "ListTools.Call", Err: err}
	}

	var resp types.ListToolsResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, &Error{Op: "ListTools.Unmarshal", Err: err}
	}

	return &resp, nil
}

// CallTool invokes a tool on the server.
func (c *StdioClient) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*types.CallToolResponse, error) {
	if !c.initialized {
		return nil, &Error{Op: "CallTool", Err: fmt.Errorf("client not initialized")}
	}

	req := types.CallToolRequest{
		Name:      name,
		Arguments: arguments,
	}

	params, err := json.Marshal(req)
	if err != nil {
		return nil, &Error{Op: "CallTool.Marshal", Err: err}
	}

	result, err := c.call(ctx, "tools/call", params)
	if err != nil {
		return nil, &Error{Op: "CallTool.Call", Err: err}
	}

	var resp types.CallToolResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, &Error{Op: "CallTool.Unmarshal", Err: err}
	}

	if resp.IsError {
		return &resp, &Error{Op: "CallTool", Err: ErrToolFailed}
	}

	return &resp, nil
}

// ListResources lists available resources from the server.
func (c *StdioClient) ListResources(ctx context.Context) (*types.ListResourcesResponse, error) {
	if !c.initialized {
		return nil, &Error{Op: "ListResources", Err: fmt.Errorf("client not initialized")}
	}

	params, err := json.Marshal(types.ListResourcesRequest{})
	if err != nil {
		return nil, &Error{Op: "ListResources.Marshal", Err: err}
	}

	result, err := c.call(ctx, "resources/list", params)
	if err != nil {
		return nil, &Error{Op: "ListResources.Call", Err: err}
	}

	var resp types.ListResourcesResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, &Error{Op: "ListResources.Unmarshal", Err: err}
	}

	return &resp, nil
}

// ReadResource reads a resource by its URI.
func (c *StdioClient) ReadResource(ctx context.Context, uri string) (*types.ReadResourceResponse, error) {
	if !c.initialized {
		return nil, &Error{Op: "ReadResource", Err: fmt.Errorf("client not initialized")}
	}

	params, err := json.Marshal(types.ReadResourceRequest{URI: uri})
	if err != nil {
		return nil, &Error{Op: "ReadResource.Marshal", Err: err}
	}

	result, err := c.call(ctx, "resources/read", params)
	if err != nil {
		return nil, &Error{Op: "ReadResource.Call", Err: err}
	}

	var resp types.ReadResourceResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, &Error{Op: "ReadResource.Unmarshal", Err: err}
	}

	return &resp, nil
}

// Close closes the client and cleans up resources.
func (c *StdioClient) Close() error {
	var err error
	c.closeOnce.Do(func() {
		// Cancel context to stop read loop
		c.cancel()

		// Close stdin to signal server to exit
		if c.stdin != nil {
			c.stdin.Close()
		}

		// Wait for process with timeout
		done := make(chan error, 1)
		go func() {
			done <- c.cmd.Wait()
		}()

		select {
		case err = <-done:
		case <-time.After(5 * time.Second):
			// Force kill if not exited
			if c.cmd.Process != nil {
				c.cmd.Process.Kill()
			}
			err = fmt.Errorf("server did not exit gracefully")
		}

		// Wait for read loop to finish
		<-c.done
	})
	return err
}

// call sends a JSON-RPC request and waits for the response.
func (c *StdioClient) call(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	// Generate request ID
	id := c.nextID.Add(1)

	// Create response channel
	respCh := make(chan *jsonrpcResponse, 1)
	c.pendingMu.Lock()
	c.pending[id] = respCh
	c.pendingMu.Unlock()

	// Clean up on exit
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	// Build request
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// Send request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	data = append(data, '\n')
	if _, err := c.stdin.Write(data); err != nil {
		return nil, err
	}

	// Wait for response with timeout
	timeout := c.cfg.Timeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	select {
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, fmt.Errorf("%s: %s", resp.Error.Message, string(resp.Error.Data))
		}
		return resp.Result, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrConnectionClosed
	}
}

// readLoop reads JSON-RPC responses from stdout.
func (c *StdioClient) readLoop() {
	defer close(c.done)

	scanner := bufio.NewScanner(c.stdout)
	for scanner.Scan() {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		var resp jsonrpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			// Invalid JSON-RPC message, skip
			continue
		}

		// Deliver to waiting caller
		c.pendingMu.RLock()
		ch, ok := c.pending[resp.ID]
		c.pendingMu.RUnlock()

		if ok {
			select {
			case ch <- &resp:
			default:
				// Channel full, drop message
			}
		}
	}

	// Scanner finished (EOF or error)
	if err := scanner.Err(); err != nil {
		c.err = err
	}
}
