package protocol

import (
	"time"

	"github.com/google/uuid"
)

// ConversationHistory maintains the message history
type ConversationHistory struct {
	Messages []HistoryMessage `json:"messages"`
}

// HistoryMessage is a single message in conversation history
type HistoryMessage struct {
	Role      Role          `json:"role"`
	Content   []ContentItem `json:"content"`
	Timestamp time.Time     `json:"timestamp"`
}

// Role defines message sender
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// InitialHistory is restored conversation state for resuming
type InitialHistory struct {
	ConversationID ConversationID   `json:"conversation_id"`
	Messages       []HistoryMessage `json:"messages"`
}

// ConversationID uniquely identifies a conversation thread
type ConversationID struct {
	ID uuid.UUID `json:"id"`
}

// NewConversationID generates a new conversation ID
func NewConversationID() ConversationID {
	return ConversationID{ID: uuid.New()}
}

// String returns string representation
func (c ConversationID) String() string {
	return c.ID.String()
}

// ParseConversationID parses a string into a ConversationID
