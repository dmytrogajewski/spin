package main

import (
	"testing"
)

func TestNewDebugCmd(t *testing.T) {
	cmd := newDebugCmd()
	if cmd == nil {
		t.Errorf("newDebugCmd() returned nil")
	}

	if cmd.Use != "debug" {
		t.Errorf("newDebugCmd().Use = %v, want %v", cmd.Use, "debug")
	}

	if cmd.Short != "Debug and development utilities" {
		t.Errorf("newDebugCmd().Short = %v, want %v", cmd.Short, "Debug and development utilities")
	}
}

func TestNewDebugEventsCmd(t *testing.T) {
	cmd := newDebugEventsCmd()
	if cmd == nil {
		t.Errorf("newDebugEventsCmd() returned nil")
	}

	if cmd.Use != "events <prompt>" {
		t.Errorf("newDebugEventsCmd().Use = %v, want %v", cmd.Use, "events <prompt>")
	}

	if cmd.Short != "Execute a task and log all core events" {
		t.Errorf("newDebugEventsCmd().Short = %v, want %v", cmd.Short, "Execute a task and log all core events")
	}
}

func TestNewDebugSandboxCmd(t *testing.T) {
	cmd := newDebugSandboxCmd()
	if cmd == nil {
		t.Errorf("newDebugSandboxCmd() returned nil")
	}

	if cmd.Use != "sandbox <command>" {
		t.Errorf("newDebugSandboxCmd().Use = %v, want %v", cmd.Use, "sandbox <command>")
	}

	if cmd.Short != "Execute a command in a sandboxed environment" {
		t.Errorf("newDebugSandboxCmd().Short = %v, want %v", cmd.Short, "Execute a command in a sandboxed environment")
	}
}

func TestNewDebugLandlockCmd(t *testing.T) {
	cmd := newDebugLandlockCmd()
	if cmd == nil {
		t.Errorf("newDebugLandlockCmd() returned nil")
	}

	if cmd.Use != "landlock <command>" {
		t.Errorf("newDebugLandlockCmd().Use = %v, want %v", cmd.Use, "landlock <command>")
	}

	if cmd.Short != "Execute a command with Landlock restrictions" {
		t.Errorf("newDebugLandlockCmd().Short = %v, want %v", cmd.Short, "Execute a command with Landlock restrictions")
	}
}

func TestDebugCmdSubcommands(t *testing.T) {
	cmd := newDebugCmd()

	// Test that all expected subcommands exist.
	expectedSubcommands := []string{
		"events",
		"sandbox",
		"landlock",
	}

	subcommands := cmd.Commands()
	if len(subcommands) != len(expectedSubcommands) {
		t.Errorf("Debug command has %d subcommands, want %d", len(subcommands), len(expectedSubcommands))
	}

	for _, expected := range expectedSubcommands {
		found := false

		for _, subcmd := range subcommands {
			if subcmd.Name() == expected {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("Debug subcommand %s not found", expected)
		}
	}
}

func TestDebugCmdHelp(t *testing.T) {
	cmd := newDebugCmd()

	// Test that help text is properly set.
	if cmd.Long == "" {
		t.Errorf("Debug command Long description is empty")
	}

	if cmd.Short == "" {
		t.Errorf("Debug command Short description is empty")
	}
}

func TestDebugEventsCmdFlags(t *testing.T) {
	cmd := newDebugEventsCmd()

	// Test that expected flags exist.
	expectedFlags := []string{
		"format",
		"filter",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Flag %s not found", flagName)
		}
	}
}

func TestDebugEventsCmdDefaultValues(t *testing.T) {
	cmd := newDebugEventsCmd()

	// Test default values.
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil || formatFlag.DefValue != "text" {
		t.Errorf("format flag default = %v, want %v", formatFlag.DefValue, "text")
	}
}

func TestDebugEventsCmdExamples(t *testing.T) {
	cmd := newDebugEventsCmd()

	// Test that examples are included in help text.
	helpText := cmd.Example
	expectedExamples := []string{
		"spin debug events \"list files in current directory\"",
		"spin debug events --filter tool \"run tests\"",
		"spin debug events --format json \"build project\"",
	}

	for _, example := range expectedExamples {
		if !contains(helpText, example) {
			t.Errorf("Help text missing example: %s", example)
		}
	}
}

func TestDebugSandboxCmdFlags(t *testing.T) {
	cmd := newDebugSandboxCmd()

	// Test that expected flags exist.
	expectedFlags := []string{
		"read-only",
		"network",
		"timeout",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Flag %s not found", flagName)
		}
	}
}

func TestDebugSandboxCmdDefaultValues(t *testing.T) {
	cmd := newDebugSandboxCmd()

	// Test default values.
	readOnlyFlag := cmd.Flags().Lookup("read-only")
	if readOnlyFlag == nil || readOnlyFlag.DefValue != "true" {
		t.Errorf("read-only flag default = %v, want %v", readOnlyFlag.DefValue, "true")
	}

	networkFlag := cmd.Flags().Lookup("network")
	if networkFlag == nil || networkFlag.DefValue != "false" {
		t.Errorf("network flag default = %v, want %v", networkFlag.DefValue, "false")
	}

	timeoutFlag := cmd.Flags().Lookup("timeout")
	if timeoutFlag == nil || timeoutFlag.DefValue != "30s" {
		t.Errorf("timeout flag default = %v, want %v", timeoutFlag.DefValue, "30s")
	}
}

func TestDebugLandlockCmdFlags(t *testing.T) {
	cmd := newDebugLandlockCmd()

	// Test that expected flags exist.
	expectedFlags := []string{
		"allow-read",
		"allow-write",
		"timeout",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Flag %s not found", flagName)
		}
	}
}

func TestDebugLandlockCmdDefaultValues(t *testing.T) {
	cmd := newDebugLandlockCmd()

	// Test default values.
	allowReadFlag := cmd.Flags().Lookup("allow-read")
	if allowReadFlag == nil || allowReadFlag.DefValue != "true" {
		t.Errorf("allow-read flag default = %v, want %v", allowReadFlag.DefValue, "true")
	}

	allowWriteFlag := cmd.Flags().Lookup("allow-write")
	if allowWriteFlag == nil || allowWriteFlag.DefValue != "false" {
		t.Errorf("allow-write flag default = %v, want %v", allowWriteFlag.DefValue, "false")
	}

	timeoutFlag := cmd.Flags().Lookup("timeout")
	if timeoutFlag == nil || timeoutFlag.DefValue != "30s" {
		t.Errorf("timeout flag default = %v, want %v", timeoutFlag.DefValue, "30s")
	}
}

func TestDebugSandboxCmdExamples(t *testing.T) {
	cmd := newDebugSandboxCmd()

	// Test that examples are included in help text.
	helpText := cmd.Example
	expectedExamples := []string{
		"spin debug sandbox \"ls -la\"",
		"spin debug sandbox --network \"curl https://example.com\"",
		"spin debug sandbox --read-only=false \"touch test.txt\"",
	}

	for _, example := range expectedExamples {
		if !contains(helpText, example) {
			t.Errorf("Help text missing example: %s", example)
		}
	}
}

func TestDebugLandlockCmdExamples(t *testing.T) {
	cmd := newDebugLandlockCmd()

	// Test that examples are included in help text.
	helpText := cmd.Example
	expectedExamples := []string{
		"spin debug landlock \"ls -la\"",
		"spin debug landlock --allow-write \"touch test.txt\"",
		"spin debug landlock --allow-read=false \"cat file.txt\"",
	}

	for _, example := range expectedExamples {
		if !contains(helpText, example) {
			t.Errorf("Help text missing example: %s", example)
		}
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
