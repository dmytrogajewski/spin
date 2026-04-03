package ollama

// Ollama LLM Conversion Boundary
//
// This file contains conversion functions between OpenAI API types and Ollama API types.
// These conversions are NECESSARY because Ollama uses a different API format than OpenAI.
//
// Boundary conversions kept:
// - convertMessageToOllama: OpenAI ChatCompletionMessageParamUnion → Ollama api.Message
// - convertToolToOllama: OpenAI ChatCompletionToolParam → Ollama api.Tool
// - convertOllamaResponseToOpenAI: Ollama api.ChatResponse → OpenAI ChatCompletion
// - convertOllamaChunkToOpenAI: Ollama streaming chunk → OpenAI ChatCompletionChunk
//
// These conversions enable Ollama to work with the OpenAI-compatible interface
// used throughout the rest of the codebase.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/pkg/llmutil"
)

const roleToolResult = "tool"

// convertMessageToOllama converts an OpenAI message to Ollama format.
// This uses a simplified approach by marshaling and unmarshaling JSON.
// The toolCallIDToName map is used to resolve tool_call_id to tool_name for tool result messages.
func convertMessageToOllama(
	ctx context.Context,
	msg openai.ChatCompletionMessageParamUnion, toolCallIDToName map[string]string,
	logger *slog.Logger,
) api.Message {
	genericMsg, err := parseGenericMessage(msg)
	if err != nil {
		return api.Message{}
	}

	result := api.Message{
		Role:    genericMsg.Role,
		Content: llmutil.ExtractContent(genericMsg.Content),
	}

	if len(genericMsg.ToolCalls) > 0 {
		result.ToolCalls = extractToolCalls(genericMsg.ToolCalls)
	}

	// For tool result messages, set ToolName from the mapping
	// This is required for Gemini and other models that need tool_name instead of tool_call_id.
	if genericMsg.Role == roleToolResult && genericMsg.ToolCallID != "" && toolCallIDToName != nil {
		if toolName, ok := toolCallIDToName[genericMsg.ToolCallID]; ok {
			result.ToolName = toolName
			logger.DebugContext(ctx, "set tool_name for tool result message",
				"tool_call_id", genericMsg.ToolCallID,
				"tool_name", toolName)
		} else {
			// Log the mapping keys to help diagnose why this ID isn't found.
			mapKeys := make([]string, 0, len(toolCallIDToName))
			for k := range toolCallIDToName {
				mapKeys = append(mapKeys, k)
			}

			logger.WarnContext(ctx, "tool_call_id not found in mapping, tool result may fail",
				"tool_call_id", genericMsg.ToolCallID,
				"mapping_size", len(toolCallIDToName),
				"mapping_keys", mapKeys)
		}
	} else if genericMsg.Role == roleToolResult {
		logger.WarnContext(ctx, "tool message without tool_call_id or mapping",
			"has_tool_call_id", genericMsg.ToolCallID != "",
			"has_mapping", toolCallIDToName != nil)
	}

	return result
}

type genericMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// buildToolCallIDToNameMap scans messages to build a mapping from tool_call_id to tool_name.
// This is needed because Ollama/Gemini requires tool_name in tool result messages,
// but OpenAI format uses tool_call_id to reference the original call.
//
// It uses two strategies:
//  1. Primary: Extract id->name from assistant messages' tool_calls.
//  2. Positional fallback: For tool messages whose tool_call_id is not in the mapping
//     (e.g. due to assistant message parsing issues or serialization differences),
//     infer tool_name from the preceding assistant message by tool message order.
func buildToolCallIDToNameMap(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion, logger *slog.Logger,
) map[string]string {
	result := make(map[string]string)

	// Pass 1: Extract from assistant messages.
	extractToolCallMappings(ctx, messages, result, logger)

	// Pass 2: Positional fallback for tool messages with missing mapping.
	resolveUnmappedToolMessages(ctx, messages, result, logger)

	logger.DebugContext(ctx, "buildToolCallIDToNameMap: complete",
		"mapping_size", len(result),
		"total_messages", len(messages))

	return result
}

// extractToolCallMappings extracts tool_call_id -> tool_name from assistant messages.
func extractToolCallMappings(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion, result map[string]string,
	logger *slog.Logger,
) {
	for msgIdx, msg := range messages {
		genericMsg, err := parseGenericMessage(msg)
		if err != nil {
			logger.DebugContext(ctx, "buildToolCallIDToNameMap: failed to parse message",
				"index", msgIdx, "error", err)

			continue
		}

		if genericMsg.Role != "assistant" || len(genericMsg.ToolCalls) == 0 {
			continue
		}

		logger.DebugContext(ctx, "buildToolCallIDToNameMap: found assistant message with tool_calls",
			"index", msgIdx, "tool_calls_raw_len", len(genericMsg.ToolCalls))

		extractMappingsFromToolCalls(ctx, genericMsg.ToolCalls, msgIdx, result, logger)
	}
}

// extractMappingsFromToolCalls unmarshals and extracts id->name mappings from raw tool calls JSON.
func extractMappingsFromToolCalls(
	ctx context.Context,
	toolCallsJSON json.RawMessage, msgIdx int, result map[string]string,
	logger *slog.Logger,
) {
	var toolCalls []struct {
		ID       string `json:"id"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}

	err := json.Unmarshal(toolCallsJSON, &toolCalls)
	if err != nil {
		logger.DebugContext(ctx, "buildToolCallIDToNameMap: failed to unmarshal tool_calls",
			"index", msgIdx, "error", err, "raw", string(toolCallsJSON))

		return
	}

	logger.DebugContext(ctx, "buildToolCallIDToNameMap: parsed tool_calls",
		"index", msgIdx, "count", len(toolCalls))

	for _, tc := range toolCalls {
		if tc.ID != "" && tc.Function.Name != "" {
			result[tc.ID] = tc.Function.Name
			logger.DebugContext(ctx, "buildToolCallIDToNameMap: added mapping",
				"id", tc.ID, "name", tc.Function.Name)
		}
	}
}

// resolveUnmappedToolMessages applies positional fallback for tool messages without a mapping.
func resolveUnmappedToolMessages(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion, result map[string]string,
	logger *slog.Logger,
) {
	for i, msg := range messages {
		genericMsg, err := parseGenericMessage(msg)
		if err != nil || genericMsg.Role != "tool" || genericMsg.ToolCallID == "" {
			continue
		}

		if _, ok := result[genericMsg.ToolCallID]; ok {
			continue
		}

		toolName := resolveToolNameByPosition(messages, i)
		if toolName != "" {
			result[genericMsg.ToolCallID] = toolName
			logger.DebugContext(ctx, "buildToolCallIDToNameMap: positional fallback added mapping",
				"index", i, "tool_call_id", genericMsg.ToolCallID, "tool_name", toolName)
		} else {
			logger.DebugContext(ctx, "buildToolCallIDToNameMap: positional fallback failed",
				"index", i, "tool_call_id", genericMsg.ToolCallID)
		}
	}
}

// resolveToolNameByPosition finds the tool name for a tool message at index i
// by locating the preceding assistant message and using tool message order.
func resolveToolNameByPosition(messages []openai.ChatCompletionMessageParamUnion, toolMsgIndex int) string {
	toolCountBeforeMe, assistantToolNames := scanPrecedingMessages(messages, toolMsgIndex)

	if toolCountBeforeMe >= len(assistantToolNames) {
		return ""
	}

	return assistantToolNames[toolCountBeforeMe]
}

// scanPrecedingMessages walks backwards from toolMsgIndex to find the preceding
// assistant message's tool names and count how many tool messages precede us.
func scanPrecedingMessages(messages []openai.ChatCompletionMessageParamUnion, toolMsgIndex int) (toolCount int, toolNames []string) {
	for j := toolMsgIndex - 1; j >= 0; j-- {
		genericMsg, err := parseGenericMessage(messages[j])
		if err != nil {
			continue
		}

		if genericMsg.Role == roleToolResult {
			toolCount++

			continue
		}

		// Other role without tool_calls means no match.
		if genericMsg.Role != "assistant" || len(genericMsg.ToolCalls) == 0 {
			return toolCount, nil
		}

		// Found assistant with tool_calls.
		toolNames = extractAssistantToolNames(genericMsg.ToolCalls)

		return toolCount, toolNames
	}

	return toolCount, nil
}

// extractAssistantToolNames extracts tool function names from raw tool calls JSON.
func extractAssistantToolNames(toolCallsJSON json.RawMessage) []string {
	var toolCalls []struct {
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}

	if json.Unmarshal(toolCallsJSON, &toolCalls) != nil {
		return nil
	}

	names := make([]string, len(toolCalls))
	for i, tc := range toolCalls {
		names[i] = tc.Function.Name
	}

	return names
}

// parseGenericMessage converts an OpenAI message to a generic structure.
func parseGenericMessage(msg openai.ChatCompletionMessageParamUnion) (*genericMessage, error) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal message: %w", err)
	}

	var genericMsg genericMessage

	err = json.Unmarshal(jsonData, &genericMsg)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	return &genericMsg, nil
}

// extractToolCalls converts tool calls from JSON to Ollama format.
func extractToolCalls(toolCallsJSON json.RawMessage) []api.ToolCall {
	var toolCalls []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}

	err := json.Unmarshal(toolCallsJSON, &toolCalls)
	if err != nil {
		return nil
	}

	result := make([]api.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		tcArgs := api.NewToolCallFunctionArguments()

		var args map[string]any
		if unmarshalErr := json.Unmarshal([]byte(tc.Function.Arguments), &args); unmarshalErr == nil {
			for k, v := range args {
				tcArgs.Set(k, v)
			}
		}

		result[i] = api.ToolCall{
			Function: api.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tcArgs,
			},
		}
	}

	return result
}

// inferToolName attempts to match a nameless tool call to a tool from the schema
// by comparing argument keys against each tool's parameter properties.
// Returns the inferred name, or "" if no unique match is found.
func inferToolName(ctx context.Context, args map[string]any, tools []api.Tool, logger *slog.Logger) string {
	if len(args) == 0 || len(tools) == 0 {
		return ""
	}

	argKeys := make(map[string]bool, len(args))
	for k := range args {
		argKeys[k] = true
	}

	bestMatch, ambiguous := findBestToolMatch(argKeys, tools)

	if ambiguous {
		logger.DebugContext(ctx, "ollama: ambiguous tool name inference, using first match",
			"name", bestMatch, "args_keys", argKeys)
	}

	return bestMatch
}

// findBestToolMatch finds the tool whose parameters best match the given argument keys.
// Returns the best match name and whether the match was ambiguous.
func findBestToolMatch(argKeys map[string]bool, tools []api.Tool) (string, bool) {
	var (
		bestMatch string
		bestScore int
		ambiguous bool
	)

	for _, tool := range tools {
		score := scoreToolMatch(argKeys, tool.Function.Parameters.Properties.ToMap())
		if score <= 0 {
			continue
		}

		if score > bestScore {
			bestMatch = tool.Function.Name
			bestScore = score
			ambiguous = false
		} else if score == bestScore {
			ambiguous = true
		}
	}

	return bestMatch, ambiguous
}

// scoreToolMatch returns the match score (number of matching keys) if all arg keys
// match the tool's properties. Returns 0 if not a full match or properties are empty.
func scoreToolMatch(argKeys map[string]bool, props map[string]api.ToolProperty) int {
	if len(props) == 0 {
		return 0
	}

	score := 0

	for k := range argKeys {
		if _, ok := props[k]; ok {
			score++
		}
	}

	// All arg keys must match (no extra unknown keys).
	if score == len(argKeys) && score > 0 {
		return score
	}

	return 0
}

// convertToolToOllama converts an OpenAI tool to Ollama format.
func convertToolToOllama(tool openai.ChatCompletionToolParam) api.Tool {
	// Serialize to JSON and parse to extract fields.
	jsonData, err := json.Marshal(tool)
	if err != nil {
		return api.Tool{}
	}

	var genericTool struct {
		Type     string `json:"type"`
		Function struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			Parameters  map[string]any `json:"parameters"`
		} `json:"function"`
	}

	if unmarshalErr := json.Unmarshal(jsonData, &genericTool); unmarshalErr != nil {
		return api.Tool{}
	}

	propsMap := extractProperties(genericTool.Function.Parameters)

	toolProps := api.NewToolPropertiesMap()
	for k, v := range propsMap {
		toolProps.Set(k, v)
	}

	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        genericTool.Function.Name,
			Description: genericTool.Function.Description,
			Parameters: api.ToolFunctionParameters{
				Type:       "object",
				Properties: toolProps,
				Required:   extractRequired(genericTool.Function.Parameters),
			},
		},
	}
}

func extractProperties(params map[string]any) map[string]api.ToolProperty {
	props, propsOk := params["properties"].(map[string]any)
	if !propsOk {
		return nil
	}

	result := make(map[string]api.ToolProperty)

	for k, v := range props {
		propMap, mapOk := v.(map[string]any)
		if !mapOk {
			continue
		}

		result[k] = buildToolProperty(propMap)
	}

	return result
}

// buildToolProperty constructs an api.ToolProperty from a property map.
func buildToolProperty(propMap map[string]any) api.ToolProperty {
	prop := api.ToolProperty{}
	if typ, typOk := propMap["type"].(string); typOk {
		prop.Type = api.PropertyType([]string{typ})
	}

	if desc, descOk := propMap["description"].(string); descOk {
		prop.Description = desc
	}

	return prop
}

func extractRequired(params map[string]any) []string {
	if req, reqOk := params["required"].([]any); reqOk {
		result := make([]string, len(req))
		for i, v := range req {
			if s, strOk := v.(string); strOk {
				result[i] = s
			}
		}

		return result
	}

	return nil
}

// marshalToolCallArgs marshals an Ollama tool call's arguments to a JSON string.
// On failure it logs a warning and returns "{}".
func marshalToolCallArgs(ctx context.Context, tc api.ToolCall, logger *slog.Logger) string {
	argsJSON, err := json.Marshal(tc.Function.Arguments)
	if err != nil {
		logger.WarnContext(ctx, "failed to marshal tool call arguments",
			"tool", tc.Function.Name,
			"error", err,
			"arguments", fmt.Sprintf("%v", tc.Function.Arguments))

		return "{}"
	}

	logger.DebugContext(ctx, "ollama tool call received",
		"tool", tc.Function.Name,
		"arguments", string(argsJSON),
		"raw_arguments", fmt.Sprintf("%+v", tc.Function.Arguments))

	return string(argsJSON)
}

// convertOllamaResponseToOpenAI converts an Ollama ChatResponse to OpenAI ChatCompletion.
func convertOllamaResponseToOpenAI(ctx context.Context, resp api.ChatResponse, model string, logger *slog.Logger) *openai.ChatCompletion {
	result := &openai.ChatCompletion{
		ID:      fmt.Sprintf("chatcmpl-ollama-%d", time.Now().UnixNano()),
		Created: time.Now().Unix(),
		Model:   model,
		Object:  "chat.completion",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: resp.Message.Content,
				},
				FinishReason: mapOllamaDoneReasonToOpenAI(ctx, resp.DoneReason, len(resp.Message.ToolCalls) > 0, logger),
			},
		},
	}

	// Convert tool calls if present.
	if len(resp.Message.ToolCalls) > 0 {
		result.Choices[0].Message.ToolCalls = make([]openai.ChatCompletionMessageToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			args := marshalToolCallArgs(ctx, tc, logger)
			result.Choices[0].Message.ToolCalls[i] = openai.ChatCompletionMessageToolCall{
				ID:   fmt.Sprintf("%s-%d", result.ID, i),
				Type: "function",
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: args,
				},
			}
		}
		// FinishReason already set by mapOllamaDoneReasonToOpenAI above.
	}

	// Add usage information if available.
	if resp.PromptEvalCount > 0 || resp.EvalCount > 0 {
		result.Usage = openai.CompletionUsage{
			PromptTokens:     int64(resp.PromptEvalCount),
			CompletionTokens: int64(resp.EvalCount),
			TotalTokens:      int64(resp.PromptEvalCount + resp.EvalCount),
		}
	}

	return result
}

// mapOllamaDoneReasonToOpenAI maps Ollama's done_reason to OpenAI's finish_reason.
// Ollama uses: "stop", "load", "unload", "length" (when hitting token limit)
// OpenAI uses: "stop", "length", "tool_calls", "content_filter", "function_call".
func mapOllamaDoneReasonToOpenAI(
	ctx context.Context, doneReason string, hasToolCalls bool, logger *slog.Logger,
) string {
	// Tool calls take precedence.
	if hasToolCalls {
		return llm.FinishReasonToolCalls
	}

	switch doneReason {
	case "length":
		return llm.FinishReasonLength
	case "stop", "load", "unload", "":
		return llm.FinishReasonStop
	default:
		// Unknown reason, default to stop.
		logger.DebugContext(ctx, "unknown ollama done_reason, defaulting to stop", "done_reason", doneReason)

		return llm.FinishReasonStop
	}
}

// convertOllamaChunkToOpenAI converts an Ollama ChatResponse (streaming) to OpenAI ChatCompletionChunk.
func convertOllamaChunkToOpenAI(
	ctx context.Context, resp api.ChatResponse, chunkID, model string, logger *slog.Logger,
) openai.ChatCompletionChunk {
	chunk := openai.ChatCompletionChunk{
		ID:      chunkID,
		Created: time.Now().Unix(),
		Model:   model,
		Object:  "chat.completion.chunk",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Index: 0,
				Delta: openai.ChatCompletionChunkChoiceDelta{
					Role:    "assistant",
					Content: resp.Message.Content,
				},
			},
		},
	}

	// Handle finish reason - map Ollama's done_reason to OpenAI format.
	if resp.Done {
		chunk.Choices[0].FinishReason = mapOllamaDoneReasonToOpenAI(ctx, resp.DoneReason, len(resp.Message.ToolCalls) > 0, logger)
	}

	// Convert tool calls if present.
	if len(resp.Message.ToolCalls) > 0 {
		chunk.Choices[0].Delta.ToolCalls = make([]openai.ChatCompletionChunkChoiceDeltaToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			args := marshalToolCallArgs(ctx, tc, logger)
			chunk.Choices[0].Delta.ToolCalls[i] = openai.ChatCompletionChunkChoiceDeltaToolCall{
				Index: int64(i),
				ID:    fmt.Sprintf("%s-%d", chunkID, i),
				Type:  "function",
				Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: args,
				},
			}
		}
	}

	return chunk
}
