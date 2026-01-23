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
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/openai/openai-go"
)

// convertMessageToOllama converts an OpenAI message to Ollama format.
// This uses a simplified approach by marshaling and unmarshaling JSON.
// The toolCallIDToName map is used to resolve tool_call_id to tool_name for tool result messages.
func convertMessageToOllama(msg openai.ChatCompletionMessageParamUnion, toolCallIDToName map[string]string) api.Message {
	genericMsg, err := parseGenericMessage(msg)
	if err != nil {
		return api.Message{}
	}

	result := api.Message{
		Role:    genericMsg.Role,
		Content: extractContent(genericMsg.Content),
	}

	if len(genericMsg.ToolCalls) > 0 {
		result.ToolCalls = extractToolCalls(genericMsg.ToolCalls)
	}

	// For tool result messages, set ToolName from the mapping
	// This is required for Gemini and other models that need tool_name instead of tool_call_id
	if genericMsg.Role == "tool" && genericMsg.ToolCallID != "" && toolCallIDToName != nil {
		if toolName, ok := toolCallIDToName[genericMsg.ToolCallID]; ok {
			result.ToolName = toolName
			slog.Debug("set tool_name for tool result message",
				"tool_call_id", genericMsg.ToolCallID,
				"tool_name", toolName)
		} else {
			slog.Warn("tool_call_id not found in mapping, tool result may fail",
				"tool_call_id", genericMsg.ToolCallID,
				"mapping_size", len(toolCallIDToName))
		}
	} else if genericMsg.Role == "tool" {
		slog.Warn("tool message without tool_call_id or mapping",
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
func buildToolCallIDToNameMap(messages []openai.ChatCompletionMessageParamUnion) map[string]string {
	result := make(map[string]string)

	for _, msg := range messages {
		genericMsg, err := parseGenericMessage(msg)
		if err != nil {
			continue
		}

		// Only assistant messages can have tool_calls
		if genericMsg.Role != "assistant" || len(genericMsg.ToolCalls) == 0 {
			continue
		}

		// Parse tool calls to extract id -> name mapping
		var toolCalls []struct {
			ID       string `json:"id"`
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		if err := json.Unmarshal(genericMsg.ToolCalls, &toolCalls); err != nil {
			continue
		}

		for _, tc := range toolCalls {
			if tc.ID != "" && tc.Function.Name != "" {
				result[tc.ID] = tc.Function.Name
			}
		}
	}

	return result
}

// parseGenericMessage converts an OpenAI message to a generic structure.
func parseGenericMessage(msg openai.ChatCompletionMessageParamUnion) (*genericMessage, error) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	var genericMsg genericMessage
	if err := json.Unmarshal(jsonData, &genericMsg); err != nil {
		return nil, err
	}

	return &genericMsg, nil
}

// extractContent extracts content from raw JSON, handling both string and array formats.
func extractContent(content json.RawMessage) string {
	var contentStr string
	if err := json.Unmarshal(content, &contentStr); err == nil {
		return contentStr
	}

	// Try as array of content parts
	var contentParts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &contentParts); err == nil {
		var result string
		for _, part := range contentParts {
			if part.Type == "text" {
				result += part.Text
			}
		}
		return result
	}

	return ""
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

	if err := json.Unmarshal(toolCallsJSON, &toolCalls); err != nil {
		return nil
	}

	result := make([]api.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		result[i] = api.ToolCall{
			Function: api.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: args,
			},
		}
	}

	return result
}

// convertToolToOllama converts an OpenAI tool to Ollama format.
func convertToolToOllama(tool openai.ChatCompletionToolParam) api.Tool {
	// Serialize to JSON and parse to extract fields
	jsonData, _ := json.Marshal(tool)

	var genericTool struct {
		Type     string `json:"type"`
		Function struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			Parameters  map[string]interface{} `json:"parameters"`
		} `json:"function"`
	}
	json.Unmarshal(jsonData, &genericTool)

	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        genericTool.Function.Name,
			Description: genericTool.Function.Description,
			Parameters: api.ToolFunctionParameters{
				Type:       "object",
				Properties: extractProperties(genericTool.Function.Parameters),
				Required:   extractRequired(genericTool.Function.Parameters),
			},
		},
	}
}

func extractProperties(params map[string]interface{}) map[string]api.ToolProperty {
	if props, ok := params["properties"].(map[string]interface{}); ok {
		result := make(map[string]api.ToolProperty)
		for k, v := range props {
			if propMap, ok := v.(map[string]interface{}); ok {
				prop := api.ToolProperty{}
				if typ, ok := propMap["type"].(string); ok {
					prop.Type = api.PropertyType{typ}
				}
				if desc, ok := propMap["description"].(string); ok {
					prop.Description = desc
				}
				result[k] = prop
			}
		}
		return result
	}
	return nil
}

func extractRequired(params map[string]interface{}) []string {
	if req, ok := params["required"].([]interface{}); ok {
		result := make([]string, len(req))
		for i, v := range req {
			if s, ok := v.(string); ok {
				result[i] = s
			}
		}
		return result
	}
	return nil
}

// mapOllamaDoneReasonToOpenAICompletion maps Ollama's done_reason to OpenAI's finish_reason for non-streaming.
// This is separate from the chunk version because OpenAI uses different types for streaming vs non-streaming.
func mapOllamaDoneReasonToOpenAICompletion(doneReason string, hasToolCalls bool) openai.ChatCompletionChoicesFinishReason {
	// Tool calls take precedence
	if hasToolCalls {
		return openai.ChatCompletionChoicesFinishReasonToolCalls
	}

	switch doneReason {
	case "length":
		return openai.ChatCompletionChoicesFinishReasonLength
	case "stop", "load", "unload", "":
		return openai.ChatCompletionChoicesFinishReasonStop
	default:
		// Unknown reason, default to stop
		slog.Debug("unknown ollama done_reason, defaulting to stop", "done_reason", doneReason)
		return openai.ChatCompletionChoicesFinishReasonStop
	}
}

// convertOllamaResponseToOpenAI converts an Ollama ChatResponse to OpenAI ChatCompletion.
func convertOllamaResponseToOpenAI(resp api.ChatResponse, model string) *openai.ChatCompletion {
	result := &openai.ChatCompletion{
		ID:      fmt.Sprintf("chatcmpl-ollama-%d", time.Now().UnixNano()),
		Created: time.Now().Unix(),
		Model:   model,
		Object:  "chat.completion",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatCompletionMessageRoleAssistant,
					Content: resp.Message.Content,
				},
				FinishReason: mapOllamaDoneReasonToOpenAICompletion(resp.DoneReason, len(resp.Message.ToolCalls) > 0),
			},
		},
	}

	// Convert tool calls if present
	if len(resp.Message.ToolCalls) > 0 {
		result.Choices[0].Message.ToolCalls = make([]openai.ChatCompletionMessageToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			argsJSON, err := json.Marshal(tc.Function.Arguments)
			if err != nil {
				// Log the error and use empty object as fallback
				slog.Warn("failed to marshal tool call arguments",
					"tool", tc.Function.Name,
					"error", err,
					"arguments", fmt.Sprintf("%v", tc.Function.Arguments))
				argsJSON = []byte("{}")
			}
			result.Choices[0].Message.ToolCalls[i] = openai.ChatCompletionMessageToolCall{
				ID:   fmt.Sprintf("%s-%d", result.ID, i),
				Type: openai.ChatCompletionMessageToolCallTypeFunction,
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: string(argsJSON),
				},
			}
		}
		// FinishReason already set by mapOllamaDoneReasonToOpenAICompletion above
	}

	// Add usage information if available
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
// OpenAI uses: "stop", "length", "tool_calls", "content_filter", "function_call"
func mapOllamaDoneReasonToOpenAI(doneReason string, hasToolCalls bool) openai.ChatCompletionChunkChoicesFinishReason {
	// Tool calls take precedence
	if hasToolCalls {
		return openai.ChatCompletionChunkChoicesFinishReasonToolCalls
	}

	switch doneReason {
	case "length":
		return openai.ChatCompletionChunkChoicesFinishReasonLength
	case "stop", "load", "unload", "":
		return openai.ChatCompletionChunkChoicesFinishReasonStop
	default:
		// Unknown reason, default to stop
		slog.Debug("unknown ollama done_reason, defaulting to stop", "done_reason", doneReason)
		return openai.ChatCompletionChunkChoicesFinishReasonStop
	}
}

// convertOllamaChunkToOpenAI converts an Ollama ChatResponse (streaming) to OpenAI ChatCompletionChunk.
func convertOllamaChunkToOpenAI(resp api.ChatResponse, chunkID, model string) openai.ChatCompletionChunk {
	chunk := openai.ChatCompletionChunk{
		ID:      chunkID,
		Created: time.Now().Unix(),
		Model:   model,
		Object:  "chat.completion.chunk",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Index: 0,
				Delta: openai.ChatCompletionChunkChoicesDelta{
					Role:    openai.ChatCompletionChunkChoicesDeltaRoleAssistant,
					Content: resp.Message.Content,
				},
			},
		},
	}

	// Handle finish reason - map Ollama's done_reason to OpenAI format
	if resp.Done {
		chunk.Choices[0].FinishReason = mapOllamaDoneReasonToOpenAI(resp.DoneReason, len(resp.Message.ToolCalls) > 0)
	}

	// Convert tool calls if present
	if len(resp.Message.ToolCalls) > 0 {
		chunk.Choices[0].Delta.ToolCalls = make([]openai.ChatCompletionChunkChoicesDeltaToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			argsJSON, err := json.Marshal(tc.Function.Arguments)
			if err != nil {
				// Log the error and use empty object as fallback
				slog.Warn("failed to marshal tool call arguments",
					"tool", tc.Function.Name,
					"error", err,
					"arguments", fmt.Sprintf("%v", tc.Function.Arguments))
				argsJSON = []byte("{}")
			}
			chunk.Choices[0].Delta.ToolCalls[i] = openai.ChatCompletionChunkChoicesDeltaToolCall{
				Index: int64(i),
				ID:    fmt.Sprintf("%s-%d", chunkID, i),
				Type:  openai.ChatCompletionChunkChoicesDeltaToolCallsTypeFunction,
				Function: openai.ChatCompletionChunkChoicesDeltaToolCallsFunction{
					Name:      tc.Function.Name,
					Arguments: string(argsJSON),
				},
			}
		}
	}

	return chunk
}
