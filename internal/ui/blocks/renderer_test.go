package blocks

import (
	"strings"
	"testing"
)

// Test that WRITE block does not render failure/success before completion
func TestWriteRender_BeforeCompletion_NoStatusOrFooter(t *testing.T) {
	r := NewRenderer(80)

	b := NewBlock(BlockTypeApplyPatch)
	b.Title = "file.txt"

	meta := &PatchMeta{File: "file.txt", Succeeded: false, Completed: false}
	if err := SetPatchMeta(b, meta); err != nil {
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

// Test that after successful completion, WRITE block shows success
func TestWriteRender_AfterSuccess_ShowsSuccess(t *testing.T) {
	r := NewRenderer(80)

	b := NewBlock(BlockTypeApplyPatch)
	b.Title = "file.txt"

	meta := &PatchMeta{File: "file.txt", Succeeded: true, Completed: true}
	if err := SetPatchMeta(b, meta); err != nil {
		t.Fatalf("set meta: %v", err)
	}

	out, err := r.Render(b)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(out, "File written successfully.") {
		t.Errorf("expected success status line in output: %s", out)
	}
	if !strings.Contains(out, "✓ Succeeded") {
		t.Errorf("expected success footer chip in output: %s", out)
	}
	if strings.Contains(out, "● Failed") {
		t.Errorf("should not show failed footer chip on success")
	}
}

// Test that after failed completion, WRITE block shows failure
func TestWriteRender_AfterFailure_ShowsFailure(t *testing.T) {
	r := NewRenderer(80)

	b := NewBlock(BlockTypeApplyPatch)
	b.Title = "file.txt"

	meta := &PatchMeta{File: "file.txt", Succeeded: false, Completed: true, ErrorMsg: "boom"}
	if err := SetPatchMeta(b, meta); err != nil {
		t.Fatalf("set meta: %v", err)
	}

	out, err := r.Render(b)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(out, "Failed to write file.") {
		t.Errorf("expected failure status line in output: %s", out)
	}
	if !strings.Contains(out, "● Failed") {
		t.Errorf("expected failed footer chip in output: %s", out)
	}
}
