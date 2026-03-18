//go:build e2e_llm_test

package testprovider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// fixtureResponse represents one LLM response in a fixture file.
// Each line of the JSONL fixture file unmarshals into one fixtureResponse.
// The chunks are sent sequentially on the streaming channel.
// Optional delay_ms pauses before sending chunks (for timeout testing).
type fixtureResponse struct {
	Chunks  []json.RawMessage `json:"chunks"`
	DelayMS int               `json:"delay_ms,omitempty"`
}

// FixtureProvider implements llm.Provider by replaying responses from a JSONL fixture file.
// Each call to Stream() returns the next response in sequence.
// This enables deterministic, fixture-driven E2E tests for the spin exec command.
//
// Fixture format (JSONL — one JSON object per line):
//
//	{"chunks":[{"id":"c1","model":"fix","object":"chat.completion.chunk","created":0,"choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":"stop"}]}]}
//
// Each line corresponds to one Stream() call. Lines are consumed sequentially.
type FixtureProvider struct {
	mu        sync.Mutex
	responses []fixtureResponse
	callIdx   int
}

// NewFixtureProvider creates a provider that replays responses from a JSONL fixture file.
func NewFixtureProvider(path string) (*FixtureProvider, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open fixture %s: %w", path, err)
	}
	defer f.Close()

	var responses []fixtureResponse

	scanner := bufio.NewScanner(f)
	// Increase buffer for large fixture lines.
	const maxLineSize = 1024 * 1024
	scanner.Buffer(make([]byte, 0, maxLineSize), maxLineSize)

	lineNum := 0
	for scanner.Scan() {
		lineNum++

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp fixtureResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			return nil, fmt.Errorf("fixture %s line %d: %w", path, lineNum, err)
		}

		if len(resp.Chunks) == 0 {
			return nil, fmt.Errorf("fixture %s line %d: no chunks", path, lineNum)
		}

		responses = append(responses, resp)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read fixture %s: %w", path, err)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("fixture %s: empty file", path)
	}

	return &FixtureProvider{responses: responses}, nil
}

// Stream returns chunks from the next fixture response.
func (p *FixtureProvider) Stream(ctx context.Context, _ openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	p.mu.Lock()
	idx := p.callIdx
	p.callIdx++
	p.mu.Unlock()

	if idx >= len(p.responses) {
		return nil, fmt.Errorf("fixture provider: no more responses (call %d, have %d)", idx+1, len(p.responses))
	}

	resp := p.responses[idx]
	ch := make(chan openai.ChatCompletionChunk, len(resp.Chunks))

	go func() {
		defer close(ch)

		// Apply delay if specified (for timeout testing).
		if resp.DelayMS > 0 {
			select {
			case <-time.After(time.Duration(resp.DelayMS) * time.Millisecond):
			case <-ctx.Done():
				return
			}
		}

		for i, raw := range resp.Chunks {
			var chunk openai.ChatCompletionChunk
			if err := json.Unmarshal(raw, &chunk); err != nil {
				// Send an empty stop chunk so the caller doesn't hang.
				ch <- openai.ChatCompletionChunk{
					ID:    fmt.Sprintf("fixture-error-%d", i),
					Model: "fixture",
					Choices: []openai.ChatCompletionChunkChoice{
						{FinishReason: openai.ChatCompletionChunkChoicesFinishReasonStop},
					},
				}
				return
			}

			select {
			case <-ctx.Done():
				return
			case ch <- chunk:
			}
		}
	}()

	return ch, nil
}

// Complete accumulates a streamed response into a ChatCompletion.
func (p *FixtureProvider) Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	ch, err := p.Stream(ctx, params)
	if err != nil {
		return nil, err
	}

	acc := openai.ChatCompletionAccumulator{}

	for chunk := range ch {
		acc.AddChunk(chunk)
	}

	return &acc.ChatCompletion, nil
}

// Models returns an empty model list.
func (p *FixtureProvider) Models(_ context.Context) ([]openai.Model, error) {
	return nil, nil
}

// Capabilities reports streaming and function calling support.
func (p *FixtureProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: true,
	}
}

// Name returns the provider name.
func (p *FixtureProvider) Name() string {
	return "fixture-llm"
}

// Close is a no-op.
func (p *FixtureProvider) Close() error {
	return nil
}
