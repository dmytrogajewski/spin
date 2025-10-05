// Package builder provides a unified interface for creating LLM providers
// from multiple configuration sources (flags, config files, environment variables).
package builder

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/factory"
)

// Config holds provider configuration from all sources.
type Config struct {
	// Core settings
	Provider string
	Model    string
	BaseURL  string
	Timeout  time.Duration

	// Auth (mutually exclusive, priority: KeyName > APIKey > env vars)
	KeyName string // Recommended: retrieve from keystore
	APIKey  string // Deprecated: direct key

	// Provider-specific options
	Options map[string]interface{}
}

// Builder builds LLM providers from multiple configuration sources.
//
// Configuration precedence (highest to lowest):
//  1. Explicit Config parameter
//  2. Environment variables
//  3. Config file
//  4. Built-in defaults
type Builder struct {
	configLoader *config.Loader
	authMgr      *auth.Manager
	factory      *factory.Factory
}

// NewBuilder creates a new provider builder.
//
// The builder combines configuration from:
//   - Config file (via configLoader)
//   - Secure keystore (via authMgr)
//   - Environment variables (automatic)
//   - Explicit Config struct (highest priority)
//
// Example:
//
//	configLoader := config.NewLoader()
//	authMgr := auth.NewManager(auth.NewKeystore())
//	builder := builder.NewBuilder(configLoader, authMgr)
//
//	provider, err := builder.Build(ctx, builder.Config{
//	    Provider: "openai",
//	    Model:    "gpt-4o",
//	    KeyName:  "my-openai-key",
//	})
func NewBuilder(cfg *config.Loader, authMgr *auth.Manager) *Builder {
	return &Builder{
		configLoader: cfg,
		authMgr:      authMgr,
		factory:      factory.NewFactory(authMgr),
	}
}

// Build creates an LLM provider from merged configuration.
//
// Configuration is merged in order of precedence:
//  1. Explicit cfg parameter (highest)
//  2. Environment variables
//  3. Config file
//  4. Built-in defaults (lowest)
//
// Returns an error if:
//   - Required fields are missing (provider, model)
//   - Provider type is unknown
//   - Authentication is required but not provided
//   - Provider creation fails
//
// Example:
//
//	// Use explicit config (overrides all)
//	provider, err := builder.Build(ctx, builder.Config{
//	    Provider: "ollama",
//	    Model:    "llama3.1",
//	})
//
//	// Use config file + explicit overrides
//	provider, err := builder.Build(ctx, builder.Config{
//	    Model: "mixtral", // Override model from config file
//	})
func (b *Builder) Build(ctx context.Context, cfg Config) (llm.Provider, error) {
	// Merge with config file settings
	merged := b.mergeConfig(cfg)

	// Resolve auth from environment BEFORE validation
	if merged.KeyName == "" && merged.APIKey == "" {
		if key := b.resolveAPIKeyFromEnv(merged.Provider); key != "" {
			merged.APIKey = key
		}
	}

	// Validate
	if err := b.validate(merged); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create provider
	providerCfg := factory.ProviderConfig{
		Type:    merged.Provider,
		BaseURL: merged.BaseURL,
		Model:   merged.Model,
		Timeout: merged.Timeout,
		KeyName: merged.KeyName,
		APIKey:  merged.APIKey,
		Options: merged.Options,
	}

	return b.factory.NewProvider(ctx, providerCfg)
}

// mergeConfig merges explicit config with file-based config.
//
// Precedence: explicit > config file > defaults
func (b *Builder) mergeConfig(explicit Config) Config {
	merged := explicit

	// Track if provider came from explicit config
	providerFromExplicit := explicit.Provider != ""

	// Fill in missing values from config file
	if merged.Provider == "" {
		merged.Provider = b.configLoader.GetString("llm.provider")
	}
	if merged.Model == "" {
		merged.Model = b.configLoader.GetString("llm.model")
	}
	if merged.Timeout == 0 {
		if t := b.configLoader.GetString("llm.timeout"); t != "" {
			if duration, err := time.ParseDuration(t); err == nil {
				merged.Timeout = duration
			}
		}
	}
	if merged.KeyName == "" {
		merged.KeyName = b.configLoader.GetString("llm.key_name")
	}

	// Apply provider defaults first
	if merged.Provider == "" {
		merged.Provider = "ollama" // Default to local Ollama
	}

	// BaseURL logic: Only use config file BaseURL if provider wasn't explicitly overridden
	if merged.BaseURL == "" {
		if !providerFromExplicit {
			// Provider from config, use config BaseURL
			merged.BaseURL = b.configLoader.GetString("llm.base_url")
		}
		// If still empty or provider was explicit, use default for that provider
		if merged.BaseURL == "" {
			merged.BaseURL = b.defaultBaseURL(merged.Provider)
		}
	}

	// Timeout default
	if merged.Timeout == 0 {
		merged.Timeout = 30 * time.Second
	}

	return merged
}

// validate validates the merged configuration.
func (b *Builder) validate(cfg Config) error {
	if cfg.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if cfg.Model == "" {
		return fmt.Errorf("model is required")
	}

	// Provider-specific validation
	switch cfg.Provider {
	case "openai", "anthropic", "openai-compatible":
		if cfg.KeyName == "" && cfg.APIKey == "" {
			envKey := envKeyForProvider(cfg.Provider)
			if envKey != "" {
				return fmt.Errorf("authentication required for %s (use --key-name or set %s env var)",
					cfg.Provider, envKey)
			}
			return fmt.Errorf("authentication required for %s", cfg.Provider)
		}
	}

	return nil
}

// resolveAPIKeyFromEnv attempts to resolve API key from environment.
func (b *Builder) resolveAPIKeyFromEnv(provider string) string {
	envKey := envKeyForProvider(provider)
	if envKey == "" {
		return ""
	}
	return os.Getenv(envKey)
}

// envKeyForProvider returns the env var name for a provider's API key.
func envKeyForProvider(provider string) string {
	switch provider {
	case "openai", "openai-compatible":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	default:
		return ""
	}
}

// defaultBaseURL returns the default base URL for a provider.
func (b *Builder) defaultBaseURL(provider string) string {
	switch provider {
	case "openai":
		return "https://api.openai.com/v1"
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "ollama":
		return "http://localhost:11434"
	case "lmstudio":
		return "http://localhost:1234/v1"
	default:
		return ""
	}
}
