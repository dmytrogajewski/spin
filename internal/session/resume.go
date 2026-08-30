package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/pkg/alg/ds"
	"github.com/dmytrogajewski/spin/pkg/ui/textwidth"
)

var (
	// ErrNoResumableSessions is returned when the catalog is empty.
	ErrNoResumableSessions = errors.New("no resumable sessions")
	// ErrResumeSelectorUnknown is returned when the selector matches nothing.
	ErrResumeSelectorUnknown = errors.New("no session matches selector")
	// ErrResumeSelectorAmbiguous is returned when a prefix matches more than one ID.
	ErrResumeSelectorAmbiguous = errors.New("session selector is ambiguous")
	// ErrResumeAlreadyCurrent is returned when the selector is the live session.
	ErrResumeAlreadyCurrent = errors.New("session is already current")
	// ErrResumeEmptyTranscript is returned when the chosen session has no messages.
	ErrResumeEmptyTranscript = errors.New("session transcript is empty")
)

const (
	// TranscriptFileName is the JSONL filename under each session directory.
	TranscriptFileName = "transcript.jsonl"
	// ShortSessionIDLen is the visible ID prefix in /resume listings.
	ShortSessionIDLen = 8
	// ResumePreviewWidth caps the first-user-message excerpt.
	ResumePreviewWidth = 48
	// ResumeListWidth is the target listing line width.
	ResumeListWidth = 80
	// SelectorLast is the alias for the newest resumable session.
	SelectorLast = "last"

	hoursPerDay = 24
	daysPerWeek = 7
)

// ResumeCandidate is one session the user can continue.
type ResumeCandidate struct {
	Ordinal      int
	ID           string
	WorkDir      string
	MessageCount int
	LastModified time.Time
	Preview      string
}

// ListResumableOptions controls catalog filtering.
type ListResumableOptions struct {
	Index      *Index
	SessionDir string
	WorkDir    string
	ExcludeID  string
}

// TranscriptPath returns the JSONL path for a session.
func TranscriptPath(sessionDir, sessionID string) string {
	return filepath.Join(sessionDir, sessionID, TranscriptFileName)
}

// ReadTranscript loads messages from a session transcript.
// A missing file is [ErrTranscriptNotFound].
func ReadTranscript(_ context.Context, path string) ([]message.Message, error) {
	if _, statErr := os.Stat(path); statErr != nil {
		if os.IsNotExist(statErr) {
			return nil, ErrTranscriptNotFound
		}

		return nil, fmt.Errorf("stat transcript: %w", statErr)
	}

	msgs, err := ds.ReadJSONL[message.Message](path)
	if err != nil {
		return nil, err
	}

	if msgs == nil {
		return []message.Message{}, nil
	}

	return msgs, nil
}

// ListResumable returns newest-first sessions that have a non-empty transcript,
// excluding ExcludeID. Index metadata is enriched from the transcript when
// MessageCount is zero.
func ListResumable(ctx context.Context, opts ListResumableOptions) []ResumeCandidate {
	if opts.Index == nil {
		return nil
	}

	entries := opts.Index.List(opts.WorkDir)
	out := make([]ResumeCandidate, 0, len(entries))

	for _, entry := range entries {
		if entry.ID == opts.ExcludeID {
			continue
		}

		cand := candidateFromEntry(ctx, opts.SessionDir, entry)
		if cand.MessageCount == 0 {
			continue
		}

		out = append(out, cand)
	}

	for i := range out {
		out[i].Ordinal = i + 1
	}

	return out
}

// ResolveSelector picks a candidate by 1-based index, "last", full ID, or unique prefix.
func ResolveSelector(candidates []ResumeCandidate, selector string) (ResumeCandidate, error) {
	if len(candidates) == 0 {
		return ResumeCandidate{}, ErrNoResumableSessions
	}

	sel := strings.TrimSpace(strings.ToLower(selector))
	if sel == "" {
		return ResumeCandidate{}, ErrResumeSelectorUnknown
	}

	if sel == SelectorLast {
		return candidates[0], nil
	}

	if n, err := strconv.Atoi(sel); err == nil {
		for _, c := range candidates {
			if c.Ordinal == n {
				return c, nil
			}
		}

		return ResumeCandidate{}, fmt.Errorf("%w: %s", ErrResumeSelectorUnknown, selector)
	}

	return resolveByID(candidates, sel)
}

// FormatResumeList renders a numbered catalog. now is the clock for relative ages.
func FormatResumeList(candidates []ResumeCandidate, now time.Time) string {
	if len(candidates) == 0 {
		return "No resumable sessions.\nType a prompt to start one, then /resume after the next launch."
	}

	var b strings.Builder

	b.WriteString("Resumable sessions:\n")

	for _, c := range candidates {
		line := formatResumeLine(c, now)
		if unisegWidth(line) > ResumeListWidth {
			line = textwidth.TruncateRight(line, ResumeListWidth)
		}

		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString("\nType /resume <n>, /resume last, or /resume <id>")

	return b.String()
}

func candidateFromEntry(ctx context.Context, sessionDir string, entry IndexEntry) ResumeCandidate {
	cand := ResumeCandidate{
		ID:           entry.ID,
		WorkDir:      entry.WorkDir,
		MessageCount: entry.MessageCount,
		LastModified: entry.LastModified,
		Preview:      entry.Title,
	}

	if sessionDir == "" {
		return cand
	}

	msgs, err := ReadTranscript(ctx, TranscriptPath(sessionDir, entry.ID))
	if err != nil {
		return cand
	}

	if cand.MessageCount == 0 {
		cand.MessageCount = len(msgs)
	}

	if cand.Preview == "" {
		cand.Preview = firstUserPreview(msgs)
	}

	if latest := newestMessageTime(msgs); !latest.IsZero() &&
		(cand.LastModified.IsZero() || latest.After(cand.LastModified)) {
		cand.LastModified = latest
	}

	return cand
}

func resolveByID(candidates []ResumeCandidate, sel string) (ResumeCandidate, error) {
	var matches []ResumeCandidate

	for _, c := range candidates {
		id := strings.ToLower(c.ID)
		if id == sel || strings.HasPrefix(id, sel) {
			matches = append(matches, c)
		}
	}

	switch len(matches) {
	case 0:
		return ResumeCandidate{}, fmt.Errorf("%w: %s", ErrResumeSelectorUnknown, sel)
	case 1:
		return matches[0], nil
	default:
		return ResumeCandidate{}, fmt.Errorf("%w: %s", ErrResumeSelectorAmbiguous, sel)
	}
}

func formatResumeLine(c ResumeCandidate, now time.Time) string {
	preview := c.Preview
	if preview == "" {
		preview = "(no preview)"
	}

	return fmt.Sprintf("  %d. %s  %d msgs  %s  %s",
		c.Ordinal,
		ShortID(c.ID),
		c.MessageCount,
		relativeAge(now, c.LastModified),
		preview,
	)
}

// ShortID returns the listing prefix for a session ID.
func ShortID(id string) string {
	if utf8.RuneCountInString(id) <= ShortSessionIDLen {
		return id
	}

	return string([]rune(id)[:ShortSessionIDLen])
}

func firstUserPreview(msgs []message.Message) string {
	for _, msg := range msgs {
		if msg.Role != message.RoleUser {
			continue
		}

		if preview := TruncatePreview(msg.Content); preview != "" {
			return preview
		}
	}

	return ""
}

// TruncatePreview collapses whitespace and caps width for list/index titles.
func TruncatePreview(text string) string {
	return textwidth.TruncateRight(strings.Join(strings.Fields(text), " "), ResumePreviewWidth)
}

func newestMessageTime(msgs []message.Message) time.Time {
	var latest time.Time

	for _, msg := range msgs {
		if msg.Timestamp.After(latest) {
			latest = msg.Timestamp
		}
	}

	return latest
}

func relativeAge(now, then time.Time) string {
	if then.IsZero() || now.Before(then) {
		return "just now"
	}

	d := now.Sub(then)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < hoursPerDay*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	case d < daysPerWeek*hoursPerDay*time.Hour:
		return fmt.Sprintf("%d d ago", int(d.Hours())/hoursPerDay)
	default:
		return then.UTC().Format(time.DateOnly)
	}
}

func unisegWidth(s string) int {
	return textwidth.TotalWidth(textwidth.ExtractGraphemes(s))
}
