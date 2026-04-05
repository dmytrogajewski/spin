package reminder

// Detector names for all 8 event patterns.
const (
	NameToolFailure         = "tool_failure"
	NameExplorationSpiral   = "exploration_spiral"
	NamePrematureComplete   = "premature_completion"
	NameEmptyCompletion     = "empty_completion"
	NameDeniedToolRetry     = "denied_tool_retry"
	NameCompletedTodos      = "completed_todos"
	NamePlanNotExecuted     = "plan_not_executed"
	NameUnprocessedSubagent = "unprocessed_subagent"
)

// Default configuration.
const (
	defaultMaxFires           = 3
	explorationSpiralMinReads = 5
)

// Reminder template strings.
const (
	tmplToolFailure = "The previous tool call failed. " +
		"Review the error message and try a different approach."
	tmplExplorationSpiral = "You have been reading files extensively. " +
		"Consider taking action based on what you have learned so far."
	tmplPrematureComplete = "There appear to be incomplete tasks. " +
		"Please review your work before finishing."
	tmplEmptyCompletion = "Your last response was empty. " +
		"Please provide a substantive response."
	tmplDeniedToolRetry = "The tool call was denied by the approval system. " +
		"Do not retry denied tools — try an alternative approach."
	tmplCompletedTodos = "All tasks have been completed. " +
		"Consider wrapping up rather than continuing unnecessary work."
	tmplPlanNotExecuted = "An approved plan remains unexecuted. " +
		"Please proceed with executing the plan steps."
	tmplUnprocessedSubagent = "There are unprocessed subagent results. " +
		"Please review and incorporate the subagent output."
)

// SimpleDetector is a generic detector driven by a name, fire cap, and
// check function. It replaces the 8 zero-size detector structs that all
// followed the same pattern.
type SimpleDetector struct {
	name     string
	maxFires int
	check    func(CheckContext) bool
}

// NewToolFailureDetector returns a detector that fires when the last tool call failed.
func NewToolFailureDetector() *SimpleDetector {
	return &SimpleDetector{
		name:     NameToolFailure,
		maxFires: defaultMaxFires,
		check:    func(ctx CheckContext) bool { return ctx.LastToolFailed },
	}
}

// NewExplorationSpiralDetector returns a detector that fires when too many
// consecutive reads occur.
func NewExplorationSpiralDetector() *SimpleDetector {
	return &SimpleDetector{
		name:     NameExplorationSpiral,
		maxFires: defaultMaxFires,
		check:    func(ctx CheckContext) bool { return ctx.ConsecutiveReads >= explorationSpiralMinReads },
	}
}

// NewPrematureCompletionDetector returns a detector that fires when the agent
// appears done but has incomplete todos.
func NewPrematureCompletionDetector() *SimpleDetector {
	return &SimpleDetector{
		name:     NamePrematureComplete,
		maxFires: defaultMaxFires,
		check:    func(ctx CheckContext) bool { return ctx.HasIncompleteTodos },
	}
}

// NewEmptyCompletionDetector returns a detector that fires when the last
// assistant message was empty.
func NewEmptyCompletionDetector() *SimpleDetector {
	return &SimpleDetector{
		name:     NameEmptyCompletion,
		maxFires: defaultMaxFires,
		check:    func(ctx CheckContext) bool { return ctx.LastAssistantEmpty },
	}
}

// NewDeniedToolRetryDetector returns a detector that fires when the agent
// retries a tool call denied by approval.
func NewDeniedToolRetryDetector() *SimpleDetector {
	return &SimpleDetector{
		name:     NameDeniedToolRetry,
		maxFires: defaultMaxFires,
		check:    func(ctx CheckContext) bool { return ctx.LastToolDenied },
	}
}

// NewCompletedTodosDetector returns a detector that fires when the agent
// continues working after all todos are done.
func NewCompletedTodosDetector() *SimpleDetector {
	return &SimpleDetector{
		name:     NameCompletedTodos,
		maxFires: defaultMaxFires,
		check:    func(ctx CheckContext) bool { return ctx.AllTodosComplete },
	}
}

// NewPlanNotExecutedDetector returns a detector that fires when a plan was
// approved but remains unexecuted.
func NewPlanNotExecutedDetector() *SimpleDetector {
	return &SimpleDetector{
		name:     NamePlanNotExecuted,
		maxFires: defaultMaxFires,
		check:    func(ctx CheckContext) bool { return ctx.PlanApprovedNotExecuted },
	}
}

// NewUnprocessedSubagentDetector returns a detector that fires when subagent
// results were returned but not processed.
func NewUnprocessedSubagentDetector() *SimpleDetector {
	return &SimpleDetector{
		name:     NameUnprocessedSubagent,
		maxFires: defaultMaxFires,
		check:    func(ctx CheckContext) bool { return ctx.HasUnprocessedSubagentResults },
	}
}

// Name returns the detector identifier.
func (d *SimpleDetector) Name() string { return d.name }

// MaxFires returns the fire cap.
func (d *SimpleDetector) MaxFires() int { return d.maxFires }

// Check evaluates the detector's condition against the given context.
func (d *SimpleDetector) Check(ctx CheckContext) bool { return d.check(ctx) }

// DefaultDetectors returns all 8 event-pattern detectors.
func DefaultDetectors() []Detector {
	return []Detector{
		NewToolFailureDetector(),
		NewExplorationSpiralDetector(),
		NewPrematureCompletionDetector(),
		NewEmptyCompletionDetector(),
		NewDeniedToolRetryDetector(),
		NewCompletedTodosDetector(),
		NewPlanNotExecutedDetector(),
		NewUnprocessedSubagentDetector(),
	}
}

// DefaultTemplates returns reminder templates for all default detectors.
func DefaultTemplates() map[string]string {
	return map[string]string{
		NameToolFailure:         tmplToolFailure,
		NameExplorationSpiral:   tmplExplorationSpiral,
		NamePrematureComplete:   tmplPrematureComplete,
		NameEmptyCompletion:     tmplEmptyCompletion,
		NameDeniedToolRetry:     tmplDeniedToolRetry,
		NameCompletedTodos:      tmplCompletedTodos,
		NamePlanNotExecuted:     tmplPlanNotExecuted,
		NameUnprocessedSubagent: tmplUnprocessedSubagent,
	}
}
