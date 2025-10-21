package agent

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
