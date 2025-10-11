package output

import (
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/types"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

// ToolFormatter formats tool call events into compact display strings.
// Each tool type has its own formatter implementation.
type ToolFormatter interface {
	// FormatStart formats the tool call initiation.
	// Returns a formatted string like: " EXECUTE  (go test ./...)"
	FormatStart(toolID string, params types.ToolCallArguments) string

	// FormatComplete formats the tool call result.
	// Returns a formatted string like: " ↳ Exit code: 0. Output: 42 lines."
	FormatComplete(success bool, output string, errorMsg string) string

	// Tag returns the display tag for this tool (e.g., "EXECUTE", "READ")
	Tag() string

	// Color returns the block color for this tool's tag
	Color() blocks.Color
}

// FormatterRegistry manages tool formatters using the Strategy pattern.
type FormatterRegistry struct {
	formatters map[string]ToolFormatter // toolName -> formatter
	defaults   ToolFormatter             // Fallback for unknown tools
	width      int                       // Terminal width for truncation
}

// NewFormatterRegistry creates a registry with built-in formatters.
func NewFormatterRegistry(width int) *FormatterRegistry {
	if width < 40 {
		width = 40 // Minimum reasonable width
	}

	r := &FormatterRegistry{
		formatters: make(map[string]ToolFormatter),
		width:      width,
	}

	// Register built-in formatters
	r.Register("execute_command", &ExecuteFormatter{width: width})
	r.Register("read_file", &ReadFormatter{width: width})
	r.Register("write_file", &WriteFormatter{width: width})
	r.Register("grep", &GrepFormatter{width: width})
	r.Register("list_directory", &ListFormatter{width: width})

	// Set default formatter for unknown tools
	r.defaults = &DefaultFormatter{width: width}

	return r
}

// Register adds a formatter for a specific tool name.
// This allows external packages to register custom formatters.
func (r *FormatterRegistry) Register(toolName string, formatter ToolFormatter) {
	r.formatters[toolName] = formatter
}

// FormatStart formats a tool call start event.
func (r *FormatterRegistry) FormatStart(toolName string, toolID string, params types.ToolCallArguments) string {
	formatter := r.getFormatter(toolName)

	// Get tag with color
	tag := r.formatTag(formatter.Tag(), formatter.Color())

	// Get formatted parameters from tool-specific formatter
	paramsStr := formatter.FormatStart(toolID, params)

	// Assemble: " TAG  (params)"
	if paramsStr == "" {
		return fmt.Sprintf(" %s", tag)
	}
	return fmt.Sprintf(" %s  (%s)", tag, paramsStr)
}

// FormatComplete formats a tool call completion event.
func (r *FormatterRegistry) FormatComplete(toolName string, success bool, output string, errorMsg string) string {
	formatter := r.getFormatter(toolName)
	result := formatter.FormatComplete(success, output, errorMsg)
	return fmt.Sprintf(" ↳ %s", result)
}

// getFormatter returns the formatter for a tool, or the default if not found.
func (r *FormatterRegistry) getFormatter(toolName string) ToolFormatter {
	if f, ok := r.formatters[toolName]; ok {
		return f
	}
	return r.defaults
}

// formatTag formats a tag with colored background.
func (r *FormatterRegistry) formatTag(tag string, color blocks.Color) string {
	// Convert foreground color to background color
	bgColor := fgToBg(color)

	// Add padding to make tag visually distinct
	paddedTag := " " + tag + " "

	return string(bgColor) + string(blocks.ColorBold) + paddedTag + string(blocks.ColorReset)
}

// fgToBg converts foreground color code to background color code.
func fgToBg(fg blocks.Color) blocks.Color {
	// Replace "38;5;" with "48;5;" (foreground → background)
	s := string(fg)
	s = strings.Replace(s, "38;5;", "48;5;", 1)
	// Add black text for readability on colored background
	s = strings.Replace(s, "\x1b[", "\x1b[30;", 1) // Add black foreground (30)
	return blocks.Color(s)
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// countLines counts the number of lines in a string.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	count := strings.Count(s, "\n")
	// If string doesn't end with newline, add 1
	if !strings.HasSuffix(s, "\n") {
		count++
	}
	return count
}

// getStringParam safely extracts a string parameter.
func getStringParam(params types.ToolCallArguments, key string) string {
	var s string
	if err := params.Get(key, &s); err == nil {
		return s
	}
	return ""
}

// getIntParam safely extracts an int parameter.
func getIntParam(params types.ToolCallArguments, key string) int {
	var i int
	if err := params.Get(key, &i); err == nil {
		return i
	}
	return 0
}

// --- Built-in Formatter Implementations ---

// ExecuteFormatter formats execute_command tool calls.
type ExecuteFormatter struct {
	width int
}

func (f *ExecuteFormatter) Tag() string {
	return "EXECUTE"
}

func (f *ExecuteFormatter) Color() blocks.Color {
	return blocks.GetTagColor(blocks.BlockTypeExecute)
}

func (f *ExecuteFormatter) FormatStart(toolID string, params types.ToolCallArguments) string {
	cmd := getStringParam(params, "command")
	if cmd == "" {
		return ""
	}

	// Truncate command to fit terminal
	maxLen := f.width - 20 // Reserve space for tag and formatting
	return truncateString(cmd, maxLen)
}

func (f *ExecuteFormatter) FormatComplete(success bool, output string, errorMsg string) string {
	if !success {
		if errorMsg != "" {
			return fmt.Sprintf("Failed: %s", errorMsg)
		}
		return "Failed"
	}

	lines := countLines(output)
	if lines == 0 {
		return "Exit code: 0. No output."
	}
	return fmt.Sprintf("Exit code: 0. Output: %d lines.", lines)
}

// ReadFormatter formats read_file tool calls.
type ReadFormatter struct {
	width int
}

func (f *ReadFormatter) Tag() string {
	return "READ"
}

func (f *ReadFormatter) Color() blocks.Color {
	return blocks.GetTagColor(blocks.BlockTypeRead)
}

func (f *ReadFormatter) FormatStart(toolID string, params types.ToolCallArguments) string {
	path := getStringParam(params, "path")
	if path == "" {
		return ""
	}

	var parts []string
	parts = append(parts, path)

	if offset := getIntParam(params, "offset"); offset != 0 {
		parts = append(parts, fmt.Sprintf("offset: %d", offset))
	}
	if limit := getIntParam(params, "limit"); limit != 0 {
		parts = append(parts, fmt.Sprintf("limit: %d", limit))
	}

	result := strings.Join(parts, ", ")
	maxLen := f.width - 20
	return truncateString(result, maxLen)
}

func (f *ReadFormatter) FormatComplete(success bool, output string, errorMsg string) string {
	if !success {
		if errorMsg != "" {
			return fmt.Sprintf("Failed: %s", errorMsg)
		}
		return "Failed"
	}

	lines := countLines(output)
	if lines == 0 {
		return "File is empty."
	}
	return fmt.Sprintf("Read %d lines.", lines)
}

// WriteFormatter formats write_file tool calls.
type WriteFormatter struct {
	width int
}

func (f *WriteFormatter) Tag() string {
	return "WRITE"
}

func (f *WriteFormatter) Color() blocks.Color {
	return blocks.GetTagColor(blocks.BlockTypeApplyPatch)
}

func (f *WriteFormatter) FormatStart(toolID string, params types.ToolCallArguments) string {
	path := getStringParam(params, "path")
	if path == "" {
		return ""
	}

	content := getStringParam(params, "content")
	if content != "" {
		return fmt.Sprintf("%s, %d bytes", path, len(content))
	}
	return path
}

func (f *WriteFormatter) FormatComplete(success bool, output string, errorMsg string) string {
	if !success {
		if errorMsg != "" {
			return fmt.Sprintf("Failed: %s", errorMsg)
		}
		return "Failed"
	}
	return "File written successfully."
}

// GrepFormatter formats grep tool calls.
type GrepFormatter struct {
	width int
}

func (f *GrepFormatter) Tag() string {
	return "GREP"
}

func (f *GrepFormatter) Color() blocks.Color {
	return blocks.GetTagColor(blocks.BlockTypeGrep)
}

func (f *GrepFormatter) FormatStart(toolID string, params types.ToolCallArguments) string {
	pattern := getStringParam(params, "pattern")
	if pattern == "" {
		return ""
	}

	var parts []string
	parts = append(parts, "pattern: "+pattern)

	if mode := getStringParam(params, "mode"); mode != "" {
		parts = append(parts, "mode: "+mode)
	}

	result := strings.Join(parts, ", ")
	maxLen := f.width - 20
	return truncateString(result, maxLen)
}

func (f *GrepFormatter) FormatComplete(success bool, output string, errorMsg string) string {
	if !success {
		if errorMsg != "" {
			return fmt.Sprintf("Failed: %s", errorMsg)
		}
		return "Failed"
	}

	lines := countLines(output)
	return fmt.Sprintf("Found %d matches.", lines)
}

// ListFormatter formats list_directory tool calls.
type ListFormatter struct {
	width int
}

func (f *ListFormatter) Tag() string {
	return "LIST"
}

func (f *ListFormatter) Color() blocks.Color {
	return blocks.GetTagColor(blocks.BlockTypeExecute)
}

func (f *ListFormatter) FormatStart(toolID string, params types.ToolCallArguments) string {
	path := getStringParam(params, "path")
	if path == "" {
		return ""
	}

	// Format as "ls <path>" for better UX
	result := "ls " + path
	maxLen := f.width - 20
	return truncateString(result, maxLen)
}

func (f *ListFormatter) FormatComplete(success bool, output string, errorMsg string) string {
	if !success {
		if errorMsg != "" {
			return fmt.Sprintf("Failed: %s", errorMsg)
		}
		return "Failed"
	}

	lines := countLines(output)
	return fmt.Sprintf("Listed %d items.", lines)
}

// DefaultFormatter is used for unknown tool types.
type DefaultFormatter struct {
	width int
}

func (f *DefaultFormatter) Tag() string {
	return "TOOL"
}

func (f *DefaultFormatter) Color() blocks.Color {
	return blocks.GetTagColor(blocks.BlockTypeNotice)
}

func (f *DefaultFormatter) FormatStart(toolID string, params types.ToolCallArguments) string {
	// Format all parameters as key:value pairs
	var parts []string
	for k, v := range params.ToMap() {
		parts = append(parts, fmt.Sprintf("%s: %v", k, v))
	}

	if len(parts) == 0 {
		return ""
	}

	result := strings.Join(parts, ", ")
	maxLen := f.width - 20
	return truncateString(result, maxLen)
}

func (f *DefaultFormatter) FormatComplete(success bool, output string, errorMsg string) string {
	if !success {
		if errorMsg != "" {
			return fmt.Sprintf("Failed: %s", errorMsg)
		}
		return "Failed"
	}
	return "Completed successfully."
}
