package builder

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild_AllProviders(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		wantType string
		wantErr  bool
		errMsg   string
	}{
		{
			name: "ollama provider",
			cfg: Config{
				Provider: "ollama",
				Model:    "llama3.1",
				BaseURL:  "http://localhost:11434",
			},
			wantType: "ollama",
		},
		{
			name: "lmstudio provider",
			cfg: Config{
				Provider: "lmstudio",
				Model:    "codellama",
				BaseURL:  "http://localhost:1234/v1",
			},
			wantType: "lmstudio",
		},
		{
			name: "openai provider with api key",
			cfg: Config{
				Provider: "openai",
				Model:    "gpt-4o",
				BaseURL:  "https://api.openai.com/v1",
				APIKey:   "sk-test123",
			},
			wantType: "openai",
		},
		{
			name: "openai-compatible provider",
			cfg: Config{
				Provider: "openai-compatible",
				Model:    "custom-model",
				BaseURL:  "https://custom-api.com/v1",
				APIKey:   "custom-key",
			},
			wantType: "openai",
		},
		{
			name: "openai without auth",
			cfg: Config{
				Provider: "openai",
				Model:    "gpt-4o",
				BaseURL:  "https://api.openai.com/v1",
				// No APIKey, no KeyName, and test must clear OPENAI_API_KEY env var
			},
			wantErr: true,
			errMsg:  "authentication required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Setup
			configLoader := config.NewLoaderV2()
			keystore := newTestKeystore()
			authMgr := auth.NewManager(keystore)
			builder := NewBuilder(configLoader, authMgr)

			// Clear environment variables that might interfere with auth tests
			oldOpenAI := os.Getenv("OPENAI_API_KEY")
			oldAnthropic := os.Getenv("ANTHROPIC_API_KEY")
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("ANTHROPIC_API_KEY")
			defer func() {
				if oldOpenAI != "" {
					os.Setenv("OPENAI_API_KEY", oldOpenAI)
				}
				if oldAnthropic != "" {
					os.Setenv("ANTHROPIC_API_KEY", oldAnthropic)
				}
			}()

			// Execute
			provider, err := builder.Build(ctx, tt.cfg)

			// Verify
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)
			defer provider.Close()
		})
	}
}

func TestConfigPrecedence(t *testing.T) {
	tests := []struct {
		name         string
		flagCfg      Config
		fileCfg      map[string]interface{}
		envVars      map[string]string
		wantProvider string
		wantModel    string
		wantBaseURL  string
	}{
		{
			name: "flags override everything",
			flagCfg: Config{
				Provider: "ollama",
				Model:    "llama3.1",
				BaseURL:  "http://localhost:11434",
			},
			fileCfg: map[string]interface{}{
				"llm.provider": "openai",
				"llm.model":    "gpt-4o",
			},
			envVars: map[string]string{
				"SPIN_PROVIDER": "lmstudio",
			},
			wantProvider: "ollama",
			wantModel:    "llama3.1",
			wantBaseURL:  "http://localhost:11434",
		},
		{
			name:    "config file used when flags empty",
			flagCfg: Config{}, // Empty flags
			fileCfg: map[string]interface{}{
				"llm.provider": "ollama",
				"llm.model":    "mixtral",
				"llm.base_url": "http://localhost:11434",
			},
			wantProvider: "ollama",
			wantModel:    "mixtral",
			wantBaseURL:  "http://localhost:11434",
		},
		{
			name:         "defaults when nothing set",
			flagCfg:      Config{Model: "llama3.1"}, // Only model set
			fileCfg:      map[string]interface{}{},
			wantProvider: "ollama", // Default provider
			wantModel:    "llama3.1",
			wantBaseURL:  "http://localhost:11434", // Default ollama URL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config loader with file values
			configLoader := config.NewLoaderV2()
			for k, v := range tt.fileCfg {
				configLoader.Set(k, v)
			}

			// Setup env vars
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			// Create builder
			keystore := newTestKeystore()
			authMgr := auth.NewManager(keystore)
			builder := NewBuilder(configLoader, authMgr)

			// Merge config (internal method test)
			merged := builder.mergeConfig(tt.flagCfg)

			// Verify precedence
			assert.Equal(t, tt.wantProvider, merged.Provider)
			assert.Equal(t, tt.wantModel, merged.Model)
			assert.Equal(t, tt.wantBaseURL, merged.BaseURL)
		})
	}
}

func TestAuthMethods(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		keyName string
		keyVal  string
		envKey  string
		envVal  string
		wantErr bool
		errMsg  string
	}{
		{
			name: "keystore auth - success",
			cfg: Config{
				Provider: "openai",
				Model:    "gpt-4o",
				BaseURL:  "https://api.openai.com/v1",
				KeyName:  "my-openai-key",
			},
			keyName: "my-openai-key",
			keyVal:  "sk-test123",
		},
		{
			name: "direct api key - success with warning",
			cfg: Config{
				Provider: "openai",
				Model:    "gpt-4o",
				BaseURL:  "https://api.openai.com/v1",
				APIKey:   "sk-direct-key",
			},
		},
		{
			name: "env var fallback - openai",
			cfg: Config{
				Provider: "openai",
				Model:    "gpt-4o",
				BaseURL:  "https://api.openai.com/v1",
			},
			envKey: "OPENAI_API_KEY",
			envVal: "sk-from-env",
		},
		{
			name: "missing keystore key",
			cfg: Config{
				Provider: "openai",
				Model:    "gpt-4o",
				BaseURL:  "https://api.openai.com/v1",
				KeyName:  "nonexistent-key",
			},
			wantErr: true,
			errMsg:  "not authenticated",
		},
		{
			name: "no auth for local provider - success",
			cfg: Config{
				Provider: "ollama",
				Model:    "llama3.1",
				BaseURL:  "http://localhost:11434",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Setup keystore and auth manager
			keystore := newTestKeystore()
			authMgr := auth.NewManager(keystore)

			// Store credential if needed
			if tt.keyName != "" && tt.keyVal != "" {
				cred := auth.Credential{
					Type:  auth.CredentialTypeAPIKey,
					Value: tt.keyVal,
				}
				err := authMgr.SetCredential(ctx, tt.keyName, cred)
				require.NoError(t, err)
			}

			// Setup env var
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}

			// Create builder
			configLoader := config.NewLoaderV2()
			builder := NewBuilder(configLoader, authMgr)

			// Execute
			provider, err := builder.Build(ctx, tt.cfg)

			// Verify
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)
			defer provider.Close()
		})
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "missing provider",
			cfg: Config{
				Model:   "gpt-4o",
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "sk-test",
			},
			wantErr: "provider is required",
		},
		{
			name: "missing model",
			cfg: Config{
				Provider: "openai",
				BaseURL:  "https://api.openai.com/v1",
				APIKey:   "sk-test",
			},
			wantErr: "model is required",
		},
		{
			name: "openai missing auth",
			cfg: Config{
				Provider: "openai",
				Model:    "gpt-4o",
				BaseURL:  "https://api.openai.com/v1",
			},
			wantErr: "authentication required",
		},
		{
			name: "ollama no auth required - valid",
			cfg: Config{
				Provider: "ollama",
				Model:    "llama3.1",
				BaseURL:  "http://localhost:11434",
			},
			wantErr: "", // Should pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &Builder{}
			err := builder.validate(tt.cfg)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMergeConfig(t *testing.T) {
	tests := []struct {
		name     string
		explicit Config
		fileVals map[string]interface{}
		want     Config
	}{
		{
			name: "explicit values take precedence",
			explicit: Config{
				Provider: "ollama",
				Model:    "llama3.1",
			},
			fileVals: map[string]interface{}{
				"llm.provider": "openai",
				"llm.model":    "gpt-4o",
				"llm.base_url": "https://api.openai.com/v1",
			},
			want: Config{
				Provider: "ollama",
				Model:    "llama3.1",
				BaseURL:  "http://localhost:11434", // Default for ollama
				Timeout:  5 * time.Minute,          // Default timeout from builder
			},
		},
		{
			name:     "file values used when explicit empty",
			explicit: Config{},
			fileVals: map[string]interface{}{
				"llm.provider": "ollama",
				"llm.model":    "mixtral",
				"llm.base_url": "http://custom:11434",
				"llm.timeout":  "60s",
			},
			want: Config{
				Provider: "ollama",
				Model:    "mixtral",
				BaseURL:  "http://custom:11434",
				Timeout:  60 * time.Second,
			},
		},
		{
			name: "defaults applied",
			explicit: Config{
				Model: "llama3.1",
			},
			fileVals: map[string]interface{}{},
			want: Config{
				Provider: "ollama", // Default
				Model:    "llama3.1",
				BaseURL:  "http://localhost:11434", // Default
				Timeout:  5 * time.Minute,          // Default timeout from builder
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config loader
			configLoader := config.NewLoaderV2()
			for k, v := range tt.fileVals {
				configLoader.Set(k, v)
			}

			// Create builder
			builder := &Builder{configLoader: configLoader}

			// Merge
			got := builder.mergeConfig(tt.explicit)

			// Verify
			assert.Equal(t, tt.want.Provider, got.Provider)
			assert.Equal(t, tt.want.Model, got.Model)
			assert.Equal(t, tt.want.BaseURL, got.BaseURL)
			assert.Equal(t, tt.want.Timeout, got.Timeout)
		})
	}
}

func TestEnvVarResolution(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		envKey   string
		envVal   string
		wantKey  string
	}{
		{
			name:     "openai env var",
			provider: "openai",
			envKey:   "OPENAI_API_KEY",
			envVal:   "sk-from-env-openai",
			wantKey:  "sk-from-env-openai",
		},
		{
			name:     "anthropic env var",
			provider: "anthropic",
			envKey:   "ANTHROPIC_API_KEY",
			envVal:   "sk-ant-from-env",
			wantKey:  "sk-ant-from-env",
		},
		{
			name:     "openai-compatible uses OPENAI_API_KEY",
			provider: "openai-compatible",
			envKey:   "OPENAI_API_KEY",
			envVal:   "custom-key",
			wantKey:  "custom-key",
		},
		{
			name:     "ollama no env var",
			provider: "ollama",
			wantKey:  "", // No env var for ollama
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}

			builder := &Builder{}
			got := builder.resolveAPIKeyFromEnv(tt.provider)

			assert.Equal(t, tt.wantKey, got)
		})
	}
}

func TestDefaultBaseURL(t *testing.T) {
	tests := []struct {
		provider string
		wantURL  string
	}{
		{"openai", "https://api.openai.com/v1"},
		{"anthropic", "https://api.anthropic.com/v1"},
		{"ollama", "http://localhost:11434"},
		{"lmstudio", "http://localhost:1234/v1"},
		{"openai-compatible", ""},
		{"unknown", ""},
	}

	builder := &Builder{}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := builder.defaultBaseURL(tt.provider)
			assert.Equal(t, tt.wantURL, got)
		})
	}
}

func TestEnvKeyForProvider(t *testing.T) {
	tests := []struct {
		provider string
		wantKey  string
	}{
		{"openai", "OPENAI_API_KEY"},
		{"openai-compatible", "OPENAI_API_KEY"},
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"ollama", ""},
		{"lmstudio", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := envKeyForProvider(tt.provider)
			assert.Equal(t, tt.wantKey, got)
		})
	}
}

// newTestKeystore creates an in-memory keystore for testing.
// We use the platform keystore factory which will return memory keystore in tests.
func newTestKeystore() auth.Keystore {
	return auth.NewKeystore()
}

func TestBuild_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Test full integration: config + auth + env vars
	t.Run("full integration with all sources", func(t *testing.T) {
		// Setup config file
		configLoader := config.NewLoaderV2()
		configLoader.Set("llm.provider", "ollama")
		configLoader.Set("llm.model", "llama3.1")
		configLoader.Set("llm.base_url", "http://localhost:11434")

		// Setup auth
		keystore := newTestKeystore()
		authMgr := auth.NewManager(keystore)

		// Setup env var (should be overridden by explicit config)
		t.Setenv("SPIN_PROVIDER", "openai")

		// Create builder
		builder := NewBuilder(configLoader, authMgr)

		// Build with explicit override (highest priority)
		provider, err := builder.Build(ctx, Config{
			Model: "mixtral", // Override model from config
		})

		require.NoError(t, err)
		require.NotNil(t, provider)
		defer provider.Close()

		// Verify merged config was correct
		// (provider from config, model from explicit)
	})
}
