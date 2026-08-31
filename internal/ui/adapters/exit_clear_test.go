package adapters

// Journey: specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

func TestWriteExitClear_UsesClearHome(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writeExitClear(&buf)

	if !strings.Contains(buf.String(), term.ClearHome) {
		t.Fatal("TUI teardown must still write ClearHome")
	}
}
