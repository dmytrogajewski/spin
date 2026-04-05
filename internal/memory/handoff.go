package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

var (
	// ErrNoPersistentStoreConfigured is a sentinel error.
	ErrNoPersistentStoreConfigured = errors.New("no persistent store configured")
	// ErrSessionIDIsRequired is a sentinel error.
	ErrSessionIDIsRequired = errors.New("session ID is required")
)

// SessionHandoff manages context transfer between sessions.
type SessionHandoff struct {
	store      *PersistentStore
	summarizer Summarizer
}

// Summarizer provides summarization capabilities.
type Summarizer interface {
	// Summarize creates a summary of the given content.
	Summarize(ctx context.Context, content string, maxTokens int) (string, error)
}

// HandoffData contains the session state for handoff.
type HandoffData struct {
	// SessionID identifies the session.
	SessionID string `json:"session_id"`

	// Summary is a brief summary of the session.
	Summary string `json:"summary"`

	// PendingTasks lists tasks that weren't completed.
	PendingTasks []string `json:"pending_tasks,omitempty"`

	// Decisions lists key decisions made during the session.
	Decisions []string `json:"decisions,omitempty"`

	// KeyReferences maps important references by name.
	KeyReferences map[string]string `json:"key_references,omitempty"`

	// LastActivity is when the session was last active.
	LastActivity time.Time `json:"last_activity"`

	// WorkDir is the working directory of the session.
	WorkDir string `json:"work_dir,omitempty"`

	// Context contains additional context data.
	Context map[string]string `json:"context,omitempty"`
}

// NewSessionHandoff creates a new session handoff manager.
func NewSessionHandoff(store *PersistentStore, summarizer Summarizer) *SessionHandoff {
	return &SessionHandoff{
		store:      store,
		summarizer: summarizer,
	}
}

// SaveSession saves the current session state for future continuation.
func (h *SessionHandoff) SaveSession(ctx context.Context, data HandoffData) error {
	if h.store == nil {
		return ErrNoPersistentStoreConfigured
	}

	if data.SessionID == "" {
		return ErrSessionIDIsRequired
	}

	// Set last activity if not provided.
	if data.LastActivity.IsZero() {
		data.LastActivity = time.Now()
	}

	// Serialize handoff data.
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal handoff data: %w", err)
	}

	// Store with session namespace.
	key := "session_" + data.SessionID

	return h.store.Put(ctx, key, string(jsonData), PutOptions{
		Namespace: "sessions",
		Overwrite: true,
	})
}

// LoadSession retrieves a previously saved session state.
func (h *SessionHandoff) LoadSession(ctx context.Context, sessionID string) (*HandoffData, error) {
	if h.store == nil {
		return nil, ErrNoPersistentStoreConfigured
	}

	key := "session_" + sessionID

	entry, err := h.store.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}

	var data HandoffData

	err = json.Unmarshal([]byte(entry.Value), &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal handoff data: %w", err)
	}

	return &data, nil
}

// ListSessions returns all saved session IDs.
func (h *SessionHandoff) ListSessions(ctx context.Context) ([]string, error) {
	if h.store == nil {
		return nil, ErrNoPersistentStoreConfigured
	}

	keys, err := h.store.List(ctx, "session_*")
	if err != nil {
		return nil, err
	}

	sessions := make([]string, 0, len(keys))
	for _, key := range keys {
		// Remove "session_" prefix.
		if strings.HasPrefix(key, "session_") {
			sessions = append(sessions, key[8:])
		}
	}

	return sessions, nil
}

// DeleteSession removes a saved session.
func (h *SessionHandoff) DeleteSession(ctx context.Context, sessionID string) error {
	if h.store == nil {
		return ErrNoPersistentStoreConfigured
	}

	key := "session_" + sessionID

	return h.store.Delete(ctx, key)
}

// BuildContinuationPrompt creates a prompt for continuing a previous session.
func (h *SessionHandoff) BuildContinuationPrompt(data *HandoffData) string {
	if data == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("[Continuing from previous session]\n\n")

	if data.Summary != "" {
		sb.WriteString("Previous session summary:\n")
		sb.WriteString(data.Summary)
		sb.WriteString("\n\n")
	}

	if len(data.PendingTasks) > 0 {
		sb.WriteString("Pending tasks:\n")

		for _, task := range data.PendingTasks {
			sb.WriteString("- ")
			sb.WriteString(task)
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	if len(data.Decisions) > 0 {
		sb.WriteString("Key decisions made:\n")

		for _, decision := range data.Decisions {
			sb.WriteString("- ")
			sb.WriteString(decision)
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	if len(data.KeyReferences) > 0 {
		sb.WriteString("Key references:\n")

		for name, ref := range data.KeyReferences {
			sb.WriteString("- ")
			sb.WriteString(name)
			sb.WriteString(": ")
			sb.WriteString(ref)
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	if data.WorkDir != "" {
		fmt.Fprintf(&sb, "Working directory: %s\n", data.WorkDir)
	}

	if !data.LastActivity.IsZero() {
		fmt.Fprintf(&sb, "Last activity: %s\n", data.LastActivity.Format(time.DateTime))
	}

	return sb.String()
}

// SimpleSummarizer provides a basic summarization that truncates content.
// This is used when no LLM-based summarizer is available.
type SimpleSummarizer struct {
	maxLength int
}

// NewSimpleSummarizer creates a simple summarizer with the given max length.
func NewSimpleSummarizer(maxLength int) *SimpleSummarizer {
	if maxLength <= 0 {
		maxLength = 500
	}

	return &SimpleSummarizer{maxLength: maxLength}
}

// Summarize truncates content to the max length.
func (s *SimpleSummarizer) Summarize(_ context.Context, content string, maxTokens int) (string, error) {
	// Use maxTokens if provided, otherwise use configured maxLength.
	limit := s.maxLength
	if maxTokens > 0 {
		// Rough conversion: 1 token ≈ 4 characters.
		limit = maxTokens * stringsx.CharsPerToken
	}

	if len(content) <= limit {
		return content, nil
	}

	// Truncate and add ellipsis.
	return content[:limit-3] + "...", nil
}
