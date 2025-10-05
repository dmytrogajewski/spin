package protocol

import (
	"encoding/json"
	"testing"
)

func TestLocalShellStatus_Pending(t *testing.T) {
	status := NewPendingStatus()

	if status.Pending == nil {
		t.Error("Pending should not be nil")
	}
	if status.Running != nil || status.Completed != nil || status.Failed != nil {
		t.Error("Other status fields should be nil")
	}
}

func TestLocalShellStatus_Running(t *testing.T) {
	status := NewRunningStatus()

	if status.Running == nil {
		t.Error("Running should not be nil")
	}
	if status.Pending != nil || status.Completed != nil || status.Failed != nil {
		t.Error("Other status fields should be nil")
	}
}

func TestLocalShellStatus_Completed(t *testing.T) {
	status := NewCompletedStatus(0, "output")

	if status.Completed == nil {
		t.Error("Completed should not be nil")
	}
	if status.Pending != nil || status.Running != nil || status.Failed != nil {
		t.Error("Other status fields should be nil")
	}
	if status.Completed.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", status.Completed.ExitCode)
	}
	if status.Completed.Output != "output" {
		t.Errorf("Expected output 'output', got '%s'", status.Completed.Output)
	}
}

func TestLocalShellStatus_Failed(t *testing.T) {
	status := NewFailedStatus("error message")

	if status.Failed == nil {
		t.Error("Failed should not be nil")
	}
	if status.Pending != nil || status.Running != nil || status.Completed != nil {
		t.Error("Other status fields should be nil")
	}
	if status.Failed.Error != "error message" {
		t.Errorf("Expected error 'error message', got '%s'", status.Failed.Error)
	}
}

func TestContentItem_Text(t *testing.T) {
	item := ContentItem{
		Type: "text",
		Text: &TextContent{Text: "Hello"},
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ContentItem
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != "text" {
		t.Errorf("Expected type 'text', got '%s'", decoded.Type)
	}
	if decoded.Text == nil || decoded.Text.Text != "Hello" {
		t.Error("Text content mismatch")
	}
}

func TestContentItem_Image(t *testing.T) {
	url := "https://example.com/image.png"
	item := ContentItem{
		Type: "image",
		Image: &ImageContent{
			URL:      &url,
			MimeType: "image/png",
		},
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ContentItem
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != "image" {
		t.Errorf("Expected type 'image', got '%s'", decoded.Type)
	}
	if decoded.Image == nil || decoded.Image.URL == nil || *decoded.Image.URL != url {
		t.Error("Image content mismatch")
	}
}

func TestContentItem_FilePointer(t *testing.T) {
	mimeType := "text/plain"
	item := ContentItem{
		Type: "file_pointer",
		FilePointer: &FilePointerContent{
			Path:     "/path/to/file.txt",
			MimeType: &mimeType,
		},
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ContentItem
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != "file_pointer" {
		t.Errorf("Expected type 'file_pointer', got '%s'", decoded.Type)
	}
	if decoded.FilePointer == nil || decoded.FilePointer.Path != "/path/to/file.txt" {
		t.Error("FilePointer content mismatch")
	}
}

func TestToolCallItem(t *testing.T) {
	item := ToolCallItem{
		ID:        "call-123",
		Name:      "read_file",
		Arguments: `{"path":"test.go"}`,
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ToolCallItem
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != item.ID {
		t.Errorf("Expected ID '%s', got '%s'", item.ID, decoded.ID)
	}
	if decoded.Name != item.Name {
		t.Errorf("Expected name '%s', got '%s'", item.Name, decoded.Name)
	}
}

func TestReasoningItem(t *testing.T) {
	item := ReasoningItem{
		Reasoning: "Let me think about this...",
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ReasoningItem
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Reasoning != item.Reasoning {
		t.Errorf("Expected reasoning '%s', got '%s'", item.Reasoning, decoded.Reasoning)
	}
}
