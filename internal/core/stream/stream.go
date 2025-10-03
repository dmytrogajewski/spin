// Package stream provides streaming infrastructure for event handling.
package stream

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Stream manages a data stream with buffering and flow control.
// It provides thread-safe operations for sending and receiving events
// with configurable backpressure handling.
//
// Example:
//
//	stream := NewStream("example", DefaultStreamConfig())
//	defer stream.Close()
//
//	event := StreamEvent{Type: ChunkContent, Data: []byte("hello")}
//	stream.Send(context.Background(), event)
//
//	for evt := range stream.Receive() {
//	    fmt.Println(string(evt.Data))
//	}
type Stream struct {
	id       string
	buffer   *Buffer
	output   chan StreamEvent
	errors   chan error
	done     chan struct{}
	state    StreamState
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	sequence int64

	// Configuration
	config StreamConfig
}

// StreamConfig configures stream behavior.
type StreamConfig struct {
	BufferSize      int                  // Buffer capacity
	InputBuffer     int                  // Input channel buffer size
	OutputBuffer    int                  // Output channel buffer size
	Backpressure    BackpressureStrategy // How to handle slow consumers
	ErrorHandler    ErrorHandler         // Error handling callback
	MaxSequenceSkip int                  // Max allowed sequence gap
}

// StreamState represents stream lifecycle state.
type StreamState int

const (
	// StreamStateOpen indicates stream is accepting input
	StreamStateOpen StreamState = iota
	// StreamStateDraining indicates stream is closing but draining events
	StreamStateDraining
	// StreamStateClosed indicates stream is fully closed
	StreamStateClosed
)

// BackpressureStrategy defines how to handle backpressure.
type BackpressureStrategy int

const (
	// BackpressureBlock blocks until consumer ready
	BackpressureBlock BackpressureStrategy = iota
	// BackpressureDrop drops oldest events
	BackpressureDrop
	// BackpressureError returns error
	BackpressureError
)

// ErrorHandler is called when stream error occurs.
type ErrorHandler func(err error)

// DefaultStreamConfig returns default configuration.
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		BufferSize:      DefaultBufferCapacity,
		InputBuffer:     100,
		OutputBuffer:    100,
		Backpressure:    BackpressureBlock,
		ErrorHandler:    func(err error) {}, // no-op
		MaxSequenceSkip: 10,
	}
}

// NewStream creates a new stream with the given configuration.
func NewStream(id string, config StreamConfig) *Stream {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Stream{
		id:       id,
		buffer:   NewBuffer(config.BufferSize),
		output:   make(chan StreamEvent, config.OutputBuffer),
		errors:   make(chan error, 10), // Small buffer for errors
		done:     make(chan struct{}),
		state:    StreamStateOpen,
		ctx:      ctx,
		cancel:   cancel,
		sequence: 0,
		config:   config,
	}

	return s
}

// Send sends an event to the stream.
// Returns error if stream is closed or context is cancelled.
func (s *Stream) Send(ctx context.Context, event StreamEvent) error {
	s.mu.RLock()
	if s.state == StreamStateClosed {
		s.mu.RUnlock()
		return ErrStreamClosed
	}
	s.mu.RUnlock()

	// Auto-assign sequence number
	seq := atomic.AddInt64(&s.sequence, 1)
	event.Sequence = seq

	// Handle backpressure
	return s.handleBackpressure(ctx, event)
}

// Receive returns a channel for receiving events.
func (s *Stream) Receive() <-chan StreamEvent {
	return s.output
}

// Errors returns a channel for receiving errors.
func (s *Stream) Errors() <-chan error {
	return s.errors
}

// Close closes the stream gracefully.
// Allows draining of remaining events.
func (s *Stream) Close() error {
	s.mu.Lock()
	if s.state == StreamStateClosed {
		s.mu.Unlock()
		return nil // Already closed
	}

	s.state = StreamStateClosed
	s.mu.Unlock()

	// Signal shutdown
	s.cancel()

	// Close output channel
	close(s.output)
	close(s.done)

	return nil
}

// State returns current stream state.
func (s *Stream) State() StreamState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// IsOpen returns true if stream is accepting input.
func (s *Stream) IsOpen() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == StreamStateOpen
}

// handleBackpressure applies the configured backpressure strategy.
func (s *Stream) handleBackpressure(ctx context.Context, event StreamEvent) error {
	strategy := s.config.Backpressure

	switch strategy {
	case BackpressureBlock:
		return s.sendBlocking(ctx, event)

	case BackpressureDrop:
		return s.sendDropping(event)

	case BackpressureError:
		return s.sendWithError(event)

	default:
		return ErrInvalidStrategy
	}
}

// sendBlocking blocks until event can be sent or context is cancelled.
func (s *Stream) sendBlocking(ctx context.Context, event StreamEvent) error {
	// Check context first (avoid race with ready channel)
	select {
	case <-ctx.Done():
		err := ctx.Err()
		s.propagateError(err)
		return err
	case <-s.ctx.Done():
		err := ErrStreamClosed
		s.propagateError(err)
		return err
	default:
	}

	// Now do the actual send with context monitoring
	select {
	case s.output <- event:
		return nil
	case <-ctx.Done():
		err := ctx.Err()
		s.propagateError(err)
		return err
	case <-s.ctx.Done():
		err := ErrStreamClosed
		s.propagateError(err)
		return err
	}
}

// sendDropping drops oldest event if buffer is full.
func (s *Stream) sendDropping(event StreamEvent) error {
	select {
	case s.output <- event:
		return nil
	default:
		// Buffer full, try to drop oldest
		select {
		case <-s.output:
			// Dropped oldest event
		default:
		}
		// Try again
		select {
		case s.output <- event:
			return nil
		default:
			return ErrEventDropped
		}
	}
}

// sendWithError returns error immediately if buffer is full.
func (s *Stream) sendWithError(event StreamEvent) error {
	select {
	case s.output <- event:
		return nil
	default:
		return ErrBackpressure
	}
}

// propagateError sends error to error channel and optionally to handler.
func (s *Stream) propagateError(err error) {
	if err == nil {
		return
	}

	// Call error handler if configured
	if s.config.ErrorHandler != nil {
		s.config.ErrorHandler(err)
	}

	// Send to error channel (non-blocking)
	select {
	case s.errors <- err:
	default:
		// Error channel full, discard
	}
}

// RecoverableError wraps errors that can be recovered from.
type RecoverableError struct {
	Err         error
	Recoverable bool
}

// Error implements the error interface.
func (e *RecoverableError) Error() string {
	return fmt.Sprintf("stream error (recoverable=%v): %v", e.Recoverable, e.Err)
}

// Unwrap returns the underlying error.
func (e *RecoverableError) Unwrap() error {
	return e.Err
}

// IsRecoverable checks if error can be recovered from.
func IsRecoverable(err error) bool {
	if err == nil {
		return false
	}
	// Check if error is or wraps RecoverableError
	for err != nil {
		if re, ok := err.(*RecoverableError); ok {
			return re.Recoverable
		}
		// Try to unwrap
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}
