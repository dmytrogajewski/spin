package delta

import (
	"time"

	"github.com/google/uuid"
)

// Operation represents the type of change to a bullet.
type Operation string

const (
	// OpUpdateContent changes bullet content.
	OpUpdateContent Operation = "update_content"
	// OpIncrementHelpful increments helpful count.
	OpIncrementHelpful Operation = "increment_helpful"
	// OpIncrementHarmful increments harmful count.
	OpIncrementHarmful Operation = "increment_harmful"
	// OpAddTag adds or updates a tag.
	OpAddTag Operation = "add_tag"
	// OpRemoveTag removes a tag.
	OpRemoveTag Operation = "remove_tag"
	// OpUpdateEmbedding updates semantic embedding.
	OpUpdateEmbedding Operation = "update_embedding"
)

// Metadata contains contextual information about a delta.
type Metadata struct {
	Source    string            `json:"source"`     // "reflector", "curator", "adapter", "manual".
	SessionID string            `json:"session_id"` // Adaptation session ID (if applicable).
	Reason    string            `json:"reason"`     // Human-readable reason for change.
	Tags      map[string]string `json:"tags"`       // Arbitrary metadata.
}

// Fields contains operation-specific data.
// Only one field should be set based on the operation type.
type Fields struct {
	// For OpUpdateContent.
	Content *string `json:"content,omitempty"`

	// For OpAddTag, OpRemoveTag.
	TagKey   *string `json:"tag_key,omitempty"`
	TagValue *string `json:"tag_value,omitempty"`

	// For OpUpdateEmbedding.
	Embedding []float32 `json:"embedding,omitempty"`
}

// Delta represents a single change to a bullet.
type Delta struct {
	// ID is the unique identifier for this delta.
	ID string `json:"id"`

	// BulletID is the ID of the bullet being changed.
	BulletID string `json:"bullet_id"`

	// Operation is the type of change.
	Operation Operation `json:"operation"`

	// Fields contains the changes (operation-specific).
	Fields Fields `json:"fields"`

	// Metadata contains contextual information.
	Metadata Metadata `json:"metadata"`

	// CreatedAt is when the delta was created.
	CreatedAt time.Time `json:"created_at"`
}

// NewContentUpdate creates a delta for updating bullet content.
func NewContentUpdate(bulletID, newContent string, metadata Metadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpUpdateContent,
		Fields: Fields{
			Content: &newContent,
		},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// NewIncrementHelpful creates a delta for incrementing helpful count.
func NewIncrementHelpful(bulletID string, metadata Metadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpIncrementHelpful,
		Fields:    Fields{},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// NewIncrementHarmful creates a delta for incrementing harmful count.
func NewIncrementHarmful(bulletID string, metadata Metadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpIncrementHarmful,
		Fields:    Fields{},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// NewAddTag creates a delta for adding or updating a tag.
func NewAddTag(bulletID, key, value string, metadata Metadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpAddTag,
		Fields: Fields{
			TagKey:   &key,
			TagValue: &value,
		},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// NewRemoveTag creates a delta for removing a tag.
func NewRemoveTag(bulletID, key string, metadata Metadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpRemoveTag,
		Fields: Fields{
			TagKey: &key,
		},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// NewUpdateEmbedding creates a delta for updating semantic embedding.
func NewUpdateEmbedding(bulletID string, embedding []float32, metadata Metadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpUpdateEmbedding,
		Fields: Fields{
			Embedding: embedding,
		},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}
