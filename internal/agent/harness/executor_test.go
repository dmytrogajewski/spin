package harness_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/agent/scaffold"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Journey: specs/journeys/JOURNEY-2.1.md.

// Sentinel errors for test doubles.
var (
	errCallerUnavailable = errors.New("LLM unavailable")
	errGuardFailure      = errors.New("guard failure")
	errCallerFail        = errors.New("caller fail")
)

const (
	testQuery      = "hello agent"
	testOutput     = "I am done"
	testToolName   = "test_tool"
	testToolCallID = "call_001"
	testMaxTurns   = 10
)

// stubCaller returns pre-configured content and tool calls.
type stubCaller struct {
	content      string
	toolCalls    []message.ToolCall
	finishReason string
	err          error
	callCount    int
}

func (s *stubCaller) Call(
	_ context.Context, _ []message.Message, _ []tools.ToolSchema, _ int,
) (string, []message.ToolCall, string, error) {
	s.callCount++

	return s.content, s.toolCalls, s.finishReason, s.err
}

// multiTurnCaller returns tool calls for the first N turns, then text only.
type multiTurnCaller struct {
	toolTurns int
	callCount int
}

func (m *multiTurnCaller) Call(
	_ context.Context, _ []message.Message, _ []tools.ToolSchema, _ int,
) (string, []message.ToolCall, string, error) {
	m.callCount++

	if m.callCount <= m.toolTurns {
		return "", []message.ToolCall{{
			ID:   testToolCallID,
			Type: "function",
			Function: message.ToolCallFunction{
				Name:      testToolName,
				Arguments: "{}",
			},
		}}, "", nil
	}

	return testOutput, nil, "stop", nil
}

// stubDispatcher records dispatch calls and returns messages with tool results appended.
type stubDispatcher struct {
	dispatchCount int
}

func (s *stubDispatcher) Dispatch(
	_ context.Context, msgs []message.Message,
	content string, toolCalls []message.ToolCall,
) []message.Message {
	s.dispatchCount++

	result := make([]message.Message, 0, len(msgs)+len(toolCalls)+1)
	result = append(result, msgs...)

	if content != "" || len(toolCalls) > 0 {
		result = append(result, message.Message{
			Role:    message.RoleAssistant,
			Content: content,
		})
	}

	for _, tc := range toolCalls {
		result = append(result, message.Message{
			Role:       message.RoleTool,
			Content:    "tool result",
			ToolCallID: tc.ID,
		})
	}

	return result
}

// stubGuard optionally halts or injects messages.
type stubGuard struct {
	halt     bool
	injected []message.Message
	err      error
	checks   int
}

func (s *stubGuard) Check(
	_ context.Context, _ *harness.IterationContext,
	_ string, _ []message.ToolCall,
) ([]message.Message, bool, error) {
	s.checks++

	return s.injected, s.halt, s.err
}

// stubMiddleware records BeforeTurn and AfterExecution calls.
type stubMiddleware struct {
	beforeTurnCount     int
	afterExecutionCount int
	lastResponse        *harness.Response
}

func (s *stubMiddleware) BeforeTurn(_ context.Context, _ *harness.IterationContext) {
	s.beforeTurnCount++
}

func (s *stubMiddleware) AfterExecution(
	_ context.Context, _ *harness.IterationContext, resp *harness.Response,
) {
	s.afterExecutionCount++
	s.lastResponse = resp
}

func testSpec(maxTurns int) *scaffold.Spec {
	return &scaffold.Spec{
		SystemPrompt: "You are a test agent.",
		ToolSchemas: []tools.ToolSchema{{
			Type: "function",
			Function: tools.FunctionSchema{
				Name:        testToolName,
				Description: "A test tool",
			},
		}},
		Config: scaffold.SpecConfig{
			MaxTurns: maxTurns,
		},
	}
}

func newTestExecutor(
	t *testing.T,
	caller harness.Caller,
	dispatcher harness.ToolDispatcher,
	guards []harness.Guard,
	middlewares []harness.Middleware,
) *harness.Executor {
	t.Helper()

	exec, err := harness.NewExecutor(
		testSpec(testMaxTurns), caller, dispatcher, guards, middlewares,
		slog.Default(),
	)
	require.NoError(t, err)

	return exec
}

// TestNewExecutor_Valid verifies that a valid executor is created.
// Kills mutant: returning nil executor on valid inputs.
func TestNewExecutor_Valid(t *testing.T) {
	t.Parallel()

	exec, err := harness.NewExecutor(
		testSpec(testMaxTurns),
		&stubCaller{content: testOutput},
		&stubDispatcher{},
		nil, nil,
		slog.Default(),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

// TestNewExecutor_NilSpec verifies spec nil-check.
// Kills mutant: accepting nil spec would panic later.
func TestNewExecutor_NilSpec(t *testing.T) {
	t.Parallel()

	_, err := harness.NewExecutor(nil, &stubCaller{}, &stubDispatcher{}, nil, nil, nil)
	require.ErrorIs(t, err, harness.ErrNilSpec)
}

// TestNewExecutor_NilCaller verifies caller nil-check.
// Kills mutant: accepting nil caller would panic on Call.
func TestNewExecutor_NilCaller(t *testing.T) {
	t.Parallel()

	_, err := harness.NewExecutor(testSpec(testMaxTurns), nil, &stubDispatcher{}, nil, nil, nil)
	require.ErrorIs(t, err, harness.ErrNilCaller)
}

// TestNewExecutor_NilDispatcher verifies dispatcher nil-check.
// Kills mutant: accepting nil dispatcher would panic on Dispatch.
func TestNewExecutor_NilDispatcher(t *testing.T) {
	t.Parallel()

	_, err := harness.NewExecutor(testSpec(testMaxTurns), &stubCaller{}, nil, nil, nil, nil)
	require.ErrorIs(t, err, harness.ErrNilDispatcher)
}

// TestNewExecutor_NilLoggerUsesDefault verifies fallback to [slog.Default].
// Kills mutant: nil logger would panic on first log call.
func TestNewExecutor_NilLoggerUsesDefault(t *testing.T) {
	t.Parallel()

	exec, err := harness.NewExecutor(
		testSpec(testMaxTurns), &stubCaller{content: testOutput},
		&stubDispatcher{}, nil, nil, nil,
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

// TestExecute_ImplicitCompletion verifies that text-only LLM response stops the loop.
// Kills mutant: not checking len(toolCalls)==0 would continue looping.
func TestExecute_ImplicitCompletion(t *testing.T) {
	t.Parallel()

	caller := &stubCaller{content: testOutput, finishReason: "stop"}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, nil)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)
	assert.Equal(t, harness.FinishReasonStop, resp.FinishReason)
	assert.Equal(t, 1, caller.callCount)
	assert.Positive(t, resp.Duration.Nanoseconds())
}

// TestExecute_MaxTurnsLimit verifies the loop stops after MaxTurns iterations.
// Kills mutant: ignoring MaxTurns would loop forever.
func TestExecute_MaxTurnsLimit(t *testing.T) {
	t.Parallel()

	const maxTurns = 3

	// Caller always returns tool calls, never completes implicitly.
	caller := &stubCaller{
		toolCalls: []message.ToolCall{{
			ID:   testToolCallID,
			Type: "function",
			Function: message.ToolCallFunction{
				Name:      testToolName,
				Arguments: "{}",
			},
		}},
	}
	dispatcher := &stubDispatcher{}

	exec, err := harness.NewExecutor(
		testSpec(maxTurns), caller, dispatcher, nil, nil, slog.Default(),
	)
	require.NoError(t, err)

	resp, execErr := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, execErr)
	assert.Equal(t, harness.FinishReasonMaxTurn, resp.FinishReason)
	assert.Equal(t, maxTurns, caller.callCount)
	assert.Equal(t, maxTurns, dispatcher.dispatchCount)
}

// TestExecute_ContextCancellation verifies that a canceled context stops the loop.
// Kills mutant: not checking ctx.Err() would ignore cancellation.
func TestExecute_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	caller := &stubCaller{content: testOutput}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, nil)

	resp, err := exec.Execute(ctx, testQuery, nil)

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, harness.FinishReasonTimeout, resp.FinishReason)
	assert.Equal(t, 0, caller.callCount)
}

// TestExecute_GuardHalt verifies that a guard can halt the loop.
// Kills mutant: ignoring guard halt would continue the loop.
func TestExecute_GuardHalt(t *testing.T) {
	t.Parallel()

	caller := &multiTurnCaller{toolTurns: testMaxTurns}
	guard := &stubGuard{halt: true}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, []harness.Guard{guard}, nil)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, harness.FinishReasonGuard, resp.FinishReason)
	assert.Equal(t, 1, guard.checks)
	assert.Equal(t, 1, caller.callCount)
}

// TestExecute_GuardInjectsMessages verifies that guard-injected messages are appended.
// Kills mutant: discarding injected messages would lose guard context.
func TestExecute_GuardInjectsMessages(t *testing.T) {
	t.Parallel()

	injected := []message.Message{{
		Role:    message.RoleSystem,
		Content: "guard warning",
	}}

	caller := &multiTurnCaller{toolTurns: 1}
	guard := &stubGuard{injected: injected}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, []harness.Guard{guard}, nil)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)

	// The guard-injected message should appear in final messages.
	found := false

	for _, msg := range resp.Messages {
		if msg.Content == "guard warning" {
			found = true

			break
		}
	}

	assert.True(t, found, "guard-injected message should be in response messages")
}

// TestExecute_BeforeTurnMiddleware verifies BeforeTurn is called each iteration.
// Kills mutant: skipping middleware would break hooks.
func TestExecute_BeforeTurnMiddleware(t *testing.T) {
	t.Parallel()

	const toolTurns = 2

	caller := &multiTurnCaller{toolTurns: toolTurns}
	mw := &stubMiddleware{}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, []harness.Middleware{mw})

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)

	// BeforeTurn called: 2 tool turns + 1 completion turn = 3.
	expectedTurns := toolTurns + 1
	assert.Equal(t, expectedTurns, mw.beforeTurnCount)
}

// TestExecute_AfterExecutionMiddleware verifies AfterExecution is called once.
// Kills mutant: not calling AfterExecution would break post-loop hooks.
func TestExecute_AfterExecutionMiddleware(t *testing.T) {
	t.Parallel()

	caller := &stubCaller{content: testOutput}
	mw := &stubMiddleware{}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, []harness.Middleware{mw})

	_, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, 1, mw.afterExecutionCount)
	require.NotNil(t, mw.lastResponse)
	assert.Equal(t, testOutput, mw.lastResponse.Output)
}

// TestExecute_CallerError verifies that caller errors propagate.
// Kills mutant: swallowing caller errors would hide failures.
func TestExecute_CallerError(t *testing.T) {
	t.Parallel()

	caller := &stubCaller{err: errCallerUnavailable}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, nil)

	_, err := exec.Execute(t.Context(), testQuery, nil)

	require.Error(t, err)
	require.ErrorIs(t, err, errCallerUnavailable)
}

// TestExecute_GuardError verifies that guard errors propagate.
// Kills mutant: swallowing guard errors would hide check failures.
func TestExecute_GuardError(t *testing.T) {
	t.Parallel()

	caller := &multiTurnCaller{toolTurns: 1}
	guard := &stubGuard{err: errGuardFailure}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, []harness.Guard{guard}, nil)

	_, err := exec.Execute(t.Context(), testQuery, nil)

	require.Error(t, err)
	require.ErrorIs(t, err, errGuardFailure)
}

// TestExecute_ToolCallsRecorded verifies that completed tool calls are in the response.
// Kills mutant: not recording tool calls loses execution history.
func TestExecute_ToolCallsRecorded(t *testing.T) {
	t.Parallel()

	const toolTurns = 2

	caller := &multiTurnCaller{toolTurns: toolTurns}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, nil)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Len(t, resp.ToolCalls, toolTurns)
}

// TestExecute_HistoryPreserved verifies that provided history is included in messages.
// Kills mutant: ignoring history would lose conversation context.
func TestExecute_HistoryPreserved(t *testing.T) {
	t.Parallel()

	history := []message.Message{{
		Role:    message.RoleAssistant,
		Content: "previous response",
	}}

	caller := &stubCaller{content: testOutput}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, nil)

	resp, err := exec.Execute(t.Context(), testQuery, history)

	require.NoError(t, err)
	require.NotEmpty(t, resp.Messages)
	assert.Equal(t, "previous response", resp.Messages[0].Content)
}

// TestExecute_MultiTurnDispatch verifies tool dispatch across multiple turns.
// Kills mutant: not dispatching would skip tool execution.
func TestExecute_MultiTurnDispatch(t *testing.T) {
	t.Parallel()

	const toolTurns = 3

	caller := &multiTurnCaller{toolTurns: toolTurns}
	dispatcher := &stubDispatcher{}
	exec := newTestExecutor(t, caller, dispatcher, nil, nil)

	resp, err := exec.Execute(t.Context(), testQuery, nil)

	require.NoError(t, err)
	assert.Equal(t, testOutput, resp.Output)
	assert.Equal(t, harness.FinishReasonStop, resp.FinishReason)
	assert.Equal(t, toolTurns, dispatcher.dispatchCount)
}

// TestExecute_AfterExecutionCalledOnError verifies AfterExecution runs even on error.
// Kills mutant: skipping AfterExecution on error would break cleanup hooks.
func TestExecute_AfterExecutionCalledOnError(t *testing.T) {
	t.Parallel()

	caller := &stubCaller{err: errCallerFail}
	mw := &stubMiddleware{}
	exec := newTestExecutor(t, caller, &stubDispatcher{}, nil, []harness.Middleware{mw})

	_, err := exec.Execute(t.Context(), testQuery, nil)

	require.Error(t, err)
	assert.Equal(t, 1, mw.afterExecutionCount)
}
