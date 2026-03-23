package tool

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

// XML tag constants used by the parser and prompt formatter.
const (
	xmlTagFunctionOpen  = "<function="
	xmlTagFunctionClose = "</function>"
	xmlTagParamOpen     = "<parameter="
	xmlTagParamClose    = "</parameter>"
)

// maxXMLToolCalls is the maximum number of tool calls to parse from a single response.
// Prevents excessive memory allocation from pathological input.
const maxXMLToolCalls = 64

// ParseToolCallsFromXML extracts tool calls from content using a forward-scanning parser.
// Returns parsed tool calls and any warnings encountered during parsing.
//
// The parser handles nested content correctly — parameter values containing
// </parameter> or other XML-like content are handled via depth tracking.
func ParseToolCallsFromXML(content string) ([]openai.ChatCompletionMessageToolCall, []string) {
	var toolCalls []openai.ChatCompletionMessageToolCall

	var warnings []string

	pos := 0
	for pos < len(content) && len(toolCalls) < maxXMLToolCalls {
		toolCall, newPos, funcWarnings := parseFunctionBlock(content, pos)
		warnings = append(warnings, funcWarnings...)

		if toolCall == nil {
			if newPos == pos {
				// No progress — no more function blocks to find.
				break
			}

			// Skipped a malformed block; continue scanning.
			pos = newPos

			continue
		}

		toolCalls = append(toolCalls, *toolCall)
		pos = newPos
	}

	if pos < len(content) && len(toolCalls) >= maxXMLToolCalls {
		warnings = append(warnings, fmt.Sprintf("reached maximum tool call limit (%d)", maxXMLToolCalls))
	}

	return toolCalls, warnings
}

// parseFunctionBlock extracts a single <function=NAME>...</function> block starting from pos.
// Returns nil toolCall if no function block is found.
func parseFunctionBlock(content string, pos int) (*openai.ChatCompletionMessageToolCall, int, []string) {
	var warnings []string

	// Scan for next <function= tag.
	funcStart := strings.Index(content[pos:], xmlTagFunctionOpen)
	if funcStart == -1 {
		return nil, len(content), nil
	}

	funcStart += pos

	// Extract function name: <function=NAME>.
	nameStart := funcStart + len(xmlTagFunctionOpen)

	nameEnd := strings.Index(content[nameStart:], ">")
	if nameEnd == -1 {
		warnings = append(warnings, fmt.Sprintf("unclosed <function= tag at position %d", funcStart))

		return nil, nameStart, warnings
	}

	nameEnd += nameStart

	funcName := content[nameStart:nameEnd]
	if funcName == "" {
		warnings = append(warnings, fmt.Sprintf("empty function name at position %d", funcStart))

		return nil, nameEnd + 1, warnings
	}

	// Find matching </function> with depth tracking.
	bodyStart := nameEnd + 1

	funcClose := stringsx.FindMatchingClose(content, bodyStart, xmlTagFunctionOpen, xmlTagFunctionClose)
	if funcClose == -1 {
		warnings = append(warnings, fmt.Sprintf("unclosed <function=%s> block", funcName))

		return nil, bodyStart, warnings
	}

	funcBody := content[bodyStart:funcClose]

	// Parse parameters from the function body.
	args, paramWarnings := parseParameters(funcBody)

	for _, w := range paramWarnings {
		warnings = append(warnings, fmt.Sprintf("in function %q: %s", funcName, w))
	}

	argsBytes, err := json.Marshal(args)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to marshal args for function %q: %v", funcName, err))

		return nil, funcClose + len(xmlTagFunctionClose), warnings
	}

	toolCall := &openai.ChatCompletionMessageToolCall{
		ID:   "call_" + uuid.New().String(),
		Type: "function",
		Function: openai.ChatCompletionMessageToolCallFunction{
			Name:      funcName,
			Arguments: string(argsBytes),
		},
	}

	return toolCall, funcClose + len(xmlTagFunctionClose), warnings
}

// parseParameters extracts parameter key-value pairs from a function body.
func parseParameters(body string) (map[string]any, []string) {
	args := make(map[string]any)

	var warnings []string

	pos := 0
	for pos < len(body) {
		name, value, newPos, paramWarnings := parseOneParameter(body, pos)
		if name == "" && newPos == pos {
			break
		}

		warnings = append(warnings, paramWarnings...)

		if name != "" {
			args[name] = value
		}

		pos = newPos
	}

	return args, warnings
}

// parseOneParameter extracts a single <parameter=NAME>value</parameter> from body at pos.
// Returns empty name if no parameter found or on error.
func parseOneParameter(body string, pos int) (string, any, int, []string) {
	var warnings []string

	paramStart := strings.Index(body[pos:], xmlTagParamOpen)
	if paramStart == -1 {
		return "", nil, pos, nil
	}

	paramStart += pos

	// Extract parameter name: <parameter=NAME>.
	nameStart := paramStart + len(xmlTagParamOpen)

	nameEnd := strings.Index(body[nameStart:], ">")
	if nameEnd == -1 {
		warnings = append(warnings, fmt.Sprintf("unclosed <parameter= tag at position %d", paramStart))

		return "", nil, nameStart, warnings
	}

	nameEnd += nameStart

	paramName := body[nameStart:nameEnd]
	if paramName == "" {
		warnings = append(warnings, fmt.Sprintf("empty parameter name at position %d", paramStart))

		return "", nil, nameEnd + 1, warnings
	}

	// Find matching </parameter> with depth tracking.
	valueStart := nameEnd + 1

	paramClose := stringsx.FindMatchingClose(body, valueStart, xmlTagParamOpen, xmlTagParamClose)
	if paramClose == -1 {
		warnings = append(warnings, fmt.Sprintf("unclosed <parameter=%s> block", paramName))

		return "", nil, valueStart, warnings
	}

	rawValue := strings.TrimSpace(body[valueStart:paramClose])

	return paramName, inferTypedValue(rawValue), paramClose + len(xmlTagParamClose), warnings
}

// inferTypedValue attempts to parse a string value as a JSON-native type.
// Returns the typed value if parsing succeeds (number, bool, null, object, array),
// otherwise returns the original string.
func inferTypedValue(s string) any {
	if s == "" {
		return s
	}

	// Try JSON parsing for non-string types.
	var typed any
	if err := json.Unmarshal([]byte(s), &typed); err == nil {
		// Only use the parsed value if it's NOT a string (strings should stay as-is
		// to preserve values that happen to be valid JSON strings like "true").
		switch typed.(type) {
		case string:
			return s
		default:
			return typed
		}
	}

	return s
}

// FormatToolsAsXMLPrompt generates system prompt instructions for models that
// don't support structured function calling. The prompt teaches the model to
// emit tool calls in XML format that the parser can extract.
func FormatToolsAsXMLPrompt(toolList []tools.Tool) string {
	if len(toolList) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("# Tool Calling\n\n")
	sb.WriteString("You have access to the following tools. To call a tool, output XML in this exact format:\n\n")
	sb.WriteString("```xml\n")
	sb.WriteString("<function=TOOL_NAME>\n")
	sb.WriteString("<parameter=PARAM_NAME>value</parameter>\n")
	sb.WriteString("</function>\n")
	sb.WriteString("```\n\n")
	sb.WriteString("You may call multiple tools in a single response. ")
	sb.WriteString("Each tool call must use this exact XML format.\n\n")
	sb.WriteString("## Available Tools\n\n")

	for _, tool := range toolList {
		writeToolSection(&sb, tool)
	}

	return sb.String()
}

// writeToolSection writes a single tool's documentation to the string builder.
func writeToolSection(sb *strings.Builder, tool tools.Tool) {
	schema := tool.Schema()

	sb.WriteString("### ")
	sb.WriteString(schema.Function.Name)
	sb.WriteString("\n\n")

	if schema.Function.Description != "" {
		sb.WriteString(schema.Function.Description)
		sb.WriteString("\n\n")
	}

	sb.WriteString("**Parameters:**\n")

	required := make(map[string]bool, len(schema.Function.Parameters.Required))
	for _, r := range schema.Function.Parameters.Required {
		required[r] = true
	}

	for name, prop := range schema.Function.Parameters.Properties {
		writePropertyLine(sb, name, prop, required[name])
	}

	sb.WriteString("\n")
}

// writePropertyLine writes a single property definition line to the builder.
func writePropertyLine(sb *strings.Builder, name string, prop tools.PropertyDefinition, isRequired bool) {
	sb.WriteString("- `")
	sb.WriteString(name)
	sb.WriteString("` (")
	sb.WriteString(prop.Type)

	if isRequired {
		sb.WriteString(", required")
	}

	sb.WriteString("): ")
	sb.WriteString(prop.Description)

	if len(prop.Enum) > 0 {
		writeEnumValues(sb, prop.Enum)
	}

	sb.WriteString("\n")
}

// writeEnumValues writes the allowed enum values to the builder.
func writeEnumValues(sb *strings.Builder, values []string) {
	sb.WriteString(" Allowed values: ")

	for i, v := range values {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString("`")
		sb.WriteString(v)
		sb.WriteString("`")
	}
}
