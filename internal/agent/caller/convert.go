package caller

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func convertMessageToOpenAI(msg message.Message) openai.ChatCompletionMessageParamUnion {
	role := string(msg.Role)

	switch role {
	case "user":
		return openai.UserMessage(msg.Content)

	case "system":
		return openai.SystemMessage(msg.Content)

	case "assistant":
		if len(msg.ToolCalls) > 0 {
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
		return openai.UserMessage(msg.Content)
	}
}

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

func convertToolsToOpenAI(toolList []tools.Tool) []openai.ChatCompletionToolParam {
	result := make([]openai.ChatCompletionToolParam, len(toolList))
	for i, tool := range toolList {
		schema := tool.Schema()

		paramsMap := map[string]any{
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

func convertOpenAIToolCalls(toolCalls []openai.ChatCompletionMessageToolCall) []agent.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]agent.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = agent.ToolCall{
			ID:   tc.ID,
			Type: string(tc.Type),
			Function: agent.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}

	return result
}

func getContent(completion *openai.ChatCompletion) string {
	if completion == nil || len(completion.Choices) == 0 {
		return ""
	}

	return completion.Choices[0].Message.Content
}

func getToolCalls(completion *openai.ChatCompletion) []agent.ToolCall {
	if completion == nil || len(completion.Choices) == 0 {
		return nil
	}

	return convertOpenAIToolCalls(completion.Choices[0].Message.ToolCalls)
}

func getFinishReason(completion *openai.ChatCompletion) string {
	if completion == nil || len(completion.Choices) == 0 {
		return ""
	}

	return string(completion.Choices[0].FinishReason)
}
