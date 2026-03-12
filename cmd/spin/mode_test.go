package main

import (
	"strings"
	"testing"
)

func TestNewModeCmd(t *testing.T) {
	t.Parallel()

	cmd := newModeCmd()

	if cmd.Use != "mode [command]" {
		t.Errorf("expected Use to be 'mode [command]', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}

	// Check that subcommands are registered.
	if len(cmd.Commands()) != 2 {
		t.Errorf("expected 2 subcommands, got %d", len(cmd.Commands()))
	}

	// Verify subcommand names.
	subcommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		subcommands[subcmd.Use] = true
	}

	if !subcommands["list"] {
		t.Error("expected 'list' subcommand to be registered")
	}

	if !subcommands["describe <mode-name>"] {
		t.Error("expected 'describe' subcommand to be registered")
	}
}

func TestRunModeList(t *testing.T) {
	t.Parallel()

	cmd := newModeListCmd()

	// Execute command - it will print to stdout.
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Since we can't easily capture fmt.Println output in tests,
	// we'll test the command runs without error.
	// The actual output was verified manually above.
}

// verifyModeInfo validates that a mode's info is properly structured.
func verifyModeInfo(t *testing.T, modeName string, info modeInfo) {
	t.Helper()

	if info.name != modeName {
		t.Errorf("mode info has wrong name: got %s, want %s", info.name, modeName)
	}

	if info.description == "" {
		t.Error("mode info has empty description")
	}

	if info.maxTokens <= 0 {
		t.Error("mode info has invalid maxTokens")
	}

	if len(info.tools) == 0 {
		t.Error("mode info has no tools")
	}

	if len(info.bestFor) == 0 {
		t.Error("mode info has no bestFor cases")
	}
}

func TestRunModeDescribe_ValidMode(t *testing.T) {
	t.Parallel()

	modes := []string{"regular", "review", "compact", "planning"}

	for _, modeName := range modes {
		t.Run(modeName, func(t *testing.T) {
			t.Parallel()

			cmd := newModeDescribeCmd()
			cmd.SetArgs([]string{modeName})

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error for mode '%s': %v", modeName, err)
			}

			verifyModeInfo(t, modeName, allModes[modeName])
		})
	}
}

func TestRunModeDescribe_InvalidMode(t *testing.T) {
	t.Parallel()

	cmd := newModeDescribeCmd()
	cmd.SetArgs([]string{"invalid-mode"})

	// Execute command - should error.
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid mode, got nil")
	}

	// Verify error message.
	if !strings.Contains(err.Error(), "unknown mode") {
		t.Errorf("expected error to mention 'unknown mode', got: %v", err)
	}

	if !strings.Contains(err.Error(), "invalid-mode") {
		t.Errorf("expected error to mention the invalid mode name, got: %v", err)
	}

	// Verify helpful message about valid modes.
	if !strings.Contains(err.Error(), "valid modes:") {
		t.Errorf("expected error to list valid modes, got: %v", err)
	}
}

func TestRunModeDescribe_NoArgument(t *testing.T) {
	t.Parallel()

	cmd := newModeDescribeCmd()

	// Execute command with no args - should error.
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no argument provided, got nil")
	}

	// cobra should handle this with "requires 1 arg" or similar.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "arg") && !strings.Contains(errMsg, "argument") {
		t.Errorf("expected error about missing argument, got: %v", err)
	}
}

func TestAllModesInfo_Consistency(t *testing.T) {
	t.Parallel()

	// Verify all expected modes are present.
	expectedModes := []string{"regular", "review", "compact", "planning"}
	for _, modeName := range expectedModes {
		info, exists := allModes[modeName]
		if !exists {
			t.Errorf("expected mode '%s' to exist in allModes", modeName)

			continue
		}

		// Verify mode info is complete.
		if info.name != modeName {
			t.Errorf("mode '%s' has inconsistent name field: %s", modeName, info.name)
		}

		if info.description == "" {
			t.Errorf("mode '%s' has empty description", modeName)
		}

		if info.maxTokens <= 0 {
			t.Errorf("mode '%s' has invalid maxTokens: %d", modeName, info.maxTokens)
		}

		if len(info.tools) == 0 {
			t.Errorf("mode '%s' has no tools", modeName)
		}

		if len(info.bestFor) == 0 {
			t.Errorf("mode '%s' has no bestFor cases", modeName)
		}
	}
}

func TestModeInfo_TokenBudgets(t *testing.T) {
	t.Parallel()

	// Verify token budgets match documented values.
	expectedBudgets := map[string]int{
		"regular":  16384,
		"review":   12288,
		"compact":  4096,
		"planning": 4096,
	}

	for mode, expectedTokens := range expectedBudgets {
		info := allModes[mode]
		if info.maxTokens != expectedTokens {
			t.Errorf("mode '%s' has incorrect token budget: got %d, want %d",
				mode, info.maxTokens, expectedTokens)
		}
	}
}

func TestModeInfo_ToolCounts(t *testing.T) {
	t.Parallel()

	// Verify tool counts for each mode.
	expectedToolCounts := map[string]int{
		"regular":  8, // all tools.
		"review":   5, // read-only tools.
		"compact":  3, // minimal tools.
		"planning": 3, // context tools.
	}

	for mode, expectedCount := range expectedToolCounts {
		info := allModes[mode]
		if len(info.tools) != expectedCount {
			t.Errorf("mode '%s' has incorrect tool count: got %d, want %d",
				mode, len(info.tools), expectedCount)
		}
	}
}

func TestModeInfo_RegularModeHasAllTools(t *testing.T) {
	t.Parallel()

	info := allModes["regular"]

	// Verify regular mode has all expected tools.
	expectedTools := []string{
		"read_file",
		"write_file",
		"list_directory",
		"execute_command",
		"get_context",
		"file_search",
		"apply_patch",
		"git_context",
	}

	toolSet := make(map[string]bool)
	for _, tool := range info.tools {
		toolSet[tool] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolSet[expectedTool] {
			t.Errorf("regular mode missing expected tool: %s", expectedTool)
		}
	}
}

func TestModeInfo_ReviewModeIsReadOnly(t *testing.T) {
	t.Parallel()

	info := allModes["review"]

	// Verify review mode does not have write tools.
	prohibitedTools := []string{"write_file", "execute_command", "apply_patch"}

	for _, tool := range info.tools {
		for _, prohibited := range prohibitedTools {
			if tool == prohibited {
				t.Errorf("review mode should not have write tool: %s", tool)
			}
		}
	}

	// Verify review mode has read tools.
	requiredTools := []string{"read_file", "get_context"}

	toolSet := make(map[string]bool)
	for _, tool := range info.tools {
		toolSet[tool] = true
	}

	for _, required := range requiredTools {
		if !toolSet[required] {
			t.Errorf("review mode missing required read tool: %s", required)
		}
	}
}

func TestModeInfo_CompactModeIsMinimal(t *testing.T) {
	t.Parallel()

	info := allModes["compact"]

	// Compact mode should have exactly 3 tools.
	if len(info.tools) != 3 {
		t.Errorf("compact mode should have exactly 3 tools, got %d", len(info.tools))
	}

	// Verify compact mode has essential tools.
	requiredTools := []string{"read_file", "get_context", "file_search"}

	toolSet := make(map[string]bool)
	for _, tool := range info.tools {
		toolSet[tool] = true
	}

	for _, required := range requiredTools {
		if !toolSet[required] {
			t.Errorf("compact mode missing essential tool: %s", required)
		}
	}
}

func TestModeInfo_PlanningModeIsContextOnly(t *testing.T) {
	t.Parallel()

	info := allModes["planning"]

	// Planning mode should have exactly 3 context tools.
	if len(info.tools) != 3 {
		t.Errorf("planning mode should have exactly 3 tools, got %d", len(info.tools))
	}

	// Verify planning mode has only context tools.
	expectedTools := []string{"get_context", "file_search", "git_context"}

	toolSet := make(map[string]bool)
	for _, tool := range info.tools {
		toolSet[tool] = true
	}

	for _, expected := range expectedTools {
		if !toolSet[expected] {
			t.Errorf("planning mode missing context tool: %s", expected)
		}
	}

	// Verify no file operations.
	prohibitedTools := []string{"read_file", "write_file", "list_directory"}
	for _, tool := range info.tools {
		for _, prohibited := range prohibitedTools {
			if tool == prohibited {
				t.Errorf("planning mode should not have file operation tool: %s", tool)
			}
		}
	}
}
