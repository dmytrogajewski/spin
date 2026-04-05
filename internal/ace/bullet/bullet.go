// Package bullet provides bullet-point extraction and formatting.
package bullet

import (
	"maps"
	"time"

	"github.com/google/uuid"
)

// Bullet represents a single unit of context knowledge.
// It stores reusable strategies, domain concepts, or failure modes
// that can be accumulated and refined over time.
type Bullet struct {
	// ID is the unique identifier (UUID v4).
	ID string `json:"id"`

	// Content is the actual knowledge content.
	Content string `json:"content"`

	// HelpfulCount tracks how often this bullet was marked helpful.
	HelpfulCount int `json:"helpful_count"`

	// HarmfulCount tracks how often this bullet was marked harmful.
	HarmfulCount int `json:"harmful_count"`

	// Embedding is the semantic vector (optional)
	// Dimension: 1536 (OpenAI text-embedding-ada-002 compatible).
	Embedding []float32 `json:"embedding,omitempty"`

	// CreatedAt is when the bullet was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the bullet was last modified.
	UpdatedAt time.Time `json:"updated_at"`

	// Tags are arbitrary metadata key-value pairs.
	Tags map[string]string `json:"tags,omitempty"`
}

// New creates a new bullet with auto-generated ID and timestamps.
func New(content string, opts ...Option) (*Bullet, error) {
	now := time.Now()
	b := &Bullet{
		ID:           uuid.New().String(),
		Content:      content,
		HelpfulCount: 0,
		HarmfulCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	for _, opt := range opts {
		opt(b)
	}

	// Validate before returning.
	err := validate(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// IncrementHelpful increments the helpful counter.
func (b *Bullet) IncrementHelpful() {
	b.HelpfulCount++
}

// IncrementHarmful increments the harmful counter.
func (b *Bullet) IncrementHarmful() {
	b.HarmfulCount++
}

// Score returns a utility score between -1.0 and 1.0 based on counters.
// Score is calculated as (helpful - harmful) / (helpful + harmful).
// Returns 0.0 if there is no feedback.
func (b *Bullet) Score() float64 {
	total := b.HelpfulCount + b.HarmfulCount
	if total == 0 {
		return 0.0
	}

	return float64(b.HelpfulCount-b.HarmfulCount) / float64(total)
}

// Clone creates a deep copy of the bullet.
func (b *Bullet) Clone() *Bullet {
	clone := &Bullet{
		ID:           b.ID,
		Content:      b.Content,
		HelpfulCount: b.HelpfulCount,
		HarmfulCount: b.HarmfulCount,
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}

	// Deep copy embedding if present.
	if len(b.Embedding) > 0 {
		clone.Embedding = make([]float32, len(b.Embedding))
		copy(clone.Embedding, b.Embedding)
	}

	// Deep copy tags if present.
	if len(b.Tags) > 0 {
		clone.Tags = make(map[string]string, len(b.Tags))
		maps.Copy(clone.Tags, b.Tags)
	}

	return clone
}

// Option is a functional option for bullet creation.
type Option func(*Bullet)

// WithID sets a custom ID for the bullet.
func WithID(id string) Option {
	return func(b *Bullet) {
		b.ID = id
	}
}

// WithEmbedding sets the semantic embedding for the bullet.
func WithEmbedding(embedding []float32) Option {
	return func(b *Bullet) {
		b.Embedding = embedding
	}
}

// WithTags sets metadata tags for the bullet.
func WithTags(tags map[string]string) Option {
	return func(b *Bullet) {
		b.Tags = tags
	}
}
