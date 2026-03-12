package shell

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

// TestExecuteShellCommand_SuccessfulCommand tests successful command execution.
func TestExecuteShellCommand_SuccessfulCommand(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewContext(true, "/tmp", logger, 30*time.Second)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Test successful command.
	output, err := integration.ExecuteShellCommand(context.Background(), "echo 'hello world'")
	if err != nil {
		t.Errorf("ExecuteShellCommand failed: %v", err)
	}

	if !strings.Contains(output, "hello world") {
		t.Errorf("Expected output to contain 'hello world', got: %s", output)
	}
}

// TestExecuteShellCommand_FailedCommand tests command execution failure.
// This test reproduces the bug where stderr is lost and error messages are uninformative.
func TestExecuteShellCommand_FailedCommand(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewContext(true, "/tmp", logger, 30*time.Second)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Test command that doesn't exist.
	_, err = integration.ExecuteShellCommand(context.Background(), "nonexistentcommand12345")
	if err == nil {
		t.Fatal("Expected error for nonexistent command, got nil")
	}

	errMsg := err.Error()
	t.Logf("Error message: %s", errMsg)

	// BUG TEST: The error should include helpful information
	// Currently, it just says "shell command failed" without explaining WHY
	// Check for various languages' "not found" messages.
	notFoundPhrases := []string{"not found", "No such", "не найдена", "команда не найдена", "command not found"}
	foundAny := false

	for _, phrase := range notFoundPhrases {
		if strings.Contains(errMsg, phrase) {
			foundAny = true

			break
		}
	}

	if !foundAny {
		t.Error("BUG: Error message doesn't indicate command was not found")
		t.Errorf("Expected error to mention 'not found' or similar, got: %s", errMsg)
	}
}

// TestExecuteShellCommand_CommandWithStderr tests that stderr is captured.
// This reproduces the bug where stderr output is completely lost.
func TestExecuteShellCommand_CommandWithStderr(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewContext(true, "/tmp", logger, 30*time.Second)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Test command that writes to stderr and fails
	// Using a common error scenario: trying to list a non-existent directory.
	_, err = integration.ExecuteShellCommand(context.Background(), "ls /nonexistent/directory/path12345")
	if err == nil {
		t.Fatal("Expected error for ls of non-existent directory, got nil")
	}

	errMsg := err.Error()
	t.Logf("Error message: %s", errMsg)

	// BUG TEST: The error should include the stderr output explaining the problem
	// Currently, stderr is discarded by using cmd.Output() instead of cmd.CombinedOutput()
	// Check for various languages' "no such file" messages.
	noSuchFilePhrases := []string{"no such file", "cannot access", "not found", "Нет такого файла", "невозможно получить доступ"}
	foundAny := false

	for _, phrase := range noSuchFilePhrases {
		if strings.Contains(strings.ToLower(errMsg), strings.ToLower(phrase)) {
			foundAny = true

			break
		}
	}

	if !foundAny {
		t.Error("BUG: Error message doesn't include stderr output")
		t.Errorf("Expected error to contain stderr info about missing file, got: %s", errMsg)
	}
}

// TestExecuteShellCommand_ExitCode tests that exit codes are reported.
func TestExecuteShellCommand_ExitCode(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewContext(true, "/tmp", logger, 30*time.Second)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Test command that exits with non-zero status.
	_, err = integration.ExecuteShellCommand(context.Background(), "exit 42")
	if err == nil {
		t.Fatal("Expected error for command with exit code 42, got nil")
	}

	errMsg := err.Error()
	t.Logf("Error message: %s", errMsg)

	// BUG TEST: The error should include the exit code
	// Currently, we lose this information.
	if !strings.Contains(errMsg, "42") && !strings.Contains(errMsg, "exit") {
		t.Error("BUG: Error message doesn't include exit code information")
		t.Errorf("Expected error to mention exit code 42, got: %s", errMsg)
	}
}

// TestExecuteShellCommand_MissingTool tests a realistic scenario.
// This simulates the user's deb-to-rpm conversion issue where tools are missing.
func TestExecuteShellCommand_MissingTool(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewContext(true, "/tmp", logger, 30*time.Second)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Simulate the user's scenario: trying to use a tool that doesn't exist
	// This is what happened with dpkg/rpm/alien.
	_, err = integration.ExecuteShellCommand(context.Background(), "alien --help")
	if err == nil {
		// If alien is installed, skip this test.
		t.Skip("alien is installed, can't test missing tool scenario")
	}

	errMsg := err.Error()
	t.Logf("Error message for missing 'alien' tool: %s", errMsg)

	// BUG TEST: The error should clearly indicate that the tool is not found
	// This is critical for the agent to understand and suggest installing the tool
	// Check for various languages' "not found" messages.
	notFoundPhrases := []string{"not found", "no such", "command not found", "не найдена", "команда не найдена"}
	foundAny := false

	for _, phrase := range notFoundPhrases {
		if strings.Contains(strings.ToLower(errMsg), strings.ToLower(phrase)) {
			foundAny = true

			break
		}
	}

	if !foundAny {
		t.Error("BUG: Error message doesn't clearly indicate missing tool")
		t.Errorf("Expected error to indicate 'alien' command not found, got: %s", errMsg)
		t.Error("Without this info, the agent cannot suggest installing the tool")
	}
}

// TestExecuteShellCommand_Disabled tests behavior when integration is disabled.
func TestExecuteShellCommand_Disabled(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewContext(false, "/tmp", logger, 30*time.Second)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	_, err = integration.ExecuteShellCommand(context.Background(), "echo test")
	if err == nil {
		t.Error("Expected error when integration is disabled")
	}

	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("Expected error to mention disabled, got: %s", err.Error())
	}
}

// TestIsShellCommand tests shell command detection.
func TestIsShellCommand(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewContext(true, "/tmp", logger, 30*time.Second)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"pipe", "ls | grep test", true},
		{"redirect", "echo test > file", true},
		{"variable", "echo $HOME", true},
		{"cd", "cd /tmp", true},
		{"export", "export VAR=value", true},
		{"simple", "ls", false},
		{"with args", "ls -la /tmp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := integration.IsShellCommand(tt.command)
			if result != tt.expected {
				t.Errorf("IsShellCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

// TestExecuteShellCommand_Timeout tests shell command timeout functionality.
func TestExecuteShellCommand_Timeout(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test with a very short timeout (1 second).
	integration := NewContext(true, "/tmp", logger, 1*time.Second)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Test command that will take longer than 1 second
	// Using sleep command that should timeout.
	_, err = integration.ExecuteShellCommand(context.Background(), "sleep 2")
	if err == nil {
		t.Fatal("Expected timeout error for sleep command, got nil")
	}

	errMsg := err.Error()
	t.Logf("Timeout error message: %s", errMsg)

	// Verify the error indicates timeout.
	if !strings.Contains(errMsg, "timed out") && !strings.Contains(errMsg, "timeout") {
		t.Errorf("Expected timeout error message, got: %s", errMsg)
	}
}

// TestExecuteShellCommand_CustomTimeout tests shell command with custom timeout.
func TestExecuteShellCommand_CustomTimeout(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test with a custom timeout (5 seconds).
	customTimeout := 5 * time.Second
	integration := NewContext(true, "/tmp", logger, customTimeout)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Test command that completes within the timeout.
	output, err := integration.ExecuteShellCommand(context.Background(), "echo 'custom timeout test'")
	if err != nil {
		t.Errorf("ExecuteShellCommand failed: %v", err)
	}

	if !strings.Contains(output, "custom timeout test") {
		t.Errorf("Expected output to contain 'custom timeout test', got: %s", output)
	}

	// Test command that exceeds the timeout.
	_, err = integration.ExecuteShellCommand(context.Background(), "sleep 6")
	if err == nil {
		t.Fatal("Expected timeout error for sleep command exceeding custom timeout, got nil")
	}

	errMsg := err.Error()
	t.Logf("Custom timeout error message: %s", errMsg)

	// Verify the error indicates timeout and mentions the custom timeout duration.
	if !strings.Contains(errMsg, "timed out") && !strings.Contains(errMsg, "timeout") {
		t.Errorf("Expected timeout error message, got: %s", errMsg)
	}
}

// TestExecuteShellCommand_ContextTimeout tests shell command with context timeout.
func TestExecuteShellCommand_ContextTimeout(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Integration with longer timeout than context.
	integration := NewContext(true, "/tmp", logger, 10*time.Second)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Create a context with shorter timeout than integration timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test command that will exceed context timeout but not integration timeout.
	_, err = integration.ExecuteShellCommand(ctx, "sleep 2")
	if err == nil {
		t.Fatal("Expected context timeout error, got nil")
	}

	errMsg := err.Error()
	t.Logf("Context timeout error message: %s", errMsg)

	// Should be context timeout, not integration timeout.
	if !strings.Contains(errMsg, "context deadline exceeded") &&
		!strings.Contains(errMsg, "deadline exceeded") &&
		!strings.Contains(errMsg, "timed out") {
		t.Errorf("Expected context timeout error, got: %s", errMsg)
	}
}

// TestExecuteShellCommand_ZeroTimeout tests shell command with zero timeout.
func TestExecuteShellCommand_ZeroTimeout(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test with zero timeout (should fail validation).
	integration := NewContext(true, "/tmp", logger, 0)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Even with zero timeout, the command should still execute
	// because we use context.WithTimeout which handles zero duration
	// However, zero timeout means immediate timeout, so we expect it to fail.
	_, err = integration.ExecuteShellCommand(context.Background(), "echo 'zero timeout test'")
	if err == nil {
		t.Error("Expected timeout error for zero timeout, got nil")
	}

	errMsg := err.Error()
	t.Logf("Zero timeout error message: %s", errMsg)

	// Should be a timeout error.
	if !strings.Contains(errMsg, "timed out") && !strings.Contains(errMsg, "timeout") {
		t.Errorf("Expected timeout error for zero timeout, got: %s", errMsg)
	}
}
