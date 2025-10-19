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

func TestRenderer_Render_LongText(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 20, 24)

	// Text that's too long for the terminal width
	longText := "This is a very long status text that will be truncated"
	err := renderer.Render(longText)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	// Should contain truncation indicator
	if !containsRenderer(output, "...") {
		t.Error("Expected truncation indicator '...' for long text")
	}
}

func TestRenderer_Render_WithPadding(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 80, 24)

	// Short text that will be centered with padding
	shortText := "Status"
	err := renderer.Render(shortText)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	// Should contain the status text
	if !containsRenderer(output, "Status") {
		t.Error("Expected 'Status' in output")
	}
	// Should have cursor positioning and formatting
	if !containsRenderer(output, "\x1b[37;1m") {
		t.Error("Expected bright white formatting")
	}
}

func TestRenderer_Render_WithANSICodes(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 80, 24)

	// Status text with ANSI codes that should be stripped
	statusWithANSI := "\x1b[31mRed Status\x1b[0m"
	err := renderer.Render(statusWithANSI)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	// The ANSI codes should be stripped, but "Red Status" should still be there
	if !containsRenderer(output, "Red Status") {
		t.Error("Expected 'Red Status' text in output after stripping ANSI")
	}
}

func TestRenderer_Render_EmptyText(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 80, 24)

	// Empty status text
	err := renderer.Render("")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	// Should still have cursor save/restore and positioning
	if !containsRenderer(output, "\x1b7") {
		t.Error("Expected cursor save sequence")
	}
	if !containsRenderer(output, "\x1b8") {
		t.Error("Expected cursor restore sequence")
	}
}

func TestRenderer_MoveToPrompt(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 80, 24)

	err := renderer.MoveToPrompt()
	if err != nil {
		t.Fatalf("MoveToPrompt() error = %v", err)
	}

	output := buf.String()
	// Should move to line 24 (height)
	if !containsRenderer(output, "\x1b[24;1H") {
		t.Errorf("Expected cursor positioning to prompt line, got: %q", output)
	}
}

func TestRenderer_MoveToScrollRegion(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 80, 24)

	err := renderer.MoveToScrollRegion()
	if err != nil {
		t.Fatalf("MoveToScrollRegion() error = %v", err)
	}

	output := buf.String()
	// Should move to line 22 (height - 2)
	if !containsRenderer(output, "\x1b[22;1H") {
		t.Errorf("Expected cursor positioning to scroll region, got: %q", output)
	}
}

func TestRenderer_MoveToScrollRegion_SmallTerminal(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 80, 2)

	err := renderer.MoveToScrollRegion()
	if err != nil {
		t.Fatalf("MoveToScrollRegion() error = %v", err)
	}

	// Should not output anything for small terminal
	output := buf.String()
	if output != "" {
		t.Errorf("Expected no output for small terminal, got: %q", output)
	}
}

func TestRenderer_RenderMetrics(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 80, 24)

	metrics := &Metrics{
		Provider:       "openai",
		Model:          "gpt-4",
		TokenCount:     1000,
		MaxTokens:      8000,
		TokensPerSec:   50.5,
		AgentState:     "thinking",
		ConversationID: "abc123def456",
	}

	err := renderer.RenderMetrics(metrics)
	if err != nil {
		t.Fatalf("RenderMetrics() error = %v", err)
	}

	output := buf.String()
	// Check for cursor save/restore
	if !containsRenderer(output, "\x1b7") {
		t.Error("Expected cursor save sequence")
	}
	if !containsRenderer(output, "\x1b8") {
		t.Error("Expected cursor restore sequence")
	}
	// Check for status line positioning (line 23 = 24-1)
	if !containsRenderer(output, "\x1b[23;1H") {
		t.Error("Expected cursor positioning to status line")
	}
	// Check for metrics content
	if !containsRenderer(output, "openai/gpt-4") {
		t.Error("Expected provider/model in output")
	}
}

func TestRenderer_RenderMetrics_SmallTerminal(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 5, 2)

	metrics := &Metrics{
		Provider: "openai",
		Model:    "gpt-4",
	}

	err := renderer.RenderMetrics(metrics)
	if err != nil {
		t.Fatalf("RenderMetrics() error = %v", err)
	}

	// Should not render for small terminal
	output := buf.String()
	if output != "" {
		t.Errorf("Expected no output for small terminal, got: %q", output)
	}
}

func TestRenderer_BuildMetricsLine(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewRenderer(&buf, 120, 24)

	tests := []struct {
		name     string
		metrics  *Metrics
		contains []string
	}{
		{
			name: "full metrics",
			metrics: &Metrics{
				Provider:       "openai",
				Model:          "gpt-4",
				TokenCount:     1000,
				MaxTokens:      8000,
				TokensPerSec:   50.5,
				AgentState:     "thinking",
				ConversationID: "abc123def456",
			},
			contains: []string{"[●]", "openai/gpt-4", "50 tok/s", "conv:abc123de", "?:help"},
		},
		{
			name: "minimal metrics",
			metrics: &Metrics{
				Provider: "",
				Model:    "",
			},
			contains: []string{"[○]", "N/A", "?:help"},
		},
		{
			name: "high token usage",
			metrics: &Metrics{
				TokenCount: 8500,
				MaxTokens:  10000,
			},
			contains: []string{"[○]", "85%"},
		},
		{
			name: "with agent state",
			metrics: &Metrics{
				AgentState: "executing",
			},
			contains: []string{"[●]", "executing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := renderer.buildMetricsLine(tt.metrics)
			for _, substr := range tt.contains {
				if !containsRenderer(line, substr) {
					t.Errorf("Expected %q in metrics line, got: %q", substr, line)
				}
			}
		})
	}
}

func TestRenderer_BuildMetricsLine_Truncation(t *testing.T) {
	var buf bytes.Buffer
	// Small width to test truncation
	renderer := NewRenderer(&buf, 20, 24)

	metrics := &Metrics{
		Provider:       "openai",
		Model:          "gpt-4-turbo-very-long-name",
		TokenCount:     1000,
		MaxTokens:      8000,
		TokensPerSec:   50.5,
		AgentState:     "thinking",
		ConversationID: "abc123def456",
	}

	line := renderer.buildMetricsLine(metrics)
	// Should be truncated to fit width
	if len(line) > renderer.width-2 {
		t.Errorf("Expected line to be truncated to %d chars, got %d", renderer.width-2, len(line))
	}
	if !containsRenderer(line, "...") {
		t.Error("Expected truncation indicator '...'")
	}
}

func TestRenderer_StripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ANSI codes",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "color code",
			input:    "\x1b[31mred text\x1b[0m",
			expected: "red text",
		},
		{
			name:     "bold text",
			input:    "\x1b[1mbold\x1b[0m",
			expected: "bold",
		},
		{
			name:     "multiple codes",
			input:    "\x1b[1m\x1b[31mbold red\x1b[0m\x1b[0m",
			expected: "bold red",
		},
		{
			name:     "cursor positioning",
			input:    "\x1b[10;5Htext",
			expected: "text",
		},
		{
			name:     "mixed content",
			input:    "normal \x1b[32mgreen\x1b[0m normal",
			expected: "normal green normal",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only ANSI codes",
			input:    "\x1b[0m\x1b[1m\x1b[2K",
			expected: "",
		},
		{
			name:     "uppercase letter terminator",
			input:    "\x1b[2Ktext",
			expected: "text",
		},
		{
			name:     "lowercase letter terminator",
			input:    "\x1b[5mtext",
			expected: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("stripANSI(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
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
