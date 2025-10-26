package history

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
	"github.com/google/uuid"
)

// Common errors for history operations.
var (
	ErrEmptyHistory    = errors.New("history is empty")
	ErrMessageNotFound = errors.New("message not found")
	ErrInvalidMessage  = errors.New("invalid message")
	ErrInvalidRole     = errors.New("invalid message role")
)

// History manages conversation message history with token-aware truncation.
//
// History provides thread-safe management of conversation messages between
// the user and AI assistant. It supports automatic token counting, smart
// truncation to fit within token budgets, and export/import capabilities.
//
// Token Management:
//   - Automatically counts tokens for each message
//   - Tracks total token usage
//   - Provides truncation to fit within budget
//   - Preserves system messages during truncation
//
// Thread Safety:
//   - All methods are thread-safe
//   - Concurrent reads and writes are supported
//   - Uses RWMutex for efficient concurrent reads
//
// Example:
//
//	history := NewHistory(4096, tokenizer)
//	history.AddSystemMessage("You are helpful")
//	history.AddUserMessage("Hello")
//
//	if history.TokenCount() > 3000 {
//	    history.Truncate(3000)
//	}
type History struct {
	messages  []message.Message
	maxTokens int
	tokenizer tokenizer.Tokenizer
	emitter   *events.EventEmitter
	mu        sync.RWMutex
}

// NewHistory creates a new history manager with the specified token budget.
//
// Parameters:
//   - maxTokens: Maximum token budget for the history
//   - tokenizer: Token counting implementation
//
// Returns an error if tokenizer is nil.
func NewHistory(maxTokens int, tok tokenizer.Tokenizer) *History {
	if tok == nil {
		tok = &tokenizer.SimpleTokenizer{}
	}

	return &History{
		messages:  make([]message.Message, 0),
		maxTokens: maxTokens,
		tokenizer: tok,
	}
}

// SetEventEmitter sets the event emitter for notifications.
func (h *History) SetEventEmitter(emitter *events.EventEmitter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.emitter = emitter
}

// NewHistoryWithDefaults creates a history with sensible default values.
//
// Default settings:
//   - maxTokens: 4096
//   - tokenizer: SimpleTokenizer
func NewHistoryWithDefaults() *History {
	return NewHistory(4096, &tokenizer.SimpleTokenizer{})
}

// AddMessage appends a message to the history with automatic token counting.
//
// The message is validated before adding. Token count is automatically
// calculated if not already set. This method is thread-safe.
//
// Returns an error if the message is invalid (e.g., empty role).
func (h *History) AddMessage(msg message.Message) error {
	if msg.Role == "" {
		return fmt.Errorf("%w: role is required", ErrInvalidMessage)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Generate ID if not set
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	// Set timestamp if not set
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Count tokens if not already set
	if msg.Tokens == 0 {
		msg.Tokens = h.tokenizer.Count(msg.Content)

		// Count tokens for tool calls if present
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				// Count function name and arguments
				msg.Tokens += h.tokenizer.Count(tc.Function.Name)
				msg.Tokens += h.tokenizer.Count(tc.Function.Arguments)
				// Add tool call overhead (formatting, IDs, etc.)
				msg.Tokens += 8
			}
		}

		// Add message overhead
		msg.Tokens += 4
	}

	h.messages = append(h.messages, msg)

	return nil
}

// AddSystemMessage adds a system message to the history.
//
// This is a convenience method for adding system role messages.
func (h *History) AddSystemMessage(content string) error {
	return h.AddMessage(message.Message{
		Role:    message.RoleSystem,
		Content: content,
	})
}

// AddUserMessage adds a user message to the history.
//
// This is a convenience method for adding user role messages.
func (h *History) AddUserMessage(content string) error {
	return h.AddMessage(message.Message{
		Role:    message.RoleUser,
		Content: content,
	})
}

// Messages returns a copy of all messages in the history.
//
// This returns a defensive copy to prevent external modification of the
// internal message slice. This method is thread-safe.
func (h *History) Messages() []message.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy to prevent external modification
	msgs := make([]message.Message, len(h.messages))
	copy(msgs, h.messages)
	return msgs
}

// MessagesForLLM returns messages formatted for LLM API consumption.
//
// Currently this returns the same as Messages(), but in the future may
// perform additional formatting or filtering specific to LLM requirements.
func (h *History) MessagesForLLM() []message.Message {
	return h.Messages()
}

// TokenCount returns the total number of tokens in the history.
func (h *History) TokenCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.tokenCountLocked()
}

// tokenCountLocked returns token count without acquiring lock.
// Must be called with lock already held.
func (h *History) tokenCountLocked() int {
	total := 0
	for _, msg := range h.messages {
		total += msg.Tokens
	}
	return total
}
