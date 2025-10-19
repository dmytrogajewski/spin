package status

import (
	"testing"
	"time"
)

func TestManager_GetStatus(t *testing.T) {
	m := NewManager()
	status := m.GetStatus()

	if status.Text != "" {
		t.Errorf("Expected empty status text, got %q", status.Text)
	}

	if status.Metrics.SessionStart.IsZero() {
		t.Error("Expected session start to be set")
	}
}

func TestManager_SetStatus(t *testing.T) {
	m := NewManager()
	m.SetStatus("Processing...")

	status := m.GetStatus()
	if status.Text != "Processing..." {
		t.Errorf("Expected status 'Processing...', got %q", status.Text)
	}

	if status.Metrics.LastUpdate.IsZero() {
		t.Error("Expected last update to be set")
	}
}

func TestManager_SetProvider(t *testing.T) {
	m := NewManager()
	m.SetProvider("openai", "gpt-4")

	metrics := m.GetMetrics()
	if metrics.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got %q", metrics.Provider)
	}
	if metrics.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got %q", metrics.Model)
	}
	if !metrics.Connected {
		t.Error("Expected connected to be true")
	}
}

func TestManager_IncrementTurn(t *testing.T) {
	m := NewManager()

	// Initial turn count should be 0
	metrics := m.GetMetrics()
	if metrics.TurnCount != 0 {
		t.Errorf("Expected initial turn count 0, got %d", metrics.TurnCount)
	}

	// Increment twice
	m.IncrementTurn()
	m.IncrementTurn()

	metrics = m.GetMetrics()
	if metrics.TurnCount != 2 {
		t.Errorf("Expected turn count 2, got %d", metrics.TurnCount)
	}
}

func TestManager_AddTokens(t *testing.T) {
	m := NewManager()
	m.SetMaxTokens(1000) // Set max tokens first

	// Add tokens
	m.AddTokens(100, 50) // 150 total tokens

	metrics := m.GetMetrics()
	if metrics.TokenCount != 150 {
		t.Errorf("Expected token count 150, got %d", metrics.TokenCount)
	}
	if metrics.TokenUsage != 15.0 { // 150/1000 * 100
		t.Errorf("Expected token usage 15.0%%, got %.1f%%", metrics.TokenUsage)
	}

	// Add more tokens
	m.AddTokens(200, 100) // 300 more tokens (450 total)

	metrics = m.GetMetrics()
	if metrics.TokenCount != 450 {
		t.Errorf("Expected token count 450, got %d", metrics.TokenCount)
	}
	if metrics.TokenUsage != 45.0 { // 450/1000 * 100
		t.Errorf("Expected token usage 45.0%%, got %.1f%%", metrics.TokenUsage)
	}
}

func TestManager_SetResponseTime(t *testing.T) {
	m := NewManager()

	duration := 2 * time.Second
	tokens := int64(100)
	m.SetResponseTime(duration, tokens)

	metrics := m.GetMetrics()
	if metrics.ResponseTime != duration {
		t.Errorf("Expected response time %v, got %v", duration, metrics.ResponseTime)
	}
	if metrics.TokensPerSec != 50.0 { // 100 tokens / 2 seconds
		t.Errorf("Expected tokens per sec 50.0, got %.1f", metrics.TokensPerSec)
	}
}

func TestManager_EnableDisable(t *testing.T) {
	m := NewManager()

	if !m.IsEnabled() {
		t.Error("Expected manager to be enabled by default")
	}

	m.Disable()
	if m.IsEnabled() {
		t.Error("Expected manager to be disabled")
	}

	m.Enable()
	if !m.IsEnabled() {
		t.Error("Expected manager to be enabled")
	}
}

func TestManager_Reset(t *testing.T) {
	m := NewManager()

	// Set some data
	m.SetStatus("Test")
	m.SetProvider("test", "model")
	m.IncrementTurn()
	m.AddTokens(100, 50)

	// Reset
	m.Reset()

	status := m.GetStatus()
	if status.Text != "" {
		t.Errorf("Expected empty status text after reset, got %q", status.Text)
	}

	metrics := status.Metrics
	if metrics.TurnCount != 0 {
		t.Errorf("Expected turn count 0 after reset, got %d", metrics.TurnCount)
	}
	if metrics.TokenCount != 0 {
		t.Errorf("Expected token count 0 after reset, got %d", metrics.TokenCount)
	}
	if metrics.Provider != "" {
		t.Errorf("Expected empty provider after reset, got %q", metrics.Provider)
	}
	if metrics.SessionStart.IsZero() {
		t.Error("Expected session start to be set after reset")
	}
}

func TestManager_Concurrency(t *testing.T) {
	m := NewManager()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				m.SetStatus("Test")
				m.IncrementTurn()
				m.AddTokens(10, 5)
				_ = m.GetStatus()
				_ = m.GetMetrics()
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	metrics := m.GetMetrics()
	expectedTurns := 10 * 100 // 10 goroutines * 100 iterations
	if metrics.TurnCount != expectedTurns {
		t.Errorf("Expected turn count %d, got %d", expectedTurns, metrics.TurnCount)
	}
}

func TestManager_SetAgentState(t *testing.T) {
	m := NewManager()

	m.SetAgentState("Calling tools")

	metrics := m.GetMetrics()
	if metrics.AgentState != "Calling tools" {
		t.Errorf("Expected agent state 'Calling tools', got '%s'", metrics.AgentState)
	}
}

func TestManager_SetTaskMode(t *testing.T) {
	m := NewManager()

	m.SetTaskMode("review")

	metrics := m.GetMetrics()
	if metrics.TaskMode != "review" {
		t.Errorf("Expected task mode 'review', got '%s'", metrics.TaskMode)
	}
}

func TestManager_SetConversationID(t *testing.T) {
	m := NewManager()

	testID := "abc123def456"
	m.SetConversationID(testID)

	metrics := m.GetMetrics()
	if metrics.ConversationID != testID {
		t.Errorf("Expected conversation ID '%s', got '%s'", testID, metrics.ConversationID)
	}
}

func TestManager_CalculateTPS(t *testing.T) {
	m := NewManager()

	// 100 tokens in 2 seconds = 50 tok/s
	m.CalculateTPS(100, 2*time.Second)

	metrics := m.GetMetrics()
	expected := 50.0
	if metrics.TokensPerSec != expected {
		t.Errorf("Expected TPS %.1f, got %.1f", expected, metrics.TokensPerSec)
	}
}

func TestManager_CalculateTPS_ZeroDuration(t *testing.T) {
	m := NewManager()

	// Zero duration should not cause panic
	m.CalculateTPS(100, 0)

	metrics := m.GetMetrics()
	if metrics.TokensPerSec != 0 {
		t.Errorf("Expected TPS 0 for zero duration, got %.1f", metrics.TokensPerSec)
	}
}

func TestManager_SetConnected(t *testing.T) {
	m := NewManager()

	// Test setting connected to true
	m.SetConnected(true)
	metrics := m.GetMetrics()
	if !metrics.Connected {
		t.Error("Expected connected to be true")
	}

	// Test setting connected to false
	m.SetConnected(false)
	metrics = m.GetMetrics()
	if metrics.Connected {
		t.Error("Expected connected to be false")
	}
}
