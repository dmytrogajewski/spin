package cycle

import (
	"context"
	"testing"
	"time"
)

func TestReflectionIntervention_Apply(t *testing.T) {
	intervention := &ReflectionIntervention{}

	messages := []Message{
		&messageImpl{
			role:      "user",
			content:   "Help me with this task",
			timestamp: time.Now(),
		},
		&messageImpl{
			role:      "assistant",
			content:   "I'll help you with that",
			timestamp: time.Now(),
		},
	}

	result, err := intervention.Apply(context.Background(), messages)

	if err != nil {
		t.Errorf("ReflectionIntervention.Apply() unexpected error: %v", err)
	}

	if len(result) != len(messages)+1 {
		t.Errorf("ReflectionIntervention.Apply() result length = %d, want %d", len(result), len(messages)+1)
	}

	// Check that the reflection message was added
	reflectionMsg := result[len(result)-1]
	if reflectionMsg.GetRole() != "user" {
		t.Errorf("ReflectionIntervention.Apply() reflection message role = %s, want 'user'", reflectionMsg.GetRole())
	}

	if reflectionMsg.GetContent() == "" {
		t.Errorf("ReflectionIntervention.Apply() reflection message content should not be empty")
	}
}

func TestReflectionIntervention_Name(t *testing.T) {
	intervention := &ReflectionIntervention{}

	name := intervention.Name()

	if name != "Reflection" {
		t.Errorf("ReflectionIntervention.Name() = %s, want 'Reflection'", name)
	}
}

func TestReflectionIntervention_Description(t *testing.T) {
	intervention := &ReflectionIntervention{}

	description := intervention.Description()

	if description == "" {
		t.Errorf("ReflectionIntervention.Description() should not be empty")
	}

	if description != "Injects a reflection prompt to help the agent recognize repetitive patterns and consider alternative approaches" {
		t.Errorf("ReflectionIntervention.Description() = %s, want expected description", description)
	}
}

func TestReflectionIntervention_Severity(t *testing.T) {
	intervention := &ReflectionIntervention{}

	severity := intervention.Severity()

	if severity != 1 {
		t.Errorf("ReflectionIntervention.Severity() = %d, want 1", severity)
	}
}

func TestEscalateIntervention_Apply(t *testing.T) {
	// Create a mock event emitter
	mockEmitter := &mockEventEmitter{}

	intervention := &EscalateIntervention{
		Emitter: mockEmitter,
	}

	messages := []Message{
		&messageImpl{
			role:      "user",
			content:   "Help me with this task",
			timestamp: time.Now(),
		},
	}

	result, err := intervention.Apply(context.Background(), messages)

	if err != nil {
		t.Errorf("EscalateIntervention.Apply() unexpected error: %v", err)
	}

	if len(result) != len(messages) {
		t.Errorf("EscalateIntervention.Apply() result length = %d, want %d", len(result), len(messages))
	}

	// Check that an event was emitted
	if len(mockEmitter.events) != 1 {
		t.Errorf("EscalateIntervention.Apply() expected 1 event, got %d", len(mockEmitter.events))
	}

	event := mockEmitter.events[0]
	if event.GetType() != "turn_paused" {
		t.Errorf("EscalateIntervention.Apply() event type = %s, want 'turn_paused'", event.GetType())
	}
}

func TestEscalateIntervention_Apply_NoEmitter(t *testing.T) {
	intervention := &EscalateIntervention{
		Emitter: nil, // No emitter
	}

	messages := []Message{
		&messageImpl{
			role:      "user",
			content:   "Help me with this task",
			timestamp: time.Now(),
		},
	}

	result, err := intervention.Apply(context.Background(), messages)

	if err != nil {
		t.Errorf("EscalateIntervention.Apply() unexpected error: %v", err)
	}

	if len(result) != len(messages) {
		t.Errorf("EscalateIntervention.Apply() result length = %d, want %d", len(result), len(messages))
	}
}

func TestEscalateIntervention_Name(t *testing.T) {
	intervention := &EscalateIntervention{}

	name := intervention.Name()

	if name != "User Escalation" {
		t.Errorf("EscalateIntervention.Name() = %s, want 'User Escalation'", name)
	}
}

func TestEscalateIntervention_Description(t *testing.T) {
	intervention := &EscalateIntervention{}

	description := intervention.Description()

	if description == "" {
		t.Errorf("EscalateIntervention.Description() should not be empty")
	}

	if description != "Pauses agent execution and requests user guidance when automated interventions fail to break the cycle" {
		t.Errorf("EscalateIntervention.Description() = %s, want expected description", description)
	}
}

func TestEscalateIntervention_Severity(t *testing.T) {
	intervention := &EscalateIntervention{}

	severity := intervention.Severity()

	if severity != 3 {
		t.Errorf("EscalateIntervention.Severity() = %d, want 3", severity)
	}
}

func TestMessageImpl(t *testing.T) {
	msg := &messageImpl{
		role:      "user",
		content:   "test content",
		timestamp: time.Now(),
	}

	if msg.GetRole() != "user" {
		t.Errorf("messageImpl.GetRole() = %s, want 'user'", msg.GetRole())
	}

	if msg.GetContent() != "test content" {
		t.Errorf("messageImpl.GetContent() = %s, want 'test content'", msg.GetContent())
	}

	if msg.GetTimestamp().IsZero() {
		t.Errorf("messageImpl.GetTimestamp() should not be zero")
	}
}

func TestEventImpl(t *testing.T) {
	data := map[string]interface{}{
		"key": "value",
	}

	event := &eventImpl{
		eventType: "test_event",
		timestamp: time.Now(),
		data:      data,
	}

	if event.GetType() != "test_event" {
		t.Errorf("eventImpl.GetType() = %s, want 'test_event'", event.GetType())
	}

	if event.GetTimestamp().IsZero() {
		t.Errorf("eventImpl.GetTimestamp() should not be zero")
	}

	eventData := event.GetData()
	if eventData == nil {
		t.Errorf("eventImpl.GetData() = nil, want non-nil")
	}
}

// mockEventEmitter is a test implementation of EventEmitter
type mockEventEmitter struct {
	events []Event
}

func (m *mockEventEmitter) Emit(event Event) {
	m.events = append(m.events, event)
}
