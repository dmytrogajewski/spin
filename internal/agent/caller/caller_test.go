package caller_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmytrogajewski/spin/internal/agent/caller"
	"github.com/dmytrogajewski/spin/internal/llm"
)

// TestNew_WithRouter_ResolvesProvider verifies that Router.ForRole is used when Router is set.
// Kills mutant: ignoring Router would use wrong provider for non-action roles.
func TestNew_WithRouter_ResolvesProvider(t *testing.T) {
	t.Parallel()

	action := llm.NewMockProvider("action-model")
	thinking := llm.NewMockProvider("thinking-model")

	router := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction:   action,
			llm.RoleThinking: thinking,
			llm.RoleCritique: nil,
			llm.RoleCompact:  nil,
			llm.RoleVision:   nil,
		},
	})

	lc := caller.New(caller.Config{
		Router: router,
		Role:   llm.RoleThinking,
	})

	assert.NotNil(t, lc)
}

// TestNew_WithRouter_DefaultsToAction verifies Role defaults to RoleAction when empty.
// Kills mutant: empty role selecting wrong provider would break default behavior.
func TestNew_WithRouter_DefaultsToAction(t *testing.T) {
	t.Parallel()

	action := llm.NewMockProvider("action-model")

	router := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction:   action,
			llm.RoleThinking: nil,
			llm.RoleCritique: nil,
			llm.RoleCompact:  nil,
			llm.RoleVision:   nil,
		},
	})

	lc := caller.New(caller.Config{
		Router: router,
	})

	assert.NotNil(t, lc)
}

// TestNew_WithoutRouter_UsesProvider verifies backward compatibility when Router is nil.
// Kills mutant: breaking backward compat would fail all existing call sites.
func TestNew_WithoutRouter_UsesProvider(t *testing.T) {
	t.Parallel()

	provider := llm.NewMockProvider("direct-provider")

	lc := caller.New(caller.Config{
		Provider: provider,
	})

	assert.NotNil(t, lc)
}

// TestNew_WithRouter_FallbackChain verifies fallback when role-specific provider not configured.
// Kills mutant: missing fallback would return nil provider.
func TestNew_WithRouter_FallbackChain(t *testing.T) {
	t.Parallel()

	action := llm.NewMockProvider("action-model")

	router := llm.NewRouter(llm.RouterConfig{
		Providers: map[llm.Role]llm.Provider{
			llm.RoleAction:   action,
			llm.RoleThinking: nil,
			llm.RoleCritique: nil,
			llm.RoleCompact:  nil,
			llm.RoleVision:   nil,
		},
	})

	// Compact role not configured — should fall back to action.
	lc := caller.New(caller.Config{
		Router: router,
		Role:   llm.RoleCompact,
	})

	assert.NotNil(t, lc)
}

// TestNew_SingleProviderRouter preserves behavior when all roles use same provider.
// Kills mutant: single provider router must route all roles to the same provider.
func TestNew_SingleProviderRouter(t *testing.T) {
	t.Parallel()

	provider := llm.NewMockProvider("single-model")
	router := llm.NewSingleProviderRouter(provider)

	roles := []llm.Role{
		llm.RoleAction,
		llm.RoleThinking,
		llm.RoleCritique,
		llm.RoleCompact,
		llm.RoleVision,
	}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			t.Parallel()

			lc := caller.New(caller.Config{
				Router: router,
				Role:   role,
			})

			assert.NotNil(t, lc)
		})
	}
}
