package delta

import (
	"time"

	"github.com/google/uuid"
)

// DeltaOperation represents the type of change to a bullet.
type DeltaOperation string

const (
	// OpUpdateContent changes bullet content
	OpUpdateContent DeltaOperation = "update_content"
	// OpIncrementHelpful increments helpful count
	OpIncrementHelpful DeltaOperation = "increment_helpful"
	// OpIncrementHarmful increments harmful count
	OpIncrementHarmful DeltaOperation = "increment_harmful"
	// OpAddTag adds or updates a tag
	OpAddTag DeltaOperation = "add_tag"
	// OpRemoveTag removes a tag
	OpRemoveTag DeltaOperation = "remove_tag"
	// OpUpdateEmbedding updates semantic embedding
	OpUpdateEmbedding DeltaOperation = "update_embedding"
)

// DeltaMetadata contains contextual information about a delta.
type DeltaMetadata struct {
	Source    string            `json:"source"`     // "reflector", "curator", "adapter", "manual"
	SessionID string            `json:"session_id"` // Adaptation session ID (if applicable)
	Reason    string            `json:"reason"`     // Human-readable reason for change
	Tags      map[string]string `json:"tags"`       // Arbitrary metadata
}

// DeltaFields contains operation-specific data.
// Only one field should be set based on the operation type.
type DeltaFields struct {
	// For OpUpdateContent
	Content *string `json:"content,omitempty"`

	// For OpAddTag, OpRemoveTag
	TagKey   *string `json:"tag_key,omitempty"`
	TagValue *string `json:"tag_value,omitempty"`

	// For OpUpdateEmbedding
	Embedding []float32 `json:"embedding,omitempty"`
}

// Delta represents a single change to a bullet.
type Delta struct {
	// ID is the unique identifier for this delta
	ID string `json:"id"`

	// BulletID is the ID of the bullet being changed
	BulletID string `json:"bullet_id"`

	// Operation is the type of change
	Operation DeltaOperation `json:"operation"`

	// Fields contains the changes (operation-specific)
	Fields DeltaFields `json:"fields"`

	// Metadata contains contextual information
	Metadata DeltaMetadata `json:"metadata"`

	// CreatedAt is when the delta was created
	CreatedAt time.Time `json:"created_at"`
}

// NewContentUpdate creates a delta for updating bullet content.
func NewContentUpdate(bulletID, newContent string, metadata DeltaMetadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpUpdateContent,
		Fields: DeltaFields{
			Content: &newContent,
		},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// NewIncrementHelpful creates a delta for incrementing helpful count.
func NewIncrementHelpful(bulletID string, metadata DeltaMetadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpIncrementHelpful,
		Fields:    DeltaFields{},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// NewIncrementHarmful creates a delta for incrementing harmful count.
func NewIncrementHarmful(bulletID string, metadata DeltaMetadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpIncrementHarmful,
		Fields:    DeltaFields{},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// NewAddTag creates a delta for adding or updating a tag.
func NewAddTag(bulletID, key, value string, metadata DeltaMetadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpAddTag,
		Fields: DeltaFields{
			TagKey:   &key,
			TagValue: &value,
		},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// NewRemoveTag creates a delta for removing a tag.
func NewRemoveTag(bulletID, key string, metadata DeltaMetadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpRemoveTag,
		Fields: DeltaFields{
			TagKey: &key,
		},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// NewUpdateEmbedding creates a delta for updating semantic embedding.
func NewUpdateEmbedding(bulletID string, embedding []float32, metadata DeltaMetadata) *Delta {
	return &Delta{
		ID:        uuid.New().String(),
		BulletID:  bulletID,
		Operation: OpUpdateEmbedding,
		Fields: DeltaFields{
			Embedding: embedding,
		},
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}
