package events

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// TestEvent_Structure tests Event struct serialization
func TestEvent_Structure(t *testing.T) {
	event := Event{
		Type:      EventContentDelta,
		Timestamp: time.Now(),
		Data:      "test content",
	}

	// Can serialize to JSON
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	// Can deserialize
	var decoded Event
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if decoded.Type != EventContentDelta {
		t.Errorf("Type mismatch: got %v, want %v", decoded.Type, EventContentDelta)
	}
}

// TestEventType_String tests EventType String() method
func TestEventType_String(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventContentDelta, "content_delta"},
		{EventContentComplete, "content_complete"},
		{EventToolCallStart, "tool_call_start"},
		{EventToolCallProgress, "tool_call_progress"},
		{EventToolCallComplete, "tool_call_complete"},
		{EventTurnStart, "turn_start"},
		{EventTurnComplete, "turn_complete"},
		{EventTurnFailed, "turn_failed"},
		{EventCommandApproval, "command_approval"},
		{EventCommandApproved, "command_approved"},
		{EventCommandDenied, "command_denied"},
		{EventError, "error"},
		{EventWarning, "warning"},
		{EventInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.eventType.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestEventType_String_Unknown tests unknown event type
func TestEventType_String_Unknown(t *testing.T) {
	unknown := EventType(999)
	if got := unknown.String(); got != "unknown" {
		t.Errorf("String() = %v, want unknown", got)
	}
}

// TestNewEventEmitter tests emitter creation
func TestNewEventEmitter(t *testing.T) {
	emitter := NewEventEmitter(100)

	if emitter == nil {
		t.Fatal("NewEventEmitter() returned nil")
	}

	if emitter.subscribers == nil {
		t.Error("subscribers map should be initialized")
	}

	if emitter.bufferSize != 100 {
		t.Errorf("bufferSize = %d, want 100", emitter.bufferSize)
	}

	if emitter.closed {
		t.Error("new emitter should not be closed")
	}
}

// TestEventEmitter_Subscribe tests subscription
func TestEventEmitter_Subscribe(t *testing.T) {
	emitter := NewEventEmitter(10)

	id, events, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe() failed: %v", err)
	}

	if id == "" {
		t.Error("Subscribe() returned empty ID")
	}

	if events == nil {
		t.Fatal("Subscribe() returned nil channel")
	}

	// Channel should be buffered with correct size
	if cap(events) != 10 {
		t.Errorf("channel capacity = %d, want 10", cap(events))
	}
}

// TestEventEmitter_Subscribe_AfterClose tests subscription after close
func TestEventEmitter_Subscribe_AfterClose(t *testing.T) {
	emitter := NewEventEmitter(10)
	emitter.Close()

	_, _, err := emitter.Subscribe()
	if err == nil {
		t.Error("Subscribe() after Close() should return error")
	}
}

// TestEventEmitter_Subscribe_MultipleSubscribers tests multiple subscriptions
func TestEventEmitter_Subscribe_MultipleSubscribers(t *testing.T) {
	emitter := NewEventEmitter(10)

	id1, _, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("First Subscribe() failed: %v", err)
	}

	id2, _, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Second Subscribe() failed: %v", err)
	}

	// IDs should be unique
	if id1 == id2 {
		t.Error("Subscribe() returned duplicate IDs")
	}
}

// TestEventEmitter_Unsubscribe tests unsubscription
func TestEventEmitter_Unsubscribe(t *testing.T) {
	emitter := NewEventEmitter(10)

	id, events, _ := emitter.Subscribe()

	// Unsubscribe
	emitter.Unsubscribe(id)

	// Channel should be closed
	_, ok := <-events
	if ok {
		t.Error("channel should be closed after Unsubscribe()")
	}
}

// TestEventEmitter_Unsubscribe_NonExistent tests unsubscribing non-existent ID
func TestEventEmitter_Unsubscribe_NonExistent(t *testing.T) {
	emitter := NewEventEmitter(10)

	// Should not panic
	emitter.Unsubscribe("non-existent-id")
}

// TestEventEmitter_Emit tests event emission
func TestEventEmitter_Emit(t *testing.T) {
	emitter := NewEventEmitter(10)

	_, events, _ := emitter.Subscribe()

	// Emit event
	emitter.Emit(Event{
		Type: EventContentDelta,
		Data: "test",
	})

	// Subscriber should receive event
	select {
	case event := <-events:
		if event.Type != EventContentDelta {
			t.Errorf("event.Type = %v, want %v", event.Type, EventContentDelta)
		}
		if event.Data != "test" {
			t.Errorf("event.Data = %v, want test", event.Data)
		}
		if event.Timestamp.IsZero() {
			t.Error("event.Timestamp should be set")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

// TestEventEmitter_Emit_WithTimestamp tests emission with pre-set timestamp
func TestEventEmitter_Emit_WithTimestamp(t *testing.T) {
	emitter := NewEventEmitter(10)

	_, events, _ := emitter.Subscribe()

	customTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	emitter.Emit(Event{
		Type:      EventInfo,
		Timestamp: customTime,
	})

	event := <-events
	if !event.Timestamp.Equal(customTime) {
		t.Errorf("timestamp should be preserved: got %v, want %v", event.Timestamp, customTime)
	}
}

// TestEventEmitter_Emit_MultipleSubscribers tests broadcast to multiple subscribers
func TestEventEmitter_Emit_MultipleSubscribers(t *testing.T) {
	emitter := NewEventEmitter(10)

	_, events1, _ := emitter.Subscribe()
	_, events2, _ := emitter.Subscribe()

	// Emit event
	emitter.Emit(Event{Type: EventInfo, Data: "broadcast"})

	// Both should receive
	event1 := <-events1
	event2 := <-events2

	if event1.Type != EventInfo {
		t.Errorf("subscriber 1: Type = %v, want %v", event1.Type, EventInfo)
	}
	if event2.Type != EventInfo {
		t.Errorf("subscriber 2: Type = %v, want %v", event2.Type, EventInfo)
	}
}

// TestEventEmitter_Emit_SlowSubscriber tests fire-and-forget with slow subscriber
func TestEventEmitter_Emit_SlowSubscriber(t *testing.T) {
	emitter := NewEventEmitter(2) // Small buffer

	_, events, _ := emitter.Subscribe()

	// Fill the buffer
	emitter.Emit(Event{Type: EventInfo, Data: "1"})
	emitter.Emit(Event{Type: EventInfo, Data: "2"})

	// This should not block (fire-and-forget)
	done := make(chan bool)
	go func() {
		emitter.Emit(Event{Type: EventInfo, Data: "3"})
		done <- true
	}()

	select {
	case <-done:
		// Success - didn't block
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Emit() blocked on slow subscriber")
	}

	// Drain events
	<-events
	<-events
}

// TestEventEmitter_Close tests emitter closure
func TestEventEmitter_Close(t *testing.T) {
	emitter := NewEventEmitter(10)

	_, events1, _ := emitter.Subscribe()
	_, events2, _ := emitter.Subscribe()

	// Close emitter
	emitter.Close()

	// All channels should be closed
	_, ok1 := <-events1
	_, ok2 := <-events2

	if ok1 {
		t.Error("events1 channel should be closed")
	}
	if ok2 {
		t.Error("events2 channel should be closed")
	}

	// Cannot subscribe after close
	_, _, err := emitter.Subscribe()
	if err == nil {
		t.Error("Subscribe() after Close() should return error")
	}
}

// TestEventEmitter_Close_Idempotent tests multiple closes
func TestEventEmitter_Close_Idempotent(t *testing.T) {
	emitter := NewEventEmitter(10)

	// Should not panic
	emitter.Close()
	emitter.Close()
	emitter.Close()
}

// TestEventEmitter_EventOrdering tests event order preservation
func TestEventEmitter_EventOrdering(t *testing.T) {
	emitter := NewEventEmitter(100)

	_, events, _ := emitter.Subscribe()

	// Emit sequence
	for i := 0; i < 10; i++ {
		emitter.Emit(Event{
			Type: EventInfo,
			Data: i,
		})
	}

	// Receive in order
	for i := 0; i < 10; i++ {
		event := <-events
		if event.Data != i {
			t.Errorf("event %d: Data = %v, want %d", i, event.Data, i)
		}
	}
}

// TestEventEmitter_ConcurrentSubscribe tests concurrent subscriptions
func TestEventEmitter_ConcurrentSubscribe(t *testing.T) {
	emitter := NewEventEmitter(10)

	var wg sync.WaitGroup
	subscribers := 100

	for i := 0; i < subscribers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := emitter.Subscribe()
			if err != nil {
				t.Errorf("Subscribe() failed: %v", err)
			}
		}()
	}

	wg.Wait()

	// Should have all subscribers
	emitter.mu.RLock()
	count := len(emitter.subscribers)
	emitter.mu.RUnlock()

	if count != subscribers {
		t.Errorf("subscriber count = %d, want %d", count, subscribers)
	}
}

// TestEventEmitter_ConcurrentEmit tests concurrent emissions
func TestEventEmitter_ConcurrentEmit(t *testing.T) {
	emitter := NewEventEmitter(100)

	_, events, _ := emitter.Subscribe()

	var wg sync.WaitGroup
	emitCount := 100

	for i := 0; i < emitCount; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			emitter.Emit(Event{
				Type: EventInfo,
				Data: n,
			})
		}(i)
	}

	wg.Wait()

	// Collect all events
	received := 0
	timeout := time.After(1 * time.Second)
	for received < emitCount {
		select {
		case <-events:
			received++
		case <-timeout:
			t.Fatalf("only received %d/%d events", received, emitCount)
		}
	}
}

// TestEventEmitter_ConcurrentMixed tests mixed concurrent operations
func TestEventEmitter_ConcurrentMixed(t *testing.T) {
	emitter := NewEventEmitter(50)

	var wg sync.WaitGroup

	// Concurrent subscribes and unsubscribes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, _, err := emitter.Subscribe()
			if err == nil {
				time.Sleep(10 * time.Millisecond)
				emitter.Unsubscribe(id)
			}
		}()
	}

	// Concurrent emits
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			emitter.Emit(Event{Type: EventInfo})
		}()
	}

	wg.Wait()
}

// TestEventEmitter_NoMemoryLeaks tests cleanup
func TestEventEmitter_NoMemoryLeaks(t *testing.T) {
	emitter := NewEventEmitter(10)

	// Create and cleanup many subscribers
	for i := 0; i < 1000; i++ {
		id, _, _ := emitter.Subscribe()
		emitter.Unsubscribe(id)
	}

	emitter.mu.RLock()
	count := len(emitter.subscribers)
	emitter.mu.RUnlock()

	// Should be empty
	if count != 0 {
		t.Errorf("subscriber count = %d, want 0 (memory leak)", count)
	}
}

// BenchmarkEventEmitter_Emit benchmarks single emission
func BenchmarkEventEmitter_Emit(b *testing.B) {
	emitter := NewEventEmitter(100)
	emitter.Subscribe() // One subscriber

	event := Event{Type: EventInfo, Data: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		emitter.Emit(event)
	}
}

// BenchmarkEventEmitter_EmitMultipleSubscribers benchmarks with multiple subscribers
func BenchmarkEventEmitter_EmitMultipleSubscribers(b *testing.B) {
	emitter := NewEventEmitter(100)

	// 10 subscribers
	for i := 0; i < 10; i++ {
		emitter.Subscribe()
	}

	event := Event{Type: EventInfo, Data: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		emitter.Emit(event)
	}
}

// BenchmarkEventEmitter_Subscribe benchmarks subscription
func BenchmarkEventEmitter_Subscribe(b *testing.B) {
	emitter := NewEventEmitter(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		emitter.Subscribe()
	}
}
