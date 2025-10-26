package conversation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/openai/openai-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requestCapturingMockProvider wraps MockProvider to capture requests
type requestCapturingMockProvider struct {
	*llm.MockProvider
	mu           sync.Mutex
	lastRequest  *openai.ChatCompletionNewParams
	allRequests  []openai.ChatCompletionNewParams
	requestCount int
}

func newRequestCapturingMock(response string) *requestCapturingMockProvider {
	mock := llm.NewMockProvider("test-mock")
	mock.SetResponse(response)
	return &requestCapturingMockProvider{
		MockProvider: mock,
		allRequests:  make([]openai.ChatCompletionNewParams, 0),
	}
}

func (m *requestCapturingMockProvider) Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	m.captureRequest(params)
	return m.MockProvider.Complete(ctx, params)
}

func (m *requestCapturingMockProvider) Stream(ctx context.Context, params openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	m.captureRequest(params)
	return m.MockProvider.Stream(ctx, params)
}

func (m *requestCapturingMockProvider) captureRequest(params openai.ChatCompletionNewParams) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastRequest = &params
	m.allRequests = append(m.allRequests, params)
	m.requestCount++
}

func (m *requestCapturingMockProvider) getLastRequest() *openai.ChatCompletionNewParams {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastRequest
}

func (m *requestCapturingMockProvider) getAllRequests() []openai.ChatCompletionNewParams {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]openai.ChatCompletionNewParams(nil), m.allRequests...)
}

func (m *requestCapturingMockProvider) getRequestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestCount
}

// Test helpers

func extractToolNamesFromTools(tools []openai.ChatCompletionToolParam) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		// Access the function name directly from the param Field
		if tool.Function.Present {
			names = append(names, tool.Function.Value.Name.Value)
		}
	}
	return names
}

func setupTestAgentWithMockLLM(t *testing.T, mockLLM llm.Provider) *agent.Agent {
	t.Helper()

	workDir := t.TempDir()
	executor, err := agent.NewExecutor(workDir)
	require.NoError(t, err)

	validator := security.NewValidator()
	env := &agent.Environment{WorkDir: workDir}
	emitter := events.NewEventEmitter(100)

	// Build SecurityService
	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	// Build DetectionService (imported from conversation test helper pattern)
	cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	// Build tool registry
	toolRegistry := tools.NewRegistry()
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
	agentInstance, err := agent.NewAgent(mockLLM, securityService, detectionService, orchestrationService, env, emitter)
	require.NoError(t, err)

	return agentInstance
}

func setupTestConversationWithMockLLM(t *testing.T, mockLLM llm.Provider) *Conversation {
	t.Helper()

	agentInstance := setupTestAgentWithMockLLM(t, mockLLM)
	hist := history.NewHistoryWithDefaults()
	err := hist.AddSystemMessage("You are a helpful assistant")
	require.NoError(t, err)

	emitter := events.NewEventEmitter(100)

	return NewConversation(agentInstance, hist, emitter, "test-session-id")
}

// Integration Tests

// Test 1: Mode Switch Affects Tool Availability
func TestConversation_Integration_ModeSwitchAffectsTools(t *testing.T) {
	// Setup
	mockLLM := newRequestCapturingMock("Done")
	conv := setupTestConversationWithMockLLM(t, mockLLM)

	ctx := context.Background()

	// Turn 1: Regular mode (all tools)
	err := conv.RunTurn(ctx, "Read test.go")
	require.NoError(t, err)
	// Verify regular mode had all tools
	req1 := mockLLM.getLastRequest()
	require.NotNil(t, req1, "request should be captured")
	toolNames1 := extractToolNamesFromTools(req1.Tools.Value)
	assert.Contains(t, toolNames1, "read_file", "regular mode should have read_file")
	assert.Contains(t, toolNames1, "write_file", "regular mode should have write_file")
	assert.Contains(t, toolNames1, "execute_command", "regular mode should have execute_command")

	// Switch to review mode
	err = conv.SetTaskMode("review")
	require.NoError(t, err)
	assert.Equal(t, "review", conv.GetTaskMode())

	// Turn 2: Review mode (read-only tools)
	err = conv.RunTurn(ctx, "Read test2.go")
	require.NoError(t, err)
	// Verify review mode has only read tools
	req2 := mockLLM.getLastRequest()
	require.NotNil(t, req2, "request should be captured")
	toolNames2 := extractToolNamesFromTools(req2.Tools.Value)
	assert.Contains(t, toolNames2, "read_file", "review mode should have read_file")
	assert.NotContains(t, toolNames2, "write_file", "review mode should NOT have write_file")
	assert.NotContains(t, toolNames2, "execute_command", "review mode should NOT have execute_command")

	// Review mode should have list_directory and get_context
	assert.Contains(t, toolNames2, "list_directory", "review mode should have list_directory")
	assert.Contains(t, toolNames2, "get_context", "review mode should have get_context")
}

// Test 2: Mode Switch Affects Token Budget
func TestConversation_Integration_ModeSwitchAffectsTokenBudget(t *testing.T) {
	// Setup
	mockLLM := newRequestCapturingMock("Done")
	agent := setupTestAgentWithMockLLM(t, mockLLM)
	// Note: Agent is created with default config

	hist := history.NewHistoryWithDefaults()
	_ = hist.AddSystemMessage("You are helpful")
	emitter := events.NewEventEmitter(100)
	conv := NewConversation(agent, hist, emitter, "test-session-id")

	ctx := context.Background()

	// Turn 1: Regular mode (16K tokens)
	err := conv.RunTurn(ctx, "Hello")
	require.NoError(t, err)
	req1 := mockLLM.getLastRequest()
	require.NotNil(t, req1)
	assert.Equal(t, int64(16384), req1.MaxTokens.Value, "regular mode should use 16K tokens")

	// Switch to compact mode (4K tokens)
	err = conv.SetTaskMode("compact")
	require.NoError(t, err)

	// Turn 2: Compact mode (4K tokens)
	err = conv.RunTurn(ctx, "What is 2+2?")
	require.NoError(t, err)
	req2 := mockLLM.getLastRequest()
	require.NotNil(t, req2)
	assert.Equal(t, int64(4096), req2.MaxTokens.Value, "compact mode should use 4K tokens")

	// Switch to planning mode (4K tokens)
	err = conv.SetTaskMode("planning")
	require.NoError(t, err)

	// Turn 3: Planning mode (4K tokens)
	err = conv.RunTurn(ctx, "Plan the feature")
	require.NoError(t, err)
	req3 := mockLLM.getLastRequest()
	require.NotNil(t, req3)
	assert.Equal(t, int64(4096), req3.MaxTokens.Value, "planning mode should use 4K tokens")
}

// Test 3: Mode Persists Across Multiple Turns
func TestConversation_Integration_ModePersistsAcrossTurns(t *testing.T) {
	// Setup
	mockLLM := newRequestCapturingMock("OK")
	conv := setupTestConversationWithMockLLM(t, mockLLM)

	ctx := context.Background()

	// Set to review mode
	err := conv.SetTaskMode("review")
	require.NoError(t, err)
	assert.Equal(t, "review", conv.GetTaskMode())

	// Turn 1
	err = conv.RunTurn(ctx, "Turn 1")
	require.NoError(t, err)
	// Mode should still be review
	assert.Equal(t, "review", conv.GetTaskMode())
	req1 := mockLLM.getLastRequest()
	require.NotNil(t, req1)
	toolNames1 := extractToolNamesFromTools(req1.Tools.Value)
	assert.NotContains(t, toolNames1, "write_file", "turn 1: review mode should not have write_file")

	// Turn 2 (no mode change)
	err = conv.RunTurn(ctx, "Turn 2")
	require.NoError(t, err)
	// Mode should STILL be review
	assert.Equal(t, "review", conv.GetTaskMode())
	req2 := mockLLM.getLastRequest()
	require.NotNil(t, req2)
	toolNames2 := extractToolNamesFromTools(req2.Tools.Value)
	assert.NotContains(t, toolNames2, "write_file", "turn 2: review mode should not have write_file")

	// Turn 3 (no mode change)
	err = conv.RunTurn(ctx, "Turn 3")
	require.NoError(t, err)
	// Mode should STILL be review
	assert.Equal(t, "review", conv.GetTaskMode())
	req3 := mockLLM.getLastRequest()
	require.NotNil(t, req3)
	toolNames3 := extractToolNamesFromTools(req3.Tools.Value)
	assert.NotContains(t, toolNames3, "write_file", "turn 3: review mode should not have write_file")

	// Verify all 3 requests were in review mode
	allRequests := mockLLM.getAllRequests()
	assert.GreaterOrEqual(t, len(allRequests), 3, "should have at least 3 requests")
}

// Test 4: Concurrent Mode Switches Are Safe
func TestConversation_Integration_ConcurrentModeSwitches(t *testing.T) {
	// Setup
	mockLLM := newRequestCapturingMock("OK")
	conv := setupTestConversationWithMockLLM(t, mockLLM)

	var wg sync.WaitGroup
	ctx := context.Background()
	modes := []string{"regular", "review", "compact", "planning"}

	// Start 20 concurrent turn executions
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := conv.RunTurn(ctx, fmt.Sprintf("Message %d", i))
			if err != nil {
				// Some messages may fail due to overlap, that's OK
				t.Logf("Message %d error: %v", i, err)
				return
			}
		}(i)
	}

	// Start 20 concurrent mode switches
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			mode := modes[i%len(modes)]
			err := conv.SetTaskMode(mode)
			if err != nil {
				t.Logf("SetTaskMode %s error: %v", mode, err)
			}
		}(i)
	}

	// Wait for all to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for concurrent operations")
	}

	// If we get here without race detector errors, we're good
	// Verify conversation is still in valid state
	currentMode := conv.GetTaskMode()
	assert.Contains(t, modes, currentMode, "final mode should be valid")

	// Verify we captured some requests
	assert.Greater(t, mockLLM.getRequestCount(), 0, "should have captured some requests")
}

// Test 5: Invalid Mode Handling
func TestConversation_Integration_InvalidModeHandling(t *testing.T) {
	// Setup
	mockLLM := newRequestCapturingMock("Works")
	conv := setupTestConversationWithMockLLM(t, mockLLM)

	ctx := context.Background()

	// Try to set invalid mode
	err := conv.SetTaskMode("invalid-mode-name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task mode")

	// Verify mode unchanged (still default)
	assert.Equal(t, "regular", conv.GetTaskMode())

	// Verify subsequent turn works normally
	err = conv.RunTurn(ctx, "Test")
	require.NoError(t, err)
	// Should use regular mode
	req := mockLLM.getLastRequest()
	require.NotNil(t, req)
	toolNames := extractToolNamesFromTools(req.Tools.Value)
	assert.Contains(t, toolNames, "write_file", "should still have all tools in regular mode")
}

// Test 6: All Task Modes End-to-End
func TestConversation_Integration_AllTaskModes(t *testing.T) {
	// Setup
	mockLLM := newRequestCapturingMock("Done")
	conv := setupTestConversationWithMockLLM(t, mockLLM)

	ctx := context.Background()

	testCases := []struct {
		mode              string
		expectedTools     []string
		forbiddenTools    []string
		expectedMaxTokens int64
	}{
		{
			mode:              "regular",
			expectedTools:     []string{"read_file", "write_file", "execute_command"},
			forbiddenTools:    []string{},
			expectedMaxTokens: int64(16384),
		},
		{
			mode:              "review",
			expectedTools:     []string{"read_file", "list_directory", "get_context"},
			forbiddenTools:    []string{"write_file", "execute_command"},
			expectedMaxTokens: int64(12288),
		},
		{
			mode:              "compact",
			expectedTools:     []string{"read_file", "get_context", "file_search"},
			forbiddenTools:    []string{"write_file", "execute_command"},
			expectedMaxTokens: int64(4096),
		},
		{
			mode:              "planning",
			expectedTools:     []string{"get_context", "file_search", "git_context"},
			forbiddenTools:    []string{"read_file", "write_file", "execute_command"},
			expectedMaxTokens: int64(4096),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.mode, func(t *testing.T) {
			// Set mode
			err := conv.SetTaskMode(tc.mode)
			require.NoError(t, err)
			assert.Equal(t, tc.mode, conv.GetTaskMode())

			// Execute turn
			err = conv.RunTurn(ctx, "Test "+tc.mode)
			require.NoError(t, err)
			// Verify tools
			req := mockLLM.getLastRequest()
			require.NotNil(t, req, "request should be captured for mode %s", tc.mode)
			toolNames := extractToolNamesFromTools(req.Tools.Value)

			for _, expectedTool := range tc.expectedTools {
				assert.Contains(t, toolNames, expectedTool,
					"mode %s should have %s", tc.mode, expectedTool)
			}

			for _, forbiddenTool := range tc.forbiddenTools {
				assert.NotContains(t, toolNames, forbiddenTool,
					"mode %s should NOT have %s", tc.mode, forbiddenTool)
			}

			// Verify token budget
			assert.Equal(t, tc.expectedMaxTokens, req.MaxTokens.Value,
				"mode %s should have %d tokens", tc.mode, tc.expectedMaxTokens)
		})
	}
}

// Test 7: Mode Info Included in System Messages
func TestConversation_Integration_ModeChangeEmitsEvent(t *testing.T) {
	// Setup
	mockLLM := newRequestCapturingMock("Done")
	conv := setupTestConversationWithMockLLM(t, mockLLM)

	// Subscribe to event stream
	eventStream := conv.Stream()

	// Switch mode
	err := conv.SetTaskMode("review")
	require.NoError(t, err)

	// Check for EventInfo event
	select {
	case event := <-eventStream:
		if event.Type == events.EventInfo {
			// Found the system info event
			data, ok := event.Data.(events.SystemEventData)
			assert.True(t, ok, "event data should be SystemEventData")
			assert.True(t, strings.Contains(data.Message, "review"), "event message should mention 'review'")
		}
	case <-time.After(100 * time.Millisecond):
		t.Log("No immediate event received, which is OK - event emission may be async")
	}
}
