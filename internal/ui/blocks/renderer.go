package blocks

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/pkg/ui/textwidth"
)

// ErrBlockIsNil is a sentinel error.
var ErrBlockIsNil = errors.New("block is nil")

// Rendering constants.
const (
	msPerSecond = 1000.0 // milliseconds per second for duration formatting.
	// ANSI 256-color codes for block type badges.
	colorBlue256    = 63
	colorOrange256  = 208
	colorYellow256  = 220
	colorMagenta256 = 205
	colorPurple256  = 141
	colorCyan256    = 45
	colorGreen256   = 34
	colorGray256    = 244
	colorRed256     = 196

	// Block ID generation.
	blockIDSeqMod = 100

	// Header layout constants.
	minTitleWidth      = 10
	rightMetaReserved  = 20
	labelPaddingSpaces = 2
	accentBarCells     = 2 // glyph + trailing space.
)

// Renderer renders blocks to ANSI terminal output.
type Renderer struct {
	width           int                      // Terminal width in columns.
	paramsFormatter *ParamsFormatterRegistry // Tool parameter formatter (Strategy pattern).
}

// NewRenderer creates a new block renderer with the given terminal width.
// Uses legacy hardcoded colors for backward compatibility.
func NewRenderer(width int) *Renderer {
	if width <= 0 {
		width = 80 // Default width.
	}

	return &Renderer{
		width:           width,
		paramsFormatter: NewParamsFormatterRegistry(),
	}
}

// SetWidth updates the terminal width (e.g., on window resize).
func (r *Renderer) SetWidth(width int) {
	if width > 0 {
		r.width = width
	}
}

// Render renders a complete block to a string.
// Returns ANSI-formatted string suitable for terminal output.
func (r *Renderer) Render(b *Block) (string, error) {
	if b == nil {
		return "", ErrBlockIsNil
	}

	var out strings.Builder

	// Always start with a newline to ensure block appears on its own line
	// This prevents overlap when tool blocks are appended while streaming is still active.
	out.WriteString("\n")

	if activity := FormatActivity(b); activity != "" {
		out.WriteString(activity)
		out.WriteString("\n")
	} else {
		out.WriteString(r.RenderHeader(b))
		out.WriteString("\n")
	}

	// Render completion status line (if tool has completed).
	statusLine := r.RenderCompletionStatus(b)
	if statusLine != "" {
		out.WriteString(statusLine)
		out.WriteString("\n")
	}

	// Render body (if expanded).
	if b.FoldState == FoldStateExpanded && b.Body != "" {
		body, err := r.RenderBody(b)
		if err != nil {
			return "", fmt.Errorf("render body: %w", err)
		}

		out.WriteString(body)
	} else if b.FoldState == FoldStateCollapsed {
		// Show collapsed badge.
		out.WriteString(strings.Repeat(" ", S2))
		out.WriteString(string(ColorMuted))
		out.WriteString("⟦ collapsed ⟧")
		out.WriteString(string(ColorReset))
		out.WriteString("\n")
	}

	// Render footer.
	footer := r.RenderFooter(b)
	if footer != "" {
		out.WriteString(footer)
		out.WriteString("\n")
	}

	return out.String(), nil
}

// RenderHeader renders only the block header.
func (r *Renderer) RenderHeader(b *Block) string {
	if b == nil {
		return ""
	}

	var out strings.Builder

	// 1-cell accent gutter on the same line as the badge.
	tagColor := GetTagColor(b.Type)
	out.WriteString(string(tagColor))
	out.WriteString(AccentBarGlyph)
	out.WriteString(string(ColorReset))
	out.WriteString(" ")

	// Tag badge with colored background
	// Get background color and label for block type.
	bgColor := r.getBlockTypeColor(b.Type)
	label := r.getBlockTypeLabel(b.Type)

	fmt.Fprintf(&out, "\x1b[48;5;%dm", bgColor) // Background color.
	out.WriteString("\x1b[38;5;232m")           // Black text for contrast.
	out.WriteString(string(ColorBold))
	fmt.Fprintf(&out, " %s ", label)
	out.WriteString(string(ColorReset))

	// Spacing after tag.
	out.WriteString(strings.Repeat(" ", S2))

	// Title/Meta (left-aligned).
	title := r.formatTitle(b)
	// Calculate available width: total - accent - spacing - label - spacing - rightmeta.
	maxTitleWidth := max(
		r.width-accentBarCells-S2-len(label)-labelPaddingSpaces-S2-S3-rightMetaReserved, minTitleWidth)

	title = textwidth.MidEllipsize(title, maxTitleWidth)
	out.WriteString(title)

	return out.String()
}

// formatTitle formats the title/meta line based on block type.
// Delegates to tool-specific formatters using Strategy pattern.
func (r *Renderer) formatTitle(b *Block) string {
	// Try tool-specific formatter first (for EXECUTE, READ, WRITE, GREP).
	if formatted := r.paramsFormatter.FormatTitle(b); formatted != "" {
		return formatted
	}

	// Fallback for special block types without dedicated formatters.
	switch b.Type {
	case BlockTypePlan:
		return r.formatPlanTitle(b)
	default:
		return r.formatGenericTitle(b)
	}
}

// formatPlanTitle formats the title for plan blocks.
func (r *Renderer) formatPlanTitle(b *Block) string {
	var parts []string
	if b.Title != "" {
		parts = append(parts, string(ColorBold)+b.Title+string(ColorReset))
	}

	meta, err := ParsePlanMeta(b)
	if err == nil {
		metaStr := fmt.Sprintf("Updated: %d total (%d pending, %d in progress, %d completed)",
			meta.Total, meta.Pending, meta.InProgress, meta.Completed)
		parts = append(parts, metaStr)
	}

	return strings.Join(parts, " ")
}

// formatGenericTitle formats the title for generic blocks.
func (r *Renderer) formatGenericTitle(b *Block) string {
	if b.Title != "" {
		return string(ColorBold) + b.Title + string(ColorReset)
	}

	return ""
}

// RenderBody renders only the block body based on type.
func (r *Renderer) RenderBody(b *Block) (string, error) {
	if b == nil || b.Body == "" {
		return "", nil
	}

	switch b.Type {
	case BlockTypeExecute, BlockTypeNotice, BlockTypeTool, BlockTypeSkill,
		BlockTypeTask, BlockTypeSubagent, BlockTypeHook, BlockTypeCompact:
		return r.renderTranscript(b)
	case BlockTypeRead, BlockTypeGrep:
		return r.renderCode(b)
	case BlockTypeApplyPatch:
		return r.renderDiff(b)
	case BlockTypePlan, BlockTypeSummary, BlockTypeTesting:
		return r.renderList(b)
	case BlockTypeError:
		return r.renderError(b)
	default:
		return r.renderTranscript(b)
	}
}

// RenderFooter renders only the block footer.
func (r *Renderer) RenderFooter(b *Block) string {
	if b == nil {
		return ""
	}

	chips := r.getFooterChips(b)
	if len(chips) == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString(strings.Repeat(" ", S2))
	out.WriteString(strings.Join(chips, " "))

	return out.String()
}

// getFooterChips returns footer chips based on block type.
func (r *Renderer) getFooterChips(b *Block) []string {
	switch b.Type {
	case BlockTypeExecute:
		return r.getExecuteFooterChips(b)
	case BlockTypeApplyPatch:
		return r.getPatchFooterChips(b)
	default:
		return nil
	}
}

// getExecuteFooterChips returns footer chips for execute blocks.
func (r *Renderer) getExecuteFooterChips(b *Block) []string {
	meta, err := ParseExecuteMeta(b)
	if err != nil {
		return nil
	}

	var chips []string
	if meta.ExitCode != nil {
		chips = append(chips, fmt.Sprintf("[exit: %d]", *meta.ExitCode))
	}

	if meta.LinesOut != nil {
		chips = append(chips, fmt.Sprintf("[out: %d lines]", *meta.LinesOut))
	}

	if meta.DurationMS != nil {
		dur := float64(*meta.DurationMS) / msPerSecond
		chips = append(chips, fmt.Sprintf("[dur: %.1fs]", dur))
	}

	return chips
}

// getPatchFooterChips returns footer chips for patch blocks.
func (r *Renderer) getPatchFooterChips(b *Block) []string {
	meta, err := ParsePatchMeta(b)
	if err != nil {
		return nil
	}

	// Only show footer chips after completion.
	if !meta.Completed {
		return nil
	}

	if meta.Succeeded {
		return r.getPatchSuccessChips(meta)
	}

	return r.getPatchFailureChips(meta)
}

// getPatchSuccessChips returns success chips for patch blocks.
func (r *Renderer) getPatchSuccessChips(meta *PatchMeta) []string {
	prefix := string(ColorGreen) + "✓" + string(ColorReset)

	msg := " Succeeded. File edited."
	if meta.LinesAdded != nil && *meta.LinesAdded > 0 {
		msg += fmt.Sprintf(" (+%d added)", *meta.LinesAdded)
	}

	return []string{prefix + msg}
}

// getPatchFailureChips returns failure chips for patch blocks.
func (r *Renderer) getPatchFailureChips(meta *PatchMeta) []string {
	prefix := string(ColorRed) + "●" + string(ColorReset)

	msg := " Failed"
	if meta.ErrorMsg != "" {
		msg += ": " + meta.ErrorMsg
	}

	return []string{prefix + msg}
}

// renderTranscript renders plain text transcript (for EXECUTE, NOTICE).
func (r *Renderer) renderTranscript(b *Block) (string, error) {
	lines := strings.Split(b.Body, "\n")

	var out strings.Builder

	for _, line := range lines {
		out.WriteString(strings.Repeat(" ", S2))
		out.WriteString(line)
		out.WriteString("\n")
	}

	return out.String(), nil
}

// renderCode renders code with line numbers (for READ, GREP).
func (r *Renderer) renderCode(b *Block) (string, error) {
	lines := strings.Split(b.Body, "\n")
	visible, hidden := truncatePreview(lines, maxPreviewLines)
	lineCount := len(visible)
	gutterWidth := textwidth.GutterWidth(lineCount)

	var out strings.Builder

	for i, line := range visible {
		lineNum := i + 1

		// Gutter.
		out.WriteString(strings.Repeat(" ", S2))
		out.WriteString(string(ColorMuted))
		fmt.Fprintf(&out, "%*d ", gutterWidth-1, lineNum)
		out.WriteString(string(ColorReset))

		// Line content.
		out.WriteString(line)
		out.WriteString("\n")
	}

	if hidden > 0 {
		out.WriteString(r.paintTruncationFooter(hidden))
		out.WriteString("\n")
	}

	return out.String(), nil
}

// renderDiff renders unified diff (for APPLY_PATCH).
func (r *Renderer) renderDiff(b *Block) (string, error) {
	visible, hidden := truncatePreview(strings.Split(b.Body, "\n"), maxPreviewLines)

	var out strings.Builder

	for _, line := range visible {
		out.WriteString(r.paintBoxLine(line))
		out.WriteString("\n")
	}

	if hidden > 0 {
		out.WriteString(r.paintTruncationFooter(hidden))
		out.WriteString("\n")
	}

	return out.String(), nil
}

// renderList renders bullet lists (for PLAN, SUMMARY, TESTING).
func (r *Renderer) renderList(b *Block) (string, error) {
	lines := strings.Split(b.Body, "\n")

	var out strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			out.WriteString("\n")

			continue
		}

		out.WriteString(strings.Repeat(" ", S2))

		// Detect bullet type.
		switch {
		case strings.HasPrefix(line, "✓ ") || strings.HasPrefix(line, "- [x] "):
			// Done item.
			out.WriteString(string(ColorGreen))
			out.WriteString("✓")
			out.WriteString(string(ColorReset))
			out.WriteString(" ")

			text := strings.TrimPrefix(line, "✓ ")
			text = strings.TrimPrefix(text, "- [x] ")

			out.WriteString(string(ColorDim))
			out.WriteString(text)
			out.WriteString(string(ColorReset))
		case strings.HasPrefix(line, "◦ "):
			// Skipped item.
			out.WriteString(string(ColorMuted))
			out.WriteString("◦")
			out.WriteString(string(ColorReset))
			out.WriteString(" ")

			text := strings.TrimPrefix(line, "◦ ")

			out.WriteString(string(ColorMuted))
			out.WriteString(text)
			out.WriteString(string(ColorReset))
		case strings.HasPrefix(line, "• ") || strings.HasPrefix(line, "- [ ] "):
			// Pending item.
			out.WriteString("•")
			out.WriteString(" ")

			text := strings.TrimPrefix(line, "• ")
			text = strings.TrimPrefix(text, "- [ ] ")
			out.WriteString(text)
		default:
			// Plain text (paragraph).
			out.WriteString(line)
		}

		out.WriteString("\n")
	}

	return out.String(), nil
}

// renderError renders error blocks (specialized transcript).
func (r *Renderer) renderError(b *Block) (string, error) {
	lines := strings.Split(b.Body, "\n")

	var out strings.Builder

	for i, line := range lines {
		out.WriteString(strings.Repeat(" ", S2))

		if i == 0 {
			// First line is bold (error message).
			out.WriteString(string(ColorRed))
			out.WriteString(string(ColorBold))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		} else {
			// Subsequent lines are dim (stack trace).
			out.WriteString(string(ColorDim))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		}

		out.WriteString("\n")
	}

	return out.String(), nil
}

// blockTypeColors maps block types to 256-color background codes for badges.
var blockTypeColors = map[BlockType]int{
	BlockTypeExecute:    colorBlue256,
	BlockTypeRead:       colorOrange256,
	BlockTypeGrep:       colorYellow256,
	BlockTypeApplyPatch: colorMagenta256,
	BlockTypePlan:       colorPurple256,
	BlockTypeSummary:    colorCyan256,
	BlockTypeTesting:    colorGreen256,
	BlockTypeNotice:     colorGray256,
	BlockTypeError:      colorRed256,
	BlockTypeTool:       colorCyan256,
	BlockTypeSkill:      colorPurple256,
	BlockTypeTask:       colorCyan256,
	BlockTypeSubagent:   colorBlue256,
	BlockTypeHook:       colorYellow256,
	BlockTypeCompact:    colorGray256,
}

// getBlockTypeColor returns the 256-color background color for a block type badge.
func (r *Renderer) getBlockTypeColor(blockType BlockType) int {
	if color, exists := blockTypeColors[blockType]; exists {
		return color
	}

	return colorGray256
}

// getBlockTypeLabel returns the display label for a block type.
// Labels match ToolFormatter tags for consistency.
func (r *Renderer) getBlockTypeLabel(blockType BlockType) string {
	switch blockType {
	case BlockTypeExecute:
		return "EXECUTE"
	case BlockTypeApplyPatch:
		return "WRITE" // Match FRD format (WRITE instead of APPLY_PATCH).
	case BlockTypeTool:
		return "TOOL"
	default:
		return string(blockType)
	}
}

// RenderCompletionStatus renders the completion status line for completed tools.
// Returns empty string if tool hasn't completed or has no status to show.
func (r *Renderer) RenderCompletionStatus(b *Block) string {
	if b == nil {
		return ""
	}

	switch b.Type {
	case BlockTypeExecute:
		return r.renderExecuteCompletionStatus(b)
	case BlockTypeApplyPatch:
		return r.renderWriteCompletionStatus(b)
	case BlockTypeTool:
		return r.renderToolCompletionStatus(b)
	default:
		return ""
	}
}

// renderExecuteCompletionStatus renders completion status for EXECUTE blocks.
func (r *Renderer) renderExecuteCompletionStatus(b *Block) string {
	meta, err := ParseExecuteMeta(b)
	if err != nil || meta == nil || meta.ExitCode == nil {
		return "" // Not completed yet.
	}

	var parts []string

	// Exit code.
	if *meta.ExitCode == 0 {
		parts = append(parts, "Exit code: 0")
	} else {
		parts = append(parts, fmt.Sprintf("Exit code: %d", *meta.ExitCode))
	}

	// Output summary.
	if meta.LinesOut != nil {
		switch *meta.LinesOut {
		case 0:
			parts = append(parts, "No output")
		case 1:
			parts = append(parts, "Output: 1 line")
		default:
			parts = append(parts, fmt.Sprintf("Output: %d lines", *meta.LinesOut))
		}
	}

	// Duration.
	if meta.DurationMS != nil && *meta.DurationMS > 0 {
		dur := float64(*meta.DurationMS) / msPerSecond
		parts = append(parts, fmt.Sprintf("Duration: %.1fs", dur))
	}

	result := strings.Join(parts, ". ") + "."

	return fmt.Sprintf("%s %s", string(ColorMuted)+"⤷"+string(ColorReset), result)
}

// renderWriteCompletionStatus renders completion status for WRITE blocks.
func (r *Renderer) renderWriteCompletionStatus(b *Block) string {
	meta, err := ParsePatchMeta(b)
	if err != nil || meta == nil {
		return ""
	}

	// Only render status once the write/apply has completed.
	if !meta.Completed {
		return ""
	}

	if meta.Succeeded {
		return fmt.Sprintf("%s File written successfully.", string(ColorMuted)+"⤷"+string(ColorReset))
	}

	return fmt.Sprintf("%s Failed to write file.", string(ColorMuted)+"⤷"+string(ColorReset))
}

// renderToolCompletionStatus renders completion status for TOOL blocks.
func (r *Renderer) renderToolCompletionStatus(b *Block) string {
	meta, err := ParseToolMeta(b)
	if err != nil || meta == nil {
		return ""
	}

	// Only show completion message if the tool has actually completed
	// Check if the block title indicates completion (contains "completed" case-insensitive)
	// or if the body is not empty (tool has produced output).
	if b.Body == "" {
		return "" // Tool hasn't completed yet (no output).
	}

	// Show simple completion message.
	return fmt.Sprintf("%s Tool completed: %s", string(ColorMuted)+"⤷"+string(ColorReset), meta.ToolName)
}
