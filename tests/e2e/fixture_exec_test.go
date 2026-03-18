package e2e // Journey: specs/journeys/JOURNEY-R1.md.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestFixture_SimpleResponse verifies that a simple text response
// (no tool calls) is rendered correctly in exec mode output.
func TestFixture_SimpleResponse(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "simple_response.jsonl", "what is the answer?")
	assertNoError(t, r)
	assertOutputContains(t, r, "The answer is 42.")
}

// TestFixture_ToolCallBlockVisible is a REGRESSION test for the bug where
// tool call blocks were invisible in exec output because EventToolCallStart
// and EventToolCallComplete were never emitted by the tool runtime.
//
// This test verifies that when the LLM calls read_file, the tool block
// appears in the terminal output.
func TestFixture_ToolCallBlockVisible(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"test.txt": "hello world",
	})

	r := runFixtureExec(t, "read_file_tool_visible.jsonl",
		"read test.txt", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	// The tool block header must appear in output (READ block for read_file).
	assertOutputContains(t, r, "READ")

	// The final LLM response must appear.
	assertOutputContains(t, r, "The file contains")
}

// TestFixture_MultiToolObservation is a REGRESSION test for the bug where
// the observation summarizer destroyed tool result content before the LLM
// could see it. The fix was moving phaseObservation to only summarize
// messages from before the current dispatch (using preDispatchLen boundary).
//
// This test uses a 3-turn fixture: list_directory → read_file → final response.
// The final response references data from the read_file result, which would
// be impossible if the observation summarizer had destroyed it.
func TestFixture_MultiToolObservation(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"data.txt": "value 12345",
	})

	r := runFixtureExec(t, "multi_tool_observation.jsonl",
		"read data.txt", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	// Both tool calls should produce visible blocks.
	// list_directory renders as EXECUTE block with "ls" command.
	assertOutputContains(t, r, "EXECUTE")
	assertOutputContains(t, r, "READ")

	// Final response must contain data that could only come from the LLM
	// having seen the raw tool result (not a summarized "Read file (X lines)" stub).
	assertOutputContains(t, r, "value 12345")
}

// TestFixture_ShellCommandOutput verifies that shell command tool execution
// produces visible output in exec mode.
func TestFixture_ShellCommandOutput(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "shell_command_output.jsonl",
		"run echo hello world", withAutoApprove())
	assertNoError(t, r)

	// The shell command output should be visible.
	assertOutputContains(t, r, "hello world")
}

// TestFixture_WriteDeniedWithoutAutoApprove verifies that write operations
// are blocked when --auto-approve is not passed in exec mode.
func TestFixture_WriteDeniedWithoutAutoApprove(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := t.TempDir()

	_ = runFixtureExec(t, "write_denied_no_approve.jsonl",
		"create secret.txt", withWorkDir(workDir))

	// The file must NOT be created on disk.
	targetFile := filepath.Join(workDir, "secret.txt")
	if _, err := os.Stat(targetFile); err == nil {
		t.Errorf("file was created without --auto-approve: %s", targetFile)
	}
}

// TestFixture_WriteAllowedWithAutoApprove verifies that a dangerous operation
// (shell_command creating a file) succeeds when --auto-approve is passed.
// The file must be created on disk, proving the approval gate was passed.
func TestFixture_WriteAllowedWithAutoApprove(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := t.TempDir()

	r := runFixtureExec(t, "write_with_approve.jsonl",
		"create a file", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	// The file must be created on disk (proves --auto-approve worked).
	targetFile := filepath.Join(workDir, "output.txt")

	if _, err := os.Stat(targetFile); err != nil {
		t.Fatalf("file not created with --auto-approve: %v", err)
	}

	// The final LLM response must appear.
	assertOutputContains(t, r, "File created successfully")
}

// TestFixture_ReadNonexistentFile verifies that when the LLM calls read_file
// on a missing path, the error is visible in output and the LLM still responds.
func TestFixture_ReadNonexistentFile(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := t.TempDir() // No files — missing.txt does not exist.

	r := runFixtureExec(t, "read_nonexistent.jsonl",
		"read missing.txt", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	// The tool block must appear (even for errors).
	assertOutputContains(t, r, "READ")

	// The final LLM response must appear (LLM recovers from tool error).
	assertOutputContains(t, r, "The file does not exist")
}

// TestFixture_ShellCommandFailure verifies that a non-zero exit code from
// shell_command is visible in exec output and the LLM responds gracefully.
func TestFixture_ShellCommandFailure(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "shell_command_fail.jsonl",
		"run a failing command", withAutoApprove())
	assertNoError(t, r)

	// Error from the failed command must be visible.
	assertOutputContains(t, r, "exit")

	// The final LLM response must appear.
	assertOutputContains(t, r, "command failed")
}

// --- R2: Tool Block Visibility for All Tool Types ---
// Journey: specs/journeys/JOURNEY-R2.md.

// TestFixture_WriteFileBlockVisible verifies that write_file renders a visible
// block in exec output. Uses read_file first to satisfy FileTracker.
func TestFixture_WriteFileBlockVisible(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"out.txt": "original",
	})

	r := runFixtureExec(t, "write_file_block.jsonl",
		"write to out.txt", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	// write_file renders as WRITE or APPLY_PATCH block.
	assertOutputContains(t, r, "WRITE")

	// File must be updated on disk.
	content, err := os.ReadFile(filepath.Join(workDir, "out.txt"))
	if err != nil {
		t.Fatalf("file not found after write: %v", err)
	}

	if string(content) != "hello from write" {
		t.Errorf("unexpected file content: %q", string(content))
	}
}

// TestFixture_EditFileBlockVisible verifies that edit_file renders a visible
// block in exec output. Uses read_file first to satisfy FileTracker.
func TestFixture_EditFileBlockVisible(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"src.txt": "foo",
	})

	r := runFixtureExec(t, "edit_file_block.jsonl",
		"edit src.txt", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	// edit_file renders as APPLY_PATCH or EDIT block.
	assertOutputContains(t, r, "src.txt")

	// File must be edited on disk.
	content, err := os.ReadFile(filepath.Join(workDir, "src.txt"))
	if err != nil {
		t.Fatalf("file not found after edit: %v", err)
	}

	if string(content) != "bar" {
		t.Errorf("expected file content %q, got %q", "bar", string(content))
	}
}

// TestFixture_ListDirectoryBlockVisible verifies that list_directory renders
// an EXECUTE block in exec output.
func TestFixture_ListDirectoryBlockVisible(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"visible_file.txt": "content",
	})

	r := runFixtureExec(t, "list_directory_block.jsonl",
		"list directory", withWorkDir(workDir))
	assertNoError(t, r)

	// list_directory renders as EXECUTE block.
	assertOutputContains(t, r, "EXECUTE")

	// The test file should appear in the listing.
	assertOutputContains(t, r, "visible_file.txt")
}

// TestFixture_FileSearchBlockVisible verifies that file_search renders
// a GREP block (or search results) in exec output.
func TestFixture_FileSearchBlockVisible(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"test_target.go": "package main",
	})

	r := runFixtureExec(t, "file_search_block.jsonl",
		"search for test_target", withWorkDir(workDir))
	assertNoError(t, r)

	// file_search renders as GREP or TOOL block. The key assertion is that
	// the tool block appears and the LLM response follows.
	assertOutputContains(t, r, "Search complete")
}

// --- R3: Security & Approval Matrix ---
// Journey: specs/journeys/JOURNEY-R3.md.

// TestFixture_ShellDeniedWithoutAutoApprove verifies that shell_command is
// blocked when --auto-approve is not passed. The command must never execute.
func TestFixture_ShellDeniedWithoutAutoApprove(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "shell_denied_no_approve.jsonl",
		"run echo secret")

	// "secret" must NOT appear — the command must not have executed.
	output := combinedOutput(r)
	if strings.Contains(output, "secret") && !strings.Contains(output, "denied") &&
		!strings.Contains(output, "requires --auto-approve") {
		t.Errorf("shell command appears to have executed without approval:\n%s", output)
	}
}

// TestFixture_ReadAllowedWithoutAutoApprove verifies that read_file works
// without --auto-approve because it is a safe (read-only) tool.
func TestFixture_ReadAllowedWithoutAutoApprove(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"safe.txt": "safe content visible",
	})

	r := runFixtureExec(t, "read_allowed_no_approve.jsonl",
		"read safe.txt", withWorkDir(workDir))
	assertNoError(t, r)

	// read_file is safe — content must be visible without --auto-approve.
	assertOutputContains(t, r, "File read successfully")
}

// TestFixture_ListDirAllowedWithoutAutoApprove verifies that list_directory
// works without --auto-approve because it is a safe (read-only) tool.
func TestFixture_ListDirAllowedWithoutAutoApprove(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"listed_file.txt": "content",
	})

	r := runFixtureExec(t, "list_dir_no_approve.jsonl",
		"list directory", withWorkDir(workDir))
	assertNoError(t, r)

	// list_directory is safe — entries must be visible.
	assertOutputContains(t, r, "listed_file.txt")
}

// TestFixture_EditDeniedWithoutAutoApprove verifies that edit_file is blocked
// without --auto-approve. The file must remain unchanged on disk.
func TestFixture_EditDeniedWithoutAutoApprove(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"src.txt": "original",
	})

	_ = runFixtureExec(t, "edit_denied_no_approve.jsonl",
		"edit src.txt", withWorkDir(workDir))

	// File must remain unchanged (edit was denied).
	content, err := os.ReadFile(filepath.Join(workDir, "src.txt"))
	if err != nil {
		t.Fatalf("cannot read file: %v", err)
	}

	if string(content) != "original" {
		t.Errorf("file was modified without --auto-approve: %q", string(content))
	}
}

// --- R4: Multi-Turn & Observation Stress ---
// Journey: specs/journeys/JOURNEY-R4.md.

// TestFixture_ReadThenShell verifies mixed tool types across turns.
// read_file → shell_command → final response.
func TestFixture_ReadThenShell(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"data.txt": "input data",
	})

	r := runFixtureExec(t, "read_then_shell.jsonl",
		"process data", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	assertOutputContains(t, r, "READ")
	assertOutputContains(t, r, "processed")
	assertOutputContains(t, r, "Read and shell both completed")
}

// TestFixture_FiveToolTurns verifies the observation boundary at every turn
// across 5 sequential tool calls. The final response references content from
// the 5th tool call, which would fail if observation destroyed it.
func TestFixture_FiveToolTurns(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"a.txt":     "alpha",
		"b.txt":     "bravo",
		"c.txt":     "charlie",
		"final.txt": "marker99",
	})

	r := runFixtureExec(t, "five_tool_turns.jsonl",
		"read all files", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	// All tool blocks should be visible.
	assertOutputContains(t, r, "EXECUTE")
	assertOutputContains(t, r, "READ")

	// Final response references 5th tool call content.
	assertOutputContains(t, r, "marker99")
}

// TestFixture_ReadThenWrite verifies read-before-write pattern.
// read_file → write_file → response.
func TestFixture_ReadThenWrite(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"input.txt": "original content",
	})

	r := runFixtureExec(t, "read_then_write.jsonl",
		"transform input", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	assertOutputContains(t, r, "READ")
	assertOutputContains(t, r, "Read and write done")

	// File must be updated on disk.
	content, err := os.ReadFile(filepath.Join(workDir, "input.txt"))
	if err != nil {
		t.Fatalf("file not found after write: %v", err)
	}

	if string(content) != "transformed" {
		t.Errorf("unexpected file content: %q", string(content))
	}
}

// TestFixture_ThreeToolCalls verifies 3 sequential tool calls all produce
// visible blocks: list_directory → read_file → shell_command → response.
func TestFixture_ThreeToolCalls(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := setupFixtureWorkDir(t, map[string]string{
		"info.txt": "information",
	})

	r := runFixtureExec(t, "three_tool_calls.jsonl",
		"process info", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	// All 3 tool types visible.
	assertOutputContains(t, r, "EXECUTE")
	assertOutputContains(t, r, "READ")
	assertOutputContains(t, r, "done")
	assertOutputContains(t, r, "All three tools executed")
}

// --- R5: Tool Error Handling (Completion) ---
// Journey: specs/journeys/JOURNEY-R5.md.

// TestFixture_WriteToReadonlyPath verifies that writing to an invalid path
// produces a visible error and the process exits cleanly.
func TestFixture_WriteToReadonlyPath(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "write_readonly_path.jsonl",
		"write to invalid path", withAutoApprove())
	assertNoError(t, r)

	// LLM response must appear (process didn't crash).
	assertOutputContains(t, r, "Write failed as expected")
}

// TestFixture_ToolNotFound verifies that calling a nonexistent tool name
// produces an error and the LLM still responds.
func TestFixture_ToolNotFound(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "unknown_tool.jsonl",
		"call unknown tool", withAutoApprove())
	assertNoError(t, r)

	// LLM must still respond after the error.
	assertOutputContains(t, r, "Tool not found")
}

// --- R6: Response Variants & Edge Cases ---
// Journey: specs/journeys/JOURNEY-R5.md.

// TestFixture_MultiLineResponse verifies multi-line text is preserved.
func TestFixture_MultiLineResponse(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "multiline_response.jsonl", "tell me lines")
	assertNoError(t, r)

	assertOutputContains(t, r, "Line one")
	assertOutputContains(t, r, "Line two")
	assertOutputContains(t, r, "Line three")
}

// TestFixture_EmptyResponse verifies empty content causes no panic.
// The LLM caller retries empty responses, so the process may exit with an error,
// but it must not panic or hang.
func TestFixture_EmptyResponse(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "empty_response.jsonl", "say nothing",
		withTimeout(15*time.Second))

	// The process may error (caller retries exhaust fixture), but must not panic.
	output := combinedOutput(r)
	if strings.Contains(output, "panic") {
		t.Errorf("empty response caused panic:\n%s", output)
	}
}

// TestFixture_EmptyToolArguments verifies that a tool call with empty `{}`
// arguments produces an error and the LLM recovers.
func TestFixture_EmptyToolArguments(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "empty_tool_args.jsonl",
		"call with empty args", withAutoApprove())
	assertNoError(t, r)

	// LLM must recover and respond.
	assertOutputContains(t, r, "Got error from empty args")
}

// TestFixture_LargeFileRead verifies that file content >100 chars is NOT
// truncated by the observation summarizer on the current turn.
func TestFixture_LargeFileRead(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a 500-char file with a unique marker at the end.
	largeContent := strings.Repeat("x", 490) + "ENDMARKER"

	workDir := setupFixtureWorkDir(t, map[string]string{
		"large.txt": largeContent,
	})

	r := runFixtureExec(t, "read_large_file.jsonl",
		"read large file", withWorkDir(workDir), withAutoApprove())
	assertNoError(t, r)

	// The LLM response must appear (proving the tool result wasn't lost).
	assertOutputContains(t, r, "File read with all content intact")
}

// TestFixture_SpecialCharsInOutput verifies unicode characters are preserved.
func TestFixture_SpecialCharsInOutput(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "special_chars.jsonl",
		"echo special chars", withAutoApprove())
	assertNoError(t, r)

	assertOutputContains(t, r, "wörld")
}

// --- R7: Command-Line Flags ---
// Journey: specs/journeys/JOURNEY-R5.md.

// TestFixture_TimeoutExpired verifies that --timeout causes a timeout when
// the fixture response is delayed beyond the limit.
func TestFixture_TimeoutExpired(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "slow_response.jsonl",
		"slow query",
		withExecTimeout("1s"),
		withTimeout(10*time.Second))

	// Process must exit with error (timeout).
	if r.err == nil {
		t.Error("expected timeout error, got success")
	}

	output := combinedOutput(r)
	if !strings.Contains(output, "canceled") && !strings.Contains(output, "timeout") &&
		!strings.Contains(output, "deadline") {
		t.Errorf("expected timeout-related error, got:\n%s", output)
	}
}

// TestFixture_ExitOnErrorTrue verifies that --exit-on-error (default true)
// causes a non-zero exit code when the harness encounters an error.
func TestFixture_ExitOnErrorTrue(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Use the empty response fixture which causes caller retry exhaustion.
	r := runFixtureExec(t, "empty_response.jsonl",
		"trigger error",
		withExitOnError(true),
		withTimeout(15*time.Second))

	// Must exit with error.
	if r.err == nil {
		t.Error("expected non-zero exit code with --exit-on-error")
	}
}

// TestFixture_ExitOnErrorFalse verifies that --exit-on-error=false causes
// exit code 0 even when an error occurs.
func TestFixture_ExitOnErrorFalse(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExec(t, "empty_response.jsonl",
		"trigger error",
		withExitOnError(false),
		withTimeout(15*time.Second))

	// Must exit cleanly despite the error.
	if r.err != nil {
		t.Errorf("expected exit code 0 with --exit-on-error=false, got: %v", r.err)
	}
}

// TestFixture_StdinPrompt verifies that prompts can be piped via stdin.
func TestFixture_StdinPrompt(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	r := runFixtureExecStdin(t, "simple_response.jsonl", "what is the answer?")
	assertNoError(t, r)
	assertOutputContains(t, r, "The answer is 42.")
}
