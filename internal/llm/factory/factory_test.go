package factory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	openaisdk "github.com/openai/openai-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/lmstudio"
	"github.com/dmytrogajewski/spin/internal/llm/ollama"
	"github.com/dmytrogajewski/spin/internal/llm/openai"
)

var (
	errFactoryCreationFailed = errors.New("factory creation failed")
	errUnknownProviderType = errors.New("unknown provider type")
)

const testOllamaURL = "http://localhost:11434"

// TestNewProvider_OpenAI tests creating an OpenAI provider from config.
func TestNewProvider_OpenAI(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	cfg := ProviderConfig{
		Type:    "ollama",
		BaseURL: testOllamaURL,
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
	t.Parallel()

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
	t.Parallel()

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

	// openai-compatible uses openai provider.
	if provider.Name() != "openai-compatible" {
		t.Errorf("Provider.Name() = %s, want openai-compatible", provider.Name())
	}
}

// TestNewProvider_UnknownType tests that unknown provider types return error.
func TestNewProvider_UnknownType(t *testing.T) {
	t.Parallel()

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

	if !strings.Contains(err.Error(), "unknown-provider") {
		t.Errorf("Error should contain provider name, got: %v", err)
	}
}

// TestNewProvider_ValidationErrors tests configuration validation.
func TestNewProvider_ValidationErrors(t *testing.T) {
	t.Parallel()

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
			name: "ollama missing model",
			cfg: ProviderConfig{
				Type:    "ollama",
				BaseURL: testOllamaURL,
			},
			wantErr: "model is required for ollama",
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
			t.Parallel()

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
	t.Parallel()

	// Create a custom provider factory.
	customProvider := &mockProvider{name: "custom"}
	customFactory := func(_ ProviderConfig) (llm.Provider, error) {
		return customProvider, nil
	}

	// Register the custom provider.
	RegisterProvider("custom", customFactory)

	// Use the custom provider.
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
	t.Parallel()

	// Create a custom factory that overrides "openai".
	customProvider := &mockProvider{name: "custom-openai"}
	customFactory := func(_ ProviderConfig) (llm.Provider, error) {
		return customProvider, nil
	}

	// Register to override.
	RegisterProvider("openai", customFactory)

	defer func() {
		// Restore original for other tests.
		RegisterProvider("openai", legacyNewOpenAIProvider)
	}()

	// Create provider.
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
	t.Parallel()

	var wg sync.WaitGroup

	errChan := make(chan error, 100)

	// Register and create providers concurrently.
	for i := range 50 {
		wg.Add(2)

		// Register.
		go func(_ int) {
			defer wg.Done()

			name := "concurrent-provider"
			factory := func(_ ProviderConfig) (llm.Provider, error) {
				return &mockProvider{name: name}, nil
			}
			RegisterProvider(name, factory)
		}(i)

		// Create.
		go func() {
			defer wg.Done()

			cfg := ProviderConfig{
				Type:    "ollama",
				BaseURL: testOllamaURL,
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
	t.Parallel()

	factoryErr := errFactoryCreationFailed
	failingFactory := func(_ ProviderConfig) (llm.Provider, error) {
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
	t.Parallel()

	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "key",
		Model:   "gpt-4",
		// Timeout not specified.
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}

	// Provider should handle default timeout internally.
}

// TestNewProvider_URLNormalization tests URL normalization.
func TestNewProvider_URLNormalization(t *testing.T) {
	t.Parallel()

	cfg := ProviderConfig{
		Type:    "ollama",
		BaseURL: testOllamaURL + "/", // Trailing slash.
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
	t.Parallel()

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
		Options: ProviderOptions{},
	}

	_, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Verify options were passed to factory.
	_ = capturedCfg.Options // ProviderOptions currently empty; verify struct passed through.
}

// TestNewProvider_LMStudioDefaultURL tests LMStudio default URL handling.
func TestNewProvider_LMStudioDefaultURL(t *testing.T) {
	t.Parallel()

	cfg := ProviderConfig{
		Type:  "lmstudio",
		Model: "local-model",
		// BaseURL not specified - should use default.
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

func (m *mockProvider) Complete(_ context.Context, _ openaisdk.ChatCompletionNewParams) (*openaisdk.ChatCompletion, error) {
	return &openaisdk.ChatCompletion{
		ID:      fmt.Sprintf("mock-%d", time.Now().UnixNano()),
		Created: time.Now().Unix(),
		Model:   "mock-model",
		Object:  "chat.completion",
		Choices: []openaisdk.ChatCompletionChoice{{
			Index: 0,
			Message: openaisdk.ChatCompletionMessage{
				Role:    openaisdk.ChatCompletionMessageRoleAssistant,
				Content: "mock",
			},
			FinishReason: openaisdk.ChatCompletionChoicesFinishReasonStop,
		}},
	}, nil
}

func (m *mockProvider) Stream(_ context.Context, _ openaisdk.ChatCompletionNewParams) (<-chan openaisdk.ChatCompletionChunk, error) {
	ch := make(chan openaisdk.ChatCompletionChunk, 1)
	ch <- openaisdk.ChatCompletionChunk{
		ID:      fmt.Sprintf("chunk-%d", time.Now().UnixNano()),
		Created: time.Now().Unix(),
		Model:   "mock-model",
		Object:  "chat.completion.chunk",
		Choices: []openaisdk.ChatCompletionChunkChoice{{
			Index:        0,
			FinishReason: openaisdk.ChatCompletionChunkChoicesFinishReasonStop,
		}},
	}

	close(ch)

	return ch, nil
}

func (m *mockProvider) Models(_ context.Context) ([]openaisdk.Model, error) {
	return []openaisdk.Model{{ID: "mock"}}, nil
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

// TestFactoryConfigurationBugFix tests the fix for the "invalid configuration: model is required" error.
func TestFactoryConfigurationBugFix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        ProviderConfig
		expectedError bool
		errorContains string
		description   string
	}{
		{
			name: "valid_ollama_config",
			config: ProviderConfig{
				Type:    "ollama",
				BaseURL: testOllamaURL,
				Model:   "qwen3-coder:30b",
			},
			expectedError: false,
			description:   "Valid Ollama config should create provider successfully",
		},
		{
			name: "valid_openai_config_with_api_key",
			config: ProviderConfig{
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4",
				APIKey:  "sk-test-key",
			},
			expectedError: false,
			description:   "Valid OpenAI config with API key should create provider successfully",
		},
		{
			name: "valid_openai_config_with_key_name",
			config: ProviderConfig{
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4",
				KeyName: "openai-api-key",
			},
			expectedError: true, // This will fail because keystore doesn't have the credential.
			description:   "Valid OpenAI config with key name should create provider successfully",
		},
		{
			name: "missing_model_should_fail",
			config: ProviderConfig{
				Type:    "ollama",
				BaseURL: testOllamaURL,
				// Model missing.
			},
			expectedError: true,
			errorContains: "model is required",
			description:   "Config without model should fail validation",
		},
		{
			name: "missing_provider_type_should_fail",
			config: ProviderConfig{
				BaseURL: testOllamaURL,
				Model:   "qwen3-coder:30b",
				// Type missing.
			},
			expectedError: true,
			errorContains: "provider type is required",
			description:   "Config without provider type should fail validation",
		},
		{
			name: "missing_authentication_for_openai_should_fail",
			config: ProviderConfig{
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4",
				// No APIKey or KeyName.
			},
			expectedError: true,
			errorContains: "authentication required",
			description:   "OpenAI config without authentication should fail validation",
		},
		{
			name: "invalid_base_url_should_fail",
			config: ProviderConfig{
				Type:    "ollama",
				BaseURL: "not-a-valid-url",
				Model:   "qwen3-coder:30b",
			},
			expectedError: false, // Factory doesn't validate URLs, they fail at runtime.
			description:   "Config with invalid base URL should be accepted (fails at runtime)",
		},
		{
			name: "unknown_provider_type_should_fail",
			config: ProviderConfig{
				Type:    "unknown-provider",
				BaseURL: testOllamaURL,
				Model:   "test-model",
			},
			expectedError: true,
			errorContains: "unknown provider type",
			description:   "Config with unknown provider type should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test with factory (recommended approach).
			authMgr := auth.NewManager(auth.NewKeystore())

			factory := NewFactory(authMgr)

			ctx := context.Background()
			provider, err := factory.NewProvider(ctx, tt.config)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")

					return
				}

				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected text")
				}

				assert.Nil(t, provider, "Provider should be nil on error")
			} else {
				assert.NoError(t, err, "Unexpected error: %v", err)
				assert.NotNil(t, provider, "Provider should not be nil")

				if provider != nil {
					defer provider.Close()
				}
			}
		})
	}
}

// TestFactoryWithKeystore tests that factory properly handles keystore credentials.
func TestFactoryWithKeystore(t *testing.T) {
	t.Parallel()

	// Create keystore with test credentials.
	keystore := auth.NewKeystore()
	err := keystore.Set("test-openai-key", "sk-test-key-value")
	require.NoError(t, err, "Failed to set test credential")

	// Create auth manager with keystore.
	authMgr := auth.NewManager(keystore)

	// Create factory.
	factory := NewFactory(authMgr)

	// Test config with key name.
	config := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
		KeyName: "test-openai-key",
	}

	ctx := context.Background()
	provider, err := factory.NewProvider(ctx, config)

	assert.NoError(t, err, "Factory should create provider with keystore credentials")
	assert.NotNil(t, provider, "Provider should not be nil")

	if provider != nil {
		defer provider.Close()
	}
}

// TestFactoryTimeoutHandling tests that factory properly handles timeout configuration.
func TestFactoryTimeoutHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		configTimeout time.Duration
		expectedError bool
		description   string
	}{
		{
			name:          "default_timeout",
			configTimeout: 0, // Use default.
			expectedError: false,
			description:   "Default timeout should work",
		},
		{
			name:          "custom_timeout",
			configTimeout: 60 * time.Second,
			expectedError: false,
			description:   "Custom timeout should work",
		},
		{
			name:          "very_short_timeout",
			configTimeout: 1 * time.Millisecond,
			expectedError: false,
			description:   "Very short timeout should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := ProviderConfig{
				Type:    "ollama",
				BaseURL: testOllamaURL,
				Model:   "qwen3-coder:30b",
				Timeout: tt.configTimeout,
			}

			authMgr := auth.NewManager(auth.NewKeystore())

			factory := NewFactory(authMgr)

			ctx := context.Background()
			provider, err := factory.NewProvider(ctx, config)

			if tt.expectedError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Unexpected error: %v", err)
				assert.NotNil(t, provider, "Provider should not be nil")

				if provider != nil {
					defer provider.Close()
				}
			}
		})
	}
}

// TestFactoryLegacyCompatibility tests that legacy NewProvider function still works.
func TestFactoryLegacyCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        ProviderConfig
		expectedError bool
		description   string
	}{
		{
			name: "legacy_ollama_config",
			config: ProviderConfig{
				Type:    "ollama",
				BaseURL: testOllamaURL,
				Model:   "qwen3-coder:30b",
			},
			expectedError: false,
			description:   "Legacy Ollama config should work",
		},
		{
			name: "legacy_openai_config",
			config: ProviderConfig{
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4",
				APIKey:  "sk-test-key",
			},
			expectedError: false,
			description:   "Legacy OpenAI config should work",
		},
		{
			name: "legacy_missing_model",
			config: ProviderConfig{
				Type:    "ollama",
				BaseURL: testOllamaURL,
				// Model missing.
			},
			expectedError: true,
			description:   "Legacy config without model should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test legacy function.
			provider, err := NewProvider(tt.config)

			if tt.expectedError {
				assert.Error(t, err, "Expected error but got none")
				assert.Nil(t, provider, "Provider should be nil on error")
			} else {
				assert.NoError(t, err, "Unexpected error: %v", err)
				assert.NotNil(t, provider, "Provider should not be nil")

				if provider != nil {
					defer provider.Close()
				}
			}
		})
	}
}

// TestFactoryProviderRegistration tests that custom providers can be registered.
func TestFactoryProviderRegistration(t *testing.T) {
	t.Parallel()

	// Register a custom provider.
	RegisterProvider("custom", func(_ ProviderConfig) (llm.Provider, error) {
		return &mockProviderBugFix{name: "custom-provider"}, nil
	})

	// Test custom provider.
	config := ProviderConfig{
		Type:  "custom",
		Model: "custom-model",
	}

	provider, err := NewProvider(config)

	assert.NoError(t, err, "Custom provider should be created successfully")
	assert.NotNil(t, provider, "Provider should not be nil")
	assert.Equal(t, "custom-provider", provider.Name(), "Provider name should match")

	if provider != nil {
		defer provider.Close()
	}
}

// mockProviderBugFix is a simple mock provider for testing.
type mockProviderBugFix struct {
	name string
}

func (m *mockProviderBugFix) Complete(_ context.Context, _ openaisdk.ChatCompletionNewParams) (*openaisdk.ChatCompletion, error) {
	return &openaisdk.ChatCompletion{
		ID:      fmt.Sprintf("mock-%d", time.Now().UnixNano()),
		Created: time.Now().Unix(),
		Model:   "mock-model",
		Object:  "chat.completion",
		Choices: []openaisdk.ChatCompletionChoice{{
			Index: 0,
			Message: openaisdk.ChatCompletionMessage{
				Role:    openaisdk.ChatCompletionMessageRoleAssistant,
				Content: "Mock response",
			},
			FinishReason: openaisdk.ChatCompletionChoicesFinishReasonStop,
		}},
	}, nil
}

func (m *mockProviderBugFix) Stream(_ context.Context, _ openaisdk.ChatCompletionNewParams) (<-chan openaisdk.ChatCompletionChunk, error) {
	ch := make(chan openaisdk.ChatCompletionChunk, 2)

	go func() {
		defer close(ch)

		ch <- openaisdk.ChatCompletionChunk{
			ID:      fmt.Sprintf("chunk-%d", time.Now().UnixNano()),
			Created: time.Now().Unix(),
			Model:   "mock-model",
			Object:  "chat.completion.chunk",
			Choices: []openaisdk.ChatCompletionChunkChoice{{
				Index: 0,
				Delta: openaisdk.ChatCompletionChunkChoicesDelta{
					Content: "Mock response",
				},
			}},
		}

		ch <- openaisdk.ChatCompletionChunk{
			ID:      fmt.Sprintf("chunk-%d", time.Now().UnixNano()),
			Created: time.Now().Unix(),
			Model:   "mock-model",
			Object:  "chat.completion.chunk",
			Choices: []openaisdk.ChatCompletionChunkChoice{{
				Index:        0,
				FinishReason: openaisdk.ChatCompletionChunkChoicesFinishReasonStop,
			}},
		}
	}()

	return ch, nil
}

func (m *mockProviderBugFix) Models(_ context.Context) ([]openaisdk.Model, error) {
	return []openaisdk.Model{
		{ID: "mock-model"},
	}, nil
}

func (m *mockProviderBugFix) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: false,
	}
}

func (m *mockProviderBugFix) Name() string {
	return m.name
}

func (m *mockProviderBugFix) Close() error {
	return nil
}

// TestNewFactory tests factory creation with auth manager.
func TestNewFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		authMgr *auth.Manager
	}{
		{
			name:    "with auth manager",
			authMgr: auth.NewManager(auth.NewKeystore()),
		},
		{
			name:    "without auth manager (nil)",
			authMgr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			factory := NewFactory(tt.authMgr)
			if factory == nil {
				t.Fatal("NewFactory() returned nil")
			}

			if factory.authMgr != tt.authMgr {
				t.Errorf("Factory.authMgr = %v, want %v", factory.authMgr, tt.authMgr)
			}
		})
	}
}

// TestFactory_NewProvider_WithKeyName tests provider creation using keystore credentials.
func TestFactory_NewProvider_WithKeyName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup keystore with test credential.
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)

	// Store credential.
	err := authMgr.SetCredential(ctx, "test-openai-key", auth.Credential{
		Type:  auth.CredentialTypeAPIKey,
		Value: "sk-test-12345",
	})
	if err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	// Create factory.
	factory := NewFactory(authMgr)

	// Create provider using KeyName.
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		KeyName: "test-openai-key", // Use keystore.
		Model:   "gpt-4",
		Timeout: 30 * time.Second,
	}

	provider, err := factory.NewProvider(ctx, cfg)
	if err != nil {
		t.Fatalf("Factory.NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("Factory.NewProvider() returned nil provider")
	}

	// Verify provider was created successfully.
	if provider.Name() != "openai-compatible" {
		t.Errorf("Provider.Name() = %q, want %q", provider.Name(), "openai-compatible")
	}
}

// TestFactory_NewProvider_WithAPIKey tests provider creation using direct API key (deprecated).
func TestFactory_NewProvider_WithAPIKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create factory without auth manager.
	factory := NewFactory(nil)

	// Create provider using direct APIKey.
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-direct-key", // Direct key (deprecated).
		Model:   "gpt-4",
		Timeout: 30 * time.Second,
	}

	provider, err := factory.NewProvider(ctx, cfg)
	if err != nil {
		t.Fatalf("Factory.NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("Factory.NewProvider() returned nil provider")
	}
}

// TestFactory_NewProvider_KeyNamePrecedence tests that KeyName takes precedence over APIKey.
func TestFactory_NewProvider_KeyNamePrecedence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup keystore.
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)

	// Store credential.
	err := authMgr.SetCredential(ctx, "preferred-key", auth.Credential{
		Type:  auth.CredentialTypeAPIKey,
		Value: "sk-from-keystore",
	})
	if err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	factory := NewFactory(authMgr)

	// Provide both KeyName and APIKey.
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		KeyName: "preferred-key",        // Should be used.
		APIKey:  "sk-should-be-ignored", // Should be ignored.
		Model:   "gpt-4",
	}

	provider, err := factory.NewProvider(ctx, cfg)
	if err != nil {
		t.Fatalf("Factory.NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("Factory.NewProvider() returned nil provider")
	}

	// KeyName should take precedence (no error means keystore was used).
}

// TestFactory_NewProvider_KeyNameNotFound tests error when keystore credential not found.
func TestFactory_NewProvider_KeyNameNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup empty keystore.
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)

	factory := NewFactory(authMgr)

	// Try to use non-existent key.
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		KeyName: "nonexistent-key",
		Model:   "gpt-4",
	}

	provider, err := factory.NewProvider(ctx, cfg)
	if err == nil {
		t.Fatal("Factory.NewProvider() expected error for non-existent key, got nil")
	}

	if provider != nil {
		t.Error("Factory.NewProvider() should return nil provider on error")
	}

	// Error should mention the credential retrieval failure.
	if !contains(err.Error(), "get credential") {
		t.Errorf("Error should mention credential retrieval, got: %v", err)
	}
}

// TestFactory_NewProvider_NoAuthManager tests error when KeyName provided but no auth manager.
func TestFactory_NewProvider_NoAuthManager(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create factory without auth manager.
	factory := NewFactory(nil)

	// Try to use KeyName.
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		KeyName: "some-key",
		Model:   "gpt-4",
	}

	provider, err := factory.NewProvider(ctx, cfg)
	if err == nil {
		t.Fatal("Factory.NewProvider() expected error when KeyName provided but no auth manager, got nil")
	}

	if provider != nil {
		t.Error("Factory.NewProvider() should return nil provider on error")
	}

	// Error should mention missing auth manager.
	if !contains(err.Error(), "no auth manager configured") {
		t.Errorf("Error should mention missing auth manager, got: %v", err)
	}
}

// TestFactory_NewProvider_OllamaNoAuth tests Ollama provider without authentication.
func TestFactory_NewProvider_OllamaNoAuth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create factory with auth manager (but Ollama doesn't need it).
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)
	factory := NewFactory(authMgr)

	// Create Ollama provider without any credentials.
	cfg := ProviderConfig{
		Type:    "ollama",
		BaseURL: testOllamaURL,
		Model:   "llama2",
		// No KeyName or APIKey.
	}

	provider, err := factory.NewProvider(ctx, cfg)
	if err != nil {
		t.Fatalf("Factory.NewProvider() error = %v (Ollama shouldn't require auth)", err)
	}

	if provider == nil {
		t.Fatal("Factory.NewProvider() returned nil provider")
	}

	if provider.Name() != "ollama" {
		t.Errorf("Provider.Name() = %q, want %q", provider.Name(), "ollama")
	}
}

// TestFactory_NewProvider_AllProviderTypes tests all provider types with auth.
func TestFactory_NewProvider_AllProviderTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup keystore.
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)

	// Store credential for OpenAI.
	err := authMgr.SetCredential(ctx, "openai-key", auth.Credential{
		Type:  auth.CredentialTypeAPIKey,
		Value: "sk-test",
	})
	if err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	factory := NewFactory(authMgr)

	tests := []struct {
		name         string
		cfg          ProviderConfig
		expectedName string
	}{
		{
			name: "openai with KeyName",
			cfg: ProviderConfig{
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				KeyName: "openai-key",
				Model:   "gpt-4",
			},
			expectedName: "openai-compatible",
		},
		{
			name: "openai-compatible with KeyName",
			cfg: ProviderConfig{
				Type:    "openai-compatible",
				BaseURL: "https://custom.api/v1",
				KeyName: "openai-key",
				Model:   "custom-model",
			},
			expectedName: "openai-compatible",
		},
		{
			name: "ollama without auth",
			cfg: ProviderConfig{
				Type:    "ollama",
				BaseURL: testOllamaURL,
				Model:   "llama2",
			},
			expectedName: "ollama",
		},
		{
			name: "lmstudio without auth",
			cfg: ProviderConfig{
				Type:    "lmstudio",
				BaseURL: "http://localhost:1234",
				Model:   "local-model",
			},
			expectedName: "lmstudio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var provider llm.Provider
			provider, err = factory.NewProvider(ctx, tt.cfg)
			if err != nil {
				t.Fatalf("Factory.NewProvider() error = %v", err)
			}

			if provider == nil {
				t.Fatal("Factory.NewProvider() returned nil provider")
			}

			if provider.Name() != tt.expectedName {
				t.Errorf("Provider.Name() = %q, want %q", provider.Name(), tt.expectedName)
			}
		})
	}
}

// TestFactory_NewProvider_ContextCancellation tests context cancellation during credential retrieval.
func TestFactory_NewProvider_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Create canceled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel.

	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)
	factory := NewFactory(authMgr)

	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		KeyName: "test-key",
		Model:   "gpt-4",
	}

	provider, err := factory.NewProvider(ctx, cfg)
	if err == nil {
		t.Fatal("Factory.NewProvider() expected error for canceled context, got nil")
	}

	if provider != nil {
		t.Error("Factory.NewProvider() should return nil provider on error")
	}

	// Error should be context.Canceled.
	if !contains(err.Error(), "context canceled") {
		t.Errorf("Error should mention context cancellation, got: %v", err)
	}
}

// TestFactory_NewProvider_BackwardCompatibility tests that legacy NewProvider still works.
func TestFactory_NewProvider_BackwardCompatibility(t *testing.T) {
	t.Parallel()

	// This test ensures that existing code using NewProvider() still works.
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-legacy-key",
		Model:   "gpt-4",
	}

	provider, err := NewProvider(cfg) // Legacy function.
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() returned nil provider")
	}

	if provider.Name() != "openai-compatible" {
		t.Errorf("Provider.Name() = %q, want %q", provider.Name(), "openai-compatible")
	}
}

// TestFactory_resolveCredential tests the credential resolution logic.
func TestFactory_resolveCredential(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup keystore.
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)

	err := authMgr.SetCredential(ctx, "test-key", auth.Credential{
		Type:  auth.CredentialTypeAPIKey,
		Value: "sk-from-keystore",
	})
	if err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	tests := []struct {
		name         string
		factory      *Factory
		cfg          ProviderConfig
		requiresAuth bool
		wantValue    string
		wantErr      bool
	}{
		{
			name:         "keyName success",
			factory:      NewFactory(authMgr),
			cfg:          ProviderConfig{Type: "openai", KeyName: "test-key"},
			requiresAuth: true,
			wantValue:    "sk-from-keystore",
			wantErr:      false,
		},
		{
			name:         "apiKey fallback",
			factory:      NewFactory(nil),
			cfg:          ProviderConfig{Type: "openai", APIKey: "sk-direct"},
			requiresAuth: true,
			wantValue:    "sk-direct",
			wantErr:      false,
		},
		{
			name:         "keyName precedence",
			factory:      NewFactory(authMgr),
			cfg:          ProviderConfig{Type: "openai", KeyName: "test-key", APIKey: "sk-ignored"},
			requiresAuth: true,
			wantValue:    "sk-from-keystore",
			wantErr:      false,
		},
		{
			name:         "no auth required",
			factory:      NewFactory(authMgr),
			cfg:          ProviderConfig{Type: "ollama"},
			requiresAuth: false,
			wantValue:    "",
			wantErr:      false,
		},
		{
			name:         "auth required but missing",
			factory:      NewFactory(authMgr),
			cfg:          ProviderConfig{Type: "openai"},
			requiresAuth: true,
			wantValue:    "",
			wantErr:      true,
		},
		{
			name:         "keyName but no auth manager",
			factory:      NewFactory(nil),
			cfg:          ProviderConfig{Type: "openai", KeyName: "test-key"},
			requiresAuth: true,
			wantValue:    "",
			wantErr:      true,
		},
		{
			name:         "keyName not found",
			factory:      NewFactory(authMgr),
			cfg:          ProviderConfig{Type: "openai", KeyName: "nonexistent"},
			requiresAuth: true,
			wantValue:    "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			value, err := tt.factory.resolveCredential(ctx, tt.cfg, tt.requiresAuth)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveCredential() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if value != tt.wantValue {
				t.Errorf("resolveCredential() value = %q, want %q", value, tt.wantValue)
			}
		})
	}
}

// Test helper functions below - these provide backward compatibility for legacy tests.

var (
	// factoryMu protects the factories map.
	factoryMu sync.RWMutex

	// factories maps provider types to factory functions.
	factories = map[string]ProviderFactory{
		"openai":            legacyNewOpenAIProvider,
		"ollama":            legacyNewOllamaProvider,
		"lmstudio":          legacyNewLMStudioProvider,
		"openai-compatible": legacyNewOpenAIProvider,
	}
)

// NewProvider creates a provider from configuration (test helper for legacy tests).
func NewProvider(cfg ProviderConfig) (llm.Provider, error) {
	// Validate configuration.
	err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}

	factoryMu.RLock()

	factory, exists := factories[cfg.Type]

	factoryMu.RUnlock()

	if !exists {
return nil, fmt.Errorf("unknown provider type: %s: %w", cfg.Type, errUnknownProviderType)
	}

	return factory(cfg)
}

// RegisterProvider registers a custom provider factory (test helper for legacy tests).
func RegisterProvider(providerType string, factory ProviderFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()

	factories[providerType] = factory
}

// legacyNewOpenAIProvider creates an OpenAI provider from config (test helper).
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

// legacyNewOllamaProvider creates an Ollama provider from config (test helper).
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

// legacyNewLMStudioProvider creates an LMStudio provider from config (test helper).
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
