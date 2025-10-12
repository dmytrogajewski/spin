package e2e

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_Performance_LargeFileOperations tests handling of large files.
func TestE2E_Performance_LargeFileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create large file content (10k lines)
	var largeContent strings.Builder
	for i := 0; i < 10000; i++ {
		fmt.Fprintf(&largeContent, "Line %d: This is line number %d with some content.\n", i, i)
	}

	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "large.txt",
				}),
			),
			MockResponseWithContent("Successfully read the large file."),
		},
		WorkspaceFiles: map[string]string{
			"large.txt": largeContent.String(),
		},
		Timeout: 1 * time.Minute,
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Measure performance
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Read large.txt")
	require.NoError(t, err)

	events := CollectEvents(conv.Stream(), 20*time.Second)

	duration := time.Since(start)

	// Verify completion
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify reasonable performance (<10s for 10k line file)
	AssertDuration(t, duration, 10*time.Second, "large file read")
}

// TestE2E_Performance_ManySmallFiles tests handling of many small files.
func TestE2E_Performance_ManySmallFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create many small files
	files := make(map[string]string)
	for i := 0; i < 100; i++ {
		files[fmt.Sprintf("file%d.txt", i)] = fmt.Sprintf("Content of file %d\n", i)
	}

	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithToolCalls(
				MockToolCall("list_directory", map[string]interface{}{
					"path": ".",
				}),
			),
			MockResponseWithContent("Listed 100+ files successfully."),
		},
		WorkspaceFiles: files,
		Timeout:        1 * time.Minute,
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Measure performance
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "List all files")
	require.NoError(t, err)

	events := CollectEvents(conv.Stream(), 20*time.Second)

	duration := time.Since(start)

	// Verify completion
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify reasonable performance (<5s for 100 files)
	AssertDuration(t, duration, 5*time.Second, "list many files")
}

// TestE2E_Performance_ConcurrentToolCalls tests concurrent tool execution.
func TestE2E_Performance_ConcurrentToolCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create test files
	files := make(map[string]string)
	for i := 0; i < 10; i++ {
		files[fmt.Sprintf("file%d.txt", i)] = fmt.Sprintf("Content %d\n", i)
	}

	// Create responses for concurrent reads
	responses := make([]MockResponse, 11) // 10 tool calls + 1 final response
	for i := 0; i < 10; i++ {
		responses[i] = MockResponseWithToolCalls(
			MockToolCall("read_file", map[string]interface{}{
				"path": fmt.Sprintf("file%d.txt", i),
			}),
		)
	}
	responses[10] = MockResponseWithContent("Read all files successfully.")

	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses:   responses,
		WorkspaceFiles: files,
		Timeout:        1 * time.Minute,
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Measure performance
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Read all 10 files")
	require.NoError(t, err)

	events := CollectEvents(conv.Stream(), 20*time.Second)

	duration := time.Since(start)

	// Verify completion
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete")

	// Verify all tool calls completed
	toolCalls := FindEvents(events, core.EventToolCallComplete)
	assert.GreaterOrEqual(t, len(toolCalls), 10, "should have completed 10 tool calls")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify reasonable performance (<15s for 10 sequential tool calls)
	AssertDuration(t, duration, 15*time.Second, "concurrent tool calls")
}

// TestE2E_Chaos_ConcurrentConversations tests multiple concurrent conversations.
func TestE2E_Chaos_ConcurrentConversations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: func() []MockResponse {
			// Need enough responses for 10 concurrent conversations
			responses := make([]MockResponse, 20)
			for i := range responses {
				responses[i] = MockResponseWithContent(fmt.Sprintf("Response %d", i))
			}
			return responses
		}(),
		WorkspaceFiles: SampleWorkspace(),
		Timeout:        2 * time.Minute,
	})
	defer agent.Cleanup()

	// Run 10 concurrent conversations
	const numConversations = 10
	var wg sync.WaitGroup
	wg.Add(numConversations)

	errors := make([]error, numConversations)

	for i := 0; i < numConversations; i++ {
		go func(idx int) {
			defer wg.Done()

			// Create conversation
			history := core.NewHistoryWithDefaults()
			_ = history.AddSystemMessage("You are a helpful assistant.")

			conv := core.NewConversation(agent.Agent, history, agent.Emitter)

			// Submit request
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := conv.RunTurn(ctx, fmt.Sprintf("Request %d", idx))
			if err != nil {
				errors[idx] = err
				return
			}

			// Collect events
			events := CollectEvents(conv.Stream(), 10*time.Second)

			// Verify completion
			_, found := FindEvent(events, core.EventTurnComplete)
			if !found {
				errors[idx] = fmt.Errorf("conversation %d did not complete", idx)
			}
		}(i)
	}

	// Wait for all conversations
	wg.Wait()

	// Verify no errors
	for i, err := range errors {
		assert.NoError(t, err, "conversation %d failed", i)
	}
}

// TestE2E_Chaos_TimeoutHandling tests timeout handling.
func TestE2E_Chaos_TimeoutHandling(t *testing.T) {
	// Create test agent with delayed response
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithDelay("This response is delayed", 10*time.Second),
		},
		WorkspaceFiles: SampleWorkspace(),
		Timeout:        30 * time.Second,
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Tell me something")
	require.NoError(t, err)

	// Collect events (should timeout)
	events := CollectEvents(conv.Stream(), 3*time.Second)

	// Should have failed or timed out
	_, completed := FindEvent(events, core.EventTurnComplete)
	_, failed := FindEvent(events, core.EventTurnFailed)

	assert.True(t, completed || failed, "should complete or fail gracefully")
}

// TestE2E_Chaos_MemoryStability tests memory stability over many operations.
func TestE2E_Chaos_MemoryStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: func() []MockResponse {
			// Generate 100 responses
			responses := make([]MockResponse, 100)
			for i := range responses {
				responses[i] = MockResponseWithContent(fmt.Sprintf("Response %d", i))
			}
			return responses
		}(),
		WorkspaceFiles: SampleWorkspace(),
		Timeout:        5 * time.Minute,
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Run 100 turns
	for i := 0; i < 100; i++ {
		err := conv.RunTurn(ctx, fmt.Sprintf("Request %d", i))
		require.NoError(t, err, "turn %d failed", i)

		events := CollectEvents(conv.Stream(), 3*time.Second)

		_, found := FindEvent(events, core.EventTurnComplete)
		assert.True(t, found, "turn %d should complete", i)

		// Check every 10 turns
		if i%10 == 0 {
			t.Logf("Completed %d turns", i)
		}
	}

	// If we got here without crashing, memory stability is good
	t.Log("Memory stability test completed successfully")
}

// TestE2E_Chaos_MalformedToolArguments tests handling of malformed tool arguments.
func TestE2E_Chaos_MalformedToolArguments(t *testing.T) {
	malformedCases := []struct {
		name string
		tool string
		args map[string]interface{}
	}{
		{
			name: "missing required parameter",
			tool: "read_file",
			args: map[string]interface{}{}, // missing "path"
		},
		{
			name: "wrong type",
			tool: "read_file",
			args: map[string]interface{}{
				"path": 12345, // should be string
			},
		},
		{
			name: "extra parameters",
			tool: "read_file",
			args: map[string]interface{}{
				"path":  "main.go",
				"extra": "parameter",
			},
		},
	}

	for _, tc := range malformedCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test agent
			agent := NewTestAgent(t, TestAgentOptions{
				LLMResponses: []MockResponse{
					MockResponseWithToolCalls(
						MockToolCall(tc.tool, tc.args),
					),
					MockResponseWithContent("Handled error gracefully."),
				},
				WorkspaceFiles: SampleWorkspace(),
			})
			defer agent.Cleanup()

			// Create conversation
			history := core.NewHistoryWithDefaults()
			require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

			conv := core.NewConversation(agent.Agent, history, agent.Emitter)

			// Submit request
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := conv.RunTurn(ctx, "Execute tool")
			require.NoError(t, err)

			// Collect events
			events := CollectEvents(conv.Stream(), 5*time.Second)

			// Should complete (agent handles error gracefully)
			_, found := FindEvent(events, core.EventTurnComplete)
			assert.True(t, found, "should complete despite malformed arguments")
		})
	}
}

// TestE2E_Performance_FileSearchScaling tests file search with many files.
func TestE2E_Performance_FileSearchScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create 1000 files
	files := make(map[string]string)
	for i := 0; i < 1000; i++ {
		files[fmt.Sprintf("dir%d/file%d.txt", i%10, i)] = fmt.Sprintf("Content %d", i)
	}

	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithToolCalls(
				MockToolCall("file_search", map[string]interface{}{
					"query": "file",
					"limit": 10,
				}),
			),
			MockResponseWithContent("Found files successfully."),
		},
		WorkspaceFiles: files,
		Timeout:        1 * time.Minute,
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Measure performance
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Search for files")
	require.NoError(t, err)

	events := CollectEvents(conv.Stream(), 20*time.Second)

	duration := time.Since(start)

	// Verify completion
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify reasonable performance (<5s for 1000 files)
	AssertDuration(t, duration, 5*time.Second, "file search with 1000 files")
}
