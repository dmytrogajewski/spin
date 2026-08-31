package conversation

// Journey: specs/journeys/JOURNEY-015-assemble-retrieval-on-the-turn-path.md.

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/agent/harness/bridge"
	"github.com/dmytrogajewski/spin/internal/agent/scaffold"
	"github.com/dmytrogajewski/spin/internal/contexteng/history"
	"github.com/dmytrogajewski/spin/internal/contexteng/retrieval"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

const journey015Sentinel = "UNIQUE_JOURNEY015_COMPOSED_TURN_FRAGMENT"

type fakeTurnSource struct {
	content string
}

func (f fakeTurnSource) Name() string { return "fake-turn" }

func (f fakeTurnSource) Retrieve(
	_ context.Context, _ retrieval.Request,
) ([]retrieval.Fragment, error) {
	return []retrieval.Fragment{{Source: "fake-turn", Content: f.content}}, nil
}

type finishTurnCaller struct {
	saw []message.Message
}

func (c *finishTurnCaller) Call(
	_ context.Context, msgs []message.Message, _ []tools.ToolSchema, _ int,
) (content string, toolCalls []message.ToolCall, finishReason string, err error) {
	c.saw = append([]message.Message(nil), msgs...)

	return "done", nil, "stop", nil
}

func TestRunTurn_AssembledFragmentsInComposedTurn(t *testing.T) {
	t.Parallel()

	caller := &finishTurnCaller{}
	exec, err := harness.NewExecutor(
		&scaffold.Spec{SystemPrompt: "test", Config: scaffold.SpecConfig{MaxTurns: 1}},
		caller, noopDispatch{}, nil, nil, slog.Default(),
	)
	require.NoError(t, err)

	conv := &Conversation{
		emitter:           events.NewEventEmitter(8),
		history:           history.NewHistoryWithDefaults(),
		harnessExecutor:   bridge.NewTurnExecutor(exec),
		retrievalPipeline: retrieval.NewPipeline(fakeTurnSource{content: journey015Sentinel}),
		workDir:           t.TempDir(),
	}

	require.NoError(t, conv.RunTurn(t.Context(), "assemble me"))

	joined := joinTurnContents(caller.saw)
	require.Contains(t, joined, journey015Sentinel)
	require.Equal(t, 1, strings.Count(joined, journey015Sentinel))
	require.NotNil(t, conv.GetRetrievalPipeline())
}

func TestRunTurn_NilRetrievalPipeline_NoOp(t *testing.T) {
	t.Parallel()

	conv := &Conversation{
		emitter:         events.NewEventEmitter(8),
		history:         history.NewHistoryWithDefaults(),
		harnessExecutor: stubTurnOK{},
	}

	require.Nil(t, conv.GetRetrievalPipeline())
	require.NoError(t, conv.RunTurn(t.Context(), "hello"))
}

func joinTurnContents(msgs []message.Message) string {
	parts := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		parts = append(parts, msg.Content)
	}

	return strings.Join(parts, "\n")
}
