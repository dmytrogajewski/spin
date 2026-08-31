package harness_test

// Journey: specs/journeys/JOURNEY-015-assemble-retrieval-on-the-turn-path.md.

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/contexteng/retrieval"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

const retrievedSentinel = "UNIQUE_JOURNEY015_RETRIEVED_FRAGMENT"

type fakeRetrievalSource struct {
	fragments []retrieval.Fragment
	calls     int
}

func (f *fakeRetrievalSource) Name() string { return "fake" }

func (f *fakeRetrievalSource) Retrieve(
	_ context.Context, _ retrieval.Request,
) ([]retrieval.Fragment, error) {
	f.calls++

	return f.fragments, nil
}

type recordingCaller struct {
	messages []message.Message
}

func (r *recordingCaller) Call(
	_ context.Context, msgs []message.Message, _ []tools.ToolSchema, _ int,
) (content string, toolCalls []message.ToolCall, finishReason string, err error) {
	r.messages = append([]message.Message(nil), msgs...)

	return testOutput, nil, "stop", nil
}

func TestExecute_RetrievalPipeline_InjectsFragments(t *testing.T) {
	t.Parallel()

	src := &fakeRetrievalSource{
		fragments: []retrieval.Fragment{{
			Source:  "fake",
			Content: retrievedSentinel,
		}},
	}
	caller := &recordingCaller{}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{},
		harness.WithRetrievalPipeline(retrieval.NewPipeline(src)),
	)

	_, err := exec.Execute(t.Context(), testQuery, nil)
	require.NoError(t, err)

	joined := joinMessageContents(caller.messages)
	require.Contains(t, joined, retrievedSentinel)
	require.Equal(t, 1, strings.Count(joined, retrievedSentinel))
}

func TestExecute_RetrievalPipeline_NilIsNoOp(t *testing.T) {
	t.Parallel()

	caller := &recordingCaller{}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{})

	_, err := exec.Execute(t.Context(), testQuery, nil)
	require.NoError(t, err)
	require.NotContains(t, joinMessageContents(caller.messages), "# Retrieved Context")
}

func TestExecute_RetrievalPipeline_EmptyFragmentsSkipHeading(t *testing.T) {
	t.Parallel()

	src := &fakeRetrievalSource{}
	caller := &recordingCaller{}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{},
		harness.WithRetrievalPipeline(retrieval.NewPipeline(src)),
	)

	_, err := exec.Execute(t.Context(), testQuery, nil)
	require.NoError(t, err)
	require.NotContains(t, joinMessageContents(caller.messages), "# Retrieved Context")
}

func TestExecute_RetrievalPipeline_DoesNotDuplicateFragment(t *testing.T) {
	t.Parallel()

	src := &fakeRetrievalSource{
		fragments: []retrieval.Fragment{{
			Source:  "fake",
			Content: retrievedSentinel,
		}},
	}
	caller := &recordingCaller{}
	exec := newContextEngExecutor(t, caller, &stubDispatcher{},
		harness.WithRetrievalPipeline(retrieval.NewPipeline(src)),
	)

	_, err := exec.Execute(t.Context(), testQuery, []message.Message{{
		Role:    message.RoleUser,
		Content: retrievedSentinel,
	}})
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(joinMessageContents(caller.messages), retrievedSentinel))
}

func TestExecute_RetrievalPipeline_AssemblesOncePerExecute(t *testing.T) {
	t.Parallel()

	src := &fakeRetrievalSource{
		fragments: []retrieval.Fragment{{
			Source:  "fake",
			Content: retrievedSentinel,
		}},
	}
	exec := newContextEngExecutor(t, &multiTurnCaller{toolTurns: 1}, &stubDispatcher{},
		harness.WithRetrievalPipeline(retrieval.NewPipeline(src)),
	)

	_, err := exec.Execute(t.Context(), testQuery, nil)
	require.NoError(t, err)
	require.Equal(t, 1, src.calls)
}

func joinMessageContents(msgs []message.Message) string {
	parts := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		parts = append(parts, msg.Content)
	}

	return strings.Join(parts, "\n")
}
