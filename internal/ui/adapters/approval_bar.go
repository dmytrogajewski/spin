package adapters

import (
	"os"
	"strings"

	"github.com/rivo/uniseg"

	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/pkg/ui/textwidth"
)

// Approval modes cycled with Shift+Tab.
const (
	// ApprovalModeAsk prompts for every command that needs approval (default).
	ApprovalModeAsk = "ask"
	// ApprovalModeYolo auto-approves every command without prompting.
	ApprovalModeYolo = "yolo"
)

// approvalModeCycle is the Shift+Tab cycling order.
var approvalModeCycle = []string{ApprovalModeAsk, ApprovalModeYolo}

const (
	barAccent = "\x1b[1;33m" // Bold yellow.
	barBold   = "\x1b[1m"
	barDim    = "\x1b[2m"
	barReset  = "\x1b[0m"

	barLabel   = "▌ approve? "
	barKeys    = "[a]once [s]session [g]always [d]eny [esc]"
	barGap     = "  "
	barWDLabel = "in "

	// minBarCmdWidth is the narrowest readable command excerpt; below it
	// the workdir segment is dropped to give the command more room.
	minBarCmdWidth = 16
	// maxBarWDWidth caps the workdir segment (left-truncated, tail kept).
	maxBarWDWidth = 32
	// minApprovalBarWidth is the narrowest terminal that fits the colored
	// bar; below it a plain mid-ellipsized status text is used instead.
	minApprovalBarWidth = 60
	// barMargin keeps the last cell free to avoid line wrap.
	barMargin = 1
)

// normalizeApprovalMode maps unknown or empty modes to the default.
func normalizeApprovalMode(mode string) string {
	for _, m := range approvalModeCycle {
		if m == mode {
			return m
		}
	}

	return ApprovalModeAsk
}

// nextApprovalMode returns the mode following cur in the cycle.
func nextApprovalMode(cur string) string {
	cur = normalizeApprovalMode(cur)
	for i, m := range approvalModeCycle {
		if m == cur {
			return approvalModeCycle[(i+1)%len(approvalModeCycle)]
		}
	}

	return ApprovalModeAsk
}

// buildApprovalBar renders a one-line colored approval prompt that fits
// within width terminal cells: accent label, command, optional workdir,
// and key hints. Segments are measured plain and colorized afterwards.
func buildApprovalBar(req safety.ApprovalRequest, width int) string {
	cmd := sanitizeApprovalCommand(req.Command.Raw)

	budget := width - barMargin -
		uniseg.StringWidth(barLabel) - uniseg.StringWidth(barKeys) - uniseg.StringWidth(barGap)

	wdSeg := approvalWorkDirSegment(req.WorkDir)

	// Include the workdir only when the command keeps a readable width.
	wdCost := uniseg.StringWidth(wdSeg) + uniseg.StringWidth(barGap)
	if wdSeg != "" && budget-wdCost >= minBarCmdWidth {
		budget -= wdCost
	} else {
		wdSeg = ""
	}

	cmd = textwidth.TruncateRight(cmd, budget)

	var b strings.Builder

	b.WriteString(barAccent + barLabel + barReset)
	b.WriteString(barBold + cmd + barReset)

	if wdSeg != "" {
		b.WriteString(barGap + barDim + wdSeg + barReset)
	}

	b.WriteString(barGap + barDim + barKeys + barReset)

	return b.String()
}

// compactApprovalText is the plain fallback for narrow terminals; the
// status renderer mid-ellipsizes it to the available width.
func compactApprovalText(req safety.ApprovalRequest) string {
	return "approve? " + sanitizeApprovalCommand(req.Command.Raw) + " [a/s/g/d/esc]"
}

// approvalWorkDirSegment formats the workdir hint, home-abbreviated and
// left-truncated so the informative tail survives.
func approvalWorkDirSegment(workDir string) string {
	if workDir == "" {
		return ""
	}

	wd := textwidth.TruncateLeft(abbreviateHomeDir(workDir), maxBarWDWidth)

	return barWDLabel + wd
}

// sanitizeApprovalCommand collapses all whitespace runs (including
// newlines) into single spaces so the command stays on one line.
func sanitizeApprovalCommand(raw string) string {
	return strings.Join(strings.Fields(raw), " ")
}

// abbreviateHomeDir replaces the user's home prefix with "~".
func abbreviateHomeDir(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}

	if path == home {
		return "~"
	}

	if strings.HasPrefix(path, home+string(os.PathSeparator)) {
		return "~" + path[len(home):]
	}

	return path
}
