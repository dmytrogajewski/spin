package turn

import (
	"errors"
	"testing"
	"time"
)

func TestNewResult(t *testing.T) {
	// Create a completed turn
	turn := NewTurn("session-123", "test input")
	_ = turn.Start()
	turn.UpdateTokens(TokenUsage{
		PromptTokens:     50,
		CompletionTokens: 30,
		TotalTokens:      80,
	})
	turn.AddToolCall(ToolCall{ID: "call-1", Name: "shell"})
	turn.AddToolCall(ToolCall{ID: "call-2", Name: "read_file"})
	_ = turn.Complete("response", turn.Tokens)

	result := NewResult(turn)

	if result == nil {
		t.Fatal("NewResult() returned nil")
	}
	if !result.Success {
		t.Error("NewResult() Success should be true for completed turn")
	}
	if result.FinalState != StateCompleted {
		t.Errorf("NewResult() FinalState = %v, want %v", result.FinalState, StateCompleted)
	}
	if result.Response != "response" {
		t.Errorf("NewResult() Response = %v, want 'response'", result.Response)
	}
	if result.Tokens.TotalTokens != 80 {
		t.Errorf("NewResult() Tokens.TotalTokens = %v, want 80", result.Tokens.TotalTokens)
	}
	if result.ToolCount != 2 {
		t.Errorf("NewResult() ToolCount = %v, want 2", result.ToolCount)
	}
	if result.Duration == 0 {
		t.Error("NewResult() Duration should be greater than 0")
	}
}

func TestNewResult_Failed(t *testing.T) {
	// Create a failed turn
	turn := NewTurn("session-123", "test input")
	_ = turn.Start()
	testErr := errors.New("execution failed")
	_ = turn.Fail(testErr)

	result := NewResult(turn)

	if result == nil {
		t.Fatal("NewResult() returned nil")
	}
	if result.Success {
		t.Error("NewResult() Success should be false for failed turn")
	}
	if result.FinalState != StateFailed {
		t.Errorf("NewResult() FinalState = %v, want %v", result.FinalState, StateFailed)
	}
	if result.Error == nil {
		t.Error("NewResult() Error should be set for failed turn")
	}
	if !errors.Is(result.Error, testErr) {
		t.Errorf("NewResult() Error = %v, want %v", result.Error, testErr)
	}
}

func TestNewResult_Cancelled(t *testing.T) {
	// Create a cancelled turn
	turn := NewTurn("session-123", "test input")
	_ = turn.Start()
	_ = turn.Cancel()

	result := NewResult(turn)

	if result == nil {
		t.Fatal("NewResult() returned nil")
	}
	if result.Success {
		t.Error("NewResult() Success should be false for cancelled turn")
	}
	if result.FinalState != StateCancelled {
		t.Errorf("NewResult() FinalState = %v, want %v", result.FinalState, StateCancelled)
	}
}

func TestResult_IsSuccess(t *testing.T) {
	tests := []struct {
		name       string
		finalState TurnState
		want       bool
	}{
		{"Completed is success", StateCompleted, true},
		{"Failed is not success", StateFailed, false},
		{"Cancelled is not success", StateCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				Success:    tt.finalState == StateCompleted,
				FinalState: tt.finalState,
			}

			if got := result.IsSuccess(); got != tt.want {
				t.Errorf("Result.IsSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_GetError(t *testing.T) {
	testErr := errors.New("test error")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{"Has error", testErr, testErr},
		{"No error", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{Error: tt.err}

			got := result.GetError()
			if (got != nil) != (tt.want != nil) {
				t.Errorf("Result.GetError() = %v, want %v", got, tt.want)
			}
			if got != nil && !errors.Is(got, tt.want) {
				t.Errorf("Result.GetError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_Duration(t *testing.T) {
	turn := NewTurn("session-123", "test")
	_ = turn.Start()

	// Sleep to ensure duration is measurable
	time.Sleep(10 * time.Millisecond)

	_ = turn.Complete("done", TokenUsage{TotalTokens: 50})

	result := NewResult(turn)

	if result.Duration < 10*time.Millisecond {
		t.Errorf("Result.Duration = %v, want >= 10ms", result.Duration)
	}
}

func TestResult_ContextSize(t *testing.T) {
	turn := NewTurn("session-123", "test input")
	_ = turn.Start()
	_ = turn.Complete("test response", TokenUsage{TotalTokens: 50})

	result := NewResult(turn)

	// Context size should be at least the length of input + response
	expectedMinSize := len(turn.UserInput) + len(turn.AIResponse)
	if result.ContextSize < expectedMinSize {
		t.Errorf("Result.ContextSize = %v, want >= %v", result.ContextSize, expectedMinSize)
	}
}
