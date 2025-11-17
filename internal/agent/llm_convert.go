package agent

// LLM Conversion Boundary
//
// This file contains conversion functions between internal types and OpenAI API types.
// These conversions are NECESSARY because they interface with an external LLM API.
//
// Boundary conversions kept:
// - convertMessageToOpenAI: internal message.Message → OpenAI ChatCompletionMessageParamUnion
// - convertToolCallsToOpenAI: internal ToolCall → OpenAI tool call params
// - convertToolsToOpenAI: internal tools.Tool → OpenAI tool params
// - convertOpenAIToolCalls: OpenAI tool calls → internal agent.ToolCall
//
// Do NOT add internal-to-internal conversion functions here.
// Use unified types (e.g., agent.ToolCall) throughout the codebase instead.

import (
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

// convertMessageToOpenAI converts a message.Message to openai.ChatCompletionMessageParamUnion.
func convertMessageToOpenAI(msg message.Message) openai.ChatCompletionMessageParamUnion {
	role := string(msg.Role)

	switch role {
	case "user":
		return openai.UserMessage(msg.Content)

	case "system":
		return openai.SystemMessage(msg.Content)

	case "assistant":
		if len(msg.ToolCalls) > 0 {
			// Assistant message with tool calls
			param := openai.ChatCompletionAssistantMessageParam{
				Role:      openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
				ToolCalls: openai.F(convertToolCallsToOpenAI(msg.ToolCalls)),
			}
			if msg.Content != "" {
				param.Content = openai.F([]openai.ChatCompletionAssistantMessageParamContentUnion{
					openai.ChatCompletionAssistantMessageParamContent{
						Type: openai.F(openai.ChatCompletionAssistantMessageParamContentTypeText),
						Text: openai.F(msg.Content),
					},
				})
			}
			return param
		}
		return openai.AssistantMessage(msg.Content)

	case "tool":
		return openai.ChatCompletionToolMessageParam{
			Role: openai.F(openai.ChatCompletionToolMessageParamRoleTool),
			Content: openai.F([]openai.ChatCompletionContentPartTextParam{
				{
					Type: openai.F(openai.ChatCompletionContentPartTextTypeText),
					Text: openai.F(msg.Content),
				},
			}),
			ToolCallID: openai.F(msg.ToolCallID),
		}

	default:
		// Default to user message
		return openai.UserMessage(msg.Content)
	}
}

// convertToolCallsToOpenAI converts message.ToolCall slice to OpenAI tool calls.
func convertToolCallsToOpenAI(toolCalls []message.ToolCall) []openai.ChatCompletionMessageToolCallParam {
	result := make([]openai.ChatCompletionMessageToolCallParam, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = openai.ChatCompletionMessageToolCallParam{
			ID:   openai.F(tc.ID),
			Type: openai.F(openai.ChatCompletionMessageToolCallType(tc.Type)),
			Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{
				Name:      openai.F(tc.Function.Name),
				Arguments: openai.F(tc.Function.Arguments),
			}),
		}
	}
	return result
}

// convertToolsToOpenAI converts tools.Tool slice to OpenAI tool params.
func convertToolsToOpenAI(toolList []tools.Tool) []openai.ChatCompletionToolParam {
	result := make([]openai.ChatCompletionToolParam, len(toolList))
	for i, tool := range toolList {
		schema := tool.Schema()

		// Convert ParameterSchema to map[string]interface{} for OpenAI SDK
		paramsMap := map[string]interface{}{
			"type":       schema.Function.Parameters.Type,
			"properties": schema.Function.Parameters.Properties,
			"required":   schema.Function.Parameters.Required,
		}

		result[i] = openai.ChatCompletionToolParam{
			Type: openai.F(openai.ChatCompletionToolTypeFunction),
			Function: openai.F(openai.FunctionDefinitionParam{
				Name:        openai.F(schema.Function.Name),
				Description: openai.F(schema.Function.Description),
				Parameters:  openai.F(shared.FunctionParameters(paramsMap)),
			}),
		}
	}
	return result
}

// convertOpenAIToolCalls converts OpenAI tool calls to internal ToolCall.
// Note: message.ToolCall is now an alias for agent.ToolCall, eliminating duplication.
func convertOpenAIToolCalls(toolCalls []openai.ChatCompletionMessageToolCall) []ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = ToolCall{
			ID:   tc.ID,
			Type: string(tc.Type),
			Function: ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}

// Helper functions to extract data from openai.ChatCompletion

// getContent extracts the content from the first choice in a ChatCompletion.
func getContent(completion *openai.ChatCompletion) string {
	if completion == nil || len(completion.Choices) == 0 {
		return ""
	}
	return completion.Choices[0].Message.Content
}

// getToolCalls extracts tool calls from the first choice in a ChatCompletion.
func getToolCalls(completion *openai.ChatCompletion) []ToolCall {
	if completion == nil || len(completion.Choices) == 0 {
		return nil
	}
	return convertOpenAIToolCalls(completion.Choices[0].Message.ToolCalls)
}

// getFinishReason extracts the finish reason from the first choice in a ChatCompletion.
func getFinishReason(completion *openai.ChatCompletion) string {
	if completion == nil || len(completion.Choices) == 0 {
		return ""
	}
	return string(completion.Choices[0].FinishReason)
}
