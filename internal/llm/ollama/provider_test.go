package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

				// Verify capabilities.
				caps := provider.Capabilities()
				assert.False(t, caps.Vision) // Ollama typically doesn't support vision.
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

	// Ollama supports streaming and function calling via OpenAI compatibility.
	assert.True(t, caps.Streaming)
	assert.True(t, caps.FunctionCalling)

	// But typically not vision.
	assert.False(t, caps.Vision)
}

func TestProvider_Models(t *testing.T) {
	// Mock /api/tags response matching Ollama's ListResponse format.
	mockResponse := map[string]any{
		"models": []map[string]any{
			{
				"name":        "llama3.1",
				"model":       "llama3.1",
				"modified_at": "2024-01-01T00:00:00Z",
				"size":        1234567,
				"digest":      "abc123",
				"details": map[string]any{
					"format":             "gguf",
					"family":             "llama",
					"parameter_size":     "8B",
					"quantization_level": "Q4_0",
				},
			},
			{
				"name":        "mistral",
				"model":       "mistral",
				"modified_at": "2024-02-01T00:00:00Z",
				"size":        7654321,
				"digest":      "def456",
				"details": map[string]any{
					"format":             "gguf",
					"family":             "mistral",
					"parameter_size":     "7B",
					"quantization_level": "Q4_0",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)

			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	provider, err := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama3.1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	models, err := provider.Models(ctx)
	require.NoError(t, err)
	require.Len(t, models, 2)

	assert.Equal(t, "llama3.1", models[0].ID)
	assert.EqualValues(t, "model", models[0].Object)

	assert.Equal(t, "mistral", models[1].ID)
	assert.EqualValues(t, "model", models[1].Object)
}
