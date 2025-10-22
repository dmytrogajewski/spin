package shell

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// TestShellOperationTool_ExecuteCommand_Success tests successful command execution.
func TestShellOperationTool_ExecuteCommand_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)
	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	tool := NewShellOperationTool(integration)

	params := map[string]interface{}{
		"operation": "execute_command",
		"command":   "echo hello",
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !strings.Contains(result.Output, "hello") {
		t.Errorf("Expected output to contain 'hello', got: %s", result.Output)
	}
}

// TestShellOperationTool_ExecuteCommand_Failure tests command execution failure.
// This reproduces the bug where the error message doesn't include:
// 1. The command that failed
// 2. The stderr output explaining why it failed
func TestShellOperationTool_ExecuteCommand_Failure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)
	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	tool := NewShellOperationTool(integration)

	// Test with a command that doesn't exist
	params := map[string]interface{}{
		"operation": "execute_command",
		"command":   "nonexistentcommand12345",
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for nonexistent command")
	}

	t.Logf("Error from tool: %s", result.Error)

	// BUG TEST #1: The error should include the command name
	// Currently it just says "Failed to execute command: ..." without saying WHICH command
	if !strings.Contains(result.Error, "nonexistentcommand12345") {
		t.Error("BUG: Error message doesn't include the command that failed")
		t.Errorf("Expected error to mention 'nonexistentcommand12345', got: %s", result.Error)
		t.Error("This makes debugging impossible for users and agents")
	}

	// BUG TEST #2: The error should include "not found" or similar stderr output
	if !strings.Contains(strings.ToLower(result.Error), "not found") &&
		!strings.Contains(strings.ToLower(result.Error), "no such") {
		t.Error("BUG: Error message doesn't include stderr output explaining the failure")
		t.Errorf("Expected error to explain why command failed, got: %s", result.Error)
	}
}

// TestShellOperationTool_MissingToolScenario reproduces the exact user scenario.
// User tries to convert deb to rpm, but alien/dpkg/rpm tools are not installed.
func TestShellOperationTool_MissingToolScenario(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)
	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	tool := NewShellOperationTool(integration)

	// Simulate trying to use dpkg (likely not installed on RPM-based systems)
	params := map[string]interface{}{
		"operation": "execute_command",
		"command":   "dpkg --version",
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	// If dpkg is installed, skip
	if result.Success {
		t.Skip("dpkg is installed, can't reproduce missing tool scenario")
	}

	t.Logf("Error from tool when dpkg missing: %s", result.Error)

	// BUG TEST: The error message should make it OBVIOUS that dpkg is not installed
	// This is critical for the agent to understand it needs to suggest:
	// 1. Installing dpkg, OR
	// 2. Using an alternative approach
	expectedPhrases := []string{"not found", "no such", "command not found"}
	foundAny := false
	for _, phrase := range expectedPhrases {
		if strings.Contains(strings.ToLower(result.Error), phrase) {
			foundAny = true
			break
		}
	}

	if !foundAny {
		t.Error("BUG: Error message doesn't clearly indicate dpkg is missing")
		t.Errorf("Expected error to contain one of %v, got: %s", expectedPhrases, result.Error)
		t.Error("In the user's logs, this appears as:")
		t.Error(`  error="Failed to execute command: shell`)
		t.Error("Which is completely useless for debugging")
	}

	// The error should also include the command name
	if !strings.Contains(result.Error, "dpkg") {
		t.Error("BUG: Error doesn't mention which command failed (dpkg)")
	}
}

// TestShellOperationTool_GetEnvironment tests environment variable retrieval.
func TestShellOperationTool_GetEnvironment(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)
	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	tool := NewShellOperationTool(integration)

	params := map[string]interface{}{
		"operation": "get_environment",
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !strings.Contains(result.Output, "Environment Variables") {
		t.Errorf("Expected output to contain 'Environment Variables', got: %s", result.Output)
	}
}

// TestShellOperationTool_MissingOperation tests missing operation parameter.
func TestShellOperationTool_MissingOperation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)
	tool := NewShellOperationTool(integration)

	params := map[string]interface{}{}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for missing operation parameter")
	}

	if !strings.Contains(result.Error, "operation parameter is required") {
		t.Errorf("Expected error about missing operation, got: %s", result.Error)
	}
}

// TestShellOperationTool_MissingCommand tests missing command parameter.
func TestShellOperationTool_MissingCommand(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)
	tool := NewShellOperationTool(integration)

	params := map[string]interface{}{
		"operation": "execute_command",
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for missing command parameter")
	}

	if !strings.Contains(result.Error, "command is required") {
		t.Errorf("Expected error about missing command, got: %s", result.Error)
	}
}
