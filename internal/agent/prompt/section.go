package prompt

import "github.com/dmytrogajewski/spin/internal/agent"

// Priority tiers for prompt sections (from SPEC2.md §3.2.4).
const (
	// PriorityIdentityMin is the start of the Core Identity tier (0-99).
	PriorityIdentityMin = 0
	// PriorityToolsMin is the start of the Tool Definitions tier (100-199).
	PriorityToolsMin = 100
	// PrioritySafetyMin is the start of the Safety and Rules tier (200-299).
	PrioritySafetyMin = 200
	// PriorityProviderMin is the start of the Provider-Specific tier (300-399).
	PriorityProviderMin = 300
	// PriorityDynamicMin is the start of the Dynamic Context tier (400-499).
	PriorityDynamicMin = 400
)

// Condition evaluates whether a section should be included based on the environment.
type Condition func(env *agent.Environment) bool

// Section represents a modular prompt section with conditional inclusion.
type Section struct {
	// Name identifies this section uniquely within the composer.
	Name string

	// Priority controls ordering; lower values appear earlier in the prompt.
	Priority int

	// Condition determines whether this section is included.
	// A nil condition means the section is always included.
	Condition Condition

	// Cacheable marks this section as part of the stable prefix for prompt caching.
	Cacheable bool

	// Template is the markdown content with optional ${VAR} placeholders.
	Template string
}
