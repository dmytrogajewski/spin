package recorder_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/openai/openai-go"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/recorder"
)

// fakeProvider is a minimal llm.Provider for testing.
type fakeProvider struct {
	chunks []openai.ChatCompletionChunk
}

func (f *fakeProvider) Stream(_ context.Context, _ openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	ch := make(chan openai.ChatCompletionChunk, len(f.chunks))

	for _, c := range f.chunks {
		ch <- c
	}

	close(ch)

	return ch, nil
}

func (f *fakeProvider) Complete(_ context.Context, _ openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return &openai.ChatCompletion{}, nil
}

func (f *fakeProvider) Models(_ context.Context) ([]openai.Model, error) {
	return nil, nil
}

func (f *fakeProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{Streaming: true, FunctionCalling: true}
}

func (f *fakeProvider) Name() string { return "fake" }
func (f *fakeProvider) Close() error { return nil }

func TestRecorder_CapturesStreamChunks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fixturePath := filepath.Join(dir, "recorded.jsonl")

	chunks := []openai.ChatCompletionChunk{
		{
			ID:    "c1",
			Model: "test-model",
			Choices: []openai.ChatCompletionChunkChoice{
				{Delta: openai.ChatCompletionChunkChoiceDelta{Content: "Hello "}},
			},
		},
		{
			ID:    "c2",
			Model: "test-model",
			Choices: []openai.ChatCompletionChunkChoice{
				{Delta: openai.ChatCompletionChunkChoiceDelta{Content: "world!"}, FinishReason: "stop"},
			},
		},
	}

	inner := &fakeProvider{chunks: chunks}

	rec, err := recorder.New(inner, fixturePath)
	require.NoError(t, err)

	// Stream and consume all chunks.
	ctx := context.Background()

	ch, streamErr := rec.Stream(ctx, openai.ChatCompletionNewParams{})
	require.NoError(t, streamErr)

	var received []openai.ChatCompletionChunk

	for c := range ch {
		received = append(received, c)
	}

	require.Len(t, received, 2, "should receive all chunks from inner provider")
	require.Equal(t, "Hello ", received[0].Choices[0].Delta.Content)
	require.Equal(t, "world!", received[1].Choices[0].Delta.Content)

	// Close to flush.
	require.NoError(t, rec.Close())

	// Read back the fixture file and verify format.
	f, openErr := os.Open(fixturePath)
	require.NoError(t, openErr)

	defer f.Close()

	scanner := bufio.NewScanner(f)
	require.True(t, scanner.Scan(), "fixture file should have at least one line")

	var resp struct {
		Chunks []json.RawMessage `json:"chunks"`
	}

	require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
	require.Len(t, resp.Chunks, 2, "fixture line should contain 2 chunks")

	// Verify first chunk content.
	var chunk0 openai.ChatCompletionChunk

	require.NoError(t, json.Unmarshal(resp.Chunks[0], &chunk0))
	require.Equal(t, "Hello ", chunk0.Choices[0].Delta.Content)
}

func TestRecorder_MultipleStreamCalls_MultipleLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fixturePath := filepath.Join(dir, "multi.jsonl")

	inner := &fakeProvider{chunks: []openai.ChatCompletionChunk{
		{ID: "c1", Model: "m", Choices: []openai.ChatCompletionChunkChoice{
			{Delta: openai.ChatCompletionChunkChoiceDelta{Content: "response"}, FinishReason: "stop"},
		}},
	}}

	rec, err := recorder.New(inner, fixturePath)
	require.NoError(t, err)

	ctx := context.Background()

	// Call Stream twice.
	for range 2 {
		ch, streamErr := rec.Stream(ctx, openai.ChatCompletionNewParams{})
		require.NoError(t, streamErr)

		for c := range ch {
			_ = c
		}
	}

	require.NoError(t, rec.Close())

	// Verify 2 lines in fixture file.
	f, openErr := os.Open(fixturePath)
	require.NoError(t, openErr)

	defer f.Close()

	lineScanner := bufio.NewScanner(f)
	lineCount := 0

	for lineScanner.Scan() {
		if len(lineScanner.Bytes()) > 0 {
			lineCount++
		}
	}

	require.Equal(t, 2, lineCount, "should have 2 JSONL lines for 2 Stream() calls")
}

func TestRecorder_NilProvider_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := recorder.New(nil, filepath.Join(t.TempDir(), "out.jsonl"))
	require.ErrorIs(t, err, recorder.ErrNilProvider)
}

func TestRecorder_Name_IncludesSuffix(t *testing.T) {
	t.Parallel()

	rec, err := recorder.New(&fakeProvider{}, filepath.Join(t.TempDir(), "out.jsonl"))
	require.NoError(t, err)

	defer rec.Close()

	require.Equal(t, "fake+recorder", rec.Name())
}

func TestRecorder_Capabilities_DelegatesToInner(t *testing.T) {
	t.Parallel()

	rec, err := recorder.New(&fakeProvider{}, filepath.Join(t.TempDir(), "out.jsonl"))
	require.NoError(t, err)

	defer rec.Close()

	caps := rec.Capabilities()
	require.True(t, caps.Streaming)
	require.True(t, caps.FunctionCalling)
}
