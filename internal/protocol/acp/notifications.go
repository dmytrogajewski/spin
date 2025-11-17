package acp

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Thinking block tags used by the agent
// The agent uses <think> tags in the system prompt
const (
	thinkBlockStartTag = "<think>"
	thinkBlockEndTag   = "</think>"
)

// fileContentTracker tracks old file content for write_file operations.
// Used to generate diffs when file modifications complete.
type fileContentTracker struct {
	mu         sync.Mutex
	oldContent map[string]string // toolID -> old file content
	filePaths  map[string]string // toolID -> file path
	newContent map[string]string // toolID -> new file content (from parameters)
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

	// Read existing file content if file exists
	content, err := os.ReadFile(filePath)
	if err != nil {
		// File doesn't exist or error reading - treat as new file (empty old content)
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

	oldContent = t.oldContent[toolID] // Empty string if new file
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
func convertEventToSessionUpdate(event events.Event, tracker *fileContentTracker) (acp.SessionUpdate, bool) {
	switch event.Type {
	case events.EventContentDelta:
		return convertContentDelta(event)
	case events.EventToolCallStart:
		return convertToolCallStart(event, tracker)
	case events.EventToolCallProgress:
		return convertToolCallProgress(event)
	case events.EventToolCallComplete:
		return convertToolCallComplete(event, tracker)
	default:
		// Event type not mapped to ACP notification
		return acp.SessionUpdate{}, false
	}
}

// thinkingBlockTracker tracks thinking blocks across multiple content deltas.
type thinkingBlockTracker struct {
	buffer          strings.Builder
	inThink         bool
	thinkBuffer     strings.Builder
	sentMessageLen  int // Track how much message content we've already sent
	sentThinkingLen int // Track how much thinking content we've already sent
}

// newThinkingBlockTracker creates a new thinking block tracker.
func newThinkingBlockTracker() *thinkingBlockTracker {
	return &thinkingBlockTracker{}
}

// processContent processes a content delta and returns updates for thinking blocks and message chunks.
// Returns thinking update (if any), message update (if any), and whether any update was generated.
func (t *thinkingBlockTracker) processContent(content string) (thinkUpdate acp.SessionUpdate, messageUpdate acp.SessionUpdate, hasUpdate bool) {
	// Append to buffer to handle partial tags across chunks
	t.buffer.WriteString(content)
	bufferedContent := t.buffer.String()

	var messageParts []string
	var thinkingParts []string
	i := 0

	for i < len(bufferedContent) {
		if !t.inThink {
			// Look for opening tag <think>
			thinkStartIdx := strings.Index(bufferedContent[i:], thinkBlockStartTag)
			thinkStartLen := len(thinkBlockStartTag)

			if thinkStartIdx >= 0 {
				// Found opening tag - add everything before it as message content
				if thinkStartIdx > 0 {
					messageParts = append(messageParts, bufferedContent[i:i+thinkStartIdx])
				}
				t.inThink = true
				i += thinkStartIdx + thinkStartLen
				continue
			}

			// No opening tag found - all remaining content is message content
			if i < len(bufferedContent) {
				messageParts = append(messageParts, bufferedContent[i:])
			}
			break
		} else {
			// Inside thinking block - look for closing tag
			thinkEndIdx := strings.Index(bufferedContent[i:], thinkBlockEndTag)
			thinkEndLen := len(thinkBlockEndTag)

			if thinkEndIdx >= 0 {
				// Found closing tag - add everything before it as thinking content
				t.thinkBuffer.WriteString(bufferedContent[i : i+thinkEndIdx])
				thinkingParts = append(thinkingParts, t.thinkBuffer.String())
				t.thinkBuffer.Reset()
				t.inThink = false
				i += thinkEndIdx + thinkEndLen
				continue
			}

			// No closing tag found - buffer the content for next chunk
			t.thinkBuffer.WriteString(bufferedContent[i:])
			break
		}
	}

	// Reset buffer if we processed everything
	if i >= len(bufferedContent) {
		t.buffer.Reset()
	} else if !t.inThink {
		// Keep unprocessed content in buffer if we're not in a thinking block
		remaining := bufferedContent[i:]
		t.buffer.Reset()
		t.buffer.WriteString(remaining)
	} else {
		// In thinking block - clear buffer as content is in thinkBuffer
		t.buffer.Reset()
	}

	// Create thinking update if we have new thinking content
	if len(thinkingParts) > 0 {
		thinkingContent := strings.Join(thinkingParts, "\n\n")
		// Only send the new portion (delta) of thinking content
		if len(thinkingContent) > t.sentThinkingLen {
			newThinkingContent := thinkingContent[t.sentThinkingLen:]
			thinkUpdate = acp.UpdateAgentThoughtText(newThinkingContent)
			t.sentThinkingLen = len(thinkingContent)
			hasUpdate = true
		}
	}

	// Create message update if we have new message content
	if len(messageParts) > 0 {
		messageContent := strings.Join(messageParts, "")
		// Only send the new portion (delta) of message content
		if len(messageContent) > t.sentMessageLen {
			newMessageContent := messageContent[t.sentMessageLen:]
			if newMessageContent != "" {
				messageUpdate = acp.UpdateAgentMessageText(newMessageContent)
				t.sentMessageLen = len(messageContent)
				hasUpdate = true
			}
		}
	}

	return thinkUpdate, messageUpdate, hasUpdate
}

// convertContentDelta converts EventContentDelta to agent_message_chunk and agent_thought_chunk.
// Uses a tracker to handle thinking blocks that span multiple content deltas.
func convertContentDelta(event events.Event) (acp.SessionUpdate, bool) {
	data, ok := event.ContentDeltaData()
	if !ok {
		return acp.SessionUpdate{}, false
	}

	// Only convert assistant content (agent messages)
	if data.Role != "assistant" {
		return acp.SessionUpdate{}, false
	}

	// For now, return message chunk - thinking block extraction will be handled
	// in processEvents with stateful tracker
	// Use SDK helper to create agent message chunk
	update := acp.UpdateAgentMessageText(data.Content)
	return update, true
}

// mapToolNameToKind maps Spin tool names to ACP tool kinds.
// Returns nil if the tool name is not recognized.
func mapToolNameToKind(toolName string) *acp.ToolKind {
	switch toolName {
	case "read_file":
		return acp.Ptr(acp.ToolKindRead)
	case "write_file":
		return acp.Ptr(acp.ToolKindEdit)
	case "shell_command":
		return acp.Ptr(acp.ToolKindExecute)
	case "file_search":
		return acp.Ptr(acp.ToolKindSearch)
	case "list_directory":
		return acp.Ptr(acp.ToolKindRead)
	default:
		return nil // No kind specified for unknown tools
	}
}

// extractFileLocations extracts file locations from tool parameters.
// Returns a slice of ToolCallLocation objects with file paths.
func extractFileLocations(toolName string, params tools.ToolParameters) []acp.ToolCallLocation {
	var locations []acp.ToolCallLocation

	switch toolName {
	case "read_file", "write_file", "list_directory":
		if path, err := params.GetString("path"); err == nil && path != "" {
			locations = append(locations, acp.ToolCallLocation{
				Path: path,
			})
		}
	case "file_search":
		if root, err := params.GetString("workspace_root"); err == nil && root != "" {
			locations = append(locations, acp.ToolCallLocation{
				Path: root,
			})
		}
	}

	return locations
}

// convertToolCallStart converts EventToolCallStart to tool_call.
// If tracker is provided and tool is write_file, stores old file content for diff generation.
func convertToolCallStart(event events.Event, tracker *fileContentTracker) (acp.SessionUpdate, bool) {
	data, ok := event.ToolCallStartData()
	if !ok {
		return acp.SessionUpdate{}, false
	}

	// Use tool name as title
	title := data.ToolName

	// Convert Spin tool ID to ACP ToolCallId
	toolCallID := acp.ToolCallId(data.ToolID)

	// Map tool kind
	kind := mapToolNameToKind(data.ToolName)

	// Extract file locations
	locations := extractFileLocations(data.ToolName, data.Parameters)

	// Extract raw input (parameters as map)
	rawInput := data.Parameters.ToMap()

	// For write_file operations, track old file content for diff generation
	if tracker != nil && data.ToolName == "write_file" {
		if path, err := data.Parameters.GetString("path"); err == nil && path != "" {
			tracker.storeOldContent(data.ToolID, path)
			// Also store new content from parameters
			if content, err := data.Parameters.GetString("content"); err == nil {
				tracker.storeNewContent(data.ToolID, content)
			}
		}
	}

	// Build options
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

	// Use SDK helper to create tool call start
	update := acp.StartToolCall(toolCallID, title, opts...)
	return update, true
}

// convertToolCallProgress converts EventToolCallProgress to tool_call_update (in_progress).
func convertToolCallProgress(event events.Event) (acp.SessionUpdate, bool) {
	data, ok := event.ToolProgressData()
	if !ok {
		return acp.SessionUpdate{}, false
	}

	// Convert Spin tool ID to ACP ToolCallId
	toolCallID := acp.ToolCallId(data.ToolID)

	// Use SDK helper with in_progress status
	update := acp.UpdateToolCall(toolCallID, acp.WithUpdateStatus(acp.ToolCallStatusInProgress))
	return update, true
}

// generateUnifiedDiff generates a unified diff between old and new file content.
// Returns ACP ToolCallContent with diff, or nil if generation fails.
func generateUnifiedDiff(oldText, newText, filePath string) acp.ToolCallContent {
	// Use SDK helper to create diff content
	// If oldText is empty, it's a new file
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

	// Convert Spin tool ID to ACP ToolCallId
	toolCallID := acp.ToolCallId(data.ToolID)

	// Determine status based on success
	var status acp.ToolCallStatus
	if data.Success {
		status = acp.ToolCallStatusCompleted
	} else {
		status = acp.ToolCallStatusFailed
	}

	// Build options
	opts := []acp.ToolCallUpdateOpt{
		acp.WithUpdateStatus(status),
	}

	// Build content array
	var content []acp.ToolCallContent

	// For write_file operations, generate diff if tracker is available
	if tracker != nil && data.ToolName == "write_file" {
		oldContent, newContent, filePath, hasContent := tracker.getContentForDiff(data.ToolID)
		if hasContent {
			// Generate diff content
			diffContent := generateUnifiedDiff(oldContent, newContent, filePath)
			content = append(content, diffContent)
		}
		// Clean up tracked content
		tracker.cleanup(data.ToolID)
	}

	// Add text output if available (alongside or instead of diff)
	if data.Output != "" {
		// Wrap text output as a content block
		textBlock := acp.TextBlock(data.Output)
		content = append(content, acp.ToolContent(textBlock))
	}

	// Add content if we have any
	if len(content) > 0 {
		opts = append(opts, acp.WithUpdateContent(content))
	}

	// Add raw output
	rawOutput := map[string]interface{}{
		"output":  data.Output,
		"success": data.Success,
	}
	if data.Error != "" {
		rawOutput["error"] = data.Error
	}
	opts = append(opts, acp.WithUpdateRawOutput(rawOutput))

	// Use SDK helper with all options
	update := acp.UpdateToolCall(toolCallID, opts...)
	return update, true
}

// detectPlanFromOutput detects plan-like structures in agent output text.
// This is a basic implementation that looks for common plan patterns:
// - Numbered lists (1., 2., 3., etc.)
// - Bullet lists with plan-like content
// - "Plan:" or "Steps:" headers followed by list items
// - Task-like descriptions
//
// Returns ACP PlanEntry objects for detected plan items.
// Full plan system integration is deferred to Feature 9.2.
func detectPlanFromOutput(output string) []acp.PlanEntry {
	if output == "" {
		return nil
	}

	var entries []acp.PlanEntry
	lines := strings.Split(output, "\n")

	// Look for plan patterns
	var inPlanSection bool
	var currentPlanPrefix string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this line starts a plan section
		lowerLine := strings.ToLower(line)
		if strings.HasPrefix(lowerLine, "plan:") || strings.HasPrefix(lowerLine, "steps:") ||
			strings.HasPrefix(lowerLine, "task:") || strings.HasPrefix(lowerLine, "tasks:") {
			inPlanSection = true
			currentPlanPrefix = ""
			continue
		}

		// Skip if not in a plan section and no plan-like patterns detected
		if !inPlanSection {
			// Check for numbered list pattern (1., 2., 3., etc.) or bullet points
			if matchesPlanPattern(line) {
				inPlanSection = true
			} else {
				continue
			}
		}

		// Extract plan entry from line
		entry := extractPlanEntry(line, currentPlanPrefix)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries
}

// matchesPlanPattern checks if a line matches common plan patterns.
func matchesPlanPattern(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	// Check for numbered list (1., 2., 3., etc.)
	if len(line) >= 2 && line[0] >= '1' && line[0] <= '9' && (line[1] == '.' || line[1] == ')') {
		return true
	}
	// Check for bullet points
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		return true
	}
	return false
}

// extractPlanEntry extracts a plan entry from a line of text.
func extractPlanEntry(line, prefix string) *acp.PlanEntry {
	// Remove common list prefixes
	content := line
	content = strings.TrimPrefix(content, "- ")
	content = strings.TrimPrefix(content, "* ")
	// Remove numbered prefixes (1., 2., etc.)
	for i := 1; i <= 9; i++ {
		numberedPrefix := fmt.Sprintf("%d.", i)
		if strings.HasPrefix(content, numberedPrefix) {
			content = strings.TrimPrefix(content, numberedPrefix)
			break
		}
		parenPrefix := fmt.Sprintf("%d)", i)
		if strings.HasPrefix(content, parenPrefix) {
			content = strings.TrimPrefix(content, parenPrefix)
			break
		}
	}
	content = strings.TrimSpace(content)

	if content == "" {
		return nil
	}

	// Add prefix if provided
	if prefix != "" {
		content = prefix + " " + content
	}

	// Determine priority (basic heuristics)
	priority := acp.PlanEntryPriorityMedium
	lowerContent := strings.ToLower(content)
	if strings.Contains(lowerContent, "critical") || strings.Contains(lowerContent, "urgent") ||
		strings.Contains(lowerContent, "important") || strings.Contains(lowerContent, "priority") {
		priority = acp.PlanEntryPriorityHigh
	} else if strings.Contains(lowerContent, "optional") || strings.Contains(lowerContent, "nice to have") {
		priority = acp.PlanEntryPriorityLow
	}

	// Create plan entry
	entry := acp.PlanEntry{
		Content:  content,
		Priority: priority,
		Status:   acp.PlanEntryStatusPending,
	}

	return &entry
}
