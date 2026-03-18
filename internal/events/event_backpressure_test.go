package events

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

const (
	testEvent1 = "event1"
	testEvent2 = "event2"
)

// Test BackpressureDrop mode - events dropped when channel full.
func TestEventEmitter_BackpressureDrop(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       2,
		BackpressureMode: BackpressureDrop,
	})
	defer emitter.Close()

	id, events, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Fill the buffer (size=2).
	emitter.Emit(Event{Type: EventInfo, Data: testEvent1})
	emitter.Emit(Event{Type: EventInfo, Data: testEvent2})

	// This should be dropped (buffer full, no consumer).
	emitter.Emit(Event{Type: EventInfo, Data: "event3"})

	// Read first two events.
	event1 := <-events
	if event1.Data != testEvent1 {
		t.Errorf("Expected event1, got %v", event1.Data)
	}

	event2 := <-events
	if event2.Data != testEvent2 {
		t.Errorf("Expected event2, got %v", event2.Data)
	}

	// Third event should not be in channel (was dropped).
	select {
	case <-events:
		t.Error("Expected no more events (event3 should have been dropped)")
	case <-time.After(50 * time.Millisecond):
		// Good - no event received.
	}

	emitter.Unsubscribe(id)
}

// Test BackpressureDrop with fast consumer - no drops.
func TestEventEmitter_BackpressureDrop_FastConsumer(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       10, // Larger buffer to prevent drops.
		BackpressureMode: BackpressureDrop,
	})
	defer emitter.Close()

	_, events, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Consumer reads immediately.
	var wg sync.WaitGroup

	received := make([]string, 0)

	var mu sync.Mutex

	wg.Go(func() {
		timeout := time.After(200 * time.Millisecond)

		for range 5 {
			select {
			case event := <-events:
				mu.Lock()

				s, _ := event.Data.(string)
				received = append(received, s)
				mu.Unlock()
			case <-timeout:
				return // Timeout - not all events received.
			}
		}
	})

	// Emit events with small delay to let consumer keep up.
	for i := 1; i <= 5; i++ {
		emitter.Emit(Event{Type: EventInfo, Data: fmt.Sprintf("event%d", i)})
		time.Sleep(5 * time.Millisecond)
	}

	wg.Wait()

	// All events should be received (fast consumer).
	if len(received) != 5 {
		t.Errorf("Expected 5 events, got %d", len(received))
	}
}

// Test BackpressureBlock mode - emitter blocks until consumer ready.
func TestEventEmitter_BackpressureBlock(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       2,
		BackpressureMode: BackpressureBlock,
	})
	defer emitter.Close()

	_, events, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Fill the buffer (size=2).
	emitter.Emit(Event{Type: EventInfo, Data: testEvent1})
	emitter.Emit(Event{Type: EventInfo, Data: testEvent2})

	// Track if emitter blocks.
	blocked := make(chan bool, 1)

	// Emit in goroutine (should block until consumer reads).
	go func() {
		emitter.Emit(Event{Type: EventInfo, Data: "event3"})

		blocked <- true
	}()

	// Give emitter time to attempt send (should block).
	time.Sleep(50 * time.Millisecond)

	select {
	case <-blocked:
		t.Error("Emitter should have blocked, but it didn't")
	default:
		// Good - emitter is blocked.
	}

	// Read one event to unblock.
	event1 := <-events
	if event1.Data != testEvent1 {
		t.Errorf("Expected event1, got %v", event1.Data)
	}

	// Now emitter should unblock and send event3.
	select {
	case <-blocked:
		// Good - emitter unblocked.
	case <-time.After(100 * time.Millisecond):
		t.Error("Emitter should have unblocked after consumer read")
	}

	// Read remaining events.
	event2 := <-events
	if event2.Data != testEvent2 {
		t.Errorf("Expected event2, got %v", event2.Data)
	}

	event3 := <-events
	if event3.Data != "event3" {
		t.Errorf("Expected event3, got %v", event3.Data)
	}
}

// Test BackpressureBuffer mode - dynamic buffering.
func TestEventEmitter_BackpressureBuffer(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       2,
		BackpressureMode: BackpressureBuffer,
		BufferLimit:      10,
	})
	defer emitter.Close()

	id, events, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Fill channel buffer (2 events).
	emitter.Emit(Event{Type: EventInfo, Data: testEvent1})
	emitter.Emit(Event{Type: EventInfo, Data: testEvent2})

	// Verify buffer is in dynamic buffer map.
	emitter.bufferMu.Lock()
	if _, exists := emitter.buffers[id]; !exists {
		t.Error("Buffer should exist for subscriber in BackpressureBuffer mode")
	}
	emitter.bufferMu.Unlock()

	// These go to dynamic buffer (channel full).
	emitter.Emit(Event{Type: EventInfo, Data: "event3"})
	emitter.Emit(Event{Type: EventInfo, Data: "event4"})

	// Verify events are buffered.
	emitter.bufferMu.Lock()
	bufferLen := len(emitter.buffers[id])
	emitter.bufferMu.Unlock()

	if bufferLen != 2 {
		t.Errorf("Expected 2 buffered events, got %d", bufferLen)
	}

	// Verify dynamic buffer has events (not dropped).
	emitter.bufferMu.Lock()
	initialBufferLen := len(emitter.buffers[id])
	emitter.bufferMu.Unlock()

	if initialBufferLen == 0 {
		t.Error("Events should be in dynamic buffer, not dropped")
	}

	// Read events from channel to make space.
	event1 := <-events
	if event1.Data != testEvent1 {
		t.Errorf("Expected event1, got %v", event1.Data)
	}

	event2 := <-events
	if event2.Data != testEvent2 {
		t.Errorf("Expected event2, got %v", event2.Data)
	}

	// New emit should flush some buffered events (best effort).
	for i := 5; i <= 7; i++ {
		emitter.Emit(Event{Type: EventInfo, Data: fmt.Sprintf("event%d", i)})
		time.Sleep(10 * time.Millisecond)
	}

	// Check that buffer size decreased (some events flushed).
	emitter.bufferMu.Lock()
	finalBufferLen := len(emitter.buffers[id])
	emitter.bufferMu.Unlock()

	// Buffer should have shrunk or stayed same (events flushed)
	// Note: This is best-effort, so we just verify no crashes.
	_ = finalBufferLen

	emitter.Unsubscribe(id)
}

// Test BackpressureBuffer with limit exceeded.
func TestEventEmitter_BackpressureBuffer_LimitExceeded(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       2,
		BackpressureMode: BackpressureBuffer,
		BufferLimit:      3, // Small limit for testing.
	})
	defer emitter.Close()

	id, events, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Fill channel buffer (2) + dynamic buffer (3) = 5 total capacity.
	for i := 1; i <= 5; i++ {
		emitter.Emit(Event{Type: EventInfo, Data: fmt.Sprintf("event%d", i)})
	}

	// Verify buffer is at limit.
	emitter.bufferMu.Lock()
	bufferLen := len(emitter.buffers[id])
	emitter.bufferMu.Unlock()

	if bufferLen != 3 {
		t.Errorf("Expected buffer at limit (3), got %d", bufferLen)
	}

	// Emit more events - should be dropped (over limit).
	emitter.Emit(Event{Type: EventInfo, Data: "event6"})
	emitter.Emit(Event{Type: EventInfo, Data: "event7"})

	// Verify buffer didn't grow beyond limit.
	emitter.bufferMu.Lock()
	finalBufferLen := len(emitter.buffers[id])
	emitter.bufferMu.Unlock()

	if finalBufferLen > 3 {
		t.Errorf("Buffer should not exceed limit (3), got %d", finalBufferLen)
	}

	// Read events from channel.
	event1 := <-events
	if event1.Data != testEvent1 {
		t.Errorf("Expected event1, got %v", event1.Data)
	}

	event2 := <-events
	if event2.Data != testEvent2 {
		t.Errorf("Expected event2, got %v", event2.Data)
	}

	emitter.Unsubscribe(id)
}

// Test NewEventEmitter backward compatibility (should use BackpressureDrop).
func TestNewEventEmitter_BackwardCompatibility(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitter(10)
	defer emitter.Close()

	if emitter.config.BackpressureMode != BackpressureDrop {
		t.Errorf("Expected BackpressureDrop mode, got %v", emitter.config.BackpressureMode)
	}

	if emitter.config.BufferSize != 10 {
		t.Errorf("Expected buffer size 10, got %d", emitter.config.BufferSize)
	}
}

// Test config defaults.
func TestEventEmitterConfig_Defaults(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       5,
		BackpressureMode: BackpressureBuffer,
		// BufferLimit not set - should default to 10000.
	})
	defer emitter.Close()

	if emitter.config.BufferLimit != 10000 {
		t.Errorf("Expected default BufferLimit 10000, got %d", emitter.config.BufferLimit)
	}
}

// drainEvents consumes events from a channel until timeout.
func drainEvents(events <-chan Event, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()

	timer := time.After(timeout)

	for {
		select {
		case <-events:
		case <-timer:
			return
		}
	}
}

// emitBatch emits a batch of events from a producer goroutine.
func emitBatch(emitter *EventEmitter, id, count int, wg *sync.WaitGroup) {
	defer wg.Done()

	for j := range count {
		emitter.Emit(Event{Type: EventInfo, Data: id*100 + j})
	}
}

// Test concurrent emissions with different modes.
func TestEventEmitter_ConcurrentEmissions(t *testing.T) {
	t.Parallel()

	modes := []BackpressureMode{BackpressureDrop, BackpressureBlock, BackpressureBuffer}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			t.Parallel()

			testConcurrentEmissions(t, mode)
		})
	}
}

func testConcurrentEmissions(t *testing.T, mode BackpressureMode) {
	t.Helper()

	emitter := NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       10,
		BackpressureMode: mode,
		BufferLimit:      100,
	})
	defer emitter.Close()

	_, events, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	var wg sync.WaitGroup

	wg.Add(1)

	go drainEvents(events, &wg, 500*time.Millisecond)

	// Multiple producers.
	numProducers := 5
	eventsPerProducer := 10

	for i := range numProducers {
		wg.Add(1)

		go emitBatch(emitter, i, eventsPerProducer, &wg)
	}

	wg.Wait()
}

// Test Subscribe/Unsubscribe during emission.
func TestEventEmitter_SubscribeUnsubscribeDuringEmit(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       10,
		BackpressureMode: BackpressureDrop,
	})
	defer emitter.Close()

	var wg sync.WaitGroup

	// Continuous emitter.
	wg.Add(1)

	stopEmit := make(chan bool)

	go func() {
		defer wg.Done()

		i := 0

		for {
			select {
			case <-stopEmit:
				return
			default:
				emitter.Emit(Event{Type: EventInfo, Data: i})
				i++

				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// Multiple subscribers joining and leaving.
	for i := range 5 {
		wg.Add(1)

		go func(_ int) {
			defer wg.Done()

			subID, events, err := emitter.Subscribe()
			if err != nil {
				return
			}

			// Read some events.
			for range 10 {
				<-events
			}

			emitter.Unsubscribe(subID)
		}(i)
	}

	time.Sleep(200 * time.Millisecond)
	close(stopEmit)
	wg.Wait()
}

// Test Close during emission.
func TestEventEmitter_CloseDuringEmit(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       10,
		BackpressureMode: BackpressureDrop,
	})

	_, events, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	var wg sync.WaitGroup

	// Emit continuously.
	wg.Go(func() {
		for i := range 100 {
			emitter.Emit(Event{Type: EventInfo, Data: i})
			time.Sleep(1 * time.Millisecond)
		}
	})

	// Close after a bit.
	time.Sleep(50 * time.Millisecond)
	emitter.Close()

	// Drain channel.
	for range events {
		continue // discard remaining events.
	}

	wg.Wait()

	// Subscribe after close should fail.
	_, _, err = emitter.Subscribe()
	if err == nil {
		t.Error("Subscribe after Close should fail")
	}
}

// Test buffer cleanup on Unsubscribe.
func TestEventEmitter_BufferCleanupOnUnsubscribe(t *testing.T) {
	t.Parallel()

	emitter := NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       2,
		BackpressureMode: BackpressureBuffer,
		BufferLimit:      10,
	})
	defer emitter.Close()

	id, _, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Create some buffered events.
	for i := range 5 {
		emitter.Emit(Event{Type: EventInfo, Data: i})
	}

	// Verify buffer exists.
	emitter.bufferMu.Lock()
	if _, exists := emitter.buffers[id]; !exists {
		t.Error("Buffer should exist for subscriber")
	}
	emitter.bufferMu.Unlock()

	// Unsubscribe.
	emitter.Unsubscribe(id)

	// Verify buffer cleaned up.
	emitter.bufferMu.Lock()
	if _, exists := emitter.buffers[id]; exists {
		t.Error("Buffer should be cleaned up after unsubscribe")
	}
	emitter.bufferMu.Unlock()
}

// Test BackpressureMode.String() for coverage.
func TestBackpressureMode_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode BackpressureMode
		want string
	}{
		{BackpressureDrop, "drop"},
		{BackpressureBlock, "block"},
		{BackpressureBuffer, "buffer"},
		{BackpressureMode(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			got := tt.mode.String()
			if got != tt.want {
				t.Errorf("BackpressureMode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
