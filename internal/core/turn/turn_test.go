package turn

import (
	"errors"
	"testing"
	"time"
)

func TestTurn_Start(t *testing.T) {
	tests := []struct {
		name         string
		initialState TurnState
		wantErr      bool
		wantState    TurnState
	}{
		{
			name:         "start from pending",
			initialState: StatePending,
			wantErr:      false,
			wantState:    StateRunning,
		},
		{
			name:         "start from running",
			initialState: StateRunning,
			wantErr:      true,
			wantState:    StateRunning,
		},
		{
			name:         "start from completed",
			initialState: StateCompleted,
			wantErr:      true,
			wantState:    StateCompleted,
		},
		{
			name:         "start from failed",
			initialState: StateFailed,
			wantErr:      true,
			wantState:    StateFailed,
		},
		{
			name:         "start from cancelled",
			initialState: StateCancelled,
			wantErr:      true,
			wantState:    StateCancelled,
		},
		{
			name:         "start from waiting approval",
			initialState: StateWaitingApproval,
			wantErr:      false,
			wantState:    StateRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{
				ID:        "test-turn",
				SessionID: "test-session",
				State:     tt.initialState,
			}

			err := turn.Start()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Turn.Start() expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Errorf("Turn.Start() expected ErrInvalidTransition, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Turn.Start() unexpected error: %v", err)
				}
				if turn.State != tt.wantState {
					t.Errorf("Turn.Start() state = %v, want %v", turn.State, tt.wantState)
				}
				if turn.StartedAt.IsZero() {
					t.Errorf("Turn.Start() StartedAt should be set")
				}
			}
		})
	}
}

func TestTurn_Complete(t *testing.T) {
	tests := []struct {
		name         string
		initialState TurnState
		response     string
		tokens       TokenUsage
		wantErr      bool
		wantState    TurnState
	}{
		{
			name:         "complete from running",
			initialState: StateRunning,
			response:     "Task completed successfully",
			tokens:       TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
			wantErr:      false,
			wantState:    StateCompleted,
		},
		{
			name:         "complete from pending",
			initialState: StatePending,
			response:     "Task completed",
			tokens:       TokenUsage{TotalTokens: 100},
			wantErr:      true,
			wantState:    StatePending,
		},
		{
			name:         "complete from completed",
			initialState: StateCompleted,
			response:     "Already done",
			tokens:       TokenUsage{TotalTokens: 10},
			wantErr:      true,
			wantState:    StateCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{
				ID:        "test-turn",
				SessionID: "test-session",
				State:     tt.initialState,
			}

			err := turn.Complete(tt.response, tt.tokens)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Turn.Complete() expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Errorf("Turn.Complete() expected ErrInvalidTransition, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Turn.Complete() unexpected error: %v", err)
				}
				if turn.State != tt.wantState {
					t.Errorf("Turn.Complete() state = %v, want %v", turn.State, tt.wantState)
				}
				if turn.AIResponse != tt.response {
					t.Errorf("Turn.Complete() AIResponse = %v, want %v", turn.AIResponse, tt.response)
				}
				if turn.Tokens != tt.tokens {
					t.Errorf("Turn.Complete() Tokens = %v, want %v", turn.Tokens, tt.tokens)
				}
				if turn.CompletedAt.IsZero() {
					t.Errorf("Turn.Complete() CompletedAt should be set")
				}
			}
		})
	}
}

func TestTurn_Fail(t *testing.T) {
	tests := []struct {
		name         string
		initialState TurnState
		err          error
		wantErr      bool
		wantState    TurnState
	}{
		{
			name:         "fail from running",
			initialState: StateRunning,
			err:          errors.New("test error"),
			wantErr:      false,
			wantState:    StateFailed,
		},
		{
			name:         "fail from pending",
			initialState: StatePending,
			err:          errors.New("test error"),
			wantErr:      true,
			wantState:    StatePending,
		},
		{
			name:         "fail from completed",
			initialState: StateCompleted,
			err:          errors.New("test error"),
			wantErr:      true,
			wantState:    StateCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{
				ID:        "test-turn",
				SessionID: "test-session",
				State:     tt.initialState,
			}

			err := turn.Fail(tt.err)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Turn.Fail() expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Errorf("Turn.Fail() expected ErrInvalidTransition, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Turn.Fail() unexpected error: %v", err)
				}
				if turn.State != tt.wantState {
					t.Errorf("Turn.Fail() state = %v, want %v", turn.State, tt.wantState)
				}
				if turn.Error != tt.err {
					t.Errorf("Turn.Fail() Error = %v, want %v", turn.Error, tt.err)
				}
				if turn.CompletedAt.IsZero() {
					t.Errorf("Turn.Fail() CompletedAt should be set")
				}
			}
		})
	}
}

func TestTurn_Cancel(t *testing.T) {
	tests := []struct {
		name         string
		initialState TurnState
		wantErr      bool
		wantState    TurnState
	}{
		{
			name:         "cancel from running",
			initialState: StateRunning,
			wantErr:      false,
			wantState:    StateCancelled,
		},
		{
			name:         "cancel from waiting approval",
			initialState: StateWaitingApproval,
			wantErr:      false,
			wantState:    StateCancelled,
		},
		{
			name:         "cancel from pending",
			initialState: StatePending,
			wantErr:      true,
			wantState:    StatePending,
		},
		{
			name:         "cancel from completed",
			initialState: StateCompleted,
			wantErr:      true,
			wantState:    StateCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{
				ID:        "test-turn",
				SessionID: "test-session",
				State:     tt.initialState,
			}

			err := turn.Cancel()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Turn.Cancel() expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Errorf("Turn.Cancel() expected ErrInvalidTransition, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Turn.Cancel() unexpected error: %v", err)
				}
				if turn.State != tt.wantState {
					t.Errorf("Turn.Cancel() state = %v, want %v", turn.State, tt.wantState)
				}
				if turn.CompletedAt.IsZero() {
					t.Errorf("Turn.Cancel() CompletedAt should be set")
				}
			}
		})
	}
}

func TestTurn_RequestApproval(t *testing.T) {
	tests := []struct {
		name         string
		initialState TurnState
		wantErr      bool
		wantState    TurnState
	}{
		{
			name:         "request approval from running",
			initialState: StateRunning,
			wantErr:      false,
			wantState:    StateWaitingApproval,
		},
		{
			name:         "request approval from pending",
			initialState: StatePending,
			wantErr:      true,
			wantState:    StatePending,
		},
		{
			name:         "request approval from waiting approval",
			initialState: StateWaitingApproval,
			wantErr:      true,
			wantState:    StateWaitingApproval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{
				ID:        "test-turn",
				SessionID: "test-session",
				State:     tt.initialState,
			}

			err := turn.RequestApproval()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Turn.RequestApproval() expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Errorf("Turn.RequestApproval() expected ErrInvalidTransition, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Turn.RequestApproval() unexpected error: %v", err)
				}
				if turn.State != tt.wantState {
					t.Errorf("Turn.RequestApproval() state = %v, want %v", turn.State, tt.wantState)
				}
			}
		})
	}
}

func TestTurn_Approve(t *testing.T) {
	tests := []struct {
		name         string
		initialState TurnState
		wantErr      bool
		wantState    TurnState
	}{
		{
			name:         "approve from waiting approval",
			initialState: StateWaitingApproval,
			wantErr:      false,
			wantState:    StateRunning,
		},
		{
			name:         "approve from running",
			initialState: StateRunning,
			wantErr:      true,
			wantState:    StateRunning,
		},
		{
			name:         "approve from pending",
			initialState: StatePending,
			wantErr:      true,
			wantState:    StatePending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{
				ID:        "test-turn",
				SessionID: "test-session",
				State:     tt.initialState,
			}

			err := turn.Approve()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Turn.Approve() expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Errorf("Turn.Approve() expected ErrInvalidTransition, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Turn.Approve() unexpected error: %v", err)
				}
				if turn.State != tt.wantState {
					t.Errorf("Turn.Approve() state = %v, want %v", turn.State, tt.wantState)
				}
			}
		})
	}
}

func TestTurn_Deny(t *testing.T) {
	tests := []struct {
		name         string
		initialState TurnState
		wantErr      bool
		wantState    TurnState
	}{
		{
			name:         "deny from waiting approval",
			initialState: StateWaitingApproval,
			wantErr:      false,
			wantState:    StateCancelled,
		},
		{
			name:         "deny from running",
			initialState: StateRunning,
			wantErr:      true,
			wantState:    StateRunning,
		},
		{
			name:         "deny from pending",
			initialState: StatePending,
			wantErr:      true,
			wantState:    StatePending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{
				ID:        "test-turn",
				SessionID: "test-session",
				State:     tt.initialState,
			}

			err := turn.Deny()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Turn.Deny() expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Errorf("Turn.Deny() expected ErrInvalidTransition, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Turn.Deny() unexpected error: %v", err)
				}
				if turn.State != tt.wantState {
					t.Errorf("Turn.Deny() state = %v, want %v", turn.State, tt.wantState)
				}
				if turn.CompletedAt.IsZero() {
					t.Errorf("Turn.Deny() CompletedAt should be set")
				}
			}
		})
	}
}

func TestTurn_AddToolCall(t *testing.T) {
	turn := &Turn{
		ID:        "test-turn",
		SessionID: "test-session",
		State:     StateRunning,
	}

	call := ToolCall{
		ID:       "call-1",
		Name:     "test-tool",
		Args:     map[string]interface{}{"arg1": "value1"},
		CallTime: time.Now(),
	}

	turn.AddToolCall(call)

	if len(turn.ToolCalls) != 1 {
		t.Errorf("Turn.AddToolCall() expected 1 tool call, got %d", len(turn.ToolCalls))
	}

	if turn.ToolCalls[0].ID != call.ID {
		t.Errorf("Turn.AddToolCall() tool call ID = %v, want %v", turn.ToolCalls[0].ID, call.ID)
	}
}

func TestTurn_AddToolResult(t *testing.T) {
	turn := &Turn{
		ID:        "test-turn",
		SessionID: "test-session",
		State:     StateRunning,
	}

	result := ToolResult{
		ToolCallID: "call-1",
		Result:     "success",
		Duration:   time.Second,
	}

	turn.AddToolResult(result)

	if len(turn.ToolResults) != 1 {
		t.Errorf("Turn.AddToolResult() expected 1 tool result, got %d", len(turn.ToolResults))
	}

	if turn.ToolResults[0].ToolCallID != result.ToolCallID {
		t.Errorf("Turn.AddToolResult() tool result ToolCallID = %v, want %v", turn.ToolResults[0].ToolCallID, result.ToolCallID)
	}
}

func TestTurn_UpdateTokens(t *testing.T) {
	turn := &Turn{
		ID:        "test-turn",
		SessionID: "test-session",
		State:     StateRunning,
	}

	usage := TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	turn.UpdateTokens(usage)

	if turn.Tokens != usage {
		t.Errorf("Turn.UpdateTokens() tokens = %v, want %v", turn.Tokens, usage)
	}
}

func TestTurn_GetTotalTokens(t *testing.T) {
	turn := &Turn{
		ID:        "test-turn",
		SessionID: "test-session",
		State:     StateRunning,
		Tokens: TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	total := turn.GetTotalTokens()

	if total != 150 {
		t.Errorf("Turn.GetTotalTokens() = %d, want 150", total)
	}
}

func TestTurn_Concurrency(t *testing.T) {
	turn := &Turn{
		ID:        "test-turn",
		SessionID: "test-session",
		State:     StateRunning,
	}

	// Test concurrent access to AddToolCall
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			call := ToolCall{
				ID:       "call-" + string(rune(i)),
				Name:     "test-tool",
				Args:     map[string]interface{}{"arg": i},
				CallTime: time.Now(),
			}
			turn.AddToolCall(call)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	if len(turn.ToolCalls) != 10 {
		t.Errorf("Turn.AddToolCall() concurrent access expected 10 tool calls, got %d", len(turn.ToolCalls))
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from TurnState
		to   TurnState
		want bool
	}{
		{
			name: "pending to running",
			from: StatePending,
			to:   StateRunning,
			want: true,
		},
		{
			name: "running to completed",
			from: StateRunning,
			to:   StateCompleted,
			want: true,
		},
		{
			name: "running to failed",
			from: StateRunning,
			to:   StateFailed,
			want: true,
		},
		{
			name: "running to waiting approval",
			from: StateRunning,
			to:   StateWaitingApproval,
			want: true,
		},
		{
			name: "waiting approval to running",
			from: StateWaitingApproval,
			to:   StateRunning,
			want: true,
		},
		{
			name: "waiting approval to cancelled",
			from: StateWaitingApproval,
			to:   StateCancelled,
			want: true,
		},
		{
			name: "running to cancelled",
			from: StateRunning,
			to:   StateCancelled,
			want: true,
		},
		{
			name: "completed to running",
			from: StateCompleted,
			to:   StateRunning,
			want: false,
		},
		{
			name: "failed to running",
			from: StateFailed,
			to:   StateRunning,
			want: false,
		},
		{
			name: "cancelled to running",
			from: StateCancelled,
			to:   StateRunning,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanTransition(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CanTransition(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
