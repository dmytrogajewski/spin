package llm

import (
	"time"
)

// ProviderConfig represents the unified configuration interface for all LLM providers.
// This standardizes configuration patterns across different providers.
type ProviderConfig interface {
	// GetBaseURL returns the base URL for the provider
	GetBaseURL() string

	// GetModel returns the model name
	GetModel() string

	// GetTimeout returns the request timeout
	GetTimeout() time.Duration

	// GetAPIKey returns the API key (if applicable)
	GetAPIKey() string

	// Validate validates the configuration
	Validate() error
}

// BaseConfig provides common configuration fields for all providers.
type BaseConfig struct {
	// BaseURL is the API endpoint URL
	BaseURL string

	// Model is the model name to use
	Model string

	// Timeout is the request timeout (defaults to 5 minutes)
	Timeout time.Duration

	// APIKey is the API key for authentication (optional for local providers)
	APIKey string
}

// DefaultTimeout is the default timeout for all providers
const DefaultTimeout = 5 * time.Minute
