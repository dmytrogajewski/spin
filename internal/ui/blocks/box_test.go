package blocks

import (
	"strings"
	"testing"
)

// Journey: specs/bugs/BUG-tui-visual-polish.md.
func TestRenderDiff_BoxesAndTruncates(t *testing.T) {
	t.Parallel()

	r := NewRenderer(40)
	b := NewBlock(BlockTypeApplyPatch)
	b.FoldState = FoldStateExpanded

	var body strings.Builder
	for range maxPreviewLines + 5 {
		body.WriteString("+line\n")
	}

	b.Body = body.String()

	out, err := r.RenderBody(b)
	if err != nil {
		t.Fatalf("RenderBody: %v", err)
	}

	if !strings.Contains(out, ColorDiffBoxBg) {
		t.Fatalf("diff missing green box background, got %q", out)
	}

	plain := stripANSI(out)
	if !strings.Contains(plain, "... truncated (5 more lines)") {
		t.Fatalf("missing truncation footer, got %q", plain)
	}

	if strings.Contains(plain, "ctrl+r") {
		t.Fatal("must not advertise ctrl+r; that key is not bound")
	}
}

func TestRender_WriteUsesActivityLine(t *testing.T) {
	t.Parallel()

	r := NewRenderer(80)
	b := NewBlock(BlockTypeApplyPatch)
	b.Title = testFileName

	if err := SetPatchMeta(b, &PatchMeta{File: testFileName}); err != nil {
		t.Fatalf("set meta: %v", err)
	}

	out, err := r.Render(b)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	plain := strings.TrimLeft(stripANSI(out), "\n")
	firstLine, _, _ := strings.Cut(plain, "\n")

	if !strings.Contains(firstLine, "Edited") || !strings.Contains(firstLine, testFileName) {
		t.Fatalf("first visible line should be activity, got %q", firstLine)
	}

	if strings.Contains(firstLine, "WRITE") {
		t.Fatalf("activity line should replace the WRITE badge, got %q", firstLine)
	}
}
