package hooks

// EventContext carries contextual data passed to hook scripts as JSON on stdin.
type EventContext struct {
	// Event is the lifecycle event name.
	Event Event `json:"event"`
	// SessionID identifies the current session.
	SessionID string `json:"session_id"`
	// WorkDir is the working directory of the session.
	WorkDir string `json:"work_dir"`
	// ToolName is the tool being invoked (empty for non-tool events).
	ToolName string `json:"tool_name,omitempty"`
	// ToolInput contains the tool's input arguments (empty for non-tool events).
	ToolInput string `json:"tool_input,omitempty"`
	// ToolResponse contains the tool output (only for post-tool events).
	ToolResponse string `json:"tool_response,omitempty"`
}

// HookResult captures the outcome of a hook script execution.
type HookResult struct {
	// Blocked is true when a blocking hook vetoed the operation (exit code 2).
	Blocked bool
	// Reason explains why the operation was blocked (from hook stdout).
	Reason string
	// UpdatedInput contains mutated tool input from the hook (JSON field).
	UpdatedInput string
}
