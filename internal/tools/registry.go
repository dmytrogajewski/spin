package tools

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages tool registration, lookup, and execution.
// It provides a centralized registry for all tools available to the agent.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
// Returns ErrDuplicateTool if a tool with the same name already exists.
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateTool, name)
	}

	r.tools[name] = tool
	return nil
}

// RegisterOrReplace adds a tool to the registry, replacing any existing tool with the same name.
// This is useful when you want to override default tools with custom implementations.
func (r *Registry) RegisterOrReplace(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools[tool.Name()] = tool
	return nil
}

// Get retrieves a tool by name.
// Returns ErrToolNotFound if the tool doesn't exist.
func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	return tool, nil
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// ListSchemas returns the schemas for all registered tools.
// This is used to provide available tools to the LLM.
func (r *Registry) ListSchemas() []ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]ToolSchema, 0, len(r.tools))
	for _, tool := range r.tools {
		schemas = append(schemas, tool.Schema())
	}

	return schemas
}

// Execute runs a tool by name with the given parameters.
// It validates parameters against the tool's schema before execution.
func (r *Registry) Execute(ctx context.Context, name string, params map[string]interface{}) (ToolResult, error) {
	// Get the tool
	tool, err := r.Get(name)
	if err != nil {
		return ToolResult{}, err
	}

	// Validate parameters
	if err := r.validateParams(tool.Schema(), params); err != nil {
		return ToolResult{}, err
	}

	// Execute the tool
	result, err := tool.Execute(ctx, params)
	if err != nil {
		return ToolResult{}, fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

// validateParams validates tool parameters against the schema.
func (r *Registry) validateParams(schema ToolSchema, params map[string]interface{}) error {
	paramSchema := schema.Function.Parameters

	if err := r.validateRequiredParams(paramSchema, params); err != nil {
		return err
	}

	return r.validateParameterTypes(paramSchema, params)
}

// validateRequiredParams checks that all required parameters are present.
func (r *Registry) validateRequiredParams(paramSchema ParameterSchema, params map[string]interface{}) error {
	for _, required := range paramSchema.Required {
		if _, exists := params[required]; !exists {
			return fmt.Errorf("%w: missing required parameter %s", ErrInvalidParameters, required)
		}
	}
	return nil
}

// validateParameterTypes validates the types and values of all parameters.
func (r *Registry) validateParameterTypes(paramSchema ParameterSchema, params map[string]interface{}) error {
	for name, value := range params {
		if err := r.validateParameter(paramSchema, name, value); err != nil {
			return err
		}
	}
	return nil
}

// validateParameter validates a single parameter.
func (r *Registry) validateParameter(paramSchema ParameterSchema, name string, value interface{}) error {
	propDef, exists := paramSchema.Properties[name]
	if !exists {
		return r.createUnknownParameterError(name, paramSchema.Properties)
	}

	if !r.validateType(value, propDef.Type) {
		return fmt.Errorf("%w: parameter %s has wrong type (expected %s)",
			ErrInvalidParameters, name, propDef.Type)
	}

	if len(propDef.Enum) > 0 {
		if err := r.validateEnum(value, propDef.Enum); err != nil {
			return fmt.Errorf("%w: parameter %s %v", ErrInvalidParameters, name, err)
		}
	}

	return nil
}

// createUnknownParameterError creates an error for unknown parameters.
func (r *Registry) createUnknownParameterError(name string, properties map[string]PropertyDefinition) error {
	validParams := make([]string, 0, len(properties))
	for pname := range properties {
		validParams = append(validParams, pname)
	}
	return fmt.Errorf("%w: unknown parameter %q (valid parameters: %v)",
		ErrInvalidParameters, name, validParams)
}

// validateType checks if a value matches the expected JSON schema type.
func (r *Registry) validateType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			return true
		default:
			return false
		}
	case "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
			return true
		default:
			return false
		}
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		// Check if it's a slice or array
		switch value.(type) {
		case []interface{}, []string, []int, []float64:
			return true
		default:
			return false
		}
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		// Unknown type - accept for now
		return true
	}
}

// validateEnum checks if a value is in the allowed enum values.
func (r *Registry) validateEnum(value interface{}, enum []string) error {
	strValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("enum value must be string")
	}

	for _, allowed := range enum {
		if strValue == allowed {
			return nil
		}
	}

	return fmt.Errorf("value %q not in allowed values %v", strValue, enum)
}
