package events

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

const (
	testCallID1     = "call_1"
	testUnknownType = "unknown"
)

// TestEvent_Structure tests Event struct serialization.
func TestEvent_Structure(t *testing.T) {
	t.Parallel()

	event := Event{
		Type:      EventContentDelta,
		Timestamp: time.Now(),
		Data:      "test content",
	}

	// Can serialize to JSON.
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	// Can deserialize.
	var decoded Event

	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if decoded.Type != EventContentDelta {
		t.Errorf("Type mismatch: got %v, want %v", decoded.Type, EventContentDelta)
	}
}

// TestEventType_String tests EventType String() method.
func TestEventType_String(t *testing.T) {
	t.Parallel()

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
		{EventACERetrieval, "ace_retrieval"},
		{EventCommandApproved, "command_approved"},
		{EventCommandDenied, "command_denied"},
		{EventError, "error"},
		{EventWarning, "warning"},
		{EventInfo, "info"},
		{EventCompactionTriggered, "compaction_triggered"},
		{EventDoomLoopDetected, "doom_loop_detected"},
		{EventReminderInjected, "reminder_injected"},
		{EventSubagentSpawn, "subagent_spawn"},
		{EventSubagentComplete, "subagent_complete"},
		{EventPhaseThinking, "phase_thinking"},
		{EventPhaseCritique, "phase_critique"},
		{EventUndoRecorded, "undo_recorded"},
		{EventBackgroundTaskStarted, "background_task_started"},
		{EventBackgroundTaskStopped, "background_task_stopped"},
		{EventSnapshotTaken, "snapshot_taken"},
		{EventSessionIndexRebuilt, "session_index_rebuilt"},
		{EventLSPDiagnostics, "lsp_diagnostics"},
		{EventHookVeto, "hook_veto"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			if got := tt.eventType.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestEventType_String_Unknown tests unknown event type.
func TestEventType_String_Unknown(t *testing.T) {
	t.Parallel()

	unknown := EventType(999)
	if got := unknown.String(); got != testUnknownType {
		t.Errorf("String() = %v, want unknown", got)
	}
}

// TestNewEventEmitter tests emitter creation.
func TestNewEventEmitter(t *testing.T) {
	t.Parallel()

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

// TestEventEmitter_Subscribe tests subscription.
func TestEventEmitter_Subscribe(t *testing.T) {
	t.Parallel()

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

	// Channel should be buffered with correct size.
	if cap(events) != 10 {
		t.Errorf("channel capacity = %d, want 10", cap(events))
	}
}

// TestEventEmitter_Subscribe_AfterClose tests subscription after close.
func TestEventEmitter_Subscribe_AfterClose(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)
	emitter.Close()

	_, _, err := emitter.Subscribe()
	if err == nil {
		t.Error("Subscribe() after Close() should return error")
	}
}

// TestEventEmitter_Subscribe_MultipleSubscribers tests multiple subscriptions.
func TestEventEmitter_Subscribe_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)

	id1, _, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("First Subscribe() failed: %v", err)
	}

	id2, _, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Second Subscribe() failed: %v", err)
	}

	// IDs should be unique.
	if id1 == id2 {
		t.Error("Subscribe() returned duplicate IDs")
	}
}

// TestEventEmitter_Unsubscribe tests unsubscription.
func TestEventEmitter_Unsubscribe(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)

	id, events, _ := emitter.Subscribe()

	// Unsubscribe.
	emitter.Unsubscribe(id)

	// Channel should be closed.
	_, ok := <-events
	if ok {
		t.Error("channel should be closed after Unsubscribe()")
	}
}

// TestEventEmitter_Unsubscribe_NonExistent tests unsubscribing non-existent ID.
func TestEventEmitter_Unsubscribe_NonExistent(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)

	// Should not panic.
	emitter.Unsubscribe("non-existent-id")
}

// TestEventEmitter_Emit tests event emission.
func TestEventEmitter_Emit(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)

	_, events, _ := emitter.Subscribe()

	// Emit event.
	emitter.Emit(Event{
		Type: EventContentDelta,
		Data: "test",
	})

	// Subscriber should receive event.
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

// TestEventEmitter_Emit_WithTimestamp tests emission with pre-set timestamp.
func TestEventEmitter_Emit_WithTimestamp(t *testing.T) {
	t.Parallel()

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

// TestEventEmitter_Emit_MultipleSubscribers tests broadcast to multiple subscribers.
func TestEventEmitter_Emit_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)

	_, events1, _ := emitter.Subscribe()
	_, events2, _ := emitter.Subscribe()

	// Emit event.
	emitter.Emit(Event{Type: EventInfo, Data: "broadcast"})

	// Both should receive.
	event1 := <-events1
	event2 := <-events2

	if event1.Type != EventInfo {
		t.Errorf("subscriber 1: Type = %v, want %v", event1.Type, EventInfo)
	}

	if event2.Type != EventInfo {
		t.Errorf("subscriber 2: Type = %v, want %v", event2.Type, EventInfo)
	}
}

// TestEventEmitter_Emit_SlowSubscriber tests fire-and-forget with slow subscriber.
func TestEventEmitter_Emit_SlowSubscriber(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(2) // Small buffer.

	_, events, _ := emitter.Subscribe()

	// Fill the buffer.
	emitter.Emit(Event{Type: EventInfo, Data: "1"})
	emitter.Emit(Event{Type: EventInfo, Data: "2"})

	// This should not block (fire-and-forget).
	done := make(chan bool)

	go func() {
		emitter.Emit(Event{Type: EventInfo, Data: "3"})

		done <- true
	}()

	select {
	case <-done:
		// Success - didn't block.
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Emit() blocked on slow subscriber")
	}

	// Drain events.
	<-events
	<-events
}

// TestEventEmitter_Close tests emitter closure.
func TestEventEmitter_Close(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)

	_, events1, _ := emitter.Subscribe()
	_, events2, _ := emitter.Subscribe()

	// Close emitter.
	emitter.Close()

	// All channels should be closed.
	_, ok1 := <-events1
	_, ok2 := <-events2

	if ok1 {
		t.Error("events1 channel should be closed")
	}

	if ok2 {
		t.Error("events2 channel should be closed")
	}

	// Cannot subscribe after close.
	_, _, err := emitter.Subscribe()
	if err == nil {
		t.Error("Subscribe() after Close() should return error")
	}
}

// TestEventEmitter_Close_Idempotent tests multiple closes.
func TestEventEmitter_Close_Idempotent(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)

	// Should not panic.
	emitter.Close()
	emitter.Close()
	emitter.Close()
}

// TestEventEmitter_EventOrdering tests event order preservation.
func TestEventEmitter_EventOrdering(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(100)

	_, events, _ := emitter.Subscribe()

	// Emit sequence.
	for i := range 10 {
		emitter.Emit(Event{
			Type: EventInfo,
			Data: i,
		})
	}

	// Receive in order.
	for i := range 10 {
		event := <-events
		if event.Data != i {
			t.Errorf("event %d: Data = %v, want %d", i, event.Data, i)
		}
	}
}

// TestEventEmitter_ConcurrentSubscribe tests concurrent subscriptions.
func TestEventEmitter_ConcurrentSubscribe(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)

	var wg sync.WaitGroup

	subscribers := 100

	for range subscribers {
		wg.Go(func() {
			_, _, err := emitter.Subscribe()
			if err != nil {
				t.Errorf("Subscribe() failed: %v", err)
			}
		})
	}

	wg.Wait()

	// Should have all subscribers.
	emitter.mu.RLock()
	count := len(emitter.subscribers)
	emitter.mu.RUnlock()

	if count != subscribers {
		t.Errorf("subscriber count = %d, want %d", count, subscribers)
	}
}

// TestEventEmitter_ConcurrentEmit tests concurrent emissions.
func TestEventEmitter_ConcurrentEmit(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(100)

	_, events, _ := emitter.Subscribe()

	var wg sync.WaitGroup

	emitCount := 100

	for i := range emitCount {
		wg.Go(func() {
			emitter.Emit(Event{
				Type: EventInfo,
				Data: i,
			})
		})
	}

	wg.Wait()

	// Collect all events.
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

// TestEventEmitter_ConcurrentMixed tests mixed concurrent operations.
func TestEventEmitter_ConcurrentMixed(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(50)

	var wg sync.WaitGroup

	// Concurrent subscribes and unsubscribes.
	for range 10 {
		wg.Go(func() {
			id, _, err := emitter.Subscribe()
			if err == nil {
				time.Sleep(10 * time.Millisecond)
				emitter.Unsubscribe(id)
			}
		})
	}

	// Concurrent emits.
	for range 50 {
		wg.Go(func() {
			emitter.Emit(Event{Type: EventInfo})
		})
	}

	wg.Wait()
}

// TestEventEmitter_NoMemoryLeaks tests cleanup.
func TestEventEmitter_NoMemoryLeaks(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)

	// Create and cleanup many subscribers.
	for range 1000 {
		id, _, _ := emitter.Subscribe()
		emitter.Unsubscribe(id)
	}

	emitter.mu.RLock()
	count := len(emitter.subscribers)
	emitter.mu.RUnlock()

	// Should be empty.
	if count != 0 {
		t.Errorf("subscriber count = %d, want 0 (memory leak)", count)
	}
}

// BenchmarkEventEmitter_Emit benchmarks single emission.
func BenchmarkEventEmitter_Emit(b *testing.B) {
	emitter := NewEventEmitter(100)
	_, _, subErr := emitter.Subscribe() // One subscriber.
	_ = subErr

	event := Event{Type: EventInfo, Data: "test"}

	b.ResetTimer()

	for range b.N {
		emitter.Emit(event)
	}
}

// BenchmarkEventEmitter_EmitMultipleSubscribers benchmarks with multiple subscribers.
func BenchmarkEventEmitter_EmitMultipleSubscribers(b *testing.B) {
	emitter := NewEventEmitter(100)

	// 10 subscribers.
	for range 10 {
		_, _, _ = emitter.Subscribe()
	}

	event := Event{Type: EventInfo, Data: "test"}

	b.ResetTimer()

	for range b.N {
		emitter.Emit(event)
	}
}

// BenchmarkEventEmitter_Subscribe benchmarks subscription.
func BenchmarkEventEmitter_Subscribe(b *testing.B) {
	emitter := NewEventEmitter(100)

	b.ResetTimer()

	for range b.N {
		_, _, _ = emitter.Subscribe()
	}
}

// TestEvent_ToolCallStartData tests type-safe helper for ToolCallStartData.
func TestEvent_ToolCallStartData(t *testing.T) {
	t.Parallel()

	t.Run("valid data", func(t *testing.T) {
		t.Parallel()

		// Create event with ToolCallStartData.
		event := Event{
			Type: EventToolCallStart,
			Data: ToolCallStartData{
				ToolName: "read_file",
				ToolID:   testCallID1,
			},
		}

		// Should successfully extract data.
		data, ok := event.ToolCallStartData()
		if !ok {
			t.Fatal("ToolCallStartData() returned false for valid data")
		}

		if data.ToolName != "read_file" {
			t.Errorf("ToolName = %q, want %q", data.ToolName, "read_file")
		}

		if data.ToolID != testCallID1 {
			t.Errorf("ToolID = %q, want %q", data.ToolID, testCallID1)
		}
	})

	t.Run("wrong data type", func(t *testing.T) {
		t.Parallel()

		// Create event with wrong data type.
		event := Event{
			Type: EventContentDelta,
			Data: ContentDeltaData{Content: "test"},
		}

		// Should return false.
		_, ok := event.ToolCallStartData()
		if ok {
			t.Error("ToolCallStartData() returned true for wrong data type")
		}
	})
}

// TestEvent_TypeSafeHelpers tests all type-safe helper methods.
func TestEvent_TypeSafeHelpers(t *testing.T) {
	t.Parallel()

	t.Run("ToolCallCompleteData", func(t *testing.T) {
		t.Parallel()

		e := Event{Type: EventToolCallComplete, Data: ToolCallCompleteData{ToolID: testCallID1, Success: true}}
		data, ok := e.ToolCallCompleteData()
		assertTypeSafeHelper(t, ok, data.ToolID == testCallID1 && data.Success)
	})

	t.Run("ToolProgressData", func(t *testing.T) {
		t.Parallel()

		e := Event{Type: EventToolCallProgress, Data: ToolProgressData{ToolID: testCallID1, Status: "running"}}
		data, ok := e.ToolProgressData()
		assertTypeSafeHelper(t, ok, data.ToolID == testCallID1 && data.Status == "running")
	})

	t.Run("ContentDeltaData", func(t *testing.T) {
		t.Parallel()

		e := Event{Type: EventContentDelta, Data: ContentDeltaData{Content: "test", Role: "assistant"}}
		data, ok := e.ContentDeltaData()
		assertTypeSafeHelper(t, ok, data.Content == "test" && data.Role == "assistant")
	})

	t.Run("TurnEventData", func(t *testing.T) {
		t.Parallel()

		e := Event{Type: EventTurnStart, Data: TurnEventData{Turn: 5, TurnID: "turn_5"}}
		data, ok := e.TurnEventData()
		assertTypeSafeHelper(t, ok, data.Turn == 5 && data.TurnID == "turn_5")
	})

	t.Run("ApprovalEventData", func(t *testing.T) {
		t.Parallel()

		e := Event{Type: EventCommandApproval, Data: ApprovalEventData{RequestID: "req_1", Command: "rm -rf /"}}
		data, ok := e.ApprovalEventData()
		assertTypeSafeHelper(t, ok, data.RequestID == "req_1" && data.Command == "rm -rf /")
	})

	t.Run("SystemEventData", func(t *testing.T) {
		t.Parallel()

		e := Event{Type: EventWarning, Data: SystemEventData{Level: "warning", Message: "test warning"}}
		data, ok := e.SystemEventData()
		assertTypeSafeHelper(t, ok, data.Level == "warning" && data.Message == "test warning")
	})

	t.Run("ErrorData", func(t *testing.T) {
		t.Parallel()

		e := Event{Type: EventError, Data: ErrorData{Message: "test error", Code: "ERR_TEST"}}
		data, ok := e.ErrorData()
		assertTypeSafeHelper(t, ok, data.Message == "test error" && data.Code == "ERR_TEST")
	})
}

func assertTypeSafeHelper(t *testing.T, ok, fieldsMatch bool) {
	t.Helper()

	if !ok {
		t.Error("type assertion returned false")
	}

	if !fieldsMatch {
		t.Error("extracted fields do not match expected values")
	}
}

// TestEvent_TypeSafeHelpers_WrongType tests helpers return false for wrong types.
func TestEvent_TypeSafeHelpers_WrongType(t *testing.T) {
	t.Parallel()

	// Create event with ContentDeltaData.
	event := Event{
		Type: EventContentDelta,
		Data: ContentDeltaData{Content: "test"},
	}

	// All other helpers should return false.
	if _, ok := event.ToolCallCompleteData(); ok {
		t.Error("ToolCallCompleteData() should return false")
	}

	if _, ok := event.ToolProgressData(); ok {
		t.Error("ToolProgressData() should return false")
	}

	if _, ok := event.TurnEventData(); ok {
		t.Error("TurnEventData() should return false")
	}

	if _, ok := event.ApprovalEventData(); ok {
		t.Error("ApprovalEventData() should return false")
	}

	if _, ok := event.SystemEventData(); ok {
		t.Error("SystemEventData() should return false")
	}

	if _, ok := event.ErrorData(); ok {
		t.Error("ErrorData() should return false")
	}
}

// TestEvent_ACERetrievalData tests ACERetrievalData type assertion.
func TestEvent_ACERetrievalData(t *testing.T) {
	t.Parallel()

	aceData := ACERetrievalData{
		Turn:             5,
		Trigger:          "error",
		Query:            "install nodejs Error: command not found",
		BulletsRetrieved: 3,
		BulletsNew:       1,
		CacheSize:        10,
		CacheHitRate:     0.67,
	}

	event := Event{
		Type: EventACERetrieval,
		Data: aceData,
	}

	// Should successfully extract ACERetrievalData.
	data, ok := event.ACERetrievalData()
	if !ok {
		t.Fatal("ACERetrievalData() should return true")
	}

	if data.Turn != 5 {
		t.Errorf("Turn = %d, want 5", data.Turn)
	}

	if data.Trigger != "error" {
		t.Errorf("Trigger = %q, want \"error\"", data.Trigger)
	}

	if data.Query != "install nodejs Error: command not found" {
		t.Errorf("Query = %q, want \"install nodejs Error: command not found\"", data.Query)
	}

	if data.BulletsRetrieved != 3 {
		t.Errorf("BulletsRetrieved = %d, want 3", data.BulletsRetrieved)
	}

	if data.BulletsNew != 1 {
		t.Errorf("BulletsNew = %d, want 1", data.BulletsNew)
	}

	if data.CacheSize != 10 {
		t.Errorf("CacheSize = %d, want 10", data.CacheSize)
	}

	if data.CacheHitRate != 0.67 {
		t.Errorf("CacheHitRate = %f, want 0.67", data.CacheHitRate)
	}

	// Other type assertions should return false.
	if _, toolOk := event.ToolCallStartData(); toolOk {
		t.Error("ToolCallStartData() should return false for ACE event")
	}

	if _, errOk := event.ErrorData(); errOk {
		t.Error("ErrorData() should return false for ACE event")
	}
}

// Journey: specs/journeys/JOURNEY-7.6.md.

// TestEvent_HarnessPhaseHelpers tests type-safe helpers for harness phase events.
func TestEvent_HarnessPhaseHelpers(t *testing.T) {
	t.Parallel()

	t.Run("CompactionTriggeredData", func(t *testing.T) {
		t.Parallel()

		e := Event{
			Type: EventCompactionTriggered,
			Data: CompactionTriggeredData{Turn: 3, Stage: "compacted"},
		}
		data, ok := e.CompactionTriggeredData()
		assertTypeSafeHelper(t, ok, data.Turn == 3 && data.Stage == "compacted")
	})

	t.Run("DoomLoopDetectedData", func(t *testing.T) {
		t.Parallel()

		e := Event{
			Type: EventDoomLoopDetected,
			Data: DoomLoopDetectedData{
				Turn:        5,
				Fingerprint: "abc123",
				Count:       3,
				ToolName:    "shell_command",
			},
		}
		data, ok := e.DoomLoopDetectedData()
		assertTypeSafeHelper(t, ok,
			data.Turn == 5 &&
				data.Fingerprint == "abc123" &&
				data.Count == 3 &&
				data.ToolName == "shell_command")
	})

	t.Run("ReminderInjectedData", func(t *testing.T) {
		t.Parallel()

		e := Event{
			Type: EventReminderInjected,
			Data: ReminderInjectedData{Turn: 2, Count: 1},
		}
		data, ok := e.ReminderInjectedData()
		assertTypeSafeHelper(t, ok, data.Turn == 2 && data.Count == 1)
	})

	t.Run("SubagentSpawnData", func(t *testing.T) {
		t.Parallel()

		e := Event{
			Type: EventSubagentSpawn,
			Data: SubagentSpawnData{
				AgentType: "research",
				Query:     "find API docs",
			},
		}
		data, ok := e.SubagentSpawnData()
		assertTypeSafeHelper(t, ok,
			data.AgentType == "research" && data.Query == "find API docs")
	})

	t.Run("SubagentCompleteData", func(t *testing.T) {
		t.Parallel()

		e := Event{
			Type: EventSubagentComplete,
			Data: SubagentCompleteData{
				AgentType:    "research",
				Summary:      "found docs",
				InputTokens:  100,
				OutputTokens: 50,
			},
		}
		data, ok := e.SubagentCompleteData()
		assertTypeSafeHelper(t, ok,
			data.AgentType == "research" &&
				data.Summary == "found docs" &&
				data.InputTokens == 100 &&
				data.OutputTokens == 50)
	})

	t.Run("PhaseThinkingData", func(t *testing.T) {
		t.Parallel()

		e := Event{
			Type: EventPhaseThinking,
			Data: PhaseThinkingData{Turn: 1, Status: "started"},
		}
		data, ok := e.PhaseThinkingData()
		assertTypeSafeHelper(t, ok, data.Turn == 1 && data.Status == "started")
	})

	t.Run("PhaseCritiqueData", func(t *testing.T) {
		t.Parallel()

		e := Event{
			Type: EventPhaseCritique,
			Data: PhaseCritiqueData{Turn: 2, Status: "completed"},
		}
		data, ok := e.PhaseCritiqueData()
		assertTypeSafeHelper(t, ok, data.Turn == 2 && data.Status == "completed")
	})
}

// TestEvent_HarnessPhaseHelpers_WrongType tests helpers return false for wrong types.
func TestEvent_HarnessPhaseHelpers_WrongType(t *testing.T) {
	t.Parallel()

	event := Event{
		Type: EventContentDelta,
		Data: ContentDeltaData{Content: "test"},
	}

	if _, ok := event.CompactionTriggeredData(); ok {
		t.Error("CompactionTriggeredData() should return false")
	}

	if _, ok := event.DoomLoopDetectedData(); ok {
		t.Error("DoomLoopDetectedData() should return false")
	}

	if _, ok := event.ReminderInjectedData(); ok {
		t.Error("ReminderInjectedData() should return false")
	}

	if _, ok := event.SubagentSpawnData(); ok {
		t.Error("SubagentSpawnData() should return false")
	}

	if _, ok := event.SubagentCompleteData(); ok {
		t.Error("SubagentCompleteData() should return false")
	}

	if _, ok := event.PhaseThinkingData(); ok {
		t.Error("PhaseThinkingData() should return false")
	}

	if _, ok := event.PhaseCritiqueData(); ok {
		t.Error("PhaseCritiqueData() should return false")
	}
}
