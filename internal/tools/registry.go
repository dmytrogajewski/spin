package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/dmytrogajewski/spin/pkg/alg/ds/syncmap"
)

var (
	// ErrEnumValueMustBeString is a sentinel error.
	ErrEnumValueMustBeString = errors.New("enum value must be string")
	// ErrValueNotInAllowedValues is a sentinel error.
	ErrValueNotInAllowedValues = errors.New("value  not in allowed values")
)

// Registry manages tool registration, lookup, and execution.
// It provides a centralized registry for all tools available to the agent.
type Registry struct {
	tools *syncmap.Map[string, Tool]
}

// NewRegistry creates a new empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: syncmap.New[string, Tool](),
	}
}

// NewRegistryWithBuiltins creates a new registry pre-populated with all builtin tools.
// This is the recommended constructor for most use cases.
func NewRegistryWithBuiltins() *Registry {
	registry := NewRegistry()
	for _, tool := range BuiltinTools {
		_ = registry.Register(tool)
	}

	return registry
}

// NewDefaultRegistry creates a new registry with all builtin tools properly configured.
// This factory function accepts workDir string and environment interface for tools that need them.
//
// Tools registered:
//   - read_file, write_file, list_directory (no parameters needed)
//   - shell_command (accepts nil parameters, can be configured separately)
//   - get_context (requires env interface{}, can be *agent.Environment or nil)
//   - apply_patch (requires workDir string)
//   - file_search (requires workDir string)
//   - git_context (requires workDir string)
//
// This is the recommended factory for most use cases where tools need proper configuration.
// If workDir is empty, tools that require WorkDir are created with empty string.
// If env is nil, get_context is created with nil.
func NewDefaultRegistry(workDir string, env fmt.Stringer) *Registry {
	// Create registry with builtin tools as base.
	registry := NewRegistryWithBuiltins()

	// Replace tools that need configuration with properly configured versions
	// Note: shell_command is left as-is (nil parameters) since it can be configured
	// separately via RegisterOrReplace if needed.
	_ = registry.RegisterOrReplace(NewGetContextTool(env))
	_ = registry.RegisterOrReplace(NewApplyPatchTool(workDir))
	_ = registry.RegisterOrReplace(NewFileSearchTool(workDir))
	_ = registry.RegisterOrReplace(NewGitContextTool(workDir))

	return registry
}

// Register adds a tool to the registry.
// Returns ErrDuplicateTool if a tool with the same name already exists.
func (r *Registry) Register(tool Tool) error {
	if !r.tools.SetIfAbsent(tool.Name(), tool) {
		return fmt.Errorf("%w: %s", ErrDuplicateTool, tool.Name())
	}

	return nil
}

// RegisterOrReplace adds a tool to the registry, replacing any existing tool with the same name.
// This is useful when you want to override default tools with custom implementations.
func (r *Registry) RegisterOrReplace(tool Tool) error {
	r.tools.Set(tool.Name(), tool)

	return nil
}

// Get retrieves a tool by name.
// Returns ErrToolNotFound if the tool doesn't exist.
func (r *Registry) Get(name string) (Tool, error) {
	tool, exists := r.tools.Get(name)
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	return tool, nil
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	return r.tools.Values()
}

// ListSchemas returns the schemas for all registered tools.
// This is used to provide available tools to the LLM.
func (r *Registry) ListSchemas() []ToolSchema {
	var schemas []ToolSchema

	r.tools.Range(func(_ string, tool Tool) bool {
		schemas = append(schemas, tool.Schema())

		return true
	})

	return schemas
}

// Execute runs a tool by name with the given parameters.
// It validates parameters against the tool's schema before execution.
func (r *Registry) Execute(ctx context.Context, name string, params ToolParameters) (ToolResult, error) {
	// Get the tool.
	tool, err := r.Get(name)
	if err != nil {
		return ToolResult{}, err
	}

	// Validate parameters.
	err = validateParams(tool.Schema(), params)
	if err != nil {
		return ToolResult{}, err
	}

	// Execute the tool.
	result, err := tool.Execute(ctx, params)
	if err != nil {
		return ToolResult{}, fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

// validateParams validates tool parameters against the schema.
func validateParams(schema ToolSchema, params ToolParameters) error {
	paramSchema := schema.Function.Parameters

	err := validateRequiredParams(paramSchema, params)
	if err != nil {
		return err
	}

	return validateParameterTypes(paramSchema, params)
}

// validateRequiredParams checks that all required parameters are present.
func validateRequiredParams(paramSchema ParameterSchema, params ToolParameters) error {
	for _, required := range paramSchema.Required {
		if !params.Has(required) {
			return fmt.Errorf("%w: missing required parameter %s", ErrInvalidParameters, required)
		}
	}

	return nil
}

// validateParameterTypes validates the types and values of all parameters.
func validateParameterTypes(paramSchema ParameterSchema, params ToolParameters) error {
	for _, name := range params.Keys() {
		err := validateParameter(paramSchema, name, params)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateParameter validates a single parameter.
func validateParameter(paramSchema ParameterSchema, name string, params ToolParameters) error {
	propDef, exists := paramSchema.Properties[name]
	if !exists {
		return createUnknownParameterError(name, paramSchema.Properties)
	}

	// Get the raw JSON value for this parameter.
	rawValue, exists := params.raw[name]
	if !exists {
		return fmt.Errorf("%w: parameter %s not found", ErrInvalidParameters, name)
	}

	if !validateTypeFromJSON(rawValue, propDef.Type) {
		return fmt.Errorf("%w: parameter %s has wrong type (expected %s)",
			ErrInvalidParameters, name, propDef.Type)
	}

	if len(propDef.Enum) > 0 {
		err := validateEnumFromJSON(rawValue, propDef.Enum)
		if err != nil {
			return fmt.Errorf("%w: parameter %s %w", ErrInvalidParameters, name, err)
		}
	}

	return nil
}

// createUnknownParameterError creates an error for unknown parameters.
func createUnknownParameterError(name string, properties map[string]PropertyDefinition) error {
	validParams := make([]string, 0, len(properties))
	for pname := range properties {
		validParams = append(validParams, pname)
	}

	return fmt.Errorf("%w: unknown parameter %q (valid parameters: %v)",
		ErrInvalidParameters, name, validParams)
}

// validateTypeFromJSON checks if a JSON value matches the expected JSON schema type.
func validateTypeFromJSON(rawValue json.RawMessage, expectedType string) bool {
	switch expectedType {
	case "string":
		var s string

		return json.Unmarshal(rawValue, &s) == nil && string(rawValue[0]) == `"`
	case "number":
		var f float64

		return json.Unmarshal(rawValue, &f) == nil
	case "integer":
		// Check if it's a valid number.
		var f float64

		err := json.Unmarshal(rawValue, &f)
		if err != nil {
			return false
		}
		// Check if it's an integer (no decimal point in JSON).
		return f == float64(int64(f))
	case "boolean":
		var b bool

		return json.Unmarshal(rawValue, &b) == nil && (string(rawValue) == "true" || string(rawValue) == "false")
	case "array":
		return len(rawValue) > 0 && rawValue[0] == '['
	case "object":
		return len(rawValue) > 0 && rawValue[0] == '{'
	default:
		// Unknown type - accept.
		return true
	}
}

// validateEnumFromJSON checks if a JSON value is in the allowed enum values.
func validateEnumFromJSON(rawValue json.RawMessage, enum []string) error {
	var strValue string

	err := json.Unmarshal(rawValue, &strValue)
	if err != nil {
		return ErrEnumValueMustBeString
	}

	if slices.Contains(enum, strValue) {
		return nil
	}

	return fmt.Errorf("value %q not in allowed values %v: %w", strValue, enum, ErrValueNotInAllowedValues)
}
