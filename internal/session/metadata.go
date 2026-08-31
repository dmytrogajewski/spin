package session

// CostTracking holds cumulative LLM cost metrics for a session.
type CostTracking struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	APICallCount int     `json:"api_call_count"`
}

// AgentTask is a persisted A2A registry row (id, spec, state).
type AgentTask struct {
	ID    string `json:"id"`
	Spec  string `json:"spec"`
	State string `json:"state"`
}

// Metadata contains session metadata.
type Metadata struct {
	Title        string       // User-friendly session title.
	Description  string       // Session description.
	Tags         []string     // User-defined tags.
	TotalTurns   int          // Total turn count.
	TokensUsed   int          // Total tokens consumed (input + output).
	LastError    string       // Last error message (if any).
	CostTracking CostTracking `json:"cost_tracking"`
	AgentTasks   []AgentTask  `json:"agent_tasks,omitempty"`
}
