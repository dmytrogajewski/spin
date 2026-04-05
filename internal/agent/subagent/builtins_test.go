package subagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Journey: specs/journeys/JOURNEY-1.3.md.

// TestBuiltins_Count tests that Builtins returns all Phase 1 subagent specs.
// Kills mutant: removing a builtin would make this test fail.
func TestBuiltins_Count(t *testing.T) {
	t.Parallel()

	builtins := Builtins()

	// Phase 1 has 4 builtins: explorer, planner, reviewer, ask_user.
	require.Len(t, builtins, 4)
}

// TestBuiltins_ExplorerExcludesWriteTools tests that the explorer spec
// does not include write_file or shell_command.
// Kills mutant: adding write tools to explorer would make this test fail.
func TestBuiltins_ExplorerExcludesWriteTools(t *testing.T) {
	t.Parallel()

	builtins := Builtins()

	var explorer *Spec

	for _, s := range builtins {
		if s.Name == NameExplorer {
			explorer = s

			break
		}
	}

	require.NotNil(t, explorer, "explorer spec should exist in builtins")
	assert.False(t, explorer.HasTool("write_file"), "explorer should not have write_file")
	assert.False(t, explorer.HasTool("shell_command"), "explorer should not have shell_command")
	assert.False(t, explorer.HasTool("apply_patch"), "explorer should not have apply_patch")
	assert.True(t, explorer.HasTool("read_file"), "explorer should have read_file")
}

// TestBuiltins_PlannerExcludesShellCommand tests that the planner spec
// does not include shell_command (structural safety).
// Kills mutant: adding shell_command to planner would make this test fail.
func TestBuiltins_PlannerExcludesShellCommand(t *testing.T) {
	t.Parallel()

	builtins := Builtins()

	var planner *Spec

	for _, s := range builtins {
		if s.Name == NamePlanner {
			planner = s

			break
		}
	}

	require.NotNil(t, planner, "planner spec should exist in builtins")
	assert.False(t, planner.HasTool("shell_command"), "planner should not have shell_command")
	assert.False(t, planner.HasTool("write_file"), "planner should not have write_file")
	assert.True(t, planner.HasTool("read_file"), "planner should have read_file")
}

// TestBuiltins_AskUserHasOnlyAskUser tests that the ask_user spec has
// only the ask_user tool.
// Kills mutant: adding extra tools to ask_user would make this test fail.
func TestBuiltins_AskUserHasOnlyAskUser(t *testing.T) {
	t.Parallel()

	builtins := Builtins()

	var askUser *Spec

	for _, s := range builtins {
		if s.Name == NameAskUser {
			askUser = s

			break
		}
	}

	require.NotNil(t, askUser, "ask_user spec should exist in builtins")
	require.Len(t, askUser.AllowedTools, 1, "ask_user should have exactly 1 tool")
	assert.Equal(t, "ask_user", askUser.AllowedTools[0])
}

// TestBuiltins_AllHaveMaxIterations tests that all builtins have a positive MaxIterations.
// Kills mutant: setting MaxIterations to 0 would make this test fail.
func TestBuiltins_AllHaveMaxIterations(t *testing.T) {
	t.Parallel()

	for _, spec := range Builtins() {
		assert.Positive(t, spec.MaxIterations,
			"builtin %q should have positive MaxIterations", spec.Name)
	}
}

// TestBuiltins_AllHaveSystemPrompt tests that all builtins have non-empty system prompts.
// Kills mutant: clearing system prompts would make this test fail.
func TestBuiltins_AllHaveSystemPrompt(t *testing.T) {
	t.Parallel()

	for _, spec := range Builtins() {
		assert.NotEmpty(t, spec.SystemPrompt,
			"builtin %q should have a system prompt", spec.Name)
	}
}
