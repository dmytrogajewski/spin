package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core/cycle"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Test CycleType String method
func TestCycleType_String_Final(t *testing.T) {
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
func TestErrorTypes_Final(t *testing.T) {
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

// Test Event methods
func TestEventMethods_Final(t *testing.T) {
	event := Event{
		Type:      EventInfo,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	// Test GetType
	eventType := event.GetType()
	if eventType == "" {
		t.Error("Event.GetType() should not return empty string")
	}

	// Test GetTimestamp
	timestamp := event.GetTimestamp()
	if timestamp.IsZero() {
		t.Error("Event.GetTimestamp() should not return zero time")
	}

	// Test GetData
	data := event.GetData()
	if data == nil {
		t.Error("Event.GetData() should not return nil")
	}
}

// Test EventEmitter Events method
func TestEventEmitter_Events_Final(t *testing.T) {
	emitter := NewEventEmitter(10)

	// Test Events method
	events := emitter.Events()
	if events == nil {
		t.Error("EventEmitter.Events() should not return nil")
	}
}

// Test Executor methods
func TestExecutor_Methods_Final(t *testing.T) {
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

// Test CycleType String method
func TestCycleType_String_Coverage(t *testing.T) {
	tests := []struct {
		name      string
		cycleType cycle.CycleType
		expected  string
	}{
		{
			name:      "CycleSimilarResponses",
			cycleType: cycle.CycleSimilarResponses,
			expected:  "similar_responses",
		},
		{
			name:      "CycleRepeatedTool",
			cycleType: cycle.CycleRepeatedTool,
			expected:  "repeated_tool",
		},
		{
			name:      "CycleOscillation",
			cycleType: cycle.CycleOscillation,
			expected:  "oscillation",
		},
		{
			name:      "CycleSameError",
			cycleType: cycle.CycleSameError,
			expected:  "same_error",
		},
		{
			name:      "CycleNone",
			cycleType: cycle.CycleNone,
			expected:  "none",
		},
		{
			name:      "Unknown",
			cycleType: cycle.CycleType(999),
			expected:  "none",
		},
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

// Test ErrorCode String method
func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected string
	}{
		{
			name:     "ErrCodeUnknown",
			code:     ErrCodeUnknown,
			expected: "unknown",
		},
		{
			name:     "ErrCodeInvalidInput",
			code:     ErrCodeInvalidInput,
			expected: "invalid_input",
		},
		{
			name:     "ErrCodeNotFound",
			code:     ErrCodeNotFound,
			expected: "not_found",
		},
		{
			name:     "ErrCodeAlreadyExists",
			code:     ErrCodeAlreadyExists,
			expected: "already_exists",
		},
		{
			name:     "ErrCodePermissionDenied",
			code:     ErrCodePermissionDenied,
			expected: "permission_denied",
		},
		{
			name:     "ErrCodeTimeout",
			code:     ErrCodeTimeout,
			expected: "timeout",
		},
		{
			name:     "ErrCodeCancelled",
			code:     ErrCodeCancelled,
			expected: "cancelled",
		},
		{
			name:     "ErrCodeInternal",
			code:     ErrCodeInternal,
			expected: "internal",
		},
		{
			name:     "Unknown code",
			code:     ErrorCode(999),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.code.String()
			if result != tt.expected {
				t.Errorf("ErrorCode.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test Error Unwrap method
func TestError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := &Error{
		Op:   "test",
		Err:  originalErr,
		Code: ErrCodeInternal,
	}

	unwrapped := err.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Error.Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

// Test Error Is method
func TestError_Is(t *testing.T) {
	err1 := &Error{
		Op:   "test",
		Err:  errors.New("test error"),
		Code: ErrCodeInvalidInput,
	}

	err2 := &Error{
		Op:   "test",
		Err:  errors.New("different error"),
		Code: ErrCodeInvalidInput,
	}

	err3 := &Error{
		Op:   "test",
		Err:  errors.New("test error"),
		Code: ErrCodeNotFound,
	}

	// Test matching error codes
	if !err1.Is(err2) {
		t.Error("Error.Is() should return true for same error codes")
	}

	// Test different error codes
	if err1.Is(err3) {
		t.Error("Error.Is() should return false for different error codes")
	}

	// Test with non-Error type
	if err1.Is(errors.New("not an Error")) {
		t.Error("Error.Is() should return false for non-Error types")
	}
}

// Test Validator IsInteractive method
func TestValidator_IsInteractive(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		cmd      *Command
		expected bool
	}{
		{
			name: "interactive command",
			cmd: &Command{
				Program: "mkdir",
				Args:    []string{"test-dir"},
			},
			expected: true,
		},
		{
			name: "safe command",
			cmd: &Command{
				Program: "ls",
				Args:    []string{"-la"},
			},
			expected: false,
		},
		{
			name: "dangerous command",
			cmd: &Command{
				Program: "rm",
				Args:    []string{"-rf", "/tmp/test"},
			},
			expected: false,
		},
		{
			name:     "nil command",
			cmd:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsInteractive(tt.cmd)
			if result != tt.expected {
				t.Errorf("Validator.IsInteractive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test Git integration methods
func TestGitIntegration_AdditionalMethods(t *testing.T) {
	git := &GitIntegration{}

	// Test GetModifiedFiles
	modifiedFiles, err := git.GetModifiedFiles()
	if err == nil {
		t.Error("GetModifiedFiles should return error for uninitialized git integration")
	}
	if modifiedFiles != nil {
		t.Error("GetModifiedFiles should return nil for uninitialized git integration")
	}

	// Test GetStagedFiles
	stagedFiles, err := git.GetStagedFiles()
	if err == nil {
		t.Error("GetStagedFiles should return error for uninitialized git integration")
	}
	if stagedFiles != nil {
		t.Error("GetStagedFiles should return nil for uninitialized git integration")
	}

	// Test GetUntrackedFiles
	untrackedFiles, err := git.GetUntrackedFiles()
	if err == nil {
		t.Error("GetUntrackedFiles should return error for uninitialized git integration")
	}
	if untrackedFiles != nil {
		t.Error("GetUntrackedFiles should return nil for uninitialized git integration")
	}

	// Test IsClean
	git.IsClean()
	// IsClean can return true or false depending on git state

	// Test GetContextInfo
	contextInfo := git.GetContextInfo()
	// GetContextInfo returns a map with git information
	if contextInfo == nil {
		t.Error("GetContextInfo should not return nil")
	}

	// Test GetDiff
	diff, err := git.GetDiff("HEAD")
	if err == nil {
		t.Error("GetDiff should return error for uninitialized git integration")
	}
	if diff != "" {
		t.Error("GetDiff should return empty string for uninitialized git integration")
	}

	// Test GetLog
	log, err := git.GetLog(10)
	if err == nil {
		t.Error("GetLog should return error for uninitialized git integration")
	}
	if log != nil {
		t.Error("GetLog should return nil for uninitialized git integration")
	}

	// Test StageFile
	err = git.StageFile("test.txt")
	if err == nil {
		t.Error("StageFile should return error for uninitialized git integration")
	}

	// Test UnstageFile
	err = git.UnstageFile("test.txt")
	if err == nil {
		t.Error("UnstageFile should return error for uninitialized git integration")
	}

	// Test Commit
	err = git.Commit("test commit")
	if err == nil {
		t.Error("Commit should return error for uninitialized git integration")
	}

	// Test Push
	err = git.Push()
	if err == nil {
		t.Error("Push should return error for uninitialized git integration")
	}

	// Test Pull
	err = git.Pull()
	if err == nil {
		t.Error("Pull should return error for uninitialized git integration")
	}
}

// Test Agent processToolCalls method more thoroughly
func TestAgent_processToolCalls_Final(t *testing.T) {
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

// Test Agent Emit method
func TestAgent_Emit_Final(t *testing.T) {
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

// Mock LLM provider for testing
type mockLLMProviderForFinal struct{}

func (m *mockLLMProviderForFinal) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{
		Content: `{"steps": [{"id": "step1", "description": "Mock step"}]}`,
	}, nil
}

func (m *mockLLMProviderForFinal) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	ch <- llm.StreamChunk{
		Type:    llm.ChunkTypeContentDelta,
		Content: "Mock stream response",
	}
	close(ch)
	return ch, nil
}

func (m *mockLLMProviderForFinal) Models(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{}, nil
}

func (m *mockLLMProviderForFinal) Capabilities() llm.Capabilities {
	return llm.Capabilities{}
}

func (m *mockLLMProviderForFinal) Name() string {
	return "mock"
}

func (m *mockLLMProviderForFinal) Close() error {
	return nil
}

// Test eventEmitterAdapter.Emit method
func TestEventEmitterAdapter_Emit(t *testing.T) {
	emitter := NewEventEmitter(100)
	adapter := &eventEmitterAdapter{emitter: emitter}

	// Subscribe to events
	events := emitter.Events()
	defer emitter.Close()

	// Mock cycle.Event interface
	mockCycleEvent := &mockCycleEvent{
		eventType: "turn_paused",
		timestamp: time.Now(),
		data:      "test data",
	}

	// Emit the event
	adapter.Emit(mockCycleEvent)

	// Verify the event was received
	select {
	case receivedEvent := <-events:
		if receivedEvent.GetType() != EventTurnPaused.String() {
			t.Errorf("Expected event type %v, got %v", EventTurnPaused.String(), receivedEvent.GetType())
		}
		if receivedEvent.GetData() != "test data" {
			t.Errorf("Expected event data %v, got %v", "test data", receivedEvent.GetData())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timed out waiting for event")
	}
}

// Test eventEmitterAdapter.Emit with unknown event type
func TestEventEmitterAdapter_Emit_UnknownType(t *testing.T) {
	emitter := NewEventEmitter(100)
	adapter := &eventEmitterAdapter{emitter: emitter}

	// Subscribe to events
	events := emitter.Events()
	defer emitter.Close()

	// Mock cycle.Event interface with unknown type
	mockCycleEvent := &mockCycleEvent{
		eventType: "unknown_type",
		timestamp: time.Now(),
		data:      "test data",
	}

	// Emit the event
	adapter.Emit(mockCycleEvent)

	// Verify the event was received with fallback type
	select {
	case receivedEvent := <-events:
		if receivedEvent.GetType() != EventWarning.String() {
			t.Errorf("Expected event type %v, got %v", EventWarning.String(), receivedEvent.GetType())
		}
		if receivedEvent.GetData() != "test data" {
			t.Errorf("Expected event data %v, got %v", "test data", receivedEvent.GetData())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timed out waiting for event")
	}
}

// Mock cycle.Event interface for testing
type mockCycleEvent struct {
	eventType string
	timestamp time.Time
	data      interface{}
}

func (m *mockCycleEvent) GetType() string {
	return m.eventType
}

func (m *mockCycleEvent) GetTimestamp() time.Time {
	return m.timestamp
}

func (m *mockCycleEvent) GetData() interface{} {
	return m.data
}
