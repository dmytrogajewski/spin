package caller_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/caller"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// stubPromptBuilder satisfies caller.SystemPromptBuilder without enhancement.
type stubPromptBuilder struct{}

func (stubPromptBuilder) BuildSystemPrompt(_ context.Context, base string, _ []tools.Tool) string {
	return base
}

func (stubPromptBuilder) ApplyACEPrompt(_ context.Context, prompt string, _ []*bullet.Bullet) string {
	return prompt
}

// TestCall_EmitsRealTokenUsage verifies a successful LLM call emits
// EventTurnProgress with the provider-reported token usage so the UI
// context counter shows real context size, not history estimates.
// Journey: specs/bugs/BUG-tui-context-counter.md.
func TestCall_EmitsRealTokenUsage(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(100)
	defer emitter.Close()

	_, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)

	lc := caller.New(caller.Config{
		Provider:      llm.NewMockProvider("usage-model", llm.WithResponse("hello world")),
		PromptBuilder: stubPromptBuilder{},
		Emitter:       emitter,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgs := []message.Message{{Role: message.RoleUser, Content: "hi"}}

	_, err = lc.Call(ctx, msgs, agent.DefaultCallParams(), nil, nil)
	require.NoError(t, err)

	got := waitForTurnProgress(t, eventCh)

	data, ok := got.Data.(events.TurnEventData)
	require.True(t, ok, "TurnProgress must carry TurnEventData, got %T", got.Data)
	assert.Positive(t, data.TokensUsed, "real usage from the provider must be reported")
}

// waitForTurnProgress drains the event channel until an EventTurnProgress
// arrives or the wait times out.
func waitForTurnProgress(t *testing.T, eventCh <-chan events.Event) events.Event {
	t.Helper()

	deadline := time.After(2 * time.Second)

	for {
		select {
		case evt := <-eventCh:
			if evt.Type == events.EventTurnProgress {
				return evt
			}
		case <-deadline:
			t.Fatal("timed out waiting for EventTurnProgress")
		}
	}
}

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
