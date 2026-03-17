package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// Journey: specs/journeys/JOURNEY-4.1.md.

const (
	nameAction   = "action-provider"
	nameThinking = "thinking-provider"
	nameCritique = "critique-provider"
	nameCompact  = "compact-provider"
	nameVision   = "vision-provider"
)

func actionProvider() llm.Provider   { return llm.NewMockProvider(nameAction) }
func thinkingProvider() llm.Provider { return llm.NewMockProvider(nameThinking) }
func critiqueProvider() llm.Provider { return llm.NewMockProvider(nameCritique) }
func compactProvider() llm.Provider  { return llm.NewMockProvider(nameCompact) }
func visionProvider() llm.Provider   { return llm.NewMockProvider(nameVision) }

// TestRouter_ActionRole verifies action role returns the action provider.
// Kills mutant: action role resolving to wrong provider.
func TestRouter_ActionRole(t *testing.T) {
	t.Parallel()

	r := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction: actionProvider(),
		},
	})

	p := r.ForRole(llm.RoleAction)

	require.NotNil(t, p)
	assert.Equal(t, nameAction, p.Name())
}

// TestRouter_ThinkingConfigured verifies thinking role uses configured provider.
// Kills mutant: ignoring configured thinking provider.
func TestRouter_ThinkingConfigured(t *testing.T) {
	t.Parallel()

	r := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction:   actionProvider(),
			llm.RoleThinking: thinkingProvider(),
		},
	})

	p := r.ForRole(llm.RoleThinking)

	require.NotNil(t, p)
	assert.Equal(t, nameThinking, p.Name())
}

// TestRouter_ThinkingFallsBackToAction verifies fallback when thinking not configured.
// Kills mutant: returning nil instead of falling back.
func TestRouter_ThinkingFallsBackToAction(t *testing.T) {
	t.Parallel()

	r := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction: actionProvider(),
		},
	})

	p := r.ForRole(llm.RoleThinking)

	require.NotNil(t, p)
	assert.Equal(t, nameAction, p.Name())
}

// TestRouter_CritiqueFallsBackToThinking verifies critique fallback chain.
// Kills mutant: critique skipping thinking in fallback chain.
func TestRouter_CritiqueFallsBackToThinking(t *testing.T) {
	t.Parallel()

	r := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction:   actionProvider(),
			llm.RoleThinking: thinkingProvider(),
		},
	})

	p := r.ForRole(llm.RoleCritique)

	require.NotNil(t, p)
	assert.Equal(t, nameThinking, p.Name())
}

// TestRouter_CritiqueFallsBackToAction verifies full fallback chain.
// Kills mutant: critique not reaching action when thinking is absent.
func TestRouter_CritiqueFallsBackToAction(t *testing.T) {
	t.Parallel()

	r := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction: actionProvider(),
		},
	})

	p := r.ForRole(llm.RoleCritique)

	require.NotNil(t, p)
	assert.Equal(t, nameAction, p.Name())
}

// TestRouter_CompactFallsBackToAction verifies compact fallback.
// Kills mutant: compact not falling back.
func TestRouter_CompactFallsBackToAction(t *testing.T) {
	t.Parallel()

	r := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction: actionProvider(),
		},
	})

	p := r.ForRole(llm.RoleCompact)

	require.NotNil(t, p)
	assert.Equal(t, nameAction, p.Name())
}

// TestRouter_VisionFallsBackToAction verifies vision fallback.
// Kills mutant: vision not falling back.
func TestRouter_VisionFallsBackToAction(t *testing.T) {
	t.Parallel()

	r := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction: actionProvider(),
		},
	})

	p := r.ForRole(llm.RoleVision)

	require.NotNil(t, p)
	assert.Equal(t, nameAction, p.Name())
}

// TestRouter_AllRolesConfigured verifies each role gets its own provider.
// Kills mutant: roles sharing providers when independently configured.
func TestRouter_AllRolesConfigured(t *testing.T) {
	t.Parallel()

	r := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction:   actionProvider(),
			llm.RoleThinking: thinkingProvider(),
			llm.RoleCritique: critiqueProvider(),
			llm.RoleCompact:  compactProvider(),
			llm.RoleVision:   visionProvider(),
		},
	})

	assert.Equal(t, nameAction, r.ForRole(llm.RoleAction).Name())
	assert.Equal(t, nameThinking, r.ForRole(llm.RoleThinking).Name())
	assert.Equal(t, nameCritique, r.ForRole(llm.RoleCritique).Name())
	assert.Equal(t, nameCompact, r.ForRole(llm.RoleCompact).Name())
	assert.Equal(t, nameVision, r.ForRole(llm.RoleVision).Name())
}

// TestRouter_SingleProviderRouter verifies all roles resolve to same provider.
// Kills mutant: single provider router not covering all roles.
func TestRouter_SingleProviderRouter(t *testing.T) {
	t.Parallel()

	r := llm.NewSingleProviderRouter(actionProvider())

	assert.Equal(t, nameAction, r.ForRole(llm.RoleAction).Name())
	assert.Equal(t, nameAction, r.ForRole(llm.RoleThinking).Name())
	assert.Equal(t, nameAction, r.ForRole(llm.RoleCritique).Name())
	assert.Equal(t, nameAction, r.ForRole(llm.RoleCompact).Name())
	assert.Equal(t, nameAction, r.ForRole(llm.RoleVision).Name())
}

// TestRouter_LazyResolution verifies providers are cached after first resolution.
// Kills mutant: resolving on every call instead of caching.
func TestRouter_LazyResolution(t *testing.T) {
	t.Parallel()

	r := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction: actionProvider(),
		},
	})

	// First call resolves and caches.
	first := r.ForRole(llm.RoleThinking)
	// Second call returns cached result.
	second := r.ForRole(llm.RoleThinking)

	// Same instance (pointer equality).
	assert.Same(t, first, second)
}

// TestRouter_NilProviderIgnored verifies nil providers in config are skipped.
// Kills mutant: nil provider stored and returned.
func TestRouter_NilProviderIgnored(t *testing.T) {
	t.Parallel()

	r := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction:   actionProvider(),
			llm.RoleThinking: nil,
		},
	})

	p := r.ForRole(llm.RoleThinking)

	require.NotNil(t, p)
	assert.Equal(t, nameAction, p.Name())
}

// TestFallbackChain_Action verifies action has no fallbacks.
// Kills mutant: action having unintended fallback.
func TestFallbackChain_Action(t *testing.T) {
	t.Parallel()

	chain := llm.FallbackChain(llm.RoleAction)

	assert.Empty(t, chain)
}

// TestFallbackChain_Critique verifies critique chain order.
// Kills mutant: wrong fallback order.
func TestFallbackChain_Critique(t *testing.T) {
	t.Parallel()

	chain := llm.FallbackChain(llm.RoleCritique)

	require.Len(t, chain, 2)
	assert.Equal(t, llm.RoleThinking, chain[0])
	assert.Equal(t, llm.RoleAction, chain[1])
}

// TestRoleConstants verifies role string values.
// Kills mutant: wrong role constant values.
func TestRoleConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, llm.RoleAction, llm.Role("action"))
	assert.Equal(t, llm.RoleThinking, llm.Role("thinking"))
	assert.Equal(t, llm.RoleCritique, llm.Role("critique"))
	assert.Equal(t, llm.RoleCompact, llm.Role("compact"))
	assert.Equal(t, llm.RoleVision, llm.Role("vision"))
}
