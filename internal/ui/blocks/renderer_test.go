package blocks

import (
	"regexp"
	"strings"
	"testing"
)

const (
	testFileName = "file.txt"
)

// Journey: specs/bugs/BUG-tui-blocks-and-thinking.md.
func TestRenderHeader_AccentBarOnBadgeLine(t *testing.T) {
	t.Parallel()

	r := NewRenderer(80)
	b := NewBlock(BlockTypeApplyPatch)
	b.Title = testFileName

	header := r.RenderHeader(b)
	if !strings.Contains(header, AccentBarGlyph) {
		t.Fatalf("accent glyph %q missing from header %q", AccentBarGlyph, header)
	}

	plain := strings.TrimLeft(stripANSI(header), "\n")
	firstLine, _, _ := strings.Cut(plain, "\n")

	if !strings.Contains(firstLine, AccentBarGlyph) {
		t.Fatalf("accent glyph %q missing from badge line %q (raw %q)", AccentBarGlyph, firstLine, header)
	}

	if !strings.Contains(firstLine, "WRITE") {
		t.Fatalf("WRITE badge not on same line as accent: %q", firstLine)
	}

	if strings.HasPrefix(plain, "\n") || strings.Contains(plain, "\n\n") {
		t.Fatalf("header should not insert a blank line before the badge: %q", plain)
	}
}

func TestRender_NoColorOnlyBlankLineBeforeBadge(t *testing.T) {
	t.Parallel()

	r := NewRenderer(80)
	b := NewBlock(BlockTypeApplyPatch)
	b.Title = testFileName

	out, err := r.Render(b)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	plain := strings.TrimLeft(stripANSI(out), "\n")
	firstLine, _, _ := strings.Cut(plain, "\n")

	if !strings.Contains(firstLine, AccentBarGlyph) || !strings.Contains(firstLine, "WRITE") {
		t.Fatalf("first visible line should be accent+WRITE, got %q", firstLine)
	}
}

// Test that WRITE block does not render failure/success before completion.
func TestWriteRender_BeforeCompletion_NoStatusOrFooter(t *testing.T) {
	t.Parallel()

	r := NewRenderer(80)

	b := NewBlock(BlockTypeApplyPatch)
	b.Title = testFileName

	meta := &PatchMeta{File: testFileName, Succeeded: false, Completed: false}

	err := SetPatchMeta(b, meta)
	if err != nil {
		t.Fatalf("set meta: %v", err)
	}

	out, err := r.Render(b)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if strings.Contains(out, "Failed to write file.") {
		t.Errorf("should not show failure status before completion")
	}

	if strings.Contains(out, "File written successfully.") {
		t.Errorf("should not show success status before completion")
	}

	if strings.Contains(out, "● Failed") {
		t.Errorf("should not show failed footer chip before completion")
	}

	if strings.Contains(out, "✓ Succeeded") {
		t.Errorf("should not show success footer chip before completion")
	}
}

// Test that after successful completion, WRITE block shows success.
func TestWriteRender_AfterSuccess_ShowsSuccess(t *testing.T) {
	t.Parallel()

	r := NewRenderer(80)

	b := NewBlock(BlockTypeApplyPatch)
	b.Title = testFileName

	meta := &PatchMeta{File: testFileName, Succeeded: true, Completed: true}

	err := SetPatchMeta(b, meta)
	if err != nil {
		t.Fatalf("set meta: %v", err)
	}

	out, err := r.Render(b)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	plainOut := stripANSI(out)

	if !strings.Contains(plainOut, "File written successfully.") {
		t.Errorf("expected success status line in output: %s", out)
	}

	if !strings.Contains(plainOut, "Succeeded. File edited.") {
		t.Errorf("expected success footer chip in output: %s", out)
	}

	if strings.Contains(plainOut, "Failed") {
		t.Errorf("should not show failed footer chip on success")
	}
}

// Test that after failed completion, WRITE block shows failure.
func TestWriteRender_AfterFailure_ShowsFailure(t *testing.T) {
	t.Parallel()

	r := NewRenderer(80)

	b := NewBlock(BlockTypeApplyPatch)
	b.Title = testFileName

	meta := &PatchMeta{File: testFileName, Succeeded: false, Completed: true, ErrorMsg: "boom"}

	err := SetPatchMeta(b, meta)
	if err != nil {
		t.Fatalf("set meta: %v", err)
	}

	out, err := r.Render(b)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	plainOut := stripANSI(out)

	if !strings.Contains(plainOut, "Failed to write file.") {
		t.Errorf("expected failure status line in output: %s", out)
	}

	if !strings.Contains(plainOut, "Failed: boom") {
		t.Errorf("expected failed footer chip in output: %s", out)
	}
}

var ansiCSI = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI SGR sequences without splitting UTF-8 runes.
func stripANSI(s string) string {
	return ansiCSI.ReplaceAllString(s, "")
}
