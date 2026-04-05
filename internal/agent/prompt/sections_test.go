package prompt_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/prompt"
)

// Journey: specs/journeys/JOURNEY-3.2.md.

const (
	minSectionCount     = 6
	regularSectionCount = 5
	testAgentsMDContent = "Follow TDD.\nUse Go 1.26."
)

// TestDefaultRegularSections_Count verifies at least 4 sections exist.
// Kills mutant: missing sections from DefaultRegularSections.
func TestDefaultRegularSections_Count(t *testing.T) {
	t.Parallel()

	sections := prompt.DefaultRegularSections()

	require.Len(t, sections, regularSectionCount)
}

// TestDefaultRegularSections_AllHaveNames verifies every section has a name.
// Kills mutant: empty section name.
func TestDefaultRegularSections_AllHaveNames(t *testing.T) {
	t.Parallel()

	for _, s := range prompt.DefaultRegularSections() {
		assert.NotEmpty(t, s.Name, "section has empty name")
	}
}

// TestDefaultRegularSections_AllHaveTemplates verifies every section has content.
// Kills mutant: empty template.
func TestDefaultRegularSections_AllHaveTemplates(t *testing.T) {
	t.Parallel()

	for _, s := range prompt.DefaultRegularSections() {
		assert.NotEmpty(t, s.Template, "section %q has empty template", s.Name)
	}
}

// TestDefaultRegularSections_DiffMatchesLegacy verifies composed output matches
// the legacy RegularSystemPrompt constant.
// Kills mutant: factored sections diverge from monolithic prompt.
func TestDefaultRegularSections_DiffMatchesLegacy(t *testing.T) {
	t.Parallel()

	c := prompt.NewComposer()
	for _, s := range prompt.DefaultRegularSections() {
		c.AddSection(s)
	}

	composed := c.Compose(nonGitEnv())

	assert.Equal(t, agent.RegularSystemPrompt, composed)
}

// TestProjectInstructionsSection_Format verifies AGENTS.md section format.
// Kills mutant: wrong header or missing separator.
func TestProjectInstructionsSection_Format(t *testing.T) {
	t.Parallel()

	s := prompt.ProjectInstructionsSection(testAgentsMDContent)

	assert.Equal(t, prompt.SectionProjectInstr, s.Name)
	assert.Contains(t, s.Template, "# Project Instructions")
	assert.Contains(t, s.Template, testAgentsMDContent)
	assert.Contains(t, s.Template, "---")
	assert.False(t, s.Cacheable)
}

// TestComposer_WithProjectInstructions_CacheableFirst verifies that composed
// output puts cacheable sections before dynamic (project instructions) sections.
// This enables prompt caching: stable prefix + dynamic suffix.
// Kills mutant: dynamic sections mixed into cacheable prefix.
func TestComposer_WithProjectInstructions_CacheableFirst(t *testing.T) {
	t.Parallel()

	composer := prompt.NewComposer()
	composer.AddSection(prompt.ProjectInstructionsSection(testAgentsMDContent))

	for _, s := range prompt.DefaultRegularSections() {
		composer.AddSection(s)
	}

	composed := composer.Compose(nonGitEnv())

	// Cacheable sections (identity, tool principle, tool guidance, response style) come first.
	// Dynamic section (project instructions) comes last.
	assert.Contains(t, composed, agent.RegularSystemPrompt)
	assert.Contains(t, composed, "# Project Instructions")
	assert.Contains(t, composed, testAgentsMDContent)

	// Project instructions should appear AFTER the cacheable system prompt.
	sysPromptIdx := strings.Index(composed, "You are an expert")
	projInstrIdx := strings.Index(composed, "# Project Instructions")

	assert.Greater(t, projInstrIdx, sysPromptIdx,
		"dynamic project instructions should follow cacheable system prompt")
}

// TestDefaultRegularSections_PlusProjInstr_AtLeastFive verifies total count
// meets the 5-section minimum when AGENTS.md is included.
// Kills mutant: fewer than 5 sections.
func TestDefaultRegularSections_PlusProjInstr_AtLeastFive(t *testing.T) {
	t.Parallel()

	sections := prompt.DefaultRegularSections()
	sections = append(sections, prompt.ProjectInstructionsSection("instructions"))

	require.GreaterOrEqual(t, len(sections), minSectionCount)
}

// TestDefaultRegularSections_UniqueNames verifies no duplicate section names.
// Kills mutant: duplicate names causing overwrites.
func TestDefaultRegularSections_UniqueNames(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)

	for _, s := range prompt.DefaultRegularSections() {
		require.False(t, seen[s.Name], "duplicate section name: %s", s.Name)

		seen[s.Name] = true
	}
}

// TestDefaultRegularSections_PriorityOrdering verifies sections are in priority order.
// Kills mutant: wrong priority assignment.
func TestDefaultRegularSections_PriorityOrdering(t *testing.T) {
	t.Parallel()

	sections := prompt.DefaultRegularSections()

	for i := 1; i < len(sections); i++ {
		assert.LessOrEqual(t, sections[i-1].Priority, sections[i].Priority,
			"section %q (priority %d) should come before %q (priority %d)",
			sections[i-1].Name, sections[i-1].Priority,
			sections[i].Name, sections[i].Priority)
	}
}
