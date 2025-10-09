package blocks

import (
	"fmt"
	"time"
)

// Block represents a single block in the TUI timeline.
//
// Each block has a unique ID, type, optional title, type-specific metadata,
// renderable body content, fold state, severity level, and timestamp.
type Block struct {
	// ID is the unique block identifier (format: blk_{timestamp_ms}_{seq}).
	ID string `json:"id"`

	// Type is the block category (EXECUTE, PLAN, READ, etc.).
	Type BlockType `json:"type"`

	// Title is an optional concise title for the block.
	Title string `json:"title,omitempty"`

	// Meta holds type-specific metadata as key-value pairs.
	Meta map[string]interface{} `json:"meta"`

	// Body contains the renderable content (logs, code, diff, etc.).
	Body string `json:"body"`

	// FoldState indicates if the block is expanded or collapsed.
	FoldState FoldState `json:"fold_state"`

	// Severity indicates the importance level (info, warn, error).
	Severity Severity `json:"severity"`

	// Timestamp is the Unix timestamp in milliseconds.
	Timestamp int64 `json:"timestamp"`
}

// NewBlock creates a new block with default values.
//
// The block is initialized with:
//   - Unique ID (generated from current timestamp)
//   - Given block type
//   - Empty metadata map
//   - Expanded fold state
//   - Info severity
//   - Current timestamp
func NewBlock(blockType BlockType) *Block {
	return &Block{
		ID:        GenerateBlockID(0), // sequence will be set by caller if needed
		Type:      blockType,
		Meta:      make(map[string]interface{}),
		Body:      "",
		FoldState: FoldStateExpanded,
		Severity:  SeverityInfo,
		Timestamp: time.Now().UnixMilli(),
	}
}

// Validate validates the block structure.
//
// Returns an error if:
//   - ID is empty
//   - Type is invalid
//   - FoldState is invalid
//   - Severity is invalid
//   - Timestamp is zero or negative
func (b *Block) Validate() error {
	if b.ID == "" {
		return fmt.Errorf("block ID is empty")
	}
	if !b.Type.Valid() {
		return fmt.Errorf("invalid block type: %s", b.Type)
	}
	if !b.FoldState.Valid() {
		return fmt.Errorf("invalid fold state: %s", b.FoldState)
	}
	if !b.Severity.Valid() {
		return fmt.Errorf("invalid severity: %s", b.Severity)
	}
	if b.Timestamp <= 0 {
		return fmt.Errorf("invalid timestamp: %d", b.Timestamp)
	}
	return nil
}

// GetMeta retrieves a metadata value by key.
//
// Returns the value and true if the key exists, or nil and false otherwise.
func (b *Block) GetMeta(key string) (interface{}, bool) {
	val, ok := b.Meta[key]
	return val, ok
}

// SetMeta sets a metadata value.
//
// If the key already exists, its value is replaced.
func (b *Block) SetMeta(key string, value interface{}) {
	if b.Meta == nil {
		b.Meta = make(map[string]interface{})
	}
	b.Meta[key] = value
}

// GenerateBlockID creates a unique block ID.
//
// Format: blk_{unix_timestamp_ms}_{sequence}
// Example: blk_1738950123456_01
//
// The sequence parameter allows generating multiple IDs within the same
// millisecond. Pass 0 for auto-sequence (uses nanoseconds for uniqueness).
func GenerateBlockID(seq int) string {
	ts := time.Now().UnixMilli()
	if seq == 0 {
		// Use nanoseconds mod 100 as sequence for uniqueness within same millisecond
		seq = int(time.Now().UnixNano()%100) + 1
	}
	return fmt.Sprintf("blk_%d_%02d", ts, seq)
}
