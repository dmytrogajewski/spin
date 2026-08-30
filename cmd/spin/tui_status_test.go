package main

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/config"
)

// Journey: specs/bugs/BUG-tui-context-and-spinner.md.

func TestResolveUIContextWindow_UsesConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.V2{}
	cfg.LLM.ContextWindow = 262144

	got := resolveUIContextWindow(cfg, nil)
	if got != 262144 {
		t.Fatalf("resolveUIContextWindow() = %d, want 262144", got)
	}
}

func TestResolveUIContextWindow_DefaultWhenUnset(t *testing.T) {
	t.Parallel()

	got := resolveUIContextWindow(&config.V2{}, nil)
	if got != defaultMaxTokens {
		t.Fatalf("resolveUIContextWindow() = %d, want default %d", got, defaultMaxTokens)
	}
}
