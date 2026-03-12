package blocks

import (
	"encoding/json"
	"errors"
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

	// Meta holds type-specific metadata as JSON.
	Meta json.RawMessage `json:"meta,omitempty"`

	// Body contains the renderable content (logs, code, diff, etc.).
	Body string `json:"body"`

	// FoldState indicates if the block is expanded or collapsed.
	FoldState FoldState `json:"fold_state"`

	// Severity indicates the importance level (info, warn, error).
	Severity Severity `json:"severity"`

	// Timestamp is the Unix timestamp in milliseconds.
	Timestamp int64 `json:"timestamp"`

	// CompletionPrinted tracks whether the completion status line was already printed (UI state only).
	// This prevents duplicate "Tool completed" messages when UpdateBlock is called multiple times.
	CompletionPrinted bool `json:"-"`
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
		ID:        GenerateBlockID(0), // sequence is set by caller if needed.
		Type:      blockType,
		Meta:      nil,
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
		return errors.New("block ID is empty")
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

// Type-safe metadata accessors.

// GetExecuteMeta retrieves ExecuteMeta from the block.
func (b *Block) GetExecuteMeta() (*ExecuteMeta, error) {
	return ParseExecuteMeta(b)
}

// SetExecuteMeta sets ExecuteMeta on the block.
func (b *Block) SetExecuteMeta(m *ExecuteMeta) error {
	return SetExecuteMeta(b, m)
}

// GetReadMeta retrieves ReadMeta from the block.
func (b *Block) GetReadMeta() (*ReadMeta, error) {
	return ParseReadMeta(b)
}

// SetReadMeta sets ReadMeta on the block.
func (b *Block) SetReadMeta(m *ReadMeta) error {
	return SetReadMeta(b, m)
}

// GetGrepMeta retrieves GrepMeta from the block.
func (b *Block) GetGrepMeta() (*GrepMeta, error) {
	return ParseGrepMeta(b)
}

// SetGrepMeta sets GrepMeta on the block.
func (b *Block) SetGrepMeta(m *GrepMeta) error {
	return SetGrepMeta(b, m)
}

// GetToolMeta retrieves ToolMeta from the block.
func (b *Block) GetToolMeta() (*ToolMeta, error) {
	return ParseToolMeta(b)
}

// SetToolMeta sets ToolMeta on the block.
func (b *Block) SetToolMeta(m *ToolMeta) error {
	return SetToolMeta(b, m)
}

// GetPatchMeta retrieves PatchMeta from the block.
func (b *Block) GetPatchMeta() (*PatchMeta, error) {
	return ParsePatchMeta(b)
}

// SetPatchMeta sets PatchMeta on the block.
func (b *Block) SetPatchMeta(m *PatchMeta) error {
	return SetPatchMeta(b, m)
}

// GetPlanMeta retrieves PlanMeta from the block.
func (b *Block) GetPlanMeta() (*PlanMeta, error) {
	return ParsePlanMeta(b)
}

// SetPlanMeta sets PlanMeta on the block.
func (b *Block) SetPlanMeta(m *PlanMeta) error {
	return SetPlanMeta(b, m)
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
		// Use nanoseconds mod 100 as sequence for uniqueness within same millisecond.
		seq = int(time.Now().UnixNano()%100) + 1
	}

	return fmt.Sprintf("blk_%d_%02d", ts, seq)
}
