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
	// ID is the unique identifier for this tool call.
	// This links the result back to the original ToolCall.
	ID string `json:"id,omitempty"`

	// Success indicates whether the tool execution succeeded.
	Success bool `json:"success"`

	// Output contains the tool's output message for the LLM.
	Output string `json:"output"`

	// Error contains an error message if the tool failed (for JSON serialization).
	Error string `json:"error,omitempty"`

	// Err contains the actual error if the tool failed.
	// This field is not serialized to JSON.
	Err error `json:"-"`

	// ExitCode contains the exit code for command-based tools.
	// Zero indicates success, non-zero indicates failure.
	ExitCode int `json:"exit_code,omitempty"`

	// Metadata contains additional tool-specific data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewToolResult creates a successful tool result with the given output.
func NewToolResult(output string) ToolResult {
	return ToolResult{
		Success: true,
		Output:  output,
	}
}

// NewToolError creates a failed tool result from an error.
func NewToolError(err error) ToolResult {
	return ToolResult{
		Success: false,
		Err:     err,
		Error:   err.Error(),
	}
}

// NewToolErrorWithID creates a failed tool result with ID from an error.
func NewToolErrorWithID(id string, err error) ToolResult {
	return ToolResult{
		ID:      id,
		Success: false,
		Err:     err,
		Error:   err.Error(),
	}
}

// WithID returns a copy of the result with the given ID.
func (r ToolResult) WithID(id string) ToolResult {
	r.ID = id

	return r
}

// WithExitCode returns a copy of the result with the given exit code.
func (r ToolResult) WithExitCode(code int) ToolResult {
	r.ExitCode = code

	return r
}

// WithMetadata returns a copy of the result with the given metadata.
func (r ToolResult) WithMetadata(metadata map[string]any) ToolResult {
	r.Metadata = metadata

	return r
}

// GetErr returns the error if present, or nil.
func (r ToolResult) GetErr() error {
	return r.Err
}

// String returns a string representation of the result.
func (r ToolResult) String() string {
	if r.Success {
		return r.Output
	}

	if r.Err != nil {
		return r.Err.Error()
	}

	return r.Error
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
