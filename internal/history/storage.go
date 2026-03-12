package history

import (
	"errors"
	"time"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/storage"
)

// HistoryData is the serializable format for history persistence.
type HistoryData struct {
	Version   int               `json:"version"`
	SessionID string            `json:"session_id"`
	Messages  []message.Message `json:"messages"`
	MaxTokens int               `json:"max_tokens"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// CurrentHistoryVersion is the current schema version for migrations.
const CurrentHistoryVersion = 1

// Storage is a type alias for the generic store with HistoryData type.
type Storage = storage.Store[HistoryData]

// NewFileStorage creates file-based history storage.
func NewFileStorage(baseDir string) (Storage, error) {
	return storage.NewFileStore[HistoryData](storage.FileStoreConfig{
		BaseDir: baseDir,
		Suffix:  ".history.json",
	})
}

// Save persists the history to storage.
func (h *History) Save(store Storage, sessionID string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data := HistoryData{
		Version:   CurrentHistoryVersion,
		SessionID: sessionID,
		Messages:  make([]message.Message, len(h.messages)),
		MaxTokens: h.maxTokens,
		UpdatedAt: time.Now(),
	}
	copy(data.Messages, h.messages)

	return store.Save(sessionID, data)
}

// Load restores history from storage.
func (h *History) Load(store Storage, sessionID string) error {
	data, err := store.Load(sessionID)
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.messages = make([]message.Message, len(data.Messages))
	copy(h.messages, data.Messages)

	if data.MaxTokens > 0 {
		h.maxTokens = data.MaxTokens
	}

	return nil
}

// ToHistoryData exports the current history state as HistoryData.
func (h *History) ToHistoryData(sessionID string) *HistoryData {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data := &HistoryData{
		Version:   CurrentHistoryVersion,
		SessionID: sessionID,
		Messages:  make([]message.Message, len(h.messages)),
		MaxTokens: h.maxTokens,
		UpdatedAt: time.Now(),
	}
	copy(data.Messages, h.messages)

	return data
}

// FromHistoryData imports history state from HistoryData.
func (h *History) FromHistoryData(data *HistoryData) error {
	if data == nil {
		return errors.New("history data cannot be nil")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.messages = make([]message.Message, len(data.Messages))
	copy(h.messages, data.Messages)

	if data.MaxTokens > 0 {
		h.maxTokens = data.MaxTokens
	}

	return nil
}
