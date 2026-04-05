package scaffold

import "time"

// ThinkingLevel controls the extended reasoning phase behavior.
type ThinkingLevel int

const (
	// ThinkingOff disables the thinking phase entirely.
	ThinkingOff ThinkingLevel = iota

	// ThinkingLow enables a single thinking LLM call without critique.
	ThinkingLow

	// ThinkingHigh enables thinking with a follow-up critique evaluation.
	ThinkingHigh
)

// SpecConfig holds runtime-tunable parameters for an agent specification.
type SpecConfig struct {
	// MaxTurns is the maximum number of ReAct loop iterations.
	MaxTurns int

	// Timeout is the maximum wall-clock time for the agent execution.
	Timeout time.Duration

	// Temperature controls LLM sampling randomness (0.0 to 2.0).
	Temperature float64

	// MaxTokens is the maximum number of tokens in the LLM response.
	MaxTokens int

	// ThinkingLevel controls the extended reasoning phase.
	ThinkingLevel ThinkingLevel
}
