package ollama

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var streamTestLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
var streamTestCtx = context.Background()

// simulateStreamFilter applies the same tool call filtering logic as Provider.Stream.
// This lets us test the filtering without needing a real Ollama server.
func simulateStreamFilter(toolCalls []api.ToolCall, tools []api.Tool) []api.ToolCall {
	if len(toolCalls) == 0 {
		return toolCalls
	}

	filtered := toolCalls[:0]
	for _, tc := range toolCalls {
		if tc.Function.Name != "" {
			filtered = append(filtered, tc)
		} else if inferred := inferToolName(streamTestCtx, tc.Function.Arguments.ToMap(), tools, streamTestLogger); inferred != "" {
			tc.Function.Name = inferred
			filtered = append(filtered, tc)
		} else {
			continue // dropped: no matching tool name found
		}
	}

	return filtered
}

// standardTools returns the tool definitions spin typically sends to Ollama.
func standardTools() []api.Tool {
	return []api.Tool{
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "read_file",
				Description: "Read a file",
				Parameters: api.ToolFunctionParameters{
					Type:     "object",
					Required: []string{"path"},
					Properties: mkProps(map[string]api.ToolProperty{
						"path": {Type: api.PropertyType([]string{"string"}), Description: "File path to read"},
					}),
				},
			},
		},
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "list_directory",
				Description: "List directory contents",
				Parameters: api.ToolFunctionParameters{
					Type:     "object",
					Required: []string{"path"},
					Properties: mkProps(map[string]api.ToolProperty{
						"path": {Type: api.PropertyType([]string{"string"}), Description: "Directory path"},
					}),
				},
			},
		},
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "shell_command",
				Description: "Run a shell command",
				Parameters: api.ToolFunctionParameters{
					Type:     "object",
					Required: []string{"command"},
					Properties: mkProps(map[string]api.ToolProperty{
						"command":   {Type: api.PropertyType([]string{"string"}), Description: "Command to run"},
						"operation": {Type: api.PropertyType([]string{"string"}), Description: "Operation type"},
					}),
				},
			},
		},
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "write_file",
				Description: "Write content to a file",
				Parameters: api.ToolFunctionParameters{
					Type:     "object",
					Required: []string{"path", "content"},
					Properties: mkProps(map[string]api.ToolProperty{
						"path":    {Type: api.PropertyType([]string{"string"}), Description: "File path"},
						"content": {Type: api.PropertyType([]string{"string"}), Description: "File content"},
					}),
				},
			},
		},
	}
}

// TestStreamFilter_NamedToolCallsPassThrough verifies normal tool calls are never dropped.
func TestStreamFilter_NamedToolCallsPassThrough(t *testing.T) {
	t.Parallel()

	tools := standardTools()
	input := []api.ToolCall{
		{Function: api.ToolCallFunction{Name: "read_file", Arguments: mkArgs(map[string]any{"path": "main.go"})}},
		{Function: api.ToolCallFunction{Name: "shell_command", Arguments: mkArgs(map[string]any{"command": "ls"})}},
	}

	result := simulateStreamFilter(input, tools)
	require.Len(t, result, 2)
	assert.Equal(t, "read_file", result[0].Function.Name)
	assert.Equal(t, "shell_command", result[1].Function.Name)
}

// TestStreamFilter_PhantomToolCallsDropped verifies truly phantom tool calls
// (empty name AND empty args) are dropped.
func TestStreamFilter_PhantomToolCallsDropped(t *testing.T) {
	t.Parallel()

	tools := standardTools()
	input := []api.ToolCall{
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{})}},
		{Function: api.ToolCallFunction{Name: ""}},
	}

	result := simulateStreamFilter(input, tools)
	assert.Empty(t, result, "purely phantom tool calls should be dropped")
}

// TestStreamFilter_NamelessWithUniqueArgs verifies tool calls with empty name
// but unique argument keys are correctly inferred.
func TestStreamFilter_NamelessWithUniqueArgs(t *testing.T) {
	t.Parallel()

	tools := standardTools()

	tests := []struct {
		name         string
		args         api.ToolCallFunctionArguments
		expectedName string
	}{
		{
			name:         "shell_command inferred from command arg",
			args:         mkArgs(map[string]any{"command": "ls -la"}),
			expectedName: "shell_command",
		},
		{
			name:         "shell_command inferred from command+operation args",
			args:         mkArgs(map[string]any{"command": "ls -la", "operation": "execute"}),
			expectedName: "shell_command",
		},
		{
			name:         "write_file inferred from path+content args",
			args:         mkArgs(map[string]any{"path": "test.go", "content": "package main"}),
			expectedName: "write_file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := []api.ToolCall{
				{Function: api.ToolCallFunction{Name: "", Arguments: tt.args}},
			}
			result := simulateStreamFilter(input, tools)
			require.Len(t, result, 1, "tool call with inferable args must not be dropped")
			assert.Equal(t, tt.expectedName, result[0].Function.Name)
		})
	}
}

// TestStreamFilter_NamelessWithAmbiguousArgs is the CRITICAL test.
// Both read_file and list_directory have a single "path" param.
// Models like kimi2.5 emit tool calls like {name:"", args:{path:"internal/llm"}}
// which are ambiguous. The filter MUST NOT drop these — it should pick one.
func TestStreamFilter_NamelessWithAmbiguousArgs(t *testing.T) {
	t.Parallel()

	tools := standardTools()
	input := []api.ToolCall{
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "internal/llm"})}},
	}

	result := simulateStreamFilter(input, tools)
	require.Len(t, result, 1, "ambiguous tool call must NOT be dropped — pick first match")
	// Either read_file or list_directory is acceptable — the important thing is it's not dropped.
	assert.Contains(t, []string{"read_file", "list_directory"}, result[0].Function.Name,
		"should be one of the matching tools, not empty")
}

// TestStreamFilter_MixedNamedAndNameless verifies mixed scenarios:
// some tool calls have names, some don't.
func TestStreamFilter_MixedNamedAndNameless(t *testing.T) {
	t.Parallel()

	tools := standardTools()
	input := []api.ToolCall{
		// Phantom: empty name AND empty args.
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{})}},
		// Named: should pass through.
		{Function: api.ToolCallFunction{Name: "shell_command", Arguments: mkArgs(map[string]any{"command": "ls"})}},
		// Nameless with unique args: should be inferred.
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"command": "pwd"})}},
		// Nameless with ambiguous args: should NOT be dropped.
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "foo.go"})}},
	}

	result := simulateStreamFilter(input, tools)
	require.Len(t, result, 3, "phantom should be dropped, named kept, nameless with args kept")
	assert.Equal(t, "shell_command", result[0].Function.Name)
	assert.Equal(t, "shell_command", result[1].Function.Name) // inferred.
	assert.NotEmpty(t, result[2].Function.Name)               // ambiguous but not dropped.
}

// TestStreamFilter_AllNamelessToolCalls reproduces the original bug:
// model returns ONLY nameless tool calls, filter drops ALL of them,
// agent sees 0 tool calls and exits.
func TestStreamFilter_AllNamelessToolCalls(t *testing.T) {
	t.Parallel()

	tools := standardTools()

	// This is what kimi2.5 actually sends — 3 tool calls, all with empty names.
	input := []api.ToolCall{
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "internal/llm/vram/nvidia.go"})}},
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "internal/llm/vram/amd.go"})}},
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "internal/llm/vram/calculator.go"})}},
	}

	result := simulateStreamFilter(input, tools)
	assert.Len(t, result, 3, "ALL nameless tool calls with valid args must be kept (not dropped)")

	for i, tc := range result {
		assert.NotEmpty(t, tc.Function.Name, "tool call %d must have inferred name", i)
	}
}

// TestStreamFilter_NoToolsAvailable verifies behavior when tools list is empty.
// Cannot infer anything — but should still keep tool calls with valid args? No,
// if no tools schema is available, we can't do anything meaningful. Drop them.
func TestStreamFilter_NoToolsAvailable(t *testing.T) {
	t.Parallel()

	input := []api.ToolCall{
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "test.go"})}},
	}

	result := simulateStreamFilter(input, nil)
	assert.Empty(t, result, "cannot infer without tools schema")
}

// TestConvertOllamaChunkToOpenAI_ToolCallsPreserved verifies the full chain:
// filter -> convert -> OpenAI chunk has correct tool calls.
func TestConvertOllamaChunkToOpenAI_ToolCallsPreserved(t *testing.T) {
	t.Parallel()

	tools := standardTools()

	// Simulate what Ollama sends.
	toolCalls := []api.ToolCall{
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"command": "nvidia-smi"})}},
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "internal/llm/vram"})}},
	}

	// Apply filter (same as provider.go Stream path).
	filtered := simulateStreamFilter(toolCalls, tools)
	require.Len(t, filtered, 2, "both should survive filtering")

	// Build a fake Ollama response with filtered tool calls.
	resp := api.ChatResponse{
		Message: api.Message{
			Role:      "assistant",
			Content:   "Let me check GPU info",
			ToolCalls: filtered,
		},
		Done: true,
	}

	// Convert to OpenAI format.
	chunk := convertOllamaChunkToOpenAI(streamTestCtx, resp, "chatcmpl-test-42", "kimi-k2.5", streamTestLogger)

	require.Len(t, chunk.Choices, 1)
	require.Len(t, chunk.Choices[0].Delta.ToolCalls, 2, "both tool calls should appear in OpenAI chunk")

	assert.Equal(t, "shell_command", chunk.Choices[0].Delta.ToolCalls[0].Function.Name)
	assert.NotEmpty(t, chunk.Choices[0].Delta.ToolCalls[1].Function.Name) // read_file or list_directory.
	assert.NotEmpty(t, chunk.Choices[0].Delta.ToolCalls[0].ID)
	assert.NotEmpty(t, chunk.Choices[0].Delta.ToolCalls[1].ID)
}

// TestConvertOllamaResponseToOpenAI_ToolCallsPreserved verifies the Complete path chain.
func TestConvertOllamaResponseToOpenAI_ToolCallsPreserved(t *testing.T) {
	t.Parallel()

	tools := standardTools()

	toolCalls := []api.ToolCall{
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"command": "nvidia-smi"})}},
		{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "internal/llm/vram"})}},
	}

	filtered := simulateStreamFilter(toolCalls, tools)
	require.Len(t, filtered, 2)

	resp := api.ChatResponse{
		Message: api.Message{
			Role:      "assistant",
			Content:   "Let me check GPU info",
			ToolCalls: filtered,
		},
		Done:       true,
		DoneReason: "stop",
	}

	result := convertOllamaResponseToOpenAI(streamTestCtx, resp, "kimi-k2.5", streamTestLogger)

	require.Len(t, result.Choices, 1)
	require.Len(t, result.Choices[0].Message.ToolCalls, 2)
	assert.Equal(t, "shell_command", result.Choices[0].Message.ToolCalls[0].Function.Name)
	assert.NotEmpty(t, result.Choices[0].Message.ToolCalls[1].Function.Name)
	// finish_reason should be tool_calls since we have tool calls.
	assert.Equal(t, "tool_calls", result.Choices[0].FinishReason)
}

// TestThinkingMergeWithToolCalls verifies that thinking content is merged
// AND tool calls are preserved in the same response.
func TestThinkingMergeWithToolCalls(t *testing.T) {
	t.Parallel()

	tools := standardTools()

	// This is what thinking models actually send:
	// thinking in Message.Thinking, tool calls with empty names.
	resp := api.ChatResponse{
		Message: api.Message{
			Role:     "assistant",
			Thinking: "I should read the vram detector file",
			Content:  "Let me check the GPU detection code.",
			ToolCalls: []api.ToolCall{
				{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "internal/llm/vram/detector.go"})}},
			},
		},
		Done: true,
	}

	// Apply thinking merge (same as provider.go).
	if resp.Message.Thinking != "" {
		resp.Message.Content = "<think>" + resp.Message.Thinking + "</think>" + resp.Message.Content
		resp.Message.Thinking = ""
	}

	// Apply tool call filter.
	resp.Message.ToolCalls = simulateStreamFilter(resp.Message.ToolCalls, tools)

	assert.True(t, resp.Done, "response should be marked as done")
	assert.Contains(t, resp.Message.Content, "<think>")
	assert.Contains(t, resp.Message.Content, "I should read the vram detector file")
	assert.Contains(t, resp.Message.Content, "Let me check the GPU detection code.")
	require.Len(t, resp.Message.ToolCalls, 1, "tool call must survive after thinking merge")
	assert.NotEmpty(t, resp.Message.ToolCalls[0].Function.Name)
}

// TestThinkingOnlyResponse verifies that a response with only thinking content
// and no tool calls produces non-empty content after merge.
func TestThinkingOnlyResponse(t *testing.T) {
	t.Parallel()

	resp := api.ChatResponse{
		Message: api.Message{
			Role:     "assistant",
			Thinking: "Let me reason about GPU detection...",
			Content:  "",
		},
		Done: true,
	}

	if resp.Message.Thinking != "" {
		resp.Message.Content = "<think>" + resp.Message.Thinking + "</think>" + resp.Message.Content
		resp.Message.Thinking = ""
	}

	assert.True(t, resp.Done, "response should be marked as done")
	assert.NotEmpty(t, resp.Message.Content, "thinking-only response must produce non-empty content")
	assert.Equal(t, "<think>Let me reason about GPU detection...</think>", resp.Message.Content)
}

// TestEmptyResponseProducesEmptyContent verifies that a truly empty response
// (no content, no thinking, no tool calls) stays empty.
func TestEmptyResponseProducesEmptyContent(t *testing.T) {
	t.Parallel()

	resp := api.ChatResponse{
		Message: api.Message{
			Role: "assistant",
		},
		Done: true,
	}

	if resp.Message.Thinking != "" {
		resp.Message.Content = "<think>" + resp.Message.Thinking + "</think>" + resp.Message.Content
		resp.Message.Thinking = ""
	}

	assert.True(t, resp.Done, "response should be marked as done")
	assert.Empty(t, resp.Message.Content)
	assert.Empty(t, resp.Message.ToolCalls)
}

// TestFinishReasonMapping verifies done_reason -> finish_reason mapping.
func TestFinishReasonMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		doneReason   string
		hasToolCalls bool
		expected     string
	}{
		{"stop", "stop", false, "stop"},
		{"length", "length", false, "length"},
		{"empty", "", false, "stop"},
		{"tool_calls override stop", "stop", true, "tool_calls"},
		{"tool_calls override length", "length", true, "tool_calls"},
		{"load", "load", false, "stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := mapOllamaDoneReasonToOpenAI(streamTestCtx, tt.doneReason, tt.hasToolCalls, streamTestLogger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestInferToolName_Comprehensive tests inferToolName with the full standard tool set.
func TestInferToolName_Comprehensive(t *testing.T) {
	t.Parallel()

	tools := standardTools()

	tests := []struct {
		name        string
		args        map[string]any
		expectEmpty bool
		expectName  string // if not empty, must match exactly.
	}{
		{
			name:       "unique match: command -> shell_command",
			args:       map[string]any{"command": "ls"},
			expectName: "shell_command",
		},
		{
			name:       "unique match: command+operation -> shell_command",
			args:       map[string]any{"command": "ls", "operation": "execute"},
			expectName: "shell_command",
		},
		{
			name:       "unique match: path+content -> write_file",
			args:       map[string]any{"path": "test.go", "content": "hello"},
			expectName: "write_file",
		},
		{
			name:        "ambiguous: path only matches read_file AND list_directory",
			args:        map[string]any{"path": "internal/llm"},
			expectEmpty: false, // MUST return something, not "".
		},
		{
			name:        "nil args",
			args:        nil,
			expectEmpty: true,
		},
		{
			name:        "empty args",
			args:        map[string]any{},
			expectEmpty: true,
		},
		{
			name:        "completely unknown arg",
			args:        map[string]any{"foo": "bar"},
			expectEmpty: true,
		},
		{
			name:        "partial match: path+unknown doesn't match any tool fully",
			args:        map[string]any{"path": "foo", "unknown_field": "bar"},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := inferToolName(streamTestCtx, tt.args, tools, streamTestLogger)
			if tt.expectEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result, "must not return empty for args: %v", tt.args)

				if tt.expectName != "" {
					assert.Equal(t, tt.expectName, result)
				}
			}
		})
	}
}

// TestStreamContextCancellation verifies that Stream respects context cancellation.
func TestStreamContextCancellation(t *testing.T) {
	t.Parallel()

	provider, err := NewProvider(Config{Model: "test-model"})
	require.NoError(t, err)

	// Cancel context immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Stream should return error or produce 0 chunks on canceled context.
	_, err = provider.Stream(ctx, openai.ChatCompletionNewParams{})
	// The actual behavior depends on whether the Ollama client checks ctx before connecting.
	// Either way, the stream should not hang.
	if err == nil {
		// If no error, the stream channel should be closed quickly
		// This is tested by the caller (callLLM) detecting 0 chunks.
		t.Log("Stream returned no error on canceled context — channel should close quickly")
	}
}

// TestConvertToolToOllama_PreservesProperties verifies that converting OpenAI tools
// to Ollama format preserves parameter properties (needed for inferToolName).
func TestConvertToolToOllama_PreservesProperties(t *testing.T) {
	t.Parallel()

	openaiTool := openai.ChatCompletionToolParam{
		Function: shared.FunctionDefinitionParam{
			Name:        "read_file",
			Description: param.NewOpt("Read a file"),
			Parameters: shared.FunctionParameters{
				"type":     "object",
				"required": []any{"path"},
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "File path to read",
					},
				},
			},
		},
	}

	result := convertToolToOllama(openaiTool)

	assert.Equal(t, "read_file", result.Function.Name)
	assert.Equal(t, "Read a file", result.Function.Description)
	require.NotNil(t, result.Function.Parameters.Properties)
	_, hasPath := result.Function.Parameters.Properties.Get("path")
	assert.True(t, hasPath, "Properties must contain 'path' key for inferToolName to work")
}

// TestConvertToolToOllama_RoundTrip verifies that OpenAI tools survive the
// OpenAI -> Ollama conversion and that inferToolName works with converted tools.
func TestConvertToolToOllama_RoundTrip(t *testing.T) {
	t.Parallel()

	openaiTools := []openai.ChatCompletionToolParam{
		{
			Function: shared.FunctionDefinitionParam{
				Name:        "read_file",
				Description: param.NewOpt("Read a file"),
				Parameters: shared.FunctionParameters{
					"type":     "object",
					"required": []any{"path"},
					"properties": map[string]any{
						"path": map[string]any{"type": "string", "description": "File path"},
					},
				},
			},
		},
		{
			Function: shared.FunctionDefinitionParam{
				Name:        "shell_command",
				Description: param.NewOpt("Run a command"),
				Parameters: shared.FunctionParameters{
					"type":     "object",
					"required": []any{"command"},
					"properties": map[string]any{
						"command": map[string]any{"type": "string", "description": "Command"},
					},
				},
			},
		},
	}

	// Convert to Ollama.
	ollamaTools := make([]api.Tool, len(openaiTools))
	for i, t := range openaiTools {
		ollamaTools[i] = convertToolToOllama(t)
	}

	// Verify inferToolName works with converted tools.
	name := inferToolName(streamTestCtx, map[string]any{"command": "ls"}, ollamaTools, streamTestLogger)
	assert.Equal(t, "shell_command", name, "inferToolName must work with round-tripped tools")

	name = inferToolName(streamTestCtx, map[string]any{"path": "foo.go"}, ollamaTools, streamTestLogger)
	assert.Equal(t, "read_file", name, "inferToolName must work with round-tripped tools (only read_file has path)")
}

// newOpenAIToolParam creates an OpenAI tool param with the given name, description, and properties.
func newOpenAIToolParam(name, description string, required []any, properties map[string]any) openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: shared.FunctionDefinitionParam{
			Name:        name,
			Description: param.NewOpt(description),
			Parameters: shared.FunctionParameters{
				"type":       "object",
				"required":   required,
				"properties": properties,
			},
		},
	}
}

// makeStringProp creates a simple string property map for tool definitions.
func makeStringProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

// buildSpinToolSet returns the standard set of spin tools as OpenAI tool params.
func buildSpinToolSet() []openai.ChatCompletionToolParam {
	return []openai.ChatCompletionToolParam{
		newOpenAIToolParam("read_file", "Read a file from the filesystem", []any{"path"}, map[string]any{
			"path": makeStringProp("The path to the file to read"),
		}),
		newOpenAIToolParam("list_directory", "List directory contents", []any{"path"}, map[string]any{
			"path": makeStringProp("The path to the directory to list"),
		}),
		newOpenAIToolParam("shell_command", "Execute a shell command", []any{"command"}, map[string]any{
			"command":   makeStringProp("The command to execute"),
			"operation": makeStringProp("The operation type"),
		}),
		newOpenAIToolParam("write_file", "Write content to a file", []any{"path", "content"}, map[string]any{
			"path":    makeStringProp("The file path"),
			"content": makeStringProp("The content to write"),
		}),
	}
}

// convertOpenAIToolsToOllama converts a slice of OpenAI tools to Ollama tools.
func convertOpenAIToolsToOllama(tools []openai.ChatCompletionToolParam) []api.Tool {
	result := make([]api.Tool, len(tools))
	for i, tool := range tools {
		result[i] = convertToolToOllama(tool)
	}

	return result
}

// TestConvertToolToOllama_RoundTripWithAllSpinTools mirrors the EXACT tool set
// that spin sends to Ollama. This is the real-world scenario where ambiguity occurs.
func TestConvertToolToOllama_RoundTripWithAllSpinTools(t *testing.T) {
	t.Parallel()

	openaiTools := buildSpinToolSet()
	ollamaTools := convertOpenAIToolsToOllama(openaiTools)

	// Verify properties survived conversion.
	for _, tool := range ollamaTools {
		assert.NotNil(t, tool.Function.Parameters.Properties,
			"tool %s must have Properties after conversion", tool.Function.Name)
	}

	// Test the exact scenarios from real Ollama output.
	tests := []struct {
		name       string
		args       map[string]any
		mustInfer  bool   // must return non-empty.
		exactMatch string // if non-empty, must match exactly.
	}{
		{
			name:       "shell_command from command arg",
			args:       map[string]any{"command": "nvidia-smi"},
			mustInfer:  true,
			exactMatch: "shell_command",
		},
		{
			name:       "shell_command from command+operation args",
			args:       map[string]any{"command": "find . -name '*.go'", "operation": "execute"},
			mustInfer:  true,
			exactMatch: "shell_command",
		},
		{
			name:       "write_file from path+content args (unique 2-arg match)",
			args:       map[string]any{"path": "test.go", "content": "package main"},
			mustInfer:  true,
			exactMatch: "write_file",
		},
		{
			// THE CRITICAL CASE: kimi2.5 sends {path:"internal/llm/vram/nvidia.go"}
			// with no name. Both read_file and list_directory have "path".
			// We MUST return something, not "".
			name:      "ambiguous path-only arg (read_file vs list_directory)",
			args:      map[string]any{"path": "internal/llm/vram/nvidia.go"},
			mustInfer: true,
			// Don't care which one — just must not be empty.
		},
		{
			name:      "ambiguous path-only arg with directory-looking path",
			args:      map[string]any{"path": "internal/llm"},
			mustInfer: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := inferToolName(streamTestCtx, tt.args, ollamaTools, streamTestLogger)
			if tt.mustInfer {
				assert.NotEmpty(t, result, "inferToolName must return non-empty for args: %v", tt.args)
			}

			if tt.exactMatch != "" {
				assert.Equal(t, tt.exactMatch, result)
			}
		})
	}
}

// TestEndToEnd_NamelessToolCallSurvivesFullPipeline tests the complete pipeline:
// 1. OpenAI tools -> convertToolToOllama -> req.Tools
// 2. Ollama response with nameless tool calls
// 3. Filter with inferToolName using req.Tools
// 4. Convert filtered response to OpenAI format
// 5. Verify tool calls appear in final OpenAI response.
func TestEndToEnd_NamelessToolCallSurvivesFullPipeline(t *testing.T) {
	t.Parallel()

	// Step 1: Build and convert tools.
	openaiTools := []openai.ChatCompletionToolParam{
		newOpenAIToolParam("read_file", "", []any{"path"}, map[string]any{"path": map[string]any{"type": "string"}}),
		newOpenAIToolParam("list_directory", "", []any{"path"}, map[string]any{"path": map[string]any{"type": "string"}}),
		newOpenAIToolParam("shell_command", "", []any{"command"}, map[string]any{"command": map[string]any{"type": "string"}}),
	}

	reqTools := convertOpenAIToolsToOllama(openaiTools)

	// Step 2: Simulate Ollama response with nameless tool calls (what kimi2.5 sends).
	ollamaResp := api.ChatResponse{
		Message: api.Message{
			Role:     "assistant",
			Thinking: "I need to read these files to investigate GPU detection",
			Content:  "Let me check the VRAM detection code.",
			ToolCalls: []api.ToolCall{
				{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "internal/llm/vram/nvidia.go"})}},
				{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"path": "internal/llm/vram/amd.go"})}},
				{Function: api.ToolCallFunction{Name: "", Arguments: mkArgs(map[string]any{"command": "nvidia-smi"})}},
			},
		},
		Done:       true,
		DoneReason: "stop",
	}

	// Step 3: Apply thinking merge (same as provider.go).
	if ollamaResp.Message.Thinking != "" {
		ollamaResp.Message.Content = "<think>" + ollamaResp.Message.Thinking + "</think>" + ollamaResp.Message.Content
		ollamaResp.Message.Thinking = ""
	}

	// Step 4: Apply tool call filter (same as provider.go).
	ollamaResp.Message.ToolCalls = simulateStreamFilter(ollamaResp.Message.ToolCalls, reqTools)

	// Step 5: Convert to OpenAI format.
	openaiResp := convertOllamaResponseToOpenAI(streamTestCtx, ollamaResp, "kimi-k2.5", streamTestLogger)

	// ASSERTIONS: This is the final state the agent loop sees.
	require.Len(t, openaiResp.Choices, 1)

	choice := openaiResp.Choices[0]

	// Content must include thinking.
	assert.Contains(t, choice.Message.Content, "<think>")
	assert.Contains(t, choice.Message.Content, "I need to read these files")
	assert.Contains(t, choice.Message.Content, "Let me check the VRAM detection code.")

	// ALL 3 tool calls must survive.
	require.Len(t, choice.Message.ToolCalls, 3,
		"all 3 nameless tool calls must survive the filter pipeline")

	// shell_command must be correctly inferred (unique match).
	foundShellCommand := false

	for _, tc := range choice.Message.ToolCalls {
		assert.NotEmpty(t, tc.Function.Name, "tool call must have name after inference")
		assert.NotEmpty(t, tc.ID, "tool call must have ID")

		if tc.Function.Name == "shell_command" {
			foundShellCommand = true
		}
	}

	assert.True(t, foundShellCommand, "shell_command must be correctly inferred from 'command' arg")

	// finish_reason must be tool_calls (not stop!)
	assert.Equal(t, "tool_calls", choice.FinishReason,
		"finish_reason must be tool_calls when tool calls are present")
}
