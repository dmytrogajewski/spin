package conversation

// Journey: specs/journeys/JOURNEY-tui-resume.md.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/session"
)

func setupTestConvWithSessionDir(t *testing.T, sessionDir string) *Conversation {
	t.Helper()

	cfg := testConfig()
	cfg.Agent.SessionDir = sessionDir
	workDir := t.TempDir()
	rt, emitter, provider := createTestRuntime(t, workDir)

	conv, err := NewBuilder(cfg, workDir, rt, emitter, provider).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build conversation: %v", err)
	}

	return conv
}

func seedSessionTranscript(t *testing.T, sessionDir, id, workDir, userText string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Join(sessionDir, id), 0o750))

	msg := message.Message{
		Role:      message.RoleUser,
		Content:   userText,
		Timestamp: time.Now(),
		Tokens:    8,
	}
	line, err := json.Marshal(msg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(session.TranscriptPath(sessionDir, id), append(line, '\n'), 0o600))

	idx, err := session.NewSessionIndex(t.Context(), filepath.Join(sessionDir, "index.json"), nil)
	require.NoError(t, err)
	require.NoError(t, idx.Update(t.Context(), session.IndexEntry{
		ID:           id,
		WorkDir:      workDir,
		MessageCount: 1,
		LastModified: time.Now(),
	}))
}

func TestConversation_Resume_RestoresHistory(t *testing.T) {
	t.Parallel()

	sessionDir := t.TempDir()
	conv := setupTestConvWithSessionDir(t, sessionDir)
	priorID := "11111111-1111-4111-8111-111111111111"
	seedSessionTranscript(t, sessionDir, priorID, conv.GetWorkDir(), "continue the cat blink")

	require.NoError(t, conv.Resume(context.Background(), priorID))
	require.Equal(t, priorID, conv.ID())

	msgs := conv.GetHistoryMessages()
	require.Len(t, msgs, 1)
	require.Equal(t, "continue the cat blink", msgs[0].Content)
}

func TestConversation_Resume_RejectsCurrent(t *testing.T) {
	t.Parallel()

	conv := setupTestConvWithSessionDir(t, t.TempDir())
	err := conv.Resume(context.Background(), conv.ID())
	require.ErrorIs(t, err, session.ErrResumeAlreadyCurrent)
}

func TestConversation_Resume_MissingTranscript(t *testing.T) {
	t.Parallel()

	conv := setupTestConvWithSessionDir(t, t.TempDir())
	err := conv.Resume(context.Background(), "22222222-2222-4222-8222-222222222222")
	require.ErrorIs(t, err, session.ErrTranscriptNotFound)
}

func TestConversation_ListResumable_ExcludesSelf(t *testing.T) {
	t.Parallel()

	sessionDir := t.TempDir()
	conv := setupTestConvWithSessionDir(t, sessionDir)
	priorID := "33333333-3333-4333-8333-333333333333"
	seedSessionTranscript(t, sessionDir, priorID, conv.GetWorkDir(), "old chat")

	// Re-open the index the conversation already holds so the seeded
	// entry is visible (the live index was loaded before the seed).
	idx, err := session.NewSessionIndex(t.Context(), filepath.Join(sessionDir, "index.json"), nil)
	require.NoError(t, err)

	conv.sessionIndex = idx

	got := conv.ListResumableSessions(context.Background())
	require.Len(t, got, 1)
	require.Equal(t, priorID, got[0].ID)
}

func TestConversation_Resume_AppendsToSameTranscript(t *testing.T) {
	t.Parallel()

	sessionDir := t.TempDir()
	conv := setupTestConvWithSessionDir(t, sessionDir)
	priorID := "44444444-4444-4444-8444-444444444444"
	seedSessionTranscript(t, sessionDir, priorID, conv.GetWorkDir(), "seeded user")

	require.NoError(t, conv.Resume(context.Background(), priorID))
	require.NoError(t, conv.AddHistoryMessage(context.Background(), message.Message{
		Role:    message.RoleAssistant,
		Content: "resumed reply",
		Tokens:  3,
	}))

	if conv.transcriptWriter != nil {
		require.NoError(t, conv.transcriptWriter.Append(context.Background(), message.Message{
			Role:    message.RoleAssistant,
			Content: "resumed reply",
			Tokens:  3,
		}))
	}

	msgs, err := session.ReadTranscript(context.Background(), session.TranscriptPath(sessionDir, priorID))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(msgs), 2)
}
