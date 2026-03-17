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

// ToolFailureDetector fires when the last tool call failed.
type ToolFailureDetector struct{}

// Name returns the detector identifier.
func (d *ToolFailureDetector) Name() string { return NameToolFailure }

// Check returns true when the last tool call failed.
func (d *ToolFailureDetector) Check(ctx CheckContext) bool { return ctx.LastToolFailed }

// MaxFires returns the fire cap.
func (d *ToolFailureDetector) MaxFires() int { return defaultMaxFires }

// ExplorationSpiralDetector fires when too many consecutive reads occur.
type ExplorationSpiralDetector struct{}

// Name returns the detector identifier.
func (d *ExplorationSpiralDetector) Name() string { return NameExplorationSpiral }

// Check returns true when consecutive reads reach the threshold.
func (d *ExplorationSpiralDetector) Check(ctx CheckContext) bool {
	return ctx.ConsecutiveReads >= explorationSpiralMinReads
}

// MaxFires returns the fire cap.
func (d *ExplorationSpiralDetector) MaxFires() int { return defaultMaxFires }

// PrematureCompletionDetector fires when the agent appears done but has incomplete todos.
type PrematureCompletionDetector struct{}

// Name returns the detector identifier.
func (d *PrematureCompletionDetector) Name() string { return NamePrematureComplete }

// Check returns true when incomplete todos are detected.
func (d *PrematureCompletionDetector) Check(ctx CheckContext) bool {
	return ctx.HasIncompleteTodos
}

// MaxFires returns the fire cap.
func (d *PrematureCompletionDetector) MaxFires() int { return defaultMaxFires }

// EmptyCompletionDetector fires when the last assistant message was empty.
type EmptyCompletionDetector struct{}

// Name returns the detector identifier.
func (d *EmptyCompletionDetector) Name() string { return NameEmptyCompletion }

// Check returns true when the last assistant message was empty.
func (d *EmptyCompletionDetector) Check(ctx CheckContext) bool {
	return ctx.LastAssistantEmpty
}

// MaxFires returns the fire cap.
func (d *EmptyCompletionDetector) MaxFires() int { return defaultMaxFires }

// DeniedToolRetryDetector fires when the agent retries a tool call denied by approval.
type DeniedToolRetryDetector struct{}

// Name returns the detector identifier.
func (d *DeniedToolRetryDetector) Name() string { return NameDeniedToolRetry }

// Check returns true when a denied tool was retried.
func (d *DeniedToolRetryDetector) Check(ctx CheckContext) bool {
	return ctx.LastToolDenied
}

// MaxFires returns the fire cap.
func (d *DeniedToolRetryDetector) MaxFires() int { return defaultMaxFires }

// CompletedTodosDetector fires when the agent continues working after all todos are done.
type CompletedTodosDetector struct{}

// Name returns the detector identifier.
func (d *CompletedTodosDetector) Name() string { return NameCompletedTodos }

// Check returns true when all todos are complete but the agent is still working.
func (d *CompletedTodosDetector) Check(ctx CheckContext) bool {
	return ctx.AllTodosComplete
}

// MaxFires returns the fire cap.
func (d *CompletedTodosDetector) MaxFires() int { return defaultMaxFires }

// PlanNotExecutedDetector fires when a plan was approved but remains unexecuted.
type PlanNotExecutedDetector struct{}

// Name returns the detector identifier.
func (d *PlanNotExecutedDetector) Name() string { return NamePlanNotExecuted }

// Check returns true when an approved plan has not been executed.
func (d *PlanNotExecutedDetector) Check(ctx CheckContext) bool {
	return ctx.PlanApprovedNotExecuted
}

// MaxFires returns the fire cap.
func (d *PlanNotExecutedDetector) MaxFires() int { return defaultMaxFires }

// UnprocessedSubagentDetector fires when subagent results were returned but not processed.
type UnprocessedSubagentDetector struct{}

// Name returns the detector identifier.
func (d *UnprocessedSubagentDetector) Name() string { return NameUnprocessedSubagent }

// Check returns true when unprocessed subagent results exist.
func (d *UnprocessedSubagentDetector) Check(ctx CheckContext) bool {
	return ctx.HasUnprocessedSubagentResults
}

// MaxFires returns the fire cap.
func (d *UnprocessedSubagentDetector) MaxFires() int { return defaultMaxFires }

// DefaultDetectors returns all 8 event-pattern detectors.
func DefaultDetectors() []Detector {
	return []Detector{
		&ToolFailureDetector{},
		&ExplorationSpiralDetector{},
		&PrematureCompletionDetector{},
		&EmptyCompletionDetector{},
		&DeniedToolRetryDetector{},
		&CompletedTodosDetector{},
		&PlanNotExecutedDetector{},
		&UnprocessedSubagentDetector{},
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
