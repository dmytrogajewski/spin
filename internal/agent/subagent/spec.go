// Package subagent defines subagent specifications and orchestration.
// Subagents run specialized ReAct loops with filtered tools, separate
// conversation histories, and concurrency-controlled execution.
package subagent

import "slices"

// ToolSpawn is the parent spawn tool. Children receive it only when allowlisted.
const ToolSpawn = "spawn"

// Spec defines a subagent's configuration before compilation.
type Spec struct {
	// Name identifies this subagent type (e.g., "explorer", "planner").
	Name string

	// Description explains the subagent's purpose for logging and debugging.
	Description string

	// SystemPrompt is the specialized prompt for this subagent.
	SystemPrompt string

	// AllowedTools restricts which tools the subagent can invoke.
	// nil means all tools are allowed; an explicit list filters to those tools only.
	AllowedTools []string

	// ModelOverride optionally specifies a different LLM model for this subagent.
	ModelOverride string

	// MaxIterations is the iteration budget that prevents unbounded execution.
	MaxIterations int
}

// HasTool returns true if the named tool is in the AllowedTools list.
// If AllowedTools is nil (unrestricted), it returns true except for ToolSpawn,
// which requires an explicit allowlist (recursion deny-by-default).
func (s *Spec) HasTool(name string) bool {
	if name == ToolSpawn {
		return slices.Contains(s.AllowedTools, ToolSpawn)
	}

	if s.AllowedTools == nil {
		return true
	}

	return slices.Contains(s.AllowedTools, name)
}
