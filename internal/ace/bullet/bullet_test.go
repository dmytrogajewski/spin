package bullet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
)

func TestNew_AutoGeneratesID(t *testing.T) {
	b, err := bullet.New("test content")

	require.NoError(t, err)
	assert.NotEmpty(t, b.ID)
	assert.Len(t, b.ID, 36) // UUID v4 format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
}

func TestNew_StoresContentAndTimestamps(t *testing.T) {
	content := "Always validate user input"
	before := time.Now()

	b, err := bullet.New(content)

	after := time.Now()
	require.NoError(t, err)
	assert.Equal(t, content, b.Content)
	assert.True(t, b.CreatedAt.After(before) || b.CreatedAt.Equal(before))
	assert.True(t, b.UpdatedAt.Before(after) || b.UpdatedAt.Equal(after))
	assert.Equal(t, b.CreatedAt, b.UpdatedAt) // Initially equal
}

func TestNew_RejectsContentTooLong(t *testing.T) {
	// Create content longer than 2048 characters
	longContent := string(make([]byte, 2049))

	b, err := bullet.New(longContent)

	assert.Error(t, err)
	assert.Nil(t, b)
	assert.Contains(t, err.Error(), "content length")
}

func TestBullet_CloneWithEmbedding(t *testing.T) {
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	original, err := bullet.New("content", bullet.WithEmbedding(embedding))
	require.NoError(t, err)

	clone := original.Clone()

	// Verify embedding is copied
	assert.Equal(t, original.Embedding, clone.Embedding)
	assert.Len(t, clone.Embedding, 5)

	// Verify deep copy (modifying clone doesn't affect original)
	clone.Embedding[0] = 0.9
	assert.Equal(t, float32(0.1), original.Embedding[0])
	assert.Equal(t, float32(0.9), clone.Embedding[0])
}

func TestBullet_CloneWithoutOptionalFields(t *testing.T) {
	// Test cloning bullet without embedding or tags
	original, err := bullet.New("simple content")
	require.NoError(t, err)

	clone := original.Clone()

	assert.Nil(t, clone.Embedding)
	assert.Nil(t, clone.Tags)
	assert.Equal(t, original.Content, clone.Content)
}

func TestBullet_IncrementHelpful(t *testing.T) {
	b, err := bullet.New("test content")
	require.NoError(t, err)

	assert.Equal(t, 0, b.HelpfulCount)

	b.IncrementHelpful()
	assert.Equal(t, 1, b.HelpfulCount)

	b.IncrementHelpful()
	assert.Equal(t, 2, b.HelpfulCount)
}

func TestBullet_IncrementHarmful(t *testing.T) {
	b, err := bullet.New("test content")
	require.NoError(t, err)

	assert.Equal(t, 0, b.HarmfulCount)

	b.IncrementHarmful()
	assert.Equal(t, 1, b.HarmfulCount)

	b.IncrementHarmful()
	assert.Equal(t, 2, b.HarmfulCount)
}

func TestBullet_Score(t *testing.T) {
	tests := []struct {
		name     string
		helpful  int
		harmful  int
		expected float64
	}{
		{
			name:     "no feedback",
			helpful:  0,
			harmful:  0,
			expected: 0.0,
		},
		{
			name:     "only helpful",
			helpful:  5,
			harmful:  0,
			expected: 1.0,
		},
		{
			name:     "only harmful",
			helpful:  0,
			harmful:  5,
			expected: -1.0,
		},
		{
			name:     "more helpful than harmful",
			helpful:  3,
			harmful:  1,
			expected: 0.5, // (3-1)/(3+1) = 2/4 = 0.5
		},
		{
			name:     "more harmful than helpful",
			helpful:  1,
			harmful:  3,
			expected: -0.5, // (1-3)/(1+3) = -2/4 = -0.5
		},
		{
			name:     "equal helpful and harmful",
			helpful:  2,
			harmful:  2,
			expected: 0.0, // (2-2)/(2+2) = 0/4 = 0.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := bullet.New("test content")
			require.NoError(t, err)

			// Set counters
			for i := 0; i < tt.helpful; i++ {
				b.IncrementHelpful()
			}
			for i := 0; i < tt.harmful; i++ {
				b.IncrementHarmful()
			}

			score := b.Score()
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestBullet_Clone(t *testing.T) {
	original, err := bullet.New("original content",
		bullet.WithTags(map[string]string{"key": "value"}),
	)
	require.NoError(t, err)
	original.IncrementHelpful()
	original.IncrementHarmful()

	clone := original.Clone()

	// Verify clone has same values
	assert.Equal(t, original.ID, clone.ID)
	assert.Equal(t, original.Content, clone.Content)
	assert.Equal(t, original.HelpfulCount, clone.HelpfulCount)
	assert.Equal(t, original.HarmfulCount, clone.HarmfulCount)
	assert.Equal(t, original.CreatedAt, clone.CreatedAt)
	assert.Equal(t, original.UpdatedAt, clone.UpdatedAt)
	assert.Equal(t, original.Tags, clone.Tags)

	// Verify clone is independent (modifying clone doesn't affect original)
	clone.Content = "modified content"
	clone.IncrementHelpful()
	clone.Tags["new"] = "tag"

	assert.Equal(t, "original content", original.Content)
	assert.Equal(t, 1, original.HelpfulCount)
	assert.NotContains(t, original.Tags, "new")
}

func TestWithID(t *testing.T) {
	customID := "custom-test-id"
	b, err := bullet.New("content", bullet.WithID(customID))

	require.NoError(t, err)
	assert.Equal(t, customID, b.ID)
}

func TestWithEmbedding(t *testing.T) {
	embedding := []float32{0.1, 0.2, 0.3}
	b, err := bullet.New("content", bullet.WithEmbedding(embedding))

	require.NoError(t, err)
	assert.Equal(t, embedding, b.Embedding)
	assert.Len(t, b.Embedding, 3)
}

func TestWithTags(t *testing.T) {
	tags := map[string]string{
		"category": "security",
		"priority": "high",
	}
	b, err := bullet.New("content", bullet.WithTags(tags))

	require.NoError(t, err)
	assert.Equal(t, tags, b.Tags)
	assert.Equal(t, "security", b.Tags["category"])
	assert.Equal(t, "high", b.Tags["priority"])
}

func TestWithMultipleOptions(t *testing.T) {
	customID := "test-123"
	embedding := []float32{0.5, 0.6}
	tags := map[string]string{"key": "value"}

	b, err := bullet.New("content",
		bullet.WithID(customID),
		bullet.WithEmbedding(embedding),
		bullet.WithTags(tags),
	)

	require.NoError(t, err)
	assert.Equal(t, customID, b.ID)
	assert.Equal(t, embedding, b.Embedding)
	assert.Equal(t, tags, b.Tags)
}
