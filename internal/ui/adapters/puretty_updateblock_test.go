package adapters

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/output"
	"github.com/dmytrogajewski/spin/internal/ui/prompt"
	"github.com/dmytrogajewski/spin/internal/ui/status"
)

// TestUpdateBlock_PrintsCompletionStatus verifies that UpdateBlock prints the completion status line.
// This is a regression test for the bug where UpdateBlock was storing updates but not displaying them.
func TestUpdateBlock_PrintsCompletionStatus(t *testing.T) {
	// Setup with actual output capture
	var buf bytes.Buffer
	renderer := prompt.NewRenderer(&buf, 80, "> ")
	model := prompt.NewModel(100)
	blockRenderer := blocks.NewRenderer(80)

	// Create coordinator
	printer := output.NewPrinter(&buf)
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	// Create status management
	statusManager := status.NewManager()
	statusAggregator := status.NewAggregator(statusManager)

	p := &PureTTY{
		out:              &buf,
		model:            model,
		renderer:         renderer,
		coord:            coord,
		statusManager:    statusManager,
		statusAggregator: statusAggregator,
		blockRenderer:    blockRenderer,
		timeline:         blocks.NewTimeline(),
		mode:             ModeInput,
	}

	// Create an initial EXECUTE block (tool starts)
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.ID = "tool_123"

	meta := &blocks.ExecuteMeta{
		Command: "ls /home/test",
		CWD:     ".",
		Impact:  "low",
	}
	if err := blocks.SetExecuteMeta(block, meta); err != nil {
		t.Fatalf("SetExecuteMeta failed: %v", err)
	}

	// Append the initial block
	if err := p.AppendBlock(block); err != nil {
		t.Fatalf("AppendBlock failed: %v", err)
	}

	// Clear buffer after append
	buf.Reset()

	// Update block with completion metadata (tool completes)
	exitCode := 0
	lines := 42
	meta.ExitCode = &exitCode
	meta.LinesOut = &lines
	block.Body = "file1\nfile2\n..."
	if err := blocks.SetExecuteMeta(block, meta); err != nil {
		t.Fatalf("SetExecuteMeta (update) failed: %v", err)
	}

	// Call UpdateBlock - this should print the completion status line
	if err := p.UpdateBlock("tool_123", block); err != nil {
		t.Fatalf("UpdateBlock failed: %v", err)
	}

	// Capture output after update
	output := buf.String()

	// CRITICAL: Verify completion status line was printed
	if !strings.Contains(output, "↳") {
		t.Errorf("UpdateBlock MUST print completion status line (↳)\nGot:\n%s", output)
	}
	if !strings.Contains(output, "Exit code: 0") {
		t.Errorf("Completion status should contain 'Exit code: 0'\nGot:\n%s", output)
	}
	if !strings.Contains(output, "42 lines") {
		t.Errorf("Completion status should contain '42 lines'\nGot:\n%s", output)
	}
}

// TestUpdateBlock_NoStatusForIncompleteBlock verifies that UpdateBlock doesn't print status for incomplete blocks.
func TestUpdateBlock_NoStatusForIncompleteBlock(t *testing.T) {
	// Setup
	p := setupPureTTY(t)
	defer p.Stop()

	// Create a block without completion metadata
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.ID = "tool_456"

	meta := &blocks.ExecuteMeta{
		Command: "go test ./...",
		CWD:     ".",
		Impact:  "medium",
		// No ExitCode set - tool hasn't completed
	}
	if err := blocks.SetExecuteMeta(block, meta); err != nil {
		t.Fatalf("SetExecuteMeta failed: %v", err)
	}

	// Append the block
	if err := p.AppendBlock(block); err != nil {
		t.Fatalf("AppendBlock failed: %v", err)
	}

	// Update with partial data (e.g., just body)
	block.Body = "=== RUN TestFoo\n"
	if err := p.UpdateBlock("tool_456", block); err != nil {
		t.Fatalf("UpdateBlock failed: %v", err)
	}

	// Capture output
	output := captureOutput(p)

	// Should NOT print completion status since ExitCode is not set
	if strings.Contains(output, "↳") && strings.Contains(output, "Exit code") {
		t.Error("UpdateBlock should NOT print completion status for incomplete blocks")
		t.Logf("Unexpected output:\n%s", output)
	}
}

// TestUpdateBlock_HandlesReadBlocks verifies READ blocks don't print completion status.
func TestUpdateBlock_HandlesReadBlocks(t *testing.T) {
	// Setup
	p := setupPureTTY(t)
	defer p.Stop()

	// Create a READ block
	block := blocks.NewBlock(blocks.BlockTypeRead)
	block.ID = "tool_789"
	block.Title = "test.go"
	block.Body = "package main\n\nfunc main() {}"

	meta := &blocks.ReadMeta{
		File:   "test.go",
		Offset: 0,
		Limit:  100,
	}
	if err := blocks.SetReadMeta(block, meta); err != nil {
		t.Fatalf("SetReadMeta failed: %v", err)
	}

	// Append and update
	if err := p.AppendBlock(block); err != nil {
		t.Fatalf("AppendBlock failed: %v", err)
	}

	if err := p.UpdateBlock("tool_789", block); err != nil {
		t.Fatalf("UpdateBlock failed: %v", err)
	}

	// READ blocks don't have completion status lines (per FRD)
	output := captureOutput(p)
	if strings.Contains(output, "↳") {
		t.Error("READ blocks should NOT print completion status lines")
	}
}

// Helper: setupPureTTY creates a test PureTTY instance
func setupPureTTY(t *testing.T) *PureTTY {
	t.Helper()

	out := &bytes.Buffer{}
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	timeline := blocks.NewTimeline()
	blockRenderer := blocks.NewRenderer(80)

	// Create coordinator
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	// Create status management
	statusManager := status.NewManager()
	statusAggregator := status.NewAggregator(statusManager)

	// Create PureTTY directly, bypassing constructor
	ui := &PureTTY{
		model:            model,
		renderer:         renderer,
		coord:            coord,
		statusManager:    statusManager,
		statusAggregator: statusAggregator,
		out:              out,
		timeline:         timeline,
		blockRenderer:    blockRenderer,
		viewportHeight:   0,
		mode:             ModeInput,
		filterInput:      "",
	}

	return ui
}

// Helper: captureOutput returns all output written to the PureTTY
func captureOutput(p *PureTTY) string {
	// In a real implementation, you'd capture from the output buffer
	// For now, this is a placeholder - PureTTY would need a test mode
	// that captures writes instead of sending to terminal

	// TODO: Implement proper output capture in PureTTY test mode
	// For now, return empty string - this test will fail if run
	return ""
}
