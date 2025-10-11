package core

import (
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

// TestToolCallFormatting_ListDirectory tests block rendering for list_directory
func TestToolCallFormatting_ListDirectory(t *testing.T) {
	// Create an EXECUTE block (what TUIMapper creates for list_directory)
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.ID = "tool_123"

	// Set metadata (simulating what createExecuteBlock does)
	meta := &blocks.ExecuteMeta{
		Command: "ls /home/dmytrogajewski",
		CWD:     ".",
		Impact:  "low",
	}
	if err := blocks.SetExecuteMeta(block, meta); err != nil {
		t.Fatalf("SetExecuteMeta failed: %v", err)
	}

	// Step 1: Render the block BEFORE completion
	renderer := blocks.NewRenderer(80)
	outputBefore, err := renderer.Render(block)
	if err != nil {
		t.Fatalf("Render(block before) failed: %v", err)
	}

	t.Logf("Block BEFORE completion:\n%s", outputBefore)

	// Verify block header
	if !strings.Contains(outputBefore, "EXECUTE") {
		t.Error("Block should contain EXECUTE tag")
	}
	if !strings.Contains(outputBefore, "ls /home/dmytrogajewski") {
		t.Error("Block should contain 'ls /home/dmytrogajewski'")
	}

	// Should NOT have completion status yet
	if strings.Contains(outputBefore, "↳") {
		t.Error("Block should NOT have completion status line before completion")
	}

	// Step 2: Simulate tool completion (update metadata)
	exitCode := 0
	lines := 95
	meta.ExitCode = &exitCode
	meta.LinesOut = &lines
	block.Body = "Desktop\nDocuments\nDownloads\n..."
	if err := blocks.SetExecuteMeta(block, meta); err != nil {
		t.Fatalf("SetExecuteMeta (after) failed: %v", err)
	}

	// Step 3: Render the block AFTER completion
	outputAfter, err := renderer.Render(block)
	if err != nil {
		t.Fatalf("Render(block after) failed: %v", err)
	}

	t.Logf("Block AFTER completion:\n%s", outputAfter)

	// Verify completion status line appears
	if !strings.Contains(outputAfter, "↳") {
		t.Error("Updated block should contain completion status line (↳)")
	}
	if !strings.Contains(outputAfter, "Exit code: 0") {
		t.Error("Updated block should contain 'Exit code: 0'")
	}
	if !strings.Contains(outputAfter, "Output: 95 lines") {
		t.Error("Updated block should contain 'Output: 95 lines'")
	}

	// Print final output for manual verification
	t.Logf("\n=== EXPECTED OUTPUT (what user should see) ===\n%s", outputAfter)
}
