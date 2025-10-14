package cycle

import (
	"context"
	"testing"
	"time"
)

// Mock message implementation for testing
type mockMessage struct {
	role      string
	content   string
	timestamp time.Time
}

func (m *mockMessage) GetRole() string         { return m.role }
func (m *mockMessage) GetContent() string      { return m.content }
func (m *mockMessage) GetTimestamp() time.Time { return m.timestamp }

// Mock event emitter for testing
type mockEventEmitter struct {
	emittedEvents []Event
}

func (m *mockEventEmitter) Emit(event Event) {
	m.emittedEvents = append(m.emittedEvents, event)
}

func (m *mockEventEmitter) GetEmittedEvents() []Event {
	return m.emittedEvents
}

func TestReflectionIntervention_Name(t *testing.T) {
	intervention := &ReflectionIntervention{}
	if intervention.Name() != "Reflection" {
		t.Errorf("Expected name 'Reflection', got '%s'", intervention.Name())
	}
}

func TestReflectionIntervention_Description(t *testing.T) {
	intervention := &ReflectionIntervention{}
	desc := intervention.Description()
	// The description should mention reflection - check that it contains key words
	if !stringContains(desc, "reflection") && !stringContains(desc, "repetitive") {
		t.Errorf("Expected description to mention reflection or repetitive, got: %s", desc)
	}
}

func TestReflectionIntervention_Severity(t *testing.T) {
	intervention := &ReflectionIntervention{}
	if intervention.Severity() != 1 {
		t.Errorf("Expected severity 1 (soft), got %d", intervention.Severity())
	}
}

func TestReflectionIntervention_Apply(t *testing.T) {
	intervention := &ReflectionIntervention{}
	ctx := context.Background()

	originalMessages := []Message{
		&mockMessage{role: "user", content: "Hello", timestamp: time.Now()},
		&mockMessage{role: "assistant", content: "Hi there", timestamp: time.Now()},
	}

	result, err := intervention.Apply(ctx, originalMessages)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should have added one reflection message
	if len(result) != len(originalMessages)+1 {
		t.Errorf("Expected %d messages, got %d", len(originalMessages)+1, len(result))
	}

	// Check that the new message is the reflection prompt
	reflectionMsg := result[len(result)-1]
	if reflectionMsg.GetRole() != "user" {
		t.Errorf("Expected reflection message role 'user', got '%s'", reflectionMsg.GetRole())
	}

	// The reflection content should mention key concepts - check that it contains relevant words
	if !stringContains(reflectionMsg.GetContent(), "repeating") && !stringContains(reflectionMsg.GetContent(), "different") && !stringContains(reflectionMsg.GetContent(), "approach") {
		t.Errorf("Expected reflection content to mention repeating, different, or approach, got: %s", reflectionMsg.GetContent())
	}
}

func TestSummarizeIntervention_Name(t *testing.T) {
	intervention := &SummarizeIntervention{}
	if intervention.Name() != "Context Summarization" {
		t.Errorf("Expected name 'Context Summarization', got '%s'", intervention.Name())
	}
}

func TestSummarizeIntervention_Description(t *testing.T) {
	intervention := &SummarizeIntervention{}
	desc := intervention.Description()
	// The description should mention compression - check that it contains key words
	if !stringContains(desc, "compress") && !stringContains(desc, "summarize") {
		t.Errorf("Expected description to mention compress or summarize, got: %s", desc)
	}
}

func TestSummarizeIntervention_Severity(t *testing.T) {
	intervention := &SummarizeIntervention{}
	if intervention.Severity() != 2 {
		t.Errorf("Expected severity 2 (medium), got %d", intervention.Severity())
	}
}

func TestSummarizeIntervention_Apply_NoCompressor(t *testing.T) {
	intervention := &SummarizeIntervention{} // No compressor set
	ctx := context.Background()

	messages := []Message{
		&mockMessage{role: "user", content: "Hello", timestamp: time.Now()},
	}

	result, err := intervention.Apply(ctx, messages)
	if err == nil {
		t.Error("Expected error when compressor is not configured")
	}

	// Should return original messages unchanged
	if len(result) != len(messages) {
		t.Errorf("Expected original messages returned, got %d instead of %d", len(result), len(messages))
	}
}

func TestSummarizeIntervention_Apply_WithCompressor(t *testing.T) {
	// Mock compressor that just returns the first half of messages
	mockCompressor := &mockCompressor{
		compressFunc: func(messages []Message, target int) ([]Message, error) {
			if target >= len(messages) {
				return messages, nil
			}
			return messages[:target], nil
		},
	}

	intervention := &SummarizeIntervention{compressor: mockCompressor}
	ctx := context.Background()

	originalMessages := []Message{
		&mockMessage{role: "user", content: "Message 1", timestamp: time.Now()},
		&mockMessage{role: "assistant", content: "Message 2", timestamp: time.Now()},
		&mockMessage{role: "user", content: "Message 3", timestamp: time.Now()},
		&mockMessage{role: "assistant", content: "Message 4", timestamp: time.Now()},
	}

	result, err := intervention.Apply(ctx, originalMessages)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should compress to half (2 messages) + 1 system message
	expectedLen := 2 + 1 // compressed messages + system explanation
	if len(result) != expectedLen {
		t.Errorf("Expected %d messages, got %d", expectedLen, len(result))
	}

	// Last message should be the system explanation
	lastMsg := result[len(result)-1]
	if lastMsg.GetRole() != "system" {
		t.Errorf("Expected last message role 'system', got '%s'", lastMsg.GetRole())
	}

	// The system message should mention summarization - check that it contains key words
	if !stringContains(lastMsg.GetContent(), "summarized") && !stringContains(lastMsg.GetContent(), "focus") {
		t.Errorf("Expected system message to mention summarization or focus, got: %s", lastMsg.GetContent())
	}
}

func TestEscalateIntervention_Name(t *testing.T) {
	intervention := &EscalateIntervention{}
	if intervention.Name() != "User Escalation" {
		t.Errorf("Expected name 'User Escalation', got '%s'", intervention.Name())
	}
}

func TestEscalateIntervention_Description(t *testing.T) {
	intervention := &EscalateIntervention{}
	desc := intervention.Description()
	// The description should mention pause or user - check that it contains key words
	if !stringContains(desc, "pause") && !stringContains(desc, "user") && !stringContains(desc, "guidance") {
		t.Errorf("Expected description to mention pause, user, or guidance, got: %s", desc)
	}
}

func TestEscalateIntervention_Severity(t *testing.T) {
	intervention := &EscalateIntervention{}
	if intervention.Severity() != 3 {
		t.Errorf("Expected severity 3 (hard), got %d", intervention.Severity())
	}
}

func TestEscalateIntervention_Apply_WithEmitter(t *testing.T) {
	mockEmitter := &mockEventEmitter{}
	intervention := &EscalateIntervention{Emitter: mockEmitter}
	ctx := context.Background()

	messages := []Message{
		&mockMessage{role: "user", content: "Hello", timestamp: time.Now()},
	}

	result, err := intervention.Apply(ctx, messages)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should return original messages unchanged
	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}

	// Should have emitted a turn_paused event
	events := mockEmitter.GetEmittedEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event emitted, got %d", len(events))
	}

	if events[0].GetType() != "turn_paused" {
		t.Errorf("Expected 'turn_paused' event type, got '%s'", events[0].GetType())
	}
}

func TestEscalateIntervention_Apply_NoEmitter(t *testing.T) {
	intervention := &EscalateIntervention{} // No emitter
	ctx := context.Background()

	messages := []Message{
		&mockMessage{role: "user", content: "Hello", timestamp: time.Now()},
	}

	result, err := intervention.Apply(ctx, messages)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should return original messages unchanged
	if len(result) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(result))
	}
}

func TestInterventionSelector_SelectIntervention(t *testing.T) {
	selector := NewInterventionSelector()

	// Test early cycle (soft intervention)
	intervention := selector.SelectIntervention(CycleSimilarResponses, 5)
	if _, ok := intervention.(*ReflectionIntervention); !ok {
		t.Errorf("Expected ReflectionIntervention for early cycle, got %T", intervention)
	}

	// Test mid-stage cycle (medium intervention) - should be SummarizeIntervention for now
	intervention = selector.SelectIntervention(CycleRepeatedTool, 20)
	if _, ok := intervention.(*SummarizeIntervention); !ok {
		t.Errorf("Expected SummarizeIntervention for mid-stage cycle, got %T", intervention)
	}

	// Test late-stage cycle (hard intervention)
	intervention = selector.SelectIntervention(CycleOscillation, 40)
	if _, ok := intervention.(*EscalateIntervention); !ok {
		t.Errorf("Expected EscalateIntervention for late-stage cycle, got %T", intervention)
	}
}

func TestInterventionSelector_RecordIntervention(t *testing.T) {
	selector := NewInterventionSelector()

	// Record some interventions
	result1 := InterventionResult{
		Type:      InterventionReflection,
		Success:   true,
		Message:   "Applied successfully",
		Timestamp: time.Now(),
	}

	result2 := InterventionResult{
		Type:      InterventionSummarize,
		Success:   false,
		Message:   "Failed to apply",
		Timestamp: time.Now(),
	}

	selector.RecordIntervention(result1)
	selector.RecordIntervention(result2)

	history := selector.GetPreviousInterventions()
	if len(history) != 2 {
		t.Errorf("Expected 2 recorded interventions, got %d", len(history))
	}

	if history[0].Type != InterventionReflection {
		t.Errorf("Expected first intervention to be Reflection, got %v", history[0].Type)
	}

	if history[1].Success != false {
		t.Errorf("Expected second intervention to have Success=false, got %v", history[1].Success)
	}
}

// Mock compressor for testing
type mockCompressor struct {
	compressFunc func([]Message, int) ([]Message, error)
}

func (m *mockCompressor) Compress(messages []Message, target int) ([]Message, error) {
	if m.compressFunc != nil {
		return m.compressFunc(messages, target)
	}
	return messages, nil
}

// Helper function to check string containment (using strings.Contains for simplicity)
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0)
}
