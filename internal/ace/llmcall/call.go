// Package llmcall provides a generic LLM calling helper for ACE subsystems.
//
// It extracts the common pattern of building messages, calling provider.Complete(),
// extracting the response text, and parsing it into a typed result.
package llmcall

import (
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/pkg/llmutil"
)

// errEmptyResponse is returned when the LLM returns an empty response.
var errEmptyResponse = errors.New("empty LLM response")

// Parser converts raw LLM response text into a typed result.
type Parser[T any] func(response string) (T, error)

// Options configures an LLM call.
type Options struct {
	// Temperature controls randomness (0.0-1.0). Required.
	Temperature float64
	// MaxTokens limits response length. 0 means no limit.
	MaxTokens int
	// Model specifies the model to use. Empty means provider default.
	Model string
	// CleanJSON strips markdown code fences before parsing.
	CleanJSON bool
}

// Call executes an LLM completion and parses the response into type T.
//
// It handles the common pattern: build params → call Complete → extract text → parse.
// The parser function converts the raw response text into the desired type.
func Call[T any](
	ctx context.Context,
	provider llm.Provider,
	messages []openai.ChatCompletionMessageParamUnion,
	parser Parser[T],
	opts Options,
) (T, error) {
	var zero T

	params := openai.ChatCompletionNewParams{
		Messages:    messages,
		Temperature: openai.Float(opts.Temperature),
	}

	if opts.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(opts.MaxTokens))
	}

	if opts.Model != "" {
		params.Model = opts.Model
	}

	completion, err := provider.Complete(ctx, params)
	if err != nil {
		return zero, fmt.Errorf("llm complete: %w", err)
	}

	responseText := extractResponseText(completion)
	if responseText == "" {
		return zero, errEmptyResponse
	}

	if opts.CleanJSON {
		responseText = llmutil.CleanJSONResponse(responseText)
	}

	result, parseErr := parser(responseText)
	if parseErr != nil {
		return zero, fmt.Errorf("parse response: %w", parseErr)
	}

	return result, nil
}

// extractResponseText safely extracts text from a completion response.
func extractResponseText(completion *openai.ChatCompletion) string {
	if completion == nil || len(completion.Choices) == 0 {
		return ""
	}

	return completion.Choices[0].Message.Content
}

// TextParser is a parser that returns the raw response text unchanged.
func TextParser(response string) (string, error) {
	return response, nil
}
