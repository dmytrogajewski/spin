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

	printTUIWelcome(ui)

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
