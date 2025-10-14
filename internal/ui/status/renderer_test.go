package status

import (
	"bytes"
	"testing"
)

func TestRenderer_Render(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 80, 24)

	// Test rendering status text
	err := renderer.Render("Test Status")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Check that ANSI sequences were written
	output := buf.String()
	if !containsRenderer(output, "\x1b[22;1H") { // Position at line 22 (24-2)
		t.Error("Expected cursor positioning to status line")
	}
	if !containsRenderer(output, "\x1b[2K") { // Clear line
		t.Error("Expected line clear sequence")
	}
	if !containsRenderer(output, "Test Status") {
		t.Error("Expected status text in output")
	}
}

func TestRenderer_Clear(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 80, 24)

	err := renderer.Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	output := buf.String()
	if !containsRenderer(output, "\x1b[22;1H") { // Position at status line
		t.Error("Expected cursor positioning")
	}
	if !containsRenderer(output, "\x1b[2K") { // Clear line
		t.Error("Expected line clear sequence")
	}
}

func TestRenderer_SetSize(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 80, 24)

	// Change size
	renderer.SetSize(120, 30)

	// Render should now position at line 28 (30-2)
	err := renderer.Render("Test")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	if !containsRenderer(output, "\x1b[28;1H") { // Position at line 28
		t.Error("Expected cursor positioning to new status line")
	}
}

func TestRenderer_SmallTerminal(t *testing.T) {
	var buf bytes.Buffer
	// Terminal too small (height < 3)
	renderer := NewRenderer(&buf, 80, 2)

	err := renderer.Render("Test")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Should not render anything for small terminal
	output := buf.String()
	if output != "" {
		t.Errorf("Expected empty output for small terminal, got: %q", output)
	}
}

// Helper function
func containsRenderer(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findIndexRenderer(s, substr) >= 0
}

func findIndexRenderer(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
