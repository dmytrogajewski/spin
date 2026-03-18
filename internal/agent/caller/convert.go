package caller

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
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
			assistantParam := openai.ChatCompletionAssistantMessageParam{
				ToolCalls: convertToolCallsToOpenAI(msg.ToolCalls),
			}
			if msg.Content != "" {
				assistantParam.Content.OfString = openai.Opt(msg.Content)
			}

			return openai.ChatCompletionMessageParamUnion{OfAssistant: &assistantParam}
		}

		return openai.AssistantMessage(msg.Content)

	case "tool":
		return openai.ToolMessage(msg.Content, msg.ToolCallID)

	default:
		return openai.UserMessage(msg.Content)
	}
}

func convertToolCallsToOpenAI(toolCalls []message.ToolCall) []openai.ChatCompletionMessageToolCallParam {
	result := make([]openai.ChatCompletionMessageToolCallParam, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = openai.ChatCompletionMessageToolCallParam{
			ID: tc.ID,
			Function: openai.ChatCompletionMessageToolCallFunctionParam{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
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
			Function: shared.FunctionDefinitionParam{
				Name:        schema.Function.Name,
				Description: param.NewOpt(schema.Function.Description),
				Parameters:  shared.FunctionParameters(paramsMap),
			},
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

	return completion.Choices[0].FinishReason
}
