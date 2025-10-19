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
	ui               ports.UI
	blockRegistry    map[string][]*blocks.Block // toolID → blocks (supports duplicates)
	applyPatchByFile map[string]*blocks.Block   // file path → latest APPLY_PATCH block (write_file)
	streamCh         chan string                // Content streaming channel
	streamMu         sync.Mutex                 // Protects streamCh
	streamCtx        context.Context
	streamCancel     context.CancelFunc
	thinkFilter      *output.ThinkFilter // Filter for <think> tags
	mu               sync.RWMutex        // Protects blockRegistry
}

// NewTUIMapper creates a new TUI event mapper.
// The mapper subscribes to core events and translates them into blocks
// that are appended to or updated in the UI timeline.
func NewTUIMapper(ui ports.UI) *TUIMapper {
	return &TUIMapper{
		ui:               ui,
		blockRegistry:    make(map[string][]*blocks.Block),
		applyPatchByFile: make(map[string]*blocks.Block),
		thinkFilter:      output.NewThinkFilter(),
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

	// Process event for status updates (Phase 1)
	if statusUI, ok := m.ui.(interface{ ProcessEvent(*Event) }); ok {
		statusUI.ProcessEvent(&event)
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

	// Create block based on tool type (block header provides compact feedback)
	block := m.createBlockForTool(data)
	if block == nil {
		// Unknown tool type, skip block creation
		return nil
	}

	// Atomically check and register (prevent race condition)
	m.mu.Lock()

	// Debug: log registry state
	slog.Debug("Checking blockRegistry for duplicate",
		"tool_id", data.ToolID,
		"registry_size", len(m.blockRegistry),
		"registry_contains", len(m.blockRegistry[data.ToolID]) > 0)

	if _, exists := m.blockRegistry[data.ToolID]; exists {
		// Duplicate tool ID from LLM - make block ID unique by appending counter
		// This is a workaround for LLM bugs that reuse tool IDs
		slog.Warn("Duplicate tool ID from LLM, making block ID unique",
			"tool_id", data.ToolID,
			"new_tool_name", data.ToolName,
			"existing_blocks_count", len(m.blockRegistry[data.ToolID]))

		// Find a unique block ID by appending -1, -2, etc.
		originalID := block.ID
		// Use the number of existing blocks as suffix to avoid collisions
		block.ID = fmt.Sprintf("%s-%d", originalID, len(m.blockRegistry[data.ToolID]))

		slog.Info("Created unique block ID for duplicate tool",
			"original_tool_id", data.ToolID,
			"new_block_id", block.ID)

		// Register duplicate block under the same tool ID for later updates
		m.blockRegistry[data.ToolID] = append(m.blockRegistry[data.ToolID], block)
	} else {
		// First time seeing this tool ID - register it normally
		m.blockRegistry[data.ToolID] = []*blocks.Block{block}
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

// createBlockForTool creates the appropriate block type for a tool call.
func (m *TUIMapper) createBlockForTool(data ToolCallStartData) *blocks.Block {
	switch data.ToolName {
	case "execute_command":
		return m.createExecuteBlock(data)
	case "read_file":
		return m.createReadBlock(data)
	case "write_file":
		return m.createOrReuseApplyPatchBlock(data)
	case "list_directory":
		return m.createExecuteBlock(data) // Treat as EXECUTE
	case "apply_patch":
		// Dedicated apply_patch tool (structured patch). Show the patch text as a diff.
		return m.createApplyPatchFromPatchTool(data)
	case "file_search":
		// Map file_search to GREP block type (files_with_matches style)
		return m.createGrepBlockFromSearch(data)
	case "git_context":
		// Map to NOTICE block; details will be filled on completion
		return m.createNoticeBlock(data, "Git Context")
	case "get_context":
		// Map to NOTICE block for environment context
		return m.createNoticeBlock(data, "Environment Context")
	default:
		return m.createToolBlock(data)
	}
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

	// Don't set block.Title for execute blocks - the renderer will use the
	// tool name (data.ToolName) and command from metadata. Setting Title
	// causes duplication in the block header.

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
		block.Meta = map[string]any{
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
		block.Meta = map[string]any{
			"file": path,
		}
	}

	return block
}

// createExecuteBlock creates an TOOL block for command execution.
func (m *TUIMapper) createToolBlock(data ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeTool)
	block.ID = data.ToolID

	// Extract command from parameters (or path for list_directory)
	command := extractString(data.Parameters, "tool_name")

	meta := &blocks.ToolMeta{
		ToolName: command,
	}
	if err := blocks.SetToolMeta(block, meta); err != nil {
		// Validation failed, set as raw map to preserve data
		block.Meta = map[string]any{
			"tool_name": command,
		}
	}
	return block
}

// createOrReuseApplyPatchBlock coalesces write_file blocks by file path.
// If a block for the same path exists, it updates and reuses it, and registers
// the current tool ID to point to that existing block for completion updates.
func (m *TUIMapper) createOrReuseApplyPatchBlock(data ToolCallStartData) *blocks.Block {
	path := extractString(data.Parameters, "path")
	content := extractString(data.Parameters, "content")

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.applyPatchByFile[path]; ok && existing != nil {
		// Map this tool ID to the existing block so completion updates it
		m.blockRegistry[data.ToolID] = append(m.blockRegistry[data.ToolID], existing)

		// Update body/title to reflect the latest write intent
		if content != "" {
			existing.Body = content
		}
		if existing.Title == "" {
			existing.Title = path
		}

		// Push in-place update; ignore error (best-effort)
		_ = m.ui.UpdateBlock(existing.ID, existing)

		// Returning nil tells caller not to append a new block
		return nil
	}

	// First write for this path: create a new block and remember it
	block := blocks.NewBlock(blocks.BlockTypeApplyPatch)
	block.ID = data.ToolID
	block.Title = path
	block.Body = content

	meta := &blocks.PatchMeta{File: path}
	if err := blocks.SetPatchMeta(block, meta); err != nil {
		block.Meta = map[string]any{"file": path}
	}

	m.applyPatchByFile[path] = block
	return block
}

// createApplyPatchFromPatchTool creates an APPLY_PATCH block for apply_patch tool.
// It renders the provided patch text as a diff and sets minimal metadata so that
// completion status can be displayed later.
func (m *TUIMapper) createApplyPatchFromPatchTool(data ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeApplyPatch)
	block.ID = data.ToolID

	// Title: show workspace root if provided, otherwise generic label
	workspaceRoot := extractString(data.Parameters, "workspace_root")
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	block.Title = workspaceRoot

	// Body: show the raw patch text so the user sees exactly what will be applied
	patchText := extractString(data.Parameters, "patch_text")
	block.Body = patchText

	// Metadata: PatchMeta requires a non-empty File; use workspace root as scope indicator
	meta := &blocks.PatchMeta{
		File:      workspaceRoot,
		Completed: false,
	}
	if err := blocks.SetPatchMeta(block, meta); err != nil {
		// Fallback to raw map if validation fails for any reason
		block.Meta = map[string]any{
			"file": workspaceRoot,
		}
	}

	return block
}

// createGrepBlockFromSearch creates a GREP block for file_search tool.
func (m *TUIMapper) createGrepBlockFromSearch(data ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeGrep)
	block.ID = data.ToolID

	query := extractString(data.Parameters, "query")
	// Map file_search semantics to GREP meta for consistent rendering
	meta := &blocks.GrepMeta{
		Pattern: query,
		Mode:    "files_with_matches",
	}
	if err := blocks.SetGrepMeta(block, meta); err != nil {
		block.Meta = map[string]any{
			"pattern": query,
			"mode":    "files_with_matches",
		}
	}

	return block
}

// createNoticeBlock creates a NOTICE block with the given title.
func (m *TUIMapper) createNoticeBlock(data ToolCallStartData, title string) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeNotice)
	block.ID = data.ToolID
	block.Title = title
	return block
}

// handleToolComplete updates an existing block when tool execution completes.
func (m *TUIMapper) handleToolComplete(event Event) error {
	data, ok := event.Data.(ToolCallCompleteData)
	if !ok {
		return nil
	}

	blocksToUpdate, exists := m.getBlocksForTool(data.ToolID)
	if !exists || len(blocksToUpdate) == 0 {
		return nil
	}

	var lastErr error
	for _, block := range blocksToUpdate {
		if block == nil {
			continue
		}

		if err := m.updateBlockWithToolResult(block, data); err != nil {
			lastErr = err
		}
	}

	m.cleanupToolRegistry(data.ToolID)
	return lastErr
}

// getBlocksForTool retrieves blocks associated with a tool ID.
func (m *TUIMapper) getBlocksForTool(toolID string) ([]*blocks.Block, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	blocksToUpdate, exists := m.blockRegistry[toolID]
	return blocksToUpdate, exists
}

// updateBlockWithToolResult updates a single block with tool execution results.
func (m *TUIMapper) updateBlockWithToolResult(block *blocks.Block, data ToolCallCompleteData) error {
	// Update block content
	m.updateBlockContent(block, data)

	// Update block severity
	m.updateBlockSeverity(block, data)

	// Update block metadata based on type
	m.updateBlockMetadata(block, data)

	// Update in UI
	return m.ui.UpdateBlock(block.ID, block)
}

// updateBlockContent sets the block body content.
func (m *TUIMapper) updateBlockContent(block *blocks.Block, data ToolCallCompleteData) {
	block.Body = data.Output
	if block.Body == "" && data.Error != "" {
		block.Body = "Error: " + data.Error
	}
}

// updateBlockSeverity sets the block severity based on success/failure.
func (m *TUIMapper) updateBlockSeverity(block *blocks.Block, data ToolCallCompleteData) {
	if !data.Success {
		block.Severity = blocks.SeverityError
		if data.Error != "" {
			block.Body += "\n\nError: " + data.Error
		}
	} else {
		block.Severity = blocks.SeverityInfo
	}
}

// updateBlockMetadata updates metadata based on block type.
func (m *TUIMapper) updateBlockMetadata(block *blocks.Block, data ToolCallCompleteData) {
	switch block.Type {
	case blocks.BlockTypeExecute:
		m.updateExecuteBlockMetadata(block, data)
	case blocks.BlockTypeApplyPatch:
		m.updatePatchBlockMetadata(block, data)
	}
}

// updateExecuteBlockMetadata updates metadata for EXECUTE blocks.
func (m *TUIMapper) updateExecuteBlockMetadata(block *blocks.Block, data ToolCallCompleteData) {
	meta, err := blocks.ParseExecuteMeta(block)
	if err != nil || meta == nil {
		return
	}

	if !data.Success {
		meta.ExitCode = intPtr(1)
	} else {
		meta.ExitCode = intPtr(0)
	}
	meta.LinesOut = countLinesPtr(block.Body)
	_ = blocks.SetExecuteMeta(block, meta)
}

// updatePatchBlockMetadata updates metadata for APPLY_PATCH blocks.
func (m *TUIMapper) updatePatchBlockMetadata(block *blocks.Block, data ToolCallCompleteData) {
	meta, err := blocks.ParsePatchMeta(block)
	if err != nil || meta == nil {
		return
	}

	meta.Succeeded = data.Success
	meta.Completed = true
	_ = blocks.SetPatchMeta(block, meta)
}

// cleanupToolRegistry removes tool ID from registry.
func (m *TUIMapper) cleanupToolRegistry(toolID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.blockRegistry, toolID)
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
	m.blockRegistry = make(map[string][]*blocks.Block)
	m.applyPatchByFile = make(map[string]*blocks.Block)
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
