package tools

import (
	"context"
	"errors"

	"github.com/dmytrogajewski/spin/pkg/llmutil/toolresult"
)

// unknownStatus is the fallback label for unrecognized enum values.
const unknownStatus = "unknown"

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

	// Schema returns the JSON schema for the tool's parameters.
	Schema() ToolSchema

	// Execute runs the tool with the given parameters.
	Execute(ctx context.Context, params ToolParameters) (ToolResult, error)
}

// Type aliases re-export result and schema types from the public package.
type (
	// ToolResult represents the result of executing a tool.
	ToolResult = toolresult.Result
	// ToolSchema defines the OpenAI-compatible tool schema.
	ToolSchema = toolresult.Schema
	// FunctionSchema defines the metadata for a function tool.
	FunctionSchema = toolresult.FunctionSchema
	// ParameterSchema defines the JSON schema for function parameters.
	ParameterSchema = toolresult.ParameterSchema
	// PropertyDefinition defines a single parameter property.
	PropertyDefinition = toolresult.PropertyDefinition
)

// Constructor aliases re-export from the public package.
var (
	// NewToolResult creates a successful tool result.
	NewToolResult = toolresult.NewResult
	// NewToolError creates a failed tool result from an error.
	NewToolError = toolresult.NewError
	// ErrToResultf converts an error into a failed ToolResult.
	ErrToResultf = toolresult.ErrToResult
	// NewToolErrorWithID creates a failed tool result with ID.
	NewToolErrorWithID = toolresult.NewErrorWithID
)

// BuiltinTools is a compile-time slice of all builtin tools.
// These tools are always available and don't require runtime registration.
var BuiltinTools = []Tool{
	NewReadFileTool(),
	NewWriteFileTool(),
	NewEditFileTool(),
	NewListDirectoryTool(),
	NewShellCommandTool(nil, nil, nil),
	NewGetContextTool(nil),
	NewApplyPatchTool(""),
	NewFileSearchTool(""),
	NewGitContextTool(""),
}

