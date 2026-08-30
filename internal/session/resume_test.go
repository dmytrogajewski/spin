package session_test

// Journey: specs/journeys/JOURNEY-tui-resume.md.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/pkg/ui/textwidth"
)

const (
	resumeIDAlpha = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	resumeIDBeta  = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	resumeWorkDir = "/home/dev/spin"
)

func writeTranscript(t *testing.T, sessionDir, id, userText string, when time.Time) {
	t.Helper()

	dir := filepath.Join(sessionDir, id)
	require.NoError(t, os.MkdirAll(dir, 0o750))

	msg := message.Message{
		Role:      message.RoleUser,
		Content:   userText,
		Timestamp: when,
		Tokens:    4,
	}
	line, err := json.Marshal(msg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		session.TranscriptPath(sessionDir, id),
		append(line, '\n'),
		0o600,
	))
}

func seedResumeIndex(t *testing.T, sessionDir string, entries ...session.IndexEntry) *session.Index {
	t.Helper()

	idx, err := session.NewSessionIndex(t.Context(), filepath.Join(sessionDir, "index.json"), nil)
	require.NoError(t, err)

	for _, entry := range entries {
		require.NoError(t, idx.Update(t.Context(), entry))
	}

	return idx
}

func TestReadTranscript_Missing(t *testing.T) {
	t.Parallel()

	_, err := session.ReadTranscript(t.Context(), filepath.Join(t.TempDir(), "nope.jsonl"))
	require.ErrorIs(t, err, session.ErrTranscriptNotFound)
}

func TestReadTranscript_LoadsMessages(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	when := time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)
	writeTranscript(t, dir, resumeIDAlpha, "hello resume", when)

	msgs, err := session.ReadTranscript(t.Context(), session.TranscriptPath(dir, resumeIDAlpha))
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, "hello resume", msgs[0].Content)
}

func TestListResumable_SkipsCurrentAndEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	writeTranscript(t, dir, resumeIDAlpha, "keep me", now.Add(-time.Hour))

	idx := seedResumeIndex(t, dir,
		session.IndexEntry{ID: resumeIDAlpha, WorkDir: resumeWorkDir, LastModified: now.Add(-time.Hour)},
		session.IndexEntry{ID: resumeIDBeta, WorkDir: resumeWorkDir, LastModified: now},
	)

	got := session.ListResumable(context.Background(), session.ListResumableOptions{
		Index:      idx,
		SessionDir: dir,
		WorkDir:    resumeWorkDir,
		ExcludeID:  resumeIDBeta,
	})

	require.Len(t, got, 1)
	require.Equal(t, resumeIDAlpha, got[0].ID)
	require.Equal(t, 1, got[0].Ordinal)
	require.Equal(t, 1, got[0].MessageCount)
	require.Equal(t, "keep me", got[0].Preview)
}

func TestResolveSelector(t *testing.T) {
	t.Parallel()

	cands := []session.ResumeCandidate{
		{Ordinal: 1, ID: resumeIDAlpha},
		{Ordinal: 2, ID: resumeIDBeta},
	}

	got, err := session.ResolveSelector(cands, "2")
	require.NoError(t, err)
	require.Equal(t, resumeIDBeta, got.ID)

	got, err = session.ResolveSelector(cands, "last")
	require.NoError(t, err)
	require.Equal(t, resumeIDAlpha, got.ID)

	got, err = session.ResolveSelector(cands, "aaaaaaaa")
	require.NoError(t, err)
	require.Equal(t, resumeIDAlpha, got.ID)

	_, err = session.ResolveSelector(cands, "zzz")
	require.ErrorIs(t, err, session.ErrResumeSelectorUnknown)

	_, err = session.ResolveSelector([]session.ResumeCandidate{
		{Ordinal: 1, ID: "aaaa-1111"},
		{Ordinal: 2, ID: "aaaa-2222"},
	}, "aaaa")
	require.ErrorIs(t, err, session.ErrResumeSelectorAmbiguous)

	_, err = session.ResolveSelector(nil, "1")
	require.ErrorIs(t, err, session.ErrNoResumableSessions)
}

func TestFormatResumeList_FitsWidth(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	text := session.FormatResumeList([]session.ResumeCandidate{{
		Ordinal:      1,
		ID:           resumeIDAlpha,
		MessageCount: 3,
		LastModified: now.Add(-2 * time.Hour),
		Preview:      "implement the approval bar for long commands",
	}}, now)

	require.Contains(t, text, "1.")
	require.Contains(t, text, "aaaaaaaa")
	require.Contains(t, text, "2 h ago")
	require.Contains(t, text, "/resume")

	for line := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(line, "  ") {
			require.LessOrEqual(t, textwidth.TotalWidth(textwidth.ExtractGraphemes(line)), session.ResumeListWidth)
		}
	}
}

func TestFormatResumeList_Empty(t *testing.T) {
	t.Parallel()

	text := session.FormatResumeList(nil, time.Now())
	require.Contains(t, text, "No resumable sessions")
}
