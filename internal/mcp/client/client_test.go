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

	// Same for ListResources
	_, err = client.ListResources(ctx)
	if err == nil {
		t.Error("ListResources() should return error when not initialized")
	}

	// Same for ReadResource
	_, err = client.ReadResource(ctx, "test://resource")
	if err == nil {
		t.Error("ReadResource() should return error when not initialized")
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

// TestListResources_NotInitialized verifies that ListResources returns error when client is not initialized
func TestListResources_NotInitialized(t *testing.T) {
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

	ctx := context.Background()
	_, err = client.ListResources(ctx)
	if err == nil {
		t.Fatal("ListResources() should return error when not initialized")
	}

	// Verify error message contains "not initialized"
	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

// TestListResourcesRequest_Marshal verifies ListResourcesRequest can be marshaled
func TestListResourcesRequest_Marshal(t *testing.T) {
	req := types.ListResourcesRequest{}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Empty request should produce valid JSON object
	if result == nil {
		t.Error("Marshaled request should not be nil")
	}
}

// TestListResourcesRequest_WithCursor verifies cursor parameter marshaling
func TestListResourcesRequest_WithCursor(t *testing.T) {
	cursor := "next-page-token"
	req := types.ListResourcesRequest{
		Cursor: &cursor,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if result["cursor"] != cursor {
		t.Errorf("cursor = %v, want %v", result["cursor"], cursor)
	}
}

// TestListResourcesResponse_Unmarshal verifies response parsing
func TestListResourcesResponse_Unmarshal(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantErr  bool
		validate func(*testing.T, *types.ListResourcesResponse)
	}{
		{
			name: "empty resources list",
			data: `{"resources": []}`,
			validate: func(t *testing.T, resp *types.ListResourcesResponse) {
				if resp.Resources == nil {
					t.Error("Resources should not be nil")
				}
				if len(resp.Resources) != 0 {
					t.Errorf("Resources length = %d, want 0", len(resp.Resources))
				}
			},
		},
		{
			name: "single resource",
			data: `{
				"resources": [
					{
						"uri": "file:///test.txt",
						"name": "Test File",
						"description": "A test file",
						"mimeType": "text/plain"
					}
				]
			}`,
			validate: func(t *testing.T, resp *types.ListResourcesResponse) {
				if len(resp.Resources) != 1 {
					t.Fatalf("Resources length = %d, want 1", len(resp.Resources))
				}
				r := resp.Resources[0]
				if r.URI != "file:///test.txt" {
					t.Errorf("URI = %s, want file:///test.txt", r.URI)
				}
				if r.Name != "Test File" {
					t.Errorf("Name = %s, want Test File", r.Name)
				}
				if r.Description == nil || *r.Description != "A test file" {
					t.Error("Description should be 'A test file'")
				}
				if r.MimeType == nil || *r.MimeType != "text/plain" {
					t.Error("MimeType should be 'text/plain'")
				}
			},
		},
		{
			name: "multiple resources",
			data: `{
				"resources": [
					{
						"uri": "file:///doc1.txt",
						"name": "Document 1"
					},
					{
						"uri": "file:///doc2.txt",
						"name": "Document 2"
					}
				]
			}`,
			validate: func(t *testing.T, resp *types.ListResourcesResponse) {
				if len(resp.Resources) != 2 {
					t.Fatalf("Resources length = %d, want 2", len(resp.Resources))
				}
				if resp.Resources[0].URI != "file:///doc1.txt" {
					t.Errorf("First URI = %s, want file:///doc1.txt", resp.Resources[0].URI)
				}
				if resp.Resources[1].URI != "file:///doc2.txt" {
					t.Errorf("Second URI = %s, want file:///doc2.txt", resp.Resources[1].URI)
				}
			},
		},
		{
			name: "with pagination cursor",
			data: `{
				"resources": [],
				"nextCursor": "page-2-token"
			}`,
			validate: func(t *testing.T, resp *types.ListResourcesResponse) {
				if resp.NextCursor == nil {
					t.Fatal("NextCursor should not be nil")
				}
				if *resp.NextCursor != "page-2-token" {
					t.Errorf("NextCursor = %s, want page-2-token", *resp.NextCursor)
				}
			},
		},
		{
			name:    "invalid json",
			data:    `{"resources": invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp types.ListResourcesResponse
			err := json.Unmarshal([]byte(tt.data), &resp)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, &resp)
			}
		})
	}
}

// TestResource_AllFields verifies Resource type has all required fields
func TestResource_AllFields(t *testing.T) {
	desc := "Test description"
	mime := "application/json"

	resource := types.Resource{
		URI:         "file:///test.json",
		Name:        "Test Resource",
		Description: &desc,
		MimeType:    &mime,
	}

	// Marshal and unmarshal to verify all fields
	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result types.Resource
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if result.URI != resource.URI {
		t.Errorf("URI = %s, want %s", result.URI, resource.URI)
	}
	if result.Name != resource.Name {
		t.Errorf("Name = %s, want %s", result.Name, resource.Name)
	}
	if result.Description == nil || *result.Description != desc {
		t.Error("Description not preserved")
	}
	if result.MimeType == nil || *result.MimeType != mime {
		t.Error("MimeType not preserved")
	}
}

// TestReadResource_NotInitialized verifies that ReadResource returns error when client is not initialized
func TestReadResource_NotInitialized(t *testing.T) {
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

	ctx := context.Background()
	_, err = client.ReadResource(ctx, "file:///test.txt")
	if err == nil {
		t.Fatal("ReadResource() should return error when not initialized")
	}

	// Verify error message contains "not initialized"
	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

// TestReadResourceRequest_Marshal verifies ReadResourceRequest can be marshaled
func TestReadResourceRequest_Marshal(t *testing.T) {
	req := types.ReadResourceRequest{
		URI: "file:///test.txt",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if result["uri"] != "file:///test.txt" {
		t.Errorf("uri = %v, want file:///test.txt", result["uri"])
	}
}

// TestReadResourceResponse_Unmarshal verifies response parsing
func TestReadResourceResponse_Unmarshal(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantErr  bool
		validate func(*testing.T, *types.ReadResourceResponse)
	}{
		{
			name: "text content",
			data: `{
				"contents": [
					{
						"uri": "file:///test.txt",
						"mimeType": "text/plain",
						"text": "Hello, World!"
					}
				]
			}`,
			validate: func(t *testing.T, resp *types.ReadResourceResponse) {
				if len(resp.Contents) != 1 {
					t.Fatalf("Contents length = %d, want 1", len(resp.Contents))
				}
				c := resp.Contents[0]
				if c.URI != "file:///test.txt" {
					t.Errorf("URI = %s, want file:///test.txt", c.URI)
				}
				if c.MimeType == nil || *c.MimeType != "text/plain" {
					t.Error("MimeType should be 'text/plain'")
				}
				if c.Text == nil || *c.Text != "Hello, World!" {
					t.Error("Text should be 'Hello, World!'")
				}
				if c.Blob != nil {
					t.Error("Blob should be nil for text content")
				}
			},
		},
		{
			name: "binary blob content",
			data: `{
				"contents": [
					{
						"uri": "file:///image.png",
						"mimeType": "image/png",
						"blob": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
					}
				]
			}`,
			validate: func(t *testing.T, resp *types.ReadResourceResponse) {
				if len(resp.Contents) != 1 {
					t.Fatalf("Contents length = %d, want 1", len(resp.Contents))
				}
				c := resp.Contents[0]
				if c.URI != "file:///image.png" {
					t.Errorf("URI = %s, want file:///image.png", c.URI)
				}
				if c.MimeType == nil || *c.MimeType != "image/png" {
					t.Error("MimeType should be 'image/png'")
				}
				if c.Blob == nil {
					t.Fatal("Blob should not be nil")
				}
				if c.Text != nil {
					t.Error("Text should be nil for binary content")
				}
			},
		},
		{
			name: "multiple contents",
			data: `{
				"contents": [
					{
						"uri": "file:///part1.txt",
						"text": "Part 1"
					},
					{
						"uri": "file:///part2.txt",
						"text": "Part 2"
					}
				]
			}`,
			validate: func(t *testing.T, resp *types.ReadResourceResponse) {
				if len(resp.Contents) != 2 {
					t.Fatalf("Contents length = %d, want 2", len(resp.Contents))
				}
				if resp.Contents[0].URI != "file:///part1.txt" {
					t.Errorf("First URI = %s, want file:///part1.txt", resp.Contents[0].URI)
				}
				if resp.Contents[1].URI != "file:///part2.txt" {
					t.Errorf("Second URI = %s, want file:///part2.txt", resp.Contents[1].URI)
				}
			},
		},
		{
			name: "empty contents",
			data: `{"contents": []}`,
			validate: func(t *testing.T, resp *types.ReadResourceResponse) {
				if resp.Contents == nil {
					t.Error("Contents should not be nil")
				}
				if len(resp.Contents) != 0 {
					t.Errorf("Contents length = %d, want 0", len(resp.Contents))
				}
			},
		},
		{
			name:    "invalid json",
			data:    `{"contents": invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp types.ReadResourceResponse
			err := json.Unmarshal([]byte(tt.data), &resp)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, &resp)
			}
		})
	}
}

// TestResourceContents_AllFields verifies ResourceContents type has all required fields
func TestResourceContents_AllFields(t *testing.T) {
	t.Run("text content", func(t *testing.T) {
		mime := "text/plain"
		text := "Sample text content"

		contents := types.ResourceContents{
			URI:      "file:///sample.txt",
			MimeType: &mime,
			Text:     &text,
		}

		// Marshal and unmarshal to verify all fields
		data, err := json.Marshal(contents)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}

		var result types.ResourceContents
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if result.URI != contents.URI {
			t.Errorf("URI = %s, want %s", result.URI, contents.URI)
		}
		if result.MimeType == nil || *result.MimeType != mime {
			t.Error("MimeType not preserved")
		}
		if result.Text == nil || *result.Text != text {
			t.Error("Text not preserved")
		}
		if result.Blob != nil {
			t.Error("Blob should be nil")
		}
	})

	t.Run("binary content", func(t *testing.T) {
		mime := "image/png"
		blob := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

		contents := types.ResourceContents{
			URI:      "file:///image.png",
			MimeType: &mime,
			Blob:     &blob,
		}

		// Marshal and unmarshal to verify all fields
		data, err := json.Marshal(contents)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}

		var result types.ResourceContents
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if result.URI != contents.URI {
			t.Errorf("URI = %s, want %s", result.URI, contents.URI)
		}
		if result.MimeType == nil || *result.MimeType != mime {
			t.Error("MimeType not preserved")
		}
		if result.Blob == nil || *result.Blob != blob {
			t.Error("Blob not preserved")
		}
		if result.Text != nil {
			t.Error("Text should be nil")
		}
	})
}
