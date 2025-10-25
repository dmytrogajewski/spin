package blocks

import (
	"fmt"
	"strings"
)

// ToolParamsFormatter formats tool parameters for block headers.
// This provides compact, tool-specific parameter display using the Strategy pattern.
type ToolParamsFormatter interface {
	// FormatTitle formats the block header title (parameters + result).
	// Returns empty string if block can't be formatted.
	FormatTitle(block *Block) string
}

// ParamsFormatterRegistry manages tool parameter formatters.
type ParamsFormatterRegistry struct {
	formatters map[BlockType]ToolParamsFormatter
	defaults   ToolParamsFormatter
}

// NewParamsFormatterRegistry creates a registry with built-in formatters.
func NewParamsFormatterRegistry() *ParamsFormatterRegistry {
	r := &ParamsFormatterRegistry{
		formatters: make(map[BlockType]ToolParamsFormatter),
	}

	// Register built-in formatters
	r.formatters[BlockTypeExecute] = &ExecuteParamsFormatter{}
	r.formatters[BlockTypeRead] = &ReadParamsFormatter{}
	r.formatters[BlockTypeApplyPatch] = &WriteParamsFormatter{}
	r.formatters[BlockTypeGrep] = &GrepParamsFormatter{}

	// Default formatter for unknown types
	r.defaults = &DefaultParamsFormatter{}

	return r
}

// FormatTitle formats the title for a block header.
func (r *ParamsFormatterRegistry) FormatTitle(block *Block) string {
	formatter := r.getFormatter(block.Type)
	return formatter.FormatTitle(block)
}

func (r *ParamsFormatterRegistry) getFormatter(blockType BlockType) ToolParamsFormatter {
	if f, ok := r.formatters[blockType]; ok {
		return f
	}
	return r.defaults
}

// --- Built-in Formatters ---

// ExecuteParamsFormatter formats execute_command parameters.
type ExecuteParamsFormatter struct{}

func (f *ExecuteParamsFormatter) FormatTitle(block *Block) string {
	meta, err := ParseExecuteMeta(block)
	if err != nil || meta == nil {
		return ""
	}

	var parts []string

	// Show title if set (for custom descriptions), otherwise show command
	if block.Title != "" {
		parts = append(parts, block.Title)
	} else if meta.Command != "" {
		parts = append(parts, meta.Command)
	}

	// Exit code (if completed)
	if meta.ExitCode != nil {
		if *meta.ExitCode == 0 {
			parts = append(parts, string(ColorMuted)+"→ exit 0"+string(ColorReset))
		} else {
			// Error color - use red (same as severity error)
			parts = append(parts, "\x1b[31m"+fmt.Sprintf("→ exit %d", *meta.ExitCode)+string(ColorReset))
		}
	}

	// Line count
	if meta.LinesOut != nil && *meta.LinesOut > 0 {
		parts = append(parts, string(ColorMuted)+fmt.Sprintf("(%d lines)", *meta.LinesOut)+string(ColorReset))
	}

	return strings.Join(parts, " ")
}

// ReadParamsFormatter formats read_file parameters.
type ReadParamsFormatter struct{}

func (f *ReadParamsFormatter) FormatTitle(block *Block) string {
	meta, err := ParseReadMeta(block)
	if err != nil || meta == nil {
		return ""
	}

	var parts []string
	parts = append(parts, meta.File)

	// Show offset/limit if set
	var opts []string
	if meta.Offset != 0 {
		opts = append(opts, fmt.Sprintf("offset: %d", meta.Offset))
	}
	if meta.Limit != 0 {
		opts = append(opts, fmt.Sprintf("limit: %d", meta.Limit))
	}

	if len(opts) > 0 {
		parts = append(parts, string(ColorMuted)+"("+strings.Join(opts, ", ")+")"+string(ColorReset))
	}

	return strings.Join(parts, " ")
}

// WriteParamsFormatter formats write_file parameters.
type WriteParamsFormatter struct{}

func (f *WriteParamsFormatter) FormatTitle(block *Block) string {
	meta, err := ParsePatchMeta(block)
	if err != nil || meta == nil {
		return ""
	}

	// Just show the file path
	return meta.File
}

// GrepParamsFormatter formats grep parameters.
type GrepParamsFormatter struct{}

func (f *GrepParamsFormatter) FormatTitle(block *Block) string {
	meta, err := ParseGrepMeta(block)
	if err != nil || meta == nil {
		return ""
	}

	// Show pattern, optionally mode
	if meta.Mode != "" && meta.Mode != "files_with_matches" {
		return fmt.Sprintf("pattern: %s "+string(ColorMuted)+"(mode: %s)"+string(ColorReset), meta.Pattern, meta.Mode)
	}

	return fmt.Sprintf("pattern: %s", meta.Pattern)
}

// DefaultParamsFormatter is used for unknown tool types.
type DefaultParamsFormatter struct{}

func (f *DefaultParamsFormatter) FormatTitle(block *Block) string {
	// Fallback: use block title if available
	if block.Title != "" {
		return block.Title
	}

	// Try to render tool name and a compact params preview if available
	if meta, err := ParseToolMeta(block); err == nil && meta != nil {
		name := meta.ToolName
		if name == "" {
			return ""
		}
		// Build compact params list: key=value for a few keys
		var parts []string
		max := 3
		count := 0
		for k, v := range meta.Params {
			// Render primitive types compactly
			switch vv := v.(type) {
			case string:
				parts = append(parts, fmt.Sprintf("%s: %s", k, vv))
			case float64, bool, int, int64:
				parts = append(parts, fmt.Sprintf("%s: %v", k, vv))
			default:
				// skip complex values in the title
				continue
			}
			count++
			if count >= max {
				break
			}
		}
		if len(parts) > 0 {
			return fmt.Sprintf("%s %s(%s)%s", name, string(ColorMuted), strings.Join(parts, ", "), string(ColorReset))
		}
		return name
	}

	// No title available
	return ""
}
