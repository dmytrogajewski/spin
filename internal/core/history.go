package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Common errors for history operations.
var (
	ErrEmptyHistory    = errors.New("history is empty")
	ErrMessageNotFound = errors.New("message not found")
	ErrInvalidMessage  = errors.New("invalid message")
	ErrInvalidRole     = errors.New("invalid message role")
	ErrExportFailed    = errors.New("export failed")
	ErrImportFailed    = errors.New("import failed")
	ErrTokenizerNil    = errors.New("tokenizer cannot be nil")
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
	messages  []Message
	maxTokens int
	tokenizer Tokenizer
	mu        sync.RWMutex
}

// historyExport represents the JSON structure for history export.
type historyExport struct {
	Version      string    `json:"version"`
	MaxTokens    int       `json:"maxTokens"`
	TotalTokens  int       `json:"totalTokens"`
	MessageCount int       `json:"messageCount"`
	Messages     []Message `json:"messages"`
}

// NewHistory creates a new history manager with the specified token budget.
//
// Parameters:
//   - maxTokens: Maximum token budget for the history
//   - tokenizer: Token counting implementation
//
// Returns an error if tokenizer is nil.
func NewHistory(maxTokens int, tokenizer Tokenizer) *History {
	if tokenizer == nil {
		tokenizer = &SimpleTokenizer{}
	}

	return &History{
		messages:  make([]Message, 0),
		maxTokens: maxTokens,
		tokenizer: tokenizer,
	}
}

// NewHistoryWithDefaults creates a history with sensible default values.
//
// Default settings:
//   - maxTokens: 4096
//   - tokenizer: SimpleTokenizer
func NewHistoryWithDefaults() *History {
	return NewHistory(4096, &SimpleTokenizer{})
}

// AddMessage appends a message to the history with automatic token counting.
//
// The message is validated before adding. Token count is automatically
// calculated if not already set. This method is thread-safe.
//
// Returns an error if the message is invalid (e.g., empty role).
func (h *History) AddMessage(msg Message) error {
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
	return h.AddMessage(Message{
		Role:    RoleSystem,
		Content: content,
	})
}

// AddUserMessage adds a user message to the history.
//
// This is a convenience method for adding user role messages.
func (h *History) AddUserMessage(content string) error {
	return h.AddMessage(Message{
		Role:    RoleUser,
		Content: content,
	})
}

// AddAssistantMessage adds an assistant message to the history.
//
// This is a convenience method for adding assistant role messages.
func (h *History) AddAssistantMessage(content string) error {
	return h.AddMessage(Message{
		Role:    RoleAssistant,
		Content: content,
	})
}

// AddToolMessage adds a tool result message to the history.
//
// Parameters:
//   - toolCallID: The ID of the tool call this result corresponds to
//   - content: The tool execution result
func (h *History) AddToolMessage(toolCallID, content string) error {
	return h.AddMessage(Message{
		Role:       RoleTool,
		ToolCallID: toolCallID,
		Content:    content,
	})
}

// Messages returns a copy of all messages in the history.
//
// This returns a defensive copy to prevent external modification of the
// internal message slice. This method is thread-safe.
func (h *History) Messages() []Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy to prevent external modification
	msgs := make([]Message, len(h.messages))
	copy(msgs, h.messages)
	return msgs
}

// MessagesForLLM returns messages formatted for LLM API consumption.
//
// Currently this returns the same as Messages(), but in the future may
// perform additional formatting or filtering specific to LLM requirements.
func (h *History) MessagesForLLM() []Message {
	return h.Messages()
}

// GetMessage retrieves a specific message by ID.
//
// Returns ErrMessageNotFound if no message with the given ID exists.
func (h *History) GetMessage(id string) (*Message, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for i := range h.messages {
		if h.messages[i].ID == id {
			// Return a copy
			msg := h.messages[i]
			return &msg, nil
		}
	}

	return nil, fmt.Errorf("%w: id=%s", ErrMessageNotFound, id)
}

// LastMessage returns the most recent message in the history.
//
// Returns ErrEmptyHistory if the history contains no messages.
func (h *History) LastMessage() (*Message, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.messages) == 0 {
		return nil, ErrEmptyHistory
	}

	// Return a copy of the last message
	msg := h.messages[len(h.messages)-1]
	return &msg, nil
}

// MessagesSince returns all messages after the specified timestamp.
//
// This is useful for retrieving recent messages or incremental updates.
func (h *History) MessagesSince(timestamp time.Time) []Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]Message, 0)
	for _, msg := range h.messages {
		if msg.Timestamp.After(timestamp) {
			result = append(result, msg)
		}
	}

	return result
}

// Count returns the number of messages in the history.
func (h *History) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.messages)
}

// TokenCount returns the total token count for all messages.
//
// This sums up the token counts of all individual messages.
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

// IsEmpty returns true if the history contains no messages.
func (h *History) IsEmpty() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.messages) == 0
}

// Truncate reduces the history to fit within the specified token budget.
//
// Truncation Strategy:
//  1. Always preserves system message (first message if role is system)
//  2. Removes oldest non-system messages until within budget
//  3. Keeps most recent messages for context continuity
//
// This method is thread-safe.
func (h *History) Truncate(budget int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.tokenCountLocked() <= budget {
		return nil // Already within budget
	}

	// Separate system message if present
	var systemMsg *Message
	var messages []Message

	if len(h.messages) > 0 && h.messages[0].Role == RoleSystem {
		systemMsg = &h.messages[0]
		messages = h.messages[1:]
	} else {
		messages = h.messages
	}

	// Calculate system message tokens
	systemTokens := 0
	if systemMsg != nil {
		systemTokens = systemMsg.Tokens
	}

	// Keep recent messages that fit in budget
	// Pre-allocate with max possible size
	truncated := make([]Message, 0, len(messages)+1)
	if systemMsg != nil {
		truncated = append(truncated, *systemMsg)
	}

	tokens := systemTokens

	// Collect messages in reverse order (most recent first)
	tempMessages := make([]Message, 0, len(messages))

	// Iterate from most recent backwards
	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := messages[i].Tokens
		if tokens+msgTokens > budget {
			break
		}
		tempMessages = append(tempMessages, messages[i])
		tokens += msgTokens
	}

	// Reverse tempMessages to maintain chronological order
	for i := len(tempMessages) - 1; i >= 0; i-- {
		truncated = append(truncated, tempMessages[i])
	}

	h.messages = truncated
	return nil
}

// TruncateToFit truncates the history to fit within the configured maxTokens.
func (h *History) TruncateToFit() error {
	return h.Truncate(h.maxTokens)
}

// WouldExceedBudget checks if adding the given message would exceed the max token budget.
//
// This is useful for proactive truncation before adding new messages.
func (h *History) WouldExceedBudget(msg Message) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	currentTokens := 0
	for _, m := range h.messages {
		currentTokens += m.Tokens
	}

	msgTokens := msg.Tokens
	if msgTokens == 0 {
		msgTokens = h.tokenizer.Count(msg.Content) + 4
	}

	return currentTokens+msgTokens > h.maxTokens
}

// Clear removes all messages except the system message.
//
// This is useful for resetting a conversation while keeping the system prompt.
func (h *History) Clear() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Keep only system message if present
	if len(h.messages) > 0 && h.messages[0].Role == RoleSystem {
		h.messages = []Message{h.messages[0]}
	} else {
		h.messages = []Message{}
	}

	return nil
}

// ClearAll removes all messages including the system message.
//
// This completely empties the history.
func (h *History) ClearAll() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.messages = []Message{}
	return nil
}

// Reset clears the history and sets a new system message.
//
// This is useful for starting a fresh conversation with a new system prompt.
func (h *History) Reset(systemMessage string) error {
	if err := h.ClearAll(); err != nil {
		return err
	}

	return h.AddSystemMessage(systemMessage)
}

// Clone creates a deep copy of the history.
//
// The cloned history is completely independent of the original.
func (h *History) Clone() *History {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clone := &History{
		maxTokens: h.maxTokens,
		tokenizer: h.tokenizer,
		messages:  make([]Message, len(h.messages)),
	}

	copy(clone.messages, h.messages)

	return clone
}

// Export saves the history to a file in JSON format.
//
// The exported file includes metadata (version, token counts) and all messages.
func (h *History) Export(path string) error {
	data, err := h.ExportJSON()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExportFailed, err)
	}

	err = os.WriteFile(path, data, 0600)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExportFailed, err)
	}

	return nil
}

// ExportJSON returns the history as JSON bytes.
//
// This is useful for embedding history in other structures or sending over the network.
func (h *History) ExportJSON() ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	export := historyExport{
		Version:      "1.0",
		MaxTokens:    h.maxTokens,
		TotalTokens:  h.tokenCountLocked(),
		MessageCount: len(h.messages),
		Messages:     h.messages,
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExportFailed, err)
	}

	return data, nil
}

// Import loads a history from a JSON file.
//
// The tokenizer must be provided as it's not serialized with the history.
func Import(path string, tokenizer Tokenizer) (*History, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrImportFailed, err)
	}

	return ImportJSON(data, tokenizer)
}

// ImportJSON creates a history from JSON bytes.
//
// Token counts are recalculated on import using the provided tokenizer.
func ImportJSON(data []byte, tokenizer Tokenizer) (*History, error) {
	if tokenizer == nil {
		return nil, ErrTokenizerNil
	}

	var export historyExport
	err := json.Unmarshal(data, &export)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrImportFailed, err)
	}

	history := NewHistory(export.MaxTokens, tokenizer)

	// Add messages (token counts will be recalculated)
	for _, msg := range export.Messages {
		// Preserve original ID and timestamp
		history.mu.Lock()
		history.messages = append(history.messages, msg)
		history.mu.Unlock()
	}

	return history, nil
}
