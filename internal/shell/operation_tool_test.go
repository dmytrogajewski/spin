package shell

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

// TestShellOperationTool_ExecuteCommand_Success tests successful command execution.
func TestShellOperationTool_ExecuteCommand_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger, 30*time.Second)
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
	integration := NewShellIntegration(true, "/tmp", logger, 30*time.Second)
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
	// Check for various languages' "not found" messages
	notFoundPhrases := []string{"not found", "no such", "не найдена", "команда не найдена", "command not found"}
	foundAny := false
	for _, phrase := range notFoundPhrases {
		if strings.Contains(strings.ToLower(result.Error), strings.ToLower(phrase)) {
			foundAny = true
			break
		}
	}
	if !foundAny {
		t.Error("BUG: Error message doesn't include stderr output explaining the failure")
		t.Errorf("Expected error to explain why command failed, got: %s", result.Error)
	}
}

// TestShellOperationTool_MissingToolScenario reproduces the exact user scenario.
// User tries to convert deb to rpm, but alien/dpkg/rpm tools are not installed.
func TestShellOperationTool_MissingToolScenario(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger, 30*time.Second)
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
	expectedPhrases := []string{"not found", "no such", "command not found", "не найдена", "команда не найдена"}
	foundAny := false
	for _, phrase := range expectedPhrases {
		if strings.Contains(strings.ToLower(result.Error), strings.ToLower(phrase)) {
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
	integration := NewShellIntegration(true, "/tmp", logger, 30*time.Second)
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
	integration := NewShellIntegration(true, "/tmp", logger, 30*time.Second)
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
	integration := NewShellIntegration(true, "/tmp", logger, 30*time.Second)
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

// TestShellOperationTool_ExecuteCommand_Timeout tests shell operation tool timeout functionality.
func TestShellOperationTool_ExecuteCommand_Timeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger, 30*time.Second)
	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	tool := NewShellOperationTool(integration)

	// Test with custom timeout parameter
	params := map[string]interface{}{
		"operation": "execute_command",
		"command":   "sleep 2",
		"timeout":   1.0, // 1 second timeout
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for command that exceeds timeout")
	}

	t.Logf("Timeout error from tool: %s", result.Error)

	// Verify the error indicates timeout
	if !strings.Contains(result.Error, "timed out") && !strings.Contains(result.Error, "timeout") {
		t.Errorf("Expected timeout error message, got: %s", result.Error)
	}
}

// TestShellOperationTool_ExecuteCommand_CustomTimeout tests shell operation tool with custom timeout.
func TestShellOperationTool_ExecuteCommand_CustomTimeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger, 30*time.Second)
	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	tool := NewShellOperationTool(integration)

	// Test with longer timeout that should succeed
	params := map[string]interface{}{
		"operation": "execute_command",
		"command":   "echo 'custom timeout test'",
		"timeout":   5.0, // 5 second timeout
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success for command within timeout, got failure: %s", result.Error)
	}

	if !strings.Contains(result.Output, "custom timeout test") {
		t.Errorf("Expected output to contain 'custom timeout test', got: %s", result.Output)
	}
}

// TestShellOperationTool_ExecuteCommand_InvalidTimeout tests shell operation tool with invalid timeout.
func TestShellOperationTool_ExecuteCommand_InvalidTimeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger, 30*time.Second)
	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	tool := NewShellOperationTool(integration)

	// Test with invalid timeout parameter (string instead of number)
	params := map[string]interface{}{
		"operation": "execute_command",
		"command":   "echo 'invalid timeout test'",
		"timeout":   "invalid", // Invalid timeout type
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	// Should still succeed because invalid timeout falls back to default
	if !result.Success {
		t.Errorf("Expected success for command with invalid timeout (should fallback to default), got failure: %s", result.Error)
	}

	if !strings.Contains(result.Output, "invalid timeout test") {
		t.Errorf("Expected output to contain 'invalid timeout test', got: %s", result.Output)
	}
}

// TestShellOperationTool_ExecuteCommand_ZeroTimeout tests shell operation tool with zero timeout.
func TestShellOperationTool_ExecuteCommand_ZeroTimeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger, 30*time.Second)
	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	tool := NewShellOperationTool(integration)

	// Test with zero timeout
	params := map[string]interface{}{
		"operation": "execute_command",
		"command":   "echo 'zero timeout test'",
		"timeout":   0.0, // Zero timeout
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	// Zero timeout should cause immediate timeout
	if result.Success {
		t.Error("Expected failure for command with zero timeout (should timeout immediately)")
	}

	// Should be a timeout error
	if !strings.Contains(result.Error, "timed out") && !strings.Contains(result.Error, "timeout") {
		t.Errorf("Expected timeout error for zero timeout, got: %s", result.Error)
	}
}
