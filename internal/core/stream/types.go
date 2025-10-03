package stream

import (
	"errors"
	"time"
)

// StreamEvent represents a chunk of data in a stream.
// It carries sequenced data with metadata for ordered processing.
type StreamEvent struct {
	Sequence  int64     `json:"sequence"`        // Sequence number for ordering
	Type      ChunkType `json:"type"`            // Type of chunk
	Data      []byte    `json:"data"`            // Actual data payload
	Metadata  Metadata  `json:"metadata"`        // Additional metadata
	Timestamp time.Time `json:"timestamp"`       // When chunk was received
	Error     error     `json:"error,omitempty"` // Error if any
}

// ChunkType identifies the type of stream chunk.
type ChunkType int

const (
	// ChunkContent represents a content chunk from LLM
	ChunkContent ChunkType = iota
	// ChunkToolCall represents a tool call chunk
	ChunkToolCall
	// ChunkFunctionCall represents a function call chunk
	ChunkFunctionCall
	// ChunkDelta represents a delta update
	ChunkDelta
	// ChunkComplete represents stream completion marker
	ChunkComplete
	// ChunkError represents an error chunk
	ChunkError
)

// String returns the string representation of ChunkType.
func (c ChunkType) String() string {
	names := []string{
		"content",
		"tool_call",
		"function_call",
		"delta",
		"complete",
		"error",
	}

	if int(c) < len(names) {
		return names[c]
	}
	return "unknown"
}

// Metadata contains chunk metadata.
type Metadata struct {
	Model        string            `json:"model,omitempty"`         // Model identifier
	Provider     string            `json:"provider,omitempty"`      // Provider name
	TokenCount   int               `json:"token_count,omitempty"`   // Token count
	FinishReason string            `json:"finish_reason,omitempty"` // Finish reason
	Custom       map[string]string `json:"custom,omitempty"`        // Custom metadata
}

// Common errors for stream operations.
var (
	ErrStreamClosed    = errors.New("stream is closed")
	ErrInvalidStrategy = errors.New("invalid backpressure strategy")
	ErrEventDropped    = errors.New("event dropped due to backpressure")
	ErrBackpressure    = errors.New("backpressure detected")
	ErrSequenceGap     = errors.New("sequence gap detected")
	ErrBufferFull      = errors.New("buffer is full")
	ErrBufferEmpty     = errors.New("buffer is empty")
)
