package adapter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

func TestMemoryConfig_Defaults(t *testing.T) {
	t.Parallel()

	config := DefaultMemoryConfig()

	assert.Equal(t, 1000, config.MaxBullets)
	assert.Equal(t, 900, config.RefinementAt)
	assert.InDelta(t, 0.2, config.PruneThreshold, 1e-9)
}

func TestMemoryManager_ShouldRefine_BelowThreshold(t *testing.T) {
	t.Parallel()

	config := MemoryConfig{
		MaxBullets:     1000,
		RefinementAt:   900,
		PruneThreshold: 0.2,
	}

	manager := NewMemoryManager(config)

	// Playbook with 800 bullets (below threshold).
	shouldRefine := manager.ShouldRefine(800)

	assert.False(t, shouldRefine)
}

func TestMemoryManager_ShouldRefine_AboveThreshold(t *testing.T) {
	t.Parallel()

	config := MemoryConfig{
		MaxBullets:     1000,
		RefinementAt:   900,
		PruneThreshold: 0.2,
	}

	manager := NewMemoryManager(config)

	// Playbook with 950 bullets (above threshold).
	shouldRefine := manager.ShouldRefine(950)

	assert.True(t, shouldRefine)
}

func TestMemoryManager_CalculateUtility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		helpful  int
		harmful  int
		expected float64
	}{
		{
			name:     "high utility",
			helpful:  10,
			harmful:  2,
			expected: 0.6666666666666666, // (10-2)/(10+2)
		},
		{
			name:     "zero utility",
			helpful:  5,
			harmful:  5,
			expected: 0.0, // (5-5)/(5+5)
		},
		{
			name:     "negative utility",
			helpful:  2,
			harmful:  10,
			expected: -0.6666666666666666, // (2-10)/(2+10)
		},
		{
			name:     "no feedback",
			helpful:  0,
			harmful:  0,
			expected: 0.0, // Score() returns 0 when no feedback.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b, err := bullet.New("test content")
			require.NoError(t, err)

			// Set counters.
			for range tt.helpful {
				b.IncrementHelpful()
			}

			for range tt.harmful {
				b.IncrementHarmful()
			}

			score := b.Score()
			assert.InDelta(t, tt.expected, score, 0.0001)
		})
	}
}

func TestMemoryManager_Prune(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := MemoryConfig{
		MaxBullets:     1000,
		RefinementAt:   900,
		PruneThreshold: 0.2, // Remove bullets with utility < 0.2.
	}

	manager := NewMemoryManager(config)

	// Create playbook with bullets of varying utility.
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	// High utility bullet (should keep).
	b1, _ := bullet.New("High utility bullet")
	for range 10 {
		b1.IncrementHelpful()
	}

	b1.IncrementHarmful()
	emb1, _ := embedder.Embed(ctx, b1.Content)
	b1.Embedding = emb1
	_ = pb.Add(ctx, b1)

	// Low utility bullet (should prune).
	b2, _ := bullet.New("Low utility bullet")
	b2.IncrementHelpful()
	b2.IncrementHarmful()
	b2.IncrementHarmful()
	b2.IncrementHarmful()
	emb2, _ := embedder.Embed(ctx, b2.Content)
	b2.Embedding = emb2
	_ = pb.Add(ctx, b2)

	// Zero utility bullet (should prune).
	b3, _ := bullet.New("Zero utility bullet")
	emb3, _ := embedder.Embed(ctx, b3.Content)
	b3.Embedding = emb3
	_ = pb.Add(ctx, b3)

	// Initial count.
	assert.Equal(t, 3, pb.Stats().TotalBullets)

	// Prune.
	pruned, err := manager.Prune(ctx, pb)
	require.NoError(t, err)

	// Should have pruned 2 bullets.
	assert.Equal(t, 2, pruned)
	assert.Equal(t, 1, pb.Stats().TotalBullets)

	// Only high utility bullet should remain.
	remaining, found := pb.Get(b1.ID)
	assert.True(t, found)
	assert.Equal(t, "High utility bullet", remaining.Content)
}
