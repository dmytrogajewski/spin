package bridge_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/harness/bridge"
	agenttool "github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Journey: specs/journeys/JOURNEY-2.8.md.

const (
	testToolName   = "read_file"
	testToolCallID = "call_001"
	testContent    = "tool output content"
)

// TestNewCallerBridge verifies construction does not return nil.
// Kills mutant: nil CallerBridge would panic on Call.
func TestNewCallerBridge(t *testing.T) {
	t.Parallel()

	cb := bridge.NewCallerBridge(nil, nil, agent.CallParams{})

	assert.NotNil(t, cb)
}

// TestNewDispatcherBridge verifies construction does not return nil.
// Kills mutant: nil DispatcherBridge would panic on Dispatch.
func TestNewDispatcherBridge(t *testing.T) {
	t.Parallel()

	db := bridge.NewDispatcherBridge(nil)

	assert.NotNil(t, db)
}

// TestDispatcherBridge_AppendsAssistantMessage verifies assistant message is appended.
// Kills mutant: missing assistant message would break OpenAI format.
func TestDispatcherBridge_AppendsAssistantMessage(t *testing.T) {
	t.Parallel()

	runtime := createTestRuntime(t)
	db := bridge.NewDispatcherBridge(runtime)

	initial := []message.Message{
		{Role: message.RoleUser, Content: "hello"},
	}

	toolCalls := []message.ToolCall{{
		ID:   testToolCallID,
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      testToolName,
			Arguments: `{"path": "main.go"}`,
		},
	}}

	result := db.Dispatch(t.Context(), initial, "thinking", toolCalls)

	// Should have: 1 user + 1 assistant + 1 tool = 3.
	expectedLen := 3
	require.Len(t, result, expectedLen)
	assert.Equal(t, message.RoleAssistant, result[1].Role)
	assert.Equal(t, "thinking", result[1].Content)
	assert.Len(t, result[1].ToolCalls, 1)
}

// TestDispatcherBridge_AppendsToolResults verifies tool results are appended.
// Kills mutant: missing tool results would break the conversation.
func TestDispatcherBridge_AppendsToolResults(t *testing.T) {
	t.Parallel()

	runtime := createTestRuntime(t)
	db := bridge.NewDispatcherBridge(runtime)

	toolCalls := []message.ToolCall{{
		ID:   testToolCallID,
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      testToolName,
			Arguments: `{"path": "main.go"}`,
		},
	}}

	result := db.Dispatch(t.Context(), nil, "", toolCalls)

	// Should have: 1 assistant + 1 tool = 2.
	expectedLen := 2
	require.Len(t, result, expectedLen)
	assert.Equal(t, message.RoleTool, result[1].Role)
	assert.Equal(t, testToolCallID, result[1].ToolCallID)
}

// TestDispatcherBridge_PreservesExistingMessages verifies existing messages are kept.
// Kills mutant: discarding history would lose conversation context.
func TestDispatcherBridge_PreservesExistingMessages(t *testing.T) {
	t.Parallel()

	runtime := createTestRuntime(t)
	db := bridge.NewDispatcherBridge(runtime)

	initial := []message.Message{
		{Role: message.RoleUser, Content: "msg1"},
		{Role: message.RoleAssistant, Content: "msg2"},
	}

	toolCalls := []message.ToolCall{{
		ID:   testToolCallID,
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      testToolName,
			Arguments: `{"path": "main.go"}`,
		},
	}}

	result := db.Dispatch(t.Context(), initial, "", toolCalls)

	// Should have: 2 existing + 1 assistant + 1 tool = 4.
	expectedLen := 4
	require.Len(t, result, expectedLen)
	assert.Equal(t, "msg1", result[0].Content)
	assert.Equal(t, "msg2", result[1].Content)
}

// TestBuildExecutor_NilSpec verifies error on nil spec.
// Kills mutant: nil spec would cause panic in executor.
func TestBuildExecutor_NilSpec(t *testing.T) {
	t.Parallel()

	_, err := bridge.BuildExecutor(bridge.Config{
		Spec: nil,
	})

	require.Error(t, err)
}

// createTestRuntime creates a tool runtime with a test tool for dispatcher tests.
func createTestRuntime(t *testing.T) *agenttool.Runtime {
	t.Helper()

	reg := tools.NewRegistry()

	err := reg.Register(&testTool{name: testToolName, output: testContent})
	require.NoError(t, err)

	return agenttool.NewRuntime(agenttool.RuntimeConfig{
		Registry: reg,
	})
}

// testTool implements tools.Tool for testing.
type testTool struct {
	name   string
	output string
}

func (t *testTool) Name() string        { return t.name }
func (t *testTool) Description() string { return "test tool" }
func (t *testTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Function: tools.FunctionSchema{Name: t.name},
	}
}

func (t *testTool) Execute(_ context.Context, _ tools.ToolParameters) (tools.ToolResult, error) {
	return tools.ToolResult{Output: t.output, Success: true}, nil
}

// errorTool returns a failed ToolResult with an error message.
type errorTool struct {
	name   string
	errMsg string
}

func (t *errorTool) Name() string        { return t.name }
func (t *errorTool) Description() string { return "error test tool" }
func (t *errorTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Function: tools.FunctionSchema{Name: t.name},
	}
}

func (t *errorTool) Execute(_ context.Context, _ tools.ToolParameters) (tools.ToolResult, error) {
	return tools.ToolResult{Success: false, Error: t.errMsg}, nil
}

// TestDispatcherBridge_ToolError_SurfacesErrorMessage reproduces the bug where
// tool errors (Success: false, Error: "...") were silently dropped because the
// dispatcher only read toolResult.Output (empty for errors), causing the LLM to
// receive an empty tool result and hallucinate success.
func TestDispatcherBridge_ToolError_SurfacesErrorMessage(t *testing.T) {
	t.Parallel()

	const (
		errToolName = "denied_tool"
		errMessage  = "command execution denied by user"
	)

	reg := tools.NewRegistry()

	regErr := reg.Register(&errorTool{name: errToolName, errMsg: errMessage})
	require.NoError(t, regErr)

	runtime := agenttool.NewRuntime(agenttool.RuntimeConfig{
		Registry: reg,
	})

	db := bridge.NewDispatcherBridge(runtime)

	toolCalls := []message.ToolCall{{
		ID:   "call-denied",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      errToolName,
			Arguments: `{}`,
		},
	}}

	result := db.Dispatch(t.Context(), nil, "", toolCalls)

	// Should have: 1 assistant + 1 tool = 2.
	require.Len(t, result, 2)

	toolMsg := result[1]
	assert.Equal(t, message.RoleTool, toolMsg.Role)

	// The tool result MUST contain the error message — not be empty.
	assert.Contains(t, toolMsg.Content, errMessage,
		"tool error must be surfaced in message content so the LLM sees it")
	assert.Contains(t, toolMsg.Content, "Error:",
		"error prefix must be present for LLM to recognize failure")
}

// outputErrorTool returns a failed ToolResult with BOTH output and error.
// This mimics shell_command where compilation output is in Output and the
// exit status is in Error.
type outputErrorTool struct {
	name   string
	output string
	errMsg string
}

func (t *outputErrorTool) Name() string        { return t.name }
func (t *outputErrorTool) Description() string { return "output+error test tool" }
func (t *outputErrorTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Function: tools.FunctionSchema{Name: t.name},
	}
}

func (t *outputErrorTool) Execute(_ context.Context, _ tools.ToolParameters) (tools.ToolResult, error) {
	return tools.ToolResult{Success: false, Output: t.output, Error: t.errMsg}, nil
}

// TestDispatcherBridge_ToolErrorWithOutput_IncludesBoth reproduces the bug
// where shell_command compilation errors (in Output) were lost because the
// dispatcher only surfaced Error, discarding the actual compiler output.
func TestDispatcherBridge_ToolErrorWithOutput_IncludesBoth(t *testing.T) {
	t.Parallel()

	const (
		toolName        = "compile_tool"
		compilerOutput  = "error[E0308]: mismatched types\n  --> src/main.rs:5:10"
		exitStatusError = "execution failed: exit status 101"
	)

	reg := tools.NewRegistry()

	regErr := reg.Register(&outputErrorTool{
		name:   toolName,
		output: compilerOutput,
		errMsg: exitStatusError,
	})
	require.NoError(t, regErr)

	runtime := agenttool.NewRuntime(agenttool.RuntimeConfig{
		Registry: reg,
	})

	db := bridge.NewDispatcherBridge(runtime)

	toolCalls := []message.ToolCall{{
		ID:   "call-compile",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      toolName,
			Arguments: `{}`,
		},
	}}

	result := db.Dispatch(t.Context(), nil, "", toolCalls)

	require.Len(t, result, 2)

	toolMsg := result[1]

	// MUST contain BOTH the compiler output AND the error status.
	assert.Contains(t, toolMsg.Content, compilerOutput,
		"compiler output must be visible to the LLM")
	assert.Contains(t, toolMsg.Content, exitStatusError,
		"exit status must be visible to the LLM")
}
