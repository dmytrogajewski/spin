package factory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/lmstudio"
	"github.com/dmytrogajewski/spin/internal/llm/ollama"
	"github.com/dmytrogajewski/spin/internal/llm/openai"
)

var (
	ErrUnknownProviderType = errors.New("unknown provider type")
	ErrKeynameProvidedButNoAuthManager = errors.New("keyName  provided but no auth manager configured")
	ErrAuthenticationRequiredFor = errors.New("authentication required for")
	ErrProviderTypeIsRequired = errors.New("provider type is required")
	ErrBaseurlIsRequiredFor = errors.New("baseURL is required for")
	ErrAuthenticationRequiredFor2 = errors.New("authentication required for")
	ErrModelIsRequiredFor = errors.New("model is required for")
	ErrModelIsRequiredForOllama = errors.New("model is required for ollama")
)

// ProviderOptions contains provider-specific configuration options.
type ProviderOptions struct {
}

// ProviderConfig contains configuration for provider creation.
type ProviderConfig struct {
	// Type is the provider type (e.g., "openai", "ollama", "lmstudio", "openai-compatible").
	Type string

	// BaseURL is the API endpoint URL.
	BaseURL string

	// KeyName is the name of the credential in the keystore (recommended).
	// Takes precedence over APIKey if both are provided.
	// Requires an auth.Manager to be provided to the factory.
	KeyName string

	// APIKey is the authentication key (optional for local providers).
	// DEPRECATED: Use KeyName with secure keystore instead.
	// This field is deprecated and should not be used for new code.
	// Only use for backward compatibility or testing.
	APIKey string

	// Model is the default model identifier.
	Model string

	// Timeout is the request timeout.
	Timeout time.Duration

	// Options contains provider-specific options.
	Options ProviderOptions
}

// ProviderFactory creates a provider from configuration.
type ProviderFactory func(ProviderConfig) (llm.Provider, error)

// Factory creates LLM providers with optional authentication support.
type Factory struct {
	authMgr *auth.Manager
	logger  *slog.Logger
}

// NewFactory creates a new provider factory with optional auth support.
//
// If authMgr is nil, only direct APIKey credentials are supported (deprecated).
// For secure credential storage, provide an auth.Manager initialized with a keystore.
//
// Example:
//
//	authMgr := auth.NewManager(auth.NewKeystore())
//	factory := factory.NewFactory(authMgr)
//	provider, err := factory.NewProvider(ctx, cfg)
func NewFactory(authMgr *auth.Manager) *Factory {
	return &Factory{
		authMgr: authMgr,
		logger:  slog.Default(),
	}
}

// NewProvider creates a provider from configuration using the factory's auth manager.
//
// This is the recommended method for creating providers with secure credential storage.
// Credentials are resolved in the following order:
//  1. KeyName (from keystore via auth.Manager) - recommended
//  2. APIKey (direct, deprecated) - backward compatibility only
//
// Supported provider types:
//   - "openai": OpenAI-compatible provider
//   - "ollama": Ollama provider
//   - "lmstudio": LMStudio provider
//   - "openai-compatible": Generic OpenAI-compatible provider
//
// Returns an error if the provider type is unknown or configuration is invalid.
//
// Example:
//
//	authMgr := auth.NewManager(auth.NewKeystore())
//	factory := factory.NewFactory(authMgr)
//
//	cfg := factory.ProviderConfig{
//	    Type:    "openai",
//	    BaseURL: "https://api.openai.com/v1",
//	    KeyName: "my-openai-key",  // Recommended
//	    Model:   "gpt-4",
//	    Timeout: 30 * time.Second,
//	}
//	provider, err := factory.NewProvider(context.Background(), cfg)
func (f *Factory) NewProvider(ctx context.Context, cfg ProviderConfig) (llm.Provider, error) {
	// Validate configuration.
	err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Create provider based on type.
	providers := map[string]func(context.Context, ProviderConfig) (llm.Provider, error){
		"openai":            f.newOpenAIProvider,
		"openai-compatible": f.newOpenAIProvider,
		"ollama":            f.newOllamaProvider,
		"lmstudio":          f.newLMStudioProvider,
	}

	// Add test provider if enabled via build tags.
	f.addTestProvider(providers)

	if provider, exists := providers[cfg.Type]; exists {
		return provider(ctx, cfg)
	}

return nil, fmt.Errorf("unknown provider type: %s: %w", cfg.Type, ErrUnknownProviderType)
}

// resolveCredential resolves a credential from configuration.
//
// Resolution order:
//  1. KeyName (from keystore) - recommended, secure
//  2. APIKey (direct) - deprecated, insecure
//
// If requiresAuth is true and no credential is found, an error is returned.
// If requiresAuth is false, empty string is returned for local providers.
func (f *Factory) resolveCredential(ctx context.Context, cfg ProviderConfig, requiresAuth bool) (string, error) {
	// Priority 1: KeyName (secure keystore).
	if cfg.KeyName != "" {
		if f.authMgr == nil {
return "", fmt.Errorf("keyName %q provided but no auth manager configured: %w", cfg.KeyName, ErrKeynameProvidedButNoAuthManager)
		}

		cred, err := f.authMgr.GetCredential(ctx, cfg.KeyName)
		if err != nil {
			return "", fmt.Errorf("get credential %q: %w", cfg.KeyName, err)
		}

		return cred.Value, nil
	}

	// Priority 2: APIKey (deprecated, direct).
	if cfg.APIKey != "" {
		// Log deprecation warning.
		f.logger.WarnContext(ctx, "Direct APIKey is deprecated for security reasons. Use KeyName with secure keystore instead", "provider_type", cfg.Type)

		return cfg.APIKey, nil
	}

	// Priority 3: No authentication.
	if requiresAuth {
return "", fmt.Errorf("authentication required for %s: provide either KeyName (recommended) or APIKey (deprecated): %w", cfg.Type, ErrAuthenticationRequiredFor)
	}

	return "", nil // No auth needed (e.g., local Ollama).
}

// validateConfig validates provider configuration.
func validateConfig(cfg ProviderConfig) error {
	if cfg.Type == "" {
		return ErrProviderTypeIsRequired
	}

	validators := map[string]func(ProviderConfig) error{
		"openai":            validateOpenAIConfig,
		"openai-compatible": validateOpenAIConfig,
		"ollama":            validateOllamaConfig,
		"lmstudio":          validateLMStudioConfig,
	}

	if validator, exists := validators[cfg.Type]; exists {
		err := validator(cfg)
		if err != nil {
			return err
		}
	}

	return validateBaseURL(cfg.BaseURL)
}

// validateOpenAIConfig validates OpenAI provider configuration.
func validateOpenAIConfig(cfg ProviderConfig) error {
	if cfg.BaseURL == "" {
return fmt.Errorf("baseURL is required for %s: %w", cfg.Type, ErrBaseurlIsRequiredFor)
	}

	if cfg.KeyName == "" && cfg.APIKey == "" {
return fmt.Errorf("authentication required for %s: provide either KeyName (recommended) or APIKey (deprecated): %w", cfg.Type, ErrAuthenticationRequiredFor2)
	}

	if cfg.Model == "" {
return fmt.Errorf("model is required for %s: %w", cfg.Type, ErrModelIsRequiredFor)
	}

	return nil
}

// validateOllamaConfig validates Ollama provider configuration.
func validateOllamaConfig(cfg ProviderConfig) error {
	if cfg.Model == "" {
		return ErrModelIsRequiredForOllama
	}

	return nil
}

// validateLMStudioConfig validates LMStudio provider configuration.
func validateLMStudioConfig(_ ProviderConfig) error {
	return nil
}

// validateBaseURL validates the base URL if provided.
func validateBaseURL(baseURL string) error {
	if baseURL == "" {
		return nil
	}

	_, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid baseURL: %w", err)
	}

	return nil
}

// newOpenAIProvider creates an OpenAI provider from config (with auth support).
func (f *Factory) newOpenAIProvider(ctx context.Context, cfg ProviderConfig) (llm.Provider, error) {
	// Resolve credential (required for OpenAI).
	apiKey, err := f.resolveCredential(ctx, cfg, true)
	if err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = llm.DefaultTimeout
	}

	openaiCfg := openai.Config{
		BaseURL: cfg.BaseURL,
		APIKey:  apiKey,
		Model:   cfg.Model,
		Timeout: timeout,
	}

	return openai.NewProvider(openaiCfg)
}

// newOllamaProvider creates an Ollama provider from config (with auth support).
func (f *Factory) newOllamaProvider(ctx context.Context, cfg ProviderConfig) (llm.Provider, error) {
	// Ollama doesn't require authentication (local provider)
	// But we still resolve in case user wants to use auth for custom setup.
	_, err := f.resolveCredential(ctx, cfg, false)
	if err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = llm.DefaultTimeout
	}

	ollamaCfg := ollama.Config{
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		Timeout: timeout,
	}

	p, err := ollama.NewProvider(ollamaCfg)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// newLMStudioProvider creates an LMStudio provider from config (with auth support).
func (f *Factory) newLMStudioProvider(ctx context.Context, cfg ProviderConfig) (llm.Provider, error) {
	// LMStudio doesn't typically require authentication (local provider).
	_, err := f.resolveCredential(ctx, cfg, false)
	if err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = llm.DefaultTimeout
	}

	lmstudioCfg := lmstudio.Config{
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		Timeout: timeout,
	}

	return lmstudio.NewProvider(lmstudioCfg)
}
