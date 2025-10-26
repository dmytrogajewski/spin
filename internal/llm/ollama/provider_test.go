package ollama

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: Config{
				BaseURL: "http://localhost:11434",
				Model:   "llama3.1",
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "default base URL",
			cfg: Config{
				Model:   "llama3.1",
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing model",
			cfg: Config{
				BaseURL: "http://localhost:11434",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "model is required",
		},
		{
			name: "invalid base URL - caught during parse",
			cfg: Config{
				BaseURL: "://invalid",
				Model:   "llama3.1",
			},
			wantErr: true,
			errMsg:  "parse base URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.cfg)

			if tt.wantErr {
				t.Logf("Expected error, got: %v", err)
				require.Error(t, err, "Expected an error but got none")
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				require.NotNil(t, provider)
				assert.Equal(t, "ollama", provider.Name())

				// Verify capabilities
				caps := provider.Capabilities()
				assert.False(t, caps.Vision) // Ollama typically doesn't support vision
			}
		})
	}
}

func TestProvider_Name(t *testing.T) {
	provider, err := NewProvider(Config{
		Model: "llama3.1",
	})
	require.NoError(t, err)

	assert.Equal(t, "ollama", provider.Name())
}

func TestProvider_Capabilities(t *testing.T) {
	provider, err := NewProvider(Config{
		Model: "llama3.1",
	})
	require.NoError(t, err)

	caps := provider.Capabilities()

	// Ollama supports streaming and function calling via OpenAI compatibility
	assert.True(t, caps.Streaming)
	assert.True(t, caps.FunctionCalling)

	// But typically not vision
	assert.False(t, caps.Vision)
}

func TestProvider_Models_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider, err := NewProvider(Config{
		BaseURL: "http://localhost:11434",
		Model:   "llama3.1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	models, err := provider.Models(ctx)

	// This test requires Ollama to be running locally
	// If it fails, that's expected in CI/CD environments
	if err != nil {
		t.Logf("Ollama not available (expected in CI): %v", err)
		return
	}

	// If we got here, Ollama is running
	assert.NotNil(t, models)
	t.Logf("Found %d models", len(models))
}

func TestProvider_AutoTune_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider, err := NewProvider(Config{
		BaseURL: "http://localhost:11434",
		Model:   "llama3.1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// AutoTune requires running Ollama with the model available
	err = provider.AutoTune(ctx, 1024*1024*1024) // 1GB headroom

	// If Ollama isn't running, that's fine for unit tests
	if err != nil {
		t.Logf("AutoTune requires running Ollama (expected in CI): %v", err)
		return
	}

	t.Logf("AutoTune succeeded")
}

func TestProvider_GetAutoTuneWarning(t *testing.T) {
	provider, err := NewProvider(Config{
		Model: "llama3.1",
	})
	require.NoError(t, err)

	// Initially no warning
	warning := provider.GetAutoTuneWarning()
	assert.Empty(t, warning)

	// Set warning directly for testing
	provider.autoTuneWarning = "test warning"
	warning = provider.GetAutoTuneWarning()
	assert.Equal(t, "test warning", warning)
}
