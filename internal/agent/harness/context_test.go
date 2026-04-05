package harness_test

// Journey: specs/journeys/JOURNEY-7.1.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/message"
)

// TestNewIterationContext_TrajectoryCtxIsNil verifies the default state.
// Kills mutant: pre-initializing TrajectoryCtx would waste allocation.
func TestNewIterationContext_TrajectoryCtxIsNil(t *testing.T) {
	t.Parallel()

	msgs := []message.Message{{Role: message.RoleUser, Content: "hello"}}
	iterCtx := harness.NewIterationContext(msgs)

	assert.Nil(t, iterCtx.TrajectoryCtx, "TrajectoryCtx should be nil by default")
	assert.NotNil(t, iterCtx.GuardFlags, "GuardFlags map should be initialized")
}

// TestNewIterationContext_MessagesPreserved verifies messages are stored.
// Kills mutant: dropping messages would lose conversation history.
func TestNewIterationContext_MessagesPreserved(t *testing.T) {
	t.Parallel()

	msgs := []message.Message{
		{Role: message.RoleSystem, Content: "system prompt"},
		{Role: message.RoleUser, Content: "query"},
	}
	iterCtx := harness.NewIterationContext(msgs)

	require.Len(t, iterCtx.Messages, 2)
	assert.Equal(t, "system prompt", iterCtx.Messages[0].Content)
	assert.Equal(t, "query", iterCtx.Messages[1].Content)
}

// TestExecute_CreatesTrajectoryCtx verifies trajectory is created per query.
// Kills mutant: not creating trajectory would leave nil for middleware.
func TestExecute_CreatesTrajectoryCtx(t *testing.T) {
	t.Parallel()

	const query = "install nodejs"

	// Middleware that captures the IterationContext to verify TrajectoryCtx.
	captureMW := &trajectoryCaptureMW{}
	caller := &stubCaller{content: testOutput}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, []harness.Middleware{captureMW})

	resp, err := exec.Execute(t.Context(), query, nil)

	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)
	require.NotNil(t, captureMW.capturedCtx, "middleware should have captured IterationContext")
	require.NotNil(t, captureMW.capturedCtx.TrajectoryCtx, "TrajectoryCtx should be set")
	assert.Equal(t, query, captureMW.capturedCtx.TrajectoryCtx.Query)
}

// TestExecute_TrajectoryCtxTurnUpdated verifies CurrentTurn is set before each turn.
// Kills mutant: not updating turn would leave stale trajectory state.
func TestExecute_TrajectoryCtxTurnUpdated(t *testing.T) {
	t.Parallel()

	const toolTurns = 2

	captureMW := &trajectoryCaptureMW{}
	caller := &multiTurnCaller{toolTurns: toolTurns}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, []harness.Middleware{captureMW})

	_, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	require.NotNil(t, captureMW.capturedCtx)
	require.NotNil(t, captureMW.capturedCtx.TrajectoryCtx)

	// After execution, CurrentTurn should reflect the last turn number.
	// 2 tool turns + 1 completion turn = turn index 2 (zero-based).
	expectedLastTurn := toolTurns
	assert.Equal(t, expectedLastTurn, captureMW.capturedCtx.TrajectoryCtx.CurrentTurn)
}

// TestExecute_TrajectoryCtxSameAcrossTurns verifies the same instance is reused.
// Kills mutant: creating new trajectory each turn would lose accumulated state.
func TestExecute_TrajectoryCtxSameAcrossTurns(t *testing.T) {
	t.Parallel()

	const toolTurns = 2

	// This middleware records the TrajectoryCtx pointer each BeforeTurn call.
	multiCaptureMW := &trajectoryMultiCaptureMW{}
	caller := &multiTurnCaller{toolTurns: toolTurns}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, []harness.Middleware{multiCaptureMW})

	_, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)

	// Should have 3 captured contexts (2 tool + 1 completion).
	expectedCaptures := toolTurns + 1
	require.Len(t, multiCaptureMW.captured, expectedCaptures)

	// All should point to the same trajectory instance.
	first := multiCaptureMW.captured[0]
	for i := 1; i < len(multiCaptureMW.captured); i++ {
		assert.Same(t, first, multiCaptureMW.captured[i],
			"turn %d should have same TrajectoryCtx instance", i)
	}
}

// TestExecute_AfterExecution_HasTrajectoryCtx verifies trajectory is accessible in AfterExecution.
// Kills mutant: clearing trajectory before AfterExecution would break post-processing.
func TestExecute_AfterExecution_HasTrajectoryCtx(t *testing.T) {
	t.Parallel()

	afterMW := &trajectoryAfterMW{}
	caller := &stubCaller{content: testOutput}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, []harness.Middleware{afterMW})

	_, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	require.NotNil(t, afterMW.afterTrajectory, "TrajectoryCtx should be available in AfterExecution")
	assert.Equal(t, testQuery, afterMW.afterTrajectory.Query)
}

// trajectoryCaptureMW captures IterationContext from the last BeforeTurn call.
type trajectoryCaptureMW struct {
	capturedCtx *harness.IterationContext
}

func (m *trajectoryCaptureMW) BeforeTurn(_ context.Context, iterCtx *harness.IterationContext) {
	m.capturedCtx = iterCtx
}

func (m *trajectoryCaptureMW) AfterExecution(
	_ context.Context, _ *harness.IterationContext, _ *harness.Response,
) {
}

// trajectoryMultiCaptureMW captures TrajectoryCtx pointer from each BeforeTurn call.
type trajectoryMultiCaptureMW struct {
	captured []*trajectory.Context
}

func (m *trajectoryMultiCaptureMW) BeforeTurn(_ context.Context, iterCtx *harness.IterationContext) {
	m.captured = append(m.captured, iterCtx.TrajectoryCtx)
}

func (m *trajectoryMultiCaptureMW) AfterExecution(
	_ context.Context, _ *harness.IterationContext, _ *harness.Response,
) {
}

// trajectoryAfterMW captures TrajectoryCtx from AfterExecution.
type trajectoryAfterMW struct {
	afterTrajectory *trajectory.Context
}

func (m *trajectoryAfterMW) BeforeTurn(_ context.Context, _ *harness.IterationContext) {
}

func (m *trajectoryAfterMW) AfterExecution(
	_ context.Context, iterCtx *harness.IterationContext, _ *harness.Response,
) {
	m.afterTrajectory = iterCtx.TrajectoryCtx
}
