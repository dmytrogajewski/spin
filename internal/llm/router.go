package llm

import "sync"

// RouterConfig holds per-role provider overrides for the Router.
// Keys are Role values; values are Provider instances or nil (use fallback).
type RouterConfig struct {
	// Providers maps roles to pre-built providers.
	// Roles not present in this map fall back through the chain.
	Providers map[Role]Provider
}

// Router dispatches provider lookups by model role with fallback chains.
// Providers are resolved lazily on first use per role and cached.
type Router struct {
	configured map[Role]Provider
	resolved   map[Role]Provider
	mu         sync.RWMutex
}

// NewRouter creates a Router from the given configuration.
// The action provider is required; other roles are optional and fall back.
func NewRouter(cfg RouterConfig) *Router {
	configured := make(map[Role]Provider, len(cfg.Providers))

	for role, p := range cfg.Providers {
		if p != nil {
			configured[role] = p
		}
	}

	return &Router{
		configured: configured,
		resolved:   make(map[Role]Provider),
	}
}

// NewSingleProviderRouter creates a Router where all roles resolve to the same provider.
// This preserves current behavior when no multi-model configuration is used.
func NewSingleProviderRouter(provider Provider) *Router {
	return NewRouter(RouterConfig{
		Providers: map[Role]Provider{
			RoleAction:   provider,
			RoleThinking: provider,
			RoleCritique: provider,
			RoleCompact:  provider,
			RoleVision:   provider,
		},
	})
}

// ForRole returns the provider for the given model role.
// If no provider is configured for the role, the fallback chain is traversed.
// Results are cached after first resolution.
func (r *Router) ForRole(role Role) Provider {
	r.mu.RLock()

	if p, ok := r.resolved[role]; ok {
		r.mu.RUnlock()

		return p
	}

	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock.
	if p, ok := r.resolved[role]; ok {
		return p
	}

	p := r.resolve(role)
	r.resolved[role] = p

	return p
}

// resolve finds the provider for a role by checking configured providers
// and then walking the fallback chain.
func (r *Router) resolve(role Role) Provider {
	// Check if the requested role has a direct provider.
	if p, ok := r.configured[role]; ok {
		return p
	}

	// Walk the fallback chain.
	for _, fallback := range FallbackChain(role) {
		if p, ok := r.configured[fallback]; ok {
			return p
		}
	}

	// Terminal fallback: action provider (should always be configured).
	if p, ok := r.configured[RoleAction]; ok {
		return p
	}

	return nil
}
