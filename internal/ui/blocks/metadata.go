package blocks

import (
	"encoding/json"
	"fmt"
)

// ExecuteMeta holds metadata for EXECUTE blocks.
type ExecuteMeta struct {
	// Command is the shell command string.
	Command string `json:"command"`

	// CWD is the working directory where the command runs.
	CWD string `json:"cwd"`

	// TimeoutSec is the timeout in seconds (optional).
	TimeoutSec int `json:"timeout_sec,omitempty"`

	// Impact indicates the risk level: low, medium, or high.
	Impact string `json:"impact"`

	// ExitCode is the command exit code (nil if still running).
	ExitCode *int `json:"exit_code,omitempty"`

	// DurationMS is the execution duration in milliseconds (optional).
	DurationMS *int64 `json:"duration_ms,omitempty"`

	// LinesOut is the number of output lines (optional).
	LinesOut *int `json:"lines_out,omitempty"`
}

// Validate validates the execute metadata.
func (m *ExecuteMeta) Validate() error {
	if m.Command == "" {
		return fmt.Errorf("command is required")
	}
	if m.CWD == "" {
		return fmt.Errorf("cwd is required")
	}
	if m.Impact != "low" && m.Impact != "medium" && m.Impact != "high" {
		return fmt.Errorf("impact must be low, medium, or high")
	}
	if m.ExitCode != nil && *m.ExitCode < 0 {
		return fmt.Errorf("exit_code must be >= 0")
	}
	if m.DurationMS != nil && *m.DurationMS < 0 {
		return fmt.Errorf("duration_ms must be >= 0")
	}
	if m.LinesOut != nil && *m.LinesOut < 0 {
		return fmt.Errorf("lines_out must be >= 0")
	}
	return nil
}

// ReadMeta holds metadata for READ blocks.
type ReadMeta struct {
	// File is the file path being read.
	File string `json:"file"`

	// Offset is the starting line number (optional, default 0).
	Offset int `json:"offset,omitempty"`

	// Limit is the maximum number of lines to read (optional).
	Limit int `json:"limit,omitempty"`
}

// Validate validates the read metadata.
func (m *ReadMeta) Validate() error {
	if m.File == "" {
		return fmt.Errorf("file is required")
	}
	if m.Offset < 0 {
		return fmt.Errorf("offset must be >= 0")
	}
	if m.Limit < 0 {
		return fmt.Errorf("limit must be >= 0")
	}
	return nil
}

// GrepMeta holds metadata for GREP blocks.
type GrepMeta struct {
	// Pattern is the search pattern (regex).
	Pattern string `json:"pattern"`

	// Mode is the output mode: content, files_with_matches, or count.
	Mode string `json:"mode"`

	// Context is the number of context lines (for -A/-B/-C, optional).
	Context int `json:"context,omitempty"`
}

// Validate validates the grep metadata.
func (m *GrepMeta) Validate() error {
	if m.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}
	if m.Mode != "content" && m.Mode != "files_with_matches" && m.Mode != "count" {
		return fmt.Errorf("mode must be content, files_with_matches, or count")
	}
	if m.Context < 0 {
		return fmt.Errorf("context must be >= 0")
	}
	return nil
}

// ToolMeta holds metadata for TOOL blocks.
type ToolMeta struct {
	// ToolName is the name of the tool.
	ToolName string `json:"tool_name"`
}

// Validate validates the tool metadata.
func (m *ToolMeta) Validate() error {
	if m.ToolName == "" {
		return fmt.Errorf("tool_name is required")
	}
	return nil
}

// ParseToolMeta extracts ToolMeta from a block's metadata.
func ParseToolMeta(b *Block) (*ToolMeta, error) {
	data, err := json.Marshal(b.Meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	var meta ToolMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ToolMeta: %w", err)
	}
	return &meta, nil
}

// PatchMeta holds metadata for APPLY_PATCH blocks.
type PatchMeta struct {
	// File is the target file path.
	File string `json:"file"`

	// Succeeded indicates if the patch applied successfully.
	Succeeded bool `json:"succeeded"`

	// Completed indicates whether the write/apply operation has finished.
	// Rendering should suppress success/failure lines until Completed is true.
	Completed bool `json:"completed,omitempty"`

	// LinesAdded is the number of lines added (optional).
	LinesAdded *int `json:"lines_added,omitempty"`

	// LinesRemoved is the number of lines removed (optional).
	LinesRemoved *int `json:"lines_removed,omitempty"`

	// ErrorMsg contains the error message if patch failed (optional).
	ErrorMsg string `json:"error_msg,omitempty"`
}

// Validate validates the patch metadata.
func (m *PatchMeta) Validate() error {
	if m.File == "" {
		return fmt.Errorf("file is required")
	}
	if m.LinesAdded != nil && *m.LinesAdded < 0 {
		return fmt.Errorf("lines_added must be >= 0")
	}
	if m.LinesRemoved != nil && *m.LinesRemoved < 0 {
		return fmt.Errorf("lines_removed must be >= 0")
	}
	return nil
}

// PlanMeta holds metadata for PLAN blocks.
type PlanMeta struct {
	// Total is the total number of plan items.
	Total int `json:"total"`

	// Pending is the number of pending items.
	Pending int `json:"pending"`

	// InProgress is the number of in-progress items.
	InProgress int `json:"in_progress"`

	// Completed is the number of completed items.
	Completed int `json:"completed"`
}

// Validate validates the plan metadata.
func (m *PlanMeta) Validate() error {
	if m.Total < 0 {
		return fmt.Errorf("total must be >= 0")
	}
	if m.Pending < 0 {
		return fmt.Errorf("pending must be >= 0")
	}
	if m.InProgress < 0 {
		return fmt.Errorf("in_progress must be >= 0")
	}
	if m.Completed < 0 {
		return fmt.Errorf("completed must be >= 0")
	}
	sum := m.Pending + m.InProgress + m.Completed
	if sum != m.Total {
		return fmt.Errorf("pending + in_progress + completed (%d) must equal total (%d)", sum, m.Total)
	}
	return nil
}

// ParseExecuteMeta extracts ExecuteMeta from a block's metadata.
func ParseExecuteMeta(b *Block) (*ExecuteMeta, error) {
	data, err := json.Marshal(b.Meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	var meta ExecuteMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ExecuteMeta: %w", err)
	}
	return &meta, nil
}

// ParseReadMeta extracts ReadMeta from a block's metadata.
func ParseReadMeta(b *Block) (*ReadMeta, error) {
	data, err := json.Marshal(b.Meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	var meta ReadMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ReadMeta: %w", err)
	}
	return &meta, nil
}

// ParseGrepMeta extracts GrepMeta from a block's metadata.
func ParseGrepMeta(b *Block) (*GrepMeta, error) {
	data, err := json.Marshal(b.Meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	var meta GrepMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GrepMeta: %w", err)
	}
	return &meta, nil
}

// ParsePatchMeta extracts PatchMeta from a block's metadata.
func ParsePatchMeta(b *Block) (*PatchMeta, error) {
	data, err := json.Marshal(b.Meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	var meta PatchMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PatchMeta: %w", err)
	}
	return &meta, nil
}

// ParsePlanMeta extracts PlanMeta from a block's metadata.
func ParsePlanMeta(b *Block) (*PlanMeta, error) {
	data, err := json.Marshal(b.Meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	var meta PlanMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PlanMeta: %w", err)
	}
	return &meta, nil
}

// SetExecuteMeta sets ExecuteMeta on a block.
func SetExecuteMeta(b *Block, m *ExecuteMeta) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("invalid ExecuteMeta: %w", err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal ExecuteMeta: %w", err)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}
	b.Meta = meta
	return nil
}

// SetReadMeta sets ReadMeta on a block.
func SetReadMeta(b *Block, m *ReadMeta) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("invalid ReadMeta: %w", err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal ReadMeta: %w", err)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}
	b.Meta = meta
	return nil
}

// SetGrepMeta sets GrepMeta on a block.
func SetGrepMeta(b *Block, m *GrepMeta) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("invalid GrepMeta: %w", err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal GrepMeta: %w", err)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}
	b.Meta = meta
	return nil
}

// SetToolMeta sets ToolMeta on a block.
func SetToolMeta(b *Block, m *ToolMeta) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("invalid ToolMeta: %w", err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal ToolMeta: %w", err)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}
	b.Meta = meta
	return nil
}

// SetPatchMeta sets PatchMeta on a block.
func SetPatchMeta(b *Block, m *PatchMeta) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("invalid PatchMeta: %w", err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal PatchMeta: %w", err)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}
	b.Meta = meta
	return nil
}

// SetPlanMeta sets PlanMeta on a block.
func SetPlanMeta(b *Block, m *PlanMeta) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("invalid PlanMeta: %w", err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal PlanMeta: %w", err)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}
	b.Meta = meta
	return nil
}
