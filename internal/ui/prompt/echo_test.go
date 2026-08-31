package prompt

import (
	"strings"
	"testing"
)

// Journey: specs/bugs/BUG-tui-visual-polish.md.
func TestFormatUserEcho_UsesArrowAndCyan(t *testing.T) {
	t.Parallel()

	got := FormatUserEcho("hello")
	if !strings.Contains(got, DefaultPrefix+"hello") {
		t.Fatalf("echo missing %q prefix, got %q", DefaultPrefix, got)
	}

	if !strings.Contains(got, ColorUserEcho) {
		t.Fatalf("echo missing cyan, got %q", got)
	}

	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Fatalf("echo missing reset, got %q", got)
	}
}

func TestDefaultPrefix_IsArrow(t *testing.T) {
	t.Parallel()

	if DefaultPrefix != "→ " {
		t.Fatalf("DefaultPrefix = %q, want %q", DefaultPrefix, "→ ")
	}
}
