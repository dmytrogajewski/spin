package blocks

import (
	"strings"
	"testing"
)

// Journey: specs/bugs/BUG-tui-visual-polish.md.
func TestFormatActivity_Read(t *testing.T) {
	t.Parallel()

	b := NewBlock(BlockTypeRead)
	if err := SetReadMeta(b, &ReadMeta{File: "internal/ui/prompt/renderer.go"}); err != nil {
		t.Fatalf("set meta: %v", err)
	}

	got := stripANSI(FormatActivity(b))
	if got != "Read internal/ui/prompt/renderer.go" {
		t.Fatalf("FormatActivity() = %q", got)
	}
}

func TestFormatActivity_Grep(t *testing.T) {
	t.Parallel()

	b := NewBlock(BlockTypeGrep)
	b.Title = "internal/tools"

	if err := SetGrepMeta(b, &GrepMeta{Pattern: `func New|Name() string`, Mode: "content"}); err != nil {
		t.Fatalf("set meta: %v", err)
	}

	got := stripANSI(FormatActivity(b))
	want := `Grepped "func New|Name() string" in internal/tools`

	if got != want {
		t.Fatalf("FormatActivity() = %q, want %q", got, want)
	}
}

func TestFormatActivity_GrepWithoutPath(t *testing.T) {
	t.Parallel()

	b := NewBlock(BlockTypeGrep)
	if err := SetGrepMeta(b, &GrepMeta{Pattern: "foo", Mode: "files_with_matches"}); err != nil {
		t.Fatalf("set meta: %v", err)
	}

	got := stripANSI(FormatActivity(b))
	if got != `Grepped "foo"` {
		t.Fatalf("FormatActivity() = %q", got)
	}
}

func TestFormatActivity_Edited(t *testing.T) {
	t.Parallel()

	added := 621
	b := NewBlock(BlockTypeApplyPatch)

	if err := SetPatchMeta(b, &PatchMeta{File: "ROADMAP.md", LinesAdded: &added}); err != nil {
		t.Fatalf("set meta: %v", err)
	}

	got := stripANSI(FormatActivity(b))
	if got != "Edited ROADMAP.md +621" {
		t.Fatalf("FormatActivity() = %q", got)
	}

	if !strings.Contains(FormatActivity(b), string(ColorGreen)) {
		t.Fatal("added-line count should be green")
	}

	if !strings.Contains(FormatActivity(b), string(ColorCyan)) {
		t.Fatal("path should be cyan")
	}
}

func TestFormatActivity_ExecuteEmpty(t *testing.T) {
	t.Parallel()

	b := NewBlock(BlockTypeExecute)
	if got := FormatActivity(b); got != "" {
		t.Fatalf("execute should keep the badge header, got %q", got)
	}
}
