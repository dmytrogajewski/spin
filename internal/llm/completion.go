package llm

// Capabilities represents provider capabilities.
type Capabilities struct {
	// Streaming indicates if the provider supports streaming
	Streaming bool

	// FunctionCalling indicates if the provider supports function calling
	FunctionCalling bool

	// Vision indicates if the provider supports vision/image inputs
	Vision bool
}
