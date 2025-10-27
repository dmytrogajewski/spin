package protocol

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewConversationID(t *testing.T) {
	id1 := NewConversationID()
	id2 := NewConversationID()

	if id1.String() == id2.String() {
		t.Error("NewConversationID should generate unique IDs")
	}

	// Verify it's a valid UUID
	_, err := uuid.Parse(id1.String())
	if err != nil {
		t.Errorf("NewConversationID should generate valid UUID: %v", err)
	}
}

func TestRole_Constants(t *testing.T) {
	roles := []Role{
		RoleUser,
		RoleAssistant,
		RoleSystem,
		RoleTool,
	}

	for _, role := range roles {
		if role == "" {
			t.Error("Role constant should not be empty")
		}
	}
}

func TestConversationHistory(t *testing.T) {
	history := ConversationHistory{
		Messages: []HistoryMessage{
			{
				Role: RoleUser,
				Content: []ContentItem{
					{
						Type: "text",
						Text: &TextContent{Text: "Hello"},
					},
				},
			},
		},
	}

	if len(history.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(history.Messages))
	}

	if history.Messages[0].Role != RoleUser {
		t.Errorf("Expected role %s, got %s", RoleUser, history.Messages[0].Role)
	}
}

func TestInitialHistory(t *testing.T) {
	convID := NewConversationID()
	history := InitialHistory{
		ConversationID: convID,
		Messages: []HistoryMessage{
			{
				Role: RoleSystem,
				Content: []ContentItem{
					{
						Type: "text",
						Text: &TextContent{Text: "System message"},
					},
				},
			},
		},
	}

	if history.ConversationID.String() != convID.String() {
		t.Error("ConversationID mismatch")
	}

	if len(history.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(history.Messages))
	}
}
