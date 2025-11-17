package session

// TokenUsage tracks token consumption for a turn.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Turn represents a single user/agent interaction within a session.
type Turn struct {
	ID         string     `json:"id"`
	SessionID  string     `json:"session_id"`
	UserInput  string     `json:"user_input"`
	AIResponse string     `json:"ai_response"`
	Tokens     TokenUsage `json:"tokens"`
}
