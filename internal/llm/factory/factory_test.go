package factory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// TestNewProvider_OpenAI tests creating an OpenAI provider from config.
func TestNewProvider_OpenAI(t *testing.T) {
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-test-key",
		Model:   "gpt-4",
		Timeout: 30 * time.Second,
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}

	if provider.Name() != "openai-compatible" {
		t.Errorf("Provider.Name() = %s, want openai-compatible", provider.Name())
	}
}

// TestNewProvider_Ollama tests creating an Ollama provider from config.
func TestNewProvider_Ollama(t *testing.T) {
	cfg := ProviderConfig{
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
		Model:   "llama2",
		Timeout: 60 * time.Second,
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}

	if provider.Name() != "ollama" {
		t.Errorf("Provider.Name() = %s, want ollama", provider.Name())
	}
}

// TestNewProvider_LMStudio tests creating an LMStudio provider from config.
func TestNewProvider_LMStudio(t *testing.T) {
	cfg := ProviderConfig{
		Type:    "lmstudio",
		BaseURL: "http://localhost:1234/v1",
		Model:   "local-model",
		Timeout: 45 * time.Second,
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}

	if provider.Name() != "lmstudio" {
		t.Errorf("Provider.Name() = %s, want lmstudio", provider.Name())
	}
}

// TestNewProvider_OpenAICompatible tests creating a generic OpenAI-compatible provider.
func TestNewProvider_OpenAICompatible(t *testing.T) {
	cfg := ProviderConfig{
		Type:    "openai-compatible",
		BaseURL: "https://custom-api.example.com/v1",
		APIKey:  "custom-key",
		Model:   "custom-model",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}

	// openai-compatible uses openai provider
	if provider.Name() != "openai-compatible" {
		t.Errorf("Provider.Name() = %s, want openai-compatible", provider.Name())
	}
}

// TestNewProvider_UnknownType tests that unknown provider types return error.
func TestNewProvider_UnknownType(t *testing.T) {
	cfg := ProviderConfig{
		Type:    "unknown-provider",
		BaseURL: "http://localhost:8080",
		Model:   "test",
	}

	provider, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("NewProvider() expected error for unknown type, got nil")
	}

	if provider != nil {
		t.Error("NewProvider() should return nil provider for unknown type")
	}

	if err.Error() != "unknown provider type: unknown-provider" {
		t.Errorf("Error message = %q, want %q", err.Error(), "unknown provider type: unknown-provider")
	}
}

// TestNewProvider_ValidationErrors tests configuration validation.
func TestNewProvider_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProviderConfig
		wantErr string
	}{
		{
			name: "empty type",
			cfg: ProviderConfig{
				BaseURL: "http://localhost",
				Model:   "test",
			},
			wantErr: "provider type is required",
		},
		{
			name: "openai missing baseURL",
			cfg: ProviderConfig{
				Type:   "openai",
				APIKey: "key",
				Model:  "gpt-4",
			},
			wantErr: "baseURL is required for openai",
		},
		{
			name: "openai missing apiKey",
			cfg: ProviderConfig{
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4",
			},
			wantErr: "authentication required for openai",
		},
		{
			name: "openai missing model",
			cfg: ProviderConfig{
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "key",
			},
			wantErr: "model is required for openai",
		},
		{
			name: "ollama missing baseURL",
			cfg: ProviderConfig{
				Type:  "ollama",
				Model: "llama2",
			},
			wantErr: "baseURL is required for ollama",
		},
		{
			name: "ollama missing model",
			cfg: ProviderConfig{
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
			},
			wantErr: "model is required for ollama",
		},
		{
			name: "lmstudio missing model",
			cfg: ProviderConfig{
				Type:    "lmstudio",
				BaseURL: "http://localhost:1234/v1",
			},
			wantErr: "model is required for lmstudio",
		},
		{
			name: "invalid URL",
			cfg: ProviderConfig{
				Type:    "ollama",
				BaseURL: "://invalid-url",
				Model:   "llama2",
			},
			wantErr: "invalid baseURL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.cfg)
			if err == nil {
				t.Fatal("NewProvider() expected error, got nil")
			}
			if provider != nil {
				t.Error("NewProvider() should return nil provider on error")
			}
			if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
				t.Errorf("Error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestRegisterProvider tests custom provider registration.
func TestRegisterProvider(t *testing.T) {
	// Create a custom provider factory
	customProvider := &mockProvider{name: "custom"}
	customFactory := func(cfg ProviderConfig) (llm.Provider, error) {
		return customProvider, nil
	}

	// Register the custom provider
	RegisterProvider("custom", customFactory)

	// Use the custom provider
	cfg := ProviderConfig{
		Type:    "custom",
		BaseURL: "http://custom",
		Model:   "test",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider != customProvider {
		t.Error("NewProvider() did not return custom provider")
	}

	if provider.Name() != "custom" {
		t.Errorf("Provider.Name() = %s, want custom", provider.Name())
	}
}

// TestRegisterProvider_Override tests overriding built-in providers.
func TestRegisterProvider_Override(t *testing.T) {
	// Create a custom factory that overrides "openai"
	customProvider := &mockProvider{name: "custom-openai"}
	customFactory := func(cfg ProviderConfig) (llm.Provider, error) {
		return customProvider, nil
	}

	// Register to override
	RegisterProvider("openai", customFactory)
	defer func() {
		// Restore original for other tests
		RegisterProvider("openai", legacyNewOpenAIProvider)
	}()

	// Create provider
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "key",
		Model:   "gpt-4",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider != customProvider {
		t.Error("NewProvider() did not use overridden provider")
	}
}

// TestRegisterProvider_Concurrent tests thread-safety of registration.
func TestRegisterProvider_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	// Register and create providers concurrently
	for i := 0; i < 50; i++ {
		wg.Add(2)

		// Register
		go func(n int) {
			defer wg.Done()
			name := "concurrent-provider"
			factory := func(cfg ProviderConfig) (llm.Provider, error) {
				return &mockProvider{name: name}, nil
			}
			RegisterProvider(name, factory)
		}(i)

		// Create
		go func() {
			defer wg.Done()
			cfg := ProviderConfig{
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "llama2",
			}
			_, err := NewProvider(cfg)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent operation error: %v", err)
	}
}

// TestProviderFactory_ErrorPropagation tests that factory errors are propagated.
func TestProviderFactory_ErrorPropagation(t *testing.T) {
	factoryErr := errors.New("factory creation failed")
	failingFactory := func(cfg ProviderConfig) (llm.Provider, error) {
		return nil, factoryErr
	}

	RegisterProvider("failing", failingFactory)

	cfg := ProviderConfig{
		Type:    "failing",
		BaseURL: "http://localhost",
		Model:   "test",
	}

	provider, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("NewProvider() expected error, got nil")
	}

	if provider != nil {
		t.Error("NewProvider() should return nil on factory error")
	}

	if !errors.Is(err, factoryErr) {
		t.Errorf("Error chain broken: got %v, want %v", err, factoryErr)
	}
}

// TestNewProvider_TimeoutDefault tests default timeout handling.
func TestNewProvider_TimeoutDefault(t *testing.T) {
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "key",
		Model:   "gpt-4",
		// Timeout not specified
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}

	// Provider should handle default timeout internally
}

// TestNewProvider_URLNormalization tests URL normalization.
func TestNewProvider_URLNormalization(t *testing.T) {
	cfg := ProviderConfig{
		Type:    "ollama",
		BaseURL: "http://localhost:11434/", // Trailing slash
		Model:   "llama2",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}
}

// TestNewProvider_OptionsPassthrough tests that Options are passed to providers.
func TestNewProvider_OptionsPassthrough(t *testing.T) {
	capturedCfg := ProviderConfig{}
	captureFactory := func(cfg ProviderConfig) (llm.Provider, error) {
		capturedCfg = cfg
		return &mockProvider{name: "capture"}, nil
	}

	RegisterProvider("capture", captureFactory)

	cfg := ProviderConfig{
		Type:    "capture",
		BaseURL: "http://localhost",
		Model:   "test",
		Options: map[string]interface{}{
			"custom_option": "value",
			"retry_count":   5,
		},
	}

	_, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if capturedCfg.Options == nil {
		t.Fatal("Options not passed to factory")
	}

	if capturedCfg.Options["custom_option"] != "value" {
		t.Error("Options not preserved correctly")
	}
}

// TestNewProvider_LMStudioDefaultURL tests LMStudio default URL handling.
func TestNewProvider_LMStudioDefaultURL(t *testing.T) {
	cfg := ProviderConfig{
		Type:  "lmstudio",
		Model: "local-model",
		// BaseURL not specified - should use default
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}

	if provider.Name() != "lmstudio" {
		t.Errorf("Provider.Name() = %s, want lmstudio", provider.Name())
	}
}

// mockProvider implements llm.Provider for testing.
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{Content: "mock"}, nil
}

func (m *mockProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	ch <- llm.StreamChunk{Type: llm.ChunkTypeDone}
	close(ch)
	return ch, nil
}

func (m *mockProvider) Models(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{{ID: "mock"}}, nil
}

func (m *mockProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{Streaming: true}
}

func (m *mockProvider) Close() error {
	return nil
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
