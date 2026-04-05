package blocks

import (
	"strings"
	"testing"
)

// TestRenderToolCompletionStatus tests that Tool completed message only appears when tool has output.
func TestRenderToolCompletionStatus(t *testing.T) {
	t.Parallel()

	renderer := NewRenderer(80)

	// Helper to create a block with metadata.
	createToolBlock := func(toolName, body string) *Block {
		b := NewBlock(BlockTypeTool)

		meta := &ToolMeta{ToolName: toolName}

		err := SetToolMeta(b, meta)
		if err != nil {
			t.Fatalf("SetToolMeta failed: %v", err)
		}

		b.Body = body

		return b
	}

	tests := []struct {
		name           string
		block          *Block
		expectComplete bool
		description    string
	}{
		{
			name:           "tool_without_output",
			block:          createToolBlock("test_tool", ""),
			expectComplete: false,
			description:    "Tool without output should not show completion status",
		},
		{
			name:           "tool_with_output",
			block:          createToolBlock("test_tool", "Tool execution result"),
			expectComplete: true,
			description:    "Tool with output should show completion status",
		},
		{
			name:           "tool_with_empty_string_body",
			block:          createToolBlock("test_tool", ""),
			expectComplete: false,
			description:    "Tool with empty body should not show completion status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status := renderer.RenderCompletionStatus(tt.block)
			hasComplete := strings.Contains(status, "Tool completed")

			if tt.expectComplete && !hasComplete {
				t.Errorf("%s: expected 'Tool completed' in status, got: %q", tt.description, status)
			}

			if !tt.expectComplete && hasComplete {
				t.Errorf("%s: did not expect 'Tool completed' in status, got: %q", tt.description, status)
			}
		})
	}
}

// TestRenderToolBlock_NoDuplicateOnInitialCreate tests that Tool blocks don't show completion on initial creation.
func TestRenderToolBlock_NoDuplicateOnInitialCreate(t *testing.T) {
	t.Parallel()

	renderer := NewRenderer(80)

	// Create a new TOOL block (simulating initial creation).
	block := NewBlock(BlockTypeTool)
	block.Title = "execute_command"

	meta := &ToolMeta{
		ToolName: "execute_command",
	}

	err := SetToolMeta(block, meta)
	if err != nil {
		t.Fatalf("SetToolMeta failed: %v", err)
	}

	// Render the block.
	output, err := renderer.Render(block)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Should NOT contain "Tool completed" since there's no body.
	if strings.Contains(output, "Tool completed") {
		t.Errorf("Initial tool block should not show 'Tool completed', but got:\n%s", output)
	}

	// Now update with actual output.
	block.Body = "Command executed successfully"

	// Render again.
	output, err = renderer.Render(block)
	if err != nil {
		t.Fatalf("Render with body failed: %v", err)
	}

	// Now it SHOULD contain "Tool completed".
	if !strings.Contains(output, "Tool completed") {
		t.Errorf("Tool block with output should show 'Tool completed', but got:\n%s", output)
	}
}
