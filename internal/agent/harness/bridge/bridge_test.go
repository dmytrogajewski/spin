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
