//go:build e2e_llm_test

package testprovider

import (
	"context"
	"strings"
	"sync"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// Provider is a deterministic test-only LLM provider used for E2E flows.
// It is only compiled when built with -tags e2e_llm_test and is never part
// of the production CLI binary.
type Provider struct {
	mu        sync.Mutex
	callCount int
}

// NewProvider creates a new test provider instance.
func NewProvider() llm.Provider {
	return &Provider{}
}

// Complete returns a deterministic ChatCompletion. On the first call, it returns
// a tool call. On subsequent calls (after tool results are provided), it returns
// a final text response to prevent infinite loops.
func (p *Provider) Complete(_ context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if there are tool results in the messages (indicates we're in a follow-up call)
	hasToolResults := false
	for _, msg := range params.Messages {
		// Check if this is a tool message
		if msg.OfTool != nil {
			hasToolResults = true
			break
		}
	}

	// If we have tool results, return final response (this is a follow-up call after tool execution)
	if hasToolResults {
		return &openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "Task completed successfully.",
					},
					FinishReason: "stop",
				},
			},
		}, nil
	}

	// Determine which tool to call based on the prompt
	// Check if the prompt mentions file operations
	promptText := extractPromptText(params.Messages)

	// Choose tool based on prompt content
	var toolCall openai.ChatCompletionMessageToolCall
	content := ""

	lowerPrompt := strings.ToLower(promptText)
	if strings.Contains(lowerPrompt, "execute plan test") {
		// Plan execution test scenario - use shell_command which is always available in ACP mode
		toolCall = openai.ChatCompletionMessageToolCall{
			ID: "test-plan-tool",
			Function: openai.ChatCompletionMessageToolCallFunction{
				Name:      "shell_command",
				Arguments: `{"command":"echo hello","cwd":"","operation":"execute"}`,
			},
		}
		content = "Plan:\n1. Run echo command"
	} else if strings.Contains(lowerPrompt, "read") || strings.Contains(lowerPrompt, "file") && !strings.Contains(lowerPrompt, "write") && !strings.Contains(lowerPrompt, "create") {
		// Read operation
		toolCall = openai.ChatCompletionMessageToolCall{
			ID: "test-read-file",
			Function: openai.ChatCompletionMessageToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path":"test.txt"}`,
			},
		}
		content = "Reading file test.txt"
	} else if strings.Contains(lowerPrompt, "list") || strings.Contains(lowerPrompt, "directory") {
		// List operation
		toolCall = openai.ChatCompletionMessageToolCall{
			ID: "test-list-dir",
			Function: openai.ChatCompletionMessageToolCallFunction{
				Name:      "list_directory",
				Arguments: `{"path":"."}`,
			},
		}
		content = "Listing directory contents"
	} else {
		// Default: write operation (shell_command or write_file)
		if strings.Contains(lowerPrompt, "create") || strings.Contains(lowerPrompt, "write") {
			toolCall = openai.ChatCompletionMessageToolCall{
				ID: "test-write-file",
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      "write_file",
					Arguments: `{"path":"should-not-exist.txt","content":"this should not be created"}`,
				},
			}
			content = "Creating file should-not-exist.txt"
		} else {
			// Default to shell_command
			toolCall = openai.ChatCompletionMessageToolCall{
				ID: "test-shell-command",
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      "shell_command",
					Arguments: `{"command":"echo approval persistence test","cwd":"","operation":"execute"}`,
				},
			}
			content = "Running shell command: echo approval persistence test"
		}
	}

	return &openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content:   content,
					ToolCalls: []openai.ChatCompletionMessageToolCall{toolCall},
				},
				FinishReason: "tool_calls",
			},
		},
	}, nil
}

// extractPromptText extracts text content from the last user message in the message list.
func extractPromptText(messages []openai.ChatCompletionMessageParamUnion) string {
	if len(messages) == 0 {
		return ""
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.OfUser == nil {
		return ""
	}

	// Try string content first
	if lastMsg.OfUser.Content.OfString.Value != "" {
		return lastMsg.OfUser.Content.OfString.Value
	}

	// Try array of content parts
	var promptText string
	for _, part := range lastMsg.OfUser.Content.OfArrayOfContentParts {
		if part.OfText != nil {
			promptText += part.OfText.Text
		}
	}
	return promptText
}

// Stream returns chunks that represent the completion. On first call, returns a tool call.
// On subsequent calls (after tool results are provided), returns a final text response.
func (p *Provider) Stream(_ context.Context, params openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	ch := make(chan openai.ChatCompletionChunk, 10)

	go func() {
		defer close(ch)

		// Check if there are tool results in the messages (indicates we're in a follow-up call)
		hasToolResults := false
		for _, msg := range params.Messages {
			// Check if this is a tool message
			if msg.OfTool != nil {
				hasToolResults = true
				break
			}
		}

		// If we have tool results, return final response (this is a follow-up call after tool execution)
		if hasToolResults {
			// Send a final chunk with the completion message
			ch <- openai.ChatCompletionChunk{
				ID:    "test-chunk-final",
				Model: "test-llm",
				Choices: []openai.ChatCompletionChunkChoice{
					{
						Index: 0,
						Delta: openai.ChatCompletionChunkChoiceDelta{
							Role:    "assistant",
							Content: "Task completed successfully.",
						},
						FinishReason: "stop",
					},
				},
			}
			return
		}

		// First call: return a tool call
		p.mu.Lock()

		// Determine which tool to call based on the prompt
		promptText := extractPromptText(params.Messages)

		// Choose tool based on prompt content
		var toolCall openai.ChatCompletionMessageToolCall
		content := ""

		lowerPrompt := strings.ToLower(promptText)
		if strings.Contains(lowerPrompt, "execute plan test") {
			// Plan execution test scenario - use shell_command which is always available in ACP mode
			toolCall = openai.ChatCompletionMessageToolCall{
				ID: "test-plan-tool",
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      "shell_command",
					Arguments: `{"command":"echo hello","cwd":"","operation":"execute"}`,
				},
			}
			content = "Plan:\n1. Run echo command"
		} else if strings.Contains(lowerPrompt, "read") || strings.Contains(lowerPrompt, "file") && !strings.Contains(lowerPrompt, "write") && !strings.Contains(lowerPrompt, "create") {
			// Read operation
			toolCall = openai.ChatCompletionMessageToolCall{
				ID: "test-read-file",
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"test.txt"}`,
				},
			}
			content = "Reading file test.txt"
		} else if strings.Contains(lowerPrompt, "list") || strings.Contains(lowerPrompt, "directory") {
			// List operation
			toolCall = openai.ChatCompletionMessageToolCall{
				ID: "test-list-dir",
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path":"."}`,
				},
			}
			content = "Listing directory contents"
		} else {
			// Default: write operation (shell_command or write_file)
			if strings.Contains(lowerPrompt, "create") || strings.Contains(lowerPrompt, "write") {
				toolCall = openai.ChatCompletionMessageToolCall{
					ID: "test-write-file",
					Function: openai.ChatCompletionMessageToolCallFunction{
						Name:      "write_file",
						Arguments: `{"path":"should-not-exist.txt","content":"this should not be created"}`,
					},
				}
				content = "Creating file should-not-exist.txt"
			} else {
				// Default to shell_command
				toolCall = openai.ChatCompletionMessageToolCall{
					ID: "test-shell-command",
					Function: openai.ChatCompletionMessageToolCallFunction{
						Name:      "shell_command",
						Arguments: `{"command":"echo approval persistence test","cwd":"","operation":"execute"}`,
					},
				}
				content = "Running shell command: echo approval persistence test"
			}
		}
		p.mu.Unlock()

		// Prepare tool call chunks
		toolCallChunks := []openai.ChatCompletionChunkChoiceDeltaToolCall{
			{
				Index: 0,
				ID:    toolCall.ID,
				Type:  string(toolCall.Type),
				Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				},
			},
		}

		// Send combined chunk with content and tool call
		ch <- openai.ChatCompletionChunk{
			ID:    "test-chunk-combined",
			Model: "test-llm",
			Choices: []openai.ChatCompletionChunkChoice{
				{
					Index: 0,
					Delta: openai.ChatCompletionChunkChoiceDelta{
						Role:      "assistant",
						Content:   content,
						ToolCalls: toolCallChunks,
					},
					FinishReason: "tool_calls",
				},
			},
		}
	}()

	return ch, nil
}

// Models returns an empty model list for the test provider.
func (p *Provider) Models(_ context.Context) ([]openai.Model, error) {
	return nil, nil
}

// Capabilities reports minimal capabilities required for tests.
func (p *Provider) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       false,
		FunctionCalling: true,
		Vision:          false,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "test-llm"
}

// Close implements llm.Provider.Close (no-op for test provider).
func (p *Provider) Close() error {
	return nil
}
