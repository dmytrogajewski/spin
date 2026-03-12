// Package lmstudio provides an LMStudio LLM provider implementation.
// LMStudio is a desktop application for running LLMs locally with an OpenAI-compatible API.
package lmstudio

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/openai"
)

const (
	// DefaultBaseURL is the default LMStudio API endpoint.
	DefaultBaseURL = "http://localhost:1234/v1"
)

// Config configures the LMStudio provider.
type Config struct {
	// BaseURL is the API endpoint URL (default: http://localhost:1234/v1)
	BaseURL string

	// Model is the model name (optional, can be specified per request).
	Model string

	// Timeout is the request timeout (defaults to 5 minutes).
	Timeout time.Duration
}

// Provider implements the LMStudio LLM provider by wrapping the OpenAI provider.
// LMStudio provides an OpenAI-compatible API, so we delegate to the OpenAI provider.
type Provider struct {
	*openai.Provider
}

// NewProvider creates a new LMStudio provider.
func NewProvider(cfg Config) (*Provider, error) {
	// Set defaults.
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = llm.DefaultTimeout
	}

	model := cfg.Model
	// If model is empty, use a placeholder - LMStudio can accept empty model
	// as it may use the loaded model automatically.
	if model == "" {
		model = "local-model"
	}

	// Create OpenAI provider with resolved config.
	openaiCfg := openai.Config{
		BaseURL: baseURL,
		APIKey:  "", // No API key for local LMStudio.
		Model:   model,
		Timeout: timeout,
	}

	openaiProvider, err := openai.NewProvider(openaiCfg)
	if err != nil {
		return nil, err
	}

	return &Provider{Provider: openaiProvider}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "lmstudio"
}

// Capabilities returns the provider's capabilities.
func (p *Provider) Capabilities() llm.Capabilities {
	// Get capabilities from OpenAI provider.
	caps := p.Provider.Capabilities()

	// LMStudio typically doesn't support vision.
	caps.Vision = false

	return caps
}

// Complete, Stream, Models, and Close are inherited from the embedded OpenAI provider
// and work automatically through Go's embedding mechanism.
