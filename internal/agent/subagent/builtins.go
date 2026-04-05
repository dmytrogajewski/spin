package subagent

const (
	// NameExplorer is the read-only codebase navigation subagent.
	NameExplorer = "explorer"

	// NamePlanner is the strategic planning subagent with read-only access.
	NamePlanner = "planner"

	// NameReviewer is the code review subagent with diff analysis.
	NameReviewer = "reviewer"

	// NameAskUser is the structured user clarification subagent.
	NameAskUser = "ask_user"

	// defaultMaxIterations is the default iteration budget for subagents.
	defaultMaxIterations = 30
)

// explorerTools are the read-only tools available to the explorer subagent.
var explorerTools = []string{
	"read_file",
	"list_directory",
	"file_search",
	"get_context",
}

// plannerTools are the read-only tools available to the planner subagent.
var plannerTools = []string{
	"read_file",
	"list_directory",
	"file_search",
	"get_context",
}

// reviewerTools are the tools available to the reviewer subagent.
var reviewerTools = []string{
	"read_file",
	"list_directory",
	"file_search",
	"git_context",
}

// askUserTools are the tools available to the ask_user subagent.
var askUserTools = []string{
	"ask_user",
}

// explorerPrompt is the system prompt for the explorer subagent.
const explorerPrompt = "You are a codebase explorer. " +
	"Navigate and analyze code using read-only tools. " +
	"Stop when evidence is clear. " +
	"Re-reading the same file triggers immediate stop."

// plannerPrompt is the system prompt for the planner subagent.
const plannerPrompt = "You are a strategic planner. " +
	"Analyze the codebase and produce structured plans. " +
	"Use only read-only tools. " +
	"Stop if progress stalls."

// reviewerPrompt is the system prompt for the reviewer subagent.
const reviewerPrompt = "You are a code reviewer. " +
	"Analyze code changes for correctness, style, and potential issues. " +
	"Stop when the review is complete."

// askUserPrompt is the system prompt for the ask_user subagent.
const askUserPrompt = "You ask the user for clarification. " +
	"Formulate clear, specific questions. " +
	"Stop after receiving the answer."

// Builtins returns the Phase 1 built-in subagent specifications.
func Builtins() []*Spec {
	return []*Spec{
		{
			Name:          NameExplorer,
			Description:   "Read-only codebase navigation and analysis.",
			SystemPrompt:  explorerPrompt,
			AllowedTools:  explorerTools,
			MaxIterations: defaultMaxIterations,
		},
		{
			Name:          NamePlanner,
			Description:   "Strategic planning with read-only codebase access.",
			SystemPrompt:  plannerPrompt,
			AllowedTools:  plannerTools,
			MaxIterations: defaultMaxIterations,
		},
		{
			Name:          NameReviewer,
			Description:   "Code review with diff analysis capabilities.",
			SystemPrompt:  reviewerPrompt,
			AllowedTools:  reviewerTools,
			MaxIterations: defaultMaxIterations,
		},
		{
			Name:          NameAskUser,
			Description:   "Structured user clarification via ask_user tool.",
			SystemPrompt:  askUserPrompt,
			AllowedTools:  askUserTools,
			MaxIterations: defaultMaxIterations,
		},
	}
}
