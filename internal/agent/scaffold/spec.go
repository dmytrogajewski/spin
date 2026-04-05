package scaffold

import (
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Spec is an immutable compiled agent specification produced by Factory.Compile.
// It contains everything needed to construct and run an agent without mutable
// builder state.
type Spec struct {
	// SystemPrompt is the assembled system prompt for this agent.
	SystemPrompt string

	// ToolSchemas are the OpenAI-compatible tool schemas available to this agent.
	ToolSchemas []tools.ToolSchema

	// AllowedTools restricts which tools the agent can invoke.
	// nil means all registered tools are allowed.
	AllowedTools []string

	// Providers maps model role names to LLM provider instances.
	Providers map[string]llm.Provider

	// SubagentSpecs contains compiled specifications for subagent types.
	SubagentSpecs map[string]*Spec

	// IsSubagent indicates whether this spec describes a subagent.
	IsSubagent bool

	// Config holds runtime-tunable parameters.
	Config SpecConfig
}

// HasTool returns true if the named tool is present in the compiled schemas.
func (s *Spec) HasTool(name string) bool {
	for _, schema := range s.ToolSchemas {
		if schema.Function.Name == name {
			return true
		}
	}

	return false
}

// ToolNames returns the names of all tools in the compiled schemas.
func (s *Spec) ToolNames() []string {
	names := make([]string, len(s.ToolSchemas))
	for i, schema := range s.ToolSchemas {
		names[i] = schema.Function.Name
	}

	return names
}
