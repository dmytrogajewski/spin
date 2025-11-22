// Package tools provides a centralized tool registry for managing
// and executing tools available to the AI agent.
package tools

import (
	"context"
	"errors"
)

// Common errors for the tools package.
var (
	// ErrToolNotFound is returned when a requested tool is not registered.
	ErrToolNotFound = errors.New("tool not found")

	// ErrDuplicateTool is returned when attempting to register a tool with a name that already exists.
	ErrDuplicateTool = errors.New("tool already registered")

	// ErrInvalidParameters is returned when tool parameters don't match the schema.
	ErrInvalidParameters = errors.New("invalid tool parameters")
)

// Tool defines the interface for all tools available to the agent.
// Each tool must provide its name, description, schema, and execution logic.
type Tool interface {
	// Name returns the unique identifier for this tool.
	Name() string

	// Description returns a human-readable description of the tool.
	Description() string

	// Schema returns the OpenAI-compatible function schema for this tool.
	Schema() ToolSchema

	// Execute runs the tool with the given parameters and returns the result.
	Execute(ctx context.Context, params ToolParameters) (ToolResult, error)
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	// Success indicates whether the tool execution succeeded.
	Success bool `json:"success"`

	// Output contains the tool's output message for the LLM.
	Output string `json:"output"`

	// Error contains an error message if the tool failed.
	Error string `json:"error,omitempty"`

	// Metadata contains additional tool-specific data.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolSchema defines the OpenAI-compatible tool schema.
// This matches the format expected by OpenAI's function calling API.
type ToolSchema struct {
	// Type is always "function" for function tools.
	Type string `json:"type"`

	// Function contains the function definition.
	Function FunctionSchema `json:"function"`
}

// FunctionSchema defines the metadata for a function tool.
type FunctionSchema struct {
	// Name is the function name.
	Name string `json:"name"`

	// Description explains what the function does.
	Description string `json:"description"`

	// Parameters defines the function's parameter schema.
	Parameters ParameterSchema `json:"parameters"`
}

// ParameterSchema defines the JSON schema for function parameters.
type ParameterSchema struct {
	// Type is always "object" for function parameters.
	Type string `json:"type"`

	// Properties maps parameter names to their definitions.
	Properties map[string]PropertyDefinition `json:"properties"`

	// Required lists the names of required parameters.
	Required []string `json:"required,omitempty"`
}

// PropertyDefinition defines a single parameter property.
type PropertyDefinition struct {
	// Type is the JSON schema type (string, number, boolean, etc.).
	Type string `json:"type"`

	// Description explains the parameter.
	Description string `json:"description"`

	// Enum lists allowed values for string parameters (optional).
	Enum []string `json:"enum,omitempty"`
}

// BuiltinTools is a compile-time slice of all builtin tools.
// These tools are always available and don't require runtime registration.
var BuiltinTools = []Tool{
	NewReadFileTool(),
	NewWriteFileTool(),
	NewListDirectoryTool(),
	NewShellCommandTool(nil, nil, nil),
	NewGetContextTool(nil),
	NewApplyPatchTool(""),
	NewFileSearchTool(""),
	NewGitContextTool(""),
}
