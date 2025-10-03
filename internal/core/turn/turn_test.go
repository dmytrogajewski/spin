package turn

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewTurn(t *testing.T) {
	sessionID := "session-123"
	userInput := "test input"

	turn := NewTurn(sessionID, userInput)

	if turn.ID == "" {
		t.Error("NewTurn() ID should not be empty")
	}
	if turn.SessionID != sessionID {
		t.Errorf("NewTurn() SessionID = %v, want %v", turn.SessionID, sessionID)
	}
	if turn.UserInput != userInput {
		t.Errorf("NewTurn() UserInput = %v, want %v", turn.UserInput, userInput)
	}
	if turn.State != StatePending {
		t.Errorf("NewTurn() State = %v, want %v", turn.State, StatePending)
	}
	if turn.ToolCalls == nil {
		t.Error("NewTurn() ToolCalls should be initialized")
	}
	if turn.ToolResults == nil {
		t.Error("NewTurn() ToolResults should be initialized")
	}
	if turn.Metadata == nil {
		t.Error("NewTurn() Metadata should be initialized")
	}
}

func TestTurn_Start(t *testing.T) {
	tests := []struct {
		name      string
		initState TurnState
		wantErr   bool
	}{
		{"Start from Pending", StatePending, false},
		{"Start from Running", StateRunning, true},
		{"Start from Completed", StateCompleted, true},
		{"Start from Failed", StateFailed, true},
		{"Start from Cancelled", StateCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{State: tt.initState}
			err := turn.Start()

			if (err != nil) != tt.wantErr {
				t.Errorf("Turn.Start() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if turn.State != StateRunning {
					t.Errorf("Turn.Start() State = %v, want %v", turn.State, StateRunning)
				}
				if turn.StartedAt.IsZero() {
					t.Error("Turn.Start() StartedAt should be set")
				}
			}
		})
	}
}

func TestTurn_Complete(t *testing.T) {
	tests := []struct {
		name      string
		initState TurnState
		wantErr   bool
	}{
		{"Complete from Running", StateRunning, false},
		{"Complete from Pending", StatePending, true},
		{"Complete from WaitingApproval", StateWaitingApproval, true},
		{"Complete from Completed", StateCompleted, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{State: tt.initState}
			response := "test response"
			tokens := TokenUsage{
				PromptTokens:     50,
				CompletionTokens: 30,
				TotalTokens:      80,
			}

			err := turn.Complete(response, tokens)

			if (err != nil) != tt.wantErr {
				t.Errorf("Turn.Complete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if turn.State != StateCompleted {
					t.Errorf("Turn.Complete() State = %v, want %v", turn.State, StateCompleted)
				}
				if turn.AIResponse != response {
					t.Errorf("Turn.Complete() AIResponse = %v, want %v", turn.AIResponse, response)
				}
				if turn.Tokens.TotalTokens != 80 {
					t.Errorf("Turn.Complete() Tokens.TotalTokens = %v, want 80", turn.Tokens.TotalTokens)
				}
				if turn.CompletedAt.IsZero() {
					t.Error("Turn.Complete() CompletedAt should be set")
				}
			}
		})
	}
}

func TestTurn_Fail(t *testing.T) {
	tests := []struct {
		name      string
		initState TurnState
		wantErr   bool
	}{
		{"Fail from Running", StateRunning, false},
		{"Fail from Pending", StatePending, true},
		{"Fail from Completed", StateCompleted, true},
	}

	testError := errors.New("test error")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{State: tt.initState}

			err := turn.Fail(testError)

			if (err != nil) != tt.wantErr {
				t.Errorf("Turn.Fail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if turn.State != StateFailed {
					t.Errorf("Turn.Fail() State = %v, want %v", turn.State, StateFailed)
				}
				if turn.Error == nil {
					t.Error("Turn.Fail() Error should be set")
				}
				if !errors.Is(turn.Error, testError) {
					t.Errorf("Turn.Fail() Error = %v, want %v", turn.Error, testError)
				}
				if turn.CompletedAt.IsZero() {
					t.Error("Turn.Fail() CompletedAt should be set")
				}
			}
		})
	}
}

func TestTurn_Cancel(t *testing.T) {
	tests := []struct {
		name      string
		initState TurnState
		wantErr   bool
	}{
		{"Cancel from Running", StateRunning, false},
		{"Cancel from WaitingApproval", StateWaitingApproval, false},
		{"Cancel from Pending", StatePending, true},
		{"Cancel from Completed", StateCompleted, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{State: tt.initState}

			err := turn.Cancel()

			if (err != nil) != tt.wantErr {
				t.Errorf("Turn.Cancel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if turn.State != StateCancelled {
					t.Errorf("Turn.Cancel() State = %v, want %v", turn.State, StateCancelled)
				}
				if turn.CompletedAt.IsZero() {
					t.Error("Turn.Cancel() CompletedAt should be set")
				}
			}
		})
	}
}

func TestTurn_RequestApproval(t *testing.T) {
	tests := []struct {
		name      string
		initState TurnState
		wantErr   bool
	}{
		{"RequestApproval from Running", StateRunning, false},
		{"RequestApproval from Pending", StatePending, true},
		{"RequestApproval from Completed", StateCompleted, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{State: tt.initState}

			err := turn.RequestApproval()

			if (err != nil) != tt.wantErr {
				t.Errorf("Turn.RequestApproval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if turn.State != StateWaitingApproval {
					t.Errorf("Turn.RequestApproval() State = %v, want %v", turn.State, StateWaitingApproval)
				}
			}
		})
	}
}

func TestTurn_Approve(t *testing.T) {
	tests := []struct {
		name      string
		initState TurnState
		wantErr   bool
	}{
		{"Approve from WaitingApproval", StateWaitingApproval, false},
		{"Approve from Running", StateRunning, true},
		{"Approve from Pending", StatePending, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{State: tt.initState}

			err := turn.Approve()

			if (err != nil) != tt.wantErr {
				t.Errorf("Turn.Approve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if turn.State != StateRunning {
					t.Errorf("Turn.Approve() State = %v, want %v", turn.State, StateRunning)
				}
			}
		})
	}
}

func TestTurn_Deny(t *testing.T) {
	tests := []struct {
		name      string
		initState TurnState
		wantErr   bool
	}{
		{"Deny from WaitingApproval", StateWaitingApproval, false},
		{"Deny from Running", StateRunning, true},
		{"Deny from Pending", StatePending, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &Turn{State: tt.initState}

			err := turn.Deny()

			if (err != nil) != tt.wantErr {
				t.Errorf("Turn.Deny() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if turn.State != StateCancelled {
					t.Errorf("Turn.Deny() State = %v, want %v", turn.State, StateCancelled)
				}
				if turn.CompletedAt.IsZero() {
					t.Error("Turn.Deny() CompletedAt should be set")
				}
			}
		})
	}
}

func TestTurn_AddToolCall(t *testing.T) {
	turn := NewTurn("session-123", "test")

	toolCall := ToolCall{
		ID:       "call-1",
		Name:     "shell",
		Args:     map[string]interface{}{"command": "ls"},
		CallTime: time.Now(),
	}

	turn.AddToolCall(toolCall)

	if len(turn.ToolCalls) != 1 {
		t.Errorf("Turn.AddToolCall() len(ToolCalls) = %v, want 1", len(turn.ToolCalls))
	}
	if turn.ToolCalls[0].ID != "call-1" {
		t.Errorf("Turn.AddToolCall() ToolCalls[0].ID = %v, want call-1", turn.ToolCalls[0].ID)
	}
}

func TestTurn_AddToolResult(t *testing.T) {
	turn := NewTurn("session-123", "test")

	result := ToolResult{
		ToolCallID: "call-1",
		Result:     "output",
		Duration:   10 * time.Millisecond,
	}

	turn.AddToolResult(result)

	if len(turn.ToolResults) != 1 {
		t.Errorf("Turn.AddToolResult() len(ToolResults) = %v, want 1", len(turn.ToolResults))
	}
	if turn.ToolResults[0].ToolCallID != "call-1" {
		t.Errorf("Turn.AddToolResult() ToolResults[0].ToolCallID = %v, want call-1", turn.ToolResults[0].ToolCallID)
	}
}

func TestTurn_UpdateTokens(t *testing.T) {
	turn := NewTurn("session-123", "test")

	tokens := TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	turn.UpdateTokens(tokens)

	if turn.Tokens.TotalTokens != 150 {
		t.Errorf("Turn.UpdateTokens() Tokens.TotalTokens = %v, want 150", turn.Tokens.TotalTokens)
	}
}

func TestTurn_GetTotalTokens(t *testing.T) {
	turn := NewTurn("session-123", "test")
	turn.Tokens = TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	total := turn.GetTotalTokens()

	if total != 150 {
		t.Errorf("Turn.GetTotalTokens() = %v, want 150", total)
	}
}

func TestTurn_Lifecycle(t *testing.T) {
	// Create new turn
	turn := NewTurn("session-123", "test input")
	if turn.State != StatePending {
		t.Errorf("Initial state = %v, want %v", turn.State, StatePending)
	}

	// Start turn
	err := turn.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if turn.State != StateRunning {
		t.Errorf("After Start() state = %v, want %v", turn.State, StateRunning)
	}
	if turn.StartedAt.IsZero() {
		t.Error("After Start() StartedAt should be set")
	}

	// Add tool call
	turn.AddToolCall(ToolCall{
		ID:   "call-1",
		Name: "shell",
		Args: map[string]interface{}{"command": "ls"},
	})

	// Add tool result
	turn.AddToolResult(ToolResult{
		ToolCallID: "call-1",
		Result:     "file1.txt\nfile2.txt",
		Duration:   10 * time.Millisecond,
	})

	// Complete turn
	tokens := TokenUsage{
		PromptTokens:     50,
		CompletionTokens: 30,
		TotalTokens:      80,
	}
	err = turn.Complete("I listed the files.", tokens)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if turn.State != StateCompleted {
		t.Errorf("After Complete() state = %v, want %v", turn.State, StateCompleted)
	}
	if turn.AIResponse != "I listed the files." {
		t.Errorf("After Complete() AIResponse = %v, want 'I listed the files.'", turn.AIResponse)
	}
	if turn.CompletedAt.IsZero() {
		t.Error("After Complete() CompletedAt should be set")
	}
}

func TestTurn_ApprovalWorkflow(t *testing.T) {
	// Create and start turn
	turn := NewTurn("session-123", "delete files")
	_ = turn.Start()

	// Request approval
	err := turn.RequestApproval()
	if err != nil {
		t.Fatalf("RequestApproval() error = %v", err)
	}
	if turn.State != StateWaitingApproval {
		t.Errorf("After RequestApproval() state = %v, want %v", turn.State, StateWaitingApproval)
	}

	// Approve
	err = turn.Approve()
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if turn.State != StateRunning {
		t.Errorf("After Approve() state = %v, want %v", turn.State, StateRunning)
	}

	// Complete
	err = turn.Complete("Done", TokenUsage{TotalTokens: 50})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if turn.State != StateCompleted {
		t.Errorf("After Complete() state = %v, want %v", turn.State, StateCompleted)
	}
}

func TestTurn_DenyWorkflow(t *testing.T) {
	// Create and start turn
	turn := NewTurn("session-123", "dangerous operation")
	_ = turn.Start()

	// Request approval
	err := turn.RequestApproval()
	if err != nil {
		t.Fatalf("RequestApproval() error = %v", err)
	}

	// Deny
	err = turn.Deny()
	if err != nil {
		t.Fatalf("Deny() error = %v", err)
	}
	if turn.State != StateCancelled {
		t.Errorf("After Deny() state = %v, want %v", turn.State, StateCancelled)
	}
	if turn.CompletedAt.IsZero() {
		t.Error("After Deny() CompletedAt should be set")
	}
}

func TestTurn_ConcurrentToolCalls(t *testing.T) {
	turn := NewTurn("session-123", "test")
	_ = turn.Start()

	const numCalls = 100
	var wg sync.WaitGroup
	wg.Add(numCalls)

	for i := 0; i < numCalls; i++ {
		go func(id int) {
			defer wg.Done()
			turn.AddToolCall(ToolCall{
				ID:   string(rune(id)),
				Name: "test",
			})
		}(i)
	}

	wg.Wait()

	if len(turn.ToolCalls) != numCalls {
		t.Errorf("ConcurrentToolCalls len(ToolCalls) = %v, want %v", len(turn.ToolCalls), numCalls)
	}
}

func TestTurn_ConcurrentToolResults(t *testing.T) {
	turn := NewTurn("session-123", "test")
	_ = turn.Start()

	const numResults = 100
	var wg sync.WaitGroup
	wg.Add(numResults)

	for i := 0; i < numResults; i++ {
		go func(id int) {
			defer wg.Done()
			turn.AddToolResult(ToolResult{
				ToolCallID: string(rune(id)),
				Result:     "result",
			})
		}(i)
	}

	wg.Wait()

	if len(turn.ToolResults) != numResults {
		t.Errorf("ConcurrentToolResults len(ToolResults) = %v, want %v", len(turn.ToolResults), numResults)
	}
}

func TestTurn_Serialization(t *testing.T) {
	original := NewTurn("session-123", "test input")
	_ = original.Start()
	original.AddToolCall(ToolCall{
		ID:   "call-1",
		Name: "shell",
		Args: map[string]interface{}{"cmd": "ls"},
	})
	original.UpdateTokens(TokenUsage{
		PromptTokens:     50,
		CompletionTokens: 30,
		TotalTokens:      80,
	})
	_ = original.Complete("response", original.Tokens)

	// Serialize
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Deserialize
	var deserialized Turn
	err = json.Unmarshal(data, &deserialized)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify
	if deserialized.ID != original.ID {
		t.Errorf("Deserialized ID = %v, want %v", deserialized.ID, original.ID)
	}
	if deserialized.SessionID != original.SessionID {
		t.Errorf("Deserialized SessionID = %v, want %v", deserialized.SessionID, original.SessionID)
	}
	if deserialized.State != original.State {
		t.Errorf("Deserialized State = %v, want %v", deserialized.State, original.State)
	}
	if deserialized.UserInput != original.UserInput {
		t.Errorf("Deserialized UserInput = %v, want %v", deserialized.UserInput, original.UserInput)
	}
	if deserialized.AIResponse != original.AIResponse {
		t.Errorf("Deserialized AIResponse = %v, want %v", deserialized.AIResponse, original.AIResponse)
	}
	if deserialized.Tokens.TotalTokens != original.Tokens.TotalTokens {
		t.Errorf("Deserialized Tokens.TotalTokens = %v, want %v", deserialized.Tokens.TotalTokens, original.Tokens.TotalTokens)
	}
	if len(deserialized.ToolCalls) != len(original.ToolCalls) {
		t.Errorf("Deserialized len(ToolCalls) = %v, want %v", len(deserialized.ToolCalls), len(original.ToolCalls))
	}
}
