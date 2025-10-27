package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type mockHandler struct {
	handleFunc func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error)
}

func (m *mockHandler) HandleRequest(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	if m.handleFunc != nil {
		return m.handleFunc(ctx, method, params)
	}
	return json.Marshal(map[string]string{"status": "ok"})
}

func TestServer_Serve_Success(t *testing.T) {
	handler := &mockHandler{
		handleFunc: func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
			return json.Marshal(map[string]string{"result": "success"})
		},
	}

	server := NewServer(handler)

	// Prepare request
	reqID := RequestID{Str: strPtr("1")}
	req := Request{
		JSONRPC: "2.0",
		ID:      &reqID,
		Method:  "test_method",
		Params:  json.RawMessage(`{}`),
	}

	reqData, _ := json.Marshal(req)
	input := bytes.NewReader(append(reqData, '\n'))
	output := &bytes.Buffer{}

	// Serve (will return when input is exhausted)
	ctx := context.Background()
	err := server.Serve(ctx, input, output)
	if err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	// Check response
	var resp Response
	if err := json.NewDecoder(output).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestServer_Serve_Error(t *testing.T) {
	handler := &mockHandler{
		handleFunc: func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
			return nil, NewError(MethodNotFound, "method not found")
		},
	}

	server := NewServer(handler)

	// Prepare request
	reqID := RequestID{Str: strPtr("1")}
	req := Request{
		JSONRPC: "2.0",
		ID:      &reqID,
		Method:  "unknown_method",
		Params:  json.RawMessage(`{}`),
	}

	reqData, _ := json.Marshal(req)
	input := bytes.NewReader(append(reqData, '\n'))
	output := &bytes.Buffer{}

	// Serve
	ctx := context.Background()
	err := server.Serve(ctx, input, output)
	if err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	// Check response
	var resp Response
	if err := json.NewDecoder(output).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error, got nil")
	}
	if resp.Error.Code != MethodNotFound {
		t.Errorf("Expected code %d, got %d", MethodNotFound, resp.Error.Code)
	}
}

func TestServer_Serve_ParseError(t *testing.T) {
	handler := &mockHandler{}
	server := NewServer(handler)

	// Invalid JSON
	input := strings.NewReader("invalid json\n")
	output := &bytes.Buffer{}

	// Serve
	ctx := context.Background()
	err := server.Serve(ctx, input, output)
	if err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	// Check response
	var resp Response
	if err := json.NewDecoder(output).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected parse error, got nil")
	}
	if resp.Error.Code != ParseError {
		t.Errorf("Expected code %d, got %d", ParseError, resp.Error.Code)
	}
}

func TestServer_Serve_Notification(t *testing.T) {
	called := false
	handler := &mockHandler{
		handleFunc: func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
			called = true
			return nil, nil
		},
	}

	server := NewServer(handler)

	// Prepare notification (no ID)
	notif := Notification{
		JSONRPC: "2.0",
		Method:  "notification",
		Params:  json.RawMessage(`{}`),
	}

	notifData, _ := json.Marshal(notif)
	input := bytes.NewReader(append(notifData, '\n'))
	output := &bytes.Buffer{}

	// Serve
	ctx := context.Background()
	err := server.Serve(ctx, input, output)
	if err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	// Handler should have been called
	if !called {
		t.Error("Handler should have been called for notification")
	}

	// No response should be sent for notifications
	if output.Len() > 0 {
		t.Error("No response should be sent for notifications")
	}
}

func TestServer_Context_Cancellation(t *testing.T) {
	handler := &mockHandler{
		handleFunc: func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
			// Simulate slow operation
			time.Sleep(100 * time.Millisecond)
			return nil, nil
		},
	}
	server := NewServer(handler)

	// Create a pipe to simulate blocking read
	r, w := bytes.NewReader(nil), &bytes.Buffer{}

	// Prepare one request
	reqID := RequestID{Str: strPtr("1")}
	req := Request{
		JSONRPC: "2.0",
		ID:      &reqID,
		Method:  "test",
		Params:  json.RawMessage(`{}`),
	}
	reqData, _ := json.Marshal(req)
	r = bytes.NewReader(append(reqData, '\n'))

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- server.Serve(ctx, r, w)
	}()

	// Cancel after a short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		// Context cancellation may return nil if request finished first
		// or context.Canceled if cancellation was processed
		if err != nil && err != context.Canceled {
			t.Errorf("Expected nil or context.Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Server did not stop after context cancellation")
	}
}

// TestServer_JSONRawMessageResult verifies that the server uses
// json.RawMessage results directly without re-marshaling
func TestServer_JSONRawMessageResult(t *testing.T) {
	expectedResult := json.RawMessage(`{"custom":"data","number":42}`)

	handler := &mockHandler{
		handleFunc: func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
			// Return pre-marshaled JSON
			return expectedResult, nil
		},
	}

	server := NewServer(handler)

	// Prepare request
	reqID := RequestID{Str: strPtr("1")}
	req := Request{
		JSONRPC: "2.0",
		ID:      &reqID,
		Method:  "test_method",
		Params:  json.RawMessage(`{}`),
	}

	reqData, _ := json.Marshal(req)
	input := bytes.NewReader(append(reqData, '\n'))
	output := &bytes.Buffer{}

	// Serve
	ctx := context.Background()
	err := server.Serve(ctx, input, output)
	if err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	// Check response
	var resp Response
	if err := json.NewDecoder(output).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}

	// Verify result is exactly what handler returned (not re-marshaled)
	if string(resp.Result) != string(expectedResult) {
		t.Errorf("Expected result %s, got %s", string(expectedResult), string(resp.Result))
	}

	// Verify it's valid JSON that can be decoded
	var decoded map[string]interface{}
	if err := json.Unmarshal(resp.Result, &decoded); err != nil {
		t.Errorf("Result is not valid JSON: %v", err)
	}

	if decoded["custom"] != "data" {
		t.Errorf("Expected custom=data, got %v", decoded["custom"])
	}
	if decoded["number"] != float64(42) {
		t.Errorf("Expected number=42, got %v", decoded["number"])
	}
}
