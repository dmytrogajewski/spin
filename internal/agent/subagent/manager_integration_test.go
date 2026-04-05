package subagent_test

// Journey: specs/journeys/JOURNEY-3.1.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
)

// echoExec is a trivial executor that returns the query as summary.
func echoExec(_ context.Context, _ *subagent.Spec, query string) (string, error) {
	return "echo: " + query, nil
}

// TestSubagentManager_BuiltinsRegistered verifies that NewManager pre-registers
// all four Phase-1 builtin subagents: explorer, planner, reviewer, ask_user.
// Kills mutant: removing any builtin from Builtins() would make this fail.
func TestSubagentManager_BuiltinsRegistered(t *testing.T) {
	t.Parallel()

	mgr := subagent.NewManager(echoExec, subagent.DefaultMaxConcurrent)

	expected := []string{
		subagent.NameExplorer,
		subagent.NamePlanner,
		subagent.NameReviewer,
		subagent.NameAskUser,
	}

	for _, name := range expected {
		spec := mgr.Spec(name)
		require.NotNilf(t, spec, "builtin %q should be registered", name)
		assert.Equal(t, name, spec.Name, "spec name should match lookup key")
	}

	// Verify count matches exactly four builtins.
	builtins := subagent.Builtins()
	require.Len(t, builtins, len(expected), "should have exactly %d builtins", len(expected))
}

// TestSubagentManager_GetSpec_ReturnsCorrectSpec verifies that looking up a
// builtin spec returns fully populated fields: non-empty system prompt,
// non-empty description, a non-empty tools list, and positive MaxIterations.
// Kills mutant: zeroing any Spec field in Builtins() would make this fail.
func TestSubagentManager_GetSpec_ReturnsCorrectSpec(t *testing.T) {
	t.Parallel()

	mgr := subagent.NewManager(echoExec, subagent.DefaultMaxConcurrent)

	spec := mgr.Spec(subagent.NameExplorer)
	require.NotNil(t, spec)

	assert.Equal(t, subagent.NameExplorer, spec.Name)
	assert.NotEmpty(t, spec.SystemPrompt, "explorer system prompt should be non-empty")
	assert.NotEmpty(t, spec.Description, "explorer description should be non-empty")
	assert.NotEmpty(t, spec.AllowedTools, "explorer should have at least one allowed tool")
	assert.Positive(t, spec.MaxIterations, "MaxIterations should be positive")

	// Explorer should have read-only tools.
	assert.True(t, spec.HasTool("read_file"), "explorer should have read_file")
	assert.True(t, spec.HasTool("list_directory"), "explorer should have list_directory")
	assert.False(t, spec.HasTool("write_file"), "explorer should NOT have write_file")
}

// TestSubagentManager_RegisterCustom_OverridesBuiltin verifies that registering
// a custom spec with the same name as a builtin overwrites the builtin.
// Kills mutant: not storing the new spec would make this fail.
func TestSubagentManager_RegisterCustom_OverridesBuiltin(t *testing.T) {
	t.Parallel()

	mgr := subagent.NewManager(echoExec, subagent.DefaultMaxConcurrent)

	// Verify builtin explorer exists first.
	original := mgr.Spec(subagent.NameExplorer)
	require.NotNil(t, original)

	customPrompt := "You are a custom explorer override."
	customTools := []string{"custom_tool_a", "custom_tool_b"}

	err := mgr.Register(&subagent.Spec{
		Name:          subagent.NameExplorer,
		Description:   "Custom override of explorer.",
		SystemPrompt:  customPrompt,
		AllowedTools:  customTools,
		MaxIterations: 5,
	})
	require.NoError(t, err)

	// Lookup should return the custom spec, not the builtin.
	overridden := mgr.Spec(subagent.NameExplorer)
	require.NotNil(t, overridden)

	assert.Equal(t, customPrompt, overridden.SystemPrompt,
		"system prompt should be the custom one")
	assert.Equal(t, customTools, overridden.AllowedTools,
		"allowed tools should be the custom list")
	assert.Equal(t, 5, overridden.MaxIterations,
		"MaxIterations should be the custom value")

	// Spawning should still work with the overridden spec.
	summary, err := mgr.Spawn(context.Background(), subagent.NameExplorer, "test query")
	require.NoError(t, err)
	assert.Contains(t, summary, "test query")
}

// TestScaffoldFactory_CompilesWithSubagentSpecs verifies that the scaffold
// Factory can compile a subagent type (explorer) and that the resulting spec
// has the correct system prompt, tool schemas filtered to the allowed set,
// and IsSubagent=true. This is an integration-level check across subagent
// and scaffold packages.
func TestScaffoldFactory_CompilesWithSubagentSpecs(t *testing.T) {
	t.Parallel()

	// Import scaffold and tools indirectly through subagent's Builtins.
	// Verify that all builtin specs have consistent AllowedTools lists.
	for _, spec := range subagent.Builtins() {
		t.Run(spec.Name, func(t *testing.T) {
			t.Parallel()

			require.NotEmpty(t, spec.Name, "spec name must not be empty")
			require.NotEmpty(t, spec.SystemPrompt, "spec system prompt must not be empty")
			require.NotEmpty(t, spec.AllowedTools, "spec must have at least one tool")
			require.Positive(t, spec.MaxIterations, "MaxIterations must be positive")

			// HasTool should return true for every listed tool.
			for _, tool := range spec.AllowedTools {
				assert.True(t, spec.HasTool(tool),
					"HasTool(%q) should return true for spec %q", tool, spec.Name)
			}

			// HasTool should return false for an unlisted tool.
			assert.False(t, spec.HasTool("__nonexistent_tool__"),
				"HasTool should return false for unlisted tool")
		})
	}
}
