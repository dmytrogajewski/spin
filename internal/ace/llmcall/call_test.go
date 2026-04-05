package llmcall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/openai/openai-go"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/llm"
)

var (
	errProviderDown       = errors.New("provider unavailable")
	errStreamNotSupported = errors.New("stream not supported in mock")
	errModelsNotSupported = errors.New("models not supported in mock")
)

type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Complete(_ context.Context, _ openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{
			{Message: openai.ChatCompletionMessage{Content: m.response}},
		},
	}, nil
}

func (m *mockProvider) Stream(_ context.Context, _ openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	return nil, errStreamNotSupported
}

func (m *mockProvider) Models(_ context.Context) ([]openai.Model, error) {
	return nil, errModelsNotSupported
}

func (m *mockProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{}
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Close() error { return nil }

type testResult struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

func TestCall_Success(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{response: `{"name":"test","score":42}`}

	result, err := Call(
		context.Background(),
		provider,
		[]openai.ChatCompletionMessageParamUnion{openai.UserMessage("hello")},
		func(resp string) (testResult, error) {
			var r testResult

			if err := json.Unmarshal([]byte(resp), &r); err != nil {
				return r, fmt.Errorf("unmarshal: %w", err)
			}

			return r, nil
		},
		Options{Temperature: 0.3, CleanJSON: false},
	)

	require.NoError(t, err)
	require.Equal(t, "test", result.Name)
	require.Equal(t, 42, result.Score)
}

func TestCall_CleanJSON(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{response: "```json\n{\"name\":\"cleaned\"}\n```"}

	result, err := Call(
		context.Background(),
		provider,
		[]openai.ChatCompletionMessageParamUnion{openai.UserMessage("hello")},
		func(resp string) (testResult, error) {
			var r testResult

			if err := json.Unmarshal([]byte(resp), &r); err != nil {
				return r, fmt.Errorf("unmarshal: %w", err)
			}

			return r, nil
		},
		Options{Temperature: 0.3, CleanJSON: true},
	)

	require.NoError(t, err)
	require.Equal(t, "cleaned", result.Name)
}

func TestCall_ProviderError(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{err: errProviderDown}

	_, err := Call(
		context.Background(),
		provider,
		[]openai.ChatCompletionMessageParamUnion{openai.UserMessage("hello")},
		TextParser,
		Options{Temperature: 0.3},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, errProviderDown)
}

func TestCall_EmptyResponse(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{response: ""}

	_, err := Call(
		context.Background(),
		provider,
		[]openai.ChatCompletionMessageParamUnion{openai.UserMessage("hello")},
		TextParser,
		Options{Temperature: 0.3},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, errEmptyResponse)
}

func TestCall_ParseError(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{response: "not json"}

	_, err := Call(
		context.Background(),
		provider,
		[]openai.ChatCompletionMessageParamUnion{openai.UserMessage("hello")},
		func(resp string) (testResult, error) {
			var r testResult

			if err := json.Unmarshal([]byte(resp), &r); err != nil {
				return r, fmt.Errorf("unmarshal: %w", err)
			}

			return r, nil
		},
		Options{Temperature: 0.3},
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "parse response")
}

func TestCall_TextParser(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{response: "plain text response"}

	result, err := Call(
		context.Background(),
		provider,
		[]openai.ChatCompletionMessageParamUnion{openai.UserMessage("hello")},
		TextParser,
		Options{Temperature: 0.7},
	)

	require.NoError(t, err)
	require.Equal(t, "plain text response", result)
}

func TestCall_WithMaxTokens(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{response: "ok"}

	result, err := Call(
		context.Background(),
		provider,
		[]openai.ChatCompletionMessageParamUnion{openai.UserMessage("hello")},
		TextParser,
		Options{Temperature: 0.3, MaxTokens: 4096},
	)

	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestExtractResponseText(t *testing.T) {
	t.Parallel()

	t.Run("nil_completion", func(t *testing.T) {
		t.Parallel()

		got := extractResponseText(nil)
		require.Empty(t, got)
	})

	t.Run("empty_choices", func(t *testing.T) {
		t.Parallel()

		got := extractResponseText(&openai.ChatCompletion{})
		require.Empty(t, got)
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		got := extractResponseText(&openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: "hello"}},
			},
		})
		require.Equal(t, "hello", got)
	})
}
