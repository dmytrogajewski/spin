package embedding

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
)

var (
	ErrOllamaReturnedNoEmbeddings = errors.New("ollama returned no embeddings")
	ErrExpectedEmbeddingDimension = errors.New("expected embedding dimension")
)

// OllamaEmbedder uses Ollama to generate semantic embeddings.
type OllamaEmbedder struct {
	client    *api.Client
	model     string
	dimension int
}

// OllamaEmbedderConfig configures the Ollama embedder.
type OllamaEmbedderConfig struct {
	// BaseURL is the Ollama server URL (default: http://localhost:11434)
	BaseURL string

	// Model is the embedding model to use (default: nomic-embed-text)
	// Popular models:
	//   - nomic-embed-text (137M params, 768 dims, 8192 context)
	//   - mxbai-embed-large (335M params, 1024 dims, 512 context)
	//   - all-minilm (23M params, 384 dims, 256 context)
	Model string

	// Dimension is the expected embedding dimension
	// Should match the model's output dimension.
	Dimension int
}

// DefaultOllamaEmbedderConfig returns the default configuration.
func DefaultOllamaEmbedderConfig() OllamaEmbedderConfig {
	return OllamaEmbedderConfig{
		BaseURL:   "http://localhost:11434",
		Model:     "nomic-embed-text",
		Dimension: 768, // nomic-embed-text produces 768-dimensional embeddings.
	}
}

// NewOllamaEmbedder creates a new Ollama embedder.
func NewOllamaEmbedder(config OllamaEmbedderConfig) (*OllamaEmbedder, error) {
	// Use defaults if not specified.
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}

	if config.Model == "" {
		config.Model = "nomic-embed-text"
	}

	if config.Dimension == 0 {
		config.Dimension = 768
	}

	// Parse and validate base URL.
	baseURLParsed, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Create Ollama client.
	client := api.NewClient(baseURLParsed, &http.Client{})

	return &OllamaEmbedder{
		client:    client,
		model:     config.Model,
		dimension: config.Dimension,
	}, nil
}

// Embed generates a semantic embedding for the given text.
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// Create embed request.
	req := &api.EmbedRequest{
		Model: e.model,
		Input: text,
	}

	// Call Ollama API.
	resp, err := e.client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed failed: %w", err)
	}

	// Validate response.
	if len(resp.Embeddings) == 0 {
		return nil, ErrOllamaReturnedNoEmbeddings
	}

	embedding := resp.Embeddings[0]

	// Validate dimension.
	if len(embedding) != e.dimension {
return nil, fmt.Errorf("expected embedding dimension %d, got %d: %w", e.dimension, len(embedding), ErrExpectedEmbeddingDimension)
	}

	return embedding, nil
}

// Dimension returns the embedding dimension.
func (e *OllamaEmbedder) Dimension() int {
	return e.dimension
}
