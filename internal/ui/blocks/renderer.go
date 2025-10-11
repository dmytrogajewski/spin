package blocks

import (
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/ui/theme"
	"github.com/rivo/uniseg"
)

// Renderer renders blocks to ANSI terminal output.
type Renderer struct {
	width int          // Terminal width in columns
	theme theme.Theme  // Color theme (optional, uses legacy colors if nil)
}

// NewRenderer creates a new block renderer with the given terminal width.
// Uses legacy hardcoded colors for backward compatibility.
func NewRenderer(width int) *Renderer {
	if width <= 0 {
		width = 80 // Default width
	}
	return &Renderer{
		width: width,
		theme: nil, // nil theme uses legacy colors
	}
}

// NewRendererWithTheme creates a new block renderer with a theme.
func NewRendererWithTheme(width int, th theme.Theme) *Renderer {
	if width <= 0 {
		width = 80 // Default width
	}
	return &Renderer{
		width: width,
		theme: th,
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
		return "", fmt.Errorf("block is nil")
	}

	var out strings.Builder

	// Render header
	header := r.RenderHeader(b)
	out.WriteString(header)
	out.WriteString("\n")

	// Render body (if expanded)
	if b.FoldState == FoldStateExpanded && b.Body != "" {
		body, err := r.RenderBody(b)
		if err != nil {
			return "", fmt.Errorf("render body: %w", err)
		}
		out.WriteString(body)
	} else if b.FoldState == FoldStateCollapsed {
		// Show collapsed badge
		out.WriteString(strings.Repeat(" ", S2))
		out.WriteString(string(ColorMuted))
		out.WriteString("⟦ collapsed ⟧")
		out.WriteString(string(ColorReset))
		out.WriteString("\n")
	}

	// Render footer
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

	// Accent bar (1ch)
	tagColor := GetTagColor(b.Type)
	out.WriteString(string(tagColor))
	out.WriteString("│")
	out.WriteString(string(ColorReset))

	// Spacing after accent bar
	out.WriteString(strings.Repeat(" ", S2))

	// Tag badge with colored background
	// Get background color and label for block type
	bgColor := r.getBlockTypeColor(b.Type)
	label := r.getBlockTypeLabel(b.Type)
	out.WriteString(fmt.Sprintf("\x1b[48;5;%dm", bgColor)) // Background color
	out.WriteString("\x1b[38;5;232m")                      // Black text for contrast
	out.WriteString(string(ColorBold))
	out.WriteString(fmt.Sprintf(" %s ", label))
	out.WriteString(string(ColorReset))

	// Spacing after tag
	out.WriteString(strings.Repeat(" ", S2))

	// Title/Meta (left-aligned)
	title := r.formatTitle(b)
	// Calculate available width: total - bar - spacing - label - spacing - rightmeta
	maxTitleWidth := r.width - 1 - S2 - len(label) - 2 - S2 - S3 - 20 // -2 for spaces around label
	if maxTitleWidth < 10 {
		maxTitleWidth = 10
	}
	title = midEllipsize(title, maxTitleWidth)
	out.WriteString(title)

	return out.String()
}

// formatTitle formats the title/meta line based on block type.
// Format: toolname argument (parameters: key "value", ...)
func (r *Renderer) formatTitle(b *Block) string {
	var parts []string

	// For tool blocks (EXECUTE), show: tool_name argument (parameters: ...)
	// For others, show title + metadata
	switch b.Type {
	case BlockTypeExecute:
		if meta, err := ParseExecuteMeta(b); err == nil {
			// Extract tool name and primary argument from command
			// Format: tool_name primary_arg (parameters: param1 "value1", param2 "value2")
			toolName := "execute" // Default
			primaryArg := ""
			params := []string{}

			// If title is set, use it as tool name
			if b.Title != "" {
				toolName = b.Title
			}

			// Try to extract primary argument (first non-option arg)
			// For now, just use command as primary arg
			primaryArg = meta.Command
			if meta.CWD != "." {
				params = append(params, fmt.Sprintf("cwd %q", meta.CWD))
			}

			// Build output
			parts = append(parts, string(ColorBold)+toolName+string(ColorReset))
			if primaryArg != "" {
				parts = append(parts, primaryArg)
			}
			if len(params) > 0 {
				paramsStr := strings.Join(params, ", ")
				parts = append(parts, string(ColorDim)+fmt.Sprintf("(parameters: %s)", paramsStr)+string(ColorReset))
			}
		}
	case BlockTypeRead:
		if meta, err := ParseReadMeta(b); err == nil {
			parts = append(parts, string(ColorBold)+"read_file"+string(ColorReset))
			parts = append(parts, meta.File)
			parts = append(parts, string(ColorDim)+"(parameters: path "+fmt.Sprintf("%q", meta.File)+")"+string(ColorReset))
		}
	case BlockTypeGrep:
		if meta, err := ParseGrepMeta(b); err == nil {
			parts = append(parts, string(ColorBold)+"grep"+string(ColorReset))
			parts = append(parts, meta.Pattern)
			params := []string{fmt.Sprintf("pattern %q", meta.Pattern), fmt.Sprintf("mode %q", meta.Mode)}
			parts = append(parts, string(ColorDim)+fmt.Sprintf("(parameters: %s)", strings.Join(params, ", "))+string(ColorReset))
		}
	case BlockTypeApplyPatch:
		if meta, err := ParsePatchMeta(b); err == nil {
			parts = append(parts, string(ColorBold)+"apply_patch"+string(ColorReset))
			parts = append(parts, meta.File)
			parts = append(parts, string(ColorDim)+fmt.Sprintf("(parameters: file %q)", meta.File)+string(ColorReset))
		}
	case BlockTypePlan:
		if b.Title != "" {
			parts = append(parts, string(ColorBold)+b.Title+string(ColorReset))
		}
		if meta, err := ParsePlanMeta(b); err == nil {
			metaStr := fmt.Sprintf("Updated: %d total (%d pending, %d in progress, %d completed)",
				meta.Total, meta.Pending, meta.InProgress, meta.Completed)
			parts = append(parts, metaStr)
		}
	default:
		// For other types, use title if available
		if b.Title != "" {
			parts = append(parts, string(ColorBold)+b.Title+string(ColorReset))
		}
	}

	return strings.Join(parts, " ")
}

// RenderBody renders only the block body based on type.
func (r *Renderer) RenderBody(b *Block) (string, error) {
	if b == nil || b.Body == "" {
		return "", nil
	}

	// Dispatch to type-specific renderer
	switch b.Type {
	case BlockTypeExecute, BlockTypeNotice:
		return r.renderTranscript(b)
	case BlockTypeRead:
		return r.renderCode(b)
	case BlockTypeApplyPatch:
		return r.renderDiff(b)
	case BlockTypePlan, BlockTypeSummary, BlockTypeTesting:
		return r.renderList(b)
	case BlockTypeGrep:
		return r.renderCode(b) // Similar to READ
	case BlockTypeError:
		return r.renderError(b)
	default:
		// Fallback: plain text
		return r.renderTranscript(b)
	}
}

// RenderFooter renders only the block footer.
func (r *Renderer) RenderFooter(b *Block) string {
	if b == nil {
		return ""
	}

	var chips []string

	// Type-specific footer chips
	switch b.Type {
	case BlockTypeExecute:
		if meta, err := ParseExecuteMeta(b); err == nil {
			if meta.ExitCode != nil {
				chips = append(chips, fmt.Sprintf("[exit: %d]", *meta.ExitCode))
			}
			if meta.LinesOut != nil {
				chips = append(chips, fmt.Sprintf("[out: %d lines]", *meta.LinesOut))
			}
			if meta.DurationMS != nil {
				dur := float64(*meta.DurationMS) / 1000.0
				chips = append(chips, fmt.Sprintf("[dur: %.1fs]", dur))
			}
		}
	case BlockTypeApplyPatch:
		if meta, err := ParsePatchMeta(b); err == nil {
			if meta.Succeeded {
				prefix := string(ColorGreen) + "✓" + string(ColorReset)
				msg := " Succeeded. File edited."
				if meta.LinesAdded != nil && *meta.LinesAdded > 0 {
					msg += fmt.Sprintf(" (+%d added)", *meta.LinesAdded)
				}
				chips = append(chips, prefix+msg)
			} else {
				prefix := string(ColorRed) + "●" + string(ColorReset)
				msg := " Failed"
				if meta.ErrorMsg != "" {
					msg += ": " + meta.ErrorMsg
				}
				chips = append(chips, prefix+msg)
			}
		}
	}

	if len(chips) == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString(strings.Repeat(" ", S2))
	out.WriteString(strings.Join(chips, " "))
	return out.String()
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

		// Gutter
		out.WriteString(strings.Repeat(" ", S2))
		out.WriteString(string(ColorMuted))
		out.WriteString(fmt.Sprintf("│%*d ", gutterWidth-1, lineNum))
		out.WriteString(string(ColorReset))

		// Line content
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

		if strings.HasPrefix(line, "@@") {
			// Hunk header
			out.WriteString(string(ColorBorder))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		} else if strings.HasPrefix(line, "+") {
			// Added line
			out.WriteString(string(ColorGreen))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		} else if strings.HasPrefix(line, "-") {
			// Removed line
			out.WriteString(string(ColorRed))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		} else {
			// Context line
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

		// Detect bullet type
		if strings.HasPrefix(line, "✓ ") || strings.HasPrefix(line, "- [x] ") {
			// Done item
			out.WriteString(string(ColorGreen))
			out.WriteString("✓")
			out.WriteString(string(ColorReset))
			out.WriteString(" ")
			text := strings.TrimPrefix(line, "✓ ")
			text = strings.TrimPrefix(text, "- [x] ")
			out.WriteString(string(ColorDim))
			out.WriteString(text)
			out.WriteString(string(ColorReset))
		} else if strings.HasPrefix(line, "◦ ") {
			// Skipped item
			out.WriteString(string(ColorMuted))
			out.WriteString("◦")
			out.WriteString(string(ColorReset))
			out.WriteString(" ")
			text := strings.TrimPrefix(line, "◦ ")
			out.WriteString(string(ColorMuted))
			out.WriteString(text)
			out.WriteString(string(ColorReset))
		} else if strings.HasPrefix(line, "• ") || strings.HasPrefix(line, "- [ ] ") {
			// Pending item
			out.WriteString("•")
			out.WriteString(" ")
			text := strings.TrimPrefix(line, "• ")
			text = strings.TrimPrefix(text, "- [ ] ")
			out.WriteString(text)
		} else {
			// Plain text (paragraph)
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
			// First line is bold (error message)
			out.WriteString(string(ColorRed))
			out.WriteString(string(ColorBold))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		} else {
			// Subsequent lines are dim (stack trace)
			out.WriteString(string(ColorDim))
			out.WriteString(line)
			out.WriteString(string(ColorReset))
		}

		out.WriteString("\n")
	}

	return out.String(), nil
}

// midEllipsize truncates a string with middle ellipsis (60/40 split).
// Example: "very long filename.go" → "very long…name.go"
func midEllipsize(s string, maxWidth int) string {
	gr := uniseg.NewGraphemes(s)
	var graphemes []string
	for gr.Next() {
		graphemes = append(graphemes, gr.Str())
	}

	totalWidth := 0
	for _, g := range graphemes {
		totalWidth += uniseg.StringWidth(g)
	}

	if totalWidth <= maxWidth {
		return s
	}

	// Calculate split point (60/40)
	leftWidth := int(float64(maxWidth-1) * 0.6) // -1 for ellipsis
	rightWidth := maxWidth - leftWidth - 1

	var left, right []string
	currentWidth := 0

	// Build left part
	for _, g := range graphemes {
		w := uniseg.StringWidth(g)
		if currentWidth+w > leftWidth {
			break
		}
		left = append(left, g)
		currentWidth += w
	}

	// Build right part (from end)
	currentWidth = 0
	for i := len(graphemes) - 1; i >= 0; i-- {
		g := graphemes[i]
		w := uniseg.StringWidth(g)
		if currentWidth+w > rightWidth {
			break
		}
		right = append([]string{g}, right...)
		currentWidth += w
	}

	return strings.Join(left, "") + "…" + strings.Join(right, "")
}

// calcGutterWidth calculates the gutter width for line numbers.
// Returns 3-6 characters based on line count.
func calcGutterWidth(lineCount int) int {
	if lineCount < 10 {
		return 3
	} else if lineCount < 100 {
		return 4
	} else if lineCount < 1000 {
		return 5
	}
	return 6
}

// getBlockTypeColor returns the 256-color background color for a block type badge.
func (r *Renderer) getBlockTypeColor(blockType BlockType) int {
	switch blockType {
	case BlockTypeExecute:
		return 063 // Blue
	case BlockTypeRead:
		return 208 // Orange
	case BlockTypeGrep:
		return 220 // Yellow
	case BlockTypeApplyPatch:
		return 205 // Magenta
	case BlockTypePlan:
		return 141 // Purple
	case BlockTypeSummary:
		return 045 // Cyan
	case BlockTypeTesting:
		return 034 // Green
	case BlockTypeNotice:
		return 244 // Gray
	case BlockTypeError:
		return 196 // Red
	default:
		return 244 // Gray (fallback)
	}
}

// getBlockTypeLabel returns the display label for a block type.
// Some types use shorter/friendlier names (e.g., "TOOL" instead of "EXECUTE").
func (r *Renderer) getBlockTypeLabel(blockType BlockType) string {
	switch blockType {
	case BlockTypeExecute:
		return "TOOL"
	case BlockTypeApplyPatch:
		return "PATCH"
	default:
		return string(blockType)
	}
}
