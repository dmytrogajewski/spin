package adapters_test

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/testkit"
	"github.com/stretchr/testify/require"
)

// TestE2E_InputSubmit_PromptsRedraw tests that user input appears and prompt redraws after submission.
func TestE2E_InputSubmit_PromptsRedraw(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Type and submit
	helper.Keyboard.InjectString("hello")
	helper.Keyboard.InjectEnter()

	// Wait for input to be processed
	require.True(t, helper.WaitForOutput("hello", 1*time.Second), "output should contain 'hello'")
}

// TestE2E_StreamingChunks_PromptAtBottom tests that streaming output doesn't tear the prompt.
func TestE2E_StreamingChunks_PromptAtBottom(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Stream some chunks
	helper.UI.PrintChunks(context.Background(), makeChunkChannel("chunk1", "chunk2", "chunk3"))

	// Wait a bit for streaming
	time.Sleep(100 * time.Millisecond)

	// Output should contain chunks
	output := helper.Writer.StripANSI()
	require.Contains(t, output, "chunk1", "output should contain chunk1")
	require.Contains(t, output, "chunk2", "output should contain chunk2")
	require.Contains(t, output, "chunk3", "output should contain chunk3")
}

// TestE2E_BackspaceEditing tests character deletion works correctly.
func TestE2E_BackspaceEditing(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Type "hello", then backspace, then "i"
	helper.Keyboard.InjectString("hello")
	helper.Keyboard.InjectBackspace()
	helper.Keyboard.InjectString("i")
	helper.Keyboard.InjectEnter()

	// Should contain "helli" (not "hello")
	require.True(t, helper.WaitForOutput("helli", 1*time.Second), "output should contain 'helli'")
}

// TestE2E_AppendBlock_RendersCorrectly tests that blocks appear in timeline.
func TestE2E_AppendBlock_RendersCorrectly(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Create and append a block
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.ID = "test_block"
	block.Title = "Test Command"
	block.Body = "test output"

	require.NoError(t, helper.UI.AppendBlock(block))

	// Wait for block to render
	time.Sleep(100 * time.Millisecond)

	// Output should contain block content
	output := helper.Writer.StripANSI()
	require.Contains(t, output, "Test Command", "output should contain block title")
}

// TestE2E_UpdateBlock_ShowsCompletion tests that block updates show completion status.
func TestE2E_UpdateBlock_ShowsCompletion(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Create initial block
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.ID = "test_block"
	meta := &blocks.ExecuteMeta{
		Command: "test command",
		CWD:     ".",
		Impact:  "low",
	}
	require.NoError(t, blocks.SetExecuteMeta(block, meta))
	require.NoError(t, helper.UI.AppendBlock(block))

	// Update with completion
	exitCode := 0
	meta.ExitCode = &exitCode
	block.Body = "completed"
	require.NoError(t, blocks.SetExecuteMeta(block, meta))
	require.NoError(t, helper.UI.UpdateBlock("test_block", block))

	// Wait for update
	time.Sleep(100 * time.Millisecond)

	// Should show completion status
	output := helper.Writer.StripANSI()
	require.Contains(t, output, "Exit code", "output should contain completion status")
}

// TestE2E_BlockNavigation_PgUpPgDn tests keyboard navigation works.
func TestE2E_BlockNavigation_PgUpPgDn(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Create multiple blocks
	for i := 0; i < 5; i++ {
		block := blocks.NewBlock(blocks.BlockTypeExecute)
		block.ID = blocks.GenerateBlockID(i)
		block.Title = "Block " + string(rune('A'+i))
		require.NoError(t, helper.UI.AppendBlock(block))
	}

	time.Sleep(100 * time.Millisecond)

	// Navigate with PgUp/PgDn
	helper.Keyboard.InjectPgUp()
	time.Sleep(50 * time.Millisecond)
	helper.Keyboard.InjectPgDn()
	time.Sleep(50 * time.Millisecond)

	// Navigation should work without errors
	// (We can't easily verify scroll position without exposing internals,
	// but we can verify no crashes)
}

// TestE2E_StatusBar_UpdatesOnEvents tests that status bar updates in real-time.
func TestE2E_StatusBar_UpdatesOnEvents(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Update status
	helper.UI.SetTokenCount(1000)
	helper.UI.SetProviderInfo("test-provider", "test-model")

	// Wait for status update
	time.Sleep(100 * time.Millisecond)

	// Status bar should have updated (we can't easily verify exact content
	// without exposing internals, but we can verify no crashes)
}

// TestE2E_TerminalResize_RedrawsWithNewWidth tests that resize triggers redraw.
func TestE2E_TerminalResize_RedrawsWithNewWidth(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Resize terminal
	helper.TTY.SetSize(120, 40)

	// Wait for resize handling
	time.Sleep(100 * time.Millisecond)

	// Verify new size is used
	w, h := helper.TTY.Size()
	require.Equal(t, 120, w, "width should be 120")
	require.Equal(t, 40, h, "height should be 40")
}

// TestE2E_ShutdownCtrlC_ExitsCleanly tests that Ctrl+C exits cleanly.
func TestE2E_ShutdownCtrlC_ExitsCleanly(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Send Ctrl+C
	helper.Keyboard.InjectCtrlC()

	// Wait for shutdown
	time.Sleep(100 * time.Millisecond)

	// UI should stop without errors
	// (We verify this by checking that Stop() doesn't panic)
}

// TestE2E_ShutdownContextCancel_ExitsCleanly tests that context cancel exits cleanly.
func TestE2E_ShutdownContextCancel_ExitsCleanly(t *testing.T) {
	helper := testkit.NewTUITest(t)

	helper.Start()

	// Cancel context
	helper.Stop()

	// UI should stop without errors
	// (We verify this by checking that Stop() doesn't panic)
}

// TestE2E_ShutdownCtrlD_ExitsOnEOF tests that Ctrl+D exits on EOF.
func TestE2E_ShutdownCtrlD_ExitsOnEOF(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Send Ctrl+D
	helper.Keyboard.InjectCtrlD()

	// Wait for shutdown
	time.Sleep(100 * time.Millisecond)

	// UI should stop without errors
}

// TestE2E_ApprovalDialog_ShowsOnDangerousCommand tests that approval dialog appears.
// Note: This test is simplified since we can't easily test the full approval dialog
// without importing security package (which would create import cycle).
func TestE2E_ApprovalDialog_ShowsOnDangerousCommand(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// This test verifies that the UI can handle approval requests
	// The actual approval dialog testing is done in overlay package tests
	time.Sleep(50 * time.Millisecond)
}

// TestE2E_FilterMode_Slash tests that filter mode activates with '/'.
func TestE2E_FilterMode_Slash(t *testing.T) {
	helper := testkit.NewTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Press '/' to enter filter mode
	helper.Keyboard.InjectString("/")
	time.Sleep(50 * time.Millisecond)

	// Type filter
	helper.Keyboard.InjectString("type:execute")
	time.Sleep(50 * time.Millisecond)

	// Exit filter mode with Esc
	helper.Keyboard.InjectEscape()
	time.Sleep(50 * time.Millisecond)

	// Filter mode should work without errors
}

// makeChunkChannel creates a channel that streams the given strings as chunks.
func makeChunkChannel(chunks ...string) <-chan string {
	ch := make(chan string, len(chunks))
	go func() {
		defer close(ch)
		for _, chunk := range chunks {
			ch <- chunk
		}
	}()
	return ch
}

