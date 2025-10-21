// +build integration

package agent

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm/factory"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
