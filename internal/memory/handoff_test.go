package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionHandoff(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewPersistentStore(tmpDir)
	require.NoError(t, err)

	handoff := NewSessionHandoff(store, nil)
	assert.NotNil(t, handoff)
}

func TestSessionHandoff_SaveAndLoad(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := NewPersistentStore(tmpDir)
	require.NoError(t, err)

	handoff := NewSessionHandoff(store, nil)

	// Save session.
	data := HandoffData{
		SessionID:    "test-session-123",
		Summary:      "Working on authentication feature",
		PendingTasks: []string{"Add unit tests", "Update documentation"},
		Decisions:    []string{"Use JWT for auth", "Use PostgreSQL"},
		WorkDir:      "/home/user/project",
		LastActivity: time.Now(),
		KeyReferences: map[string]string{
			"auth_file": "src/auth/handler.go",
		},
	}

	err = handoff.SaveSession(ctx, data)
	require.NoError(t, err)

	// Load session.
	loaded, err := handoff.LoadSession(ctx, "test-session-123")
	require.NoError(t, err)

	assert.Equal(t, data.SessionID, loaded.SessionID)
	assert.Equal(t, data.Summary, loaded.Summary)
	assert.Equal(t, data.PendingTasks, loaded.PendingTasks)
	assert.Equal(t, data.Decisions, loaded.Decisions)
	assert.Equal(t, data.WorkDir, loaded.WorkDir)
	assert.Equal(t, data.KeyReferences, loaded.KeyReferences)
}

func TestSessionHandoff_SaveSession_NoStore(t *testing.T) {
	ctx := context.Background()
	handoff := NewSessionHandoff(nil, nil)

	err := handoff.SaveSession(ctx, HandoffData{SessionID: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no persistent store")
}

func TestSessionHandoff_SaveSession_NoSessionID(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := NewPersistentStore(tmpDir)
	require.NoError(t, err)

	handoff := NewSessionHandoff(store, nil)

	err = handoff.SaveSession(ctx, HandoffData{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session ID is required")
}

func TestSessionHandoff_LoadSession_NotFound(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := NewPersistentStore(tmpDir)
	require.NoError(t, err)

	handoff := NewSessionHandoff(store, nil)

	_, err = handoff.LoadSession(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestSessionHandoff_ListSessions(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := NewPersistentStore(tmpDir)
	require.NoError(t, err)

	handoff := NewSessionHandoff(store, nil)

	// Save multiple sessions.
	for _, id := range []string{"session-1", "session-2", "session-3"} {
		err = handoff.SaveSession(ctx, HandoffData{
			SessionID: id,
			Summary:   "Test session " + id,
		})
		require.NoError(t, err)
	}

	// List sessions.
	sessions, err := handoff.ListSessions(ctx)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)
	assert.Contains(t, sessions, "session-1")
	assert.Contains(t, sessions, "session-2")
	assert.Contains(t, sessions, "session-3")
}

func TestSessionHandoff_DeleteSession(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, err := NewPersistentStore(tmpDir)
	require.NoError(t, err)

	handoff := NewSessionHandoff(store, nil)

	// Save session.
	err = handoff.SaveSession(ctx, HandoffData{
		SessionID: "to-delete",
		Summary:   "Test session",
	})
	require.NoError(t, err)

	// Delete session.
	err = handoff.DeleteSession(ctx, "to-delete")
	require.NoError(t, err)

	// Verify deleted.
	_, err = handoff.LoadSession(ctx, "to-delete")
	assert.Error(t, err)
}

func TestSessionHandoff_BuildContinuationPrompt(t *testing.T) {
	handoff := NewSessionHandoff(nil, nil)

	data := &HandoffData{
		SessionID:    "test-session",
		Summary:      "Working on authentication feature",
		PendingTasks: []string{"Add unit tests", "Update docs"},
		Decisions:    []string{"Use JWT for auth"},
		KeyReferences: map[string]string{
			"auth_file": "src/auth.go",
		},
		WorkDir:      "/home/user/project",
		LastActivity: time.Date(2025, 1, 20, 15, 30, 0, 0, time.UTC),
	}

	prompt := handoff.BuildContinuationPrompt(data)

	assert.Contains(t, prompt, "Continuing from previous session")
	assert.Contains(t, prompt, "Working on authentication feature")
	assert.Contains(t, prompt, "Add unit tests")
	assert.Contains(t, prompt, "Update docs")
	assert.Contains(t, prompt, "Use JWT for auth")
	assert.Contains(t, prompt, "auth_file")
	assert.Contains(t, prompt, "src/auth.go")
	assert.Contains(t, prompt, "/home/user/project")
	assert.Contains(t, prompt, "2025-01-20")
}

func TestSessionHandoff_BuildContinuationPrompt_Nil(t *testing.T) {
	handoff := NewSessionHandoff(nil, nil)
	prompt := handoff.BuildContinuationPrompt(nil)
	assert.Empty(t, prompt)
}

func TestSessionHandoff_BuildContinuationPrompt_Minimal(t *testing.T) {
	handoff := NewSessionHandoff(nil, nil)

	data := &HandoffData{
		SessionID: "test",
		Summary:   "Brief summary",
	}

	prompt := handoff.BuildContinuationPrompt(data)
	assert.Contains(t, prompt, "Continuing from previous session")
	assert.Contains(t, prompt, "Brief summary")
	assert.NotContains(t, prompt, "Pending tasks")
	assert.NotContains(t, prompt, "Key decisions")
}

func TestSimpleSummarizer(t *testing.T) {
	ctx := context.Background()

	summarizer := NewSimpleSummarizer(100)

	// Short content - no truncation.
	short := "This is short content."
	result, err := summarizer.Summarize(ctx, short, 0)
	require.NoError(t, err)
	assert.Equal(t, short, result)

	// Long content - truncated.
	long := "This is a very long piece of content that exceeds the maximum length " +
		"allowed by the summarizer and should be truncated with an ellipsis."
	result, err = summarizer.Summarize(ctx, long, 0)
	require.NoError(t, err)
	assert.Len(t, result, 100)
	assert.True(t, len(result) <= 100)
	assert.True(t, result[len(result)-3:] == "...")
}

func TestSimpleSummarizer_WithMaxTokens(t *testing.T) {
	ctx := context.Background()

	summarizer := NewSimpleSummarizer(1000)

	long := "This is content that will be limited by maxTokens parameter instead of the default limit."
	// maxTokens=10 -> ~40 chars.
	result, err := summarizer.Summarize(ctx, long, 10)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result), 40)
}

func TestSimpleSummarizer_DefaultMaxLength(t *testing.T) {
	summarizer := NewSimpleSummarizer(0)
	assert.Equal(t, 500, summarizer.maxLength)
}
