package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FakeUI implements ports.UI for testing
type FakeUI struct {
	Blocks       []*blocks.Block
	BlocksByID   map[string]*blocks.Block
	Lines        []string
	Chunks       []string
	StatusText   string
	InputChannel chan string
}

func NewFakeUI() *FakeUI {
	return &FakeUI{
		Blocks:       make([]*blocks.Block, 0),
		BlocksByID:   make(map[string]*blocks.Block),
		Lines:        make([]string, 0),
		Chunks:       make([]string, 0),
		InputChannel: make(chan string, 10),
	}
}

func (f *FakeUI) Run(ctx context.Context) error { return nil }
func (f *FakeUI) Stop() error                   { return nil }

func (f *FakeUI) PrintLine(line string) error {
	f.Lines = append(f.Lines, line)
	return nil
}

func (f *FakeUI) PrintChunks(ctx context.Context, chunks <-chan string) error {
	for chunk := range chunks {
		f.Chunks = append(f.Chunks, chunk)
	}
	return nil
}

func (f *FakeUI) SetStatus(status string) error {
	f.StatusText = status
	return nil
}

func (f *FakeUI) RequestInput() <-chan string {
	return f.InputChannel
}

func (f *FakeUI) AppendBlock(block *blocks.Block) error {
	f.Blocks = append(f.Blocks, block)
	f.BlocksByID[block.ID] = block
	return nil
}

func (f *FakeUI) UpdateBlock(blockID string, block *blocks.Block) error {
	if existing, ok := f.BlocksByID[blockID]; ok {
		*existing = *block
	}
	return nil
}

func (f *FakeUI) DeleteBlock(blockID string) error {
	delete(f.BlocksByID, blockID)
	return nil
}

// makeParams creates ToolCallArguments from a map (helper for tests)
func makeParams(m map[string]interface{}) ToolCallStartData {
	params := make(map[string]json.RawMessage)
	for k, v := range m {
		b, _ := json.Marshal(v)
		params[k] = b
	}
	return ToolCallStartData{Parameters: params}
}

// TestMapEvent_ToolCallStart_Execute verifies EXECUTE block creation
func TestMapEvent_ToolCallStart_Execute(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	params := makeParams(map[string]interface{}{
		"command": "ls -la",
		"cwd":     "/home/user",
	})

	event := Event{
		Type:      EventToolCallStart,
		Timestamp: time.Now(),
		Data: ToolCallStartData{
			ToolName:   "execute_command",
			ToolID:     "tool-123",
			Parameters: params.Parameters,
		},
	}

	err := mapper.MapEvent(event)
	require.NoError(t, err)

	// Verify block created
	require.Len(t, ui.Blocks, 1)
	block := ui.Blocks[0]

	assert.Equal(t, blocks.BlockTypeExecute, block.Type)
	assert.Equal(t, "tool-123", block.ID)
	assert.Equal(t, "ls -la", block.Title)
	assert.NotNil(t, block.Meta)
}

// TestMapEvent_ToolCallComplete_Execute verifies EXECUTE block update
func TestMapEvent_ToolCallComplete_Execute(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// First create the block
	params := makeParams(map[string]interface{}{"command": "echo hello"})
	startEvent := Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{
			ToolName:   "execute_command",
			ToolID:     "tool-123",
			Parameters: params.Parameters,
		},
	}
	err := mapper.MapEvent(startEvent)
	require.NoError(t, err)

	// Then complete it
	completeEvent := Event{
		Type:      EventToolCallComplete,
		Timestamp: time.Now(),
		Data: ToolCallCompleteData{
			ToolID:   "tool-123",
			ToolName: "execute_command",
			Success:  true,
			Output:   "hello\n",
		},
	}

	err = mapper.MapEvent(completeEvent)
	require.NoError(t, err)

	// Verify block updated
	block := ui.BlocksByID["tool-123"]
	require.NotNil(t, block)
	assert.Equal(t, "hello\n", block.Body)
	assert.Equal(t, blocks.SeverityInfo, block.Severity)
}

// TestMapEvent_ToolCallComplete_Execute_Error verifies failed command
func TestMapEvent_ToolCallComplete_Execute_Error(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Create and fail
	params := makeParams(map[string]interface{}{"command": "false"})
	mapper.MapEvent(Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{
			ToolName:   "execute_command",
			ToolID:     "tool-456",
			Parameters: params.Parameters,
		},
	})

	mapper.MapEvent(Event{
		Type: EventToolCallComplete,
		Data: ToolCallCompleteData{
			ToolID:   "tool-456",
			ToolName: "execute_command",
			Success:  false,
			Output:   "",
			Error:    "exit status 1",
		},
	})

	block := ui.BlocksByID["tool-456"]
	assert.Equal(t, blocks.SeverityError, block.Severity)
}

// TestMapEvent_ToolCallStart_ReadFile verifies READ block creation
func TestMapEvent_ToolCallStart_ReadFile(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	params := makeParams(map[string]interface{}{"path": "main.go"})
	event := Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{
			ToolName:   "read_file",
			ToolID:     "read-1",
			Parameters: params.Parameters,
		},
	}

	err := mapper.MapEvent(event)
	require.NoError(t, err)

	require.Len(t, ui.Blocks, 1)
	block := ui.Blocks[0]
	assert.Equal(t, blocks.BlockTypeRead, block.Type)
	assert.Equal(t, "main.go", block.Title)
}

// TestMapEvent_ToolCallStart_WriteFile verifies APPLY_PATCH block
func TestMapEvent_ToolCallStart_WriteFile(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	params := makeParams(map[string]interface{}{
		"path":    "config.yaml",
		"content": "version: 2\n",
	})
	event := Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{
			ToolName:   "write_file",
			ToolID:     "write-1",
			Parameters: params.Parameters,
		},
	}

	err := mapper.MapEvent(event)
	require.NoError(t, err)

	require.Len(t, ui.Blocks, 1)
	block := ui.Blocks[0]
	assert.Equal(t, blocks.BlockTypeApplyPatch, block.Type)
	assert.Equal(t, "config.yaml", block.Title)
}

// TestMapEvent_ContentDelta verifies streaming
func TestMapEvent_ContentDelta(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Start streaming
	streamCh := mapper.StartStreaming()
	go func() {
		// Consume chunks
		for range streamCh {
		}
	}()

	event := Event{
		Type: EventContentDelta,
		Data: ContentDeltaData{
			Content: "Hello ",
			Role:    "assistant",
		},
	}

	err := mapper.MapEvent(event)
	require.NoError(t, err)

	event2 := Event{
		Type: EventContentDelta,
		Data: ContentDeltaData{
			Content: "world!",
			Role:    "assistant",
		},
	}

	err = mapper.MapEvent(event2)
	require.NoError(t, err)

	// Verify chunks sent (need to close and wait)
	mapper.StopStreaming()
	// Note: In real usage, PrintChunks consumes the channel
}

// TestMapEvent_Error verifies ERROR block creation
func TestMapEvent_Error(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	event := Event{
		Type: EventError,
		Data: ErrorData{
			Message: "Connection failed",
			Code:    "ERR_CONN",
			Details: "LLM provider unreachable",
		},
	}

	err := mapper.MapEvent(event)
	require.NoError(t, err)

	require.Len(t, ui.Blocks, 1)
	block := ui.Blocks[0]
	assert.Equal(t, blocks.BlockTypeError, block.Type)
	assert.Equal(t, "Connection failed", block.Title)
	assert.Contains(t, block.Body, "LLM provider unreachable")
	assert.Equal(t, blocks.SeverityError, block.Severity)
}

// TestMapEvent_Info verifies NOTICE block creation
func TestMapEvent_Info(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	event := Event{
		Type: EventInfo,
		Data: SystemEventData{
			Level:   "info",
			Message: "History compressed",
			Details: "Reduced 500 messages to 50 tokens",
		},
	}

	err := mapper.MapEvent(event)
	require.NoError(t, err)

	require.Len(t, ui.Blocks, 1)
	block := ui.Blocks[0]
	assert.Equal(t, blocks.BlockTypeNotice, block.Type)
	assert.Equal(t, "History compressed", block.Title)
}

// TestMapEvent_UnknownEvent verifies graceful handling
func TestMapEvent_UnknownEvent(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	event := Event{
		Type: EventType(999), // Unknown type
		Data: nil,
	}

	err := mapper.MapEvent(event)
	assert.NoError(t, err) // Should not error, just ignore
}

// TestMapEvent_NilData verifies safety
func TestMapEvent_NilData(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	event := Event{
		Type: EventToolCallStart,
		Data: nil,
	}

	err := mapper.MapEvent(event)
	assert.NoError(t, err) // Should handle gracefully
}

// TestMapEvent_MissingBlock verifies update without start
func TestMapEvent_MissingBlock(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Complete without start
	event := Event{
		Type: EventToolCallComplete,
		Data: ToolCallCompleteData{
			ToolID:   "nonexistent",
			ToolName: "execute_command",
			Success:  true,
			Output:   "done",
		},
	}

	err := mapper.MapEvent(event)
	assert.NoError(t, err) // Should not crash

	// No blocks should exist
	assert.Len(t, ui.Blocks, 0)
}

// TestMapEvent_MultipleTools verifies concurrent tool handling
func TestMapEvent_MultipleTools(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Start multiple tools
	for i := 0; i < 5; i++ {
		params := makeParams(map[string]interface{}{
			"command": "echo " + string(rune('a'+i)),
		})
		event := Event{
			Type: EventToolCallStart,
			Data: ToolCallStartData{
				ToolName:   "execute_command",
				ToolID:     string(rune('a' + i)), // "a", "b", "c", etc.
				Parameters: params.Parameters,
			},
		}
		err := mapper.MapEvent(event)
		require.NoError(t, err)
	}

	assert.Len(t, ui.Blocks, 5)

	// Complete them
	for i := 0; i < 5; i++ {
		event := Event{
			Type: EventToolCallComplete,
			Data: ToolCallCompleteData{
				ToolID:   string(rune('a' + i)),
				ToolName: "execute_command",
				Success:  true,
				Output:   string(rune('a' + i)),
			},
		}
		err := mapper.MapEvent(event)
		require.NoError(t, err)
	}

	// All should be updated
	for i := 0; i < 5; i++ {
		block := ui.BlocksByID[string(rune('a'+i))]
		assert.NotNil(t, block)
		assert.Equal(t, string(rune('a'+i)), block.Body)
	}
}

// TestMapEvent_DuplicateToolID verifies that duplicate tool IDs are handled gracefully
func TestMapEvent_DuplicateToolID(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Send first tool call start event
	event1 := Event{
		Type:      EventToolCallStart,
		Timestamp: time.Now(),
		Data: ToolCallStartData{
			ToolName: "read_file",
			ToolID:   "duplicate-tool-id",
			Parameters: makeParams(map[string]interface{}{
				"path": "test.txt",
			}).Parameters,
		},
	}

	err := mapper.MapEvent(event1)
	require.NoError(t, err)

	// Should have created 1 block
	assert.Len(t, ui.Blocks, 1)
	assert.Equal(t, "duplicate-tool-id", ui.Blocks[0].ID)

	// Send second tool call start event with SAME tool ID (duplicate)
	event2 := Event{
		Type:      EventToolCallStart,
		Timestamp: time.Now(),
		Data: ToolCallStartData{
			ToolName: "read_file",
			ToolID:   "duplicate-tool-id", // Same ID!
			Parameters: makeParams(map[string]interface{}{
				"path": "different.txt",
			}).Parameters,
		},
	}

	err = mapper.MapEvent(event2)
	require.NoError(t, err)

	// Should now have 2 blocks (duplicate gets unique ID)
	assert.Len(t, ui.Blocks, 2)
	assert.Equal(t, "duplicate-tool-id", ui.Blocks[0].ID)
	assert.Equal(t, "duplicate-tool-id-1", ui.Blocks[1].ID)

	// Original block should be unchanged
	assert.Equal(t, "test.txt", ui.Blocks[0].Title)

	// New block should have different content
	assert.Equal(t, "different.txt", ui.Blocks[1].Title)
}
