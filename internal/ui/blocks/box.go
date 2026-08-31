package blocks

import (
	"fmt"
	"strings"

	"github.com/rivo/uniseg"

	"github.com/dmytrogajewski/spin/pkg/ui/textwidth"
)

const maxPreviewLines = 12

func truncatePreview(lines []string, maxLines int) (visible []string, hidden int) {
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if maxLines < 0 {
		maxLines = 0
	}

	if len(lines) <= maxLines {
		return lines, 0
	}

	return lines[:maxLines], len(lines) - maxLines
}

func truncationFooter(hidden int) string {
	return fmt.Sprintf("... truncated (%d more lines)", hidden)
}

func (r *Renderer) paintBoxLine(line string) string {
	plain := textwidth.StripANSI(line)
	w := uniseg.StringWidth(plain)

	if r.width > 0 && w > r.width {
		plain = textwidth.MidEllipsize(plain, r.width)
		w = uniseg.StringWidth(plain)
	}

	pad := 0
	if r.width > w {
		pad = r.width - w
	}

	return ColorDiffBoxBg + ColorDiffBoxFg + plain + strings.Repeat(" ", pad) + string(ColorReset)
}

func (r *Renderer) paintTruncationFooter(hidden int) string {
	return string(ColorMuted) + truncationFooter(hidden) + string(ColorReset)
}
