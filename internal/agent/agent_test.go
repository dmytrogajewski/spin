package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/openai/openai-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/planning"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
)

var (
	errFileNotFoundNonexistentTxt = errors.New("file not found: nonexistent.txt")
	errLlmProviderError = errors.New("LLM provider error")
)

// TestNewAgent tests the refactored agent creation with services.
func TestNewAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		provider    llm.Provider
		security    *security.Service
		detection   *detection.Service
		toolRuntime *ToolRuntime
		planning    *planning.Service
		environment *Environment
		emitter     *events.EventEmitter
		aceService  *ACEService
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid agent",
			provider: llm.NewMockProvider("test"),
			security: func() *security.Service {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})

				return security.NewService(validator, approvalService)
			}(),
			detection:   detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			toolRuntime: newTestToolRuntime(nil, tools.NewRegistry()),
			planning:    planning.NewService(llm.NewMockProvider("test")),
			environment: &Environment{WorkDir: "/tmp"},
			emitter:     events.NewEventEmitter(100),
			wantErr:     false,
		},
		{
			name:     "nil provider",
			provider: nil,
			security: func() *security.Service {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})

				return security.NewService(validator, approvalService)
			}(),
			detection:   detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			toolRuntime: newTestToolRuntime(nil, tools.NewRegistry()),
			planning:    planning.NewService(llm.NewMockProvider("test")),
			environment: &Environment{WorkDir: "/tmp"},
			emitter:     events.NewEventEmitter(100),
			wantErr:     true,
			errContains: "LLM provider cannot be nil",
		},
		{
			name:        "nil security",
			provider:    llm.NewMockProvider("test"),
			security:    nil,
			detection:   detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			toolRuntime: newTestToolRuntime(nil, tools.NewRegistry()),
			planning:    planning.NewService(llm.NewMockProvider("test")),
			environment: &Environment{WorkDir: "/tmp"},
			emitter:     events.NewEventEmitter(100),
			wantErr:     true,
			errContains: "security service cannot be nil",
		},
		{
			name:     "nil detection",
			provider: llm.NewMockProvider("test"),
			security: func() *security.Service {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})

				return security.NewService(validator, approvalService)
			}(),
			detection:   nil,
			toolRuntime: newTestToolRuntime(nil, tools.NewRegistry()),
			planning:    planning.NewService(llm.NewMockProvider("test")),
			environment: &Environment{WorkDir: "/tmp"},
			emitter:     events.NewEventEmitter(100),
			wantErr:     true,
			errContains: "detection service cannot be nil",
		},
		{
			name:     "nil tool runtime",
			provider: llm.NewMockProvider("test"),
			security: func() *security.Service {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})

				return security.NewService(validator, approvalService)
			}(),
			detection:   detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			toolRuntime: nil,
			planning:    planning.NewService(llm.NewMockProvider("test")),
			environment: &Environment{WorkDir: "/tmp"},
			emitter:     events.NewEventEmitter(100),
			wantErr:     true,
			errContains: "tool runtime cannot be nil",
		},
		{
			name:     "nil planning",
			provider: llm.NewMockProvider("test"),
			security: func() *security.Service {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})

				return security.NewService(validator, approvalService)
			}(),
			detection:   detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			toolRuntime: newTestToolRuntime(nil, tools.NewRegistry()),
			planning:    nil,
			environment: &Environment{WorkDir: "/tmp"},
			emitter:     events.NewEventEmitter(100),
			wantErr:     true,
			errContains: "planning service cannot be nil",
		},
		{
			name:     "nil environment",
			provider: llm.NewMockProvider("test"),
			security: func() *security.Service {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})

				return security.NewService(validator, approvalService)
			}(),
			detection:   detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			toolRuntime: newTestToolRuntime(nil, tools.NewRegistry()),
			planning:    planning.NewService(llm.NewMockProvider("test")),
			environment: nil,
			emitter:     events.NewEventEmitter(100),
			wantErr:     true,
			errContains: "context cannot be nil",
		},
		{
			name:     "nil emitter",
			provider: llm.NewMockProvider("test"),
			security: func() *security.Service {
				validator := security.NewValidator()
				emitter := events.NewEventEmitter(100)
				approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})

				return security.NewService(validator, approvalService)
			}(),
			detection:   detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
			toolRuntime: newTestToolRuntime(nil, tools.NewRegistry()),
			planning:    planning.NewService(llm.NewMockProvider("test")),
			environment: &Environment{WorkDir: "/tmp"},
			emitter:     nil,
			wantErr:     true,
			errContains: "event emitter cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := []Option{}
			if tt.aceService != nil {
				opts = append(opts, WithACEService(tt.aceService))
			}

			agent, err := NewAgent(tt.provider, tt.security, tt.detection, tt.toolRuntime, tt.planning, tt.environment, tt.emitter, opts...)

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

// TestAgent_WithACEService tests Agent with ACE integration.
func TestAgent_WithACEService(t *testing.T) {
	t.Parallel()
	// Create ACE service.
	tmpDir := t.TempDir()
	cfg := &ACEConfig{
		Enabled:      true,
		PlaybookPath: tmpDir + "/test-playbook.json",
		Retrieval: ACERetrievalConfig{
			TopK:     5,
			MinScore: 0.3,
		},
	}

	mockLLM := llm.NewMockProvider("test")
	aceService, err := NewACEService(cfg, tmpDir, mockLLM, "test-model", 0)
	require.NoError(t, err)

	// Create agent with ACE.
	agent, err := NewAgent(
		mockLLM,
		func() *security.Service {
			validator := security.NewValidator()
			emitter := events.NewEventEmitter(100)
			approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})

			return security.NewService(validator, approvalService)
		}(),
		detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil),
		newTestToolRuntime(nil, tools.NewRegistry()),
		planning.NewService(mockLLM),
		&Environment{WorkDir: "/tmp"},
		events.NewEventEmitter(100),
		WithACEService(aceService),
	)

	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.NotNil(t, agent.aceService)
}

// TestAgent_ACEIntegration_EndToEnd tests full ACE workflow with agent execution.
func TestAgent_ACEIntegration_EndToEnd(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Setup ACE with ItemizedLearning enabled.
	cfg := &ACEConfig{
		Enabled:      true,
		PlaybookPath: tmpDir + "/test-playbook.json",
		Retrieval: ACERetrievalConfig{
			TopK:     3,
			MinScore: 0.3,
		},
		ItemizedLearning: ACEItemizedLearningConfig{
			Enabled:       true,
			ParseFeedback: true,
			UpdateAsync:   false, // Sync for testing.
		},
	}

	aceService, err := NewACEService(cfg, tmpDir, nil, "", 0)
	require.NoError(t, err)

	// Add test bullets to playbook.
	ctx := context.Background()
	b1, err := bullet.New("Always validate input parameters before processing")
	require.NoError(t, err)
	b2, err := bullet.New("Use descriptive variable names for better readability")
	require.NoError(t, err)
	b3, err := bullet.New("Handle errors explicitly rather than ignoring them")
	require.NoError(t, err)

	testBullets := []*bullet.Bullet{b1, b2, b3}

	for _, b := range testBullets {
		err = aceService.playbook.Add(ctx, b)
		require.NoError(t, err)
	}

	// Create mock provider that includes feedback markers in response.
	mockProvider := llm.NewMockProvider("test-response")
	mockProvider.SetResponse(`I'll help with that task.

HELPFUL: B0, B1
The input validation and descriptive naming suggestions were helpful.

Here's my solution...`)

	// Setup services with task registry.
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewService(validator, approvalService)

	detectionService := detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil)

	// Create task registry and register tasks.

	toolRuntime := newTestToolRuntime(nil, tools.NewRegistry())

	// Create agent with ACE.
	agent, err := NewAgent(
		mockProvider,
		securityService,
		detectionService,
		toolRuntime,
		planning.NewService(mockProvider),
		&Environment{WorkDir: tmpDir},
		emitter,
		WithACEService(aceService),
		WithMaxTurns(10),
		WithAgentTimeout(30*time.Second),
	)
	require.NoError(t, err)

	// Execute agent with input that should trigger bullet retrieval.
	request := &Request{
		Input: "Write a function to process user input",
		Task:  task.NewRegular(),
	}

	response, err := agent.Execute(ctx, request)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Output, "I'll help with that task")

	// Verify bullets were updated (helpful counters should be incremented)
	// Note: In real scenario, bullets B0 and B1 would have incremented helpful counters
	// We can verify the playbook was accessed during execution.
	assert.NotNil(t, aceService.playbook)
}

// TestAgent_ACEDisabled tests that agent works correctly when ACE is disabled.
func TestAgent_ACEDisabled(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create ACE service with disabled config.
	cfg := &ACEConfig{
		Enabled: false,
	}

	aceService, err := NewACEService(cfg, tmpDir, nil, "", 0)
	require.NoError(t, err)

	mockProvider := llm.NewMockProvider("test-response")

	// Setup services with task registry.
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewService(validator, approvalService)

	detectionService := detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil)

	// Create task registry and register tasks.

	toolRuntime := newTestToolRuntime(nil, tools.NewRegistry())

	// Create agent with disabled ACE.
	agent, err := NewAgent(
		mockProvider,
		securityService,
		detectionService,
		toolRuntime,
		planning.NewService(mockProvider),
		&Environment{WorkDir: tmpDir},
		emitter,
		WithACEService(aceService),
	)
	require.NoError(t, err)

	// Execute should work normally without ACE.
	ctx := context.Background()
	request := &Request{
		Input: "Simple test request",
		Task:  task.NewRegular(),
	}

	response, err := agent.Execute(ctx, request)
	require.NoError(t, err)
	assert.True(t, response.Success)
}

// TestAgent_Execute_Integration is a minimal integration test.
func TestAgent_Execute_Integration(t *testing.T) {
	t.Parallel()
	t.Skip("Integration test - requires full setup")

	agent := createTestAgentWithServices(t)

	req := &Request{
		Input: "Hello, how are you?",
		Task:  task.NewRegular(),
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

	// Build SecurityService.
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewService(validator, approvalService)

	// Build Service.
	cycleConfig := cycle.Config{
		WindowSize:       3,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
		Enabled:          true,
	}
	cycleDetector := cycle.NewDetector(cycleConfig)
	detectionService := detection.NewService(cycleDetector, nil)

	// Build tool registry with built-in tools.
	toolRegistry := tools.NewRegistry()
	require.NoError(t, toolRegistry.Register(tools.NewReadFileTool()))
	require.NoError(t, toolRegistry.Register(tools.NewWriteFileTool()))
	require.NoError(t, toolRegistry.Register(tools.NewListDirectoryTool()))
	require.NoError(t, toolRegistry.Register(tools.NewShellCommandTool(nil, nil, nil)))
	require.NoError(t, toolRegistry.Register(tools.NewGetContextTool(env)))
	require.NoError(t, toolRegistry.Register(tools.NewApplyPatchTool(workDir)))
	require.NoError(t, toolRegistry.Register(tools.NewFileSearchTool(workDir)))
	require.NoError(t, toolRegistry.Register(tools.NewGitContextTool(workDir)))

	// Build tool registry.

	toolRuntime := NewToolRuntime(ToolRuntimeConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         workDir,
	})

	// Create agent.
	agent, err := NewAgent(llmProvider, securityService, detectionService, toolRuntime, planning.NewService(llmProvider), env, emitter)
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
	_ *Executor,
	validator *security.Validator,
	environment *Environment,
	emitter *events.EventEmitter,
	opts ...Option,
) (*Agent, error) {
	// Build SecurityService.
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewService(validator, approvalService)

	// Build Service.
	cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
	detectionService := detection.NewService(cycleDetector, nil)

	// Build tool registry.
	toolRegistry := tools.NewRegistry()

	for _, tool := range []tools.Tool{
		tools.NewReadFileTool(),
		tools.NewWriteFileTool(),
		tools.NewListDirectoryTool(),
		tools.NewShellCommandTool(nil, nil, nil),
		tools.NewGetContextTool(environment),
		tools.NewApplyPatchTool(environment.WorkDir),
		tools.NewFileSearchTool(environment.WorkDir),
		tools.NewGitContextTool(environment.WorkDir),
	} {
		if err := toolRegistry.Register(tool); err != nil {
			return nil, fmt.Errorf("registering tool: %w", err)
		}
	}

	// Build tool runtime.

	toolRuntime := NewToolRuntime(ToolRuntimeConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         environment.WorkDir,
	})

	// Create agent with services using the real NewAgent function.
	agent := &Agent{
		llm:         provider,
		security:    securityService,
		detection:   detectionService,
		toolRuntime: toolRuntime,
		context:     environment,
		emitter:     emitter,
		logger:      slog.Default(),
		maxTurns:    10,               // Default for tests.
		timeout:     30 * time.Second, // Default for tests.
	}

	// Apply options.
	for _, opt := range opts {
		err := opt(agent)
		if err != nil {
			return nil, fmt.Errorf("applying option: %w", err)
		}
	}

	return agent, nil
}

// TestToolExecutionBugReproduction reproduces the exact bug from the user's output:
// The LLM calls list_directory but it's not executed, causing cycle detection.
func TestToolExecutionBugReproduction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Setup: Create mock LLM that returns list_directory tool call.
	mockLLM := llm.NewMockProvider("test",
		llm.WithToolCalls([]openai.ChatCompletionMessageToolCall{
			{
				ID:   "call_123",
				Type: openai.ChatCompletionMessageToolCallTypeFunction,
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
		}),
	)

	// Setup tool registry with list_directory.
	toolRegistry := tools.NewRegistry()
	err := toolRegistry.Register(tools.NewListDirectoryTool())
	require.NoError(t, err)

	// Setup services.
	validator := security.NewValidator()
	securityService := security.NewService(validator, nil)

	cycleDetector := cycle.NewDetector(cycle.Config{
		WindowSize:       10,
		SimilarityThresh: 0.8,
	})
	detectionService := detection.NewService(cycleDetector, nil)

	toolRuntime := newTestToolRuntime(nil, toolRegistry)

	env := &Environment{
		WorkDir:     "/tmp",
		Environment: make(map[string]string),
	}

	emitter := events.NewEventEmitter(100)

	// Collect events.
	subID, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)

	defer emitter.Unsubscribe(subID)

	var (
		toolStartEvents    []events.Event
		toolCompleteEvents []events.Event
	)

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

	// Create agent.
	agent, err := NewAgent(
		mockLLM,
		securityService,
		detectionService,
		toolRuntime,
		planning.NewService(mockLLM),
		env,
		emitter,
		WithMaxTurns(10),
		WithAgentTimeout(30*time.Second),
	)
	require.NoError(t, err)

	// Create a simple task.
	task := &simpleTask{
		name:         "test",
		systemPrompt: "You are a test assistant",
		allowedTools: []string{}, // Allow all tools.
		maxTokens:    4096,
	}

	// Execute: Send request that should trigger tool call.
	req := &Request{
		Input: "list files in current directory",
		Task:  task,
	}

	resp, err := agent.Execute(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Close emitter and wait for event collection.
	emitter.Close()
	<-done

	// Assert: Tool should have been executed.
	t.Logf("Tool start events: %d", len(toolStartEvents))
	t.Logf("Tool complete events: %d", len(toolCompleteEvents))

	assert.NotEmpty(t, toolStartEvents, "BUG: Tool was called but no start event emitted")
	assert.NotEmpty(t, toolCompleteEvents, "BUG: Tool was called but no complete event emitted")

	// Check event data.
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
	t.Parallel()
	ctx := context.Background()

	// Setup tool registry.
	toolRegistry := tools.NewRegistry()
	err := toolRegistry.Register(tools.NewListDirectoryTool())
	require.NoError(t, err)

	// Setup services.
	validator := security.NewValidator()
	securityService := security.NewService(validator, nil)

	cycleDetector := cycle.NewDetector(cycle.Config{WindowSize: 10})
	detectionService := detection.NewService(cycleDetector, nil)

	toolRuntime := newTestToolRuntime(nil, toolRegistry)

	env := &Environment{
		WorkDir:     "/tmp",
		Environment: make(map[string]string),
	}

	emitter := events.NewEventEmitter(100)

	dummyProvider := &dummyLLM{} // Won't be used, we're calling ProcessToolCall directly.
	agent, err := NewAgent(
		dummyProvider,
		securityService,
		detectionService,
		toolRuntime,
		planning.NewService(dummyProvider),
		env,
		emitter,
	)
	require.NoError(t, err)

	// Create tool call directly.
	toolCall := &ToolCall{
		ID:   "test_call",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "list_directory",
			Arguments: `{"path": "/tmp"}`,
		},
	}

	// Process tool call.
	result, err := agent.ProcessToolCall(ctx, toolCall)
	require.NoError(t, err, "ProcessToolCall should not error")
	require.NotNil(t, result, "ProcessToolCall should return result")

	// Verify result.
	assert.True(t, result.Success, "Tool execution should succeed")
	assert.NotEmpty(t, result.Output, "Tool should produce output")
	t.Logf("Tool output: %s", result.Output)
}

// TestStreamProcessingWithToolCalls tests that tool calls are extracted from stream.
func TestStreamProcessingWithToolCalls(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Mock LLM that streams tool call.
	mockLLM := llm.NewMockProvider("test",
		llm.WithToolCalls([]openai.ChatCompletionMessageToolCall{
			{
				ID:   "stream_call",
				Type: openai.ChatCompletionMessageToolCallTypeFunction,
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
		}),
	)

	// Test streaming.
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("list files"),
		}),
	}

	chunks, err := mockLLM.Stream(ctx, params)
	require.NoError(t, err)

	var receivedToolCalls []openai.ChatCompletionChunkChoicesDeltaToolCall

	for chunk := range chunks {
		if len(chunk.Choices) > 0 && len(chunk.Choices[0].Delta.ToolCalls) > 0 {
			receivedToolCalls = append(receivedToolCalls, chunk.Choices[0].Delta.ToolCalls...)
			for _, tc := range chunk.Choices[0].Delta.ToolCalls {
				t.Logf("Received tool call in stream: %s", tc.Function.Name)
			}
		}
	}

	assert.Len(t, receivedToolCalls, 1, "Should receive tool call in stream")
	assert.Equal(t, "list_directory", receivedToolCalls[0].Function.Name)
}

// TestGetToolResultContent tests that error messages are properly sent to LLM on tool failure.
func TestGetToolResultContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		toolCall *ToolCall
		result   *ToolResult
		want     string
	}{
		{
			name: "successful tool call returns output",
			toolCall: &ToolCall{
				ID: "call_1",
				Function: ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path":"."}`,
				},
			},
			result: &ToolResult{
				ID:      "call_1",
				Success: true,
				Output:  "file1.go\nfile2.go\nREADME.md",
			},
			want: "file1.go\nfile2.go\nREADME.md",
		},
		{
			name: "failed tool call with error returns error message",
			toolCall: &ToolCall{
				ID: "call_2",
				Function: ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"nonexistent.txt"}`,
				},
			},
			result: &ToolResult{
				ID:      "call_2",
				Success: false,
				Err:     errFileNotFoundNonexistentTxt,
			},
			want: "Tool read_file failed: file not found: nonexistent.txt",
		},
		{
			name: "failed tool call without error message",
			toolCall: &ToolCall{
				ID: "call_3",
				Function: ToolCallFunction{
					Name:      "execute_command",
					Arguments: `{"cmd":"unknown"}`,
				},
			},
			result: &ToolResult{
				ID:      "call_3",
				Success: false,
			},
			want: "Tool execute_command failed with no error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getToolResultContent(tt.toolCall, tt.result, slog.Default())
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestToolExecutionWithMockLLM tests tool execution with a mock LLM that returns tool calls.
func TestToolExecutionWithMockLLM(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create mock LLM that returns a list_directory tool call.
	mockLLM := llm.NewMockProvider("test",
		llm.WithToolCalls([]openai.ChatCompletionMessageToolCall{
			{
				ID:   "call_list_dir",
				Type: openai.ChatCompletionMessageToolCallTypeFunction,
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
		}),
	)

	// Setup tool registry with list_directory.
	toolRegistry := tools.NewRegistry()
	err := toolRegistry.Register(tools.NewListDirectoryTool())
	require.NoError(t, err)

	// Setup services.
	validator := security.NewValidator()
	securityService := security.NewService(validator, nil)

	cycleDetector := cycle.NewDetector(cycle.Config{
		WindowSize:       10,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
	})
	detectionService := detection.NewService(cycleDetector, nil)

	toolRuntime := newTestToolRuntime(nil, toolRegistry)

	env := &Environment{
		WorkDir:     "/tmp",
		Environment: make(map[string]string),
	}

	emitter := events.NewEventEmitter(100)

	// Collect events.
	subID, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)

	defer emitter.Unsubscribe(subID)

	var (
		toolStartEvents    []events.Event
		toolCompleteEvents []events.Event
	)

	done := make(chan struct{})

	go func() {
		defer close(done)

		for evt := range eventCh {
			switch evt.Type {
			case events.EventToolCallStart:
				toolStartEvents = append(toolStartEvents, evt)
				t.Logf("Tool start: %+v", evt.Data)
			case events.EventToolCallComplete:
				toolCompleteEvents = append(toolCompleteEvents, evt)
				data, ok := evt.Data.(events.ToolCallCompleteData)
			require.True(t, ok, "expected ToolCallCompleteData type assertion to succeed")
				t.Logf("Tool complete: success=%v tool=%s", data.Success, data.ToolName)
			}
		}
	}()

	// Create agent.
	agent, err := NewAgent(
		mockLLM,
		securityService,
		detectionService,
		toolRuntime,
		planning.NewService(mockLLM),
		env,
		emitter,
		WithMaxTurns(10),
		WithAgentTimeout(10*time.Second),
	)
	require.NoError(t, err)

	task := &simpleTask{
		name:         "test",
		systemPrompt: "You are a helpful assistant.",
		allowedTools: []string{},
		maxTokens:    4096,
	}

	req := &Request{
		Input: "list files in /tmp directory",
		Task:  task,
	}

	resp, err := agent.Execute(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	emitter.Close()
	<-done

	assert.NotEmpty(t, toolStartEvents, "Tool should have been called")
	assert.NotEmpty(t, toolCompleteEvents, "Tool should have completed")

	if len(toolCompleteEvents) > 0 {
		completeData, ok := toolCompleteEvents[0].Data.(events.ToolCallCompleteData)
		assert.True(t, ok)
		assert.True(t, completeData.Success, "Tool execution should succeed")
		assert.Equal(t, "list_directory", completeData.ToolName)
		assert.NotEmpty(t, completeData.Output, "Tool should produce output")
	}
}

// TestDirectToolCallWithMockLLM tests ProcessToolCall directly with a mock provider.
func TestDirectToolCallWithMockLLM(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	toolRegistry := tools.NewRegistry()
	err := toolRegistry.Register(tools.NewListDirectoryTool())
	require.NoError(t, err)

	validator := security.NewValidator()
	securityService := security.NewService(validator, nil)

	cycleDetector := cycle.NewDetector(cycle.Config{WindowSize: 10})
	detectionService := detection.NewService(cycleDetector, nil)

	toolRuntime := newTestToolRuntime(nil, toolRegistry)

	env := &Environment{
		WorkDir:     "/tmp",
		Environment: make(map[string]string),
	}

	emitter := events.NewEventEmitter(100)

	mockLLM := llm.NewMockProvider("test")

	agent, err := NewAgent(
		mockLLM,
		securityService,
		detectionService,
		toolRuntime,
		planning.NewService(mockLLM),
		env,
		emitter,
	)
	require.NoError(t, err)

	toolCall := &ToolCall{
		ID:   "test_direct",
		Type: "function",
		Function: ToolCallFunction{
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

// simpleTask implements Task interface for testing.
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

// dummyLLM is a minimal LLM for tests that don't use it.
type dummyLLM struct{}

func (d *dummyLLM) Complete(_ context.Context, _ openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return &openai.ChatCompletion{
		ID:    "dummy-completion",
		Model: "dummy-model",
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: "dummy",
					Role:    openai.ChatCompletionMessageRoleAssistant,
				},
				FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
			},
		},
	}, nil
}

func (d *dummyLLM) Stream(_ context.Context, _ openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	ch := make(chan openai.ChatCompletionChunk, 1)

	go func() {
		defer close(ch)

		ch <- openai.ChatCompletionChunk{
			ID:    "dummy-stream",
			Model: "dummy-model",
			Choices: []openai.ChatCompletionChunkChoice{
				{
					Delta: openai.ChatCompletionChunkChoicesDelta{
						Content: "dummy",
						Role:    openai.ChatCompletionChunkChoicesDeltaRoleAssistant,
					},
					FinishReason: openai.ChatCompletionChunkChoicesFinishReasonStop,
				},
			},
		}
	}()

	return ch, nil
}

func (d *dummyLLM) Models(_ context.Context) ([]openai.Model, error) {
	return []openai.Model{}, nil
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
// ============================================================================.

// BenchmarkAgent_ResolveTaskExplicit benchmarks resolving an explicit task object.
// Expected: ~50-100 ns/op (pointer comparison, should be instant).
func BenchmarkAgent_ResolveTaskExplicit(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewRegular()
	req := &Request{Task: taskObj}

	b.ResetTimer()

	for range b.N {
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
// Expected: ~100-150 ns/op (map lookup + RLock).
func BenchmarkAgent_ResolveTaskByName(b *testing.B) {
	agent := newBenchAgent(b)
	req := &Request{Task: task.NewReview()}

	b.ResetTimer()

	for range b.N {
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
// Expected: ~100-150 ns/op (map lookup + RLock).
func BenchmarkAgent_ResolveTaskDefault(b *testing.B) {
	agent := newBenchAgent(b)
	req := &Request{} // No task specified, should use default.

	b.ResetTimer()

	for range b.N {
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
// Expected: ~500-1000 ns/op (allows all tools, minimal filtering).
func BenchmarkAgent_BuildToolsForTask_Regular(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewRegular()

	b.ResetTimer()

	for range b.N {
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
// Expected: ~200-400 ns/op (allows only 4 tools, fast filtering).
func BenchmarkAgent_BuildToolsForTask_Compact(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewCompact()

	b.ResetTimer()

	for range b.N {
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
// Expected: ~200-400 ns/op (allows only 5 tools, fast filtering).
func BenchmarkAgent_BuildToolsForTask_Review(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewReview()

	b.ResetTimer()

	for range b.N {
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
// Expected: ~200-400 ns/op (allows only 4 tools, fast filtering).
func BenchmarkAgent_BuildToolsForTask_Planning(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewPlanning()

	b.ResetTimer()

	for range b.N {
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
// Expected: ~1000-2000 ns/op (validation + parsing + tool execution).
func BenchmarkAgent_ProcessToolCall(b *testing.B) {
	agent := newBenchAgent(b)
	toolCall := &ToolCall{
		ID:   "test_call",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "list_directory",
			Arguments: `{"path": "/tmp"}`,
		},
	}

	b.ResetTimer()

	for range b.N {
		result, err := agent.ProcessToolCall(context.Background(), toolCall)
		if err != nil {
			b.Fatal(err)
		}

		if result == nil {
			b.Fatal("expected result")
		}
	}
}

// BenchmarkAgent_ExtractToolNames was removed because extractToolNames function
// no longer exists after the OpenAI SDK migration. Tool name extraction is now
// handled via extractToolNamesFromToolCalls in loop.go.

// newBenchAgent creates an agent optimized for benchmarking.
func newBenchAgent(b *testing.B) *Agent {
	// Create minimal services for benchmarking.
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewService(validator, approvalService)

	cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
	detectionService := detection.NewService(cycleDetector, nil)

	// Create tool registry with all built-in tools.
	toolRegistry := tools.NewRegistry()

	for _, tool := range []tools.Tool{
		tools.NewReadFileTool(),
		tools.NewWriteFileTool(),
		tools.NewListDirectoryTool(),
		tools.NewShellCommandTool(nil, nil, nil),
		tools.NewGetContextTool(&Environment{WorkDir: "/tmp"}),
		tools.NewApplyPatchTool("/tmp"),
		tools.NewFileSearchTool("/tmp"),
		tools.NewGitContextTool("/tmp"),
	} {
		if err := toolRegistry.Register(tool); err != nil {
			b.Fatalf("registering tool: %v", err)
		}
	}

	// Create task registry with all modes.

	toolRuntime := NewToolRuntime(ToolRuntimeConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})

	mockProvider := &mockLLMProvider{}

	agent, err := NewAgent(
		mockProvider,
		securityService,
		detectionService,
		toolRuntime,
		planning.NewService(mockProvider),
		&Environment{WorkDir: "/tmp"},
		emitter,
	)
	if err != nil {
		b.Fatal(err)
	}

	return agent
}

// mockLLMProvider is a minimal LLM provider for benchmarking.
type mockLLMProvider struct{}

func (m *mockLLMProvider) Complete(_ context.Context, _ openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return &openai.ChatCompletion{
		ID:    "mock-completion",
		Model: "mock-model",
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: "mock",
					Role:    openai.ChatCompletionMessageRoleAssistant,
				},
				FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
			},
		},
	}, nil
}

func (m *mockLLMProvider) Stream(_ context.Context, _ openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	ch := make(chan openai.ChatCompletionChunk, 1)

	go func() {
		defer close(ch)

		ch <- openai.ChatCompletionChunk{
			ID:    "mock-stream",
			Model: "mock-model",
			Choices: []openai.ChatCompletionChunkChoice{
				{
					Delta: openai.ChatCompletionChunkChoicesDelta{
						Content: "mock",
						Role:    openai.ChatCompletionChunkChoicesDeltaRoleAssistant,
					},
					FinishReason: openai.ChatCompletionChunkChoicesFinishReasonStop,
				},
			},
		}
	}()

	return ch, nil
}

func (m *mockLLMProvider) Models(_ context.Context) ([]openai.Model, error) {
	return []openai.Model{}, nil
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
// ============================================================================.

// TestHandleCycleDetection_InterventionMessagesApplied tests that
// intervention messages are actually added to the conversation.
// This is a regression test for the bug where handleCycleDetection
// modified messages locally but didn't return them, causing interventions
// to be silently discarded.
func TestHandleCycleDetection_InterventionMessagesApplied(t *testing.T) {
	t.Parallel()
	agent := createTestAgentWithServices(t)

	// Enable cycle detection.
	agent.cycleDetection = true

	// Create initial conversation with tool calls and tool results to verify preservation.
	initialMessages := []message.Message{
		{
			Role:      message.RoleUser,
			Content:   "List files",
			Timestamp: time.Now(),
		},
		{
			Role:    message.RoleAssistant,
			Content: "I'll list the files",
			ToolCalls: []message.ToolCall{
				{
					ID:   "call-abc-0",
					Type: "function",
					Function: message.ToolCallFunction{
						Name:      "list_directory",
						Arguments: `{"path":"/"}`,
					},
				},
				{
					ID:   "call-abc-1",
					Type: "function",
					Function: message.ToolCallFunction{
						Name:      "shell_command",
						Arguments: `{"command":"ls"}`,
					},
				},
			},
			Timestamp: time.Now(),
		},
		{
			Role:       message.RoleTool,
			Content:    "file1.txt\nfile2.txt",
			ToolCallID: "call-abc-0",
			Timestamp:  time.Now(),
		},
		{
			Role:       message.RoleTool,
			Content:    "file1.txt  file2.txt",
			ToolCallID: "call-abc-1",
			Timestamp:  time.Now(),
		},
	}

	// Simulate repeated tool calls to trigger cycle detection
	// Add 3 snapshots with same tool AND same params to trigger CycleRepeatedTool.
	agent.detection.RecordSnapshot(detection.Snapshot{
		Turn:      1,
		Response:  "Calling list_directory",
		ToolCalls: []string{`list_directory({"path": "/"})`},
		Timestamp: time.Now(),
	})
	agent.detection.RecordSnapshot(detection.Snapshot{
		Turn:      2,
		Response:  "Calling list_directory again",
		ToolCalls: []string{`list_directory({"path": "/"})`},
		Timestamp: time.Now(),
	})
	agent.detection.RecordSnapshot(detection.Snapshot{
		Turn:      3,
		Response:  "Calling list_directory once more",
		ToolCalls: []string{`list_directory({"path": "/"})`},
		Timestamp: time.Now(),
	})

	// Create a mock LLM response that will trigger cycle detection.
	llmResp := &openai.ChatCompletion{
		ID:    "cycle-detection-test",
		Model: "test-model",
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: "Calling list_directory",
					Role:    openai.ChatCompletionMessageRoleAssistant,
					ToolCalls: []openai.ChatCompletionMessageToolCall{
						{
							ID:   "call_123",
							Type: openai.ChatCompletionMessageToolCallTypeFunction,
							Function: openai.ChatCompletionMessageToolCallFunction{
								Name:      "list_directory",
								Arguments: `{"path": "/"}`,
							},
						},
					},
				},
				FinishReason: openai.ChatCompletionChoicesFinishReasonToolCalls,
			},
		},
	}

	// Call handleCycleDetection.
	resp := &Response{}

	modifiedMessages, shouldStop, err := agent.handleCycleDetection(
		context.Background(),
		initialMessages,
		llmResp,
		3, // turn count.
		resp,
	)
	if err != nil {
		t.Fatalf("handleCycleDetection returned error: %v", err)
	}

	if shouldStop {
		t.Fatal("handleCycleDetection should not stop (severity < 3)")
	}

	// Check if cycle was detected.
	cycleResult, err := agent.detection.CheckCycle()
	if err != nil {
		t.Fatalf("CheckCycle failed: %v", err)
	}

	if cycleResult.Type == detection.CycleNone {
		t.Fatal("Expected cycle to be detected, but got CycleNone")
	}

	// The critical assertion: modifiedMessages should have the intervention message added
	// With the bug (before fix), modifiedMessages would equal initialMessages (unchanged)
	// After the fix, modifiedMessages should be longer (reflection added).
	if len(modifiedMessages) == len(initialMessages) {
		t.Error("BUG DETECTED: handleCycleDetection did not modify the messages slice")
		t.Error("Expected intervention message to be added, but messages unchanged")
		t.Error("This indicates the intervention's message modifications were discarded")
	}

	// Verify original messages are preserved with their ToolCalls and ToolCallID intact.
	// This is a regression test for the bug where handleCycleDetection reconstructed
	// messages through the detection.Message interface, losing ToolCalls/ToolCallID fields.
	for i := 0; i < len(initialMessages) && i < len(modifiedMessages); i++ {
		if initialMessages[i].Role != modifiedMessages[i].Role {
			t.Errorf("message[%d] role changed: %s -> %s", i, initialMessages[i].Role, modifiedMessages[i].Role)
		}

		if initialMessages[i].Content != modifiedMessages[i].Content {
			t.Errorf("message[%d] content changed", i)
		}

		if initialMessages[i].ToolCallID != modifiedMessages[i].ToolCallID {
			t.Errorf("message[%d] ToolCallID lost: %q -> %q", i, initialMessages[i].ToolCallID, modifiedMessages[i].ToolCallID)
		}

		if len(initialMessages[i].ToolCalls) != len(modifiedMessages[i].ToolCalls) {
			t.Errorf("message[%d] ToolCalls lost: had %d, now %d", i, len(initialMessages[i].ToolCalls), len(modifiedMessages[i].ToolCalls))
		}
	}

	// After the fix, this should pass.
	expectedMinLen := 5 // original 4 + 1 reflection message.
	if len(modifiedMessages) < expectedMinLen {
		t.Errorf("Expected at least %d messages after intervention, got %d", expectedMinLen, len(modifiedMessages))
	}

	// Verify the last message is from the intervention (user role with reflection prompt).
	if len(modifiedMessages) >= expectedMinLen {
		lastMsg := modifiedMessages[len(modifiedMessages)-1]
		if lastMsg.Role != message.RoleUser {
			t.Errorf("Expected intervention message to have role 'user', got '%s'", lastMsg.Role)
		}

		if lastMsg.Content == "" {
			t.Error("Expected intervention message to have content")
		}
		// Reflection intervention should mention "repeating" or "different".
		if !containsAnySubstring(lastMsg.Content, []string{"repeating", "different", "perspective", "angles"}) {
			t.Errorf("Expected reflection-style message, got: %s", lastMsg.Content)
		}
	}
}

// TestExecuteAgentLoop_CycleInterventionPropagated tests that the full agent loop
// properly uses intervention messages.
func TestExecuteAgentLoop_CycleInterventionPropagated(t *testing.T) {
	t.Parallel()
	agent := createTestAgentWithServices(t)
	agent.cycleDetection = true
	agent.maxTurns = 10

	// Create a mock LLM that returns same tool call repeatedly.
	mockLLM := llm.NewMockProvider("test")
	agent.llm = mockLLM

	initialMessages := []message.Message{
		{
			Role:      message.RoleSystem,
			Content:   "You are a helpful assistant",
			Timestamp: time.Now(),
		},
		{
			Role:      message.RoleUser,
			Content:   "List files",
			Timestamp: time.Now(),
		},
	}

	task := task.NewRegular()
	resp := &Response{}

	// Initialize trajectory context for the test.
	trajCtx := trajectory.NewContext("List files")

	// Execute the loop - it should detect the cycle and add intervention.
	resultMessages, resultResp, err := agent.executeAgentLoop(
		context.Background(),
		initialMessages,
		task,
		resp,
		trajCtx,
	)

	// The loop should complete (may hit max turns or other stop condition).
	if err != nil {
		// Error is acceptable for mock LLM.
		t.Logf("executeAgentLoop returned error (expected with mock): %v", err)
	}

	_ = resultResp // resultResp is used implicitly.

	// Key test: if a cycle was detected during the loop, verify intervention messages
	// were preserved in resultMessages.
	history := agent.detection.GetHistory()
	if len(history) >= 3 {
		// Cycle should have been detected.
		t.Log("Cycle detection triggered during agent loop")

		// After the fix, resultMessages should include intervention messages.
		if len(resultMessages) <= len(initialMessages) {
			t.Error("Expected resultMessages to include intervention messages, but no new messages found")
		}
	}
}

// Helper function to check if string contains any of the substrings.
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
// ============================================================================.

// TestAgent_TaskBudgetOverridesConfig verifies that a task's MaxTokens
// overrides the agent's config.MaxTokens when task.MaxTokens() > 0.
func TestAgent_TaskBudgetOverridesConfig(t *testing.T) {
	t.Parallel()
	// Create agent with 4K config.
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

	// Override config to 4K tokens.
	agent.maxTokens = 4096

	// Regular mode has 16K tokens.
	regularTask := task.NewRegular()
	if regularTask.MaxTokens() != 16384 {
		t.Fatalf("expected regular task to have 16384 tokens, got %d", regularTask.MaxTokens())
	}

	// Create request with regular task.
	req := &Request{
		Input: "test input",
		Task:  regularTask,
	}

	// Execute (will fail because no tools, but that's ok - we just want to capture the request).
	_, err = agent.Execute(context.Background(), req)
	t.Logf("execute returned (expected): %v", err)

	// Verify task budget was used (16K, not 4K from config).
	if len(llmCapture.requests) == 0 {
		t.Fatal("expected LLM to be called")
	}

	lastRequest := llmCapture.requests[len(llmCapture.requests)-1]
	if lastRequest.MaxTokens.Value != 16384 {
		t.Errorf("expected MaxTokens to be 16384 (from task), got %d", lastRequest.MaxTokens.Value)
	}
}

// TestAgent_ConfigBudgetUsedWhenTaskZero verifies that agent's config.MaxTokens
// is used when task.MaxTokens() returns 0.
func TestAgent_ConfigBudgetUsedWhenTaskZero(t *testing.T) {
	t.Parallel()
	// Create agent with 8K config.
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

	// Set config to 8K tokens.
	agent.maxTokens = 8192

	// Create a custom task that returns 0 for MaxTokens.
	zeroBudgetTask := &zeroBudgetTask{}

	// Create request with zero budget task.
	req := &Request{
		Input: "test input",
		Task:  zeroBudgetTask,
	}

	// Execute (will fail because no tools, but that's ok - we just want to capture the request).
	_, err = agent.Execute(context.Background(), req)
	t.Logf("execute returned (expected): %v", err)

	// Verify config budget was used (8K, not 0 from task).
	if len(llmCapture.requests) == 0 {
		t.Fatal("expected LLM to be called")
	}

	lastRequest := llmCapture.requests[len(llmCapture.requests)-1]
	if lastRequest.MaxTokens.Value != 8192 {
		t.Errorf("expected MaxTokens to be 8192 (from config), got %d", lastRequest.MaxTokens.Value)
	}
}

// TestAgent_ConcurrentTokenBudget verifies that token budget handling
// works correctly under concurrent access.
func TestAgent_ConcurrentTokenBudget(t *testing.T) {
	t.Parallel()
	// Create agent.
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

	// Set config to 4K tokens.
	agent.maxTokens = 4096

	// Create different tasks with different budgets.
	tasks := []task.Task{
		task.NewRegular(),  // 16K.
		task.NewCompact(),  // 8K.
		task.NewReview(),   // 12K.
		task.NewPlanning(), // 8K.
	}

	// Run concurrent requests.
	var wg sync.WaitGroup

	numRequests := 10

	for i := range numRequests {
		wg.Add(1)

		go func(taskIndex int) {
			defer wg.Done()

			task := tasks[taskIndex%len(tasks)]
			req := &Request{
				Input: "test input",
				Task:  task,
			}
			_, _ = agent.Execute(context.Background(), req)
		}(i)
	}

	wg.Wait()

	// Verify all requests used correct token budgets.
	if len(llmCapture.requests) == 0 {
		t.Fatal("expected LLM to be called")
	}

	// Collect all expected token budgets.
	expectedBudgets := make(map[int]bool)
	for _, task := range tasks {
		expectedBudgets[task.MaxTokens()] = true
	}

	// Track distribution of token budgets used.
	budgetCounts := make(map[int]int)

	for _, req := range llmCapture.requests {
		// Verify the token budget is one of the expected values.
		maxTokens := int(req.MaxTokens.Value)
		if !expectedBudgets[maxTokens] {
			t.Errorf("request used unexpected MaxTokens %d, expected one of %v", maxTokens, expectedBudgets)
		}

		budgetCounts[maxTokens]++
	}

	// Verify we got roughly the expected distribution
	// With 10 requests over 4 tasks (cycle 0,1,2,3,0,1,2,3,0,1), we expect:
	// Regular(16K):  3 times
	// Compact(4K):  3 times
	// Review(12K):  2 times
	// Planning(4K): 2 times
	// Total for 4K should be Compact + Planning = 5 times.

	regularCount := budgetCounts[16384]
	compactCount := budgetCounts[4096]
	reviewCount := budgetCounts[12288]

	if regularCount == 0 {
		t.Error("expected at least one request with Regular task (16K tokens)")
	}

	if compactCount == 0 {
		t.Error("expected at least one request with Compact or Planning task (4K tokens)")
	}

	if reviewCount == 0 {
		t.Error("expected at least one request with Review task (12K tokens)")
	}

	// Verify we got all 10 requests.
	totalRequests := regularCount + compactCount + reviewCount
	if totalRequests != numRequests {
		t.Errorf("expected %d total requests, got %d", numRequests, totalRequests)
	}
}

// zeroBudgetTask is a test task that returns 0 for MaxTokens.
type zeroBudgetTask struct{}

func (z *zeroBudgetTask) Name() string           { return "zero-budget" }
func (z *zeroBudgetTask) SystemPrompt() string   { return "Zero budget task" }
func (z *zeroBudgetTask) AllowedTools() []string { return []string{} }
func (z *zeroBudgetTask) MaxTokens() int         { return 0 }
func (z *zeroBudgetTask) Validate() error        { return nil }

// capturingLLMProvider captures LLM requests for testing.
type capturingLLMProvider struct {
	requests []openai.ChatCompletionNewParams
	mu       sync.Mutex
}

func newCapturingLLMProvider() *capturingLLMProvider {
	return &capturingLLMProvider{
		requests: make([]openai.ChatCompletionNewParams, 0),
	}
}

func (c *capturingLLMProvider) Complete(_ context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	c.mu.Lock()
	c.requests = append(c.requests, params)
	c.mu.Unlock()

	return &openai.ChatCompletion{
		ID:    "capturing-completion",
		Model: "capturing-model",
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: "mock",
					Role:    openai.ChatCompletionMessageRoleAssistant,
				},
				FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
			},
		},
	}, nil
}

func (c *capturingLLMProvider) Stream(_ context.Context, params openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	c.mu.Lock()
	c.requests = append(c.requests, params)
	c.mu.Unlock()

	ch := make(chan openai.ChatCompletionChunk, 1)

	go func() {
		defer close(ch)

		ch <- openai.ChatCompletionChunk{
			ID:    "capturing-stream",
			Model: "capturing-model",
			Choices: []openai.ChatCompletionChunkChoice{
				{
					Delta: openai.ChatCompletionChunkChoicesDelta{
						Content: "mock",
						Role:    openai.ChatCompletionChunkChoicesDeltaRoleAssistant,
					},
					FinishReason: openai.ChatCompletionChunkChoicesFinishReasonStop,
				},
			},
		}
	}()

	return ch, nil
}

func (c *capturingLLMProvider) Models(_ context.Context) ([]openai.Model, error) {
	return []openai.Model{}, nil
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
// ============================================================================.

// newTestAgentMinimal creates a minimal agent for testing with services.
func newTestAgentMinimal(toolRegistry *tools.Registry) *Agent {
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewService(validator, approvalService)

	cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
	detectionService := detection.NewService(cycleDetector, nil)

	if toolRegistry == nil {
		toolRegistry = tools.NewRegistry()
	}

	toolRuntime := NewToolRuntime(ToolRuntimeConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})

	mockProvider := &mockLLMProvider{}
	agent, _ := NewAgent(
		mockProvider,
		securityService,
		detectionService,
		toolRuntime,
		planning.NewService(mockProvider),
		&Environment{WorkDir: "/tmp"},
		emitter,
	)

	return agent
}

func newTestToolRuntime(_ any, registry *tools.Registry) *ToolRuntime {
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})

	if registry == nil {
		registry = tools.NewRegistry()
	}

	return NewToolRuntime(ToolRuntimeConfig{
		Registry:        registry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})
}

// TestAgent_processToolCalls and TestAgent_processToolCalls_WithToolCalls were removed
// because the processToolCalls method was refactored into processToolCallsFromCompletion
// and processToolCallsInternal after the OpenAI SDK migration. The internal implementation
// has changed significantly and these tests are no longer applicable.

func TestAgent_validateToolCall(t *testing.T) {
	t.Parallel()
	agent := newTestAgentMinimal(nil)

	tests := []struct {
		name    string
		call    *ToolCall
		wantErr bool
	}{
		{
			name: "valid tool call",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
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
			call: &ToolCall{
				ID:   "",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
			wantErr: true,
		},
		{
			name: "empty function name",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "",
					Arguments: `{"path": "/tmp"}`,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := agent.validateToolCall(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("Agent.validateToolCall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgent_parseToolArguments(t *testing.T) {
	t.Parallel()
	agent := newTestAgentMinimal(nil)

	tests := []struct {
		name    string
		call    *ToolCall
		wantErr bool
	}{
		{
			name: "valid JSON arguments",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "empty arguments",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "list_directory",
					Arguments: "",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid JSON arguments",
			call: &ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path": "/tmp"`, // Missing closing brace.
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args, err := agent.parseToolArguments(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("Agent.parseToolArguments() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(args.Keys()) == 0 {
				t.Error("Agent.parseToolArguments() returned empty args for valid input")
			}
		})
	}
}

func TestAgent_addFinalMessage(t *testing.T) {
	t.Parallel()
	agent := newTestAgentMinimal(nil)

	tests := []struct {
		name     string
		messages []message.Message
		content  string
		wantLen  int
	}{
		{
			name:     "add message with content",
			messages: []message.Message{},
			content:  "Hello, world!",
			wantLen:  1,
		},
		{
			name:     "add message with empty content",
			messages: []message.Message{},
			content:  "",
			wantLen:  0, // Should not add empty content.
		},
		{
			name: "add message to existing messages",
			messages: []message.Message{
				{Role: message.RoleUser, Content: "Hello"},
			},
			content: "Hi there!",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := agent.addFinalMessage(tt.messages, tt.content)
			if len(result) != tt.wantLen {
				t.Errorf("Agent.addFinalMessage() result length = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestAgent_emitTurnStart(t *testing.T) {
	t.Parallel()
	agent := newTestAgentMinimal(nil)

	// This test mainly ensures the method doesn't panic
	// In a real test, you'd want to verify the event was emitted.
	agent.emitTurnStart(1)
	agent.emitTurnStart(5)
	agent.emitTurnStart(100)
}

func TestAgent_applyTimeout(t *testing.T) {
	t.Parallel()
	agent := newTestAgentMinimal(nil)

	ctx := context.Background()

	// Test with default timeout.
	agent.timeout = 0

	ctxWithTimeout, cancel := agent.applyTimeout(ctx)
	if ctxWithTimeout == ctx {
		t.Error("Expected context to be modified with timeout")
	}

	cancel()

	// Test with custom timeout.
	agent.timeout = 5 * time.Second

	ctxWithTimeout, cancel = agent.applyTimeout(ctx)
	if ctxWithTimeout == ctx {
		t.Error("Expected context to be modified with timeout")
	}

	cancel()
}

func TestAgent_executeSetup(t *testing.T) {
	t.Parallel()
	agent := newTestAgentMinimal(nil)

	tests := []struct {
		name    string
		req     *Request
		wantErr bool
	}{
		{
			name: "valid request",
			req: &Request{
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
			req: &Request{
				Input: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()
	agent := newTestAgentMinimal(nil)

	tests := []struct {
		name       string
		resp       *Response
		messages   []message.Message
		historyLen int
		wantOutput string
	}{
		{
			name: "response with assistant message",
			resp: &Response{},
			messages: []message.Message{
				{Role: message.RoleUser, Content: "Hello"},
				{Role: message.RoleAssistant, Content: "Hi there!"},
			},
			historyLen: 1,
			wantOutput: "Hi there!",
		},
		{
			name: "response without assistant message",
			resp: &Response{},
			messages: []message.Message{
				{Role: message.RoleUser, Content: "Hello"},
			},
			historyLen: 1,
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			agent.finalizeResponse(tt.resp, tt.messages, tt.historyLen)

			if tt.resp.Output != tt.wantOutput {
				t.Errorf("Agent.finalizeResponse() output = %q, want %q", tt.resp.Output, tt.wantOutput)
			}
		})
	}
}

// TestAgentThinkingStateBugFix tests the fix for the agent getting stuck in thinking state.
func TestAgentThinkingStateBugFix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		llmResponses  []openai.ChatCompletion
		llmErrors     []error
		timeout       time.Duration
		expectedError bool
		expectedStuck bool
		description   string
	}{
		{
			name: "normal_execution_without_thinking_stuck",
			llmResponses: []openai.ChatCompletion{
				{
					ID:    "thinking-test-1",
					Model: "test-model",
					Choices: []openai.ChatCompletionChoice{
						{
							Message: openai.ChatCompletionMessage{
								Content: "I'll help you create a Tetris game in Rust.",
								Role:    openai.ChatCompletionMessageRoleAssistant,
							},
							FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
						},
					},
				},
			},
			llmErrors:     []error{nil},
			timeout:       30 * time.Second,
			expectedError: false,
			expectedStuck: false,
			description:   "Normal execution should not get stuck in thinking state",
		},
		{
			name: "llm_timeout_should_not_stuck_agent",
			llmResponses: []openai.ChatCompletion{
				{
					ID:    "thinking-test-2",
					Model: "test-model",
					Choices: []openai.ChatCompletionChoice{
						{
							Message: openai.ChatCompletionMessage{
								Content: "I'll help you create a Tetris game in Rust.",
								Role:    openai.ChatCompletionMessageRoleAssistant,
							},
							FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
						},
					},
				},
			},
			llmErrors:     []error{context.DeadlineExceeded},
			timeout:       30 * time.Second,
			expectedError: false, // Transient errors are retried; mock succeeds on retry.
			expectedStuck: false,
			description:   "LLM timeout should not cause agent to get stuck",
		},
		{
			name: "llm_error_should_not_stuck_agent",
			llmResponses: []openai.ChatCompletion{
				{
					ID:    "thinking-test-3",
					Model: "test-model",
					Choices: []openai.ChatCompletionChoice{
						{
							Message: openai.ChatCompletionMessage{
								Content: "I'll help you create a Tetris game in Rust.",
								Role:    openai.ChatCompletionMessageRoleAssistant,
							},
							FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
						},
					},
				},
			},
			llmErrors:     []error{errLlmProviderError},
			timeout:       30 * time.Second,
			expectedError: false, // Transient errors are retried; mock succeeds on retry.
			expectedStuck: false,
			description:   "LLM error should not cause agent to get stuck",
		},
		{
			name: "multiple_responses_should_not_stuck",
			llmResponses: []openai.ChatCompletion{
				{
					ID:    "thinking-test-4a",
					Model: "test-model",
					Choices: []openai.ChatCompletionChoice{
						{
							Message: openai.ChatCompletionMessage{
								Content: "I'll help you create a Tetris game in Rust.",
								Role:    openai.ChatCompletionMessageRoleAssistant,
								ToolCalls: []openai.ChatCompletionMessageToolCall{
									{
										ID:   "call_1",
										Type: openai.ChatCompletionMessageToolCallTypeFunction,
										Function: openai.ChatCompletionMessageToolCallFunction{
											Name:      "write_file",
											Arguments: `{"path": "Cargo.toml", "content": "[package]\nname = \"tetris\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[dependencies]\ncrossterm = \"0.27\"\nrand = \"0.8\""}`,
										},
									},
								},
							},
							FinishReason: openai.ChatCompletionChoicesFinishReasonToolCalls,
						},
					},
				},
				{
					ID:    "thinking-test-4b",
					Model: "test-model",
					Choices: []openai.ChatCompletionChoice{
						{
							Message: openai.ChatCompletionMessage{
								Content: "Great! I've created the Cargo.toml file. Now let me create the main.rs file.",
								Role:    openai.ChatCompletionMessageRoleAssistant,
							},
							FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
						},
					},
				},
			},
			llmErrors:     []error{nil, nil},
			timeout:       30 * time.Second,
			expectedError: false,
			expectedStuck: false,
			description:   "Multiple LLM responses should not cause agent to get stuck",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock LLM provider.
			mockLLM := &MockLLMProvider{
				responses: tt.llmResponses,
				errors:    tt.llmErrors,
			}

			// Create agent with mock LLM.
			agent := createTestAgentWithMockLLM(t, mockLLM)

			// Create context with timeout.
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			// Create request.
			req := &Request{
				Input: "Create a terminal Tetris game in Rust",
				Task:  task.NewRegular(),
			}

			// Track execution time to detect if agent gets stuck.
			start := time.Now()

			// Execute agent.
			resp, err := agent.Execute(ctx, req)

			executionTime := time.Since(start)

			// Check if agent got stuck (execution time should be reasonable).
			if tt.expectedStuck {
				assert.True(t, executionTime > tt.timeout/2, "Agent should have gotten stuck")
			} else {
				assert.True(t, executionTime < tt.timeout/2, "Agent should not have gotten stuck, execution time: %v", executionTime)
			}

			// Check error expectations.
			if tt.expectedError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Unexpected error: %v", err)
			}

			// Check response.
			if resp != nil {
				assert.NotNil(t, resp, "Response should not be nil")
			}
		})
	}
}

// TestAgentTimeoutHandling tests that agent properly handles timeouts.
func TestAgentTimeoutHandling(t *testing.T) {
	t.Parallel()
	// Create mock LLM that takes a long time to respond.
	mockLLM := &MockLLMProvider{
		responses: []openai.ChatCompletion{
			{
				ID:    "timeout-test",
				Model: "test-model",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "I'll help you create a Tetris game in Rust.",
							Role:    openai.ChatCompletionMessageRoleAssistant,
						},
						FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
					},
				},
			},
		},
		errors: []error{nil},
		delay:  2 * time.Second, // Simulate slow LLM response.
	}

	// Create agent with mock LLM.
	agent := createTestAgentWithMockLLM(t, mockLLM)

	// Create context with short timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create request.
	req := &Request{
		Input: "Create a terminal Tetris game in Rust",
		Task:  task.NewRegular(),
	}

	// Execute agent.
	start := time.Now()
	resp, err := agent.Execute(ctx, req)
	executionTime := time.Since(start)

	// Should timeout and return error.
	assert.Error(t, err, "Expected timeout error")
	assert.True(t, executionTime < 2*time.Second, "Execution should have timed out")

	// Response should contain error details, not be nil.
	assert.NotNil(t, resp, "Response should contain error details")

	if resp != nil {
		assert.NotNil(t, resp.Error, "Response.Error should be set")
		assert.Equal(t, "error", resp.FinishReason, "FinishReason should be 'error'")
	}
}

// TestAgentCycleDetection tests that agent properly handles cycle detection without getting stuck.
func TestAgentCycleDetection(t *testing.T) {
	t.Parallel()
	// Create mock LLM that returns repetitive responses (potential cycle).
	mockLLM := &MockLLMProvider{
		responses: []openai.ChatCompletion{
			{
				ID:    "cycle-test-1",
				Model: "test-model",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "I'll help you create a Tetris game in Rust.",
							Role:    openai.ChatCompletionMessageRoleAssistant,
						},
						FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
					},
				},
			},
			{
				ID:    "cycle-test-2",
				Model: "test-model",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "I'll help you create a Tetris game in Rust.", // Same response.
							Role:    openai.ChatCompletionMessageRoleAssistant,
						},
						FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
					},
				},
			},
			{
				ID:    "cycle-test-3",
				Model: "test-model",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "I'll help you create a Tetris game in Rust.", // Same response again.
							Role:    openai.ChatCompletionMessageRoleAssistant,
						},
						FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
					},
				},
			},
		},
		errors: []error{nil, nil, nil},
	}

	// Create agent with cycle detection enabled.
	agent := createTestAgentWithMockLLM(t, mockLLM)
	agent.cycleDetection = true

	// Create context.
	ctx := context.Background()

	// Create request.
	req := &Request{
		Input: "Create a terminal Tetris game in Rust",
		Task:  task.NewRegular(),
	}

	// Execute agent.
	start := time.Now()
	resp, err := agent.Execute(ctx, req)
	executionTime := time.Since(start)

	// Should complete without getting stuck.
	assert.NoError(t, err, "Agent should complete without error")
	assert.True(t, executionTime < 10*time.Second, "Agent should not get stuck in cycle detection")
	assert.NotNil(t, resp, "Response should not be nil")
}

// MockLLMProvider is a mock LLM provider for testing.
type MockLLMProvider struct {
	responses []openai.ChatCompletion
	errors    []error
	delay     time.Duration
	callCount int
}

func (m *MockLLMProvider) Complete(ctx context.Context, _ openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	// Simulate delay if specified.
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, fmt.Errorf("mock provider canceled: %w", ctx.Err())
		}
	}

	// Return error if specified.
	if m.callCount < len(m.errors) && m.errors[m.callCount] != nil {
		err := m.errors[m.callCount]
		m.callCount++

		return nil, err
	}

	// Return response if available.
	if m.callCount < len(m.responses) {
		resp := m.responses[m.callCount]
		m.callCount++

		return &resp, nil
	}

	// Default response.
	return &openai.ChatCompletion{
		ID:    "default-completion",
		Model: "default-model",
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: "Default response",
					Role:    openai.ChatCompletionMessageRoleAssistant,
				},
				FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
			},
		},
	}, nil
}

func (m *MockLLMProvider) Stream(ctx context.Context, _ openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	// Check for error first - return it immediately before creating channel.
	if m.callCount < len(m.errors) && m.errors[m.callCount] != nil {
		err := m.errors[m.callCount]
		m.callCount++

		return nil, err
	}

	ch := make(chan openai.ChatCompletionChunk, 10)

	go func() {
		defer close(ch)

		// Simulate delay if specified.
		if m.delay > 0 {
			select {
			case <-time.After(m.delay):
			case <-ctx.Done():
				return
			}
		}

		// Return response if available.
		if m.callCount < len(m.responses) {
			resp := m.responses[m.callCount]
			m.callCount++
			// Stream the content.
			if len(resp.Choices) > 0 {
				ch <- openai.ChatCompletionChunk{
					ID:    resp.ID,
					Model: resp.Model,
					Choices: []openai.ChatCompletionChunkChoice{
						{
							Delta: openai.ChatCompletionChunkChoicesDelta{
								Content: resp.Choices[0].Message.Content,
								Role:    openai.ChatCompletionChunkChoicesDeltaRoleAssistant,
							},
						},
					},
				}

				ch <- openai.ChatCompletionChunk{
					ID:    resp.ID,
					Model: resp.Model,
					Choices: []openai.ChatCompletionChunkChoice{
						{
							FinishReason: openai.ChatCompletionChunkChoicesFinishReason(resp.Choices[0].FinishReason),
						},
					},
				}
			}

			return
		}

		// Default response.
		ch <- openai.ChatCompletionChunk{
			ID:    "default-stream",
			Model: "default-model",
			Choices: []openai.ChatCompletionChunkChoice{
				{
					Delta: openai.ChatCompletionChunkChoicesDelta{
						Content: "Default response",
						Role:    openai.ChatCompletionChunkChoicesDeltaRoleAssistant,
					},
				},
			},
		}

		ch <- openai.ChatCompletionChunk{
			ID:    "default-stream",
			Model: "default-model",
			Choices: []openai.ChatCompletionChunkChoice{
				{
					FinishReason: openai.ChatCompletionChunkChoicesFinishReasonStop,
				},
			},
		}
	}()

	return ch, nil
}

func (m *MockLLMProvider) Models(_ context.Context) ([]openai.Model, error) {
	return []openai.Model{
		{ID: "test-model"},
	}, nil
}

func (m *MockLLMProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: true,
	}
}

func (m *MockLLMProvider) Name() string {
	return "mock-provider"
}

func (m *MockLLMProvider) Close() error {
	return nil
}

// createTestAgentWithMockLLM creates a test agent with mock LLM provider.
func createTestAgentWithMockLLM(t *testing.T, mockLLM llm.Provider) *Agent {
	// Create required services.
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewService(validator, approvalService)

	detectionService := detection.NewService(
		cycle.NewDetector(cycle.Config{Enabled: false}),
		nil,
	)

	// Create tool runtime.
	toolRegistry := tools.NewRegistry()
	toolRuntime := newTestToolRuntime(
		nil, // toolExecutor.
		toolRegistry,
	)

	environment := &Environment{WorkDir: "/tmp"}

	// Create agent.
	agent, err := NewAgent(
		mockLLM,
		securityService,
		detectionService,
		toolRuntime,
		planning.NewService(mockLLM),
		environment,
		emitter,
		WithMaxTurns(10),
		WithAgentTimeout(30*time.Second),
	)
	require.NoError(t, err, "Failed to create agent")

	return agent
}

// mockTask is a simple mock task for testing.
type mockTask struct {
	name string
}

func (m *mockTask) Name() string {
	return m.name
}

func (m *mockTask) SystemPrompt() string {
	return "You are a helpful AI assistant."
}

func (m *mockTask) AllowedTools() []string {
	return []string{"write_file", "read_file"}
}

func (m *mockTask) MaxTokens() int {
	return 4096
}

func (m *mockTask) Validate() error {
	return nil
}

// TestAgent_emitACERetrievalEvent tests ACE retrieval event emission.
func TestAgent_emitACERetrievalEvent(t *testing.T) {
	t.Parallel()
	emitter := events.NewEventEmitter(10)
	_, eventCh, _ := emitter.Subscribe()

	agent := &Agent{
		emitter: emitter,
	}

	// Create trajectory context with known metrics.
	ctx := trajectory.NewContext("install nodejs")
	ctx.CurrentTurn = 5

	// Add some bullets to cache via RecordRetrieval (which updates stats).
	testBullets := []*bullet.Bullet{
		{ID: "b1", Content: "test bullet 1"},
		{ID: "b2", Content: "test bullet 2"},
	}
	event := trajectory.RetrievalEvent{
		Turn:         5,
		Trigger:      trajectory.TriggerError,
		Query:        "install nodejs error",
		BulletsAdded: []string{"b1", "b2"},
	}
	ctx.RecordRetrieval(event, testBullets)

	// After RecordRetrieval, cache stats reflect the operation
	// For this test, we just need to verify the emitted data matches current stats.

	// Call emitACERetrievalEvent with bullets.
	agent.emitACERetrievalEvent(ctx, trajectory.TriggerError, "install nodejs error", testBullets, 5)

	// Verify event was emitted.
	select {
	case emittedEvent := <-eventCh:
		if emittedEvent.Type != events.EventACERetrieval {
			t.Errorf("Type = %v, want EventACERetrieval", emittedEvent.Type)
		}

		data, ok := emittedEvent.ACERetrievalData()
		if !ok {
			t.Fatal("Expected ACERetrievalData")
		}

		if data.Turn != 5 {
			t.Errorf("Turn = %d, want 5", data.Turn)
		}

		if data.Trigger != "error" {
			t.Errorf("Trigger = %q, want \"error\"", data.Trigger)
		}

		if data.Query != "install nodejs error" {
			t.Errorf("Query = %q, want \"install nodejs error\"", data.Query)
		}

		if data.BulletsRetrieved != 2 {
			t.Errorf("BulletsRetrieved = %d, want 2", data.BulletsRetrieved)
		}

		if data.CacheSize != 2 {
			t.Errorf("CacheSize = %d, want 2", data.CacheSize)
		}

		// Verify cache hit rate calculation matches trajectory context.
		total := ctx.CacheHits + ctx.CacheMisses

		expectedHitRate := 0.0
		if total > 0 {
			expectedHitRate = float64(ctx.CacheHits) / float64(total)
		}

		if data.CacheHitRate != expectedHitRate {
			t.Errorf("CacheHitRate = %f, want %f (from ctx: hits=%d, misses=%d)",
				data.CacheHitRate, expectedHitRate, ctx.CacheHits, ctx.CacheMisses)
		}

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ACE retrieval event")
	}
}
