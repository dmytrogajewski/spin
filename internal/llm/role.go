package llm

// Role identifies a model role for multi-model routing.
type Role string

// Model roles with fallback chains (from SPEC2.md §3.4.1).
const (
	// RoleAction is the primary execution model. Required, no fallback.
	RoleAction Role = "action"
	// RoleThinking is the extended reasoning model. Falls back to Action.
	RoleThinking Role = "thinking"
	// RoleCritique is the self-evaluation model. Falls back to Thinking, then Action.
	RoleCritique Role = "critique"
	// RoleCompact is the fast model for context compaction. Falls back to Action.
	RoleCompact Role = "compact"
	// RoleVision is the vision-language model. Falls back to Action.
	RoleVision Role = "vision"
)

// fallbackChains defines the fallback order for each role.
// The chain is traversed in order; the first configured role wins.
// Action has no fallback (it is the terminal role).
var fallbackChains = map[Role][]Role{
	RoleAction:   {},
	RoleThinking: {RoleAction},
	RoleCritique: {RoleThinking, RoleAction},
	RoleCompact:  {RoleAction},
	RoleVision:   {RoleAction},
}

// FallbackChain returns the fallback roles for the given role.
// Returns nil for Action (terminal role).
func FallbackChain(role Role) []Role {
	return fallbackChains[role]
}
