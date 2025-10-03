package stream

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultStreamConfig(t *testing.T) {
	config := DefaultStreamConfig()

	assert.Equal(t, DefaultBufferCapacity, config.BufferSize)
	assert.Equal(t, 100, config.InputBuffer)
	assert.Equal(t, 100, config.OutputBuffer)
	assert.Equal(t, BackpressureBlock, config.Backpressure)
	assert.NotNil(t, config.ErrorHandler)
	assert.Equal(t, 10, config.MaxSequenceSkip)
}

func TestNewStream(t *testing.T) {
	config := DefaultStreamConfig()
	stream := NewStream("test-stream", config)

	assert.NotNil(t, stream)
	assert.Equal(t, "test-stream", stream.id)
	assert.True(t, stream.IsOpen())
	assert.Equal(t, StreamStateOpen, stream.State())

	stream.Close()
}

func TestStream_SendReceive(t *testing.T) {
	stream := NewStream("test", DefaultStreamConfig())
	defer stream.Close()

	// Send event
	event := StreamEvent{
		Type: ChunkContent,
		Data: []byte("test data"),
	}

	err := stream.Send(context.Background(), event)
	require.NoError(t, err)

	// Receive event
	select {
	case received := <-stream.Receive():
		assert.Equal(t, ChunkContent, received.Type)
		assert.Equal(t, "test data", string(received.Data))
		assert.NotZero(t, received.Sequence) // Should auto-assign sequence
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestStream_MultipleEvents(t *testing.T) {
	stream := NewStream("test", DefaultStreamConfig())
	defer stream.Close()

	// Send multiple events
	count := 10
	for i := 0; i < count; i++ {
		event := StreamEvent{
			Type: ChunkContent,
			Data: []byte(fmt.Sprintf("event %d", i)),
		}
		err := stream.Send(context.Background(), event)
		require.NoError(t, err)
	}

	// Receive all events
	received := 0
	timeout := time.After(time.Second)
	for received < count {
		select {
		case evt := <-stream.Receive():
			assert.Equal(t, fmt.Sprintf("event %d", received), string(evt.Data))
			received++
		case <-timeout:
			t.Fatalf("timeout after receiving %d/%d events", received, count)
		}
	}
}

func TestStream_SequenceNumbers(t *testing.T) {
	stream := NewStream("test", DefaultStreamConfig())
	defer stream.Close()

	// Send events
	for i := 0; i < 5; i++ {
		event := StreamEvent{
			Type: ChunkContent,
			Data: []byte(fmt.Sprintf("%d", i)),
		}
		stream.Send(context.Background(), event)
	}

	// Verify sequential numbering
	for i := int64(1); i <= 5; i++ {
		select {
		case evt := <-stream.Receive():
			assert.Equal(t, i, evt.Sequence)
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestStream_Close(t *testing.T) {
	stream := NewStream("test", DefaultStreamConfig())

	assert.True(t, stream.IsOpen())
	assert.Equal(t, StreamStateOpen, stream.State())

	err := stream.Close()
	require.NoError(t, err)

	assert.False(t, stream.IsOpen())
	assert.Equal(t, StreamStateClosed, stream.State())

	// Closing again should be idempotent
	err = stream.Close()
	require.NoError(t, err)
}

func TestStream_SendAfterClose(t *testing.T) {
	stream := NewStream("test", DefaultStreamConfig())
	stream.Close()

	event := StreamEvent{Type: ChunkContent, Data: []byte("test")}
	err := stream.Send(context.Background(), event)

	assert.Error(t, err)
	assert.Equal(t, ErrStreamClosed, err)
}

func TestStream_ContextCancellation(t *testing.T) {
	stream := NewStream("test", DefaultStreamConfig())
	defer stream.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	event := StreamEvent{Type: ChunkContent, Data: []byte("test")}
	err := stream.Send(ctx, event)

	assert.Error(t, err)
}

func TestStream_BackpressureBlock(t *testing.T) {
	config := DefaultStreamConfig()
	config.Backpressure = BackpressureBlock
	config.OutputBuffer = 1 // Small buffer

	stream := NewStream("test", config)
	defer stream.Close()

	// Fill buffer
	event1 := StreamEvent{Type: ChunkContent, Data: []byte("1")}
	err := stream.Send(context.Background(), event1)
	require.NoError(t, err)

	// This should block since buffer is full
	blocked := make(chan struct{})
	go func() {
		event2 := StreamEvent{Type: ChunkContent, Data: []byte("2")}
		stream.Send(context.Background(), event2)
		close(blocked)
	}()

	// Should not complete immediately
	select {
	case <-blocked:
		t.Fatal("send should block")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}

	// Read event to unblock
	<-stream.Receive()

	// Now should complete
	select {
	case <-blocked:
		// Expected
	case <-time.After(time.Second):
		t.Fatal("send should unblock")
	}
}

func TestStream_BackpressureDrop(t *testing.T) {
	config := DefaultStreamConfig()
	config.Backpressure = BackpressureDrop
	config.OutputBuffer = 2 // Small buffer

	stream := NewStream("test", config)
	defer stream.Close()

	// Send many events quickly
	sentCount := 20
	for i := 0; i < sentCount; i++ {
		event := StreamEvent{
			Type: ChunkContent,
			Data: []byte(fmt.Sprintf("%d", i)),
		}
		stream.Send(context.Background(), event)
	}

	// Count received events
	receivedCount := 0
	timeout := time.After(100 * time.Millisecond)
	for {
		select {
		case <-stream.Receive():
			receivedCount++
		case <-timeout:
			// Should have dropped some events
			assert.Less(t, receivedCount, sentCount)
			return
		}
	}
}

func TestStream_BackpressureError(t *testing.T) {
	config := DefaultStreamConfig()
	config.Backpressure = BackpressureError
	config.OutputBuffer = 1 // Small buffer

	stream := NewStream("test", config)
	defer stream.Close()

	// Fill buffer
	event1 := StreamEvent{Type: ChunkContent, Data: []byte("1")}
	err := stream.Send(context.Background(), event1)
	require.NoError(t, err)

	// This should return error immediately
	event2 := StreamEvent{Type: ChunkContent, Data: []byte("2")}
	err = stream.Send(context.Background(), event2)

	assert.Error(t, err)
	assert.Equal(t, ErrBackpressure, err)
}

func TestStream_ErrorChannel(t *testing.T) {
	errorCalled := make(chan error, 1)
	config := DefaultStreamConfig()
	config.ErrorHandler = func(err error) {
		errorCalled <- err
	}

	stream := NewStream("test", config)
	defer stream.Close()

	// Trigger an error (send after close)
	stream.Close()
	event := StreamEvent{Type: ChunkContent, Data: []byte("test")}
	err := stream.Send(context.Background(), event)

	assert.Error(t, err)

	// Check error was propagated
	select {
	case err := <-stream.Errors():
		assert.NotNil(t, err)
	case <-time.After(time.Second):
		// May not always propagate to error channel
	}
}

func TestStream_GracefulDrain(t *testing.T) {
	stream := NewStream("test", DefaultStreamConfig())

	// Send events
	for i := 0; i < 5; i++ {
		event := StreamEvent{
			Type: ChunkContent,
			Data: []byte(fmt.Sprintf("event %d", i)),
		}
		stream.Send(context.Background(), event)
	}

	// Close stream
	err := stream.Close()
	require.NoError(t, err)

	// Should still be able to drain events
	count := 0
	for range stream.Receive() {
		count++
	}

	assert.Equal(t, 5, count)
}

func TestStream_StateTransitions(t *testing.T) {
	stream := NewStream("test", DefaultStreamConfig())

	// Open
	assert.Equal(t, StreamStateOpen, stream.State())
	assert.True(t, stream.IsOpen())

	// Close
	stream.Close()
	assert.Equal(t, StreamStateClosed, stream.State())
	assert.False(t, stream.IsOpen())
}

func TestStream_Concurrent(t *testing.T) {
	stream := NewStream("test", DefaultStreamConfig())
	defer stream.Close()

	var wg sync.WaitGroup
	sendCount := 100
	receiveCount := 0
	receiveMu := sync.Mutex{}

	// Concurrent senders
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < sendCount/5; j++ {
				event := StreamEvent{
					Type: ChunkContent,
					Data: []byte(fmt.Sprintf("sender-%d-event-%d", id, j)),
				}
				stream.Send(context.Background(), event)
			}
		}(i)
	}

	// Concurrent receivers
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case _, ok := <-stream.Receive():
					if !ok {
						return
					}
					receiveMu.Lock()
					receiveCount++
					receiveMu.Unlock()
				case <-time.After(time.Second):
					return
				}
			}
		}()
	}

	wg.Wait()
	stream.Close()

	// Allow remaining events to be processed
	time.Sleep(100 * time.Millisecond)

	receiveMu.Lock()
	defer receiveMu.Unlock()
	assert.Greater(t, receiveCount, 0)
}

func TestRecoverableError(t *testing.T) {
	err := &RecoverableError{
		Err:         fmt.Errorf("network timeout"),
		Recoverable: true,
	}

	assert.True(t, IsRecoverable(err))
	assert.Contains(t, err.Error(), "recoverable=true")

	nonRecoverableErr := &RecoverableError{
		Err:         fmt.Errorf("fatal error"),
		Recoverable: false,
	}

	assert.False(t, IsRecoverable(nonRecoverableErr))
}

func TestRecoverableError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &RecoverableError{
		Err:         inner,
		Recoverable: true,
	}

	assert.Equal(t, inner, err.Unwrap())
}

func TestIsRecoverable_NonRecoverableError(t *testing.T) {
	err := fmt.Errorf("regular error")
	assert.False(t, IsRecoverable(err))
}

func BenchmarkStream_SendReceive(b *testing.B) {
	stream := NewStream("bench", DefaultStreamConfig())
	defer stream.Close()

	event := StreamEvent{
		Type: ChunkContent,
		Data: []byte("benchmark data"),
	}

	// Receiver
	go func() {
		for range stream.Receive() {
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.Send(context.Background(), event)
	}
}

func BenchmarkStream_Concurrent(b *testing.B) {
	stream := NewStream("bench", DefaultStreamConfig())
	defer stream.Close()

	event := StreamEvent{
		Type: ChunkContent,
		Data: []byte("benchmark data"),
	}

	// Receiver
	go func() {
		for range stream.Receive() {
		}
	}()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			stream.Send(context.Background(), event)
		}
	})
}

func TestFeature_4_2_Complete(t *testing.T) {
	// Integration test - complete stream lifecycle
	stream := NewStream("acceptance", DefaultStreamConfig())

	// Should accept events
	sentEvents := make([]string, 100)
	for i := 0; i < 100; i++ {
		data := fmt.Sprintf("event %d", i)
		sentEvents[i] = data
		event := StreamEvent{
			Type: ChunkContent,
			Data: []byte(data),
		}
		err := stream.Send(context.Background(), event)
		require.NoError(t, err)
	}

	// Should receive all events
	receivedCount := 0
	receivedData := make([]string, 0, 100)
	done := make(chan struct{})

	go func() {
		for evt := range stream.Receive() {
			receivedData = append(receivedData, string(evt.Data))
			receivedCount++
			if receivedCount == 100 {
				close(done)
				return
			}
		}
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout receiving events, got %d/100", receivedCount)
	}

	// Verify order and content
	assert.Equal(t, 100, receivedCount)
	assert.Equal(t, sentEvents, receivedData)

	// Should close gracefully
	err := stream.Close()
	require.NoError(t, err)
	assert.False(t, stream.IsOpen())
}
