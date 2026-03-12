package blocks

import (
	"strings"
	"testing"
)

// Test that WRITE block does not render failure/success before completion.
func TestWriteRender_BeforeCompletion_NoStatusOrFooter(t *testing.T) {
	t.Parallel()
	r := NewRenderer(80)

	b := NewBlock(BlockTypeApplyPatch)
	b.Title = "file.txt"

	meta := &PatchMeta{File: "file.txt", Succeeded: false, Completed: false}
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
	b.Title = "file.txt"

	meta := &PatchMeta{File: "file.txt", Succeeded: true, Completed: true}
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
	b.Title = "file.txt"

	meta := &PatchMeta{File: "file.txt", Succeeded: false, Completed: true, ErrorMsg: "boom"}
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

// stripANSI removes ANSI escape codes from a string for testing purposes.
func stripANSI(s string) string {
	// Simple ANSI stripper (strips \x1b[...m sequences).
	result := ""
	inEscape := false

	var resultSb108 strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // Skip '['.

			continue
		}

		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}

			continue
		}

		resultSb108.WriteString(string(s[i]))
	}
	result += resultSb108.String()

	return result
}
