package llm

// CompletionRequest represents a request for LLM completion.
type CompletionRequest struct {
	// Messages is the conversation history
	Messages []Message

	// Model is the model identifier (e.g., "gpt-4", "llama2")
	Model string

	// Tools available for function calling
	Tools []Tool

	// MaxTokens is the maximum tokens to generate
	MaxTokens int

	// Temperature controls randomness (0.0 = deterministic, 2.0 = very random)
	Temperature float64

	// Stream indicates whether to stream the response
	Stream bool
}

// CompletionResponse represents a response from an LLM completion.
type CompletionResponse struct {
	// ID is the unique identifier for this completion
	ID string

	// Model is the model that generated the response
	Model string

	// Content is the generated text content
	Content string

	// ToolCalls are the tool calls requested by the model
	ToolCalls []ToolCall

	// Usage contains token usage information
	Usage Usage

	// FinishReason indicates why the generation stopped
	// Values: "stop", "length", "tool_calls", "content_filter"
	FinishReason string
}

// StreamChunk represents a chunk in a streaming response.
type StreamChunk struct {
	// Type indicates the type of chunk
	Type ChunkType

	// Content is the incremental content (for ContentDelta)
	Content string

	// ToolCall is the tool call data (for ToolCall chunks)
	ToolCall *ToolCall

	// FinishReason indicates completion reason (for Done chunk)
	FinishReason string

	// Error contains any error that occurred
	Error error
}

// ChunkType defines the types of streaming chunks.
type ChunkType int

const (
	// ChunkTypeContentDelta indicates incremental content
	ChunkTypeContentDelta ChunkType = iota

	// ChunkTypeToolCallStart indicates start of a tool call
	ChunkTypeToolCallStart

	// ChunkTypeToolCallDelta indicates incremental tool call data
	ChunkTypeToolCallDelta

	// ChunkTypeToolCallComplete indicates tool call completion
	ChunkTypeToolCallComplete

	// ChunkTypeDone indicates stream completion
	ChunkTypeDone

	// ChunkTypeError indicates an error occurred
	ChunkTypeError
)

// String returns the string representation of ChunkType.
func (t ChunkType) String() string {
	switch t {
	case ChunkTypeContentDelta:
		return "content_delta"
	case ChunkTypeToolCallStart:
		return "tool_call_start"
	case ChunkTypeToolCallDelta:
		return "tool_call_delta"
	case ChunkTypeToolCallComplete:
		return "tool_call_complete"
	case ChunkTypeDone:
		return "done"
	case ChunkTypeError:
		return "error"
	default:
		return "unknown"
	}
}

// Message represents a conversation message.
type Message struct {
	// Role is the message role: "system", "user", "assistant", or "tool"
	Role string

	// Content is the message content
	Content string

	// ToolCalls are tool calls made by the assistant
	ToolCalls []ToolCall

	// ToolCallID is the ID of the tool call this message responds to
	// Only used when Role is "tool"
	ToolCallID string
}

// ToolCall represents an AI tool invocation.
type ToolCall struct {
	// ID is the unique identifier for this tool call
	ID string

	// Type is the type of tool call (typically "function")
	Type string

	// Function contains the function call details
	Function FunctionCall
}

// FunctionCall represents a function call made by the AI.
type FunctionCall struct {
	// Name is the function name
	Name string

	// Arguments is the JSON-encoded function arguments
	Arguments string
}

// Tool represents a tool definition passed to the LLM.
type Tool struct {
	// Type is the tool type (typically "function")
	Type string

	// Function contains the function definition
	Function Function
}

// Function represents a function definition.
type Function struct {
	// Name is the function name
	Name string

	// Description describes what the function does
	Description string

	// Parameters is the JSON schema for function parameters
	Parameters interface{}
}

// Model represents an available LLM model.
type Model struct {
	// ID is the model identifier
	ID string

	// Name is the human-readable model name
	Name string

	// Description describes the model
	Description string

	// ContextSize is the maximum context window size in tokens
	ContextSize int
}

// Capabilities represents provider capabilities.
type Capabilities struct {
	// Streaming indicates if the provider supports streaming
	Streaming bool

	// FunctionCalling indicates if the provider supports function calling
	FunctionCalling bool

	// Vision indicates if the provider supports vision/image inputs
	Vision bool
}

// Usage represents token usage information.
type Usage struct {
	// PromptTokens is the number of tokens in the prompt
	PromptTokens int

	// CompletionTokens is the number of tokens in the completion
	CompletionTokens int

	// TotalTokens is the total tokens used (prompt + completion)
	TotalTokens int
}
