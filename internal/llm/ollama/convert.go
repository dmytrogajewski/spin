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
	"time"

	"github.com/ollama/ollama/api"
	"github.com/openai/openai-go"
)

// convertMessageToOllama converts an OpenAI message to Ollama format.
// This uses a simplified approach by marshaling and unmarshaling JSON.
func convertMessageToOllama(msg openai.ChatCompletionMessageParamUnion) api.Message {
	// Serialize the OpenAI message to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return api.Message{}
	}

	// Parse into a generic structure to extract fields
	var genericMsg struct {
		Role       string          `json:"role"`
		Content    json.RawMessage `json:"content"`
		ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
		ToolCallID string          `json:"tool_call_id,omitempty"`
	}
	if err := json.Unmarshal(jsonData, &genericMsg); err != nil {
		return api.Message{}
	}

	result := api.Message{
		Role: genericMsg.Role,
	}

	// Handle content (can be string or array)
	var contentStr string
	if err := json.Unmarshal(genericMsg.Content, &contentStr); err == nil {
		result.Content = contentStr
	} else {
		// Try as array of content parts
		var contentParts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(genericMsg.Content, &contentParts); err == nil {
			for _, part := range contentParts {
				if part.Type == "text" {
					result.Content += part.Text
				}
			}
		}
	}

	// Handle tool calls if present
	if len(genericMsg.ToolCalls) > 0 {
		var toolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		}
		if err := json.Unmarshal(genericMsg.ToolCalls, &toolCalls); err == nil {
			result.ToolCalls = make([]api.ToolCall, len(toolCalls))
			for i, tc := range toolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				result.ToolCalls[i] = api.ToolCall{
					Function: api.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: args,
					},
				}
			}
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
				FinishReason: openai.ChatCompletionChoicesFinishReasonStop,
			},
		},
	}

	// Convert tool calls if present
	if len(resp.Message.ToolCalls) > 0 {
		result.Choices[0].Message.ToolCalls = make([]openai.ChatCompletionMessageToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Function.Arguments)
			result.Choices[0].Message.ToolCalls[i] = openai.ChatCompletionMessageToolCall{
				ID:   fmt.Sprintf("call-%d", i),
				Type: openai.ChatCompletionMessageToolCallTypeFunction,
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: string(argsJSON),
				},
			}
		}
		result.Choices[0].FinishReason = openai.ChatCompletionChoicesFinishReasonToolCalls
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

	// Handle finish reason
	if resp.Done {
		if len(resp.Message.ToolCalls) > 0 {
			chunk.Choices[0].FinishReason = openai.ChatCompletionChunkChoicesFinishReasonToolCalls
		} else {
			chunk.Choices[0].FinishReason = openai.ChatCompletionChunkChoicesFinishReasonStop
		}
	}

	// Convert tool calls if present
	if len(resp.Message.ToolCalls) > 0 {
		chunk.Choices[0].Delta.ToolCalls = make([]openai.ChatCompletionChunkChoicesDeltaToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			argsJSON, _ := json.Marshal(tc.Function.Arguments)
			chunk.Choices[0].Delta.ToolCalls[i] = openai.ChatCompletionChunkChoicesDeltaToolCall{
				Index: int64(i),
				ID:    fmt.Sprintf("call-%d", i),
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
