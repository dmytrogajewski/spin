package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/adapters"
	"github.com/dmytrogajewski/spin/internal/ui/testkit"
)

// Journey: specs/bugs/BUG-banner-scroll-ghost.md.

// TestPrintTUIWelcome_PurgesScrollback verifies startup wipes the terminal
// scrollback (ED3) so partial animation frames from previous runs cannot
// resurface when the user scrolls up.
func TestPrintTUIWelcome_PurgesScrollback(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	ui, err := adapters.NewPureTTY(&buf, adapters.WithTTY(testkit.NewFakeTTY(120, 38)))
	if err != nil {
		t.Fatalf("NewPureTTY failed: %v", err)
	}

	printTUIWelcome(ui, true)

	out := buf.String()

	purge := strings.Index(out, "\x1b[3J")
	if purge < 0 {
		t.Fatal("welcome must purge scrollback with ED3 (\\x1b[3J)")
	}

	banner := strings.Index(out, "▄")
	if banner < 0 {
		t.Fatal("welcome must render the mascot")
	}

	if purge > banner {
		t.Fatal("scrollback purge must happen before the banner is drawn")
	}
}

// Journey: specs/journeys/JOURNEY-013-compact-status-chip-and-operator-escape.md.
func TestCompactWelcomeLine_OffDoesNotClaimOn(t *testing.T) {
	t.Parallel()

	off := compactWelcomeLine(false)
	if strings.Contains(off, "compact: on") {
		t.Fatalf("disabled welcome %q must not claim compact: on", off)
	}

	if !strings.Contains(off, "compact: off") {
		t.Fatalf("disabled welcome %q want compact: off", off)
	}

	on := compactWelcomeLine(true)
	if !strings.Contains(on, "compact: on") {
		t.Fatalf("enabled welcome %q want compact: on", on)
	}
}

// Journey: specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md.
func TestPrintTUIWelcome_ListsHarnessCommands(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	ui, err := adapters.NewPureTTY(&buf, adapters.WithTTY(testkit.NewFakeTTY(120, 38)))
	if err != nil {
		t.Fatalf("NewPureTTY failed: %v", err)
	}

	printTUIWelcome(ui, true)

	out := buf.String()
	for _, cmd := range []string{"/skills", "/tasks", "/agents"} {
		if !strings.Contains(out, cmd) {
			t.Fatalf("welcome %q must list %s", out, cmd)
		}
	}

	if !strings.Contains(out, "Type / for commands and skills") {
		t.Fatalf("welcome must mention / suggestions, got %q", out)
	}
}
