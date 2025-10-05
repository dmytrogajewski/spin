// Package factory provides a centralized factory for creating LLM providers.
//
// The factory pattern allows creating different provider implementations from
// configuration, supports custom provider registration, and validates configuration
// before provider creation.
//
// # Basic Usage
//
// Create a provider using the factory:
//
//	cfg := factory.ProviderConfig{
//	    Type:    "openai",
//	    BaseURL: "https://api.openai.com/v1",
//	    APIKey:  "sk-...",
//	    Model:   "gpt-4",
//	    Timeout: 30 * time.Second,
//	}
//
//	provider, err := factory.NewProvider(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
// # Supported Providers
//
// Built-in provider types:
//   - "openai": OpenAI and OpenAI-compatible APIs
//   - "ollama": Ollama local models
//   - "lmstudio": LMStudio local server
//   - "openai-compatible": Generic OpenAI-compatible endpoints
//
// # Custom Providers
//
// Register custom provider implementations:
//
//	factory.RegisterProvider("custom", func(cfg factory.ProviderConfig) (llm.Provider, error) {
//	    return &CustomProvider{
//	        baseURL: cfg.BaseURL,
//	        model:   cfg.Model,
//	    }, nil
//	})
//
//	provider, _ := factory.NewProvider(factory.ProviderConfig{
//	    Type: "custom",
//	    // ...
//	})
//
// # Configuration Validation
//
// The factory validates configuration before creating providers. Required fields
// vary by provider type:
//
//   - OpenAI: BaseURL, APIKey, Model
//   - Ollama: BaseURL, Model
//   - LMStudio: Model (BaseURL optional)
//
// Invalid configurations return descriptive errors:
//
//	provider, err := factory.NewProvider(factory.ProviderConfig{
//	    Type: "openai",
//	    // Missing required fields
//	})
//	// err: "baseURL is required for openai"
//
// # Thread Safety
//
// Provider registration is thread-safe and can be called concurrently.
// The factory itself is safe for concurrent use.
package factory
