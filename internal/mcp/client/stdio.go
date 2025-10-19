package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/mcp/types"
)

// StdioClient implements MCP client using stdio transport.
type StdioClient struct {
	config      Config
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	stderr      io.ReadCloser
	scanner     *bufio.Scanner
	mu          sync.RWMutex
	closed      bool
	closeOnce   sync.Once
	requestID   int64
	responses   map[string]chan json.RawMessage
	responsesMu sync.RWMutex
}

// NewStdioClient creates a new stdio-based MCP client.
func NewStdioClient(config Config) *StdioClient {
	return &StdioClient{
		config:    config,
		responses: make(map[string]chan json.RawMessage),
	}
}

// Initialize establishes the MCP connection and negotiates capabilities.
func (c *StdioClient) Initialize(ctx context.Context, req types.InitializeRequest) (*types.InitializeResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	// Start the server process
	if err := c.startProcess(); err != nil {
		return nil, fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Send initialize request
	resp, err := c.sendRequest(ctx, "initialize", req)
	if err != nil {
		c.closeProcess()
		return nil, fmt.Errorf("initialize request failed: %w", err)
	}

	// Parse response
	var initResp types.InitializeResponse
	if err := json.Unmarshal(resp, &initResp); err != nil {
		c.closeProcess()
		return nil, fmt.Errorf("failed to parse initialize response: %w", err)
	}

	return &initResp, nil
}

// ListTools retrieves available tools from the server.
func (c *StdioClient) ListTools(ctx context.Context) (*types.ListToolsResponse, error) {
	resp, err := c.sendRequest(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("list tools request failed: %w", err)
	}

	var toolsResp types.ListToolsResponse
	if err := json.Unmarshal(resp, &toolsResp); err != nil {
		return nil, fmt.Errorf("failed to parse list tools response: %w", err)
	}

	return &toolsResp, nil
}

// CallTool invokes a tool on the server.
func (c *StdioClient) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*types.CallToolResponse, error) {
	req := types.CallToolRequest{
		Name:      name,
		Arguments: arguments,
	}

	resp, err := c.sendRequest(ctx, "tools/call", req)
	if err != nil {
		return nil, fmt.Errorf("call tool request failed: %w", err)
	}

	var callResp types.CallToolResponse
	if err := json.Unmarshal(resp, &callResp); err != nil {
		return nil, fmt.Errorf("failed to parse call tool response: %w", err)
	}

	return &callResp, nil
}

// ListResources retrieves available resources from the server.
func (c *StdioClient) ListResources(ctx context.Context) (*types.ListResourcesResponse, error) {
	resp, err := c.sendRequest(ctx, "resources/list", nil)
	if err != nil {
		return nil, fmt.Errorf("list resources request failed: %w", err)
	}

	var resourcesResp types.ListResourcesResponse
	if err := json.Unmarshal(resp, &resourcesResp); err != nil {
		return nil, fmt.Errorf("failed to parse list resources response: %w", err)
	}

	return &resourcesResp, nil
}

// ReadResource reads a specific resource.
func (c *StdioClient) ReadResource(ctx context.Context, uri string) (*types.ReadResourceResponse, error) {
	req := types.ReadResourceRequest{
		URI: uri,
	}

	resp, err := c.sendRequest(ctx, "resources/read", req)
	if err != nil {
		return nil, fmt.Errorf("read resource request failed: %w", err)
	}

	var readResp types.ReadResourceResponse
	if err := json.Unmarshal(resp, &readResp); err != nil {
		return nil, fmt.Errorf("failed to parse read resource response: %w", err)
	}

	return &readResp, nil
}

// Close closes the connection and cleans up resources.
func (c *StdioClient) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()

		if closeErr := c.closeProcess(); closeErr != nil {
			err = closeErr
		}
	})
	return err
}

// startProcess starts the MCP server process.
func (c *StdioClient) startProcess() error {
	// Create command
	c.cmd = exec.Command(c.config.Command, c.config.Args...)
	c.cmd.Env = os.Environ()

	// Add custom environment variables
	for key, value := range c.config.Env {
		c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set up stdio
	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		c.stdin.Close()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		c.stdin.Close()
		c.stdout.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := c.cmd.Start(); err != nil {
		c.stdin.Close()
		c.stdout.Close()
		c.stderr.Close()
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Set up scanner for reading responses
	c.scanner = bufio.NewScanner(c.stdout)

	// Start response reader goroutine
	go c.readResponses()

	return nil
}

// closeProcess closes the MCP server process.
func (c *StdioClient) closeProcess() error {
	var err error

	if c.stdin != nil {
		if closeErr := c.stdin.Close(); closeErr != nil {
			err = closeErr
		}
	}

	if c.stdout != nil {
		if closeErr := c.stdout.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
	}

	if c.stderr != nil {
		if closeErr := c.stderr.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
	}

	if c.cmd != nil && c.cmd.Process != nil {
		// Give the process a chance to exit gracefully
		done := make(chan error, 1)
		go func() {
			done <- c.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited
		case <-time.After(5 * time.Second):
			// Force kill if it doesn't exit
			if killErr := c.cmd.Process.Kill(); killErr != nil {
				if err == nil {
					err = killErr
				}
			}
		}
	}

	return err
}

// sendRequest sends a JSON-RPC request and waits for response.
func (c *StdioClient) sendRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Generate request ID
	c.mu.Lock()
	c.requestID++
	id := fmt.Sprintf("%d", c.requestID)
	c.mu.Unlock()

	// Create JSON-RPC request
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}

	if params != nil {
		req["params"] = params
	}

	// Marshal request
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create response channel
	c.responsesMu.Lock()
	responseChan := make(chan json.RawMessage, 1)
	c.responses[id] = responseChan
	c.responsesMu.Unlock()

	// Clean up response channel when done
	defer func() {
		c.responsesMu.Lock()
		delete(c.responses, id)
		c.responsesMu.Unlock()
	}()

	// Send request
	if _, err := c.stdin.Write(append(reqData, '\n')); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response
	select {
	case resp := <-responseChan:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(c.config.Timeout):
		return nil, fmt.Errorf("request timeout")
	}
}

// readResponses reads JSON-RPC responses from stdout.
func (c *StdioClient) readResponses() {
	for c.scanner.Scan() {
		line := c.scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON-RPC response
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue // Skip malformed responses
		}

		// Check if this is a response (has id)
		if id, ok := resp["id"].(string); ok {
			c.responsesMu.RLock()
			responseChan, exists := c.responses[id]
			c.responsesMu.RUnlock()

			if exists {
				// Send result or error
				if result, ok := resp["result"]; ok {
					resultData, _ := json.Marshal(result)
					responseChan <- json.RawMessage(resultData)
				} else if err, ok := resp["error"]; ok {
					errData, _ := json.Marshal(err)
					responseChan <- json.RawMessage(errData)
				}
			}
		}
	}

	// Scanner stopped, close all pending responses
	c.responsesMu.Lock()
	for _, ch := range c.responses {
		close(ch)
	}
	c.responses = make(map[string]chan json.RawMessage)
	c.responsesMu.Unlock()
}
