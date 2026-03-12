package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPatchTool_AddFile(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewApplyPatchTool(tmpDir)

	patch := `*** Begin Patch
*** Add File: test.txt
+Hello, World!
+This is a test file.
*** End Patch`

	params, _ := FromMap(map[string]any{
		"patch_text": patch,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify file was created.
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}

	expected := "Hello, World!\nThis is a test file."
	if string(content) != expected {
		t.Errorf("expected content %q, got %q", expected, string(content))
	}

	// Verify output mentions the file.
	if !strings.Contains(result.Output, "test.txt") {
		t.Errorf("expected output to mention test.txt, got: %s", result.Output)
	}
}

func TestApplyPatchTool_DeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "delete_me.txt")

	// Create file to delete.
	err := os.WriteFile(testFile, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewApplyPatchTool(tmpDir)

	patch := `*** Begin Patch
*** Delete File: delete_me.txt
*** End Patch`

	params, _ := FromMap(map[string]any{
		"patch_text": patch,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify file was deleted.
	_, err = os.Stat(testFile)
	if !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted")
	}
}

func TestApplyPatchTool_UpdateFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "update_me.txt")

	// Create file to update.
	original := "line1\nline2\nline3\n"
	err := os.WriteFile(testFile, []byte(original), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewApplyPatchTool(tmpDir)

	patch := `*** Begin Patch
*** Update File: update_me.txt
@@
 line1
-line2
+line2-modified
 line3
*** End Patch`

	params, _ := FromMap(map[string]any{
		"patch_text": patch,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify file was updated.
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	if !strings.Contains(string(content), "line2-modified") {
		t.Errorf("expected file to contain updated content, got: %s", string(content))
	}

	if strings.Contains(string(content), "line2\n") && !strings.Contains(string(content), "line2-modified") {
		t.Errorf("expected old content to be removed")
	}
}

func TestApplyPatchTool_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewApplyPatchTool(tmpDir)

	patch := `*** Begin Patch
*** Add File: dry_run_test.txt
+This file should not be created
*** End Patch`

	params, _ := FromMap(map[string]any{
		"patch_text": patch,
		"dry_run":    true,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify file was NOT created.
	_, err = os.Stat(filepath.Join(tmpDir, "dry_run_test.txt"))
	if !os.IsNotExist(err) {
		t.Errorf("expected file not to be created in dry-run mode")
	}

	// Verify output indicates dry-run.
	if !strings.Contains(strings.ToLower(result.Output), "dry run") {
		t.Errorf("expected output to indicate dry-run mode, got: %s", result.Output)
	}
}

func TestApplyPatchTool_ParseError(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewApplyPatchTool(tmpDir)

	// Invalid patch - missing End Patch marker.
	patch := `*** Begin Patch
*** Add File: test.txt
+content`

	params, _ := FromMap(map[string]any{
		"patch_text": patch,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Errorf("expected failure for invalid patch")
	}

	// Verify error message is clear.
	if !strings.Contains(result.Error, "End Patch") {
		t.Errorf("expected parse error mentioning 'End Patch', got: %s", result.Error)
	}
}

func TestApplyPatchTool_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewApplyPatchTool(tmpDir)

	// Attempt path traversal.
	patch := `*** Begin Patch
*** Add File: ../../etc/passwd
+malicious content
*** End Patch`

	params, _ := FromMap(map[string]any{
		"patch_text": patch,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Errorf("expected failure for path traversal attempt")
	}

	// Verify error message mentions path validation.
	if !strings.Contains(result.Error, "path") {
		t.Errorf("expected error about invalid path, got: %s", result.Error)
	}
}

func TestApplyPatchTool_MissingParameters(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewApplyPatchTool(tmpDir)

	tests := []struct {
		name       string
		params     map[string]any
		wantErrMsg string
	}{
		{
			name:       "missing patch_text",
			params:     map[string]any{},
			wantErrMsg: "patch_text parameter must be a non-empty string",
		},
		{
			name:       "empty patch_text",
			params:     map[string]any{"patch_text": ""},
			wantErrMsg: "patch_text parameter must be a non-empty string",
		},
		{
			name:       "non-string patch_text",
			params:     map[string]any{"patch_text": 123},
			wantErrMsg: "patch_text parameter must be a non-empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, _ := FromMap(tt.params)

			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result.Success {
				t.Errorf("expected failure")
			}

			if !strings.Contains(result.Error, tt.wantErrMsg) {
				t.Errorf("expected error containing %q, got: %s", tt.wantErrMsg, result.Error)
			}
		})
	}
}

func TestApplyPatchTool_MultipleOperations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file to delete.
	deleteFile := filepath.Join(tmpDir, "old.txt")
	err := os.WriteFile(deleteFile, []byte("old content"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewApplyPatchTool(tmpDir)

	patch := `*** Begin Patch
*** Add File: new.txt
+New file content
*** Delete File: old.txt
*** End Patch`

	params, _ := FromMap(map[string]any{
		"patch_text": patch,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify new file created.
	_, err = os.Stat(filepath.Join(tmpDir, "new.txt"))
	if err != nil {
		t.Errorf("expected new.txt to be created: %v", err)
	}

	// Verify old file deleted.
	_, err = os.Stat(deleteFile)
	if !os.IsNotExist(err) {
		t.Errorf("expected old.txt to be deleted")
	}

	// Verify output mentions both operations.
	if !strings.Contains(result.Output, "new.txt") || !strings.Contains(result.Output, "old.txt") {
		t.Errorf("expected output to mention both files, got: %s", result.Output)
	}
}

func TestApplyPatchTool_CustomWorkspace(t *testing.T) {
	// Create two directories.
	workspace1 := t.TempDir()
	workspace2 := t.TempDir()

	// Tool with workspace1 as default.
	tool := NewApplyPatchTool(workspace1)

	patch := `*** Begin Patch
*** Add File: test.txt
+content
*** End Patch`

	// Apply patch to workspace2.
	params, _ := FromMap(map[string]any{
		"patch_text":     patch,
		"workspace_root": workspace2,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify file created in workspace2, not workspace1.
	_, err = os.Stat(filepath.Join(workspace2, "test.txt"))
	if err != nil {
		t.Errorf("expected file in workspace2: %v", err)
	}

	_, err = os.Stat(filepath.Join(workspace1, "test.txt"))
	if !os.IsNotExist(err) {
		t.Errorf("did not expect file in workspace1")
	}
}

func TestApplyPatchTool_Schema(t *testing.T) {
	tool := NewApplyPatchTool("/tmp")
	schema := tool.Schema()

	if schema.Function.Name != "apply_patch" {
		t.Errorf("expected name 'apply_patch', got: %s", schema.Function.Name)
	}

	if schema.Function.Description == "" {
		t.Errorf("expected non-empty description")
	}

	// Verify required parameters.
	required := schema.Function.Parameters.Required
	if len(required) != 1 || required[0] != "patch_text" {
		t.Errorf("expected required parameter 'patch_text', got: %v", required)
	}

	// Verify all expected parameters are defined.
	props := schema.Function.Parameters.Properties

	expectedParams := []string{"patch_text", "workspace_root", "dry_run", "force"}
	for _, param := range expectedParams {
		if _, exists := props[param]; !exists {
			t.Errorf("expected parameter %q to be defined", param)
		}
	}
}

func TestApplyPatchTool_ErrorCases(t *testing.T) {
	tool := NewApplyPatchTool("/tmp/test")

	tests := []struct {
		name   string
		params map[string]any
	}{
		{
			name:   "missing patch_text",
			params: map[string]any{},
		},
		{
			name:   "invalid patch_text type",
			params: map[string]any{"patch_text": 123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, _ := FromMap(tt.params)

			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result.Success {
				t.Error("expected failure result")
			}
		})
	}
}

func TestApplyPatchTool_CheckApproval(t *testing.T) {
	tool := NewApplyPatchTool("/tmp")

	params, _ := FromMap(map[string]any{
		"patch_text": "*** a/file.go\n--- b/file.go\n@@ -1,1 +1,2 @@\n package main\n+// comment\n",
	})

	needs := tool.CheckApproval(params)

	if !needs.Required {
		t.Error("CheckApproval should require approval for patch operations")
	}

	if needs.Risk != RiskHigh {
		t.Errorf("CheckApproval Risk = %v, want %v", needs.Risk, RiskHigh)
	}

	if needs.Reason == "" {
		t.Error("CheckApproval should provide a reason")
	}
}

func TestApplyPatchTool_CheckApproval_EmptyPatch(t *testing.T) {
	tool := NewApplyPatchTool("/tmp")

	params, _ := FromMap(map[string]any{
		"patch_text": "",
	})

	needs := tool.CheckApproval(params)

	if needs.Required {
		t.Error("CheckApproval should not require approval for empty patch")
	}

	if needs.Risk != RiskSafe {
		t.Errorf("CheckApproval Risk = %v, want %v", needs.Risk, RiskSafe)
	}
}
