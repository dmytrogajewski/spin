package builder

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
)

// TestNewBuilder_Simple tests that the new simplified builder works.
func TestNewBuilder_Simple(t *testing.T) {
	cfg := &config.V2{
		LLM: config.LLMV2{
			Provider: "ollama",
			Model:    "llama3.1",
			BaseURL:  "http://localhost:11434",
			Timeout:  5 * time.Minute,
		},
	}

	authMgr := auth.NewManager(auth.NewKeystore())
	builder := NewBuilder(cfg, authMgr)

	require.NotNil(t, builder, "builder should not be nil")
	require.Equal(t, cfg, builder.cfg, "builder should store config")
	require.Equal(t, authMgr, builder.authMgr, "builder should store auth manager")
}

// TestBuild_ConfigAlreadyMerged tests that Build uses config directly.
func TestBuild_ConfigAlreadyMerged(t *testing.T) {
	// Use a mock server so we don't depend on a real Ollama instance.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()

	cfg := &config.V2{
		LLM: config.LLMV2{
			Provider: "ollama",
			Model:    "qwen2.5-coder:7b",
			BaseURL:  server.URL,
			Timeout:  5 * time.Minute,
		},
	}

	authMgr := auth.NewManager(auth.NewKeystore())
	builder := NewBuilder(cfg, authMgr)

	ctx := context.Background()
	provider, err := builder.Build(ctx)

	require.NoError(t, err)
	require.NotNil(t, provider, "provider should not be nil")
	provider.Close()
}
