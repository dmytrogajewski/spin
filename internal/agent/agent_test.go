package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/factory"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAgent tests the refactored agent creation with services
func TestNewAgent(t *testing.T) {
	tests := []struct {
		name          string
		provider      llm.Provider
		security      *security.SecurityService
		detection     *detection.DetectionService
		orchestration *orchestration.OrchestrationService
		environment   *Environment
		emitter       *events.EventEmitter
		wantErr       bool
		errContains   string
	}{
		{
			name:     "valid agent",
			provider: llm.NewMockProvider("test"),
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       events.NewEventEmitter(100),
			wantErr:       false,
		},
		{
			name:     "nil provider",
			provider: nil,
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       events.NewEventEmitter(100),
			wantErr:       true,
			errContains:   "LLM provider cannot be nil",
		},
		{
			name:          "nil security",
			provider:      llm.NewMockProvider("test"),
			security:      nil,
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       events.NewEventEmitter(100),
			wantErr:       true,
			errContains:   "security service cannot be nil",
		},
		{
			name:     "nil detection",
			provider: llm.NewMockProvider("test"),
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     nil,
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       events.NewEventEmitter(100),
			wantErr:       true,
			errContains:   "detection service cannot be nil",
		},
		{
			name:     "nil orchestration",
			provider: llm.NewMockProvider("test"),
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: nil,
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       events.NewEventEmitter(100),
			wantErr:       true,
			errContains:   "orchestration service cannot be nil",
		},
		{
			name:     "nil environment",
			provider: llm.NewMockProvider("test"),
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   nil,
			emitter:       events.NewEventEmitter(100),
			wantErr:       true,
			errContains:   "context cannot be nil",
		},
		{
			name:     "nil emitter",
			provider: llm.NewMockProvider("test"),
			security: func() *security.SecurityService {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalService(nil, emitter, validator)
				return security.NewSecurityService(validator, approvalService)
			}(),
			detection:     detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			orchestration: orchestration.NewOrchestrationService(nil, tools.NewRegistry(), orchestration.NewRegistry()),
			environment:   &Environment{WorkDir: "/tmp"},
			emitter:       nil,
			wantErr:       true,
			errContains:   "event emitter cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(tt.provider, tt.security, tt.detection, tt.orchestration, tt.environment, tt.emitter)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, agent)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, agent)
			}
		})
	}
}

// TestAgent_ListTaskModes tests the ListTaskModes method
func TestAgent_ListTaskModes(t *testing.T) {
	agent := createTestAgentWithServices(t)

	modes := agent.ListTaskModes()

	assert.NotNil(t, modes)
	assert.GreaterOrEqual(t, len(modes), 4) // At least 4 built-in modes
	assert.Contains(t, modes, "regular")
	assert.Contains(t, modes, "review")
	assert.Contains(t, modes, "compact")
	assert.Contains(t, modes, "planning")
}

// TestAgent_Execute_Integration is a minimal integration test
func TestAgent_Execute_Integration(t *testing.T) {
	t.Skip("Integration test - requires full setup")

	agent := createTestAgentWithServices(t)

	req := &AgentRequest{
		Input:    "Hello, how are you?",
		TaskName: "regular",
	}

	ctx := context.Background()
	resp, err := agent.Execute(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// createTestAgentWithServices creates a fully configured agent for testing.
// This helper consolidates agent creation across all test files.
func createTestAgentWithServices(t *testing.T) *Agent {
	t.Helper()

	llmProvider := llm.NewMockProvider("test")
	validator := security.NewValidator()
	workDir := t.TempDir()
	env := &Environment{WorkDir: workDir}
	emitter := events.NewEventEmitter(100)

	// Build SecurityService
	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	// Build DetectionService
	cycleConfig := cycle.Config{
		WindowSize:       3,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
		Enabled:          true,
	}
	cycleDetector := cycle.NewDetector(cycleConfig)
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	// Build tool registry with built-in tools
	toolRegistry := tools.NewRegistry()
	executor, err := NewExecutor(workDir)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	_ = toolRegistry.Register(tools.NewReadFileTool())
	_ = toolRegistry.Register(tools.NewWriteFileTool())
	_ = toolRegistry.Register(tools.NewListDirectoryTool())
	_ = toolRegistry.Register(tools.NewExecuteCommandTool(executor, validator))
	_ = toolRegistry.Register(tools.NewGetContextTool(env))
	_ = toolRegistry.Register(tools.NewApplyPatchTool(workDir))
	_ = toolRegistry.Register(tools.NewFileSearchTool(workDir))
	_ = toolRegistry.Register(tools.NewGitContextTool(workDir))

	// Build task registry (using orchestration.Registry, not task.Registry)
	taskRegistry := orchestration.NewRegistry()
	_ = taskRegistry.Register("regular", task.NewRegular())
	_ = taskRegistry.Register("review", task.NewReview())
	_ = taskRegistry.Register("compact", task.NewCompact())
	_ = taskRegistry.Register("planning", task.NewPlanning())
	_ = taskRegistry.SetDefault("regular")

	// Build OrchestrationService
	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         workDir,
	})
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	// Create agent
	agent, err := NewAgent(llmProvider, securityService, detectionService, orchestrationService, env, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	return agent
}

// newAgentForTest is a test-only wrapper that provides the old NewAgent signature
// but builds services internally. This allows existing tests to continue working.
// USE THIS IN TESTS INSTEAD OF NewAgent TO AVOID UPDATING EVERY TEST FILE.
func newAgentForTest(
	provider llm.Provider,
	executor *Executor,
	validator *security.Validator,
	environment *Environment,
	emitter *events.EventEmitter,
	opts ...AgentOption,
) (*Agent, error) {
	// Build SecurityService
	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	// Build DetectionService
	cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	// Build tool registry
	toolRegistry := tools.NewRegistry()
	_ = toolRegistry.Register(tools.NewReadFileTool())
	_ = toolRegistry.Register(tools.NewWriteFileTool())
	_ = toolRegistry.Register(tools.NewListDirectoryTool())
	_ = toolRegistry.Register(tools.NewExecuteCommandTool(executor, validator))
	_ = toolRegistry.Register(tools.NewGetContextTool(environment))
	_ = toolRegistry.Register(tools.NewApplyPatchTool(environment.WorkDir))
	_ = toolRegistry.Register(tools.NewFileSearchTool(environment.WorkDir))
	_ = toolRegistry.Register(tools.NewGitContextTool(environment.WorkDir))

	// Build task registry (using orchestration.Registry, not task.Registry)
	taskRegistry := orchestration.NewRegistry()
	_ = taskRegistry.Register("regular", task.NewRegular())
	_ = taskRegistry.Register("review", task.NewReview())
	_ = taskRegistry.Register("compact", task.NewCompact())
	_ = taskRegistry.Register("planning", task.NewPlanning())
	_ = taskRegistry.SetDefault("regular")

	// Build OrchestrationService
	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         environment.WorkDir,
	})
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	// Create agent with services using the real NewAgent function
	agent := &Agent{
		llm:           provider,
		security:      securityService,
		detection:     detectionService,
		orchestration: orchestrationService,
		context:       environment,
		emitter:       emitter,
		config:        DefaultConfig(),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(agent); err != nil {
			return nil, fmt.Errorf("applying option: %w", err)
		}
	}

	return agent, nil
}

// TestToolExecutionBugReproduction reproduces the exact bug from the user's output:
// The LLM calls list_directory but it's not executed, causing cycle detection.
func TestToolExecutionBugReproduction(t *testing.T) {
	ctx := context.Background()

	// Setup: Create mock LLM that returns list_directory tool call
	mockLLM := llm.NewMockProvider("test",
		llm.WithToolCalls([]llm.ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: llm.FunctionCall{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
		}),
	)

	// Setup tool registry with list_directory
	toolRegistry := tools.NewRegistry()
	err := toolRegistry.Register(tools.NewListDirectoryTool())
	require.NoError(t, err)

	// Setup services
	validator := security.NewValidator()
	securityService := security.NewSecurityService(validator, nil)

	cycleDetector := cycle.NewDetector(cycle.Config{
		WindowSize:       10,
		SimilarityThresh: 0.8,
	})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry: toolRegistry,
	})
	orchestrationService := orchestration.NewOrchestrationService(
		toolExecutor,
		toolRegistry,
		nil, // taskRegistry
	)

	env := &Environment{
		WorkDir:     "/tmp",
		Environment: make(map[string]string),
	}

	emitter := events.NewEventEmitter(100)

	// Collect events
	subID, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)
	defer emitter.Unsubscribe(subID)

	var toolStartEvents []events.Event
	var toolCompleteEvents []events.Event
	done := make(chan struct{})

	go func() {
		defer close(done)
		for evt := range eventCh {
			switch evt.Type {
			case events.EventToolCallStart:
				toolStartEvents = append(toolStartEvents, evt)
				t.Logf("Tool start event: %+v", evt.Data)
			case events.EventToolCallComplete:
				toolCompleteEvents = append(toolCompleteEvents, evt)
				t.Logf("Tool complete event: %+v", evt.Data)
			}
		}
	}()

	// Create agent
	agent, err := NewAgent(
		mockLLM,
		securityService,
		detectionService,
		orchestrationService,
		env,
		emitter,
	)
	require.NoError(t, err)

	// Create a simple task
	task := &simpleTask{
		name:         "test",
		systemPrompt: "You are a test assistant",
		allowedTools: []string{}, // Allow all tools
		maxTokens:    4096,
	}

	// Execute: Send request that should trigger tool call
	req := &AgentRequest{
		Input: "list files in current directory",
		Task:  task,
	}

	resp, err := agent.Execute(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Close emitter and wait for event collection
	emitter.Close()
	<-done

	// Assert: Tool should have been executed
	t.Logf("Tool start events: %d", len(toolStartEvents))
	t.Logf("Tool complete events: %d", len(toolCompleteEvents))

	assert.NotEmpty(t, toolStartEvents, "BUG: Tool was called but no start event emitted")
	assert.NotEmpty(t, toolCompleteEvents, "BUG: Tool was called but no complete event emitted")

	// Check event data
	if len(toolStartEvents) > 0 {
		startData, ok := toolStartEvents[0].Data.(events.ToolCallStartData)
		assert.True(t, ok, "Should have ToolCallStartData")
		assert.Equal(t, "list_directory", startData.ToolName)
	}

	if len(toolCompleteEvents) > 0 {
		completeData, ok := toolCompleteEvents[0].Data.(events.ToolCallCompleteData)
		assert.True(t, ok, "Should have ToolCallCompleteData")
		assert.Equal(t, "list_directory", completeData.ToolName)
		assert.True(t, completeData.Success, "Tool execution should succeed")
	}
}

// TestToolExecutionWithRealToolCall tests that processToolCall actually executes the tool.
func TestToolExecutionWithRealToolCall(t *testing.T) {
	ctx := context.Background()

	// Setup tool registry
	toolRegistry := tools.NewRegistry()
	err := toolRegistry.Register(tools.NewListDirectoryTool())
	require.NoError(t, err)

	// Setup services
	validator := security.NewValidator()
	securityService := security.NewSecurityService(validator, nil)

	cycleDetector := cycle.NewDetector(cycle.Config{WindowSize: 10})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry: toolRegistry,
	})
	orchestrationService := orchestration.NewOrchestrationService(
		toolExecutor,
		toolRegistry,
		nil,
	)

	env := &Environment{
		WorkDir:     "/tmp",
		Environment: make(map[string]string),
	}

	emitter := events.NewEventEmitter(100)

	agent, err := NewAgent(
		&dummyLLM{}, // Won't be used, we're calling ProcessToolCall directly
		securityService,
		detectionService,
		orchestrationService,
		env,
		emitter,
	)
	require.NoError(t, err)

	// Create tool call directly
	toolCall := &orchestration.ToolCall{
		ID:   "test_call",
		Type: "function",
		Function: orchestration.ToolCallFunction{
			Name:      "list_directory",
			Arguments: `{"path": "/tmp"}`,
		},
	}

	// Process tool call
	result, err := agent.ProcessToolCall(ctx, toolCall)
	require.NoError(t, err, "ProcessToolCall should not error")
	require.NotNil(t, result, "ProcessToolCall should return result")

	// Verify result
	assert.True(t, result.Success, "Tool execution should succeed")
	assert.NotEmpty(t, result.Output, "Tool should produce output")
	t.Logf("Tool output: %s", result.Output)
}

// TestStreamProcessingWithToolCalls tests that tool calls are extracted from stream.
func TestStreamProcessingWithToolCalls(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Mock LLM that streams tool call
	mockLLM := llm.NewMockProvider("test",
		llm.WithToolCalls([]llm.ToolCall{
			{
				ID:   "stream_call",
				Type: "function",
				Function: llm.FunctionCall{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
		}),
	)

	// Test streaming
	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "list files"},
		},
	}

	chunks, err := mockLLM.Stream(ctx, req)
	require.NoError(t, err)

	var receivedToolCalls []llm.ToolCall
	for chunk := range chunks {
		if chunk.ToolCall != nil {
			receivedToolCalls = append(receivedToolCalls, *chunk.ToolCall)
			t.Logf("Received tool call in stream: %s", chunk.ToolCall.Function.Name)
		}
	}

	assert.Len(t, receivedToolCalls, 1, "Should receive tool call in stream")
	assert.Equal(t, "list_directory", receivedToolCalls[0].Function.Name)
}

// TestGetToolResultContent tests that error messages are properly sent to LLM on tool failure
func TestGetToolResultContent(t *testing.T) {
	tests := []struct {
		name     string
		toolCall *orchestration.ToolCall
		result   *orchestration.ToolResult
		want     string
	}{
		{
			name: "successful tool call returns output",
			toolCall: &orchestration.ToolCall{
				ID: "call_1",
				Function: orchestration.ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path":"."}`,
				},
			},
			result: &orchestration.ToolResult{
				ID:      "call_1",
				Success: true,
				Output:  "file1.go\nfile2.go\nREADME.md",
			},
			want: "file1.go\nfile2.go\nREADME.md",
		},
		{
			name: "failed tool call with error returns error message",
			toolCall: &orchestration.ToolCall{
				ID: "call_2",
				Function: orchestration.ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"nonexistent.txt"}`,
				},
			},
			result: &orchestration.ToolResult{
				ID:      "call_2",
				Success: false,
				Error:   errors.New("file not found: nonexistent.txt"),
			},
			want: "Tool read_file failed: file not found: nonexistent.txt",
		},
		{
			name: "failed tool call without error message",
			toolCall: &orchestration.ToolCall{
				ID: "call_3",
				Function: orchestration.ToolCallFunction{
					Name:      "execute_command",
					Arguments: `{"cmd":"unknown"}`,
				},
			},
			result: &orchestration.ToolResult{
				ID:      "call_3",
				Success: false,
			},
			want: "Tool execute_command failed with no error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getToolResultContent(tt.toolCall, tt.result)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestToolExecutionWithRealOllama tests tool execution with actual Ollama/qwen3:1.7b
// This reproduces the exact issue the user is experiencing.
//
// Run with: go test -v -tags=integration -run TestToolExecutionWithRealOllama ./internal/agent/
func TestToolExecutionWithRealOllama(t *testing.T) {
	// Skip if Ollama isn't available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create real Ollama provider with qwen3:1.7b
	provider, err := factory.NewProvider(factory.ProviderConfig{
		Type:    "ollama",
		Model:   "qwen3:1.7b",
		BaseURL: "http://localhost:11434",
	})
	require.NoError(t, err, "Ollama must be running with qwen3:1.7b model")
	defer provider.Close()

	// Setup tool registry with list_directory
	toolRegistry := tools.NewRegistry()
	err = toolRegistry.Register(tools.NewListDirectoryTool())
	require.NoError(t, err)

	// Setup services
	validator := security.NewValidator()
	securityService := security.NewSecurityService(validator, nil)

	cycleDetector := cycle.NewDetector(cycle.Config{
		WindowSize:       10,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
	})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry: toolRegistry,
	})
	orchestrationService := orchestration.NewOrchestrationService(
		toolExecutor,
		toolRegistry,
		nil,
	)

	env := &Environment{
		WorkDir:     "/tmp",
		Environment: make(map[string]string),
	}

	emitter := events.NewEventEmitter(100)

	// Collect events
	subID, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)
	defer emitter.Unsubscribe(subID)

	var toolStartEvents []events.Event
	var toolCompleteEvents []events.Event
	var contentDeltas []string
	done := make(chan struct{})

	go func() {
		defer close(done)
		for evt := range eventCh {
			switch evt.Type {
			case events.EventToolCallStart:
				toolStartEvents = append(toolStartEvents, evt)
				t.Logf("✓ Tool start: %+v", evt.Data)
			case events.EventToolCallComplete:
				toolCompleteEvents = append(toolCompleteEvents, evt)
				data, _ := evt.Data.(events.ToolCallCompleteData)
				t.Logf("✓ Tool complete: success=%v tool=%s", data.Success, data.ToolName)
			case events.EventContentDelta:
				data, _ := evt.Data.(events.ContentDeltaData)
				contentDeltas = append(contentDeltas, data.Content)
				t.Logf("Content delta: %s", data.Content)
			case events.EventWarning:
				data, _ := evt.Data.(events.SystemEventData)
				t.Logf("⚠ WARNING: %s", data.Message)
			}
		}
	}()

	// Create agent
	agent, err := NewAgent(
		provider,
		securityService,
		detectionService,
		orchestrationService,
		env,
		emitter,
	)
	require.NoError(t, err)

	// Create task
	task := &simpleTask{
		name:         "test",
		systemPrompt: "You are a helpful assistant. When asked to list files, use the list_directory tool.",
		allowedTools: []string{}, // Allow all tools
		maxTokens:    4096,
	}

	// Execute: This should trigger list_directory tool call
	req := &AgentRequest{
		Input: "list files in /tmp directory",
		Task:  task,
	}

	t.Log("Sending request to agent...")
	resp, err := agent.Execute(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Close emitter and wait for events
	emitter.Close()
	<-done

	// Print all content for debugging
	t.Logf("Total content deltas: %d", len(contentDeltas))
	t.Logf("Response: %s", resp.Output)

	// ASSERTIONS - This is what should happen:

	// 1. Tool should have been called
	assert.NotEmpty(t, toolStartEvents, "BUG: Tool was not called even though model should call list_directory")

	// 2. Tool should have completed
	assert.NotEmpty(t, toolCompleteEvents, "BUG: Tool was called but didn't complete")

	// 3. Tool should have succeeded
	if len(toolCompleteEvents) > 0 {
		completeData, ok := toolCompleteEvents[0].Data.(events.ToolCallCompleteData)
		assert.True(t, ok)
		assert.True(t, completeData.Success, "Tool execution should succeed")
		assert.Equal(t, "list_directory", completeData.ToolName)
		assert.NotEmpty(t, completeData.Output, "Tool should produce output")
	}

	// 4. Response should contain file listing (not just "thinking" about it)
	assert.NotEmpty(t, resp.Output, "Agent should produce final output")

	// If we see content but no tool execution, that's the bug
	if len(contentDeltas) > 0 && len(toolStartEvents) == 0 {
		t.Errorf("BUG REPRODUCED: Model generated content (%d deltas) but no tool calls", len(contentDeltas))
		t.Logf("Content: %v", contentDeltas)
	}
}

// TestDirectToolCallWithOllama tests ProcessToolCall directly with Ollama running
func TestDirectToolCallWithOllama(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// Setup
	toolRegistry := tools.NewRegistry()
	err := toolRegistry.Register(tools.NewListDirectoryTool())
	require.NoError(t, err)

	validator := security.NewValidator()
	securityService := security.NewSecurityService(validator, nil)

	cycleDetector := cycle.NewDetector(cycle.Config{WindowSize: 10})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry: toolRegistry,
	})
	orchestrationService := orchestration.NewOrchestrationService(
		toolExecutor,
		toolRegistry,
		nil,
	)

	env := &Environment{
		WorkDir:     "/tmp",
		Environment: make(map[string]string),
	}

	emitter := events.NewEventEmitter(100)

	// Use real Ollama
	provider, err := factory.NewProvider(factory.ProviderConfig{
		Type:    "ollama",
		Model:   "qwen3:1.7b",
		BaseURL: "http://localhost:11434",
	})
	require.NoError(t, err)
	defer provider.Close()

	agent, err := NewAgent(
		provider,
		securityService,
		detectionService,
		orchestrationService,
		env,
		emitter,
	)
	require.NoError(t, err)

	// Test ProcessToolCall directly
	toolCall := &orchestration.ToolCall{
		ID:   "test_direct",
		Type: "function",
		Function: orchestration.ToolCallFunction{
			Name:      "list_directory",
			Arguments: `{"path": "/tmp"}`,
		},
	}

	result, err := agent.ProcessToolCall(ctx, toolCall)
	require.NoError(t, err, "Direct tool call should work")
	require.NotNil(t, result)
	assert.True(t, result.Success, "Tool should execute successfully")
	assert.NotEmpty(t, result.Output, "Tool should return output")

	t.Logf("Direct tool call output: %s", result.Output)
}

// simpleTask implements Task interface for testing
type simpleTask struct {
	name         string
	systemPrompt string
	allowedTools []string
	maxTokens    int
}

func (s *simpleTask) Name() string           { return s.name }
func (s *simpleTask) SystemPrompt() string   { return s.systemPrompt }
func (s *simpleTask) AllowedTools() []string { return s.allowedTools }
func (s *simpleTask) MaxTokens() int         { return s.maxTokens }
func (s *simpleTask) Validate() error        { return nil }

// dummyLLM is a minimal LLM for tests that don't use it
type dummyLLM struct{}

func (d *dummyLLM) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{Content: "dummy", FinishReason: "stop"}, nil
}

func (d *dummyLLM) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- llm.StreamChunk{Content: "dummy", FinishReason: "stop"}
	}()
	return ch, nil
}

func (d *dummyLLM) Models(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{}, nil
}

func (d *dummyLLM) Capabilities() llm.Capabilities {
	return llm.Capabilities{Streaming: true, FunctionCalling: true}
}

func (d *dummyLLM) Name() string {
	return "dummy"
}

func (d *dummyLLM) Close() error {
	return nil
}

// ============================================================================
// Benchmark Tests
// ============================================================================

// BenchmarkAgent_ResolveTaskExplicit benchmarks resolving an explicit task object.
// Expected: ~50-100 ns/op (pointer comparison, should be instant)
func BenchmarkAgent_ResolveTaskExplicit(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewRegular()
	req := &AgentRequest{Task: taskObj}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolvedTask, err := agent.resolveTask(req)
		if err != nil {
			b.Fatal(err)
		}
		if resolvedTask == nil {
			b.Fatal("expected task")
		}
	}
}

// BenchmarkAgent_ResolveTaskByName benchmarks resolving task by name (registry lookup).
// Expected: ~100-150 ns/op (map lookup + RLock)
func BenchmarkAgent_ResolveTaskByName(b *testing.B) {
	agent := newBenchAgent(b)
	req := &AgentRequest{TaskName: "review"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolvedTask, err := agent.resolveTask(req)
		if err != nil {
			b.Fatal(err)
		}
		if resolvedTask == nil {
			b.Fatal("expected task")
		}
	}
}

// BenchmarkAgent_ResolveTaskDefault benchmarks resolving default task.
// Expected: ~100-150 ns/op (map lookup + RLock)
func BenchmarkAgent_ResolveTaskDefault(b *testing.B) {
	agent := newBenchAgent(b)
	req := &AgentRequest{} // No task specified, should use default

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolvedTask, err := agent.resolveTask(req)
		if err != nil {
			b.Fatal(err)
		}
		if resolvedTask == nil {
			b.Fatal("expected task")
		}
	}
}

// BenchmarkAgent_BuildToolsForTask_Regular benchmarks tool filtering for regular mode.
// Expected: ~500-1000 ns/op (allows all tools, minimal filtering)
func BenchmarkAgent_BuildToolsForTask_Regular(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewRegular()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tools, err := agent.BuildToolsForTask(taskObj)
		if err != nil {
			b.Fatal(err)
		}
		if len(tools) == 0 {
			b.Fatal("expected tools")
		}
	}
}

// BenchmarkAgent_BuildToolsForTask_Compact benchmarks tool filtering for compact mode.
// Expected: ~200-400 ns/op (allows only 4 tools, fast filtering)
func BenchmarkAgent_BuildToolsForTask_Compact(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewCompact()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tools, err := agent.BuildToolsForTask(taskObj)
		if err != nil {
			b.Fatal(err)
		}
		if len(tools) == 0 {
			b.Fatal("expected tools")
		}
	}
}

// BenchmarkAgent_BuildToolsForTask_Review benchmarks tool filtering for review mode.
// Expected: ~200-400 ns/op (allows only 5 tools, fast filtering)
func BenchmarkAgent_BuildToolsForTask_Review(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewReview()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tools, err := agent.BuildToolsForTask(taskObj)
		if err != nil {
			b.Fatal(err)
		}
		if len(tools) == 0 {
			b.Fatal("expected tools")
		}
	}
}

// BenchmarkAgent_BuildToolsForTask_Planning benchmarks tool filtering for planning mode.
// Expected: ~200-400 ns/op (allows only 4 tools, fast filtering)
func BenchmarkAgent_BuildToolsForTask_Planning(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewPlanning()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tools, err := agent.BuildToolsForTask(taskObj)
		if err != nil {
			b.Fatal(err)
		}
		if len(tools) == 0 {
			b.Fatal("expected tools")
		}
	}
}

// BenchmarkAgent_ProcessToolCall benchmarks tool call processing.
// Expected: ~1000-2000 ns/op (validation + parsing + orchestration)
func BenchmarkAgent_ProcessToolCall(b *testing.B) {
	agent := newBenchAgent(b)
	toolCall := &orchestration.ToolCall{
		ID:   "test_call",
		Type: "function",
		Function: orchestration.ToolCallFunction{
			Name:      "list_directory",
			Arguments: `{"path": "/tmp"}`,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := agent.ProcessToolCall(context.Background(), toolCall)
		if err != nil {
			b.Fatal(err)
		}
		if result == nil {
			b.Fatal("expected result")
		}
	}
}

// BenchmarkAgent_ShouldApprove benchmarks approval decision making.
// Expected: ~100-200 ns/op (simple classification lookup)
func BenchmarkAgent_ShouldApprove(b *testing.B) {
	agent := newBenchAgent(b)
	cmd := &security.Command{
		Program: "rm",
		Args:    []string{"-rf", "/tmp/test"},
		WorkDir: "/tmp",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		needsApproval, reason := agent.ShouldApprove(cmd)
		if reason == "" && needsApproval {
			b.Fatal("unexpected approval requirement")
		}
	}
}

// BenchmarkAgent_ExtractToolNames benchmarks tool name extraction for cycle detection.
// Expected: ~50-100 ns/op (simple string concatenation)
func BenchmarkAgent_ExtractToolNames(b *testing.B) {
	toolCalls := []llm.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: llm.FunctionCall{
				Name:      "list_directory",
				Arguments: `{"path": "/tmp"}`,
			},
		},
		{
			ID:   "call_2",
			Type: "function",
			Function: llm.FunctionCall{
				Name:      "read_file",
				Arguments: `{"path": "test.txt"}`,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		names := extractToolNames(toolCalls)
		if len(names) != 2 {
			b.Fatal("expected 2 tool names")
		}
	}
}

// newBenchAgent creates an agent optimized for benchmarking
func newBenchAgent(b *testing.B) *Agent {
	// Create minimal services for benchmarking
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	// Create tool registry with all built-in tools
	toolRegistry := tools.NewRegistry()
	executor, _ := NewExecutor("/tmp")
	_ = toolRegistry.Register(tools.NewReadFileTool())
	_ = toolRegistry.Register(tools.NewWriteFileTool())
	_ = toolRegistry.Register(tools.NewListDirectoryTool())
	_ = toolRegistry.Register(tools.NewExecuteCommandTool(executor, validator))
	_ = toolRegistry.Register(tools.NewGetContextTool(&Environment{WorkDir: "/tmp"}))
	_ = toolRegistry.Register(tools.NewApplyPatchTool("/tmp"))
	_ = toolRegistry.Register(tools.NewFileSearchTool("/tmp"))
	_ = toolRegistry.Register(tools.NewGitContextTool("/tmp"))

	// Create task registry with all modes
	taskRegistry := orchestration.NewRegistry()
	_ = taskRegistry.Register("regular", task.NewRegular())
	_ = taskRegistry.Register("review", task.NewReview())
	_ = taskRegistry.Register("compact", task.NewCompact())
	_ = taskRegistry.Register("planning", task.NewPlanning())
	_ = taskRegistry.SetDefault("regular")

	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	agent, err := NewAgent(
		&mockLLMProvider{},
		securityService,
		detectionService,
		orchestrationService,
		&Environment{WorkDir: "/tmp"},
		emitter,
	)
	if err != nil {
		b.Fatal(err)
	}

	return agent
}

// mockLLMProvider is a minimal LLM provider for benchmarking
type mockLLMProvider struct{}

func (m *mockLLMProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{Content: "mock", FinishReason: "stop"}, nil
}

func (m *mockLLMProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- llm.StreamChunk{Content: "mock", FinishReason: "stop"}
	}()
	return ch, nil
}

func (m *mockLLMProvider) Models(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{}, nil
}

func (m *mockLLMProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{Streaming: true, FunctionCalling: true}
}

func (m *mockLLMProvider) Name() string {
	return "mock"
}

func (m *mockLLMProvider) Close() error {
	return nil
}

// ============================================================================
// Cycle Intervention Tests
// ============================================================================

// TestHandleCycleDetection_InterventionMessagesApplied tests that
// intervention messages are actually added to the conversation.
// This is a regression test for the bug where handleCycleDetection
// modified messages locally but didn't return them, causing interventions
// to be silently discarded.
func TestHandleCycleDetection_InterventionMessagesApplied(t *testing.T) {
	agent := createTestAgentWithServices(t)

	// Enable cycle detection
	agent.config.CycleDetection.Enabled = true

	// Create initial conversation with some messages
	initialMessages := []Message{
		{
			Role:      RoleUser,
			Content:   "List files",
			Timestamp: time.Now(),
		},
		{
			Role:      RoleAssistant,
			Content:   "I'll list the files",
			Timestamp: time.Now(),
		},
	}

	// Simulate repeated tool calls to trigger cycle detection
	// Add 3 snapshots with same tool AND same params to trigger CycleRepeatedTool
	agent.detection.RecordSnapshot(cycle.Snapshot{
		Turn:      1,
		Response:  "Calling list_directory",
		ToolCalls: []string{`list_directory({"path": "/"})`},
		Timestamp: time.Now(),
	})
	agent.detection.RecordSnapshot(cycle.Snapshot{
		Turn:      2,
		Response:  "Calling list_directory again",
		ToolCalls: []string{`list_directory({"path": "/"})`},
		Timestamp: time.Now(),
	})
	agent.detection.RecordSnapshot(cycle.Snapshot{
		Turn:      3,
		Response:  "Calling list_directory once more",
		ToolCalls: []string{`list_directory({"path": "/"})`},
		Timestamp: time.Now(),
	})

	// Create a mock LLM response that will trigger cycle detection
	llmResp := &llm.CompletionResponse{
		Content: "Calling list_directory",
		ToolCalls: []llm.ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: llm.FunctionCall{
					Name:      "list_directory",
					Arguments: `{"path": "/"}`,
				},
			},
		},
		FinishReason: "",
	}

	// Call handleCycleDetection
	resp := &AgentResponse{}
	modifiedMessages, shouldStop, err := agent.handleCycleDetection(
		context.Background(),
		initialMessages,
		llmResp,
		3, // turn count
		resp,
	)

	if err != nil {
		t.Fatalf("handleCycleDetection returned error: %v", err)
	}

	if shouldStop {
		t.Fatal("handleCycleDetection should not stop (severity < 3)")
	}

	// Check if cycle was detected
	cycleResult, err := agent.detection.CheckCycle()
	if err != nil {
		t.Fatalf("CheckCycle failed: %v", err)
	}
	if cycleResult.Type == cycle.CycleNone {
		t.Fatal("Expected cycle to be detected, but got CycleNone")
	}

	// The critical assertion: modifiedMessages should have the intervention message added
	// With the bug (before fix), modifiedMessages would equal initialMessages (unchanged)
	// After the fix, modifiedMessages should be longer (reflection added)
	if len(modifiedMessages) == len(initialMessages) {
		t.Error("BUG DETECTED: handleCycleDetection did not modify the messages slice")
		t.Error("Expected intervention message to be added, but messages unchanged")
		t.Error("This indicates the intervention's message modifications were discarded")
	}

	// After the fix, this should pass
	expectedMinLen := 3 // original 2 + 1 reflection message
	if len(modifiedMessages) < expectedMinLen {
		t.Errorf("Expected at least %d messages after intervention, got %d", expectedMinLen, len(modifiedMessages))
	}

	// Verify the last message is from the intervention (user role with reflection prompt)
	if len(modifiedMessages) >= expectedMinLen {
		lastMsg := modifiedMessages[len(modifiedMessages)-1]
		if lastMsg.Role != RoleUser {
			t.Errorf("Expected intervention message to have role 'user', got '%s'", lastMsg.Role)
		}
		if lastMsg.Content == "" {
			t.Error("Expected intervention message to have content")
		}
		// Reflection intervention should mention "repeating" or "different"
		if !containsAnySubstring(lastMsg.Content, []string{"repeating", "different", "perspective", "angles"}) {
			t.Errorf("Expected reflection-style message, got: %s", lastMsg.Content)
		}
	}
}

// TestExecuteAgentLoop_CycleInterventionPropagated tests that the full agent loop
// properly uses intervention messages.
func TestExecuteAgentLoop_CycleInterventionPropagated(t *testing.T) {
	agent := createTestAgentWithServices(t)
	agent.config.CycleDetection.Enabled = true
	agent.config.MaxTurns = 10

	// Create a mock LLM that returns same tool call repeatedly
	mockLLM := llm.NewMockProvider("test")
	agent.llm = mockLLM

	initialMessages := []Message{
		{
			Role:      RoleSystem,
			Content:   "You are a helpful assistant",
			Timestamp: time.Now(),
		},
		{
			Role:      RoleUser,
			Content:   "List files",
			Timestamp: time.Now(),
		},
	}

	task := task.NewRegular()
	resp := &AgentResponse{}

	// Execute the loop - it should detect the cycle and add intervention
	resultMessages, resultResp, err := agent.executeAgentLoop(
		context.Background(),
		initialMessages,
		task,
		resp,
	)

	// The loop should complete (may hit max turns or other stop condition)
	if err != nil {
		// Error is acceptable for mock LLM
		t.Logf("executeAgentLoop returned error (expected with mock): %v", err)
	}

	_ = resultResp // resultResp is used implicitly

	// Key test: if a cycle was detected during the loop, verify intervention messages
	// were preserved in resultMessages
	history := agent.detection.GetHistory()
	if len(history) >= 3 {
		// Cycle should have been detected
		t.Log("Cycle detection triggered during agent loop")

		// After the fix, resultMessages should include intervention messages
		if len(resultMessages) <= len(initialMessages) {
			t.Error("Expected resultMessages to include intervention messages, but no new messages found")
		}
	}
}

// Helper function to check if string contains any of the substrings
func containsAnySubstring(s string, substrings []string) bool {
	for _, substr := range substrings {
		if containsSubstring(s, substr) {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				indexOfSubstring(s, substr) >= 0))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// ============================================================================
// Token Budget Tests
// ============================================================================

// TestAgent_TaskBudgetOverridesConfig verifies that a task's MaxTokens
// overrides the agent's config.MaxTokens when task.MaxTokens() > 0.
func TestAgent_TaskBudgetOverridesConfig(t *testing.T) {
	// Create agent with 4K config
	llmCapture := newCapturingLLMProvider()
	validator := security.NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := events.NewEventEmitter(100)

	agent, err := newAgentForTest(llmCapture, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Override config to 4K tokens
	agent.config.MaxTokens = 4096

	// Regular mode has 16K tokens
	regularTask := task.NewRegular()
	if regularTask.MaxTokens() != 16384 {
		t.Fatalf("expected regular task to have 16384 tokens, got %d", regularTask.MaxTokens())
	}

	// Create request with regular task
	req := &AgentRequest{
		Input: "test input",
		Task:  regularTask,
	}

	// Execute (will fail because no tools, but that's ok - we just want to capture the request)
	_, _ = agent.Execute(context.Background(), req)

	// Verify task budget was used (16K, not 4K from config)
	if len(llmCapture.requests) == 0 {
		t.Fatal("expected LLM to be called")
	}

	lastRequest := llmCapture.requests[len(llmCapture.requests)-1]
	if lastRequest.MaxTokens != 16384 {
		t.Errorf("expected MaxTokens to be 16384 (from task), got %d", lastRequest.MaxTokens)
	}
}

// TestAgent_ConfigBudgetUsedWhenTaskZero verifies that agent's config.MaxTokens
// is used when task.MaxTokens() returns 0.
func TestAgent_ConfigBudgetUsedWhenTaskZero(t *testing.T) {
	// Create agent with 8K config
	llmCapture := newCapturingLLMProvider()
	validator := security.NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := events.NewEventEmitter(100)

	agent, err := newAgentForTest(llmCapture, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Set config to 8K tokens
	agent.config.MaxTokens = 8192

	// Create a custom task that returns 0 for MaxTokens
	zeroBudgetTask := &zeroBudgetTask{}

	// Create request with zero budget task
	req := &AgentRequest{
		Input: "test input",
		Task:  zeroBudgetTask,
	}

	// Execute (will fail because no tools, but that's ok - we just want to capture the request)
	_, _ = agent.Execute(context.Background(), req)

	// Verify config budget was used (8K, not 0 from task)
	if len(llmCapture.requests) == 0 {
		t.Fatal("expected LLM to be called")
	}

	lastRequest := llmCapture.requests[len(llmCapture.requests)-1]
	if lastRequest.MaxTokens != 8192 {
		t.Errorf("expected MaxTokens to be 8192 (from config), got %d", lastRequest.MaxTokens)
	}
}

// TestAgent_ConcurrentTokenBudget verifies that token budget handling
// works correctly under concurrent access.
func TestAgent_ConcurrentTokenBudget(t *testing.T) {
	// Create agent
	llmCapture := newCapturingLLMProvider()
	validator := security.NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := events.NewEventEmitter(100)

	agent, err := newAgentForTest(llmCapture, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Set config to 4K tokens
	agent.config.MaxTokens = 4096

	// Create different tasks with different budgets
	tasks := []Task{
		task.NewRegular(),  // 16K
		task.NewCompact(),  // 8K
		task.NewReview(),   // 12K
		task.NewPlanning(), // 8K
	}

	// Run concurrent requests
	var wg sync.WaitGroup
	numRequests := 10

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(taskIndex int) {
			defer wg.Done()
			task := tasks[taskIndex%len(tasks)]
			req := &AgentRequest{
				Input: "test input",
				Task:  task,
			}
			_, _ = agent.Execute(context.Background(), req)
		}(i)
	}

	wg.Wait()

	// Verify all requests used correct token budgets
	if len(llmCapture.requests) == 0 {
		t.Fatal("expected LLM to be called")
	}

	for i, req := range llmCapture.requests {
		taskIndex := i % len(tasks)
		expectedTokens := tasks[taskIndex].MaxTokens()
		if req.MaxTokens != expectedTokens {
			t.Errorf("request %d: expected MaxTokens %d, got %d", i, expectedTokens, req.MaxTokens)
		}
	}
}

// zeroBudgetTask is a test task that returns 0 for MaxTokens
type zeroBudgetTask struct{}

func (z *zeroBudgetTask) Name() string           { return "zero-budget" }
func (z *zeroBudgetTask) SystemPrompt() string   { return "Zero budget task" }
func (z *zeroBudgetTask) AllowedTools() []string { return []string{} }
func (z *zeroBudgetTask) MaxTokens() int         { return 0 }
func (z *zeroBudgetTask) Validate() error        { return nil }

// capturingLLMProvider captures LLM requests for testing
type capturingLLMProvider struct {
	requests []llm.CompletionRequest
	mu       sync.Mutex
}

func newCapturingLLMProvider() *capturingLLMProvider {
	return &capturingLLMProvider{
		requests: make([]llm.CompletionRequest, 0),
	}
}

func (c *capturingLLMProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	c.mu.Lock()
	c.requests = append(c.requests, req)
	c.mu.Unlock()
	return &llm.CompletionResponse{Content: "mock", FinishReason: "stop"}, nil
}

func (c *capturingLLMProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	c.mu.Lock()
	c.requests = append(c.requests, req)
	c.mu.Unlock()
	ch := make(chan llm.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- llm.StreamChunk{Content: "mock", FinishReason: "stop"}
	}()
	return ch, nil
}

func (c *capturingLLMProvider) Models(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{}, nil
}

func (c *capturingLLMProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{Streaming: true, FunctionCalling: true}
}

func (c *capturingLLMProvider) Name() string {
	return "capturing"
}

func (c *capturingLLMProvider) Close() error {
	return nil
}

// ============================================================================
// Coverage Tests
// ============================================================================

// newTestAgentMinimal creates a minimal agent for testing with services
func newTestAgentMinimal(toolRegistry *tools.Registry, taskRegistry *orchestration.Registry) *Agent {
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	if toolRegistry == nil {
		toolRegistry = tools.NewRegistry()
	}
	if taskRegistry == nil {
		taskRegistry = orchestration.NewRegistry()
	}

	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	agent, _ := NewAgent(
		&mockLLMProvider{},
		securityService,
		detectionService,
		orchestrationService,
		&Environment{WorkDir: "/tmp"},
		emitter,
	)
	return agent
}

func TestAgent_processToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		agent     *Agent
		messages  []Message
		llmResp   *llm.CompletionResponse
		resp      *AgentResponse
		wantCount int
	}{
		{
			name:     "successful tool processing",
			agent:    newTestAgentMinimal(nil, nil),
			messages: []Message{},
			llmResp: &llm.CompletionResponse{
				Content: "Let me use a tool to help you.",
			},
			resp:      &AgentResponse{},
			wantCount: 1, // One message added
		},
		{
			name:     "empty response",
			agent:    newTestAgentMinimal(nil, nil),
			messages: []Message{},
			llmResp: &llm.CompletionResponse{
				Content: "",
			},
			resp:      &AgentResponse{},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.processToolCalls(context.Background(), tt.messages, tt.llmResp, tt.resp)

			if len(result) != tt.wantCount {
				t.Errorf("Agent.processToolCalls() result length = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestAgent_processToolCalls_WithToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		agent     *Agent
		messages  []Message
		llmResp   *llm.CompletionResponse
		resp      *AgentResponse
		wantCount int
	}{
		{
			name:     "single tool call",
			agent:    newTestAgentMinimal(nil, nil),
			messages: []Message{},
			llmResp: &llm.CompletionResponse{
				Content: "I'll help you with that.",
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "list_directory",
							Arguments: `{"path": "/tmp"}`,
						},
					},
				},
			},
			resp:      &AgentResponse{},
			wantCount: 2, // Assistant message + tool result message
		},
		{
			name:     "multiple tool calls",
			agent:    newTestAgentMinimal(nil, nil),
			messages: []Message{},
			llmResp: &llm.CompletionResponse{
				Content: "I'll help you with multiple tasks.",
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "list_directory",
							Arguments: `{"path": "/tmp"}`,
						},
					},
					{
						ID:   "call_2",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "read_file",
							Arguments: `{"path": "test.txt"}`,
						},
					},
				},
			},
			resp:      &AgentResponse{},
			wantCount: 3, // Assistant message + 2 tool result messages
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.processToolCalls(context.Background(), tt.messages, tt.llmResp, tt.resp)

			if len(result) != tt.wantCount {
				t.Errorf("Agent.processToolCalls() result length = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestAgent_validateToolCall(t *testing.T) {
	agent := newTestAgentMinimal(nil, nil)

	tests := []struct {
		name    string
		call    *orchestration.ToolCall
		wantErr bool
	}{
		{
			name: "valid tool call",
			call: &orchestration.ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: orchestration.ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil tool call",
			call:    nil,
			wantErr: true,
		},
		{
			name: "empty ID",
			call: &orchestration.ToolCall{
				ID:   "",
				Type: "function",
				Function: orchestration.ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
			wantErr: true,
		},
		{
			name: "empty function name",
			call: &orchestration.ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: orchestration.ToolCallFunction{
					Name:      "",
					Arguments: `{"path": "/tmp"}`,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := agent.validateToolCall(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("Agent.validateToolCall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgent_parseToolArguments(t *testing.T) {
	agent := newTestAgentMinimal(nil, nil)

	tests := []struct {
		name    string
		call    *orchestration.ToolCall
		wantErr bool
	}{
		{
			name: "valid JSON arguments",
			call: &orchestration.ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: orchestration.ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "empty arguments",
			call: &orchestration.ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: orchestration.ToolCallFunction{
					Name:      "list_directory",
					Arguments: "",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid JSON arguments",
			call: &orchestration.ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: orchestration.ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"`, // Missing closing brace
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := agent.parseToolArguments(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("Agent.parseToolArguments() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && args == nil {
				t.Error("Agent.parseToolArguments() returned nil args for valid input")
			}
		})
	}
}

func TestAgent_addFinalMessage(t *testing.T) {
	agent := newTestAgentMinimal(nil, nil)

	tests := []struct {
		name     string
		messages []Message
		content  string
		wantLen  int
	}{
		{
			name:     "add message with content",
			messages: []Message{},
			content:  "Hello, world!",
			wantLen:  1,
		},
		{
			name:     "add message with empty content",
			messages: []Message{},
			content:  "",
			wantLen:  0, // Should not add empty content
		},
		{
			name: "add message to existing messages",
			messages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			content: "Hi there!",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.addFinalMessage(tt.messages, tt.content)
			if len(result) != tt.wantLen {
				t.Errorf("Agent.addFinalMessage() result length = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestAgent_emitTurnStart(t *testing.T) {
	agent := newTestAgentMinimal(nil, nil)

	// This test mainly ensures the method doesn't panic
	// In a real test, you'd want to verify the event was emitted
	agent.emitTurnStart(1)
	agent.emitTurnStart(5)
	agent.emitTurnStart(100)
}

func TestAgent_applyTimeout(t *testing.T) {
	agent := newTestAgentMinimal(nil, nil)

	ctx := context.Background()

	// Test with default timeout
	agent.config.Timeout = 0
	ctxWithTimeout, cancel := agent.applyTimeout(ctx)
	if ctxWithTimeout == ctx {
		t.Error("Expected context to be modified with timeout")
	}
	cancel()

	// Test with custom timeout
	agent.config.Timeout = 5 * time.Second
	ctxWithTimeout, cancel = agent.applyTimeout(ctx)
	if ctxWithTimeout == ctx {
		t.Error("Expected context to be modified with timeout")
	}
	cancel()
}

func TestAgent_executeSetup(t *testing.T) {
	agent := newTestAgentMinimal(nil, nil)

	tests := []struct {
		name    string
		req     *AgentRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &AgentRequest{
				Input: "Hello, world!",
			},
			wantErr: false,
		},
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty input",
			req: &AgentRequest{
				Input: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, resp, err := agent.executeSetup(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Agent.executeSetup() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && resp == nil {
				t.Error("Agent.executeSetup() returned nil response for valid input")
			}
			if ctx == nil {
				t.Error("Agent.executeSetup() returned nil context")
			}
		})
	}
}

func TestAgent_finalizeResponse(t *testing.T) {
	agent := newTestAgentMinimal(nil, nil)

	tests := []struct {
		name       string
		resp       *AgentResponse
		messages   []Message
		historyLen int
		wantOutput string
	}{
		{
			name: "response with assistant message",
			resp: &AgentResponse{},
			messages: []Message{
				{Role: RoleUser, Content: "Hello"},
				{Role: RoleAssistant, Content: "Hi there!"},
			},
			historyLen: 1,
			wantOutput: "Hi there!",
		},
		{
			name: "response without assistant message",
			resp: &AgentResponse{},
			messages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			historyLen: 1,
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent.finalizeResponse(tt.resp, tt.messages, tt.historyLen)
			if tt.resp.Output != tt.wantOutput {
				t.Errorf("Agent.finalizeResponse() output = %q, want %q", tt.resp.Output, tt.wantOutput)
			}
		})
	}
}
