package client

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/mcp/types"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Command: "test",
				Args:    []string{"arg1"},
				Timeout: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing command",
			cfg: Config{
				Args: []string{"arg1"},
			},
			wantErr: true,
		},
		{
			name: "default timeout",
			cfg: Config{
				Command: "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.cfg.Timeout == 0 {
				t.Error("Validate() should set default timeout")
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	err := &Error{
		Op:  "TestOp",
		Err: fmt.Errorf("test error"),
	}

	expected := "mcp client: TestOp: test error"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestError_Unwrap(t *testing.T) {
	innerErr := fmt.Errorf("inner error")
	err := &Error{
		Op:  "TestOp",
		Err: innerErr,
	}

	if err.Unwrap() != innerErr {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), innerErr)
	}
}

func TestTextContent_Helpers(t *testing.T) {
	// Test that we can create and marshal content using helper functions
	text := types.TextContent("test")
	data, err := json.Marshal(text)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if result["type"] != "text" {
		t.Errorf("type = %v, want text", result["type"])
	}
}

func TestJSONRPCRequest_Marshal(t *testing.T) {
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test/method",
		Params:  json.RawMessage(`{"key":"value"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if result["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", result["jsonrpc"])
	}
	if result["method"] != "test/method" {
		t.Errorf("method = %v, want test/method", result["method"])
	}
}

func TestJSONRPCResponse_Unmarshal(t *testing.T) {
	data := []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {"status": "ok"}
	}`)

	var resp jsonrpcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %s, want 2.0", resp.JSONRPC)
	}
	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("Error = %v, want nil", resp.Error)
	}
}

func TestJSONRPCResponse_WithError(t *testing.T) {
	data := []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"error": {"code": -32600, "message": "Invalid Request"}
	}`)

	var resp jsonrpcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("Error.Code = %d, want -32600", resp.Error.Code)
	}
	if resp.Error.Message != "Invalid Request" {
		t.Errorf("Error.Message = %s, want 'Invalid Request'", resp.Error.Message)
	}
}

// Simplified integration test that verifies the client can be created
// Full integration testing would require a real MCP server
func TestStdioClient_Creation(t *testing.T) {
	// This test uses 'echo' which exists on most systems
	// In a real scenario, this would be an actual MCP server
	cfg := Config{
		Command: "echo",
		Args:    []string{"test"},
		Timeout: 1 * time.Second,
	}

	client, err := NewStdioClient(cfg)
	if err != nil {
		t.Fatalf("NewStdioClient() error = %v", err)
	}
	defer client.Close()

	if client.cmd == nil {
		t.Error("client.cmd should not be nil")
	}
	if client.pending == nil {
		t.Error("client.pending should not be nil")
	}
}

func TestStdioClient_InitializeBeforeOtherCalls(t *testing.T) {
	// Create a client (this will fail to initialize with echo, but that's ok for this test)
	cfg := Config{
		Command: "echo",
		Args:    []string{"test"},
		Timeout: 100 * time.Millisecond,
	}

	client, err := NewStdioClient(cfg)
	if err != nil {
		t.Fatalf("NewStdioClient() error = %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Trying to call ListTools without Initialize should return an error
	_, err = client.ListTools(ctx)
	if err == nil {
		t.Error("ListTools() should return error when not initialized")
	}

	// Same for CallTool
	_, err = client.CallTool(ctx, "test", nil)
	if err == nil {
		t.Error("CallTool() should return error when not initialized")
	}
}

func TestSentinelErrors(t *testing.T) {
	// Verify all sentinel errors are defined
	errors := []error{
		ErrSpawnFailed,
		ErrProtocolError,
		ErrVersionMismatch,
		ErrToolFailed,
		ErrTimeout,
		ErrConnectionClosed,
		ErrInvalidResponse,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Sentinel error should not be nil")
		}
		if err.Error() == "" {
			t.Errorf("Sentinel error %v should have non-empty message", err)
		}
	}
}
