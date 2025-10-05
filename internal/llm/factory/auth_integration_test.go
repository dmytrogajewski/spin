package factory

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/auth"
)

// TestNewFactory tests factory creation with auth manager.
func TestNewFactory(t *testing.T) {
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
	ctx := context.Background()

	// Setup keystore with test credential
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)

	// Store credential
	err := authMgr.SetCredential(ctx, "test-openai-key", auth.Credential{
		Type:  auth.CredentialTypeAPIKey,
		Value: "sk-test-12345",
	})
	if err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	// Create factory
	factory := NewFactory(authMgr)

	// Create provider using KeyName
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		KeyName: "test-openai-key", // Use keystore
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

	// Verify provider was created successfully
	if provider.Name() != "openai-compatible" {
		t.Errorf("Provider.Name() = %q, want %q", provider.Name(), "openai-compatible")
	}
}

// TestFactory_NewProvider_WithAPIKey tests provider creation using direct API key (deprecated).
func TestFactory_NewProvider_WithAPIKey(t *testing.T) {
	ctx := context.Background()

	// Create factory without auth manager
	factory := NewFactory(nil)

	// Create provider using direct APIKey
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-direct-key", // Direct key (deprecated)
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
	ctx := context.Background()

	// Setup keystore
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)

	// Store credential
	err := authMgr.SetCredential(ctx, "preferred-key", auth.Credential{
		Type:  auth.CredentialTypeAPIKey,
		Value: "sk-from-keystore",
	})
	if err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	factory := NewFactory(authMgr)

	// Provide both KeyName and APIKey
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		KeyName: "preferred-key",         // Should be used
		APIKey:  "sk-should-be-ignored",  // Should be ignored
		Model:   "gpt-4",
	}

	provider, err := factory.NewProvider(ctx, cfg)
	if err != nil {
		t.Fatalf("Factory.NewProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("Factory.NewProvider() returned nil provider")
	}

	// KeyName should take precedence (no error means keystore was used)
}

// TestFactory_NewProvider_KeyNameNotFound tests error when keystore credential not found.
func TestFactory_NewProvider_KeyNameNotFound(t *testing.T) {
	ctx := context.Background()

	// Setup empty keystore
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)

	factory := NewFactory(authMgr)

	// Try to use non-existent key
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

	// Error should mention the credential retrieval failure
	if !contains(err.Error(), "get credential") {
		t.Errorf("Error should mention credential retrieval, got: %v", err)
	}
}

// TestFactory_NewProvider_NoAuthManager tests error when KeyName provided but no auth manager.
func TestFactory_NewProvider_NoAuthManager(t *testing.T) {
	ctx := context.Background()

	// Create factory without auth manager
	factory := NewFactory(nil)

	// Try to use KeyName
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

	// Error should mention missing auth manager
	if !contains(err.Error(), "no auth manager configured") {
		t.Errorf("Error should mention missing auth manager, got: %v", err)
	}
}

// TestFactory_NewProvider_OllamaNoAuth tests Ollama provider without authentication.
func TestFactory_NewProvider_OllamaNoAuth(t *testing.T) {
	ctx := context.Background()

	// Create factory with auth manager (but Ollama doesn't need it)
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)
	factory := NewFactory(authMgr)

	// Create Ollama provider without any credentials
	cfg := ProviderConfig{
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
		Model:   "llama2",
		// No KeyName or APIKey
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
	ctx := context.Background()

	// Setup keystore
	keystore := auth.NewKeystore()
	authMgr := auth.NewManager(keystore)

	// Store credential for OpenAI
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
				BaseURL: "http://localhost:11434",
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
			provider, err := factory.NewProvider(ctx, tt.cfg)
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
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

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
		t.Fatal("Factory.NewProvider() expected error for cancelled context, got nil")
	}
	if provider != nil {
		t.Error("Factory.NewProvider() should return nil provider on error")
	}

	// Error should be context.Canceled
	if !contains(err.Error(), "context canceled") {
		t.Errorf("Error should mention context cancellation, got: %v", err)
	}
}

// TestFactory_NewProvider_BackwardCompatibility tests that legacy NewProvider still works.
func TestFactory_NewProvider_BackwardCompatibility(t *testing.T) {
	// This test ensures that existing code using NewProvider() still works
	cfg := ProviderConfig{
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-legacy-key",
		Model:   "gpt-4",
	}

	provider, err := NewProvider(cfg) // Legacy function
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
	ctx := context.Background()

	// Setup keystore
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
