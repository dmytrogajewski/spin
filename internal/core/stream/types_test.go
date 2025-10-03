package stream

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamEvent_Creation(t *testing.T) {
	event := StreamEvent{
		Sequence:  1,
		Type:      ChunkContent,
		Data:      []byte("test content"),
		Timestamp: time.Now(),
	}

	assert.Equal(t, int64(1), event.Sequence)
	assert.Equal(t, ChunkContent, event.Type)
	assert.Equal(t, "test content", string(event.Data))
	assert.NotZero(t, event.Timestamp)
}

func TestStreamEvent_WithMetadata(t *testing.T) {
	metadata := Metadata{
		Model:      "gpt-4",
		Provider:   "openai",
		TokenCount: 42,
		Custom: map[string]string{
			"key": "value",
		},
	}

	event := StreamEvent{
		Sequence:  1,
		Type:      ChunkContent,
		Data:      []byte("test"),
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	assert.Equal(t, "gpt-4", event.Metadata.Model)
	assert.Equal(t, "openai", event.Metadata.Provider)
	assert.Equal(t, 42, event.Metadata.TokenCount)
	assert.Equal(t, "value", event.Metadata.Custom["key"])
}

func TestStreamEvent_WithError(t *testing.T) {
	err := ErrStreamClosed

	event := StreamEvent{
		Sequence:  1,
		Type:      ChunkError,
		Data:      []byte("error occurred"),
		Error:     err,
		Timestamp: time.Now(),
	}

	assert.Equal(t, ChunkError, event.Type)
	assert.NotNil(t, event.Error)
	assert.Equal(t, ErrStreamClosed, event.Error)
}

func TestStreamEvent_JSONSerialization(t *testing.T) {
	original := StreamEvent{
		Sequence:  42,
		Type:      ChunkContent,
		Data:      []byte("hello world"),
		Timestamp: time.Now().Round(time.Millisecond), // Round for comparison
		Metadata: Metadata{
			Model:      "test-model",
			TokenCount: 10,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal from JSON
	var decoded StreamEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Sequence, decoded.Sequence)
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.Data, decoded.Data)
	assert.Equal(t, original.Metadata.Model, decoded.Metadata.Model)
}

func TestChunkType_String(t *testing.T) {
	tests := []struct {
		chunkType ChunkType
		expected  string
	}{
		{ChunkContent, "content"},
		{ChunkToolCall, "tool_call"},
		{ChunkFunctionCall, "function_call"},
		{ChunkDelta, "delta"},
		{ChunkComplete, "complete"},
		{ChunkError, "error"},
		{ChunkType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.chunkType.String())
		})
	}
}

func TestChunkType_AllTypes(t *testing.T) {
	// Ensure all chunk types have unique values
	types := []ChunkType{
		ChunkContent,
		ChunkToolCall,
		ChunkFunctionCall,
		ChunkDelta,
		ChunkComplete,
		ChunkError,
	}

	seen := make(map[ChunkType]bool)
	for _, ct := range types {
		assert.False(t, seen[ct], "duplicate chunk type: %v", ct)
		seen[ct] = true
	}
}

func TestMetadata_Empty(t *testing.T) {
	metadata := Metadata{}

	assert.Empty(t, metadata.Model)
	assert.Empty(t, metadata.Provider)
	assert.Zero(t, metadata.TokenCount)
	assert.Nil(t, metadata.Custom)
}

func TestMetadata_WithFinishReason(t *testing.T) {
	metadata := Metadata{
		Model:        "test-model",
		FinishReason: "stop",
	}

	assert.Equal(t, "stop", metadata.FinishReason)
}

func TestMetadata_CustomFields(t *testing.T) {
	metadata := Metadata{
		Custom: map[string]string{
			"field1": "value1",
			"field2": "value2",
			"field3": "value3",
		},
	}

	assert.Len(t, metadata.Custom, 3)
	assert.Equal(t, "value1", metadata.Custom["field1"])
	assert.Equal(t, "value2", metadata.Custom["field2"])
	assert.Equal(t, "value3", metadata.Custom["field3"])
}

func TestStreamEvent_Sequence(t *testing.T) {
	// Test sequence ordering
	events := []StreamEvent{
		{Sequence: 1, Type: ChunkContent, Data: []byte("first")},
		{Sequence: 2, Type: ChunkContent, Data: []byte("second")},
		{Sequence: 3, Type: ChunkContent, Data: []byte("third")},
	}

	for i, event := range events {
		assert.Equal(t, int64(i+1), event.Sequence)
	}
}

func TestStreamEvent_CompleteMarker(t *testing.T) {
	event := StreamEvent{
		Sequence:  100,
		Type:      ChunkComplete,
		Data:      nil,
		Timestamp: time.Now(),
		Metadata: Metadata{
			FinishReason: "stop",
		},
	}

	assert.Equal(t, ChunkComplete, event.Type)
	assert.Equal(t, "stop", event.Metadata.FinishReason)
	assert.Nil(t, event.Data)
}

func TestStreamEvent_EmptyData(t *testing.T) {
	event := StreamEvent{
		Sequence:  1,
		Type:      ChunkDelta,
		Data:      []byte{},
		Timestamp: time.Now(),
	}

	assert.NotNil(t, event.Data)
	assert.Empty(t, event.Data)
}

func BenchmarkStreamEvent_Creation(b *testing.B) {
	data := []byte("benchmark data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StreamEvent{
			Sequence:  int64(i),
			Type:      ChunkContent,
			Data:      data,
			Timestamp: time.Now(),
		}
	}
}

func BenchmarkStreamEvent_JSONMarshal(b *testing.B) {
	event := StreamEvent{
		Sequence:  1,
		Type:      ChunkContent,
		Data:      []byte("benchmark data"),
		Timestamp: time.Now(),
		Metadata: Metadata{
			Model:      "test-model",
			TokenCount: 10,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(event)
	}
}
