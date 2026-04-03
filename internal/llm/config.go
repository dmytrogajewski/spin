package llm

import (
	"time"
)

// BaseConfig provides common configuration fields for all providers.
type BaseConfig struct {
	// BaseURL is the API endpoint URL.
	BaseURL string

	// Model is the model name to use.
	Model string

	// Timeout is the request timeout (defaults to 5 minutes).
	Timeout time.Duration

	// APIKey is the API key for authentication (optional for local providers).
	APIKey string
}

// DefaultTimeout is the default timeout for all providers.
const DefaultTimeout = 5 * time.Minute

// ResolveTimeout returns the given timeout if it is positive, otherwise DefaultTimeout.
func ResolveTimeout(t time.Duration) time.Duration {
	if t <= 0 {
		return DefaultTimeout
	}

	return t
}

// OpenAI-compatible finish reason constants.
const (
	FinishReasonStop      = "stop"
	FinishReasonToolCalls = "tool_calls"
	FinishReasonLength    = "length"
)
