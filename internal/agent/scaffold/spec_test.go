package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// Journey: specs/journeys/JOURNEY-1.2.md.

// TestSpec_HasTool_Present tests that HasTool returns true for a registered tool.
// Kills mutant: always returning false would make this test fail.
func TestSpec_HasTool_Present(t *testing.T) {
	t.Parallel()

	spec := &Spec{
		ToolSchemas: []tools.ToolSchema{
			{
				Type: "function",
				Function: tools.FunctionSchema{
					Name:        "read_file",
					Description: "Read a file",
				},
			},
		},
	}

	assert.True(t, spec.HasTool("read_file"))
}

// TestSpec_HasTool_Absent tests that HasTool returns false for an unregistered tool.
// Kills mutant: always returning true would make this test fail.
func TestSpec_HasTool_Absent(t *testing.T) {
	t.Parallel()

	spec := &Spec{
		ToolSchemas: []tools.ToolSchema{
			{
				Type: "function",
				Function: tools.FunctionSchema{
					Name:        "read_file",
					Description: "Read a file",
				},
			},
		},
	}

	assert.False(t, spec.HasTool("write_file"))
}

// TestSpec_HasTool_EmptySchemas tests that HasTool returns false with no schemas.
// Kills mutant: panicking on empty slice would make this test fail.
func TestSpec_HasTool_EmptySchemas(t *testing.T) {
	t.Parallel()

	spec := &Spec{}
	assert.False(t, spec.HasTool("any_tool"))
}

// TestSpec_ToolNames tests that ToolNames returns all tool names.
// Kills mutant: returning empty slice would make this test fail.
func TestSpec_ToolNames(t *testing.T) {
	t.Parallel()

	spec := &Spec{
		ToolSchemas: []tools.ToolSchema{
			{Function: tools.FunctionSchema{Name: "read_file"}},
			{Function: tools.FunctionSchema{Name: "write_file"}},
			{Function: tools.FunctionSchema{Name: "shell_command"}},
		},
	}

	names := spec.ToolNames()
	assert.Equal(t, []string{"read_file", "write_file", "shell_command"}, names)
}

// TestSpec_ToolNames_Empty tests that ToolNames returns empty slice with no schemas.
func TestSpec_ToolNames_Empty(t *testing.T) {
	t.Parallel()

	spec := &Spec{}
	names := spec.ToolNames()
	assert.Empty(t, names)
}
