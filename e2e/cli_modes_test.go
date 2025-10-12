package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/core/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_CLI_ModeFlagRegular tests --mode regular flag.
func TestE2E_CLI_ModeFlagRegular(t *testing.T) {
	// Create test agent in regular mode
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithToolCalls(
				MockToolCall("write_file", map[string]interface{}{
					"path":    "test.txt",
					"content": "test content",
				}),
			),
			MockResponseWithContent("File created successfully."),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Verify agent has regular task mode (all tools)
	registry := agent.Agent.GetTaskRegistry()
	taskMode, err := registry.Get("regular")
	require.NoError(t, err, "regular task mode should exist")

	allowedTools := taskMode.AllowedTools()
	assert.Contains(t, allowedTools, "write_file", "regular mode should have write_file")
	assert.Contains(t, allowedTools, "execute_command", "regular mode should have execute_command")
	assert.Contains(t, allowedTools, "apply_patch", "regular mode should have apply_patch")
	assert.Greater(t, len(allowedTools), 5, "regular mode should have many tools")
}

// TestE2E_CLI_ModeFlagReview tests --mode review flag (read-only).
func TestE2E_CLI_ModeFlagReview(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithContent("Code review complete. No issues found."),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Get review task mode
	registry := agent.Agent.GetTaskRegistry()
	reviewTask, err := registry.Get("review")
	require.NoError(t, err, "review task mode should exist")

	allowedTools := reviewTask.AllowedTools()

	// Verify review mode has read-only tools
	assert.Contains(t, allowedTools, "read_file", "review mode should have read_file")
	assert.Contains(t, allowedTools, "list_directory", "review mode should have list_directory")
	assert.Contains(t, allowedTools, "get_context", "review mode should have get_context")
	assert.Contains(t, allowedTools, "file_search", "review mode should have file_search")
	assert.Contains(t, allowedTools, "git_context", "review mode should have git_context")

	// Verify review mode does NOT have write tools
	assert.NotContains(t, allowedTools, "write_file", "review mode should not have write_file")
	assert.NotContains(t, allowedTools, "execute_command", "review mode should not have execute_command")
	assert.NotContains(t, allowedTools, "apply_patch", "review mode should not have apply_patch")

	// Verify tool count is limited
	assert.Equal(t, 5, len(allowedTools), "review mode should have exactly 5 tools")
}

// TestE2E_CLI_ModeFlagCompact tests --mode compact flag (minimal tools).
func TestE2E_CLI_ModeFlagCompact(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithContent("Quick answer: 42"),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Get compact task mode
	registry := agent.Agent.GetTaskRegistry()
	compactTask, err := registry.Get("compact")
	require.NoError(t, err, "compact task mode should exist")

	allowedTools := compactTask.AllowedTools()

	// Verify compact mode has minimal tools
	assert.Contains(t, allowedTools, "read_file", "compact mode should have read_file")
	assert.Contains(t, allowedTools, "get_context", "compact mode should have get_context")
	assert.Contains(t, allowedTools, "file_search", "compact mode should have file_search")

	// Verify compact mode has ONLY these tools
	assert.Equal(t, 3, len(allowedTools), "compact mode should have exactly 3 tools")

	// Verify compact mode has 4K token budget
	assert.Equal(t, 4096, compactTask.MaxTokens(), "compact mode should have 4096 token budget")
}

// TestE2E_CLI_ModeFlagPlanning tests --mode planning flag (context only).
func TestE2E_CLI_ModeFlagPlanning(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithContent("Task breakdown:\n1. Step one\n2. Step two"),
		},
		WorkspaceFiles: SampleWorkspace(),
		GitRepo:        true,
	})
	defer agent.Cleanup()

	// Get planning task mode
	registry := agent.Agent.GetTaskRegistry()
	planningTask, err := registry.Get("planning")
	require.NoError(t, err, "planning task mode should exist")

	allowedTools := planningTask.AllowedTools()

	// Verify planning mode has context tools only
	assert.Contains(t, allowedTools, "get_context", "planning mode should have get_context")
	assert.Contains(t, allowedTools, "file_search", "planning mode should have file_search")
	assert.Contains(t, allowedTools, "git_context", "planning mode should have git_context")

	// Verify planning mode does NOT have file operations
	assert.NotContains(t, allowedTools, "read_file", "planning mode should not have read_file")
	assert.NotContains(t, allowedTools, "write_file", "planning mode should not have write_file")
	assert.NotContains(t, allowedTools, "list_directory", "planning mode should not have list_directory")

	// Verify planning mode has ONLY context tools
	assert.Equal(t, 3, len(allowedTools), "planning mode should have exactly 3 tools")

	// Verify planning mode has 4K token budget
	assert.Equal(t, 4096, planningTask.MaxTokens(), "planning mode should have 4096 token budget")
}

// TestE2E_CLI_ModeSwitchingInConversation tests mode switching mid-conversation.
func TestE2E_CLI_ModeSwitchingInConversation(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// First message in regular mode
			MockResponseWithContent("Regular mode response"),
			// Second message in review mode
			MockResponseWithContent("Review mode response"),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	// Create manager and conversation
	cfg := &core.Config{
		MaxTurns: 10,
	}
	mgr, err := core.NewManager(cfg,
		core.WithLLMProvider(agent.LLM),
		core.WithManagerToolRegistry(agent.Agent.GetToolRegistry()),
		core.WithManagerTaskRegistry(agent.Agent.GetTaskRegistry()),
	)
	require.NoError(t, err, "failed to create manager")

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, agent.Workspace)
	require.NoError(t, err, "failed to create conversation")

	// Start in default (regular) mode
	assert.Equal(t, "regular", conv.GetTaskMode(), "should start in regular mode")

	// Send first message
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()

	events1, err := conv.SendMessage(ctx1, "First message")
	require.NoError(t, err, "first message should succeed")

	// Collect first events
	collected1 := CollectEvents(events1, 3*time.Second)
	AssertNoErrors(t, collected1)

	// Switch to review mode
	err = conv.SetTaskMode("review")
	require.NoError(t, err, "mode switch should succeed")
	assert.Equal(t, "review", conv.GetTaskMode(), "should be in review mode")

	// Send second message (in review mode)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	events2, err := conv.SendMessage(ctx2, "Second message")
	require.NoError(t, err, "second message should succeed")

	// Collect second events
	collected2 := CollectEvents(events2, 3*time.Second)
	AssertNoErrors(t, collected2)

	// Verify both messages succeeded
	_, found1 := FindEvent(collected1, core.EventTurnComplete)
	assert.True(t, found1, "first turn should complete")

	_, found2 := FindEvent(collected2, core.EventTurnComplete)
	assert.True(t, found2, "second turn should complete")
}

// TestE2E_CLI_InvalidModeHandling tests error handling for invalid modes.
func TestE2E_CLI_InvalidModeHandling(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create manager and conversation
	cfg := &core.Config{
		MaxTurns: 10,
	}
	mgr, err := core.NewManager(cfg,
		core.WithLLMProvider(agent.LLM),
		core.WithManagerToolRegistry(agent.Agent.GetToolRegistry()),
		core.WithManagerTaskRegistry(agent.Agent.GetTaskRegistry()),
	)
	require.NoError(t, err, "failed to create manager")

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, agent.Workspace)
	require.NoError(t, err, "failed to create conversation")

	// Try to switch to invalid mode
	err = conv.SetTaskMode("invalid-mode")
	assert.Error(t, err, "should error on invalid mode")
	assert.Contains(t, err.Error(), "task not found", "error should mention task not found")

	// Verify mode unchanged
	assert.Equal(t, "regular", conv.GetTaskMode(), "should remain in regular mode")
}

// TestE2E_CLI_ModeAffectsToolFiltering tests that mode restrictions are enforced.
func TestE2E_CLI_ModeAffectsToolFiltering(t *testing.T) {
	// Create test agent with tool call that should be filtered
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithContent("Analysis complete."),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Get review and regular tasks
	registry := agent.Agent.GetTaskRegistry()

	reviewTask, err := registry.Get("review")
	require.NoError(t, err)

	regularTask, err := registry.Get("regular")
	require.NoError(t, err)

	// Build tools for each task
	reviewTools, err := agent.Agent.BuildToolsForTask(reviewTask)
	require.NoError(t, err, "should build tools for review mode")

	regularTools, err := agent.Agent.BuildToolsForTask(regularTask)
	require.NoError(t, err, "should build tools for regular mode")

	// Verify review mode has fewer tools than regular mode
	assert.Less(t, len(reviewTools), len(regularTools),
		"review mode should have fewer tools than regular mode")

	// Verify specific tools are filtered
	reviewToolNames := make(map[string]bool)
	for _, tool := range reviewTools {
		reviewToolNames[tool.Function.Name] = true
	}

	regularToolNames := make(map[string]bool)
	for _, tool := range regularTools {
		regularToolNames[tool.Function.Name] = true
	}

	// write_file should be in regular but not review
	assert.True(t, regularToolNames["write_file"], "regular mode should have write_file")
	assert.False(t, reviewToolNames["write_file"], "review mode should not have write_file")

	// execute_command should be in regular but not review
	assert.True(t, regularToolNames["execute_command"], "regular mode should have execute_command")
	assert.False(t, reviewToolNames["execute_command"], "review mode should not have execute_command")

	// read_file should be in both
	assert.True(t, regularToolNames["read_file"], "regular mode should have read_file")
	assert.True(t, reviewToolNames["read_file"], "review mode should have read_file")
}

// TestE2E_CLI_TokenBudgetsPerMode tests that token budgets are applied correctly.
func TestE2E_CLI_TokenBudgetsPerMode(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Get all task modes
	registry := agent.Agent.GetTaskRegistry()

	tests := []struct {
		modeName      string
		expectedTokens int
	}{
		{"regular", 16384},
		{"review", 12288},
		{"compact", 4096},
		{"planning", 4096},
	}

	for _, tt := range tests {
		t.Run(tt.modeName, func(t *testing.T) {
			taskMode, err := registry.Get(tt.modeName)
			require.NoError(t, err, "should get %s task mode", tt.modeName)

			actualTokens := taskMode.MaxTokens()
			assert.Equal(t, tt.expectedTokens, actualTokens,
				"%s mode should have %d token budget", tt.modeName, tt.expectedTokens)
		})
	}
}

// TestE2E_CLI_AllModesRegistered tests that all 4 modes are registered by default.
func TestE2E_CLI_AllModesRegistered(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Get list of registered modes
	modes := agent.Agent.ListTaskModes()

	// Verify all 4 modes are present
	expectedModes := []string{"regular", "review", "compact", "planning"}

	for _, expectedMode := range expectedModes {
		assert.Contains(t, modes, expectedMode,
			"should have %s mode registered", expectedMode)
	}

	// Verify we have exactly 4 modes
	assert.Equal(t, 4, len(modes), "should have exactly 4 task modes")
}

// TestE2E_CLI_ConversationWithTaskMode tests creating conversation with specific mode.
func TestE2E_CLI_ConversationWithTaskMode(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithContent("Response in review mode"),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create manager
	cfg := &core.Config{
		MaxTurns: 10,
	}
	mgr, err := core.NewManager(cfg,
		core.WithLLMProvider(agent.LLM),
		core.WithManagerToolRegistry(agent.Agent.GetToolRegistry()),
		core.WithManagerTaskRegistry(agent.Agent.GetTaskRegistry()),
	)
	require.NoError(t, err, "failed to create manager")

	ctx := context.Background()

	// Create conversation directly in review mode
	conv, err := mgr.NewConversationWithTask(ctx, agent.Workspace, "review")
	require.NoError(t, err, "failed to create conversation with task mode")

	// Verify conversation starts in review mode
	assert.Equal(t, "review", conv.GetTaskMode(), "should start in review mode")

	// Send message
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()

	events, err := conv.SendMessage(ctx1, "Review this code")
	require.NoError(t, err, "message should succeed")

	// Collect events
	collected := CollectEvents(events, 3*time.Second)
	AssertNoErrors(t, collected)

	// Verify turn completed
	_, found := FindEvent(collected, core.EventTurnComplete)
	assert.True(t, found, "turn should complete")

	// Verify still in review mode
	assert.Equal(t, "review", conv.GetTaskMode(), "should remain in review mode")
}

// TestE2E_CLI_CustomTaskRegistry tests using a custom task registry.
func TestE2E_CLI_CustomTaskRegistry(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create custom task registry with only compact mode
	customRegistry := task.NewRegistry()
	customRegistry.Register("compact", task.NewCompact())
	customRegistry.SetDefault("compact")

	// Create manager with custom registry
	cfg := &core.Config{
		MaxTurns: 10,
	}
	mgr, err := core.NewManager(cfg,
		core.WithLLMProvider(agent.LLM),
		core.WithManagerToolRegistry(agent.Agent.GetToolRegistry()),
		core.WithManagerTaskRegistry(customRegistry),
	)
	require.NoError(t, err, "failed to create manager with custom registry")

	ctx := context.Background()

	// Create conversation - should use custom registry
	conv, err := mgr.NewConversation(ctx, agent.Workspace)
	require.NoError(t, err, "failed to create conversation")

	// Verify conversation uses compact mode as default
	assert.Equal(t, "compact", conv.GetTaskMode(), "should use compact mode from custom registry")

	// Try to switch to regular mode (should fail - not in custom registry)
	err = conv.SetTaskMode("regular")
	assert.Error(t, err, "should error when switching to mode not in custom registry")
	assert.Contains(t, err.Error(), "task not found", "error should mention task not found")
}

// TestE2E_CLI_ModeSystemPrompts tests that each mode has a distinct system prompt.
func TestE2E_CLI_ModeSystemPrompts(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	registry := agent.Agent.GetTaskRegistry()

	tests := []struct {
		modeName           string
		shouldContain      []string
		shouldNotContain   []string
	}{
		{
			modeName: "regular",
			shouldContain: []string{"regular"},
			shouldNotContain: []string{},
		},
		{
			modeName: "review",
			shouldContain: []string{"review", "read-only", "analysis"},
			shouldNotContain: []string{"write", "modify"},
		},
		{
			modeName: "compact",
			shouldContain: []string{"compact", "quick", "minimal"},
			shouldNotContain: []string{},
		},
		{
			modeName: "planning",
			shouldContain: []string{"planning", "decomposition", "task"},
			shouldNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.modeName, func(t *testing.T) {
			taskMode, err := registry.Get(tt.modeName)
			require.NoError(t, err, "should get %s task mode", tt.modeName)

			systemPrompt := taskMode.SystemPrompt()
			assert.NotEmpty(t, systemPrompt, "%s mode should have system prompt", tt.modeName)

			// Check required content
			for _, substring := range tt.shouldContain {
				assert.True(t,
					strings.Contains(strings.ToLower(systemPrompt), strings.ToLower(substring)),
					"%s mode system prompt should contain %q", tt.modeName, substring)
			}

			// Check prohibited content
			for _, substring := range tt.shouldNotContain {
				assert.False(t,
					strings.Contains(strings.ToLower(systemPrompt), strings.ToLower(substring)),
					"%s mode system prompt should not contain %q", tt.modeName, substring)
			}
		})
	}
}
