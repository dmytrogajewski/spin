package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	// Title should be empty for execute blocks to avoid duplication in renderer
	assert.Equal(t, "", block.Title)
	// Command should be in metadata
	assert.NotNil(t, block.Meta)
	meta, err := blocks.ParseExecuteMeta(block)
	require.NoError(t, err)
	assert.Equal(t, "ls -la", meta.Command)
	assert.Equal(t, "/home/user", meta.CWD)
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

	_ = mapper.MapEvent(Event{
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
	// Body should contain the error since output is empty
	assert.Contains(t, block.Body, "exit status 1")
}

// New test: failed execute with non-empty error must not render "No output"
func TestMapEvent_ToolCallComplete_Execute_Error_NoOutputBody(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Start block
	_ = mapper.MapEvent(Event{Type: EventToolCallStart, Data: ToolCallStartData{ToolName: "execute_command", ToolID: "e1", Parameters: makeParams(map[string]interface{}{"command": "python t.py"}).Parameters}})

	// Complete with no stdout/stderr in Output but with Error message
	_ = mapper.MapEvent(Event{Type: EventToolCallComplete, Data: ToolCallCompleteData{ToolID: "e1", ToolName: "execute_command", Success: false, Output: "", Error: "Traceback: NameError"}})

	b := ui.BlocksByID["e1"]
	if b == nil {
		t.Fatalf("block not found")
	}
	// Body should show the error text so users see what failed
	assert.Contains(t, b.Body, "Traceback: NameError")
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

// Integration: WRITE block should NOT show status before completion
func TestWriteBlock_NoStatusBeforeCompletion_Render(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Start write_file
	params := makeParams(map[string]interface{}{
		"path":    "file.txt",
		"content": "hello",
	})
	start := Event{Type: EventToolCallStart, Data: ToolCallStartData{ToolName: "write_file", ToolID: "w1", Parameters: params.Parameters}}
	require.NoError(t, mapper.MapEvent(start))

	// One block appended
	require.Len(t, ui.Blocks, 1)
	b := ui.Blocks[0]
	require.Equal(t, blocks.BlockTypeApplyPatch, b.Type)

	// Render and assert no status/footer yet
	r := blocks.NewRenderer(80)
	out, err := r.Render(b)
	require.NoError(t, err)
	if strings.Contains(out, "Failed to write file.") || strings.Contains(out, "File written successfully.") || strings.Contains(out, "● Failed") || strings.Contains(out, "✓ Succeeded") {
		t.Fatalf("WRITE block rendered premature status before completion. Output:\n%s", out)
	}
}

// Integration: After completion success, WRITE block shows success and not failed
func TestWriteBlock_AfterCompletionSuccess_Render(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Start
	params := makeParams(map[string]interface{}{"path": "ok.txt", "content": "x"})
	require.NoError(t, mapper.MapEvent(Event{Type: EventToolCallStart, Data: ToolCallStartData{ToolName: "write_file", ToolID: "w2", Parameters: params.Parameters}}))

	// Complete success
	require.NoError(t, mapper.MapEvent(Event{Type: EventToolCallComplete, Data: ToolCallCompleteData{ToolID: "w2", ToolName: "write_file", Success: true, Output: "done"}}))

	b := ui.BlocksByID["w2"]
	require.NotNil(t, b)

	r := blocks.NewRenderer(80)
	out, err := r.Render(b)
	require.NoError(t, err)
	assert.Contains(t, out, "File written successfully.")
	assert.Contains(t, out, "Succeeded. File edited.")
	assert.NotContains(t, out, "Failed")
}

// Integration: After completion failure, WRITE block shows failure
func TestWriteBlock_AfterCompletionFailure_Render(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Start
	params := makeParams(map[string]interface{}{"path": "fail.txt", "content": "x"})
	require.NoError(t, mapper.MapEvent(Event{Type: EventToolCallStart, Data: ToolCallStartData{ToolName: "write_file", ToolID: "w3", Parameters: params.Parameters}}))

	// Complete failure
	require.NoError(t, mapper.MapEvent(Event{Type: EventToolCallComplete, Data: ToolCallCompleteData{ToolID: "w3", ToolName: "write_file", Success: false, Error: "disk full"}}))

	b := ui.BlocksByID["w3"]
	require.NotNil(t, b)

	r := blocks.NewRenderer(80)
	out, err := r.Render(b)
	require.NoError(t, err)
	assert.Contains(t, out, "Failed to write file.")
	assert.Contains(t, out, "Failed")
}

// TestMapEvent_DuplicateWriteBlocks_Complete verifies both WRITE blocks reflect success
func TestMapEvent_DuplicateWriteBlocks_Complete(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Start first write_file
	start1 := Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{
			ToolName:   "write_file",
			ToolID:     "write-dup",
			Parameters: makeParams(map[string]interface{}{"path": "a.txt", "content": "hello"}).Parameters,
		},
	}
	require.NoError(t, mapper.MapEvent(start1))

	// Start second write_file with same ToolID (duplicate) to same path
	start2 := Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{
			ToolName:   "write_file",
			ToolID:     "write-dup",
			Parameters: makeParams(map[string]interface{}{"path": "a.txt", "content": "world"}).Parameters,
		},
	}
	require.NoError(t, mapper.MapEvent(start2))

	// Coalescing by path means only one block exists, updated in place
	require.Len(t, ui.Blocks, 1)
	assert.Equal(t, "write-dup", ui.Blocks[0].ID)
	assert.Equal(t, blocks.BlockTypeApplyPatch, ui.Blocks[0].Type)

	// Complete tool call successfully
	complete := Event{
		Type: EventToolCallComplete,
		Data: ToolCallCompleteData{
			ToolID:   "write-dup",
			ToolName: "write_file",
			Success:  true,
			Output:   "Successfully wrote bytes",
		},
	}
	require.NoError(t, mapper.MapEvent(complete))

	// Block should show success and have body set
	b0 := ui.BlocksByID["write-dup"]
	require.NotNil(t, b0)

	assert.Equal(t, blocks.SeverityInfo, b0.Severity)
	meta0, err := blocks.ParsePatchMeta(b0)
	require.NoError(t, err)
	assert.True(t, meta0.Succeeded)
}

// TestMapEvent_ToolCallStart_ApplyPatch verifies APPLY_PATCH block for apply_patch tool
func TestMapEvent_ToolCallStart_ApplyPatch(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	patch := "*** Begin Patch\n*** End Patch\n"
	params := makeParams(map[string]interface{}{
		"patch_text":     patch,
		"workspace_root": ".",
	})

	event := Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{
			ToolName:   "apply_patch",
			ToolID:     "patch-1",
			Parameters: params.Parameters,
		},
	}

	err := mapper.MapEvent(event)
	require.NoError(t, err)

	require.Len(t, ui.Blocks, 1)
	block := ui.Blocks[0]
	assert.Equal(t, blocks.BlockTypeApplyPatch, block.Type)
	assert.Equal(t, ".", block.Title)
	assert.Contains(t, block.Body, "*** Begin Patch")
}

// TestMapEvent_ToolCallComplete_ApplyPatch verifies completion metadata update
func TestMapEvent_ToolCallComplete_ApplyPatch(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// Start
	start := Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{
			ToolName:   "apply_patch",
			ToolID:     "patch-2",
			Parameters: makeParams(map[string]interface{}{"patch_text": "x", "workspace_root": "."}).Parameters,
		},
	}
	require.NoError(t, mapper.MapEvent(start))

	// Complete (success)
	complete := Event{
		Type: EventToolCallComplete,
		Data: ToolCallCompleteData{
			ToolID:   "patch-2",
			ToolName: "apply_patch",
			Success:  true,
			Output:   "Patch applied successfully.",
		},
	}
	require.NoError(t, mapper.MapEvent(complete))

	block := ui.BlocksByID["patch-2"]
	require.NotNil(t, block)
	meta, err := blocks.ParsePatchMeta(block)
	require.NoError(t, err)
	assert.True(t, meta.Succeeded)
}

// TestMapEvent_ToolCallStart_FileSearch verifies GREP block for file_search tool
func TestMapEvent_ToolCallStart_FileSearch(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	params := makeParams(map[string]interface{}{"query": "main.go"})
	event := Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{
			ToolName:   "file_search",
			ToolID:     "search-1",
			Parameters: params.Parameters,
		},
	}

	err := mapper.MapEvent(event)
	require.NoError(t, err)

	require.Len(t, ui.Blocks, 1)
	block := ui.Blocks[0]
	assert.Equal(t, blocks.BlockTypeGrep, block.Type)
	meta, err := blocks.ParseGrepMeta(block)
	require.NoError(t, err)
	assert.Equal(t, "main.go", meta.Pattern)
	assert.Equal(t, "files_with_matches", meta.Mode)
}

// TestMapEvent_ToolCallStart_GitAndContext verifies NOTICE blocks
func TestMapEvent_ToolCallStart_GitAndContext(t *testing.T) {
	ui := NewFakeUI()
	mapper := NewTUIMapper(ui)
	defer mapper.Close()

	// git_context
	_ = mapper.MapEvent(Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{ToolName: "git_context", ToolID: "git-1", Parameters: makeParams(map[string]interface{}{}).Parameters},
	})
	// get_context
	_ = mapper.MapEvent(Event{
		Type: EventToolCallStart,
		Data: ToolCallStartData{ToolName: "get_context", ToolID: "ctx-1", Parameters: makeParams(map[string]interface{}{}).Parameters},
	})

	require.Len(t, ui.Blocks, 2)
	assert.Equal(t, blocks.BlockTypeNotice, ui.Blocks[0].Type)
	assert.Equal(t, blocks.BlockTypeNotice, ui.Blocks[1].Type)
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
		toolID := fmt.Sprintf("tool-%d", i)
		params := makeParams(map[string]interface{}{
			"command": fmt.Sprintf("echo test-%d", i),
		})
		event := Event{
			Type: EventToolCallStart,
			Data: ToolCallStartData{
				ToolName:   "execute_command",
				ToolID:     toolID,
				Parameters: params.Parameters,
			},
		}
		err := mapper.MapEvent(event)
		require.NoError(t, err)
	}

	assert.Len(t, ui.Blocks, 5)

	// Complete them
	for i := 0; i < 5; i++ {
		toolID := fmt.Sprintf("tool-%d", i)
		event := Event{
			Type: EventToolCallComplete,
			Data: ToolCallCompleteData{
				ToolID:   toolID,
				ToolName: "execute_command",
				Success:  true,
				Output:   fmt.Sprintf("test-%d", i),
			},
		}
		err := mapper.MapEvent(event)
		require.NoError(t, err)
	}

	// All should be updated
	for i := 0; i < 5; i++ {
		toolID := fmt.Sprintf("tool-%d", i)
		block := ui.BlocksByID[toolID]
		assert.NotNil(t, block)
		assert.Equal(t, fmt.Sprintf("test-%d", i), block.Body)
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

	// Complete event should update both blocks now
	complete := Event{
		Type: EventToolCallComplete,
		Data: ToolCallCompleteData{
			ToolID:   "duplicate-tool-id",
			ToolName: "read_file",
			Success:  true,
			Output:   "file content",
		},
	}
	require.NoError(t, mapper.MapEvent(complete))

	// Both blocks registered under the same ToolID should be updated
	b0 := ui.BlocksByID["duplicate-tool-id"]
	b1 := ui.BlocksByID["duplicate-tool-id-1"]
	require.NotNil(t, b0)
	require.NotNil(t, b1)
	assert.Equal(t, "file content", b0.Body)
	assert.Equal(t, "file content", b1.Body)
}
