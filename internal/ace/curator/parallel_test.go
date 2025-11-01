package curator

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCurateBatch_EmptyRequests tests empty batch
func TestCurateBatch_EmptyRequests(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	cur := NewCurator(pb, embedder)

	req := BatchMergeRequest{
		Requests: []MergeRequest{},
	}

	result, err := cur.CurateBatch(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, len(result.Results))
	assert.Equal(t, 0, len(result.Errors))
}

// TestCurateBatch_SingleRequest tests single request in batch
func TestCurateBatch_SingleRequest(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	cur := NewCurator(pb, embedder)

	insights := []*reflector.Insight{
		{
			Content:    "Always validate input parameters before processing them",
			Confidence: 0.85,
			Category:   reflector.CategorySuccessPattern,
		},
	}

	req := BatchMergeRequest{
		Requests: []MergeRequest{
			{Insights: insights},
		},
	}

	result, err := cur.CurateBatch(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, len(result.Results))
	assert.Equal(t, 1, len(result.Errors))
	assert.Nil(t, result.Errors[0])
	assert.Equal(t, 1, result.Results[0].Added)
	assert.Equal(t, 0, result.Results[0].Skipped)
}

// TestCurateBatch_MultipleRequests tests multiple requests processed in batch
func TestCurateBatch_MultipleRequests(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)

	// Set different embeddings for each insight to avoid duplicate detection
	content1 := "Always validate input"
	content2 := "Use errors.Is for error checking"
	content3 := "Avoid panic in libraries"

	emb1 := make([]float32, 384)
	emb2 := make([]float32, 384)
	emb3 := make([]float32, 384)
	for i := 0; i < 384; i++ {
		emb1[i] = float32(i) / 384.0       // Distinct pattern 1: linear
		emb2[i] = float32(383-i) / 384.0   // Distinct pattern 2: reverse
		emb3[i] = float32(i*i%384) / 384.0 // Distinct pattern 3: quadratic
	}

	embedder.SetEmbedding(content1, emb1)
	embedder.SetEmbedding(content2, emb2)
	embedder.SetEmbedding(content3, emb3)

	pb := playbook.New(nil, embedder)
	cur := NewCurator(pb, embedder)

	req := BatchMergeRequest{
		Requests: []MergeRequest{
			{
				Insights: []*reflector.Insight{
					{Content: content1, Confidence: 0.85, Category: reflector.CategorySuccessPattern},
				},
			},
			{
				Insights: []*reflector.Insight{
					{Content: content2, Confidence: 0.90, Category: reflector.CategorySuccessPattern},
				},
			},
			{
				Insights: []*reflector.Insight{
					{Content: content3, Confidence: 0.80, Category: reflector.CategoryAntiPattern},
				},
			},
		},
	}

	result, err := cur.CurateBatch(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, len(result.Results))
	assert.Equal(t, 3, len(result.Errors))

	// All should succeed
	for i := 0; i < 3; i++ {
		assert.Nil(t, result.Errors[i], "Request %d should not have error", i)
		assert.Equal(t, 1, result.Results[i].Added, "Request %d should add 1 bullet", i)
		assert.Equal(t, 0, result.Results[i].Skipped, "Request %d should skip 0 bullets", i)
	}

	// Playbook should have 3 bullets total
	bullets := pb.List(nil)
	assert.Equal(t, 3, len(bullets))
}
