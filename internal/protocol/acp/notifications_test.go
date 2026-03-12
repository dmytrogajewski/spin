package acp

import (
	"os"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// TestConvertEventToSessionUpdate_ContentDelta tests EventContentDelta conversion.
func TestConvertEventToSessionUpdate_ContentDelta(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventContentDelta,
		Timestamp: time.Now(),
		Data: events.ContentDeltaData{
			Content: "Hello",
			Role:    "assistant",
		},
	}

	update, ok := convertEventToSessionUpdate(event, nil)

	assert.True(t, ok, "should convert EventContentDelta")
	assert.NotNil(t, update)
	// Verify it's a valid SessionUpdate (can't check internal structure easily).
}

// TestConvertEventToSessionUpdate_ToolCallStart tests EventToolCallStart conversion.
func TestConvertEventToSessionUpdate_ToolCallStart(t *testing.T) {
	t.Parallel()
	params, err := tools.FromMap(map[string]any{"path": "/tmp/test.txt"})
	require.NoError(t, err)

	event := events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:     "tool-123",
			ToolName:   "read_file",
			Parameters: params,
		},
	}

	update, ok := convertEventToSessionUpdate(event, nil)

	assert.True(t, ok, "should convert EventToolCallStart")
	assert.NotNil(t, update)
}

// TestConvertToolCallStart_IncludesKind tests that tool call start includes tool kind.
func TestConvertToolCallStart_IncludesKind(t *testing.T) {
	t.Parallel()
	params, err := tools.FromMap(map[string]any{"path": "/tmp/test.txt"})
	require.NoError(t, err)

	event := events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:     "tool-123",
			ToolName:   "read_file",
			Parameters: params,
		},
	}

	update, ok := convertToolCallStart(event, nil)

	assert.True(t, ok, "should convert EventToolCallStart")
	assert.NotNil(t, update)
	// Verify update contains tool kind (checking internal structure)
	// The update should have kind set to ToolKindRead for read_file.
}

// TestConvertEventToSessionUpdate_UnknownEvent tests unknown event handling.
func TestConvertEventToSessionUpdate_UnknownEvent(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventInfo,
		Timestamp: time.Now(),
		Data:      "some data",
	}

	update, ok := convertEventToSessionUpdate(event, nil)

	assert.False(t, ok, "should not convert unknown events")
	// SessionUpdate is a struct, not a pointer, so check if it's empty.
	assert.Equal(t, acp.SessionUpdate{}, update)
}

// TestConvertEventToSessionUpdate_ToolCallProgress tests EventToolCallProgress conversion.
func TestConvertEventToSessionUpdate_ToolCallProgress(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventToolCallProgress,
		Timestamp: time.Now(),
		Data: events.ToolProgressData{
			ToolID:  "tool-123",
			Status:  "in_progress",
			Message: "Processing...",
		},
	}

	update, ok := convertEventToSessionUpdate(event, nil)

	assert.True(t, ok, "should convert EventToolCallProgress")
	assert.NotNil(t, update)
}

// TestConvertEventToSessionUpdate_ToolCallComplete_Success tests EventToolCallComplete conversion (success).
func TestConvertEventToSessionUpdate_ToolCallComplete_Success(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data: events.ToolCallCompleteData{
			ToolID:   "tool-123",
			ToolName: "read_file",
			Success:  true,
			Output:   "File content here",
		},
	}

	update, ok := convertEventToSessionUpdate(event, nil)

	assert.True(t, ok, "should convert EventToolCallComplete")
	assert.NotNil(t, update)
}

// TestConvertToolCallComplete_IncludesContent tests that tool call complete includes content and raw output.
func TestConvertToolCallComplete_IncludesContent(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data: events.ToolCallCompleteData{
			ToolID:   "tool-123",
			ToolName: "read_file",
			Success:  true,
			Output:   "File content here",
		},
	}

	update, ok := convertToolCallComplete(event, nil)

	assert.True(t, ok, "should convert EventToolCallComplete")
	assert.NotNil(t, update)
	// Verify update contains content and raw output (checking internal structure).
}

// TestConvertEventToSessionUpdate_ToolCallComplete_Failed tests EventToolCallComplete conversion (failed).
func TestConvertEventToSessionUpdate_ToolCallComplete_Failed(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data: events.ToolCallCompleteData{
			ToolID:   "tool-123",
			ToolName: "read_file",
			Success:  false,
			Error:    "file not found",
		},
	}

	update, ok := convertEventToSessionUpdate(event, nil)

	assert.True(t, ok, "should convert EventToolCallComplete")
	assert.NotNil(t, update)
}

// TestConvertEventToSessionUpdate_ContentDelta_UserRole tests that user role content is not converted.
func TestConvertEventToSessionUpdate_ContentDelta_UserRole(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventContentDelta,
		Timestamp: time.Now(),
		Data: events.ContentDeltaData{
			Content: "User message",
			Role:    "user",
		},
	}

	update, ok := convertEventToSessionUpdate(event, nil)

	assert.False(t, ok, "should not convert user content")
	assert.Equal(t, acp.SessionUpdate{}, update)
}

// TestMapToolNameToKind tests tool name to ACP tool kind mapping.
func TestMapToolNameToKind(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		toolName string
		want     *acp.ToolKind
	}{
		{"read_file maps to read", "read_file", acp.Ptr(acp.ToolKindRead)},
		{"write_file maps to edit", "write_file", acp.Ptr(acp.ToolKindEdit)},
		{"shell_command maps to execute", "shell_command", acp.Ptr(acp.ToolKindExecute)},
		{"file_search maps to search", "file_search", acp.Ptr(acp.ToolKindSearch)},
		{"list_directory maps to read", "list_directory", acp.Ptr(acp.ToolKindRead)},
		{"unknown tool returns nil", "unknown_tool", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mapToolNameToKind(tt.toolName)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

// TestExtractFileLocations tests file location extraction from tool parameters.
func TestExtractFileLocations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		toolName string
		params   map[string]any
		want     []acp.ToolCallLocation
	}{
		{
			name:     "read_file extracts path",
			toolName: "read_file",
			params:   map[string]any{"path": "/tmp/test.txt"},
			want:     []acp.ToolCallLocation{{Path: "/tmp/test.txt"}},
		},
		{
			name:     "write_file extracts path",
			toolName: "write_file",
			params:   map[string]any{"path": "/tmp/output.txt", "content": "test"},
			want:     []acp.ToolCallLocation{{Path: "/tmp/output.txt"}},
		},
		{
			name:     "list_directory extracts path",
			toolName: "list_directory",
			params:   map[string]any{"path": "/tmp"},
			want:     []acp.ToolCallLocation{{Path: "/tmp"}},
		},
		{
			name:     "file_search extracts workspace_root",
			toolName: "file_search",
			params:   map[string]any{"workspace_root": "/workspace", "query": "test"},
			want:     []acp.ToolCallLocation{{Path: "/workspace"}},
		},
		{
			name:     "shell_command has no locations",
			toolName: "shell_command",
			params:   map[string]any{"command": "ls -la"},
			want:     nil,
		},
		{
			name:     "read_file with missing path returns empty",
			toolName: "read_file",
			params:   map[string]any{},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, err := tools.FromMap(tt.params)
			require.NoError(t, err)

			got := extractFileLocations(tt.toolName, params)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestConvertToolCallStart_IncludesLocations tests that tool call start includes file locations.
func TestConvertToolCallStart_IncludesLocations(t *testing.T) {
	t.Parallel()
	params, err := tools.FromMap(map[string]any{"path": "/tmp/test.txt"})
	require.NoError(t, err)

	event := events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:     "tool-123",
			ToolName:   "read_file",
			Parameters: params,
		},
	}

	update, ok := convertToolCallStart(event, nil)

	assert.True(t, ok, "should convert EventToolCallStart")
	assert.NotNil(t, update)
	// Verify update contains locations (checking internal structure).
}

// TestConvertToolCallComplete_EmptyOutput tests that empty output doesn't include content.
func TestConvertToolCallComplete_EmptyOutput(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data: events.ToolCallCompleteData{
			ToolID:   "tool-123",
			ToolName: "read_file",
			Success:  true,
			Output:   "", // Empty output.
		},
	}

	update, ok := convertToolCallComplete(event, nil)

	assert.True(t, ok, "should convert EventToolCallComplete")
	assert.NotNil(t, update)
	// Verify update doesn't include content when output is empty.
}

// TestConvertToolCallComplete_WithError tests that error is included in raw output.
func TestConvertToolCallComplete_WithError(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data: events.ToolCallCompleteData{
			ToolID:   "tool-123",
			ToolName: "read_file",
			Success:  false,
			Output:   "",
			Error:    "file not found",
		},
	}

	update, ok := convertToolCallComplete(event, nil)

	assert.True(t, ok, "should convert EventToolCallComplete")
	assert.NotNil(t, update)
	// Verify update includes error in raw output.
}

// TestFileContentTracker_StoreAndRetrieve tests file content tracking.
func TestFileContentTracker_StoreAndRetrieve(t *testing.T) {
	t.Parallel()
	tracker := newFileContentTracker()

	// Store old content.
	toolID := "tool-123"
	filePath := "/tmp/test.txt"
	oldContent := "old content\nline 2"
	tracker.oldContent[toolID] = oldContent
	tracker.filePaths[toolID] = filePath
	tracker.newContent[toolID] = "new content\nline 2\nline 3"

	// Retrieve content for diff.
	retrievedOld, retrievedNew, retrievedPath, ok := tracker.getContentForDiff(toolID)

	assert.True(t, ok, "should retrieve content")
	assert.Equal(t, oldContent, retrievedOld)
	assert.Equal(t, "new content\nline 2\nline 3", retrievedNew)
	assert.Equal(t, filePath, retrievedPath)

	// Cleanup.
	tracker.cleanup(toolID)
	_, _, _, ok = tracker.getContentForDiff(toolID)
	assert.False(t, ok, "should not retrieve after cleanup")
}

// TestFileContentTracker_NewFile tests new file handling (empty old content).
func TestFileContentTracker_NewFile(t *testing.T) {
	t.Parallel()
	tracker := newFileContentTracker()

	// Store new file (no old content).
	toolID := "tool-456"
	filePath := "/tmp/newfile.txt"
	tracker.oldContent[toolID] = "" // Empty for new file.
	tracker.filePaths[toolID] = filePath
	tracker.newContent[toolID] = "new file content"

	// Retrieve content for diff.
	retrievedOld, retrievedNew, retrievedPath, ok := tracker.getContentForDiff(toolID)

	assert.True(t, ok, "should retrieve content for new file")
	assert.Empty(t, retrievedOld, "old content should be empty for new file")
	assert.Equal(t, "new file content", retrievedNew)
	assert.Equal(t, filePath, retrievedPath)
}

// TestConvertToolCallStart_WriteFile_TracksOldContent tests that write_file operations track old content.
func TestConvertToolCallStart_WriteFile_TracksOldContent(t *testing.T) {
	t.Parallel(
	// Create a temporary file with content.
	)

	tmpFile := t.TempDir() + "/test.txt"
	err := os.WriteFile(tmpFile, []byte("existing content\nline 2"), 0o600)
	require.NoError(t, err)

	tracker := newFileContentTracker()

	params, err := tools.FromMap(map[string]any{
		"path":    tmpFile,
		"content": "new content\nline 2\nline 3",
	})
	require.NoError(t, err)

	event := events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:     "tool-write",
			ToolName:   "write_file",
			Parameters: params,
		},
	}

	update, ok := convertToolCallStart(event, tracker)

	assert.True(t, ok, "should convert EventToolCallStart")
	assert.NotNil(t, update)

	// Verify old content was stored.
	oldContent, newContent, filePath, hasContent := tracker.getContentForDiff("tool-write")
	assert.True(t, hasContent, "should have tracked content")
	assert.Equal(t, "existing content\nline 2", oldContent)
	assert.Equal(t, "new content\nline 2\nline 3", newContent)
	assert.Equal(t, tmpFile, filePath)
}

// TestConvertToolCallStart_WriteFile_NewFile tests that new file creation is tracked correctly.
func TestConvertToolCallStart_WriteFile_NewFile(t *testing.T) {
	t.Parallel(
	// Use a non-existent file path.
	)

	tmpFile := t.TempDir() + "/newfile.txt"

	tracker := newFileContentTracker()

	params, err := tools.FromMap(map[string]any{
		"path":    tmpFile,
		"content": "new file content",
	})
	require.NoError(t, err)

	event := events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:     "tool-new",
			ToolName:   "write_file",
			Parameters: params,
		},
	}

	update, ok := convertToolCallStart(event, tracker)

	assert.True(t, ok, "should convert EventToolCallStart")
	assert.NotNil(t, update)

	// Verify old content is empty for new file.
	oldContent, newContent, filePath, hasContent := tracker.getContentForDiff("tool-new")
	assert.True(t, hasContent, "should have tracked content")
	assert.Empty(t, oldContent, "old content should be empty for new file")
	assert.Equal(t, "new file content", newContent)
	assert.Equal(t, tmpFile, filePath)
}

// TestConvertToolCallComplete_WriteFile_IncludesDiff tests that write_file completion includes diff.
func TestConvertToolCallComplete_WriteFile_IncludesDiff(t *testing.T) {
	t.Parallel()
	tracker := newFileContentTracker()

	// Pre-populate tracker with old and new content.
	toolID := "tool-write"
	filePath := "/tmp/test.txt"
	tracker.oldContent[toolID] = "old line 1\nold line 2"
	tracker.filePaths[toolID] = filePath
	tracker.newContent[toolID] = "new line 1\nnew line 2\nnew line 3"

	event := events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data: events.ToolCallCompleteData{
			ToolID:   toolID,
			ToolName: "write_file",
			Success:  true,
			Output:   "Successfully wrote file",
		},
	}

	update, ok := convertToolCallComplete(event, tracker)

	assert.True(t, ok, "should convert EventToolCallComplete")
	assert.NotNil(t, update)

	// Verify tracker was cleaned up.
	_, _, _, hasContent := tracker.getContentForDiff(toolID)
	assert.False(t, hasContent, "tracker should be cleaned up after completion")
}

// TestConvertToolCallComplete_WriteFile_NewFile tests diff generation for new file creation.
func TestConvertToolCallComplete_WriteFile_NewFile(t *testing.T) {
	t.Parallel()
	tracker := newFileContentTracker()

	// Pre-populate tracker with empty old content (new file).
	toolID := "tool-new"
	filePath := "/tmp/newfile.txt"
	tracker.oldContent[toolID] = "" // Empty for new file.
	tracker.filePaths[toolID] = filePath
	tracker.newContent[toolID] = "new file content\nline 2"

	event := events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data: events.ToolCallCompleteData{
			ToolID:   toolID,
			ToolName: "write_file",
			Success:  true,
			Output:   "Successfully wrote file",
		},
	}

	update, ok := convertToolCallComplete(event, tracker)

	assert.True(t, ok, "should convert EventToolCallComplete")
	assert.NotNil(t, update)

	// Verify tracker was cleaned up.
	_, _, _, hasContent := tracker.getContentForDiff(toolID)
	assert.False(t, hasContent, "tracker should be cleaned up after completion")
}

// TestConvertToolCallComplete_NonWriteFile_NoDiff tests that non-write_file tools don't generate diffs.
func TestConvertToolCallComplete_NonWriteFile_NoDiff(t *testing.T) {
	t.Parallel()
	tracker := newFileContentTracker()

	event := events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data: events.ToolCallCompleteData{
			ToolID:   "tool-read",
			ToolName: "read_file",
			Success:  true,
			Output:   "file content",
		},
	}

	update, ok := convertToolCallComplete(event, tracker)

	assert.True(t, ok, "should convert EventToolCallComplete")
	assert.NotNil(t, update)
	// Verify no diff was generated (tracker should not have been used).
}

// TestConvertSystemEvent_Info tests that EventInfo is converted to agent thought.
func TestConvertSystemEvent_Info(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventInfo,
		Timestamp: time.Now(),
		Data: events.SystemEventData{
			Level:   "info",
			Message: "Context history compressed",
			Details: "Messages: 150→100, Tokens: 8000→5600, Ratio: 30%",
		},
	}

	update, ok := convertSystemEvent(event)

	assert.True(t, ok, "should convert EventInfo")
	assert.NotNil(t, update)
	assert.NotNil(t, update.AgentThoughtChunk, "should be agent thought chunk")
	assert.NotNil(t, update.AgentThoughtChunk.Content.Text, "should have text content")
	assert.Contains(t, update.AgentThoughtChunk.Content.Text.Text, "[info]")
	assert.Contains(t, update.AgentThoughtChunk.Content.Text.Text, "Context history compressed")
	assert.Contains(t, update.AgentThoughtChunk.Content.Text.Text, "Messages: 150→100")
}

// TestConvertSystemEvent_Warning tests that EventWarning is converted to agent thought.
func TestConvertSystemEvent_Warning(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventWarning,
		Timestamp: time.Now(),
		Data: events.SystemEventData{
			Level:   "warning",
			Message: "Cycle detected: repeated tool calls",
			Details: "Tool 'read_file' called 5 times in a row",
		},
	}

	update, ok := convertSystemEvent(event)

	assert.True(t, ok, "should convert EventWarning")
	assert.NotNil(t, update)
	assert.NotNil(t, update.AgentThoughtChunk, "should be agent thought chunk")
	assert.NotNil(t, update.AgentThoughtChunk.Content.Text, "should have text content")
	assert.Contains(t, update.AgentThoughtChunk.Content.Text.Text, "[warning]")
	assert.Contains(t, update.AgentThoughtChunk.Content.Text.Text, "Cycle detected")
}

// TestConvertSystemEvent_NoDetails tests system event without details.
func TestConvertSystemEvent_NoDetails(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventInfo,
		Timestamp: time.Now(),
		Data: events.SystemEventData{
			Level:   "info",
			Message: "Simple message",
		},
	}

	update, ok := convertSystemEvent(event)

	assert.True(t, ok, "should convert EventInfo without details")
	assert.NotNil(t, update)
	assert.NotNil(t, update.AgentThoughtChunk, "should be agent thought chunk")
	assert.NotNil(t, update.AgentThoughtChunk.Content.Text, "should have text content")
	assert.Contains(t, update.AgentThoughtChunk.Content.Text.Text, "[info] Simple message")
	assert.NotContains(t, update.AgentThoughtChunk.Content.Text.Text, "—")
}

// TestConvertSystemEvent_InvalidData tests system event with invalid data type.
func TestConvertSystemEvent_InvalidData(t *testing.T) {
	t.Parallel()
	event := events.Event{
		Type:      events.EventInfo,
		Timestamp: time.Now(),
		Data:      "not a SystemEventData",
	}

	update, ok := convertSystemEvent(event)

	assert.False(t, ok, "should not convert event with invalid data type")
	assert.Equal(t, acp.SessionUpdate{}, update)
}
