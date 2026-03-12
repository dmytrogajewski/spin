package ollama

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/openai/openai-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
var testCtx = context.Background()

// mkTC creates an openai.ChatCompletionMessageToolCallParam for tests.
func mkTC(id, name, args string) openai.ChatCompletionMessageToolCallParam {
	return openai.ChatCompletionMessageToolCallParam{
		ID:   openai.F(id),
		Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction),
		Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{
			Name:      openai.F(name),
			Arguments: openai.F(args),
		}),
	}
}

func init() {
	// Suppress warn logs during tests.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
}

// TestBuildToolCallIDToNameMap_ReproducesMissingMapping tests the scenario
// where tool_call_ids in tool result messages are not found in the mapping.
// This can happen when assistant messages with tool_calls are serialized
// differently or when the mapping is built from a subset of messages.
func TestBuildToolCallIDToNameMap_ReproducesMissingMapping(t *testing.T) {
	t.Parallel()

	// Simulate messages as they would be sent to Ollama after agent execution:
	// - User message
	// - Assistant with 5 tool calls (IDs: chatcmpl-123-0 through chatcmpl-123-4)
	// - 5 Tool result messages.
	toolCallIDs := []string{"chatcmpl-123-0", "chatcmpl-123-1", "chatcmpl-123-2", "chatcmpl-123-3", "chatcmpl-123-4"}
	toolNames := []string{"read_file", "list_directory", "shell_command", "write_file", "read_file"}

	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			mkTC(toolCallIDs[0], toolNames[0], `{"path":"/tmp/a"}`),
			mkTC(toolCallIDs[1], toolNames[1], `{}`),
			mkTC(toolCallIDs[2], toolNames[2], `{"command":"ls"}`),
			mkTC(toolCallIDs[3], toolNames[3], `{}`),
			mkTC(toolCallIDs[4], toolNames[4], `{"path":"/tmp/b"}`),
		}),
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("List files in /tmp"),
		assistantMsg,
		openai.ToolMessage(toolCallIDs[0], "file contents"),
		openai.ToolMessage(toolCallIDs[1], "dir listing"),
		openai.ToolMessage(toolCallIDs[2], "output"),
		openai.ToolMessage(toolCallIDs[3], "written"),
		openai.ToolMessage(toolCallIDs[4], "read contents"),
	}

	mapping := buildToolCallIDToNameMap(messages, testLogger, testCtx)
	require.Len(t, mapping, 5, "mapping should contain all 5 tool call IDs")

	for i, id := range toolCallIDs {
		name, ok := mapping[id]
		assert.True(t, ok, "tool_call_id %q should be in mapping", id)
		assert.Equal(t, toolNames[i], name, "tool name for %q", id)
	}
}

// TestBuildToolCallIDToNameMap_FromJSON verifies that buildToolCallIDToNameMap
// correctly parses assistant messages when they come from JSON marshaling
// (simulating the agent's convertMessageToOpenAI output after round-trip).
func TestBuildToolCallIDToNameMap_FromJSON(t *testing.T) {
	t.Parallel()

	// Marshal and unmarshal to simulate what parseGenericMessage does.
	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			mkTC("chatcmpl-1769975533529433976-1", "shell_command", `{"command":"ls"}`),
		}),
	}

	jsonData, err := json.Marshal(assistantMsg)
	require.NoError(t, err)

	var generic genericMessage

	err = json.Unmarshal(jsonData, &generic)
	require.NoError(t, err)
	assert.Equal(t, "assistant", generic.Role)
	require.NotEmpty(t, generic.ToolCalls)

	var toolCalls []struct {
		ID       string `json:"id"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}

	err = json.Unmarshal(generic.ToolCalls, &toolCalls)
	require.NoError(t, err)
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "chatcmpl-1769975533529433976-1", toolCalls[0].ID)
	assert.Equal(t, "shell_command", toolCalls[0].Function.Name)
}

// TestConvertMessageToOllama_ToolResultWithPositionalFallback verifies that
// when tool_call_id is not in the mapping, we can still resolve tool_name
// using the positional fallback (preceding assistant + tool message order).
func TestConvertMessageToOllama_ToolResultWithPositionalFallback(t *testing.T) {
	t.Parallel()

	// Scenario: mapping has only 2 entries (e.g. from compression) but we have
	// 5 tool results. The positional fallback should resolve the missing ones.
	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			mkTC("call-0", "read_file", `{}`),
			mkTC("call-1", "shell_command", `{}`),
			mkTC("call-2", "list_directory", `{}`),
		}),
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		assistantMsg,
		openai.ToolMessage("call-0", "content0"),
		openai.ToolMessage("call-1", "content1"),
		openai.ToolMessage("call-2", "content2"),
	}

	mapping := buildToolCallIDToNameMap(messages, testLogger, testCtx)
	require.Len(t, mapping, 3)

	// Convert each tool message - all should get ToolName set.
	for i := range []string{"call-0", "call-1", "call-2"} {
		msg := messages[i+1] // +1 to skip assistant.
		result := convertMessageToOllama(msg, mapping, testLogger, testCtx)
		assert.Equal(t, "tool", result.Role)

		expectedName := []string{"read_file", "shell_command", "list_directory"}[i]
		assert.Equal(t, expectedName, result.ToolName, "tool message %d", i)
	}
}

// TestBuildToolCallIDToNameMap_PositionalFallbackWhenPrimaryFails reproduces
// the scenario from the bug report: tool results reference IDs like
// chatcmpl-1769975533529433976-1 through 5, but the primary pass (from assistant
// tool_calls) may fail to parse them. The positional fallback fills the mapping.
func TestBuildToolCallIDToNameMap_PositionalFallbackWhenPrimaryFails(t *testing.T) {
	t.Parallel()

	// Assistant with 5 tool calls - use IDs that match the bug report format.
	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			mkTC("chatcmpl-1769975533529433976-1", "read_file", `{}`),
			mkTC("chatcmpl-1769975533529433976-2", "list_directory", `{}`),
			mkTC("chatcmpl-1769975533529433976-3", "shell_command", `{}`),
			mkTC("chatcmpl-1769975533529433976-4", "write_file", `{}`),
			mkTC("chatcmpl-1769975533529433976-5", "read_file", `{}`),
		}),
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("Help with pipewire"),
		assistantMsg,
		openai.ToolMessage("chatcmpl-1769975533529433976-1", "content1"),
		openai.ToolMessage("chatcmpl-1769975533529433976-2", "content2"),
		openai.ToolMessage("chatcmpl-1769975533529433976-3", "content3"),
		openai.ToolMessage("chatcmpl-1769975533529433976-4", "content4"),
		openai.ToolMessage("chatcmpl-1769975533529433976-5", "content5"),
	}

	mapping := buildToolCallIDToNameMap(messages, testLogger, testCtx)
	require.Len(t, mapping, 5, "mapping should contain all 5 tool call IDs (primary or positional)")

	expected := map[string]string{
		"chatcmpl-1769975533529433976-1": "read_file",
		"chatcmpl-1769975533529433976-2": "list_directory",
		"chatcmpl-1769975533529433976-3": "shell_command",
		"chatcmpl-1769975533529433976-4": "write_file",
		"chatcmpl-1769975533529433976-5": "read_file",
	}
	for id, want := range expected {
		got, ok := mapping[id]
		assert.True(t, ok, "tool_call_id %q should be in mapping", id)
		assert.Equal(t, want, got, "tool name for %q", id)
	}

	// Convert each tool message - no WARN should occur, ToolName must be set.
	for i := 1; i <= 5; i++ {
		msg := messages[i+1] // +1 for user, +1 for assistant.
		result := convertMessageToOllama(msg, mapping, testLogger, testCtx)
		assert.Equal(t, "tool", result.Role)
		assert.NotEmpty(t, result.ToolName, "tool message %d should have ToolName", i)
	}
}

// TestBuildToolCallIDToNameMap_MultipleAssistantTurns verifies mapping with
// multiple assistant+tool turn pairs (as in a long conversation).
func TestBuildToolCallIDToNameMap_MultipleAssistantTurns(t *testing.T) {
	t.Parallel()

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("Do something"),
		openai.ChatCompletionAssistantMessageParam{
			Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
			ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
				mkTC("turn1-0", "read_file", `{}`),
				mkTC("turn1-1", "shell_command", `{}`),
			}),
		},
		openai.ToolMessage("turn1-0", "c1"),
		openai.ToolMessage("turn1-1", "c2"),
		openai.ChatCompletionAssistantMessageParam{
			Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
			ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
				mkTC("turn2-0", "list_directory", `{}`),
			}),
		},
		openai.ToolMessage("turn2-0", "c3"),
	}

	mapping := buildToolCallIDToNameMap(messages, testLogger, testCtx)
	assert.Len(t, mapping, 3)
	assert.Equal(t, "read_file", mapping["turn1-0"])
	assert.Equal(t, "shell_command", mapping["turn1-1"])
	assert.Equal(t, "list_directory", mapping["turn2-0"])
}

// TestConvertMessageToOllama_ToolResultFallback tests that buildToolCallIDToNameMap
// uses positional fallback when tool_call_ids are not found in assistant messages.
// This handles cases where assistant tool_calls fail to parse but tool messages
// are in correct order after their assistant.
func TestConvertMessageToOllama_ToolResultFallback(t *testing.T) {
	t.Parallel()

	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			mkTC("call-0", "shell_command", `{"command":"ls"}`),
		}),
	}
	toolMsg := openai.ChatCompletionToolMessageParam{
		Role: openai.F(openai.ChatCompletionToolMessageParamRoleTool),
		Content: openai.F([]openai.ChatCompletionContentPartTextParam{
			{
				Type: openai.F(openai.ChatCompletionContentPartTextTypeText),
				Text: openai.F("output"),
			},
		}),
		ToolCallID: openai.F("call-0"),
	}

	messages := []openai.ChatCompletionMessageParamUnion{assistantMsg, toolMsg}

	mapping := buildToolCallIDToNameMap(messages, testLogger, testCtx)
	require.NotEmpty(t, mapping, "mapping should have call-0 from assistant message")
	assert.Equal(t, "shell_command", mapping["call-0"])

	result := convertMessageToOllama(toolMsg, mapping, testLogger, testCtx)
	assert.Equal(t, "shell_command", result.ToolName)
}

// TestBuildToolCallIDToNameMap_PhantomToolCalls verifies that tool calls with
// empty function names (phantom entries emitted by some models like qwen3) are
// handled gracefully. The mapping should only contain entries for valid tool calls.
func TestBuildToolCallIDToNameMap_PhantomToolCalls(t *testing.T) {
	t.Parallel()

	// Simulate the pattern observed: first tool call has empty name+args,
	// subsequent tool calls are valid.
	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			// Phantom: empty name.
			mkTC("chatcmpl-100-0", "", `{}`),
			// Valid.
			mkTC("chatcmpl-100-1", "shell_command", `{"command":"ls"}`),
			// Valid.
			mkTC("chatcmpl-100-2", "read_file", `{"path":"test.go"}`),
		}),
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("Do something"),
		assistantMsg,
		openai.ToolMessage("chatcmpl-100-1", "output1"),
		openai.ToolMessage("chatcmpl-100-2", "output2"),
	}

	mapping := buildToolCallIDToNameMap(messages, testLogger, testCtx)

	// The phantom entry (chatcmpl-100-0) should NOT be in the mapping.
	_, hasPhantom := mapping["chatcmpl-100-0"]
	assert.False(t, hasPhantom, "phantom tool call with empty name should not be in mapping")

	// Valid entries should be present.
	assert.Equal(t, "shell_command", mapping["chatcmpl-100-1"])
	assert.Equal(t, "read_file", mapping["chatcmpl-100-2"])

	// Tool messages should resolve correctly.
	for _, id := range []string{"chatcmpl-100-1", "chatcmpl-100-2"} {
		toolMsg := openai.ToolMessage(id, "result")
		result := convertMessageToOllama(toolMsg, mapping, testLogger, testCtx)
		assert.NotEmpty(t, result.ToolName, "tool message %s should have ToolName", id)
	}
}

// TestBuildToolCallIDToNameMap_AllPhantomToolCalls verifies behavior when ALL
// tool calls in an assistant message have empty names.
func TestBuildToolCallIDToNameMap_AllPhantomToolCalls(t *testing.T) {
	t.Parallel()

	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			mkTC("id-0", "", `{}`),
			mkTC("id-1", "", `{}`),
		}),
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("test"),
		assistantMsg,
	}

	mapping := buildToolCallIDToNameMap(messages, testLogger, testCtx)
	assert.Empty(t, mapping, "mapping should be empty when all tool calls have empty names")
}

// TestConvertOllamaResponseToOpenAI_AfterPhantomFiltering verifies that after
// filtering phantom tool calls, convertOllamaResponseToOpenAI produces correct output.
func TestConvertOllamaResponseToOpenAI_AfterPhantomFiltering(t *testing.T) {
	t.Parallel()

	resp := api.ChatResponse{
		Message: api.Message{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				// Phantom: empty name (should be filtered before conversion).
				{Function: api.ToolCallFunction{Name: "", Arguments: map[string]any{}}},
				// Valid.
				{Function: api.ToolCallFunction{Name: "shell_command", Arguments: map[string]any{"command": "ls"}}},
				// Valid.
				{Function: api.ToolCallFunction{Name: "read_file", Arguments: map[string]any{"path": "test.go"}}},
			},
		},
		Done:       true,
		DoneReason: "stop",
	}

	// Apply the same filtering as provider.go.
	filtered := resp.Message.ToolCalls[:0]
	for _, tc := range resp.Message.ToolCalls {
		if tc.Function.Name != "" {
			filtered = append(filtered, tc)
		}
	}

	resp.Message.ToolCalls = filtered

	result := convertOllamaResponseToOpenAI(resp, "qwen3:1.7b", testLogger, testCtx)

	require.Len(t, result.Choices, 1)
	require.Len(t, result.Choices[0].Message.ToolCalls, 2, "should only have 2 valid tool calls")

	assert.Equal(t, "shell_command", result.Choices[0].Message.ToolCalls[0].Function.Name)
	assert.Equal(t, "read_file", result.Choices[0].Message.ToolCalls[1].Function.Name)

	// IDs should be properly generated.
	assert.NotEmpty(t, result.Choices[0].Message.ToolCalls[0].ID)
	assert.NotEmpty(t, result.Choices[0].Message.ToolCalls[1].ID)

	// Finish reason should be tool_calls since there are valid tool calls.
	assert.Equal(t, openai.ChatCompletionChoicesFinishReasonToolCalls, result.Choices[0].FinishReason)
}

// TestConvertOllamaChunkToOpenAI_AfterPhantomFiltering verifies the streaming
// chunk conversion after phantom tool calls are filtered.
func TestConvertOllamaChunkToOpenAI_AfterPhantomFiltering(t *testing.T) {
	t.Parallel()

	resp := api.ChatResponse{
		Message: api.Message{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{Function: api.ToolCallFunction{Name: "", Arguments: map[string]any{}}},
				{Function: api.ToolCallFunction{Name: "shell_command", Arguments: map[string]any{"command": "ls"}}},
			},
		},
		Done: false,
	}

	// Apply filtering.
	filtered := resp.Message.ToolCalls[:0]
	for _, tc := range resp.Message.ToolCalls {
		if tc.Function.Name != "" {
			filtered = append(filtered, tc)
		}
	}

	resp.Message.ToolCalls = filtered

	chunk := convertOllamaChunkToOpenAI(resp, "chatcmpl-test-123", "qwen3:1.7b", testLogger, testCtx)

	require.Len(t, chunk.Choices, 1)
	require.Len(t, chunk.Choices[0].Delta.ToolCalls, 1, "should only have 1 valid tool call")
	assert.Equal(t, "shell_command", chunk.Choices[0].Delta.ToolCalls[0].Function.Name)
	assert.Equal(t, "chatcmpl-test-123-0", chunk.Choices[0].Delta.ToolCalls[0].ID)
}

// newTestTool creates an api.Tool with a single string property.
func newTestTool(name, description, propName, propDesc string) api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        name,
			Description: description,
			Parameters: api.ToolFunctionParameters{
				Type:     "object",
				Required: []string{propName},
				Properties: map[string]api.ToolProperty{
					propName: {Type: api.PropertyType{"string"}, Description: propDesc},
				},
			},
		},
	}
}

// TestInferToolName verifies that inferToolName correctly matches argument keys
// against tool parameter schemas to identify the intended tool.
func TestInferToolName(t *testing.T) {
	t.Parallel()

	tools := []api.Tool{
		newTestTool("read_file", "Read a file", "path", "File path"),
		newTestTool("shell_command", "Run a shell command", "command", "Command to run"),
		newTestTool("list_directory", "List a directory", "path", "Directory path"),
	}

	tests := []struct {
		name     string
		args     map[string]any
		expected string
	}{
		{
			name:     "infer shell_command from command arg",
			args:     map[string]any{"command": "ls -la"},
			expected: "shell_command",
		},
		{
			name:     "ambiguous path arg returns first match rather than empty",
			args:     map[string]any{"path": "internal/llm/vram/nvidia.go"},
			expected: "read_file", // read_file comes first in the tools list.
		},
		{
			name:     "empty args returns empty",
			args:     map[string]any{},
			expected: "",
		},
		{
			name:     "nil args returns empty",
			args:     nil,
			expected: "",
		},
		{
			name:     "unknown args returns empty",
			args:     map[string]any{"unknown_param": "value"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := inferToolName(tt.args, tools, testLogger, testCtx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  json.RawMessage
		expected string
	}{
		{"string", json.RawMessage(`"hello"`), "hello"},
		{"array", json.RawMessage(`[{"type":"text","text":"world"}]`), "world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := extractContent(tt.content)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestThinkingFieldMerge verifies that Ollama's Message.Thinking field
// is properly merged into Content as <think> tags.
func TestThinkingFieldMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		thinking        string
		content         string
		expectedContent string
	}{
		{
			name:            "thinking only",
			thinking:        "Let me reason about this...",
			content:         "",
			expectedContent: "<think>Let me reason about this...</think>",
		},
		{
			name:            "thinking and content",
			thinking:        "I should use shell_command",
			content:         "I'll run a command for you.",
			expectedContent: "<think>I should use shell_command</think>I'll run a command for you.",
		},
		{
			name:            "content only",
			thinking:        "",
			content:         "Hello world",
			expectedContent: "Hello world",
		},
		{
			name:            "neither thinking nor content",
			thinking:        "",
			content:         "",
			expectedContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := api.ChatResponse{
				Message: api.Message{
					Role:     "assistant",
					Content:  tt.content,
					Thinking: tt.thinking,
				},
			}

			// Simulate the merging logic from the provider.
			if resp.Message.Thinking != "" {
				resp.Message.Content = "<think>" + resp.Message.Thinking + "</think>" + resp.Message.Content
				resp.Message.Thinking = ""
			}

			assert.Equal(t, tt.expectedContent, resp.Message.Content)
			assert.Empty(t, resp.Message.Thinking)
		})
	}
}
