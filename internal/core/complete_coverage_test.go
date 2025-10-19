package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core/cycle"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/state"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Test Agent.Emit method
func TestAgent_Emit(t *testing.T) {
	agent := &Agent{
		emitter: NewEventEmitter(10),
	}

	event := Event{
		Type:      EventInfo,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	// Test that emitter is set
	if agent.emitter == nil {
		t.Error("Agent.emitter should not be nil")
	}

	// Test emitting event through emitter
	agent.emitter.Emit(event)
}

// Test Config merge functions
func TestConfig_MergeFunctions(t *testing.T) {
	base := Config{
		MaxTurns:    10,
		Temperature: 0.7,
		MaxTokens:   4096,
		CycleDetection: CycleDetectionConfig{
			Enabled:          true,
			SimilarityThresh: 0.8,
		},
		ProviderConfig: map[string]interface{}{
			"openai": map[string]interface{}{
				"api_key": "test-key",
			},
		},
		Provider:        "openai",
		AllowedCommands: []string{"ls", "cat"},
		RequireApproval: false,
	}

	other := Config{
		MaxTurns:    15,
		Temperature: 0.8,
		MaxTokens:   8192,
		CycleDetection: CycleDetectionConfig{
			Enabled:          false,
			SimilarityThresh: 0.9,
		},
		ProviderConfig: map[string]interface{}{
			"ollama": map[string]interface{}{
				"model": "llama2",
			},
		},
		Provider:        "ollama",
		AllowedCommands: []string{"echo", "grep"},
		RequireApproval: true,
	}

	result := base.Merge(&other)

	// Test mergeCycleDetection
	if result.CycleDetection.Enabled != false {
		t.Errorf("mergeCycleDetection Enabled = %v, want false", result.CycleDetection.Enabled)
	}
	if result.CycleDetection.SimilarityThresh != 0.9 {
		t.Errorf("mergeCycleDetection SimilarityThresh = %v, want 0.9", result.CycleDetection.SimilarityThresh)
	}

	// Test mergeProviderConfig
	if len(result.ProviderConfig) != 2 {
		t.Errorf("mergeProviderConfig providers count = %v, want 2", len(result.ProviderConfig))
	}

	// Test mergeStringFields
	if result.Provider != "ollama" {
		t.Errorf("mergeStringFields Provider = %v, want ollama", result.Provider)
	}

	// Test mergeSliceFields
	if len(result.AllowedCommands) != 4 {
		t.Errorf("mergeSliceFields AllowedCommands length = %v, want 4", len(result.AllowedCommands))
	}

	// Test mergeBoolFields
	if result.RequireApproval != true {
		t.Errorf("mergeBoolFields RequireApproval = %v, want true", result.RequireApproval)
	}
}

// Test Conversation Stop method
func TestConversation_Stop(t *testing.T) {
	conv := &Conversation{
		state: state.StateRunning,
	}

	// Test Stop with context
	err := conv.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop should not return error: %v", err)
	}
}

// Test CycleType String method
func TestCycleType_String(t *testing.T) {
	tests := []struct {
		name      string
		cycleType cycle.CycleType
		expected  string
	}{
		{"CycleNone", cycle.CycleNone, "none"},
		{"CycleSimilarResponses", cycle.CycleSimilarResponses, "similar_responses"},
		{"CycleRepeatedTool", cycle.CycleRepeatedTool, "repeated_tool"},
		{"CycleOscillation", cycle.CycleOscillation, "oscillation"},
		{"CycleSameError", cycle.CycleSameError, "same_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cycleType.String()
			if result != tt.expected {
				t.Errorf("CycleType.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test Error types
func TestErrorTypes(t *testing.T) {
	// Test ValidationError
	err := NewValidationError("test error", "field")
	if err == nil {
		t.Fatal("NewValidationError should not return nil")
	}

	// Test Error method
	errStr := err.Error()
	if errStr == "" {
		t.Error("ValidationError.Error() should not return empty string")
	}
}

// Test EventEmitter Events method
func TestEventEmitter_Events(t *testing.T) {
	tests := []struct {
		name      string
		emitter   *EventEmitter
		wantError bool
	}{
		{
			name:      "successful subscription",
			emitter:   NewEventEmitter(10),
			wantError: false,
		},
		{
			name: "subscription failure",
			emitter: func() *EventEmitter {
				// Create an emitter that will fail subscription
				emitter := NewEventEmitter(10)
				// Close the emitter to simulate failure
				emitter.Close()
				return emitter
			}(),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := tt.emitter.Events()

			if tt.wantError {
				// Should get a closed channel
				select {
				case _, ok := <-events:
					if ok {
						t.Error("Events() expected closed channel, got open channel")
					}
				case <-time.After(100 * time.Millisecond):
					t.Error("Events() expected closed channel, got timeout")
				}
			} else {
				// Should get an open channel
				if events == nil {
					t.Error("EventEmitter.Events() should not return nil")
				}

				// Test that we can receive from the channel (it should be open)
				select {
				case <-events:
					// This shouldn't happen immediately, but channel should be open
				case <-time.After(10 * time.Millisecond):
					// Timeout is expected - channel is open but no events
				}
			}
		})
	}
}

// Test Executor methods
func TestExecutor_Methods(t *testing.T) {
	// Test Result Failed method
	result := Result{
		ExitCode: 1,
		Error:    errors.New("test error"),
	}

	if !result.Failed() {
		t.Error("Result.Failed() should return true for failed result")
	}

	// Test Result Output method
	result.Stdout = "test output"
	output := result.Output()
	if output != "test output" {
		t.Errorf("Result.Output() = %v, want test output", output)
	}

	// Test Executor methods
	executor, err := NewExecutor("/tmp")
	if err != nil {
		t.Fatalf("NewExecutor should not return error: %v", err)
	}

	// Test Validate method
	cmd := &Command{
		Program: "ls",
		Args:    []string{"-la"},
	}

	err = executor.Validate(cmd)
	if err != nil {
		t.Errorf("Validate should not return error: %v", err)
	}
}

// Test Git integration methods
func TestGitIntegration_Methods(t *testing.T) {
	git := &GitIntegration{}

	// Test IsEnabled
	enabled := git.IsEnabled()
	if enabled {
		t.Error("IsEnabled should return false for uninitialized git integration")
	}

	// Test GetRepository
	repo := git.GetRepository()
	if repo != nil {
		t.Error("GetRepository should return nil for uninitialized git integration")
	}

	// Test GetStatus
	status := git.GetStatus()
	if status != nil {
		t.Error("GetStatus should return nil for uninitialized git integration")
	}

	// Test RefreshStatus
	err := git.RefreshStatus(context.Background())
	if err != nil {
		t.Errorf("RefreshStatus should not return error: %v", err)
	}

	// Test GetBranch
	branch, err := git.GetBranch()
	if err == nil {
		t.Error("GetBranch should return error for uninitialized git integration")
	}
	if branch != "" {
		t.Error("GetBranch should return empty string for uninitialized git integration")
	}

	// Test GetRemoteURL
	remoteURL, err := git.GetRemoteURL()
	if err == nil {
		t.Error("GetRemoteURL should return error for uninitialized git integration")
	}
	if remoteURL != "" {
		t.Error("GetRemoteURL should return empty string for uninitialized git integration")
	}

	// Test GetCommitHash
	commitHash, err := git.GetCommitHash()
	if err == nil {
		t.Error("GetCommitHash should return error for uninitialized git integration")
	}
	if commitHash != "" {
		t.Error("GetCommitHash should return empty string for uninitialized git integration")
	}
}

// Test Executor streaming methods
func TestExecutor_Streaming(t *testing.T) {
	executor, err := NewExecutor("/tmp")
	if err != nil {
		t.Fatalf("NewExecutor should not return error: %v", err)
	}

	// Test ExecuteStreaming
	cmd := &Command{
		Program: "echo",
		Args:    []string{"test"},
	}

	stream, err := executor.ExecuteStreaming(context.Background(), cmd, DefaultExecuteOptions())
	if err != nil {
		t.Errorf("ExecuteStreaming should not return error: %v", err)
	}
	if stream == nil {
		t.Error("ExecuteStreaming should return non-nil stream")
	}

	// Test streamOutput method (internal)
	// This is tested indirectly through ExecuteStreaming
}

// Test Agent processToolCalls method more thoroughly
func TestAgent_processToolCalls_Thorough(t *testing.T) {
	agent := &Agent{
		toolRegistry: &tools.Registry{},
	}

	// Test with tool calls in LLM response
	llmResp := &llm.CompletionResponse{
		Content: `{"tool_calls": [{"id": "call1", "name": "read_file", "args": {"path": "test.txt"}}]}`,
	}

	messages := []Message{}
	resp := &AgentResponse{}

	result := agent.processToolCalls(context.Background(), messages, llmResp, resp)

	// Should add messages for tool calls
	if len(result) == 0 {
		t.Error("processToolCalls should add messages for tool calls")
	}
}

// Mock LLM provider for testing
type mockLLMProviderForCoverage struct{}

func (m *mockLLMProviderForCoverage) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{
		Content: `{"steps": [{"id": "step1", "description": "Mock step"}]}`,
	}, nil
}

func (m *mockLLMProviderForCoverage) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	ch <- llm.StreamChunk{
		Type:    llm.ChunkTypeContentDelta,
		Content: "Mock stream response",
	}
	close(ch)
	return ch, nil
}

func (m *mockLLMProviderForCoverage) Models(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{}, nil
}

func (m *mockLLMProviderForCoverage) Capabilities() llm.Capabilities {
	return llm.Capabilities{}
}

func (m *mockLLMProviderForCoverage) Name() string {
	return "mock"
}

func (m *mockLLMProviderForCoverage) Close() error {
	return nil
}
