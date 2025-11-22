package builder

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/stretchr/testify/require"
)

// TestNewBuilder_Simple tests that the new simplified builder works.
func TestNewBuilder_Simple(t *testing.T) {
	cfg := &config.ConfigV2{
		LLM: config.LLMConfigV2{
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
	cfg := &config.ConfigV2{
		LLM: config.LLMConfigV2{
			Provider: "ollama",
			Model:    "qwen2.5-coder:7b",
			BaseURL:  "http://localhost:11434",
			Timeout:  5 * time.Minute,
		},
	}

	authMgr := auth.NewManager(auth.NewKeystore())
	builder := NewBuilder(cfg, authMgr)

	ctx := context.Background()
	provider, err := builder.Build(ctx)

	// This will fail if ollama is not running, but that's OK -
	// we're testing that the API works correctly
	if err != nil {
		// Expected error if ollama not running
		require.Contains(t, err.Error(), "connection refused",
			"error should be about connection, not about config")
	} else {
		require.NotNil(t, provider, "provider should not be nil")
		provider.Close()
	}
}
