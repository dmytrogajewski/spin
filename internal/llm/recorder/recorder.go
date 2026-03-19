// Package recorder wraps an LLM provider to capture streaming responses
// as JSONL fixture files for deterministic E2E test replay.
package recorder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// ErrNilProvider indicates a nil provider was passed to New.
var ErrNilProvider = errors.New("recorder: provider must not be nil")

// fixtureResponse matches the JSONL format used by FixtureProvider.
type fixtureResponse struct {
	Chunks []json.RawMessage `json:"chunks"`
}

// Provider wraps an llm.Provider and records all Stream() responses
// to a JSONL fixture file. Each Stream() call produces one JSONL line
// containing all chunks from that response.
type Provider struct {
	inner llm.Provider
	mu    sync.Mutex
	file  *os.File
	enc   *json.Encoder
}

// New creates a recording provider that writes fixture data to the given path.
// The file is created (or truncated) immediately.
func New(inner llm.Provider, path string) (*Provider, error) {
	if inner == nil {
		return nil, ErrNilProvider
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("recorder: create %s: %w", path, err)
	}

	return &Provider{
		inner: inner,
		file:  f,
		enc:   json.NewEncoder(f),
	}, nil
}

// Stream delegates to the inner provider and captures all chunks.
// The returned channel delivers chunks in real-time (no buffering delay).
// After the inner channel closes, the captured chunks are written as one JSONL line.
func (r *Provider) Stream(ctx context.Context, params openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	innerCh, err := r.inner.Stream(ctx, params)
	if err != nil {
		return nil, err
	}

	outCh := make(chan openai.ChatCompletionChunk)

	go func() {
		defer close(outCh)

		var chunks []json.RawMessage

		for chunk := range innerCh {
			// Serialize chunk immediately while it's fresh.
			raw, marshalErr := json.Marshal(chunk)
			if marshalErr == nil {
				chunks = append(chunks, raw)
			}

			// Forward to caller without delay.
			select {
			case outCh <- chunk:
			case <-ctx.Done():
				return
			}
		}

		// Write all chunks as one JSONL line.
		if len(chunks) > 0 {
			r.writeLine(fixtureResponse{Chunks: chunks})
		}
	}()

	return outCh, nil
}

// Complete delegates to the inner provider (not recorded — use Stream for recording).
func (r *Provider) Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return r.inner.Complete(ctx, params)
}

// Models delegates to the inner provider.
func (r *Provider) Models(ctx context.Context) ([]openai.Model, error) {
	return r.inner.Models(ctx)
}

// Capabilities delegates to the inner provider.
func (r *Provider) Capabilities() llm.Capabilities {
	return r.inner.Capabilities()
}

// Name returns the inner provider name with a recording suffix.
func (r *Provider) Name() string {
	return r.inner.Name() + "+recorder"
}

// Close flushes the fixture file and closes both the file and inner provider.
func (r *Provider) Close() error {
	r.mu.Lock()
	fileErr := r.file.Close()
	r.mu.Unlock()

	innerErr := r.inner.Close()

	return errors.Join(fileErr, innerErr)
}

// writeLine appends one JSONL line to the fixture file (thread-safe).
func (r *Provider) writeLine(resp fixtureResponse) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.enc.Encode(resp); err != nil {
		// Best-effort recording — log failures but don't interrupt the session.
		_, _ = fmt.Fprintf(os.Stderr, "recorder: write fixture line: %v\n", err)
	}
}
