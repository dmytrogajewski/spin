package shell

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// TestExecuteShellCommand_SuccessfulCommand tests successful command execution.
func TestExecuteShellCommand_SuccessfulCommand(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Test successful command
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
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Test command that doesn't exist
	_, err = integration.ExecuteShellCommand(context.Background(), "nonexistentcommand12345")
	if err == nil {
		t.Fatal("Expected error for nonexistent command, got nil")
	}

	errMsg := err.Error()
	t.Logf("Error message: %s", errMsg)

	// BUG TEST: The error should include helpful information
	// Currently, it just says "shell command failed" without explaining WHY
	if !strings.Contains(errMsg, "not found") && !strings.Contains(errMsg, "No such") {
		t.Error("BUG: Error message doesn't indicate command was not found")
		t.Errorf("Expected error to mention 'not found' or similar, got: %s", errMsg)
	}
}

// TestExecuteShellCommand_CommandWithStderr tests that stderr is captured.
// This reproduces the bug where stderr output is completely lost.
func TestExecuteShellCommand_CommandWithStderr(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Test command that writes to stderr and fails
	// Using a common error scenario: trying to list a non-existent directory
	_, err = integration.ExecuteShellCommand(context.Background(), "ls /nonexistent/directory/path12345")
	if err == nil {
		t.Fatal("Expected error for ls of non-existent directory, got nil")
	}

	errMsg := err.Error()
	t.Logf("Error message: %s", errMsg)

	// BUG TEST: The error should include the stderr output explaining the problem
	// Currently, stderr is discarded by using cmd.Output() instead of cmd.CombinedOutput()
	if !strings.Contains(strings.ToLower(errMsg), "no such file") &&
		!strings.Contains(strings.ToLower(errMsg), "cannot access") &&
		!strings.Contains(strings.ToLower(errMsg), "not found") {
		t.Error("BUG: Error message doesn't include stderr output")
		t.Errorf("Expected error to contain stderr info about missing file, got: %s", errMsg)
	}
}

// TestExecuteShellCommand_ExitCode tests that exit codes are reported.
func TestExecuteShellCommand_ExitCode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Test command that exits with non-zero status
	_, err = integration.ExecuteShellCommand(context.Background(), "exit 42")
	if err == nil {
		t.Fatal("Expected error for command with exit code 42, got nil")
	}

	errMsg := err.Error()
	t.Logf("Error message: %s", errMsg)

	// BUG TEST: The error should include the exit code
	// Currently, we lose this information
	if !strings.Contains(errMsg, "42") && !strings.Contains(errMsg, "exit") {
		t.Error("BUG: Error message doesn't include exit code information")
		t.Errorf("Expected error to mention exit code 42, got: %s", errMsg)
	}
}

// TestExecuteShellCommand_MissingTool tests a realistic scenario.
// This simulates the user's deb-to-rpm conversion issue where tools are missing.
func TestExecuteShellCommand_MissingTool(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !integration.IsEnabled() {
		t.Skip("Shell integration not enabled")
	}

	// Simulate the user's scenario: trying to use a tool that doesn't exist
	// This is what happened with dpkg/rpm/alien
	_, err = integration.ExecuteShellCommand(context.Background(), "alien --help")
	if err == nil {
		// If alien is installed, skip this test
		t.Skip("alien is installed, can't test missing tool scenario")
	}

	errMsg := err.Error()
	t.Logf("Error message for missing 'alien' tool: %s", errMsg)

	// BUG TEST: The error should clearly indicate that the tool is not found
	// This is critical for the agent to understand and suggest installing the tool
	if !strings.Contains(strings.ToLower(errMsg), "not found") &&
		!strings.Contains(strings.ToLower(errMsg), "no such") &&
		!strings.Contains(strings.ToLower(errMsg), "command not found") {
		t.Error("BUG: Error message doesn't clearly indicate missing tool")
		t.Errorf("Expected error to indicate 'alien' command not found, got: %s", errMsg)
		t.Error("Without this info, the agent cannot suggest installing the tool")
	}
}

// TestExecuteShellCommand_Disabled tests behavior when integration is disabled.
func TestExecuteShellCommand_Disabled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(false, "/tmp", logger)

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
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	integration := NewShellIntegration(true, "/tmp", logger)

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
			result := integration.IsShellCommand(tt.command)
			if result != tt.expected {
				t.Errorf("IsShellCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}
