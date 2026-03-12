// Package tui provides terminal UI mapping and rendering.
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
)

// Mapper maps core agent events to TUI blocks.
// It translates the event stream from the core agent into visual blocks
// that are displayed in the terminal UI timeline.
type Mapper struct {
	ui               ports.UI
	logger           *slog.Logger
	blockRegistry    map[string][]*blocks.Block // toolID → blocks (supports duplicates).
	applyPatchByFile map[string]*blocks.Block   // file path → latest APPLY_PATCH block (write_file).
	streamCh         chan string                // Content streaming channel.
	streamMu         sync.Mutex                 // Protects streamCh.
	streamCtx        context.Context
	streamCancel     context.CancelFunc
	// State for thinking blocks.
	thinking           bool
	thinkStart         time.Time
	thinkTokens        int
	mu                 sync.RWMutex    // Protects blockRegistry.
	lastBulletSet      map[string]bool // Track last retrieved bullet content (for deduplication).
	lastLearnedBullets map[string]bool // Track last learned bullet content (for deduplication).
	bulletMu           sync.Mutex      // Protects lastBulletSet and lastLearnedBullets.
}

// NewMapper creates a new TUI event mapper.
// The mapper subscribes to core events and translates them into blocks
// that are appended to or updated in the UI timeline.
func NewMapper(ui ports.UI) *Mapper {
	return &Mapper{
		ui:                 ui,
		logger:             slog.Default(),
		blockRegistry:      make(map[string][]*blocks.Block),
		applyPatchByFile:   make(map[string]*blocks.Block),
		lastBulletSet:      make(map[string]bool),
		lastLearnedBullets: make(map[string]bool),
	}
}

// MapEvent processes a core event and updates the TUI accordingly.
// It handles tool calls, content streaming, errors, and system messages.
// Returns an error only for critical failures; gracefully handles unexpected data.
func (m *Mapper) MapEvent(event events.Event) error {
	// Handle nil data gracefully.
	if event.Data == nil {
		return nil
	}

	// Process event for status updates (Phase 1).
	if statusUI, ok := m.ui.(interface{ ProcessEvent(*events.Event) }); ok {
		statusUI.ProcessEvent(&event)
	}

	switch event.Type {
	case events.EventToolCallStart:
		return m.handleToolStart(event)
	case events.EventToolCallComplete:
		return m.handleToolComplete(event)
	case events.EventContentDelta:
		return m.handleContentDelta(event)
	case events.EventThinkingDelta:
		return m.handleThinkingDelta(event)
	case events.EventContentComplete:
		return m.handleContentComplete(event)
	case events.EventACERetrieval:
		return m.handleACERetrieval(event)
	case events.EventACELearned:
		return m.handleACELearned(event)
	case events.EventError:
		return m.handleError(event)
	case events.EventInfo, events.EventWarning:
		return m.handleSystemEvent(event)
	default:
		// Ignore unknown events gracefully.
		return nil
	}
}

// handleToolStart creates a new block when a tool execution starts.
func (m *Mapper) handleToolStart(event events.Event) error {
	data, ok := event.Data.(events.ToolCallStartData)
	if !ok {
		return nil // Gracefully handle type assertion failure.
	}

	// Create block based on tool type (block header provides compact feedback).
	block := m.createBlockForTool(data)
	if block == nil {
		// Unknown tool type, skip block creation.
		return nil
	}

	// Atomically check and register (prevent race condition).
	m.mu.Lock()

	// Debug: log registry state.
	m.logger.DebugContext(context.Background(), "Checking blockRegistry for duplicate",
		"tool_id", data.ToolID,
		"registry_size", len(m.blockRegistry),
		"registry_contains", len(m.blockRegistry[data.ToolID]) > 0)

	if _, exists := m.blockRegistry[data.ToolID]; exists {
		// Duplicate tool ID from LLM - make block ID unique by appending counter
		// This is a workaround for LLM bugs that reuse tool IDs.
		m.logger.WarnContext(context.Background(), "Duplicate tool ID from LLM, making block ID unique",
			"tool_id", data.ToolID,
			"new_tool_name", data.ToolName,
			"existing_blocks_count", len(m.blockRegistry[data.ToolID]))

		// Find a unique block ID by appending -1, -2, etc.
		originalID := block.ID
		// Use the number of existing blocks as suffix to avoid collisions.
		block.ID = fmt.Sprintf("%s-%d", originalID, len(m.blockRegistry[data.ToolID]))

		m.logger.InfoContext(context.Background(), "Created unique block ID for duplicate tool",
			"original_tool_id", data.ToolID,
			"new_block_id", block.ID)

		// Register duplicate block under the same tool ID for updates.
		m.blockRegistry[data.ToolID] = append(m.blockRegistry[data.ToolID], block)
	} else {
		// First time seeing this tool ID - register it normally.
		m.blockRegistry[data.ToolID] = []*blocks.Block{block}
	}
	m.mu.Unlock()

	// Append to UI timeline.
	err := m.ui.AppendBlock(block)
	if err != nil {
		// Log the error with context for debugging.
		m.logger.ErrorContext(context.Background(), "Failed to append block to timeline",
			"error", err,
			"block_id", block.ID,
			"tool_id", data.ToolID,
			"tool_name", data.ToolName)

		return err
	}

	return nil
}

// createBlockForTool creates the appropriate block type for a tool call.
func (m *Mapper) createBlockForTool(data events.ToolCallStartData) *blocks.Block {
	switch data.ToolName {
	case "execute_command":
		return m.createExecuteBlock(data)
	case "read_file":
		return m.createReadBlock(data)
	case "write_file":
		return m.createOrReuseApplyPatchBlock(data)
	case "list_directory":
		return m.createExecuteBlock(data) // Treat as EXECUTE.
	case "apply_patch":
		// Dedicated apply_patch tool (structured patch). Show the patch text as a diff.
		return m.createApplyPatchFromPatchTool(data)
	case "file_search":
		// Map file_search to GREP block type (files_with_matches style).
		return m.createGrepBlockFromSearch(data)
	case "git_context":
		// Map to NOTICE block; details are filled on completion.
		return m.createNoticeBlock(data, "Git Context")
	case "get_context":
		// Map to NOTICE block for environment context.
		return m.createNoticeBlock(data, "Environment Context")
	default:
		return m.createToolBlock(data)
	}
}

// createExecuteBlock creates an EXECUTE block for command execution.
func (m *Mapper) createExecuteBlock(data events.ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeExecute)
	block.ID = data.ToolID

	// Extract command from parameters (or path for list_directory).
	command := extractString(data.Parameters, "command")
	if command == "" && data.ToolName == "list_directory" {
		// For list_directory, use the path as the command.
		command = "ls " + extractString(data.Parameters, "path")
	}

	// Don't set block.Title for execute blocks - the renderer will use the
	// tool name (data.ToolName) and command from metadata. Setting Title
	// causes duplication in the block header.

	// Store metadata.
	cwd := extractString(data.Parameters, "cwd")
	if cwd == "" {
		cwd = "."
	}

	meta := &blocks.ExecuteMeta{
		Command: command,
		CWD:     cwd,
		Impact:  "medium", // Default impact level.
	}
	err := blocks.SetExecuteMeta(block, meta)
	if err != nil {
		// Validation failed, marshal map to JSON to preserve data.
		fallback := map[string]any{
			"command": command,
			"cwd":     cwd,
		}
		data, marshalErr := json.Marshal(fallback)
		if marshalErr == nil {
			block.Meta = data
		}
	}

	return block
}

// createReadBlock creates a READ block for file reading.
func (m *Mapper) createReadBlock(data events.ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeRead)
	block.ID = data.ToolID

	path := extractString(data.Parameters, "path")
	block.Title = path

	meta := &blocks.ReadMeta{
		File:   path,
		Offset: extractIntValue(data.Parameters, "offset"),
		Limit:  extractIntValue(data.Parameters, "limit"),
	}
	err := blocks.SetReadMeta(block, meta)
	if err != nil {
		// Validation failed, marshal map to JSON.
		fallback := map[string]any{
			"file": path,
		}
		data, marshalErr := json.Marshal(fallback)
		if marshalErr == nil {
			block.Meta = data
		}
	}

	return block
}

// createExecuteBlock creates an TOOL block for command execution.
func (m *Mapper) createToolBlock(data events.ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeTool)
	block.ID = data.ToolID

	// Prefer the actual tool name from the event; fall back to param.
	toolName := data.ToolName
	if toolName == "" {
		toolName = extractString(data.Parameters, "tool_name")
	}

	// Convert parameters to a simple map for display (best-effort).
	params := data.Parameters.ToMap()

	meta := &blocks.ToolMeta{
		ToolName: toolName,
		Params:   params,
	}
	err := blocks.SetToolMeta(block, meta)
	if err != nil {
		// Validation failed, marshal map to JSON to preserve data.
		fallback := map[string]any{
			"tool_name": toolName,
		}
		data, marshalErr := json.Marshal(fallback)
		if marshalErr == nil {
			block.Meta = data
		}
	}

	return block
}

// createOrReuseApplyPatchBlock coalesces write_file blocks by file path.
// If a block for the same path exists, it updates and reuses it, and registers
// the current tool ID to point to that existing block for completion updates.
func (m *Mapper) createOrReuseApplyPatchBlock(data events.ToolCallStartData) *blocks.Block {
	path := extractString(data.Parameters, "path")
	content := extractString(data.Parameters, "content")

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.applyPatchByFile[path]; ok && existing != nil {
		// Map this tool ID to the existing block so completion updates it.
		m.blockRegistry[data.ToolID] = append(m.blockRegistry[data.ToolID], existing)

		// Update body/title to reflect the latest write intent.
		if content != "" {
			existing.Body = content
		}

		if existing.Title == "" {
			existing.Title = path
		}

		// Push in-place update; ignore error (best-effort).
		_ = m.ui.UpdateBlock(existing.ID, existing)

		// Returning nil tells caller not to append a new block.
		return nil
	}

	// First write for this path: create a new block and remember it.
	block := blocks.NewBlock(blocks.BlockTypeApplyPatch)
	block.ID = data.ToolID
	block.Title = path
	block.Body = content

	meta := &blocks.PatchMeta{File: path}
	err := blocks.SetPatchMeta(block, meta)
	if err != nil {
		data, marshalErr := json.Marshal(map[string]any{"file": path})
		if marshalErr == nil {
			block.Meta = data
		}
	}

	m.applyPatchByFile[path] = block

	return block
}

// createApplyPatchFromPatchTool creates an APPLY_PATCH block for apply_patch tool.
// It renders the provided patch text as a diff and sets minimal metadata so that
// completion status can be displayed.
func (m *Mapper) createApplyPatchFromPatchTool(data events.ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeApplyPatch)
	block.ID = data.ToolID

	// Title: show workspace root if provided, otherwise generic label.
	workspaceRoot := extractString(data.Parameters, "workspace_root")
	if workspaceRoot == "" {
		workspaceRoot = "."
	}

	block.Title = workspaceRoot

	// Body: show the raw patch text so the user sees exactly what is applied.
	patchText := extractString(data.Parameters, "patch_text")
	block.Body = patchText

	// Metadata: PatchMeta requires a non-empty File; use workspace root as scope indicator.
	meta := &blocks.PatchMeta{
		File:      workspaceRoot,
		Completed: false,
	}
	err := blocks.SetPatchMeta(block, meta)
	if err != nil {
		// Fallback to JSON if validation fails for any reason.
		fallback := map[string]any{
			"file": workspaceRoot,
		}
		data, marshalErr := json.Marshal(fallback)
		if marshalErr == nil {
			block.Meta = data
		}
	}

	return block
}

// createGrepBlockFromSearch creates a GREP block for file_search tool.
func (m *Mapper) createGrepBlockFromSearch(data events.ToolCallStartData) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeGrep)
	block.ID = data.ToolID

	query := extractString(data.Parameters, "query")
	// Map file_search semantics to GREP meta for consistent rendering.
	meta := &blocks.GrepMeta{
		Pattern: query,
		Mode:    "files_with_matches",
	}
	err := blocks.SetGrepMeta(block, meta)
	if err != nil {
		fallback := map[string]any{
			"pattern": query,
			"mode":    "files_with_matches",
		}
		data, marshalErr := json.Marshal(fallback)
		if marshalErr == nil {
			block.Meta = data
		}
	}

	return block
}

// createNoticeBlock creates a NOTICE block with the given title.
func (m *Mapper) createNoticeBlock(data events.ToolCallStartData, title string) *blocks.Block {
	block := blocks.NewBlock(blocks.BlockTypeNotice)
	block.ID = data.ToolID
	block.Title = title

	return block
}

// handleToolComplete updates an existing block when tool execution completes.
func (m *Mapper) handleToolComplete(event events.Event) error {
	data, ok := event.Data.(events.ToolCallCompleteData)
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

		err := m.updateBlockWithToolResult(block, data)
		if err != nil {
			lastErr = err
		}
	}

	m.cleanupToolRegistry(data.ToolID)

	return lastErr
}

// getBlocksForTool retrieves blocks associated with a tool ID.
func (m *Mapper) getBlocksForTool(toolID string) ([]*blocks.Block, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	blocksToUpdate, exists := m.blockRegistry[toolID]

	return blocksToUpdate, exists
}

// updateBlockWithToolResult updates a single block with tool execution results.
func (m *Mapper) updateBlockWithToolResult(block *blocks.Block, data events.ToolCallCompleteData) error {
	// Update block content.
	m.updateBlockContent(block, data)

	// Update block severity.
	m.updateBlockSeverity(block, data)

	// Update block metadata based on type.
	m.updateBlockMetadata(block, data)

	// Update in UI.
	return m.ui.UpdateBlock(block.ID, block)
}

// updateBlockContent sets the block body content.
func (m *Mapper) updateBlockContent(block *blocks.Block, data events.ToolCallCompleteData) {
	block.Body = data.Output
	if block.Body == "" && data.Error != "" {
		block.Body = "Error: " + data.Error
	}
}

// updateBlockSeverity sets the block severity based on success/failure.
func (m *Mapper) updateBlockSeverity(block *blocks.Block, data events.ToolCallCompleteData) {
	if !data.Success {
		block.Severity = blocks.SeverityError
		if data.Error != "" {
			// Avoid duplicating the error message in the body.
			// updateBlockContent may have already set block.Body to "Error: <msg>".
			if block.Body == "" {
				block.Body = "Error: " + data.Error
			} else if !strings.Contains(block.Body, data.Error) {
				block.Body += "\n\nError: " + data.Error
			}
		}
	} else {
		block.Severity = blocks.SeverityInfo
	}
}

// updateBlockMetadata updates metadata based on block type.
func (m *Mapper) updateBlockMetadata(block *blocks.Block, data events.ToolCallCompleteData) {
	switch block.Type {
	case blocks.BlockTypeExecute:
		m.updateExecuteBlockMetadata(block, data)
	case blocks.BlockTypeApplyPatch:
		m.updatePatchBlockMetadata(block, data)
	}
}

// updateExecuteBlockMetadata updates metadata for EXECUTE blocks.
func (m *Mapper) updateExecuteBlockMetadata(block *blocks.Block, data events.ToolCallCompleteData) {
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
func (m *Mapper) updatePatchBlockMetadata(block *blocks.Block, data events.ToolCallCompleteData) {
	meta, err := blocks.ParsePatchMeta(block)
	if err != nil || meta == nil {
		return
	}

	meta.Succeeded = data.Success
	meta.Completed = true
	_ = blocks.SetPatchMeta(block, meta)
}

// cleanupToolRegistry removes tool ID from registry.
func (m *Mapper) cleanupToolRegistry(toolID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.blockRegistry, toolID)
}

// handleContentDelta streams assistant content to the UI.
func (m *Mapper) handleContentDelta(event events.Event) error {
	data, ok := event.Data.(events.ContentDeltaData)
	if !ok || data.Role != "assistant" {
		return nil
	}

	// Check if we need to close a previous thinking block.
	m.checkCloseThinking()

	// If streaming is active, send chunk.
	m.streamMu.Lock()
	defer m.streamMu.Unlock()

	if m.streamCh != nil && data.Content != "" {
		select {
		case m.streamCh <- data.Content:
		case <-m.streamCtx.Done():
			// Stream closed, drop.
		default:
			// Channel full, drop (UI has coalescing).
		}
	}

	return nil
}

// handleThinkingDelta streams thinking content to the UI with dim formatting.
func (m *Mapper) handleThinkingDelta(event events.Event) error {
	data, ok := event.ThinkingDeltaData()
	if !ok {
		return nil
	}

	m.streamMu.Lock()
	defer m.streamMu.Unlock()

	// Start thinking block if not active.
	if !m.thinking {
		m.thinking = true
		m.thinkStart = time.Now()
		m.thinkTokens = 0
		// Send dim gray start code.
		if m.streamCh != nil {
			select {
			case m.streamCh <- "\x1b[2m\x1b[38;5;244m":
			default:
			}
		}
	}

	// Update metrics
	// Rough token estimation: whitespace-delimited words.
	for _, char := range data.Content {
		if char == ' ' || char == '\n' || char == '\t' {
			m.thinkTokens++
		}
	}

	// Send content.
	if m.streamCh != nil && data.Content != "" {
		select {
		case m.streamCh <- data.Content:
		default:
		}
	}

	return nil
}

// checkCloseThinking checks if we are currently in a thinking block and closes it if so.
// This prints the summary line and resets formatting.
func (m *Mapper) checkCloseThinking() {
	m.streamMu.Lock()
	defer m.streamMu.Unlock()

	if !m.thinking {
		return
	}

	duration := time.Since(m.thinkStart)
	m.thinking = false

	if m.streamCh != nil {
		// Reset dim gray formatting.
		var out strings.Builder
		out.WriteString("\x1b[0m") // Reset.

		// Summary.
		out.WriteString("\x1b[2m\x1b[38;5;242m")
		fmt.Fprintf(&out, " [thought for %.2fs, ~%d tokens]",
			duration.Seconds(), m.thinkTokens)
		out.WriteString("\x1b[0m\n")

		select {
		case m.streamCh <- out.String():
		default:
		}
	}
}

// handleContentComplete creates a NOTICE block for complete content messages.
// This is used for multi-line informational messages like ACE bullet lists.
func (m *Mapper) handleContentComplete(event events.Event) error {
	data, ok := event.Data.(events.ContentDeltaData)
	if !ok {
		return nil
	}

	// Create a NOTICE block to display the complete message.
	block := blocks.NewBlock(blocks.BlockTypeNotice)
	block.ID = generateBlockID()
	block.Body = data.Content
	block.Severity = blocks.SeverityInfo

	return m.ui.AppendBlock(block)
}

// handleError creates an ERROR block for error events.
func (m *Mapper) handleError(event events.Event) error {
	data, ok := event.Data.(events.ErrorData)
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
func (m *Mapper) handleSystemEvent(event events.Event) error {
	data, ok := event.Data.(events.SystemEventData)
	if !ok {
		return nil
	}

	block := blocks.NewBlock(blocks.BlockTypeNotice)
	block.ID = generateBlockID()
	block.Title = data.Message
	block.Body = data.Details

	// Map severity.
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

// handleACERetrieval formats and displays ACE bullets with special symbols and colors.
func (m *Mapper) handleACERetrieval(event events.Event) error {
	data, ok := event.ACERetrievalData()
	if !ok {
		return nil
	}

	// Build set of current bullet content and track new bullets.
	currentSet := make(map[string]bool)

	var newBullets []events.BulletData

	// Compare with last retrieved set to find truly new bullets.
	m.bulletMu.Lock()
	for _, bullet := range data.Bullets {
		currentSet[bullet.Content] = true
		if !m.lastBulletSet[bullet.Content] {
			newBullets = append(newBullets, bullet)
		}
	}

	// Update last bullet set for next comparison.
	m.lastBulletSet = currentSet
	m.bulletMu.Unlock()

	// Only show hint if there are truly new unique bullets.
	if len(newBullets) == 0 {
		return nil
	}

	// Build hint with actual bullet content.
	var hintText strings.Builder

	// Header.
	pluralS := "ies"
	if len(newBullets) == 1 {
		pluralS = "y"
	}

	fmt.Fprintf(&hintText, "\x1b[32m⟐\x1b[0m \x1b[90mRetrieved %d new strateg%s:\x1b[0m\n", len(newBullets), pluralS)

	// Show each new bullet.
	for _, bullet := range newBullets {
		// Truncate long bullets to first line for compact display.
		content := bullet.Content
		if idx := strings.Index(content, "\n"); idx != -1 {
			content = content[:idx] + "..."
		}
		// Limit to 120 chars per line.
		if len(content) > 120 {
			content = content[:117] + "..."
		}

		fmt.Fprintf(&hintText, "  \x1b[32m•\x1b[0m \x1b[90m%s\x1b[0m\n", content)
	}

	// Use PrintLine to show as a simple status message (not a block).
	_ = m.ui.PrintLine(hintText.String())

	return nil
}

// handleACELearned displays a compact hint when ACE learns new insights after execution.
// Only shows truly new learned bullets that haven't been displayed before.
func (m *Mapper) handleACELearned(event events.Event) error {
	data, ok := event.ACELearningData()
	if !ok {
		return nil
	}

	// Track new learned bullets separately from retrieved bullets.
	var newBullets []events.BulletData

	// Compare with last learned set to find truly new bullets.
	m.bulletMu.Lock()
	for _, bullet := range data.Bullets {
		if !m.lastLearnedBullets[bullet.Content] {
			newBullets = append(newBullets, bullet)
			m.lastLearnedBullets[bullet.Content] = true
		}
	}
	m.bulletMu.Unlock()

	// Only show hint if there are truly new unique learned bullets.
	if len(newBullets) == 0 {
		return nil
	}

	// Build hint with actual bullet content.
	var hintText strings.Builder

	// Header with success/failure indicator.
	pluralS := "s"
	if len(newBullets) == 1 {
		pluralS = ""
	}

	statusColor := "\x1b[32m" // green for success.
	statusText := "successful"

	if !data.Success {
		statusColor = "\x1b[33m" // yellow for failure.
		statusText = "failed"
	}

	fmt.Fprintf(&hintText, "\x1b[34m◆\x1b[0m \x1b[90mLearned %d new insight%s from %s%s\x1b[0m\x1b[90m execution:\x1b[0m\n",
		len(newBullets), pluralS, statusColor, statusText)

	// Show each new learned bullet.
	for _, bullet := range newBullets {
		// Truncate long bullets to first line for compact display.
		content := bullet.Content
		if idx := strings.Index(content, "\n"); idx != -1 {
			content = content[:idx] + "..."
		}
		// Limit to 120 chars per line.
		if len(content) > 120 {
			content = content[:117] + "..."
		}

		fmt.Fprintf(&hintText, "  \x1b[34m•\x1b[0m \x1b[90m%s\x1b[0m\n", content)
	}

	_ = m.ui.PrintLine(hintText.String())

	return nil
}

// StartStreaming initializes content streaming and returns the channel
// that will receive LLM content deltas. The caller should wire this to UI.PrintChunks.
func (m *Mapper) StartStreaming() <-chan string {
	m.streamMu.Lock()
	defer m.streamMu.Unlock()

	if m.streamCh != nil {
		return m.streamCh // Already started.
	}

	m.streamCh = make(chan string, 100)
	m.streamCtx, m.streamCancel = context.WithCancel(context.Background())

	return m.streamCh
}

// StopStreaming closes the content streaming channel.
func (m *Mapper) StopStreaming() {
	// Flush any open thinking block.
	m.checkCloseThinking()

	m.streamMu.Lock()
	defer m.streamMu.Unlock()

	if m.streamCancel != nil {
		m.streamCancel()
	}

	if m.streamCh != nil {
		close(m.streamCh)
		m.streamCh = nil
	}
}

// Close cleans up mapper resources (closes stream channels).
func (m *Mapper) Close() error {
	m.StopStreaming()

	m.mu.Lock()
	m.blockRegistry = make(map[string][]*blocks.Block)
	m.applyPatchByFile = make(map[string]*blocks.Block)
	m.mu.Unlock()

	return nil
}

// Helper functions.

// extractString safely extracts a string parameter from ToolCallArguments.
func extractString(params tools.ToolParameters, key string) string {
	return params.GetStringOr(key, "")
}

// extractIntValue safely extracts an int parameter from ToolCallArguments.
func extractIntValue(params tools.ToolParameters, key string) int {
	return params.GetIntOr(key, 0)
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
	// Simple counter-based ID
	// Use timestamp-based ID.
	return fmt.Sprintf("block-%d", eventIDCounter.Add(1))
}

// Simple atomic counter for block IDs (thread-safe).
var eventIDCounter = &atomicCounter{}

type atomicCounter struct {
	mu  sync.Mutex
	val int
}

// Add implements the Add operation.
func (c *atomicCounter) Add(delta int) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.val += delta

	return c.val
}
