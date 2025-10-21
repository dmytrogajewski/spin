package types

import (
	"encoding/json"
	"testing"
)

func TestImplementation_Marshal(t *testing.T) {
	impl := Implementation{
		Name:    "spin",
		Version: "0.1.0",
	}

	data, err := json.Marshal(impl)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	expected := `{"name":"spin","version":"0.1.0"}`
	if string(data) != expected {
		t.Errorf("Marshal() = %s, want %s", data, expected)
	}
}

func TestImplementation_Unmarshal(t *testing.T) {
	data := []byte(`{"name":"test-client","version":"1.2.3"}`)

	var impl Implementation
	if err := json.Unmarshal(data, &impl); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if impl.Name != "test-client" {
		t.Errorf("Name = %s, want test-client", impl.Name)
	}
	if impl.Version != "1.2.3" {
		t.Errorf("Version = %s, want 1.2.3", impl.Version)
	}
}

func TestTool_Marshal(t *testing.T) {
	desc := "Test tool"
	tool := Tool{
		Name:        "test_tool",
		Description: &desc,
		InputSchema: json.RawMessage(`{"type":"object","properties":{"arg":{"type":"string"}}}`),
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal verification error = %v", err)
	}

	if result["name"] != "test_tool" {
		t.Errorf("name = %v, want test_tool", result["name"])
	}
	if result["description"] != "Test tool" {
		t.Errorf("description = %v, want Test tool", result["description"])
	}
}

func TestTool_Unmarshal(t *testing.T) {
	data := []byte(`{
		"name": "read_file",
		"description": "Read a file",
		"inputSchema": {"type":"object","properties":{"path":{"type":"string"}}}
	}`)

	var tool Tool
	if err := json.Unmarshal(data, &tool); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if tool.Name != "read_file" {
		t.Errorf("Name = %s, want read_file", tool.Name)
	}
	if tool.Description == nil || *tool.Description != "Read a file" {
		t.Errorf("Description = %v, want 'Read a file'", tool.Description)
	}
}

func TestResource_Marshal(t *testing.T) {
	desc := "Test resource"
	mime := "text/plain"
	res := Resource{
		URI:         "file:///test.txt",
		Name:        "test.txt",
		Description: &desc,
		MimeType:    &mime,
	}

	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal verification error = %v", err)
	}

	if result["uri"] != "file:///test.txt" {
		t.Errorf("uri = %v, want file:///test.txt", result["uri"])
	}
	if result["name"] != "test.txt" {
		t.Errorf("name = %v, want test.txt", result["name"])
	}
}






func TestInitializeRequest_Marshal(t *testing.T) {
	req := InitializeRequest{
		ProtocolVersion: "2024-11-05",
		Capabilities: ClientCapabilities{
			Tools: &ToolsCapability{ListChanged: true},
		},
		ClientInfo: Implementation{
			Name:    "spin",
			Version: "0.1.0",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal verification error = %v", err)
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion = %v, want 2024-11-05", result["protocolVersion"])
	}
}

func TestCallToolRequest_Marshal(t *testing.T) {
	args := json.RawMessage(`{"path":"test.txt"}`)
	req := CallToolRequest{
		Name:      "read_file",
		Arguments: args,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal verification error = %v", err)
	}

	if result["name"] != "read_file" {
		t.Errorf("name = %v, want read_file", result["name"])
	}

	// Verify arguments are preserved as JSON
	argsMap, ok := result["arguments"].(map[string]interface{})
	if !ok {
		t.Fatalf("arguments type = %T, want map", result["arguments"])
	}
	if argsMap["path"] != "test.txt" {
		t.Errorf("arguments.path = %v, want test.txt", argsMap["path"])
	}
}

func TestCallToolResponse_Unmarshal(t *testing.T) {
	data := []byte(`{
		"content": [
			{"type": "text", "text": "File contents here"}
		],
		"isError": false
	}`)

	var resp CallToolResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.IsError {
		t.Error("IsError = true, want false")
	}
	if len(resp.Content) != 1 {
		t.Fatalf("len(Content) = %d, want 1", len(resp.Content))
	}
	if resp.Content[0].Type != "text" {
		t.Errorf("Content[0].Type = %s, want text", resp.Content[0].Type)
	}
	if resp.Content[0].Text == nil || *resp.Content[0].Text != "File contents here" {
		t.Errorf("Content[0].Text = %v, want 'File contents here'", resp.Content[0].Text)
	}
}

func TestClientCapabilities_EmptyOptional(t *testing.T) {
	caps := ClientCapabilities{}

	data, err := json.Marshal(caps)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Should produce empty object since all fields are omitempty
	expected := `{}`
	if string(data) != expected {
		t.Errorf("Marshal() = %s, want %s", data, expected)
	}
}

func TestServerCapabilities_WithTools(t *testing.T) {
	caps := ServerCapabilities{
		Tools: &ToolsCapability{
			ListChanged: true,
		},
	}

	data, err := json.Marshal(caps)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal verification error = %v", err)
	}

	tools, ok := result["tools"].(map[string]interface{})
	if !ok {
		t.Fatalf("tools type = %T, want map", result["tools"])
	}
	if tools["listChanged"] != true {
		t.Errorf("tools.listChanged = %v, want true", tools["listChanged"])
	}
}

func TestPromptMessage_Marshal(t *testing.T) {
	text := "Hello"
	msg := PromptMessage{
		Role: "user",
		Content: []Content{
			{Type: "text", Text: &text},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal verification error = %v", err)
	}

	if result["role"] != "user" {
		t.Errorf("role = %v, want user", result["role"])
	}
}
