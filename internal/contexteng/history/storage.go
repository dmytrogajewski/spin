package history

import (
	"context"
	"errors"
	"time"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/storage"
)

// ErrHistoryDataCannotBeNil is a sentinel error.
var ErrHistoryDataCannotBeNil = errors.New("history data cannot be nil")

// Data is the serializable format for history persistence.
type Data struct {
	Version   int               `json:"version"`
	SessionID string            `json:"session_id"`
	Messages  []message.Message `json:"messages"`
	MaxTokens int               `json:"max_tokens"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// CurrentHistoryVersion is the current schema version for migrations.
const CurrentHistoryVersion = 1

// Storage is a type alias for the generic store with Data type.
type Storage = storage.Store[Data]

// NewFileStorage creates file-based history storage.
func NewFileStorage(baseDir string) (Storage, error) {
	return storage.NewFileStore[Data](storage.FileStoreConfig{
		BaseDir: baseDir,
		Suffix:  ".history.json",
	})
}

// Save persists the history to storage.
func (h *History) Save(ctx context.Context, store Storage, sessionID string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data := Data{
		Version:   CurrentHistoryVersion,
		SessionID: sessionID,
		Messages:  make([]message.Message, len(h.messages)),
		MaxTokens: h.maxTokens,
		UpdatedAt: time.Now(),
	}
	copy(data.Messages, h.messages)

	return store.Save(ctx, sessionID, data)
}

// Load restores history from storage.
func (h *History) Load(ctx context.Context, store Storage, sessionID string) error {
	data, err := store.Load(ctx, sessionID)
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

// ToData exports the current history state as Data.
func (h *History) ToData(sessionID string) *Data {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data := &Data{
		Version:   CurrentHistoryVersion,
		SessionID: sessionID,
		Messages:  make([]message.Message, len(h.messages)),
		MaxTokens: h.maxTokens,
		UpdatedAt: time.Now(),
	}
	copy(data.Messages, h.messages)

	return data
}

// FromData imports history state from Data.
func (h *History) FromData(data *Data) error {
	if data == nil {
		return ErrHistoryDataCannotBeNil
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
