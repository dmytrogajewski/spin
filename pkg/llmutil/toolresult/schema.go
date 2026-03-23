package toolresult

// Schema defines the OpenAI-compatible tool schema.
type Schema struct {
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
