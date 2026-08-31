package commands

import "strings"

// Journey: specs/journeys/JOURNEY-013-compact-status-chip-and-operator-escape.md.

func writeCompactHelp(b *strings.Builder) {
	b.WriteString("\nTool output compact (default on):\n")
	b.WriteString("  Status bar shows on/off and last-turn output-bytes reduction (−N%).\n")
	b.WriteString("  Disable: SPIN_COMPACT=0 or config compact.enabled: false\n")
}
