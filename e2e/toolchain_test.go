package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_ToolChain_SearchReadAnalyze tests file_search -> read_file workflow.
func TestE2E_ToolChain_SearchReadAnalyze(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// Step 1: Search for test files
			MockResponseWithToolCalls(
				MockToolCall("file_search", map[string]interface{}{
					"query": "go",
					"limit": 5,
				}),
			),
			// Step 2: Read first result
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "main.go",
				}),
			),
			// Step 3: Provide analysis
			MockResponseWithContent("I found main.go and analyzed it. It's a simple Go program that prints a message."),
		},
		WorkspaceFiles: SampleGoProject(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit request
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Search for Go files, read the first one, and analyze it")
	require.NoError(t, err)

	// Collect events
	events := CollectEvents(conv.Stream(), 10*time.Second)

	// Verify tool chain executed
	toolCalls := FindEvents(events, core.EventToolCallStart)
	assert.GreaterOrEqual(t, len(toolCalls), 2, "should have at least 2 tool calls")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify completion
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete successfully")
}

// TestE2E_ToolChain_GitModifyCommit tests git_context -> apply_patch -> execute_command workflow.
func TestE2E_ToolChain_GitModifyCommit(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// Step 1: Get git context
			MockResponseWithToolCalls(
				MockToolCall("git_context", map[string]interface{}{}),
			),
			// Step 2: Apply patch to modify file
			MockResponseWithToolCalls(
				MockToolCall("apply_patch", map[string]interface{}{
					"patch_text": TestFixtures.Patches["update_file"],
				}),
			),
			// Step 3: Stage changes
			MockResponseWithToolCalls(
				MockToolCall("execute_command", map[string]interface{}{
					"command": "git add .",
				}),
			),
			// Step 4: Provide summary
			MockResponseWithContent("I checked the git status, modified main.go, and staged the changes."),
		},
		WorkspaceFiles: SampleGoProject(),
		GitRepo:        true,
	})
	defer agent.Cleanup()

	// Create initial file to modify
	CreateTestFile(t, agent.Workspace, "main.go", `package main

import "fmt"

func main() {
	fmt.Println("Hello, Spin!")
}
`)

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit request
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Check git status, modify main.go to print 'Hello, E2E!', and stage the changes")
	require.NoError(t, err)

	// Collect events
	events := CollectEvents(conv.Stream(), 15*time.Second)

	// Verify tool chain executed
	toolCalls := FindEvents(events, core.EventToolCallStart)
	assert.GreaterOrEqual(t, len(toolCalls), 3, "should have at least 3 tool calls")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify completion
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete successfully")
}

// TestE2E_ToolChain_SearchMultiReadPatch tests file_search -> read_file (×3) -> apply_patch.
func TestE2E_ToolChain_SearchMultiReadPatch(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// Step 1: Search for Go files
			MockResponseWithToolCalls(
				MockToolCall("file_search", map[string]interface{}{
					"query": "*.go",
					"limit": 10,
				}),
			),
			// Step 2-4: Read multiple files
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "main.go",
				}),
			),
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "internal/server/server.go",
				}),
			),
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "pkg/util/util.go",
				}),
			),
			// Step 5: Apply patch to modify files
			MockResponseWithToolCalls(
				MockToolCall("apply_patch", map[string]interface{}{
					"patch_text": TestFixtures.Patches["multi_operation"],
				}),
			),
			// Step 6: Provide summary
			MockResponseWithContent("I found and read 3 Go files, then applied patches to modify them."),
		},
		WorkspaceFiles: SampleGoProject(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Search for Go files, read the top 3, and modify them")
	require.NoError(t, err)

	// Collect events
	events := CollectEvents(conv.Stream(), 20*time.Second)

	// Verify tool chain executed
	toolCalls := FindEvents(events, core.EventToolCallStart)
	assert.GreaterOrEqual(t, len(toolCalls), 5, "should have at least 5 tool calls")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify completion
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete successfully")
}

// TestE2E_ToolChain_PartialFailure tests tool chain with partial failure.
func TestE2E_ToolChain_PartialFailure(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// Step 1: Read existing file (success)
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "main.go",
				}),
			),
			// Step 2: Read non-existent file (will fail)
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "nonexistent.go",
				}),
			),
			// Step 3: Handle error gracefully
			MockResponseWithContent("I read main.go successfully, but nonexistent.go doesn't exist."),
		},
		WorkspaceFiles: SampleGoProject(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit request
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Read main.go and nonexistent.go")
	require.NoError(t, err)

	// Collect events
	events := CollectEvents(conv.Stream(), 10*time.Second)

	// Should have tool calls (at least the successful one)
	toolCalls := FindEvents(events, core.EventToolCallStart)
	assert.NotEmpty(t, toolCalls, "should have tool calls")

	// Should complete (agent handles partial failure)
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete despite partial failure")
}

// TestE2E_ToolChain_DataFlow tests data flowing between tools.
func TestE2E_ToolChain_DataFlow(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// Step 1: List directory to find files
			MockResponseWithToolCalls(
				MockToolCall("list_directory", map[string]interface{}{
					"path": ".",
				}),
			),
			// Step 2: Use list result to read specific file
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "config.toml",
				}),
			),
			// Step 3: Use file content to create new file
			MockResponseWithToolCalls(
				MockToolCall("write_file", map[string]interface{}{
					"path":    "config.backup.toml",
					"content": "# Backup of config.toml\n[server]\nhost = \"localhost\"\nport = 8080",
				}),
			),
			// Step 4: Confirm the operation
			MockResponseWithContent("I listed the directory, read config.toml, and created a backup file."),
		},
		WorkspaceFiles: SampleGoProject(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit request
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "List files, read config.toml, and create a backup")
	require.NoError(t, err)

	// Collect events
	events := CollectEvents(conv.Stream(), 15*time.Second)

	// Verify tool chain executed
	toolCalls := FindEvents(events, core.EventToolCallStart)
	assert.GreaterOrEqual(t, len(toolCalls), 3, "should have at least 3 tool calls")

	// Verify no errors
	AssertNoErrors(t, events)

	// Verify backup file was created
	backupPath := filepath.Join(agent.Workspace, "config.backup.toml")
	_, err = os.Stat(backupPath)
	assert.NoError(t, err, "backup file should exist")

	// Verify completion
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete successfully")
}

// TestE2E_ToolChain_CircularDependency tests handling of circular tool dependencies.
func TestE2E_ToolChain_CircularDependency(t *testing.T) {
	// Create test agent with timeout protection
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// Tool calls that could loop
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "file1.txt",
				}),
			),
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "file2.txt",
				}),
			),
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "file1.txt", // Same as first
				}),
			),
			// Should eventually complete
			MockResponseWithContent("Read the files as requested."),
		},
		WorkspaceFiles: map[string]string{
			"file1.txt": "Content 1",
			"file2.txt": "Content 2",
		},
		Timeout: 10 * time.Second,
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit request
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Read file1.txt and file2.txt")
	require.NoError(t, err)

	// Collect events with timeout
	events := CollectEvents(conv.Stream(), 12*time.Second)

	// Should have tool calls
	toolCalls := FindEvents(events, core.EventToolCallStart)
	assert.NotEmpty(t, toolCalls, "should have tool calls")

	// Should complete or timeout gracefully (not hang)
	_, found := FindEvent(events, core.EventTurnComplete)
	if !found {
		// If not completed, should have failed
		_, failed := FindEvent(events, core.EventTurnFailed)
		assert.True(t, failed, "should complete or fail, not hang")
	}
}
