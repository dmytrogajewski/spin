package session_test

// Journey: specs/journeys/JOURNEY-2.3.md.

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/session"
)

func TestSessionPersistence_IndexSurvivesReopen(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := indexPath(dir)

	now := time.Now().Truncate(time.Second)

	// Create first index and populate it.
	idx1, err := session.NewSessionIndex(t.Context(), path, nil)
	require.NoError(t, err)

	require.NoError(t, idx1.Update(t.Context(), testEntry(testSessionID1, "Session Alpha", testWorkDirA, 10, now)))
	require.NoError(t, idx1.Update(t.Context(), testEntry(testSessionID2, "Session Beta", testWorkDirB, 20, now.Add(time.Hour))))

	// Create a NEW index from the same path — simulates process restart.
	idx2, err := session.NewSessionIndex(t.Context(), path, nil)
	require.NoError(t, err)

	entries := idx2.List("")
	require.Len(t, entries, 2, "reopened index should contain both entries")

	// Verify ordering (newest first) and field integrity.
	require.Equal(t, testSessionID2, entries[0].ID, "newest session should be first")
	require.Equal(t, "Session Beta", entries[0].Title)
	require.Equal(t, 20, entries[0].MessageCount)
	require.Equal(t, testWorkDirB, entries[0].WorkDir)

	require.Equal(t, testSessionID1, entries[1].ID)
	require.Equal(t, "Session Alpha", entries[1].Title)
	require.Equal(t, 10, entries[1].MessageCount)
	require.Equal(t, testWorkDirA, entries[1].WorkDir)
}

func TestSessionPersistence_TranscriptWriteAndRead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	// Write multiple transcript entries.
	writer1, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	msgs := []message.Message{
		{
			ID:        "msg-persist-1",
			Role:      message.RoleUser,
			Content:   "first user message",
			Timestamp: time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC),
			Tokens:    15,
		},
		{
			ID:        "msg-persist-2",
			Role:      message.RoleAssistant,
			Content:   "assistant reply",
			Timestamp: time.Date(2024, 6, 1, 10, 0, 1, 0, time.UTC),
			Tokens:    25,
			ToolCalls: []message.ToolCall{
				{
					ID:   "call-persist-1",
					Type: "function",
					Function: message.ToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"/tmp/example.go"}`,
					},
				},
			},
		},
		{
			ID:         "msg-persist-3",
			Role:       message.RoleTool,
			Content:    "file contents here",
			Timestamp:  time.Date(2024, 6, 1, 10, 0, 2, 0, time.UTC),
			ToolCallID: "call-persist-1",
		},
	}

	for _, m := range msgs {
		require.NoError(t, writer1.Append(t.Context(), m))
	}

	require.Equal(t, len(msgs), writer1.Count())

	// Close the writer — simulates session end.
	require.NoError(t, writer1.Close())

	// Reopen with a new writer and read back — simulates session resume.
	writer2, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	defer writer2.Close()

	readBack, readErr := writer2.ReadAll(t.Context())
	require.NoError(t, readErr)
	require.Len(t, readBack, len(msgs), "all written messages should be readable after reopen")

	// Verify each message field-by-field.
	for i, expected := range msgs {
		actual := readBack[i]
		require.Equal(t, expected.ID, actual.ID, "message %d ID mismatch", i)
		require.Equal(t, expected.Role, actual.Role, "message %d Role mismatch", i)
		require.Equal(t, expected.Content, actual.Content, "message %d Content mismatch", i)
		require.Equal(t, expected.Tokens, actual.Tokens, "message %d Tokens mismatch", i)
		require.Equal(t, expected.ToolCallID, actual.ToolCallID, "message %d ToolCallID mismatch", i)
		require.Len(t, actual.ToolCalls, len(expected.ToolCalls), "message %d ToolCalls count mismatch", i)
	}

	// Verify tool call details on the assistant message.
	require.Equal(t, "read_file", readBack[1].ToolCalls[0].Function.Name)
	require.JSONEq(t, `{"path":"/tmp/example.go"}`, readBack[1].ToolCalls[0].Function.Arguments)
}

func TestSessionPersistence_IndexAndTranscriptTogether(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	idxPath := filepath.Join(dir, testIndexFile)
	txPath := filepath.Join(dir, "full-flow.transcript.jsonl")

	sessionID := "sess-full-flow"
	sessionTitle := "Full Flow Session"
	now := time.Now().Truncate(time.Second)

	// Phase 1: Create session in index and write transcript.

	idx1, err := session.NewSessionIndex(t.Context(), idxPath, nil)
	require.NoError(t, err)

	writer1, err := session.NewTranscriptWriter(txPath)
	require.NoError(t, err)

	const messageCount = 5

	for i := range messageCount {
		msg := message.Message{
			ID:        fmt.Sprintf("msg-flow-%d", i),
			Role:      message.RoleUser,
			Content:   fmt.Sprintf("message number %d", i),
			Timestamp: now.Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, writer1.Append(t.Context(), msg))
	}

	// Record session in index with the final message count.
	entry := session.IndexEntry{
		ID:           sessionID,
		Title:        sessionTitle,
		MessageCount: writer1.Count(),
		LastModified: now.Add(time.Duration(messageCount-1) * time.Second),
		WorkDir:      testWorkDirA,
	}
	require.NoError(t, idx1.Update(t.Context(), entry))

	// Close writer — simulates session end.
	require.NoError(t, writer1.Close())

	// Phase 2: Reopen both and verify consistency.

	idx2, err := session.NewSessionIndex(t.Context(), idxPath, nil)
	require.NoError(t, err)

	entries := idx2.List("")
	require.Len(t, entries, 1, "index should have exactly one session")
	require.Equal(t, sessionID, entries[0].ID)
	require.Equal(t, sessionTitle, entries[0].Title)
	require.Equal(t, messageCount, entries[0].MessageCount)
	require.Equal(t, testWorkDirA, entries[0].WorkDir)

	writer2, err := session.NewTranscriptWriter(txPath)
	require.NoError(t, err)

	defer writer2.Close()

	readBack, readErr := writer2.ReadAll(t.Context())
	require.NoError(t, readErr)

	// Transcript message count should match what the index recorded.
	require.Len(t, readBack, entries[0].MessageCount,
		"transcript message count should match index MessageCount")

	// Verify ordering and content integrity.
	for i, msg := range readBack {
		require.Equal(t, fmt.Sprintf("msg-flow-%d", i), msg.ID)
		require.Equal(t, fmt.Sprintf("message number %d", i), msg.Content)
		require.Equal(t, message.RoleUser, msg.Role)
	}
}
