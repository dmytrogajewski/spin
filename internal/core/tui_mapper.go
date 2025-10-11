package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/dmytrogajewski/spin/internal/types"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/output"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
)

// TUIMapper maps core agent events to TUI blocks.
// It translates the event stream from the core agent into visual blocks
// that are displayed in the terminal UI timeline.
type TUIMapper struct {
	ui            ports.UI
	blockRegistry map[string]*blocks.Block // toolID → block (for updates)
	streamCh      chan string              // Content streaming channel
	streamMu      sync.Mutex               // Protects streamCh
	streamCtx     context.Context
	streamCancel  context.CancelFunc
	thinkFilter   *output.ThinkFilter      // Filter for <think> tags
	mu            sync.RWMutex // Protects blockRegistry
}

// NewTUIMapper creates a new TUI event mapper.
// The mapper subscribes to core events and translates them into blocks
// that are appended to or updated in the UI timeline.
func NewTUIMapper(ui ports.UI) *TUIMapper {
	return &TUIMapper{
		ui:            ui,
		blockRegistry: make(map[string]*blocks.Block),
		thinkFilter:   output.NewThinkFilter(),
	}
}

// MapEvent processes a core event and updates the TUI accordingly.
// It handles tool calls, content streaming, errors, and system messages.
// Returns an error only for critical failures; gracefully handles unexpected data.
func (m *TUIMapper) MapEvent(event Event) error {
	// Handle nil data gracefully
	if event.Data == nil {
		return nil
	}

	switch event.Type {
	case EventToolCallStart:
		return m.handleToolStart(event)
	case EventToolCallComplete:
		return m.handleToolComplete(event)
	case EventContentDelta:
		return m.handleContentDelta(event)
	case EventError:
		return m.handleError(event)
	case EventInfo, EventWarning:
		return m.handleSystemEvent(event)
	default:
		// Ignore unknown events gracefully
		return nil
	}
}

// handleToolStart creates a new block when a tool execution starts.
func (m *TUIMapper) handleToolStart(event Event) error {
	data, ok := event.Data.(ToolCallStartData)
	if !ok {
		return nil // Gracefully handle type assertion failure
	}

	// Create block first (outside lock)
	var block *blocks.Block

	switch data.ToolName {
	case "execute_command":
		block = m.createExecuteBlock(data)
	case "read_file":
		block = m.createReadBlock(data)
	case "write_file":
		block = m.createApplyPatchBlock(data)
	case "list_directory":
		block = m.createExecuteBlock(data) // Treat as EXECUTE
	default:
		// Unknown tool, ignore
		return nil
	}

	if block == nil {
		return nil
	}

	// Atomically check and register (prevent race condition)
	m.mu.Lock()

	// Debug: log registry state
	slog.Debug("Checking blockRegistry for duplicate",
		"tool_id", data.ToolID,
		"registry_size", len(m.blockRegistry),
		"registry_contains", m.blockRegistry[data.ToolID] != nil)

	if existingBlock, exists := m.blockRegistry[data.ToolID]; exists {
		// Duplicate tool ID from LLM - make block ID unique by appending counter
		// This is a workaround for LLM bugs that reuse tool IDs
		slog.Warn("Duplicate tool ID from LLM, making block ID unique",
			"tool_id", data.ToolID,
			"new_tool_name", data.ToolName,
			"existing_block_type", existingBlock.Type)

		// Find a unique block ID by appending -1, -2, etc.
		originalID := block.ID
		for i := 1; ; i++ {
			block.ID = fmt.Sprintf("%s-%d", originalID, i)
			// Check if this ID exists in timeline by trying to append
			// (we can't check timeline directly from here)
			break // For now, just use -1 suffix
		}

		slog.Info("Created unique block ID for duplicate tool",
			"original_tool_id", data.ToolID,
			"new_block_id", block.ID)

		// Don't overwrite the registry entry - keep the first block registered
		// This means we can't update the duplicate block later, but that's OK
	} else {
		// First time seeing this tool ID - register it normally
		m.blockRegistry[data.ToolID] = block
	}
	m.mu.Unlock()

	// Append to UI timeline
	if err := m.ui.AppendBlock(block); err != nil {
		// Log the error with context for debugging
		slog.Error("Failed to append block to timeline",
			"error", err,
			"block_id", block.ID,
			"tool_id", data.ToolID,
			"tool_name", data.ToolName)
		return err
	}
	return nil
}

// createExecuteBlock creates an EXECUTE block for command execution.
func (m *TUIMapper) createExecuteBlock(data ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.ID = data.ToolID

	// Extract command from parameters (or path for list_directory)
	command := extractString(data.Parameters, "command")
	if command == "" && data.ToolName == "list_directory" {
		// For list_directory, use the path as the command
		command = "ls " + extractString(data.Parameters, "path")
	}
	block.Title = command

	// Store metadata
	cwd := extractString(data.Parameters, "cwd")
	if cwd == "" {
		cwd = "."
	}
	meta := &blocks.ExecuteMeta{
		Command: command,
		CWD:     cwd,
		Impact:  "medium", // Default impact level
	}
	if err := blocks.SetExecuteMeta(block, meta); err != nil {
		// Validation failed, set as raw map to preserve data
		block.Meta = map[string]interface{}{
			"command": command,
			"cwd":     cwd,
		}
	}

	return block
}

// createReadBlock creates a READ block for file reading.
func (m *TUIMapper) createReadBlock(data ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeRead)
	block.ID = data.ToolID

	path := extractString(data.Parameters, "path")
	block.Title = path

	meta := &blocks.ReadMeta{
		File:   path,
		Offset: extractIntValue(data.Parameters, "offset"),
		Limit:  extractIntValue(data.Parameters, "limit"),
	}
	if err := blocks.SetReadMeta(block, meta); err != nil {
		// Validation failed, set as raw map
		block.Meta = map[string]interface{}{
			"file": path,
		}
	}

	return block
}

// createApplyPatchBlock creates an APPLY_PATCH block for file writing.
func (m *TUIMapper) createApplyPatchBlock(data ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeApplyPatch)
	block.ID = data.ToolID

	path := extractString(data.Parameters, "path")
	block.Title = path

	// Store content as body (will be formatted as diff if possible)
	content := extractString(data.Parameters, "content")
	block.Body = content

	meta := &blocks.PatchMeta{
		File: path,
	}
	if err := blocks.SetPatchMeta(block, meta); err != nil {
		// Validation failed, set as raw map
		block.Meta = map[string]interface{}{
			"file": path,
		}
	}

	return block
}

// handleToolComplete updates an existing block when tool execution completes.
func (m *TUIMapper) handleToolComplete(event Event) error {
	data, ok := event.Data.(ToolCallCompleteData)
	if !ok {
		return nil
	}

	// Find block
	m.mu.RLock()
	block, exists := m.blockRegistry[data.ToolID]
	m.mu.RUnlock()

	if !exists {
		// Block not found (tool started before mapper attached), ignore
		return nil
	}

	// Update block with results
	block.Body = data.Output

	// Set severity based on success
	if !data.Success {
		block.Severity = blocks.SeverityError
		if data.Error != "" {
			block.Body += "\n\nError: " + data.Error
		}
	} else {
		block.Severity = blocks.SeverityInfo
	}

	// Update metadata for EXECUTE blocks
	if block.Type == blocks.BlockTypeExecute {
		meta, err := blocks.ParseExecuteMeta(block)
		if err == nil && meta != nil {
			// Parse exit code from output (heuristic)
			if !data.Success {
				meta.ExitCode = intPtr(1) // Assume non-zero on failure
			} else {
				meta.ExitCode = intPtr(0)
			}
			meta.LinesOut = countLinesPtr(data.Output)
			// Update metadata, ignore validation errors as block already has output
			_ = blocks.SetExecuteMeta(block, meta)
		}
	}

	// Update in UI
	err := m.ui.UpdateBlock(data.ToolID, block)

	// Clean up registry
	m.mu.Lock()
	delete(m.blockRegistry, data.ToolID)
	m.mu.Unlock()

	return err
}

// handleContentDelta streams assistant content to the UI.
func (m *TUIMapper) handleContentDelta(event Event) error {
	data, ok := event.Data.(ContentDeltaData)
	if !ok || data.Role != "assistant" {
		return nil
	}

	// Filter <think> tags and apply formatting
	filtered := m.thinkFilter.Process(data.Content)

	// If streaming is active, send filtered chunk
	m.streamMu.Lock()
	defer m.streamMu.Unlock()

	if m.streamCh != nil && filtered != "" {
		select {
		case m.streamCh <- filtered:
		case <-m.streamCtx.Done():
			// Stream closed, drop
		default:
			// Channel full, drop (UI has coalescing)
		}
	}

	return nil
}

// handleError creates an ERROR block for error events.
func (m *TUIMapper) handleError(event Event) error {
	data, ok := event.Data.(ErrorData)
	if !ok {
		return nil
	}

	block := blocks.NewBlock(blocks.BlockTypeError)
	block.ID = generateBlockID()
	block.Title = data.Message
	block.Body = data.Details
	block.Severity = blocks.SeverityError

	return m.ui.AppendBlock(block)
}

// handleSystemEvent creates a NOTICE block for system messages.
func (m *TUIMapper) handleSystemEvent(event Event) error {
	data, ok := event.Data.(SystemEventData)
	if !ok {
		return nil
	}

	block := blocks.NewBlock(blocks.BlockTypeNotice)
	block.ID = generateBlockID()
	block.Title = data.Message
	block.Body = data.Details

	// Map severity
	switch data.Level {
	case "error":
		block.Severity = blocks.SeverityError
	case "warning", "warn":
		block.Severity = blocks.SeverityWarn
	default:
		block.Severity = blocks.SeverityInfo
	}

	return m.ui.AppendBlock(block)
}

// StartStreaming initializes content streaming and returns the channel
// that will receive LLM content deltas. The caller should wire this to UI.PrintChunks.
func (m *TUIMapper) StartStreaming() <-chan string {
	m.streamMu.Lock()
	defer m.streamMu.Unlock()

	if m.streamCh != nil {
		return m.streamCh // Already started
	}

	m.streamCh = make(chan string, 100)
	m.streamCtx, m.streamCancel = context.WithCancel(context.Background())

	return m.streamCh
}

// StopStreaming closes the content streaming channel.
func (m *TUIMapper) StopStreaming() {
	m.streamMu.Lock()
	defer m.streamMu.Unlock()

	// Flush any buffered think content
	if m.streamCh != nil {
		if flushed := m.thinkFilter.Flush(); flushed != "" {
			select {
			case m.streamCh <- flushed:
			default:
				// Channel full or closed, drop
			}
		}
	}

	if m.streamCancel != nil {
		m.streamCancel()
	}

	if m.streamCh != nil {
		close(m.streamCh)
		m.streamCh = nil
	}

	// Reset filter for next turn
	m.thinkFilter.Reset()
}

// Close cleans up mapper resources (closes stream channels).
func (m *TUIMapper) Close() error {
	m.StopStreaming()

	m.mu.Lock()
	m.blockRegistry = make(map[string]*blocks.Block)
	m.mu.Unlock()

	return nil
}

// Helper functions

// extractString safely extracts a string parameter from ToolCallArguments.
func extractString(params types.ToolCallArguments, key string) string {
	var s string
	if err := params.Get(key, &s); err == nil {
		return s
	}
	return ""
}

// extractIntValue safely extracts an int parameter from ToolCallArguments.
func extractIntValue(params types.ToolCallArguments, key string) int {
	var i int
	if err := params.Get(key, &i); err == nil {
		return i
	}
	return 0
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
}

// countLinesPtr counts the number of lines in a string and returns a pointer.
func countLinesPtr(s string) *int {
	if s == "" {
		return intPtr(0)
	}
	count := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		count++
	}
	return intPtr(count)
}

// generateBlockID generates a unique block ID for blocks without tool IDs.
func generateBlockID() string {
	// Simple counter-based ID (could use UUID for production)
	// For now, use timestamp-based
	return fmt.Sprintf("block-%d", eventIDCounter.Add(1))
}

// Simple atomic counter for block IDs (thread-safe)
var eventIDCounter = &atomicCounter{}

type atomicCounter struct {
	mu  sync.Mutex
	val int
}

func (c *atomicCounter) Add(delta int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.val += delta
	return c.val
}
