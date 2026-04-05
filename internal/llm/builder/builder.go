package builder

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/factory"
)

// Builder builds LLM providers from final configuration.
// NOTE: Config merging now happens in config.Load(), not here.
type Builder struct {
	cfg     *config.V2
	authMgr *auth.Manager
	factory *factory.Factory
}

// NewBuilder creates a new provider builder.
// Expects cfg to be fully resolved (via config.Load()).
func NewBuilder(cfg *config.V2, authMgr *auth.Manager) *Builder {
	return &Builder{
		cfg:     cfg,
		authMgr: authMgr,
		factory: factory.NewFactory(authMgr),
	}
}

// Build creates an LLM provider from the config.
// Config should already be fully merged via config.Load() including env vars.
func (b *Builder) Build(ctx context.Context) (llm.Provider, error) {
	providerCfg := factory.ProviderConfig{
		Type:    b.cfg.LLM.Provider,
		BaseURL: b.cfg.LLM.BaseURL,
		Model:   b.cfg.LLM.Model,
		Timeout: b.cfg.LLM.Timeout,
		APIKey:  b.cfg.LLM.APIKey,
		Options: factory.ProviderOptions{},
	}

	return b.factory.NewProvider(ctx, providerCfg)
}
