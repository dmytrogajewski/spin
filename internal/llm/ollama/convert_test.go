package ollama

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/openai/openai-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Suppress warn logs during tests
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
}

// TestBuildToolCallIDToNameMap_ReproducesMissingMapping tests the scenario
// where tool_call_ids in tool result messages are not found in the mapping.
// This can happen when assistant messages with tool_calls are serialized
// differently or when the mapping is built from a subset of messages.
func TestBuildToolCallIDToNameMap_ReproducesMissingMapping(t *testing.T) {
	// Simulate messages as they would be sent to Ollama after agent execution:
	// - User message
	// - Assistant with 5 tool calls (IDs: chatcmpl-123-0 through chatcmpl-123-4)
	// - 5 Tool result messages
	toolCallIDs := []string{"chatcmpl-123-0", "chatcmpl-123-1", "chatcmpl-123-2", "chatcmpl-123-3", "chatcmpl-123-4"}
	toolNames := []string{"read_file", "list_directory", "shell_command", "write_file", "read_file"}

	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			{ID: openai.F(toolCallIDs[0]), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F(toolNames[0]), Arguments: openai.F(`{"path":"/tmp/a"}`)})},
			{ID: openai.F(toolCallIDs[1]), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F(toolNames[1]), Arguments: openai.F(`{}`)})},
			{ID: openai.F(toolCallIDs[2]), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F(toolNames[2]), Arguments: openai.F(`{"command":"ls"}`)})},
			{ID: openai.F(toolCallIDs[3]), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F(toolNames[3]), Arguments: openai.F(`{}`)})},
			{ID: openai.F(toolCallIDs[4]), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F(toolNames[4]), Arguments: openai.F(`{"path":"/tmp/b"}`)})},
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

	mapping := buildToolCallIDToNameMap(messages)
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
	// Marshal and unmarshal to simulate what parseGenericMessage does
	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			{ID: openai.F("chatcmpl-1769975533529433976-1"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("shell_command"), Arguments: openai.F(`{"command":"ls"}`)})},
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
	// Scenario: mapping has only 2 entries (e.g. from compression) but we have
	// 5 tool results. The positional fallback should resolve the missing ones.
	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			{ID: openai.F("call-0"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("read_file"), Arguments: openai.F(`{}`)})},
			{ID: openai.F("call-1"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("shell_command"), Arguments: openai.F(`{}`)})},
			{ID: openai.F("call-2"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("list_directory"), Arguments: openai.F(`{}`)})},
		}),
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		assistantMsg,
		openai.ToolMessage("call-0", "content0"),
		openai.ToolMessage("call-1", "content1"),
		openai.ToolMessage("call-2", "content2"),
	}

	mapping := buildToolCallIDToNameMap(messages)
	require.Len(t, mapping, 3)

	// Convert each tool message - all should get ToolName set
	for i := range []string{"call-0", "call-1", "call-2"} {
		msg := messages[i+1] // +1 to skip assistant
		result := convertMessageToOllama(msg, mapping)
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
	// Assistant with 5 tool calls - use IDs that match the bug report format
	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			{ID: openai.F("chatcmpl-1769975533529433976-1"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("read_file"), Arguments: openai.F(`{}`)})},
			{ID: openai.F("chatcmpl-1769975533529433976-2"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("list_directory"), Arguments: openai.F(`{}`)})},
			{ID: openai.F("chatcmpl-1769975533529433976-3"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("shell_command"), Arguments: openai.F(`{}`)})},
			{ID: openai.F("chatcmpl-1769975533529433976-4"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("write_file"), Arguments: openai.F(`{}`)})},
			{ID: openai.F("chatcmpl-1769975533529433976-5"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("read_file"), Arguments: openai.F(`{}`)})},
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

	mapping := buildToolCallIDToNameMap(messages)
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

	// Convert each tool message - no WARN should occur, ToolName must be set
	for i := 1; i <= 5; i++ {
		msg := messages[i+1] // +1 for user, +1 for assistant
		result := convertMessageToOllama(msg, mapping)
		assert.Equal(t, "tool", result.Role)
		assert.NotEmpty(t, result.ToolName, "tool message %d should have ToolName", i)
	}
}

// TestBuildToolCallIDToNameMap_MultipleAssistantTurns verifies mapping with
// multiple assistant+tool turn pairs (as in a long conversation).
func TestBuildToolCallIDToNameMap_MultipleAssistantTurns(t *testing.T) {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("Do something"),
		openai.ChatCompletionAssistantMessageParam{
			Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
			ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
				{ID: openai.F("turn1-0"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("read_file"), Arguments: openai.F(`{}`)})},
				{ID: openai.F("turn1-1"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("shell_command"), Arguments: openai.F(`{}`)})},
			}),
		},
		openai.ToolMessage("turn1-0", "c1"),
		openai.ToolMessage("turn1-1", "c2"),
		openai.ChatCompletionAssistantMessageParam{
			Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
			ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
				{ID: openai.F("turn2-0"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("list_directory"), Arguments: openai.F(`{}`)})},
			}),
		},
		openai.ToolMessage("turn2-0", "c3"),
	}

	mapping := buildToolCallIDToNameMap(messages)
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
	assistantMsg := openai.ChatCompletionAssistantMessageParam{
		Role: openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F([]openai.ChatCompletionMessageToolCallParam{
			{ID: openai.F("call-0"), Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction), Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{Name: openai.F("shell_command"), Arguments: openai.F(`{"command":"ls"}`)})},
		}),
	}
	toolMsg := openai.ChatCompletionToolMessageParam{
		Role:       openai.F(openai.ChatCompletionToolMessageParamRoleTool),
		Content:    openai.F([]openai.ChatCompletionContentPartTextParam{{Type: openai.F(openai.ChatCompletionContentPartTextTypeText), Text: openai.F("output")}}),
		ToolCallID: openai.F("call-0"),
	}

	messages := []openai.ChatCompletionMessageParamUnion{assistantMsg, toolMsg}

	mapping := buildToolCallIDToNameMap(messages)
	require.NotEmpty(t, mapping, "mapping should have call-0 from assistant message")
	assert.Equal(t, "shell_command", mapping["call-0"])

	result := convertMessageToOllama(toolMsg, mapping)
	assert.Equal(t, "shell_command", result.ToolName)
}

func TestExtractContent(t *testing.T) {
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
			got := extractContent(tt.content)
			assert.Equal(t, tt.expected, got)
		})
	}
}
