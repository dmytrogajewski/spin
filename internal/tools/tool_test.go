package tools

import (
	"testing"
)

// TestBuiltinTools_Count verifies that we have exactly 8 builtin tools.
func TestBuiltinTools_Count(t *testing.T) {
	if len(BuiltinTools) != 8 {
		t.Errorf("BuiltinTools count = %d, want 8", len(BuiltinTools))
	}
}

// TestBuiltinTools_Names verifies that all expected builtin tools are present.
func TestBuiltinTools_Names(t *testing.T) {
	expected := map[string]bool{
		"read_file":      false,
		"write_file":     false,
		"list_directory": false,
		"shell_command":  false,
		"get_context":    false,
		"apply_patch":    false,
		"file_search":    false,
		"git_context":    false,
	}

	for _, tool := range BuiltinTools {
		name := tool.Name()
		if _, exists := expected[name]; !exists {
			t.Errorf("unexpected builtin tool: %s", name)
			continue
		}
		expected[name] = true
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing builtin tool: %s", name)
		}
	}
}

// TestBuiltinTools_NonNil verifies that all builtin tools are non-nil.
func TestBuiltinTools_NonNil(t *testing.T) {
	for i, tool := range BuiltinTools {
		if tool == nil {
			t.Errorf("BuiltinTools[%d] is nil", i)
		}
	}
}
