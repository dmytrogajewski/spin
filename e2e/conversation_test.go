package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_FullConversation_SimpleQuery tests a simple query flow.
func TestE2E_FullConversation_SimpleQuery(t *testing.T) {
	// Create test agent with a simple response
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithContent("The current git branch is main."),
		},
		WorkspaceFiles: SampleWorkspace(),
		GitRepo:        true,
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit user input
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "What is the current git branch?")
	require.NoError(t, err, "failed to submit user input")

	// Collect events
	events := CollectEvents(conv.Stream(), 3*time.Second)

	// Verify we got events
	assert.NotEmpty(t, events, "should have received events")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify turn completed
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should have turn complete event")

	// Verify content was streamed
	contentEvents := FindEvents(events, core.EventContentDelta)
	assert.NotEmpty(t, contentEvents, "should have content delta events")
}

// TestE2E_FullConversation_WithToolCall tests a conversation with tool execution.
func TestE2E_FullConversation_WithToolCall(t *testing.T) {
	// Create test agent with tool call response
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// First response: tool call
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "main.go",
				}),
			),
			// Second response: final answer
			MockResponseWithContent("The file contains a simple Go program that prints 'Hello, Spin!'."),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit user input
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "What is in main.go?")
	require.NoError(t, err, "failed to submit user input")

	// Collect events
	events := CollectEvents(conv.Stream(), 5*time.Second)

	// Verify we got events
	assert.NotEmpty(t, events, "should have received events")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify tool call was executed
	toolCallStartEvents := FindEvents(events, core.EventToolCallStart)
	assert.NotEmpty(t, toolCallStartEvents, "should have tool call start events")

	toolCallCompleteEvents := FindEvents(events, core.EventToolCallComplete)
	assert.NotEmpty(t, toolCallCompleteEvents, "should have tool call complete events")

	// Verify turn completed
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should have turn complete event")
}

// TestE2E_FullConversation_MultiStep tests a multi-step task.
func TestE2E_FullConversation_MultiStep(t *testing.T) {
	// Create test agent with multi-step responses
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// Step 1: Search for files
			MockResponseWithToolCalls(
				MockToolCall("file_search", map[string]interface{}{
					"query": "test",
					"limit": 5,
				}),
			),
			// Step 2: Read first file
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "main.go",
				}),
			),
			// Step 3: Provide summary
			MockResponseWithContent("I found test-related files and read main.go. It contains a simple Go program."),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit user input
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Search for test files, read the first one, and summarize it")
	require.NoError(t, err, "failed to submit user input")

	// Collect events
	events := CollectEvents(conv.Stream(), 10*time.Second)

	// Verify we got events
	assert.NotEmpty(t, events, "should have received events")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify multiple tool calls
	toolCallStartEvents := FindEvents(events, core.EventToolCallStart)
	assert.GreaterOrEqual(t, len(toolCallStartEvents), 2, "should have at least 2 tool calls")

	// Verify turn completed
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should have turn complete event")
}

// TestE2E_FullConversation_ErrorRecovery tests error handling in conversation.
func TestE2E_FullConversation_ErrorRecovery(t *testing.T) {
	// Create test agent with invalid tool call that will fail
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// First response: invalid tool call (missing required param)
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					// Missing required "path" parameter
				}),
			),
			// Second response: retry with correct params
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "main.go",
				}),
			),
			// Third response: final answer
			MockResponseWithContent("Successfully read the file after retry."),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit user input
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Read main.go")
	require.NoError(t, err, "failed to submit user input")

	// Collect events
	events := CollectEvents(conv.Stream(), 10*time.Second)

	// Verify we got events
	assert.NotEmpty(t, events, "should have received events")

	// Should have at least one tool call complete (successful retry)
	toolCallCompleteEvents := FindEvents(events, core.EventToolCallComplete)
	assert.NotEmpty(t, toolCallCompleteEvents, "should have tool call complete events")

	// Verify turn completed (agent recovered from error)
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should have turn complete event after recovery")
}

// TestE2E_MultiTurnConversation_ContextPreservation tests context across turns.
func TestE2E_MultiTurnConversation_ContextPreservation(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// Turn 1: List files
			MockResponseWithToolCalls(
				MockToolCall("list_directory", map[string]interface{}{
					"path": ".",
				}),
			),
			MockResponseWithContent("The directory contains: main.go, config.toml, README.md"),

			// Turn 2: Read first file (context: "first one" refers to Turn 1)
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "main.go",
				}),
			),
			MockResponseWithContent("The file contains a Go program."),

			// Turn 3: Summarize (context: "what you found" refers to Turns 1-2)
			MockResponseWithContent("I found 3 files and read main.go which is a Go program."),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Turn 1
	err := conv.RunTurn(ctx, "What files are in the current directory?")
	require.NoError(t, err, "turn 1 failed")

	events1 := CollectEvents(conv.Stream(), 5*time.Second)
	AssertNoErrors(t, events1)
	_, found := FindEvent(events1, core.EventTurnComplete)
	require.True(t, found, "turn 1 should complete")

	// Turn 2 (references Turn 1)
	err = conv.RunTurn(ctx, "Read the first one")
	require.NoError(t, err, "turn 2 failed")

	events2 := CollectEvents(conv.Stream(), 5*time.Second)
	AssertNoErrors(t, events2)
	_, found = FindEvent(events2, core.EventTurnComplete)
	require.True(t, found, "turn 2 should complete")

	// Turn 3 (references Turns 1-2)
	err = conv.RunTurn(ctx, "Summarize what you found")
	require.NoError(t, err, "turn 3 failed")

	events3 := CollectEvents(conv.Stream(), 5*time.Second)
	AssertNoErrors(t, events3)
	_, found = FindEvent(events3, core.EventTurnComplete)
	require.True(t, found, "turn 3 should complete")

	// Verify conversation has history from all 3 turns
	messages := history.Messages()
	assert.GreaterOrEqual(t, len(messages), 6, "should have at least 6 messages (3 user + 3 assistant)")
}

// TestE2E_MultiTurnConversation_HistoryTruncation tests history management.
func TestE2E_MultiTurnConversation_HistoryTruncation(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: func() []MockResponse {
			// Generate 50 responses for 50 turns
			responses := make([]MockResponse, 50)
			for i := 0; i < 50; i++ {
				responses[i] = MockResponseWithContent("Response " + string(rune('A'+i%26)))
			}
			return responses
		}(),
		WorkspaceFiles: SampleWorkspace(),
		Timeout:        2 * time.Minute,
	})
	defer agent.Cleanup()

	// Create conversation with limited history size
	history := core.NewHistory(1000, &core.SimpleTokenizer{}) // Small token limit to trigger truncation
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Submit 50 turns
	for i := 0; i < 50; i++ {
		err := conv.RunTurn(ctx, "Tell me something")
		require.NoError(t, err, "turn %d failed", i)

		events := CollectEvents(conv.Stream(), 5*time.Second)
		AssertNoErrors(t, events)

		_, found := FindEvent(events, core.EventTurnComplete)
		require.True(t, found, "turn %d should complete", i)
	}

	// Verify history was truncated (should not have all 50 turns)
	messages := history.Messages()
	assert.Less(t, len(messages), 100, "history should be truncated (less than 50 turns * 2 messages)")

	// Verify system message is preserved
	assert.Equal(t, "system", messages[0].Role, "system message should be preserved")
}
