package agent

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
)

// ToolCallXML represents the regex patterns for extracting tool calls from XML-like output.
// Supports formats like:
// <tool_call>
// <function=name>
// <parameter=arg1>value1</parameter>
// </function>
// </tool_call>
//
// Also supports simplified format without tool_call wrapper.
var (
	// Matches <function=name>...</function> blocks
	functionBlockRegex = regexp.MustCompile(`(?s)<function=(\w+)>(.*?)</function>`)
	// Matches <parameter=name>value</parameter> inside function block
	parameterBlockRegex = regexp.MustCompile(`(?s)<parameter=(\w+)>(.*?)</parameter>`)
)

// parseToolCallsFromXML extracts tool calls from content string if they exist in XML format.
// Returns a slice of OpenAI tool calls.
func parseToolCallsFromXML(content string) []openai.ChatCompletionMessageToolCall {
	var toolCalls []openai.ChatCompletionMessageToolCall
	
	// Find all function blocks
	matches := functionBlockRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		
		functionName := match[1]
		functionBody := match[2]
		
		// Parse parameters
		args := make(map[string]interface{})
		paramMatches := parameterBlockRegex.FindAllStringSubmatch(functionBody, -1)
		
		for _, paramMatch := range paramMatches {
			if len(paramMatch) < 3 {
				continue
			}
			paramName := paramMatch[1]
			paramValue := strings.TrimSpace(paramMatch[2])
			args[paramName] = paramValue
		}
		
		// Convert args to JSON string
		argsBytes, err := json.Marshal(args)
		if err != nil {
			continue // Skip malformed args
		}
		
		// Create tool call
		toolCall := openai.ChatCompletionMessageToolCall{
			ID:   "call_" + uuid.New().String()[:8], // Generate random ID
			Type: openai.ChatCompletionMessageToolCallTypeFunction,
			Function: openai.ChatCompletionMessageToolCallFunction{
				Name:      functionName,
				Arguments: string(argsBytes),
			},
		}
		
		toolCalls = append(toolCalls, toolCall)
	}
	
	return toolCalls
}
