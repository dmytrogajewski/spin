package blocks

// BlockType represents the category of block in the timeline.
type BlockType string

const (
	// BlockTypeExecute represents a shell command execution block.
	BlockTypeExecute BlockType = "EXECUTE"
	// BlockTypePlan represents a planned steps list block.
	BlockTypePlan BlockType = "PLAN"
	// BlockTypeRead represents a file read preview block.
	BlockTypeRead BlockType = "READ"
	// BlockTypeGrep represents a search result block.
	BlockTypeGrep BlockType = "GREP"
	// BlockTypeApplyPatch represents a patch application result block.
	BlockTypeApplyPatch BlockType = "APPLY_PATCH"
	// BlockTypeSummary represents a human-readable changeset summary block.
	BlockTypeSummary BlockType = "SUMMARY"
	// BlockTypeTool represents a tool execution block.
	BlockTypeTool BlockType = "TOOL"
	// BlockTypeTesting represents a test execution rubric block.
	BlockTypeTesting BlockType = "TESTING"
	// BlockTypeNotice represents a system notification block.
	BlockTypeNotice BlockType = "NOTICE"
	// BlockTypeError represents an error message block.
	BlockTypeError BlockType = "ERROR"
	// BlockTypeSkill represents a skill activation block.
	BlockTypeSkill BlockType = "SKILL"
	// BlockTypeTask represents an A2A or shell task-state block.
	BlockTypeTask BlockType = "TASK"
	// BlockTypeSubagent represents a child-process spawn/complete block.
	BlockTypeSubagent BlockType = "SUBAGENT"
	// BlockTypeHook represents a lifecycle hook result (including veto).
	BlockTypeHook BlockType = "HOOK"
	// BlockTypeCompact represents a history-compact event block.
	BlockTypeCompact BlockType = "COMPACT"
)

// String returns the string representation of the block type.
func (bt BlockType) String() string {
	return string(bt)
}

// Valid returns true if the block type is valid.
func (bt BlockType) Valid() bool {
	switch bt {
	case BlockTypeExecute, BlockTypePlan, BlockTypeRead, BlockTypeGrep,
		BlockTypeApplyPatch, BlockTypeSummary, BlockTypeTool, BlockTypeTesting,
		BlockTypeNotice, BlockTypeError, BlockTypeSkill,
		BlockTypeTask, BlockTypeSubagent, BlockTypeHook, BlockTypeCompact:
		return true
	}

	return false
}

// FoldState represents the collapse/expand state of a block.
type FoldState string

const (
	// FoldStateExpanded indicates the block is expanded (body visible).
	FoldStateExpanded FoldState = "expanded"
	// FoldStateCollapsed indicates the block is collapsed (body hidden).
	FoldStateCollapsed FoldState = "collapsed"
)

// String returns the string representation of the fold state.
func (fs FoldState) String() string {
	return string(fs)
}

// Valid returns true if the fold state is valid.
func (fs FoldState) Valid() bool {
	return fs == FoldStateExpanded || fs == FoldStateCollapsed
}

// Severity represents the importance/criticality level of a block.
type Severity string

const (
	// SeverityInfo indicates informational content.
	SeverityInfo Severity = "info"
	// SeverityWarn indicates a warning.
	SeverityWarn Severity = "warn"
	// SeverityError indicates an error.
	SeverityError Severity = "error"
)

// String returns the string representation of the severity.
func (s Severity) String() string {
	return string(s)
}

// Valid returns true if the severity is valid.
func (s Severity) Valid() bool {
	return s == SeverityInfo || s == SeverityWarn || s == SeverityError
}
