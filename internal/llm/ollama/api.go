package ollama

import "time"

// generateRequest represents an Ollama generate API request.
type generateRequest struct {
	Model       string                 `json:"model"`
	Prompt      string                 `json:"prompt"`
	Stream      bool                   `json:"stream"`
	Temperature float64                `json:"temperature,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// generateResponse represents an Ollama generate API response.
type generateResponse struct {
	Model              string `json:"model"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

// modelInfo represents Ollama model metadata.
type modelInfo struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest,omitempty"`
}

// tagsResponse represents the response from /api/tags.
type tagsResponse struct {
	Models []modelInfo `json:"models"`
}

// errorResponse represents an Ollama API error.
type errorResponse struct {
	Error string `json:"error"`
}

// chatRequest represents an Ollama chat API request (OpenAI-compatible).
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []chatTool    `json:"tools,omitempty"`
	Stream   bool          `json:"stream"`
	Options  *chatOptions  `json:"options,omitempty"`
}

// chatMessage represents a message in the chat API.
type chatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

// chatTool represents a tool definition in the chat API.
type chatTool struct {
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

// chatToolFunction represents a tool function definition.
type chatToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// chatToolCall represents a tool call in a message.
type chatToolCall struct {
	ID       string               `json:"id,omitempty"`   // Optional - Ollama doesn't always provide this
	Type     string               `json:"type,omitempty"` // Optional - Ollama doesn't always provide this
	Function chatToolCallFunction `json:"function"`
}

// chatToolCallFunction represents the function portion of a tool call.
type chatToolCallFunction struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"` // Can be object (Ollama) or string (OpenAI)
}

// chatOptions represents options for the chat API.
type chatOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
	// VRAM auto-tune related options (supported by Ollama)
	NumCtx int `json:"num_ctx,omitempty"`
	NumGPU int `json:"num_gpu,omitempty"`
}

// chatResponse represents an Ollama chat API response.
type chatResponse struct {
	Model   string      `json:"model"`
	Message chatMessage `json:"message"`
	// Some Ollama builds still emit deltas under "response" in streaming mode (legacy).
	Response  string    `json:"response,omitempty"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`

	// Token usage stats (when done=true)
	TotalDuration      int64 `json:"total_duration,omitempty"`
	LoadDuration       int64 `json:"load_duration,omitempty"`
	PromptEvalCount    int   `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount          int   `json:"eval_count,omitempty"`
	EvalDuration       int64 `json:"eval_duration,omitempty"`
}
