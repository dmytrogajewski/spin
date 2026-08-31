package overlay

// Journey: specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md.

import (
	"strings"
	"testing"
)

func TestHookVetoDialog_RenderIncludesReason(t *testing.T) {
	t.Parallel()

	got := NewHookVetoDialog("veto").Render()
	if !strings.Contains(got, "veto") {
		t.Fatalf("overlay %q must contain veto reason", got)
	}
}
