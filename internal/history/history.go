package history

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/history/compress"
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
	messages   []message.Message
	maxTokens  int
	tokenizer  tokenizer.Tokenizer
	compressor compress.Compressor
	config     *HistoryConfig
	emitter    *events.EventEmitter
	mu         sync.RWMutex
}

// HistoryConfig configures history compression behavior.
type HistoryConfig struct {
	// CompressionEnabled enables automatic compression
	CompressionEnabled bool

	// CompressionThreshold is the token usage ratio to trigger compression (0.0-1.0)
	// Example: 0.8 means compress at 80% capacity
	CompressionThreshold float64

	// PreserveCritical ensures critical messages are always kept
	PreserveCritical bool

	// MinRetention is the minimum fraction of messages to keep (0.0-1.0)
	MinRetention float64
}

// DefaultHistoryConfig returns sensible default configuration.
func DefaultHistoryConfig() *HistoryConfig {
	return &HistoryConfig{
		CompressionEnabled:   true,
		CompressionThreshold: 0.8,
		PreserveCritical:     true,
		MinRetention:         0.3,
	}
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

	cfg := DefaultHistoryConfig()

	return &History{
		messages:   make([]message.Message, 0),
		maxTokens:  maxTokens,
		tokenizer:  tok,
		compressor: compress.NewDefaultHybridCompressor(),
		config:     cfg,
	}
}

// SetEventEmitter sets the event emitter for compression notifications.
func (h *History) SetEventEmitter(emitter *events.EventEmitter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.emitter = emitter
}

// SetCompressor sets a custom compressor (for testing or custom strategies).
func (h *History) SetCompressor(c compress.Compressor) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.compressor = c
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
// If compression is enabled and token usage exceeds the threshold,
// automatic compression will be triggered.
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
				if tcMap, ok := tc.(map[string]interface{}); ok {
					// Count function name and arguments
					if function, ok := tcMap["function"].(map[string]interface{}); ok {
						if name, ok := function["name"].(string); ok {
							msg.Tokens += h.tokenizer.Count(name)
						}
						if args, ok := function["arguments"].(string); ok {
							msg.Tokens += h.tokenizer.Count(args)
						}
					}
					// Add tool call overhead (formatting, IDs, etc.)
					msg.Tokens += 8
				}
			}
		}

		// Add message overhead
		msg.Tokens += 4
	}

	h.messages = append(h.messages, msg)

	// Check if compression needed
	if h.shouldCompressLocked() {
		// Use background context with timeout for compression
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.compressLocked(ctx); err != nil {
			// Log error but don't fail (compression is best-effort)
			// In production, this would use proper logging
			_ = err
		}
	}

	return nil
}

// shouldCompressLocked checks if compression threshold exceeded.
// Must be called with lock held.
func (h *History) shouldCompressLocked() bool {
	if h.config == nil || !h.config.CompressionEnabled || h.compressor == nil {
		return false
	}

	totalTokens := h.tokenCountLocked()
	threshold := int(float64(h.maxTokens) * h.config.CompressionThreshold)

	return totalTokens > threshold
}

// compressLocked performs compression.
// Must be called with lock held.
func (h *History) compressLocked(ctx context.Context) error {
	beforeCount := len(h.messages)
	beforeTokens := h.tokenCountLocked()

	// Calculate target tokens (70% of max to give headroom)
	targetTokens := int(float64(h.maxTokens) * 0.7)

	// Convert Message to compress.CompressibleMessage
	compressible := h.toCompressibleMessages(h.messages)

	// Create tokenizer adapter
	tokenizerAdapter := &tokenizerAdapter{tokenizer: h.tokenizer}

	// Perform compression
	compressed, err := h.compressor.Compress(ctx, compressible, targetTokens, tokenizerAdapter)
	if err != nil {
		return err
	}

	// Convert back to Message
	h.messages = h.fromCompressibleMessages(compressed)

	afterCount := len(h.messages)
	afterTokens := h.tokenCountLocked()

	// Emit compression event if emitter available
	h.emitCompressionEvent(beforeCount, beforeTokens, afterCount, afterTokens)

	return nil
}

// emitCompressionEvent sends compression statistics via event emitter.
func (h *History) emitCompressionEvent(beforeCount, beforeTokens, afterCount, afterTokens int) {
	if h.emitter == nil {
		return // No emitter configured
	}

	// Calculate compression ratio
	ratio := 0.0
	if beforeCount > 0 {
		ratio = float64(beforeCount-afterCount) / float64(beforeCount)
	}

	// Format details as JSON-like string for SystemEventData
	details := fmt.Sprintf("messages: %d→%d (%.1f%% reduction), tokens: %d→%d",
		beforeCount, afterCount, ratio*100, beforeTokens, afterTokens)

	// Emit system event
	h.emitter.Emit(events.Event{
		Type:      events.EventInfo,
		Timestamp: time.Now(),
		Data: events.SystemEventData{
			Level:   "info",
			Message: "Context history compressed",
			Details: details,
		},
	})
}

// toCompressibleMessages converts Message to compress.CompressibleMessage
func (h *History) toCompressibleMessages(messages []message.Message) []compress.CompressibleMessage {
	result := make([]compress.CompressibleMessage, len(messages))
	for i, msg := range messages {
		result[i] = compress.CompressibleMessage{
			ID:            msg.ID,
			Role:          string(msg.Role),
			Content:       msg.Content,
			ToolCallCount: len(msg.ToolCalls),
			Tokens:        msg.Tokens,
		}
	}
	return result
}

// fromCompressibleMessages converts compress.CompressibleMessage back to Message
func (h *History) fromCompressibleMessages(messages []compress.CompressibleMessage) []message.Message {
	result := make([]message.Message, len(messages))
	for i, msg := range messages {
		// Find original message to preserve all fields
		original := h.findMessageByID(msg.ID)
		if original != nil {
			result[i] = *original
		} else {
			// Fallback: create minimal message
			result[i] = message.Message{
				ID:        msg.ID,
				Role:      message.Role(msg.Role),
				Content:   msg.Content,
				Tokens:    msg.Tokens,
				Timestamp: time.Now(),
			}
		}
	}
	return result
}

// findMessageByID finds a message in the current history by ID
func (h *History) findMessageByID(id string) *message.Message {
	for i := range h.messages {
		if h.messages[i].ID == id {
			return &h.messages[i]
		}
	}
	return nil
}

// tokenizerAdapter adapts Tokenizer to compress.Tokenizer
type tokenizerAdapter struct {
	tokenizer tokenizer.Tokenizer
}

func (t *tokenizerAdapter) Count(text string) int {
	return t.tokenizer.Count(text)
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
