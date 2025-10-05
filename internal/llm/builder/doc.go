// Package builder provides a unified interface for creating LLM providers
// from multiple configuration sources.
//
// # Overview
//
// The builder package simplifies LLM provider creation by merging configuration
// from multiple sources with well-defined precedence rules. It integrates with:
//   - internal/config (YAML/TOML/JSON config files)
//   - internal/auth (secure keystore for credentials)
//   - internal/llm/factory (provider instantiation)
//   - Environment variables (fallback for API keys)
//
// # Configuration Precedence
//
// Configuration sources are merged in order of priority (highest to lowest):
//  1. Explicit Config struct parameter
//  2. Environment variables (SPIN_*, OPENAI_API_KEY, etc.)
//  3. Configuration file (~/.spin/spin.yaml)
//  4. Built-in defaults
//
// # Authentication Methods
//
// The builder supports multiple authentication methods:
//
// 1. Keystore (recommended):
//
//	provider, err := builder.Build(ctx, builder.Config{
//	    Provider: "openai",
//	    Model:    "gpt-4o",
//	    KeyName:  "my-openai-key",  // Retrieved from keystore
//	})
//
// 2. Direct API key (deprecated):
//
//	provider, err := builder.Build(ctx, builder.Config{
//	    Provider: "openai",
//	    Model:    "gpt-4o",
//	    APIKey:   "sk-...",  // Direct key (shows deprecation warning)
//	})
//
// 3. Environment variable (automatic):
//
//	// Set OPENAI_API_KEY environment variable
//	provider, err := builder.Build(ctx, builder.Config{
//	    Provider: "openai",
//	    Model:    "gpt-4o",
//	    // API key resolved from OPENAI_API_KEY env var
//	})
//
// # Supported Providers
//
// The builder supports all providers from internal/llm/factory:
//   - ollama: Local Ollama (no auth required)
//   - lmstudio: Local LMStudio (no auth required)
//   - openai: OpenAI API (auth required)
//   - anthropic: Anthropic Claude (auth required)
//   - openai-compatible: Generic OpenAI-compatible APIs (auth required)
//
// # Configuration File Format
//
// Example YAML configuration (~/.spin/spin.yaml):
//
//	llm:
//	  provider: openai
//	  model: gpt-4o
//	  base_url: https://api.openai.com/v1
//	  timeout: 30s
//	  key_name: my-openai-key  # Recommended
//	  # api_key: sk-...        # Deprecated
//
// # Environment Variables
//
// Provider-specific API keys:
//   - OPENAI_API_KEY: OpenAI and openai-compatible providers
//   - ANTHROPIC_API_KEY: Anthropic provider
//
// General configuration:
//   - SPIN_PROVIDER: Default provider type
//   - SPIN_MODEL: Default model name
//   - SPIN_BASE_URL: Default API endpoint
//
// # Usage Example
//
//	package main
//
//	import (
//	    "context"
//	    "github.com/dmytrogajewski/spin/internal/auth"
//	    "github.com/dmytrogajewski/spin/internal/config"
//	    "github.com/dmytrogajewski/spin/internal/llm/builder"
//	)
//
//	func main() {
//	    ctx := context.Background()
//
//	    // Load config file
//	    configLoader := config.NewLoader()
//	    if err := configLoader.Load(""); err != nil {
//	        // Config file optional, continue with defaults
//	    }
//
//	    // Setup auth
//	    keystore := auth.NewKeystore()
//	    authMgr := auth.NewManager(keystore)
//
//	    // Create builder
//	    b := builder.NewBuilder(configLoader, authMgr)
//
//	    // Build provider with explicit overrides
//	    provider, err := b.Build(ctx, builder.Config{
//	        Provider: "ollama",           // Override config file
//	        Model:    "llama3.1",         // Override config file
//	        // Other fields from config file or defaults
//	    })
//	    if err != nil {
//	        panic(err)
//	    }
//	    defer provider.Close()
//
//	    // Use provider...
//	}
//
// # Error Handling
//
// The Build method returns an error if:
//   - Required fields are missing (provider, model)
//   - Provider type is unknown
//   - Authentication is required but not provided
//   - Keystore credential is not found
//   - Provider creation fails
//
// Error messages include suggestions for fixing common issues:
//
//	"authentication required for openai (use --key-name or set OPENAI_API_KEY env var)"
//
// # Testing
//
// For testing, use in-memory configuration and keystores:
//
//	configLoader := config.NewLoader()
//	configLoader.Set("llm.provider", "ollama")
//	configLoader.Set("llm.model", "test-model")
//
//	keystore := auth.NewKeystore()  // Platform-specific or memory fallback
//	authMgr := auth.NewManager(keystore)
//
//	builder := builder.NewBuilder(configLoader, authMgr)
//	provider, err := builder.Build(ctx, builder.Config{})
package builder
