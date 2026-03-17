package prompt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/prompt"
)

// Journey: specs/journeys/JOURNEY-3.1.md.

const (
	testCoreIdentity    = "You are Spin."
	testToolGuidance    = "Use ${EDIT_TOOL.name} to edit files."
	testSafetyRules     = "Never delete without confirmation."
	testGitWorkflow     = "Always commit before pushing."
	testDynamicContext  = "Project type: Go"
	testProviderHint    = "Provider-specific hint."
	testEditToolName    = "edit_file"
	testResolvedToolRef = "Use edit_file to edit files."
)

func gitEnv() *agent.Environment {
	return &agent.Environment{
		Git:         &agent.GitInfo{Branch: "main"},
		Environment: map[string]string{"is_repo": "true"},
	}
}

func nonGitEnv() *agent.Environment {
	return &agent.Environment{
		Environment: map[string]string{},
	}
}

func isGitRepo(env *agent.Environment) bool {
	return env.Git != nil
}

// TestComposer_EmptyReturnsEmpty verifies an empty composer returns empty string.
// Kills mutant: non-empty default return.
func TestComposer_EmptyReturnsEmpty(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()
	result := c.Compose(nonGitEnv())

	assert.Empty(t, result)
}

// TestComposer_SingleSection verifies a single section is rendered.
// Kills mutant: skipping all sections.
func TestComposer_SingleSection(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()
	c.AddSection(prompt.Section{
		Name:     "identity",
		Priority: prompt.PriorityIdentityMin,
		Template: testCoreIdentity,
	})

	result := c.Compose(nonGitEnv())

	assert.Equal(t, testCoreIdentity, result)
}

// TestComposer_SortedByPriority verifies sections are ordered by priority.
// Kills mutant: reversed or unsorted output.
func TestComposer_SortedByPriority(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()

	// Add in reverse order to verify sorting.
	c.AddSection(prompt.Section{
		Name:     "dynamic",
		Priority: prompt.PriorityDynamicMin,
		Template: testDynamicContext,
	})
	c.AddSection(prompt.Section{
		Name:     "identity",
		Priority: prompt.PriorityIdentityMin,
		Template: testCoreIdentity,
	})

	result := c.Compose(nonGitEnv())

	assert.Equal(t, testCoreIdentity+"\n\n"+testDynamicContext, result)
}

// TestComposer_ConditionExcludesSection verifies false condition excludes a section.
// Kills mutant: ignoring conditions.
func TestComposer_ConditionExcludesSection(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()
	c.AddSection(prompt.Section{
		Name:     "identity",
		Priority: prompt.PriorityIdentityMin,
		Template: testCoreIdentity,
	})
	c.AddSection(prompt.Section{
		Name:      "git",
		Priority:  prompt.PrioritySafetyMin,
		Condition: isGitRepo,
		Template:  testGitWorkflow,
	})

	result := c.Compose(nonGitEnv())

	assert.Equal(t, testCoreIdentity, result)
	assert.NotContains(t, result, testGitWorkflow)
}

// TestComposer_ConditionIncludesSection verifies true condition includes a section.
// Kills mutant: always excluding conditional sections.
func TestComposer_ConditionIncludesSection(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()
	c.AddSection(prompt.Section{
		Name:     "identity",
		Priority: prompt.PriorityIdentityMin,
		Template: testCoreIdentity,
	})
	c.AddSection(prompt.Section{
		Name:      "git",
		Priority:  prompt.PrioritySafetyMin,
		Condition: isGitRepo,
		Template:  testGitWorkflow,
	})

	result := c.Compose(gitEnv())

	assert.Contains(t, result, testCoreIdentity)
	assert.Contains(t, result, testGitWorkflow)
}

// TestComposer_NonGitExcludesGitSections verifies non-git environment excludes git sections.
// Kills mutant: git sections leaking into non-git prompts.
func TestComposer_NonGitExcludesGitSections(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()
	c.AddSection(prompt.Section{
		Name:     "identity",
		Priority: prompt.PriorityIdentityMin,
		Template: testCoreIdentity,
	})
	c.AddSection(prompt.Section{
		Name:      "git-workflow",
		Priority:  prompt.PrioritySafetyMin,
		Condition: isGitRepo,
		Template:  testGitWorkflow,
	})

	result := c.Compose(nonGitEnv())

	require.NotContains(t, result, testGitWorkflow)
	require.Contains(t, result, testCoreIdentity)
}

// TestComposer_VariableSubstitution verifies ${VAR} placeholders are resolved.
// Kills mutant: returning raw template without substitution.
func TestComposer_VariableSubstitution(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()
	c.SetVar("EDIT_TOOL.name", testEditToolName)
	c.AddSection(prompt.Section{
		Name:     "tools",
		Priority: prompt.PriorityToolsMin,
		Template: testToolGuidance,
	})

	result := c.Compose(nonGitEnv())

	assert.Equal(t, testResolvedToolRef, result)
	assert.NotContains(t, result, "${EDIT_TOOL.name}")
}

// TestComposer_ComposeTwoPart verifies stable/dynamic split.
// Kills mutant: all sections in one part.
func TestComposer_ComposeTwoPart(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()
	c.AddSection(prompt.Section{
		Name:      "identity",
		Priority:  prompt.PriorityIdentityMin,
		Cacheable: true,
		Template:  testCoreIdentity,
	})
	c.AddSection(prompt.Section{
		Name:      "safety",
		Priority:  prompt.PrioritySafetyMin,
		Cacheable: true,
		Template:  testSafetyRules,
	})
	c.AddSection(prompt.Section{
		Name:     "dynamic",
		Priority: prompt.PriorityDynamicMin,
		Template: testDynamicContext,
	})

	stable, dynamic := c.ComposeTwoPart(nonGitEnv())

	assert.Contains(t, stable, testCoreIdentity)
	assert.Contains(t, stable, testSafetyRules)
	assert.NotContains(t, stable, testDynamicContext)

	assert.Equal(t, testDynamicContext, dynamic)
	assert.NotContains(t, dynamic, testCoreIdentity)
}

// TestComposer_ComposeTwoPartEmpty verifies empty composer returns empty parts.
// Kills mutant: non-empty default.
func TestComposer_ComposeTwoPartEmpty(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()

	stable, dynamic := c.ComposeTwoPart(nonGitEnv())

	assert.Empty(t, stable)
	assert.Empty(t, dynamic)
}

// TestComposer_FiveTiers verifies all five tiers are ordered correctly.
// Kills mutant: wrong tier ordering.
func TestComposer_FiveTiers(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()

	// Add in mixed order.
	c.AddSection(prompt.Section{
		Name: "provider", Priority: prompt.PriorityProviderMin,
		Template: testProviderHint,
	})
	c.AddSection(prompt.Section{
		Name: "safety", Priority: prompt.PrioritySafetyMin,
		Template: testSafetyRules,
	})
	c.AddSection(prompt.Section{
		Name: "dynamic", Priority: prompt.PriorityDynamicMin,
		Template: testDynamicContext,
	})
	c.AddSection(prompt.Section{
		Name: "identity", Priority: prompt.PriorityIdentityMin,
		Template: testCoreIdentity,
	})
	c.AddSection(prompt.Section{
		Name: "tools", Priority: prompt.PriorityToolsMin,
		Template: testToolGuidance,
	})

	result := c.Compose(nonGitEnv())

	// Verify ordering by checking index positions.
	idIdx := indexOf(result, testCoreIdentity)
	toolIdx := indexOf(result, testToolGuidance)
	safetyIdx := indexOf(result, testSafetyRules)
	providerIdx := indexOf(result, testProviderHint)
	dynamicIdx := indexOf(result, testDynamicContext)

	assert.Less(t, idIdx, toolIdx)
	assert.Less(t, toolIdx, safetyIdx)
	assert.Less(t, safetyIdx, providerIdx)
	assert.Less(t, providerIdx, dynamicIdx)
}

// TestComposer_ThinkingModeExcludesToolGuidance verifies thinking mode skips tools.
// Kills mutant: tool guidance leaking into thinking prompt.
func TestComposer_ThinkingModeExcludesToolGuidance(t *testing.T) {
	t.Parallel()

	isActionMode := func(env *agent.Environment) bool {
		return env.Environment["mode"] != "thinking"
	}

	c := prompt.NewComposer()
	c.AddSection(prompt.Section{
		Name:     "identity",
		Priority: prompt.PriorityIdentityMin,
		Template: testCoreIdentity,
	})
	c.AddSection(prompt.Section{
		Name:      "tool-guidance",
		Priority:  prompt.PriorityToolsMin,
		Condition: isActionMode,
		Template:  testToolGuidance,
	})

	thinkingEnv := &agent.Environment{
		Environment: map[string]string{"mode": "thinking"},
	}

	result := c.Compose(thinkingEnv)

	assert.Contains(t, result, testCoreIdentity)
	assert.NotContains(t, result, testToolGuidance)
}

// TestComposer_MultipleVars verifies multiple variables are substituted.
// Kills mutant: only first variable resolved.
func TestComposer_MultipleVars(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()
	c.SetVar("TOOL_A", "read_file")
	c.SetVar("TOOL_B", "write_file")
	c.AddSection(prompt.Section{
		Name:     "multi",
		Priority: prompt.PriorityToolsMin,
		Template: "Use ${TOOL_A} and ${TOOL_B}.",
	})

	result := c.Compose(nonGitEnv())

	assert.Equal(t, "Use read_file and write_file.", result)
}

// TestPriorityConstants verifies tier boundaries are correct.
// Kills mutant: wrong constant values.
func TestPriorityConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, prompt.PriorityIdentityMin)
	assert.Equal(t, 100, prompt.PriorityToolsMin)
	assert.Equal(t, 200, prompt.PrioritySafetyMin)
	assert.Equal(t, 300, prompt.PriorityProviderMin)
	assert.Equal(t, 400, prompt.PriorityDynamicMin)
}

// indexOf returns the byte index of substr in s, or -1.
func indexOf(s, substr string) int {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}
