package acp

import (
	"os"
	"sync"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/pkg/alg/pathx"
)

// fileContentTracker tracks old file content for write_file operations.
// Used to generate diffs when file modifications complete.
type fileContentTracker struct {
	mu         sync.Mutex
	oldContent map[string]string // toolID -> old file content.
	filePaths  map[string]string // toolID -> file path.
	newContent map[string]string // toolID -> new file content (from parameters).
}

// newFileContentTracker creates a new file content tracker.
func newFileContentTracker() *fileContentTracker {
	return &fileContentTracker{
		oldContent: make(map[string]string),
		filePaths:  make(map[string]string),
		newContent: make(map[string]string),
	}
}

// storeOldContent stores old file content for a tool call.
// Returns true if content was stored, false if file doesn't exist or error occurred.
func (t *fileContentTracker) storeOldContent(toolID, filePath string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Read existing file content if file exists.
	content, err := os.ReadFile(filePath)
	if err != nil {
		// File doesn't exist or error reading - treat as new file (empty old content).
		t.oldContent[toolID] = ""
		t.filePaths[toolID] = filePath

		return true
	}

	t.oldContent[toolID] = string(content)
	t.filePaths[toolID] = filePath

	return true
}

// storeNewContent stores new file content from tool parameters.
func (t *fileContentTracker) storeNewContent(toolID, content string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.newContent[toolID] = content
}

// getContentForDiff retrieves old and new content for generating a diff.
// Returns oldContent, newContent, filePath, and true if content is available.
func (t *fileContentTracker) getContentForDiff(toolID string) (oldContent, newContent, filePath string, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	filePath, hasPath := t.filePaths[toolID]
	if !hasPath {
		return "", "", "", false
	}

	oldContent = t.oldContent[toolID] // Empty string if new file.

	newContent, hasNew := t.newContent[toolID]
	if !hasNew {
		return "", "", "", false
	}

	return oldContent, newContent, filePath, true
}

// cleanup removes tracked content for a tool call.
func (t *fileContentTracker) cleanup(toolID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.oldContent, toolID)
	delete(t.filePaths, toolID)
	delete(t.newContent, toolID)
}

// convertEventToSessionUpdate converts a Spin event to an ACP SessionUpdate.
// Returns the SessionUpdate and true if the event was converted, false otherwise.
// For EventContentDelta, may return multiple updates (thought chunks and message chunks).
// The fileContentTracker is optional - if nil, diff generation is skipped.
func convertEventToSessionUpdate(event events.Event, tracker *fileContentTracker, workDir string) (acp.SessionUpdate, bool) {
	switch event.Type {
	case events.EventContentDelta:
		return convertContentDelta(event)
	case events.EventToolCallStart:
		return convertToolCallStart(event, tracker, workDir)
	case events.EventToolCallProgress:
		return convertToolCallProgress(event)
	case events.EventToolCallComplete:
		return convertToolCallComplete(event, tracker)
	case events.EventSubagentSpawn, events.EventSubagentComplete,
		events.EventBackgroundTaskStarted, events.EventBackgroundTaskStopped,
		events.EventHookVeto:
		return convertA2AThought(event)
	default:
		// Event type not mapped to ACP notification.
		return acp.SessionUpdate{}, false
	}
}

// convertSystemEvent converts EventInfo/EventWarning to an ACP SessionUpdate.
// System events (like context compression notifications) are sent as agent messages
// with a special format that clients can recognize and display appropriately.
func convertSystemEvent(event events.Event) (acp.SessionUpdate, bool) {
	data, ok := event.SystemEventData()
	if !ok {
		return acp.SessionUpdate{}, false
	}

	// Format the system message with level prefix and details.
	var message string

	switch data.Level {
	case "warning", "warn":
		message = "[warning] " + data.Message
	case "error":
		message = "[error] " + data.Message
	default:
		message = "[info] " + data.Message
	}

	if data.Details != "" {
		message += " — " + data.Details
	}

	// Send as agent thought (dimmed/secondary display) to distinguish from main content.
	update := acp.UpdateAgentThoughtText(message + "\n")

	return update, true
}

// convertContentDelta converts EventContentDelta to agent_message_chunk.
func convertContentDelta(event events.Event) (acp.SessionUpdate, bool) {
	data, ok := event.ContentDeltaData()
	if !ok {
		return acp.SessionUpdate{}, false
	}

	// Only convert assistant content (agent messages).
	if data.Role != roleAssistant {
		return acp.SessionUpdate{}, false
	}

	// Use SDK helper to create agent message chunk.
	update := acp.UpdateAgentMessageText(data.Content)

	return update, true
}

// toolTitleParam maps tool names to the parameter used for the title suffix.
var toolTitleParam = map[string]string{
	"read_file":       "path",
	toolWriteFile:     "path",
	"edit_file":       "path",
	"apply_patch":     "path",
	"list_directory":  "path",
	"find_symbol":     "name",
	"find_references": "name",
	"file_search":     "pattern",
	"shell_command":   "command",
}

// buildToolCallTitle creates a descriptive title for a tool call.
func buildToolCallTitle(toolName string, params tools.ToolParameters) string {
	paramKey, hasParam := toolTitleParam[toolName]
	if !hasParam {
		return toolName
	}

	val, err := params.GetString(paramKey)
	if err != nil || val == "" {
		return toolName
	}

	// Shell commands use the command as the full title.
	if toolName == "shell_command" {
		return val
	}

	return toolName + ": " + val
}

// toolsWithPathLocation lists tools that use the "path" parameter for file location.
var toolsWithPathLocation = map[string]bool{
	"read_file":       true,
	toolWriteFile:     true,
	"edit_file":       true,
	"apply_patch":     true,
	"list_directory":  true,
	"find_symbol":     true,
	"find_references": true,
}

// extractFileLocations extracts file locations from tool parameters.
// Resolves relative paths against workDir to produce absolute paths for client navigation.
func extractFileLocations(toolName string, params tools.ToolParameters, workDir string) []acp.ToolCallLocation {
	if toolsWithPathLocation[toolName] {
		if path, err := params.GetString("path"); err == nil && path != "" {
			return []acp.ToolCallLocation{{Path: resolveToolPath(path, workDir)}}
		}
	}

	if toolName == "file_search" {
		if root, err := params.GetString("workspace_root"); err == nil && root != "" {
			return []acp.ToolCallLocation{{Path: resolveToolPath(root, workDir)}}
		}
	}

	return nil
}

// resolveToolPath resolves a relative path against workDir to produce an absolute path.
func resolveToolPath(path, workDir string) string {
	return pathx.ResolvePath(workDir, path)
}

// convertToolCallStart converts EventToolCallStart to tool_call.
// If tracker is provided and tool is write_file, stores old file content for diff generation.
func convertToolCallStart(event events.Event, tracker *fileContentTracker, workDir string) (acp.SessionUpdate, bool) {
	data, ok := event.ToolCallStartData()
	if !ok {
		return acp.SessionUpdate{}, false
	}

	// Use tool name as title; for shell_command, show the actual command.
	title := buildToolCallTitle(data.ToolName, data.Parameters)

	// Convert Spin tool ID to ACP ToolCallId.
	toolCallID := acp.ToolCallId(data.ToolID)

	// Map tool kind.
	kind := mapToolNameToKind(data.ToolName)

	// Extract file locations.
	locations := extractFileLocations(data.ToolName, data.Parameters, workDir)

	// Extract raw input (parameters as map).
	rawInput := data.Parameters.ToMap()

	// For write_file operations, track old file content for diff generation.
	if tracker != nil && data.ToolName == toolWriteFile {
		path, pathErr := data.Parameters.GetString("path")
		if pathErr == nil && path != "" {
			tracker.storeOldContent(data.ToolID, path)
			// Also store new content from parameters.
			content, contentErr := data.Parameters.GetString("content")
			if contentErr == nil {
				tracker.storeNewContent(data.ToolID, content)
			}
		}
	}

	// Build options.
	opts := []acp.ToolCallStartOpt{}
	if kind != nil {
		opts = append(opts, acp.WithStartKind(*kind))
	}

	if len(locations) > 0 {
		opts = append(opts, acp.WithStartLocations(locations))
	}

	if len(rawInput) > 0 {
		opts = append(opts, acp.WithStartRawInput(rawInput))
	}

	// Use SDK helper to create tool call start.
	update := acp.StartToolCall(toolCallID, title, opts...)

	// Ensure status is set to pending (StartToolCall might not set it by default).
	if update.ToolCall != nil && update.ToolCall.Status == "" {
		update.ToolCall.Status = acp.ToolCallStatusPending
	}

	return update, true
}

// convertToolCallProgress converts EventToolCallProgress to tool_call_update (in_progress).
func convertToolCallProgress(event events.Event) (acp.SessionUpdate, bool) {
	data, ok := event.ToolProgressData()
	if !ok {
		return acp.SessionUpdate{}, false
	}

	// Convert Spin tool ID to ACP ToolCallId.
	toolCallID := acp.ToolCallId(data.ToolID)

	// Use SDK helper with in_progress status.
	update := acp.UpdateToolCall(toolCallID, acp.WithUpdateStatus(acp.ToolCallStatusInProgress))

	return update, true
}

// generateUnifiedDiff generates a unified diff between old and new file content.
// Returns ACP ToolCallContent with diff, or nil if generation fails.
func generateUnifiedDiff(oldText, newText, filePath string) acp.ToolCallContent {
	// Use SDK helper to create diff content
	// If oldText is empty, it's a new file.
	if oldText == "" {
		return acp.ToolDiffContent(filePath, newText)
	}

	return acp.ToolDiffContent(filePath, newText, oldText)
}

// convertToolCallComplete converts EventToolCallComplete to tool_call_update (completed/failed).
// If tracker is provided and tool is write_file, generates diff and includes it in notification.
func convertToolCallComplete(event events.Event, tracker *fileContentTracker) (acp.SessionUpdate, bool) {
	data, ok := event.ToolCallCompleteData()
	if !ok {
		return acp.SessionUpdate{}, false
	}

	// Convert Spin tool ID to ACP ToolCallId.
	toolCallID := acp.ToolCallId(data.ToolID)

	// Determine status based on success.
	var status acp.ToolCallStatus
	if data.Success {
		status = acp.ToolCallStatusCompleted
	} else {
		status = acp.ToolCallStatusFailed
	}

	// Build options.
	opts := []acp.ToolCallUpdateOpt{
		acp.WithUpdateStatus(status),
	}

	// Build content array.
	var content []acp.ToolCallContent

	// For write_file operations, generate diff if tracker is available.
	if tracker != nil && data.ToolName == toolWriteFile {
		oldContent, newContent, filePath, hasContent := tracker.getContentForDiff(data.ToolID)
		if hasContent {
			// Generate diff content.
			diffContent := generateUnifiedDiff(oldContent, newContent, filePath)
			content = append(content, diffContent)
		}
		// Clean up tracked content.
		tracker.cleanup(data.ToolID)
	}

	// Check for terminal execution.
	if terminalID, isStr := data.Metadata["terminal_id"].(string); isStr && terminalID != "" {
		content = append(content, acp.ToolTerminalRef(terminalID))
	} else if data.Output != "" {
		// Wrap text output as a content block (only if not using terminal content).
		textBlock := acp.TextBlock(data.Output)
		content = append(content, acp.ToolContent(textBlock))
	}

	// Add content if we have any.
	if len(content) > 0 {
		opts = append(opts, acp.WithUpdateContent(content))
	}

	// Add raw output.
	rawOutput := map[string]any{
		"output":  data.Output,
		"success": data.Success,
	}
	if data.Error != "" {
		rawOutput["error"] = data.Error
	}

	opts = append(opts, acp.WithUpdateRawOutput(rawOutput))

	// Use SDK helper with all options.
	update := acp.UpdateToolCall(toolCallID, opts...)

	return update, true
}

const kindA2A = "kind=a2a"

func convertA2AThought(event events.Event) (acp.SessionUpdate, bool) {
	text := formatA2AThought(event)
	if text == "" {
		return acp.SessionUpdate{}, false
	}

	return acp.UpdateAgentThoughtText(text + "\n"), true
}

func formatA2AThought(event events.Event) string {
	if data, ok := event.TaskStateData(); ok {
		return kindA2A + " task=" + data.TaskID + " state=" + data.State
	}

	if data, ok := event.SubagentSpawnData(); ok {
		return kindA2A + " subagent=" + data.AgentType + " state=spawn"
	}

	if data, ok := event.SubagentCompleteData(); ok {
		return kindA2A + " subagent=" + data.AgentType + " state=complete"
	}

	if data, ok := event.HookVetoData(); ok {
		return kindA2A + " hook=" + data.Event + " reason=" + data.Reason
	}

	return ""
}
