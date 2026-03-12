package blocks

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rivo/uniseg"
)

var ErrBlockIsNil = errors.New("block is nil")

// Rendering constants.
const (
	msPerSecond       = 1000.0 // milliseconds per second for duration formatting.
	ellipsisSplitLeft = 0.6    // left portion ratio for mid-ellipsis splitting.

	// Gutter width thresholds for line numbers.
	gutterThreshold10   = 10
	gutterThreshold100  = 100
	gutterThreshold1000 = 1000
	gutterWidth3        = 3
	gutterWidth4        = 4
	gutterWidth5        = 5
	gutterWidth6        = 6

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

	// Render header.
	header := r.RenderHeader(b)
	out.WriteString(header)
	out.WriteString("\n")

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

	// Accent bar (1ch).
	tagColor := GetTagColor(b.Type)
	out.WriteString(string(tagColor))
	out.WriteString(string(ColorReset))

	// Spacing after accent bar.

	// Newline before new tag.
	out.WriteString("\n")

	// out.WriteString(strings.Repeat(" ", S2)).

	// Tag badge with colored background
	// Get background color and label for block type.
	bgColor := r.getBlockTypeColor(b.Type)
	label := r.getBlockTypeLabel(b.Type)

	fmt.Fprintf(&out, "\x1b[48;5;%dm", bgColor) // Background color.
	out.WriteString("\x1b[38;5;232m")                      // Black text for contrast.
	out.WriteString(string(ColorBold))
	fmt.Fprintf(&out, " %s ", label)
	out.WriteString(string(ColorReset))

	// Spacing after tag.
	out.WriteString(strings.Repeat(" ", S2))

	// Title/Meta (left-aligned).
	title := r.formatTitle(b)
	// Calculate available width: total - bar - spacing - label - spacing - rightmeta.
	maxTitleWidth := max(
		// -2 for spaces around label.
		r.width-1-S2-len(label)-2-S2-S3-20, 10)

	title = midEllipsize(title, maxTitleWidth)
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

	renderers := map[BlockType]func(*Block) (string, error){
		BlockTypeExecute:    r.renderTranscript,
		BlockTypeNotice:     r.renderTranscript,
		BlockTypeRead:       r.renderCode,
		BlockTypeApplyPatch: r.renderDiff,
		BlockTypePlan:       r.renderList,
		BlockTypeSummary:    r.renderList,
		BlockTypeTesting:    r.renderList,
		BlockTypeGrep:       r.renderCode,
		BlockTypeError:      r.renderError,
		BlockTypeTool:       r.renderToolBody,
	}

	if renderer, exists := renderers[b.Type]; exists {
		return renderer(b)
	}
	// Fallback: plain text.
	return r.renderTranscript(b)
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
	lineCount := len(lines)
	gutterWidth := calcGutterWidth(lineCount)

	var out strings.Builder

	for i, line := range lines {
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

	return out.String(), nil
}

// renderDiff renders unified diff (for APPLY_PATCH).
func (r *Renderer) renderDiff(b *Block) (string, error) {
	lines := strings.Split(b.Body, "\n")

	var out strings.Builder

	for _, line := range lines {
		out.WriteString(strings.Repeat(" ", S2))

		switch {
		case strings.HasPrefix(line, "@@"):
			// Hunk header.
			out.WriteString(string(ColorBorder))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		case strings.HasPrefix(line, "+"):
			// Added line.
			out.WriteString(string(ColorGreen))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		case strings.HasPrefix(line, "-"):
			// Removed line.
			out.WriteString(string(ColorRed))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		default:
			// Context line.
			out.WriteString(string(ColorMuted))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		}

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

// midEllipsize truncates a string with middle ellipsis (60/40 split).
// Example: "very long filename.go" → "very long…name.go".
func midEllipsize(s string, maxWidth int) string {
	graphemes := extractGraphemes(s)
	totalWidth := calculateTotalWidth(graphemes)

	if totalWidth <= maxWidth {
		return s
	}

	leftWidth, rightWidth := calculateSplitWidths(maxWidth)
	left := buildLeftPart(graphemes, leftWidth)
	right := buildRightPart(graphemes, rightWidth)

	return strings.Join(left, "") + "…" + strings.Join(right, "")
}

// extractGraphemes extracts graphemes from a string.
func extractGraphemes(s string) []string {
	gr := uniseg.NewGraphemes(s)

	var graphemes []string
	for gr.Next() {
		graphemes = append(graphemes, gr.Str())
	}

	return graphemes
}

// calculateTotalWidth calculates the total width of graphemes.
func calculateTotalWidth(graphemes []string) int {
	totalWidth := 0
	for _, g := range graphemes {
		totalWidth += uniseg.StringWidth(g)
	}

	return totalWidth
}

// calculateSplitWidths calculates left and right widths for splitting.
func calculateSplitWidths(maxWidth int) (leftWidth, rightWidth int) {
	leftWidth = int(float64(maxWidth-1) * ellipsisSplitLeft) // -1 for ellipsis.
	rightWidth = maxWidth - leftWidth - 1

	return leftWidth, rightWidth
}

// buildLeftPart builds the left part of the ellipsized string.
func buildLeftPart(graphemes []string, leftWidth int) []string {
	var left []string

	currentWidth := 0

	for _, g := range graphemes {
		w := uniseg.StringWidth(g)
		if currentWidth+w > leftWidth {
			break
		}

		left = append(left, g)
		currentWidth += w
	}

	return left
}

// buildRightPart builds the right part of the ellipsized string.
func buildRightPart(graphemes []string, rightWidth int) []string {
	var right []string

	currentWidth := 0

	for i := len(graphemes) - 1; i >= 0; i-- {
		g := graphemes[i]

		w := uniseg.StringWidth(g)
		if currentWidth+w > rightWidth {
			break
		}

		right = append([]string{g}, right...)
		currentWidth += w
	}

	return right
}

// calcGutterWidth calculates the gutter width for line numbers.
// Returns 3-6 characters based on line count.
func calcGutterWidth(lineCount int) int {
	switch {
	case lineCount < gutterThreshold10:
		return gutterWidth3
	case lineCount < gutterThreshold100:
		return gutterWidth4
	case lineCount < gutterThreshold1000:
		return gutterWidth5
	default:
		return gutterWidth6
	}
}

// getBlockTypeColor returns the 256-color background color for a block type badge.
func (r *Renderer) getBlockTypeColor(blockType BlockType) int {
	colorMap := map[BlockType]int{
		BlockTypeExecute:    colorBlue256,
		BlockTypeRead:       colorOrange256,
		BlockTypeGrep:       colorYellow256,
		BlockTypeApplyPatch: colorMagenta256,
		BlockTypePlan:       colorPurple256,
		BlockTypeSummary:    colorCyan256,
		BlockTypeTesting:    colorGreen256,
		BlockTypeNotice:     colorGray256,
		BlockTypeError:      colorRed256,
	}

	if color, exists := colorMap[blockType]; exists {
		return color
	}

	return colorGray256 // fallback.
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

	renderers := map[BlockType]func(*Block) string{
		BlockTypeExecute:    r.renderExecuteCompletionStatus,
		BlockTypeRead:       r.renderReadCompletionStatus,
		BlockTypeApplyPatch: r.renderWriteCompletionStatus,
		BlockTypeGrep:       r.renderGrepCompletionStatus,
		BlockTypeTool:       r.renderToolCompletionStatus,
	}

	if renderer, exists := renderers[b.Type]; exists {
		return renderer(b)
	}

	return ""
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

// renderReadCompletionStatus renders completion status for READ blocks.
func (r *Renderer) renderReadCompletionStatus(_ *Block) string {
	// Read blocks typically don't show completion status
	// (the body contains the file content).
	return ""
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

// renderGrepCompletionStatus renders completion status for GREP blocks.
func (r *Renderer) renderGrepCompletionStatus(_ *Block) string {
	// Grep blocks typically don't show completion status
	// (the body contains the matches).
	return ""
}

// renderToolBody renders the body content for TOOL blocks.
func (r *Renderer) renderToolBody(b *Block) (string, error) {
	// Tool blocks typically show their raw output/result.
	return r.renderTranscript(b)
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
