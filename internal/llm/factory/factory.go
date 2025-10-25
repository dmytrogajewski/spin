package factory

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/lmstudio"
	"github.com/dmytrogajewski/spin/internal/llm/ollama"
	"github.com/dmytrogajewski/spin/internal/llm/openai"
)

// ProviderConfig contains configuration for provider creation.
type ProviderConfig struct {
	// Type is the provider type (e.g., "openai", "ollama", "lmstudio", "openai-compatible")
	Type string

	// BaseURL is the API endpoint URL
	BaseURL string

	// KeyName is the name of the credential in the keystore (recommended).
	// Takes precedence over APIKey if both are provided.
	// Requires an auth.Manager to be provided to the factory.
	KeyName string

	// APIKey is the authentication key (optional for local providers).
	// DEPRECATED: Use KeyName with secure keystore instead.
	// This field will be removed in v2.0.
	// Only use for backward compatibility or testing.
	APIKey string

	// Model is the default model identifier
	Model string

	// Timeout is the request timeout
	Timeout time.Duration

	// Options contains provider-specific options
	Options map[string]interface{}
}

// ProviderFactory creates a provider from configuration.
type ProviderFactory func(ProviderConfig) (llm.Provider, error)

// Factory creates LLM providers with optional authentication support.
type Factory struct {
	authMgr *auth.Manager
}

// NewFactory creates a new provider factory with optional auth support.
//
// If authMgr is nil, only direct APIKey credentials will be supported (deprecated).
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
	}
}

var (
	// factoryMu protects the factories map
	factoryMu sync.RWMutex

	// factories maps provider types to factory functions
	// Used for backward compatibility with standalone NewProvider function
	factories = map[string]ProviderFactory{
		"openai":            legacyNewOpenAIProvider,
		"ollama":            legacyNewOllamaProvider,
		"lmstudio":          legacyNewLMStudioProvider,
		"openai-compatible": legacyNewOpenAIProvider,
	}
)

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
	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// Create provider based on type
	providers := map[string]func(context.Context, ProviderConfig) (llm.Provider, error){
		"openai":            f.newOpenAIProvider,
		"openai-compatible": f.newOpenAIProvider,
		"ollama":            f.newOllamaProvider,
		"lmstudio":          f.newLMStudioProvider,
	}

	if provider, exists := providers[cfg.Type]; exists {
		return provider(ctx, cfg)
	}
	return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
}

// NewProvider creates a provider from configuration (legacy, backward compatible).
//
// DEPRECATED: Use NewFactory and Factory.NewProvider instead for auth support.
//
// This function only supports direct APIKey credentials and will be removed in v2.0.
// For secure credential storage, use:
//
//	factory := factory.NewFactory(authMgr)
//	provider, err := factory.NewProvider(ctx, cfg)
//
// Example:
//
//	cfg := factory.ProviderConfig{
//	    Type:    "openai",
//	    BaseURL: "https://api.openai.com/v1",
//	    APIKey:  "sk-...",  // Deprecated
//	    Model:   "gpt-4",
//	    Timeout: 30 * time.Second,
//	}
//	provider, err := factory.NewProvider(cfg)
func NewProvider(cfg ProviderConfig) (llm.Provider, error) {
	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	factoryMu.RLock()
	factory, exists := factories[cfg.Type]
	factoryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}

	return factory(cfg)
}

// RegisterProvider registers a custom provider factory.
//
// This allows applications to add support for custom LLM providers or override
// built-in providers. The factory function will be called by NewProvider when
// the specified type is requested.
//
// Registration is thread-safe and can be called concurrently.
//
// Example:
//
//	factory.RegisterProvider("custom", func(cfg factory.ProviderConfig) (llm.Provider, error) {
//	    return &CustomProvider{config: cfg}, nil
//	})
//
//	provider, _ := factory.NewProvider(factory.ProviderConfig{
//	    Type: "custom",
//	    // ...
//	})
func RegisterProvider(providerType string, factory ProviderFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	factories[providerType] = factory
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
	// Priority 1: KeyName (secure keystore)
	if cfg.KeyName != "" {
		if f.authMgr == nil {
			return "", fmt.Errorf("keyName %q provided but no auth manager configured", cfg.KeyName)
		}

		cred, err := f.authMgr.GetCredential(ctx, cfg.KeyName)
		if err != nil {
			return "", fmt.Errorf("get credential %q: %w", cfg.KeyName, err)
		}

		return cred.Value, nil
	}

	// Priority 2: APIKey (deprecated, direct)
	if cfg.APIKey != "" {
		// Log deprecation warning
		slog.Warn("Direct APIKey is deprecated for security reasons. Use KeyName with secure keystore instead", "provider_type", cfg.Type)
		return cfg.APIKey, nil
	}

	// Priority 3: No authentication
	if requiresAuth {
		return "", fmt.Errorf("authentication required for %s: provide either KeyName (recommended) or APIKey (deprecated)", cfg.Type)
	}

	return "", nil // No auth needed (e.g., local Ollama)
}

// validateConfig validates provider configuration.
func validateConfig(cfg ProviderConfig) error {
	if cfg.Type == "" {
		return fmt.Errorf("provider type is required")
	}

	validators := map[string]func(ProviderConfig) error{
		"openai":            validateOpenAIConfig,
		"openai-compatible": validateOpenAIConfig,
		"ollama":            validateOllamaConfig,
		"lmstudio":          validateLMStudioConfig,
	}

	if validator, exists := validators[cfg.Type]; exists {
		if err := validator(cfg); err != nil {
			return err
		}
	}

	return validateBaseURL(cfg.BaseURL)
}

// validateOpenAIConfig validates OpenAI provider configuration.
func validateOpenAIConfig(cfg ProviderConfig) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("baseURL is required for %s", cfg.Type)
	}
	if cfg.KeyName == "" && cfg.APIKey == "" {
		return fmt.Errorf("authentication required for %s: provide either KeyName (recommended) or APIKey (deprecated)", cfg.Type)
	}
	if cfg.Model == "" {
		return fmt.Errorf("model is required for %s", cfg.Type)
	}
	return nil
}

// validateOllamaConfig validates Ollama provider configuration.
func validateOllamaConfig(cfg ProviderConfig) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("baseURL is required for ollama")
	}
	if cfg.Model == "" {
		return fmt.Errorf("model is required for ollama")
	}
	return nil
}

// validateLMStudioConfig validates LMStudio provider configuration.
func validateLMStudioConfig(cfg ProviderConfig) error {
	if cfg.Model == "" {
		return fmt.Errorf("model is required for lmstudio")
	}
	return nil
}

// validateBaseURL validates the base URL if provided.
func validateBaseURL(baseURL string) error {
	if baseURL == "" {
		return nil
	}
	if _, err := url.Parse(baseURL); err != nil {
		return fmt.Errorf("invalid baseURL: %w", err)
	}
	return nil
}

// newOpenAIProvider creates an OpenAI provider from config (with auth support).
func (f *Factory) newOpenAIProvider(ctx context.Context, cfg ProviderConfig) (llm.Provider, error) {
	// Resolve credential (required for OpenAI)
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
	// But we still resolve in case user wants to use auth for custom setup
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

	if f.shouldAutoTune(cfg.Options) {
		headroom := f.extractVRAMHeadroom(cfg.Options)
		// best-effort; ignore error
		_ = p.AutoTune(ctx, headroom)
	}
	return p, nil
}

// shouldAutoTune determines if auto-tuning should be enabled.
func (f *Factory) shouldAutoTune(options map[string]interface{}) bool {
	if options == nil {
		return true // Auto-tune by default
	}
	if at, ok := options["auto_tune"].(bool); ok {
		return at
	}
	return true
}

// extractVRAMHeadroom extracts VRAM headroom from options.
func (f *Factory) extractVRAMHeadroom(options map[string]interface{}) int64 {
	if options == nil {
		return 1024 * 1024 * 1024 // Default 1GB
	}

	if v, ok := options["vram_headroom_mib"].(int); ok {
		return int64(v) * 1024 * 1024
	}
	if v, ok := options["vram_headroom_mib"].(float64); ok {
		return int64(v) * 1024 * 1024
	}

	return 1024 * 1024 * 1024 // Default 1GB
}

// newLMStudioProvider creates an LMStudio provider from config (with auth support).
func (f *Factory) newLMStudioProvider(ctx context.Context, cfg ProviderConfig) (llm.Provider, error) {
	// LMStudio doesn't typically require authentication (local provider)
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

// Legacy factory functions for backward compatibility (without auth support)

// legacyNewOpenAIProvider creates an OpenAI provider from config (legacy).
func legacyNewOpenAIProvider(cfg ProviderConfig) (llm.Provider, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = llm.DefaultTimeout
	}

	openaiCfg := openai.Config{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		Timeout: timeout,
	}

	return openai.NewProvider(openaiCfg)
}

// legacyNewOllamaProvider creates an Ollama provider from config (legacy).
func legacyNewOllamaProvider(cfg ProviderConfig) (llm.Provider, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = llm.DefaultTimeout
	}

	ollamaCfg := ollama.Config{
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		Timeout: timeout,
	}

	return ollama.NewProvider(ollamaCfg)
}

// legacyNewLMStudioProvider creates an LMStudio provider from config (legacy).
func legacyNewLMStudioProvider(cfg ProviderConfig) (llm.Provider, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = llm.DefaultTimeout
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = lmstudio.DefaultBaseURL
	}

	lmstudioCfg := lmstudio.Config{
		BaseURL: baseURL,
		Model:   cfg.Model,
		Timeout: timeout,
	}

	return lmstudio.NewProvider(lmstudioCfg)
}
