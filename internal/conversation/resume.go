package conversation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/contexteng/history"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/session"
)

// ErrSessionDirUnavailable is returned when resume cannot resolve the session root.
var ErrSessionDirUnavailable = errors.New("session directory is not configured")

// ListResumableSessions returns prior sessions that /resume can continue.
func (c *Conversation) ListResumableSessions(ctx context.Context) []session.ResumeCandidate {
	return session.ListResumable(ctx, session.ListResumableOptions{
		Index:      c.sessionIndex,
		SessionDir: c.sessionDir,
		WorkDir:    c.workDir,
		ExcludeID:  c.id,
	})
}

// Resume replaces in-memory history and the transcript writer with sessionID.
// Subsequent turns append to that session's existing transcript.
func (c *Conversation) Resume(ctx context.Context, sessionID string) error {
	if c.sessionDir == "" {
		return ErrSessionDirUnavailable
	}

	if sessionID == "" {
		return session.ErrResumeSelectorUnknown
	}

	if sessionID == c.id {
		return session.ErrResumeAlreadyCurrent
	}

	path := session.TranscriptPath(c.sessionDir, sessionID)

	msgs, err := session.ReadTranscript(ctx, path)
	if err != nil {
		return fmt.Errorf("read transcript: %w", err)
	}

	if len(msgs) == 0 {
		return session.ErrResumeEmptyTranscript
	}

	if loadErr := c.history.FromData(&history.Data{
		Messages:  msgs,
		MaxTokens: c.history.MaxTokens(),
	}); loadErr != nil {
		return fmt.Errorf("restore history: %w", loadErr)
	}

	if swapErr := c.swapTranscriptWriter(path); swapErr != nil {
		return swapErr
	}

	c.id = sessionID
	c.touchSessionIndex(ctx)

	return nil
}

func (c *Conversation) swapTranscriptWriter(path string) error {
	if c.transcriptWriter != nil {
		_ = c.transcriptWriter.Close()
		c.transcriptWriter = nil
	}

	tw, err := session.NewTranscriptWriter(path)
	if err != nil {
		return fmt.Errorf("reopen transcript: %w", err)
	}

	c.transcriptWriter = tw

	return nil
}

func (c *Conversation) touchSessionIndex(ctx context.Context) {
	if c.sessionIndex == nil {
		return
	}

	msgs := c.history.Messages()

	_ = c.sessionIndex.Update(ctx, session.IndexEntry{
		ID:           c.id,
		Title:        firstUserTitle(msgs),
		MessageCount: len(msgs),
		LastModified: time.Now(),
		WorkDir:      c.workDir,
	})
}

func firstUserTitle(msgs []message.Message) string {
	for _, msg := range msgs {
		if msg.Role == message.RoleUser && msg.Content != "" {
			return session.TruncatePreview(msg.Content)
		}
	}

	return ""
}
