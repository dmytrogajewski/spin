package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Journey: specs/journeys/JOURNEY-1.4.md.

// TestCompile_Planner_ExcludesWriteTools tests that Compile("planner") omits
// dangerous tools from ToolSchemas — the core structural isolation proof.
// Kills mutant: including all tools would make this test fail.
func TestCompile_Planner_ExcludesWriteTools(t *testing.T) {
	t.Parallel()

	factory := subagentTestFactory(t)

	spec, err := factory.Compile(subagent.NamePlanner)
	require.NoError(t, err)

	toolNames := schemaNames(spec.ToolSchemas)

	assert.NotContains(t, toolNames, "shell_command",
		"planner must not have shell_command")

	assert.NotContains(t, toolNames, "write_file",
		"planner must not have write_file")

	assert.NotContains(t, toolNames, "apply_patch",
		"planner must not have apply_patch")
}

// TestCompile_Planner_IncludesReadTools tests that Compile("planner") includes
// the expected read-only tools.
// Kills mutant: returning empty schemas would make this test fail.
func TestCompile_Planner_IncludesReadTools(t *testing.T) {
	t.Parallel()

	factory := subagentTestFactory(t)

	spec, err := factory.Compile(subagent.NamePlanner)
	require.NoError(t, err)

	toolNames := schemaNames(spec.ToolSchemas)

	assert.Contains(t, toolNames, "read_file",
		"planner should have read_file")

	assert.Contains(t, toolNames, "file_search",
		"planner should have file_search")
}

// TestCompile_Explorer_ReadOnlyTools tests that Compile("explorer") includes
// only read-only tools.
// Kills mutant: including write tools would make this test fail.
func TestCompile_Explorer_ReadOnlyTools(t *testing.T) {
	t.Parallel()

	factory := subagentTestFactory(t)

	spec, err := factory.Compile(subagent.NameExplorer)
	require.NoError(t, err)

	toolNames := schemaNames(spec.ToolSchemas)

	assert.NotContains(t, toolNames, "shell_command")
	assert.NotContains(t, toolNames, "write_file")
	assert.NotContains(t, toolNames, "apply_patch")
	assert.Contains(t, toolNames, "read_file")
}

// TestCompile_AskUser_OnlyAskUserTool tests that Compile("ask_user") includes
// only the ask_user tool.
// Kills mutant: including extra tools would make this test fail.
func TestCompile_AskUser_OnlyAskUserTool(t *testing.T) {
	t.Parallel()

	factory := subagentTestFactory(t)

	spec, err := factory.Compile(subagent.NameAskUser)
	require.NoError(t, err)

	toolNames := schemaNames(spec.ToolSchemas)

	// ask_user is not in the builtin registry, so it won't appear.
	// But no other tools should appear either.
	assert.NotContains(t, toolNames, "shell_command")
	assert.NotContains(t, toolNames, "write_file")
	assert.NotContains(t, toolNames, "read_file")
}

// TestCompile_Reviewer_HasGitContext tests that Compile("reviewer") includes
// git_context but excludes write tools.
// Kills mutant: wrong tool list would make this test fail.
func TestCompile_Reviewer_HasGitContext(t *testing.T) {
	t.Parallel()

	factory := subagentTestFactory(t)

	spec, err := factory.Compile(subagent.NameReviewer)
	require.NoError(t, err)

	toolNames := schemaNames(spec.ToolSchemas)

	assert.Contains(t, toolNames, "git_context",
		"reviewer should have git_context")

	assert.NotContains(t, toolNames, "write_file",
		"reviewer must not have write_file")

	assert.NotContains(t, toolNames, "shell_command",
		"reviewer must not have shell_command")
}

// TestCompile_Subagent_IsSubagent tests that all subagent specs are marked
// as subagents with IsSubagent=true.
// Kills mutant: setting IsSubagent=false would make this test fail.
func TestCompile_Subagent_IsSubagent(t *testing.T) {
	t.Parallel()

	factory := subagentTestFactory(t)

	for _, name := range []string{
		subagent.NamePlanner,
		subagent.NameExplorer,
		subagent.NameReviewer,
		subagent.NameAskUser,
	} {
		spec, err := factory.Compile(name)
		require.NoError(t, err, "compile %q", name)

		assert.True(t, spec.IsSubagent,
			"%q should be marked as subagent", name)
	}
}

// TestCompile_Subagent_HasSystemPrompt tests that subagent specs have
// non-empty system prompts from builtins.
// Kills mutant: empty system prompt would make this test fail.
func TestCompile_Subagent_HasSystemPrompt(t *testing.T) {
	t.Parallel()

	factory := subagentTestFactory(t)

	for _, name := range []string{
		subagent.NamePlanner,
		subagent.NameExplorer,
		subagent.NameReviewer,
		subagent.NameAskUser,
	} {
		spec, err := factory.Compile(name)
		require.NoError(t, err, "compile %q", name)

		assert.NotEmpty(t, spec.SystemPrompt,
			"%q should have a system prompt", name)
	}
}

// subagentTestFactory creates a Factory with a full builtin registry for subagent tests.
func subagentTestFactory(t *testing.T) *Factory {
	t.Helper()

	cfg := validTestConfig()
	registry := tools.NewRegistryWithBuiltins()

	factory, err := NewFactory(cfg, registry, nil)
	require.NoError(t, err)

	return factory
}

// schemaNames extracts tool names from a slice of ToolSchema.
func schemaNames(schemas []tools.ToolSchema) []string {
	names := make([]string, len(schemas))
	for i, s := range schemas {
		names[i] = s.Function.Name
	}

	return names
}
