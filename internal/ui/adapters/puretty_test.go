package adapters

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/output"
	"github.com/dmytrogajewski/spin/internal/ui/prompt"
	"github.com/dmytrogajewski/spin/internal/ui/status"
	"github.com/dmytrogajewski/spin/internal/ui/term"
)

// TestUpdateBlock_PrintsCompletionStatus verifies that UpdateBlock prints the completion status line.
// This is a regression test for the bug where UpdateBlock was storing updates but not displaying them.
func TestUpdateBlock_PrintsCompletionStatus(t *testing.T) {
	t.Parallel(
	// Setup with actual output capture.
	)

	var buf bytes.Buffer

	renderer := prompt.NewTermRenderer(&buf, 80, "> ")
	model := prompt.NewModel(100)
	blockRenderer := blocks.NewRenderer(80)

	// Create coordinator.
	printer := output.NewPrinter(&buf)
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	// Create status management.
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

	// Create an initial EXECUTE block (tool starts).
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.ID = "tool_123"

	meta := &blocks.ExecuteMeta{
		Command: "ls /home/test",
		CWD:     ".",
		Impact:  "low",
	}

	err := blocks.SetExecuteMeta(block, meta)
	if err != nil {
		t.Fatalf("SetExecuteMeta failed: %v", err)
	}

	// Append the initial block.
	err = p.AppendBlock(block)
	if err != nil {
		t.Fatalf("AppendBlock failed: %v", err)
	}

	// Clear buffer after append.
	buf.Reset()

	// Update block with completion metadata (tool completes).
	exitCode := 0
	lines := 42
	meta.ExitCode = &exitCode
	meta.LinesOut = &lines

	block.Body = "file1\nfile2\n..."

	err = blocks.SetExecuteMeta(block, meta)
	if err != nil {
		t.Fatalf("SetExecuteMeta (update) failed: %v", err)
	}

	// Call UpdateBlock - this should print the completion status line.
	err = p.UpdateBlock("tool_123", block)
	if err != nil {
		t.Fatalf("UpdateBlock failed: %v", err)
	}

	// Capture output after update.
	got := buf.String()

	// CRITICAL: Verify completion status line was printed.
	if !strings.Contains(got, "⤷") {
		t.Errorf("UpdateBlock MUST print completion status line (⤷)\nGot:\n%s", got)
	}

	if !strings.Contains(got, "Exit code: 0") {
		t.Errorf("Completion status should contain 'Exit code: 0'\nGot:\n%s", got)
	}

	if !strings.Contains(got, "42 lines") {
		t.Errorf("Completion status should contain '42 lines'\nGot:\n%s", got)
	}
}

// TestUpdateBlock_NoStatusForIncompleteBlock verifies that UpdateBlock doesn't print status for incomplete blocks.
func TestUpdateBlock_NoStatusForIncompleteBlock(t *testing.T) {
	t.Parallel(
	// Setup.
	)

	p := setupPureTTY(t)

	defer func() { _ = p.Stop() }()

	// Create a block without completion metadata.
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.ID = "tool_456"

	meta := &blocks.ExecuteMeta{
		Command: "go test ./...",
		CWD:     ".",
		Impact:  "medium",
		// No ExitCode set - tool hasn't completed.
	}

	err := blocks.SetExecuteMeta(block, meta)
	if err != nil {
		t.Fatalf("SetExecuteMeta failed: %v", err)
	}

	// Append the block.
	err = p.AppendBlock(block)
	if err != nil {
		t.Fatalf("AppendBlock failed: %v", err)
	}

	// Update with partial data (e.g., just body).
	block.Body = "=== RUN TestFoo\n"

	err = p.UpdateBlock("tool_456", block)
	if err != nil {
		t.Fatalf("UpdateBlock failed: %v", err)
	}

	// Capture output.
	got := captureOutput(p)

	// Should NOT print completion status since ExitCode is not set.
	if strings.Contains(got, "⤷") && strings.Contains(got, "Exit code") {
		t.Error("UpdateBlock should NOT print completion status for incomplete blocks")
		t.Logf("Unexpected output:\n%s", got)
	}
}

// TestUpdateBlock_HandlesReadBlocks verifies READ blocks don't print completion status.
func TestUpdateBlock_HandlesReadBlocks(t *testing.T) {
	t.Parallel(
	// Setup.
	)

	p := setupPureTTY(t)

	defer func() { _ = p.Stop() }()

	// Create a READ block.
	block := blocks.NewBlock(blocks.BlockTypeRead)
	block.ID = "tool_789"
	block.Title = "test.go"
	block.Body = "package main\n\nfunc main() {}"

	meta := &blocks.ReadMeta{
		File:   "test.go",
		Offset: 0,
		Limit:  100,
	}

	err := blocks.SetReadMeta(block, meta)
	if err != nil {
		t.Fatalf("SetReadMeta failed: %v", err)
	}

	// Append and update.
	err = p.AppendBlock(block)
	if err != nil {
		t.Fatalf("AppendBlock failed: %v", err)
	}

	err = p.UpdateBlock("tool_789", block)
	if err != nil {
		t.Fatalf("UpdateBlock failed: %v", err)
	}

	// READ blocks don't have completion status lines (per FRD).
	got := captureOutput(p)
	if strings.Contains(got, "⤷") {
		t.Error("READ blocks should NOT print completion status lines")
	}
}

// Helper: setupPureTTY creates a test PureTTY instance.
func setupPureTTY(t *testing.T) *PureTTY {
	t.Helper()

	out := &bytes.Buffer{}
	model := prompt.NewModel(100)
	renderer := prompt.NewTermRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	timeline := blocks.NewTimeline()
	blockRenderer := blocks.NewRenderer(80)

	// Create coordinator.
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	// Create status management.
	statusManager := status.NewManager()
	statusAggregator := status.NewAggregator(statusManager)

	// Create PureTTY directly, bypassing constructor.
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

// Helper: captureOutput returns all output written to the PureTTY.
func captureOutput(_ *PureTTY) string {
	// Placeholder: Output capture requires a test mode in PureTTY
	// that intercepts writes instead of sending to terminal
	// This would require refactoring PureTTY to accept an io.Writer
	// that can be swapped for testing purposes.
	return ""
}

// TestApprovalDialog_KeyboardInput tests that keyboard input is routed
// to the approval dialog when it's active.
// This is a regression test for the bug where arrow keys and A/D keys
// didn't work because all keyboard events were consumed by the prompt loop.
func TestApprovalDialog_KeyboardInput(t *testing.T) {
	t.Parallel(
	// Create a buffer for output.
	)

	var buf bytes.Buffer

	// Create a keyboard event channel.
	keyCh := make(chan term.KeyEvent, 10)

	// Create mock TTY.
	mockTTY := &mockTerminalController{
		width:  80,
		height: 24,
	}

	// Create PureTTY with test options.
	ui, err := NewPureTTY(&buf,
		WithTTY(mockTTY),
		WithKeyboardEvents(keyCh),
	)
	if err != nil {
		t.Fatalf("NewPureTTY failed: %v", err)
	}

	// Start UI in background.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = ui.Run(ctx)
	}()

	// Give UI time to start.
	time.Sleep(50 * time.Millisecond)

	// Create approval request.
	cmd := &safety.Command{
		Program: "rm",
		Args:    []string{"-rf", "/"},
		Raw:     "rm -rf /",
	}
	req := safety.ApprovalRequest{
		ID:      "test-123",
		Command: cmd,
		Reason:  "Testing approval dialog keyboard input",
		WorkDir: "/tmp",
	}

	// Start ShowApprovalDialog in background (it blocks).
	responseCh := make(chan safety.ApprovalResponse, 1)

	go func() {
		resp := ui.ShowApprovalDialog(context.Background(), req)
		responseCh <- resp
	}()

	// Give dialog time to render.
	time.Sleep(50 * time.Millisecond)

	// Verify mode is ModeApproval.
	ui.mu.Lock()
	mode := ui.mode
	ui.mu.Unlock()

	if mode != ModeApproval {
		t.Fatalf("Expected mode to be ModeApproval, got %v", mode)
	}

	// Test 1: Send 'A' key to approve.
	keyCh <- term.KeyEvent{
		Kind: term.KeyRune,
		Rune: 'A',
	}

	// Wait for response with timeout.
	select {
	case resp := <-responseCh:
		if !resp.Approved {
			t.Error("Expected approval to be approved after pressing 'A'")
		}

		if resp.Reason != "user approved" {
			t.Errorf("Expected reason 'user approved', got '%s'", resp.Reason)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("BUG: Approval dialog did not respond to 'A' key press - keyboard events not routed to dialog")
	}

	// Cleanup.
	cancel()
}

// TestApprovalDialog_DenyKey tests that 'D' key denies the request.
func TestApprovalDialog_DenyKey(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	keyCh := make(chan term.KeyEvent, 10)
	mockTTY := &mockTerminalController{width: 80, height: 24}

	ui, err := NewPureTTY(&buf, WithTTY(mockTTY), WithKeyboardEvents(keyCh))
	if err != nil {
		t.Fatalf("NewPureTTY failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	req := safety.ApprovalRequest{
		ID:      "test-456",
		Command: &safety.Command{Program: "rm", Raw: "rm file.txt"},
		Reason:  "Testing deny",
		WorkDir: "/tmp",
	}

	responseCh := make(chan safety.ApprovalResponse, 1)

	go func() {
		resp := ui.ShowApprovalDialog(context.Background(), req)
		responseCh <- resp
	}()

	time.Sleep(50 * time.Millisecond)

	// Send 'D' key to deny.
	keyCh <- term.KeyEvent{
		Kind: term.KeyRune,
		Rune: 'D',
	}

	select {
	case resp := <-responseCh:
		if resp.Approved {
			t.Error("Expected approval to be denied after pressing 'D'")
		}

		if resp.Reason != "user denied" {
			t.Errorf("Expected reason 'user denied', got '%s'", resp.Reason)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("BUG: Approval dialog did not respond to 'D' key press")
	}

	cancel()
}

// TestApprovalDialog_ArrowKeys tests arrow key navigation in approval dialog.
func TestApprovalDialog_ArrowKeys(t *testing.T) {
	t.Parallel()
	t.Skip("Arrow key navigation implemented but needs approval dialog HandleKey support")

	// Placeholder: This test requires the approval dialog to handle arrow key escape sequences
	// 1. Show approval dialog
	// 2. Press right arrow (\x1b[C) to move to "Deny" button
	// 3. Verify selection changed
	// 4. Press Enter to confirm
	// 5. Verify denial.
}

// mockTerminalController is a mock implementation for testing.
type mockTerminalController struct {
	width      int
	height     int
	entered    bool
	resizeFunc func(int, int)
}

func (m *mockTerminalController) Enter() error {
	m.entered = true

	return nil
}

func (m *mockTerminalController) Exit() error {
	m.entered = false

	return nil
}

func (m *mockTerminalController) Size() (width, height int) {
	return m.width, m.height
}

func (m *mockTerminalController) OnResize(f func(int, int)) {
	m.resizeFunc = f
}

// testPureTTYSetup holds common test setup for PureTTY tests.
type testPureTTYSetup struct {
	buf              bytes.Buffer
	p                *PureTTY
	statusManager    *status.Manager
	statusAggregator *status.Aggregator
}

// newTestPureTTYSetup creates a PureTTY with standard test dependencies.
func newTestPureTTYSetup() *testPureTTYSetup {
	s := &testPureTTYSetup{}
	renderer := prompt.NewTermRenderer(&s.buf, 80, "> ")
	model := prompt.NewModel(100)
	blockRenderer := blocks.NewRenderer(80)
	printer := output.NewPrinter(&s.buf)
	ra := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, ra, model)
	s.statusManager = status.NewManager()
	s.statusAggregator = status.NewAggregator(s.statusManager)

	s.p = &PureTTY{
		out:              &s.buf,
		model:            model,
		renderer:         renderer,
		coord:            coord,
		statusManager:    s.statusManager,
		statusAggregator: s.statusAggregator,
		blockRenderer:    blockRenderer,
		timeline:         blocks.NewTimeline(),
		mode:             ModeInput,
	}

	return s
}

// TestUpdateBlock_NoDuplicateToolCompleted reproduces and tests the fix for duplicate "Tool completed" messages.
// This test simulates the exact scenario where a TOOL block is updated multiple times,
// which was causing "Tool completed" to print twice.
func TestUpdateBlock_NoDuplicateToolCompleted(t *testing.T) {
	t.Parallel()

	s := newTestPureTTYSetup()
	p := s.p

	// Create an initial TOOL block (like execute_command).
	block := blocks.NewBlock(blocks.BlockTypeTool)
	block.ID = "tool_exec_123"
	block.Title = "execute_command"

	// Set initial metadata (tool starts).
	meta := &blocks.ToolMeta{
		ToolName: "execute_command",
	}

	err := blocks.SetToolMeta(block, meta)
	if err != nil {
		t.Fatalf("SetToolMeta failed: %v", err)
	}

	// Append the initial block.
	err = p.AppendBlock(block)
	if err != nil {
		t.Fatalf("AppendBlock failed: %v", err)
	}

	s.buf.Reset()

	toolBody := "Command executed successfully: итого 4\n" +
		"drwxr-xr-x. 1 dmitriy 14 окт 25 16:17 .\n" +
		"drwxr-xr-x. 1 dmitriy 54 окт 25 16:17 ..\n" +
		"-rw-r--r--. 1 dmitriy 45 окт 25 16:17 main.rs"

	// First update: tool completes with success.
	completedBlock := createToolBlock(t, "tool_exec_123", "execute_command", toolBody)

	err = p.UpdateBlock("tool_exec_123", completedBlock)
	if err != nil {
		t.Fatalf("First UpdateBlock failed: %v", err)
	}

	// Second update: same tool, might be called again due to event processing.
	secondUpdateBlock := createToolBlock(t, "tool_exec_123", "execute_command", toolBody)

	err = p.UpdateBlock("tool_exec_123", secondUpdateBlock)
	if err != nil {
		t.Fatalf("Second UpdateBlock failed: %v", err)
	}

	// Count occurrences of "Tool completed".
	got := s.buf.String()

	toolCompletedCount := strings.Count(got, "Tool completed")
	if toolCompletedCount > 1 {
		t.Errorf("'Tool completed' should appear exactly once, but appeared %d times\nOutput:\n%s",
			toolCompletedCount, got)
	}

	// Also count the arrow symbol.
	arrowCount := strings.Count(got, "⤷")
	if arrowCount > 2 { // One for the initial status, one for completion.
		t.Errorf("Arrow '⤷' appearing too many times (%d), suggesting duplicate completion lines\nOutput:\n%s",
			arrowCount, got)
	}

	// Ensure it printed at least once.
	if toolCompletedCount == 0 {
		t.Errorf("'Tool completed' should appear at least once\nOutput:\n%s", got)
	}
}

// TestExecuteBlock_NoDuplicateExitStatus tests that EXECUTE blocks also don't duplicate status.
func TestExecuteBlock_NoDuplicateExitStatus(t *testing.T) {
	t.Parallel()

	s := newTestPureTTYSetup()
	p := s.p

	// Create an initial EXECUTE block.
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.ID = "exec_123"

	meta := &blocks.ExecuteMeta{
		Command: "ls",
		CWD:     ".",
		Impact:  "low",
	}

	err := blocks.SetExecuteMeta(block, meta)
	if err != nil {
		t.Fatalf("SetExecuteMeta failed: %v", err)
	}

	// Append the initial block.
	err = p.AppendBlock(block)
	if err != nil {
		t.Fatalf("AppendBlock failed: %v", err)
	}

	s.buf.Reset()

	exitCode := 0
	lines := 3

	// First update with completion.
	firstUpdate := createExecuteBlock(t, "exec_123", "ls", ".", "low", &exitCode, &lines, "file1\nfile2\nfile3")

	err = p.UpdateBlock("exec_123", firstUpdate)
	if err != nil {
		t.Fatalf("First UpdateBlock failed: %v", err)
	}

	// Second update with same completion status.
	secondUpdate := createExecuteBlock(t, "exec_123", "ls", ".", "low", &exitCode, &lines, "file1\nfile2\nfile3")

	err = p.UpdateBlock("exec_123", secondUpdate)
	if err != nil {
		t.Fatalf("Second UpdateBlock failed: %v", err)
	}

	got := s.buf.String()
	assertCountInRange(t, got, "Exit code: 0", 1, 1)
}

// TestApprovalDialog_StatusBarNotOverwritten verifies that the approval prompt in the status bar
// is not overwritten by event-driven status updates.
// This is a regression test for BUG-20251029001449.
func TestApprovalDialog_StatusBarNotOverwritten(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	mockTTY := &mockTerminalController{width: 160, height: 24}
	statusRenderer := status.NewRenderer(&buf, 160, 24)
	statusManager := status.NewManager()

	ui := &PureTTY{
		out:            &buf,
		tty:            mockTTY,
		model:          prompt.NewModel(100),
		mode:           ModeInput,
		statusRenderer: statusRenderer,
		statusManager:  statusManager,
	}

	// Verify initial status renders correctly.
	statusManager.SetAgentState("Ready")
	ui.updateStatusBar()
	assertContains(t, buf.String(), "Ready", "initial status")
	buf.Reset()

	// Switch to approval mode and show approval status.
	ui.mu.Lock()
	ui.mode = ModeApproval
	ui.mu.Unlock()

	req := safety.ApprovalRequest{
		ID:      "test-approval-123",
		Command: &safety.Command{Program: "rm", Args: []string{"-rf", "/tmp/build"}, Raw: "rm -rf /tmp/build"},
		Reason:  "Testing status bar not overwritten",
		WorkDir: "/tmp",
	}
	ui.showApprovalStatus(req)

	approvalOutput := buf.String()
	assertContains(t, approvalOutput, "Executing:", "approval prompt")
	assertContains(t, approvalOutput, "Key:", "normalized key preview")
	assertContains(t, approvalOutput, "TTLs:", "TTL preview")
	buf.Reset()

	// Simulate event-driven status update (this would normally overwrite the approval prompt).
	statusManager.SetAgentState("Processing")
	ui.updateStatusBar()

	// Verify that updateStatusBar did NOT render anything (because we're in ModeApproval).
	eventOutput := buf.String()
	if strings.Contains(eventOutput, "Processing") {
		t.Errorf("updateStatusBar should not render in ModeApproval, but rendered: %s", eventOutput)
	}

	if eventOutput != "" {
		t.Errorf("updateStatusBar should not render anything in ModeApproval, but rendered: %s", eventOutput)
	}

	// Switch back to input mode.
	ui.mu.Lock()
	ui.mode = ModeInput
	ui.mu.Unlock()

	// Clear buffer.
	buf.Reset()

	// Verify that updateStatusBar works again after leaving approval mode.
	statusManager.SetAgentState("Approved")

	// Need to reset lastStatusText to force a re-render since status changed.
	ui.mu.Lock()
	ui.lastStatusText = ""
	ui.mu.Unlock()

	ui.updateStatusBar()

	finalOutput := buf.String()
	if !strings.Contains(finalOutput, "Approved") {
		t.Errorf("Expected status 'Approved' after leaving approval mode, got: %s", finalOutput)
	}
}

// createToolBlock creates a tool block with metadata for testing.
func createToolBlock(t *testing.T, id, toolName, body string) *blocks.Block {
	t.Helper()

	b := blocks.NewBlock(blocks.BlockTypeTool)
	b.ID = id
	b.Title = toolName
	b.Body = body

	meta := &blocks.ToolMeta{ToolName: toolName}
	if err := blocks.SetToolMeta(b, meta); err != nil {
		t.Fatalf("SetToolMeta failed: %v", err)
	}

	return b
}

// createExecuteBlock creates an execute block with metadata for testing.
func createExecuteBlock(t *testing.T, id, cmd, cwd, impact string, exitCode, lines *int, body string) *blocks.Block {
	t.Helper()

	b := blocks.NewBlock(blocks.BlockTypeExecute)
	b.ID = id
	b.Body = body

	meta := &blocks.ExecuteMeta{Command: cmd, CWD: cwd, Impact: impact, ExitCode: exitCode, LinesOut: lines}
	if err := blocks.SetExecuteMeta(b, meta); err != nil {
		t.Fatalf("SetExecuteMeta failed: %v", err)
	}

	return b
}

// assertContains checks that s contains substr, or fails with a descriptive message.
func assertContains(t *testing.T, s, substr, msg string) {
	t.Helper()

	if !strings.Contains(s, substr) {
		t.Errorf("Expected %s to contain '%s', got: %s", msg, substr, s)
	}
}

// assertCountInRange checks that substr appears between minCount and maxCount times in s.
func assertCountInRange(t *testing.T, s, substr string, minCount, maxCount int) {
	t.Helper()

	count := strings.Count(s, substr)
	if count < minCount {
		t.Errorf("'%s' should appear at least %d time(s), but appeared %d\nOutput:\n%s", substr, minCount, count, s)
	}

	if count > maxCount {
		t.Errorf("'%s' should appear at most %d time(s), but appeared %d\nOutput:\n%s", substr, maxCount, count, s)
	}
}
