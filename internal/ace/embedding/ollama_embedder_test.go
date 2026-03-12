package embedding

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOllamaEmbedder(t *testing.T) {
	t.Parallel()

	config := DefaultOllamaEmbedderConfig()
	embedder, err := NewOllamaEmbedder(config)

	require.NoError(t, err)
	assert.NotNil(t, embedder)
	assert.Equal(t, 768, embedder.Dimension())
	assert.Equal(t, "nomic-embed-text", embedder.model)
}

func TestNewOllamaEmbedder_CustomConfig(t *testing.T) {
	t.Parallel()

	config := OllamaEmbedderConfig{
		BaseURL:   "http://custom:11434",
		Model:     "mxbai-embed-large",
		Dimension: 1024,
	}

	embedder, err := NewOllamaEmbedder(config)

	require.NoError(t, err)
	assert.NotNil(t, embedder)
	assert.Equal(t, 1024, embedder.Dimension())
	assert.Equal(t, "mxbai-embed-large", embedder.model)
}

func TestNewOllamaEmbedder_InvalidURL(t *testing.T) {
	t.Parallel()

	config := OllamaEmbedderConfig{
		BaseURL: "://invalid-url",
	}

	_, err := NewOllamaEmbedder(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base URL")
}

// makeTestEmbedding generates a deterministic fake embedding of the given dimension.
// The seed value makes each embedding unique and reproducible.
func makeTestEmbedding(dimension int, seed float64) []float64 {
	emb := make([]float64, dimension)
	for i := range emb {
		emb[i] = math.Sin(seed*float64(i+1)*0.1) * 0.5
	}

	return emb
}

// newMockOllamaServer creates an httptest server that handles /api/embed requests.
// The handler function receives the parsed request body and returns the embedding to use.
func newMockOllamaServer(t *testing.T, handler func(model string, input string) ([]float64, error)) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			http.NotFound(w, r)

			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

			return
		}

		var reqBody struct {
			Model string `json:"model"`
			Input string `json:"input"`
		}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)

			return
		}

		embedding, err := handler(reqBody.Model, reqBody.Input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		resp := map[string]any{
			"model":      reqBody.Model,
			"embeddings": [][]float64{embedding},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestOllamaEmbedder_Embed(t *testing.T) {
	t.Parallel()

	const dim = 768

	fakeEmbedding := makeTestEmbedding(dim, 1.0)

	server := newMockOllamaServer(t, func(model, input string) ([]float64, error) {
		assert.Equal(t, "nomic-embed-text", model)
		assert.Equal(t, "Hello world", input)

		return fakeEmbedding, nil
	})
	defer server.Close()

	config := OllamaEmbedderConfig{
		BaseURL:   server.URL,
		Model:     "nomic-embed-text",
		Dimension: dim,
	}
	embedder, err := NewOllamaEmbedder(config)
	require.NoError(t, err)

	ctx := context.Background()
	embedding, err := embedder.Embed(ctx, "Hello world")

	require.NoError(t, err)
	assert.NotNil(t, embedding)
	assert.Equal(t, dim, len(embedding), "Expected 768-dimensional embedding")

	// Verify embedding is not all zeros.
	hasNonZero := false

	for _, val := range embedding {
		if val != 0 {
			hasNonZero = true

			break
		}
	}

	assert.True(t, hasNonZero, "Embedding should not be all zeros")

	// Verify values match what the mock returned.
	for i, val := range embedding {
		assert.InDelta(t, fakeEmbedding[i], float64(val), 1e-5, "Embedding value mismatch at index %d", i)
	}
}

func TestOllamaEmbedder_Embed_Similarity(t *testing.T) {
	t.Parallel()

	const dim = 768

	// Create three distinct embeddings where embed1 and embed2 are similar,
	// but embed3 is very different.
	// Note: the cosineSimilarity helper divides by normA*normB (not sqrt),
	// so we use unit-length-like vectors to get meaningful similarity values.
	embed1Data := make([]float64, dim)
	embed2Data := make([]float64, dim)
	embed3Data := make([]float64, dim)
	// Make embed1 a unit vector along first component.
	embed1Data[0] = 1.0
	// Make embed2 very close to embed1.
	embed2Data[0] = 0.99
	embed2Data[1] = 0.14 // small orthogonal component.
	// Make embed3 orthogonal to embed1.
	embed3Data[1] = 1.0

	embeddings := map[string][]float64{
		"The cat sits on the mat":        embed1Data,
		"A feline rests on the rug":      embed2Data,
		"Quantum physics and relativity": embed3Data,
	}

	server := newMockOllamaServer(t, func(_, input string) ([]float64, error) {
		emb, ok := embeddings[input]
		require.True(t, ok, "Unexpected input text: %s", input)

		return emb, nil
	})
	defer server.Close()

	config := OllamaEmbedderConfig{
		BaseURL:   server.URL,
		Model:     "nomic-embed-text",
		Dimension: dim,
	}
	embedder, err := NewOllamaEmbedder(config)
	require.NoError(t, err)

	ctx := context.Background()

	e1, err := embedder.Embed(ctx, "The cat sits on the mat")
	require.NoError(t, err)

	e2, err := embedder.Embed(ctx, "A feline rests on the rug")
	require.NoError(t, err)

	e3, err := embedder.Embed(ctx, "Quantum physics and relativity")
	require.NoError(t, err)

	// Calculate cosine similarities.
	sim12 := cosineSimilarity(e1, e2)
	sim13 := cosineSimilarity(e1, e3)

	// Similar texts should have higher similarity than dissimilar ones.
	assert.Greater(t, sim12, sim13, "Similar texts should have higher cosine similarity")
	assert.Greater(t, sim12, 0.5, "Similar texts should have similarity > 0.5")
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (normA * normB)
}
