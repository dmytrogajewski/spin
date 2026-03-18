package playbook_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

func TestNew_CreatesEmptyPlaybook(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)

	require.NotNil(t, pb)

	stats := pb.Stats()
	assert.Equal(t, 0, stats.TotalBullets)
	assert.Equal(t, 0, stats.TotalHelpful)
	assert.Equal(t, 0, stats.TotalHarmful)
	assert.InDelta(t, 0.0, stats.AvgScore, 1e-9)
	assert.Equal(t, int64(0), stats.TotalSizeBytes)
}

func TestAdd_AddsNewBullet(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b, err := bullet.New("test content")
	require.NoError(t, err)

	err = pb.Add(ctx, b)
	require.NoError(t, err)

	// Verify bullet was added.
	stats := pb.Stats()
	assert.Equal(t, 1, stats.TotalBullets)

	// Verify can retrieve it.
	retrieved, found := pb.Get(b.ID)
	assert.True(t, found)
	assert.Equal(t, b.ID, retrieved.ID)
	assert.Equal(t, b.Content, retrieved.Content)
}

func TestAdd_RejectsDuplicateID(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b, err := bullet.New("content", bullet.WithID("test-id"))
	require.NoError(t, err)

	err = pb.Add(ctx, b)
	require.NoError(t, err)

	// Try to add bullet with same ID.
	b2, err := bullet.New("different content", bullet.WithID("test-id"))
	require.NoError(t, err)

	err = pb.Add(ctx, b2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestUpdate_UpdatesExistingBullet(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b, err := bullet.New("original")
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b))

	// Modify and update.
	b.Content = "updated"
	b.IncrementHelpful()

	err = pb.Update(ctx, b)
	require.NoError(t, err)

	// Verify update.
	retrieved, found := pb.Get(b.ID)
	assert.True(t, found)
	assert.Equal(t, "updated", retrieved.Content)
	assert.Equal(t, 1, retrieved.HelpfulCount)
}

func TestUpdate_RejectsNonExistent(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b, err := bullet.New("content")
	require.NoError(t, err)

	err = pb.Update(ctx, b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDelete_RemovesBullet(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b, err := bullet.New("content")
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b))

	err = pb.Delete(ctx, b.ID)
	require.NoError(t, err)

	// Verify removed.
	_, found := pb.Get(b.ID)
	assert.False(t, found)

	stats := pb.Stats()
	assert.Equal(t, 0, stats.TotalBullets)
}

func TestDelete_IsIdempotent(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	// Delete non-existent ID should not error.
	err := pb.Delete(ctx, "non-existent")
	assert.NoError(t, err)
}

func TestList_ReturnsAllBullets(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b1, err := bullet.New("bullet 1")
	require.NoError(t, err)
	b2, err := bullet.New("bullet 2")
	require.NoError(t, err)
	b3, err := bullet.New("bullet 3")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b1))
	require.NoError(t, pb.Add(ctx, b2))
	require.NoError(t, pb.Add(ctx, b3))

	bullets := pb.List(nil)
	assert.Len(t, bullets, 3)
}

func TestList_WithFilter(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b1, err := bullet.New("test 1", bullet.WithTags(map[string]string{"category": "security"}))
	require.NoError(t, err)
	b2, err := bullet.New("test 2", bullet.WithTags(map[string]string{"category": "testing"}))
	require.NoError(t, err)
	b3, err := bullet.New("test 3", bullet.WithTags(map[string]string{"category": "security"}))
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b1))
	require.NoError(t, pb.Add(ctx, b2))
	require.NoError(t, pb.Add(ctx, b3))

	// Filter for security category.
	securityBullets := pb.List(func(b *bullet.Bullet) bool {
		return b.Tags["category"] == "security"
	})

	assert.Len(t, securityBullets, 2)
}

func TestSearch_WithNoEmbedder(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	results, err := pb.Search(ctx, "query", 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearch_WithEmbeddings(t *testing.T) {
	t.Parallel()

	embedder := embedding.NewMockEmbedder(3)
	pb := playbook.New(nil, embedder)
	ctx := context.Background()

	// Add bullets with embeddings.
	b1, err := bullet.New("security testing", bullet.WithEmbedding([]float32{1.0, 0.0, 0.0}))
	require.NoError(t, err)
	b2, err := bullet.New("security audit", bullet.WithEmbedding([]float32{0.9, 0.1, 0.0}))
	require.NoError(t, err)
	b3, err := bullet.New("performance test", bullet.WithEmbedding([]float32{0.0, 0.0, 1.0}))
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b1))
	require.NoError(t, pb.Add(ctx, b2))
	require.NoError(t, pb.Add(ctx, b3))

	// Mock query embedding similar to security.
	embedder.SetEmbedding("security check", []float32{0.95, 0.05, 0.0})

	results, err := pb.Search(ctx, "security check", 2)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	// Should return bullets most similar to query.
}

func TestSaveAndLoad(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b1, err := bullet.New("bullet 1")
	require.NoError(t, err)
	b2, err := bullet.New("bullet 2")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b1))
	require.NoError(t, pb.Add(ctx, b2))

	// Save to temp file.
	tmpFile := t.TempDir() + "/playbook.json"
	err = pb.Save(t.Context(), tmpFile)
	require.NoError(t, err)

	// Load from file.
	loaded, err := playbook.Load(tmpFile, nil, nil)
	require.NoError(t, err)

	// Verify loaded playbook has same bullets.
	stats := loaded.Stats()
	assert.Equal(t, 2, stats.TotalBullets)

	_, found := loaded.Get(b1.ID)
	assert.True(t, found)
	_, found = loaded.Get(b2.ID)
	assert.True(t, found)
}

func TestSnapshot_CapturesState(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b1, err := bullet.New("bullet 1")
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b1))

	snapshot := pb.Snapshot()

	require.NotNil(t, snapshot)
	assert.NotEmpty(t, snapshot.ID)
	assert.Len(t, snapshot.Bullets, 1)
	assert.Equal(t, 1, snapshot.Stats.TotalBullets)
}

func TestSnapshot_IsImmutable(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b, err := bullet.New("original")
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b))

	snapshot := pb.Snapshot()

	// Modify original bullet.
	b.Content = "modified"
	require.NoError(t, pb.Update(ctx, b))

	// Snapshot should be unchanged.
	assert.Equal(t, "original", snapshot.Bullets[0].Content)
}

func TestRestore_RestoresFromSnapshot(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b1, err := bullet.New("bullet 1")
	require.NoError(t, err)
	b2, err := bullet.New("bullet 2")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b1))
	require.NoError(t, pb.Add(ctx, b2))

	snapshot := pb.Snapshot()

	// Modify playbook.
	require.NoError(t, pb.Delete(ctx, b1.ID))

	b3, err := bullet.New("bullet 3")
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b3))

	// Restore from snapshot.
	err = pb.Restore(snapshot)
	require.NoError(t, err)

	// Should have original state.
	stats := pb.Stats()
	assert.Equal(t, 2, stats.TotalBullets)

	_, found := pb.Get(b1.ID)
	assert.True(t, found)
	_, found = pb.Get(b3.ID)
	assert.False(t, found)
}

func TestDiff_DetectsChanges(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	b1, err := bullet.New("bullet 1")
	require.NoError(t, err)
	b2, err := bullet.New("bullet 2")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b1))
	require.NoError(t, pb.Add(ctx, b2))

	snapshot1 := pb.Snapshot()

	// Add, remove, modify.
	require.NoError(t, pb.Delete(ctx, b1.ID)) // Remove.

	b2.Content = "modified bullet 2"
	require.NoError(t, pb.Update(ctx, b2)) // Modify.

	b3, err := bullet.New("bullet 3")
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b3)) // Add.

	snapshot2 := pb.Snapshot()

	diff := snapshot1.Diff(snapshot2)
	assert.Len(t, diff.Removed, 1)
	assert.Len(t, diff.Added, 1)
	assert.Len(t, diff.Modified, 1)
}

func TestConcurrentOperations(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	// 10 concurrent writers.
	done := make(chan bool, 20)

	for i := range 10 {
		go func(n int) {
			b, err := bullet.New(fmt.Sprintf("bullet %d", n))
			if err != nil {
				done <- true

				return
			}

			_ = pb.Add(ctx, b)

			done <- true
		}(i)
	}

	// 10 concurrent readers.
	for range 10 {
		go func() {
			pb.List(nil)
			pb.Stats()

			done <- true
		}()
	}

	// Wait for all.
	for range 20 {
		<-done
	}

	stats := pb.Stats()
	assert.Equal(t, 10, stats.TotalBullets)
}

func TestAdd_RejectsNilBullet(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	err := pb.Add(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestUpdate_RejectsNilBullet(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	err := pb.Update(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestRestore_RejectsNilSnapshot(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	err := pb.Restore(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestDiff_WithNilSnapshot(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	snapshot := pb.Snapshot()
	diff := snapshot.Diff(nil)
	assert.NotNil(t, diff)
	assert.Empty(t, diff.Added)
	assert.Empty(t, diff.Removed)
	assert.Empty(t, diff.Modified)
}

func TestLoad_InvalidJSON(t *testing.T) {
	t.Parallel()

	tmpFile := t.TempDir() + "/invalid.json"
	err := os.WriteFile(tmpFile, []byte("invalid json"), 0o600)
	require.NoError(t, err)

	_, err = playbook.Load(tmpFile, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestLoad_NonExistentFile(t *testing.T) {
	t.Parallel()

	_, err := playbook.Load("/nonexistent/file.json", nil, nil)
	require.Error(t, err)
}

func TestSave_ErrorCases(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)

	// Try to save to invalid directory.
	err := pb.Save(t.Context(), "/nonexistent/directory/file.json")
	require.Error(t, err)
}

func TestBulletsEqual_AllFields(t *testing.T) {
	t.Parallel()

	pb := playbook.New(nil, nil)
	ctx := context.Background()

	// Create bullets with all fields different.
	b1, err := bullet.New("content1",
		bullet.WithEmbedding([]float32{1.0, 2.0}),
		bullet.WithTags(map[string]string{"key": "value"}))
	require.NoError(t, err)
	b1.IncrementHelpful()

	b2, err := bullet.New("content2",
		bullet.WithEmbedding([]float32{3.0, 4.0}),
		bullet.WithTags(map[string]string{"key": "other"}))
	require.NoError(t, err)
	b2.IncrementHarmful()

	require.NoError(t, pb.Add(ctx, b1))
	require.NoError(t, pb.Add(ctx, b2))

	snap1 := pb.Snapshot()

	// Modify to test different comparison paths.
	b1.Content = "content2"
	require.NoError(t, pb.Update(ctx, b1))
	snap2 := pb.Snapshot()

	diff := snap1.Diff(snap2)
	assert.NotEmpty(t, diff.Modified)
}

func TestSearch_WithNoBulletsHavingEmbeddings(t *testing.T) {
	t.Parallel()

	embedder := embedding.NewMockEmbedder(3)
	pb := playbook.New(nil, embedder)
	ctx := context.Background()

	// Add bullets WITHOUT embeddings.
	b1, err := bullet.New("test 1")
	require.NoError(t, err)
	b2, err := bullet.New("test 2")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b1))
	require.NoError(t, pb.Add(ctx, b2))

	results, err := pb.Search(ctx, "query", 5)
	require.NoError(t, err)
	assert.Empty(t, results) // No bullets have embeddings.
}

func TestCosineSimilarity_EdgeCases(t *testing.T) {
	t.Parallel()

	embedder := embedding.NewMockEmbedder(3)
	pb := playbook.New(nil, embedder)
	ctx := context.Background()

	// Zero norm vector.
	b1, err := bullet.New("test", bullet.WithEmbedding([]float32{0.0, 0.0, 0.0}))
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b1))

	embedder.SetEmbedding("query", []float32{1.0, 0.0, 0.0})

	results, err := pb.Search(ctx, "query", 5)
	require.NoError(t, err)
	assert.Len(t, results, 1) // Still returns but with 0 similarity.
}
