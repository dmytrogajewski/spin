package blocks

import (
	"fmt"
	"strings"
)

func cyanPath(path string) string {
	return string(ColorCyan) + path + string(ColorReset)
}

// FormatActivity returns a one-line verb for read / grep / write blocks.
// Execute and other types return empty so Render keeps the badge header.
func FormatActivity(b *Block) string {
	if b == nil {
		return ""
	}

	switch b.Type {
	case BlockTypeRead:
		return formatReadActivity(b)
	case BlockTypeGrep:
		return formatGrepActivity(b)
	case BlockTypeApplyPatch:
		return formatEditActivity(b)
	default:
		return ""
	}
}

func formatReadActivity(b *Block) string {
	path := b.Title
	if meta, err := ParseReadMeta(b); err == nil && meta != nil && meta.File != "" {
		path = meta.File
	}

	if path == "" {
		return "Read"
	}

	return "Read " + cyanPath(path)
}

func formatGrepActivity(b *Block) string {
	pattern := ""
	if meta, err := ParseGrepMeta(b); err == nil && meta != nil {
		pattern = meta.Pattern
	}

	path := b.Title
	if path == "" {
		return fmt.Sprintf("Grepped %q", pattern)
	}

	return fmt.Sprintf("Grepped %q in %s", pattern, cyanPath(path))
}

func formatEditActivity(b *Block) string {
	file := b.Title
	added := 0

	if meta, err := ParsePatchMeta(b); err == nil && meta != nil {
		if meta.File != "" {
			file = meta.File
		}

		if meta.LinesAdded != nil {
			added = *meta.LinesAdded
		}
	}

	if added == 0 {
		added = countAddedLines(b.Body)
	}

	line := "Edited " + cyanPath(file)
	if added > 0 {
		line += " " + string(ColorGreen) + fmt.Sprintf("+%d", added) + string(ColorReset)
	}

	return line
}

func countAddedLines(body string) int {
	if body == "" {
		return 0
	}

	isDiff := looksLikeDiff(body)
	n := 0

	for line := range strings.SplitSeq(body, "\n") {
		if isDiff {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				n++
			}

			continue
		}

		if line != "" {
			n++
		}
	}

	return n
}

func looksLikeDiff(body string) bool {
	return strings.HasPrefix(body, "diff ") ||
		strings.HasPrefix(body, "@@") ||
		strings.HasPrefix(body, "+++") ||
		strings.HasPrefix(body, "---") ||
		strings.Contains(body, "\n@@") ||
		strings.Contains(body, "\n+")
}
