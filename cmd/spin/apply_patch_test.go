package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/patchapply"
)

var (
	errInvalidOperation  = errors.New("invalid operation")
	errContentMismatch   = errors.New("content mismatch")
	errFileShouldBeDeleted = errors.New("file should be deleted")
	errContentMismatch2  = errors.New("content mismatch")
)


// TestReadPatchInput_Stdin tests reading patch from stdin.
func TestReadPatchInput_Stdin(t *testing.T) {
	// Save original stdin.
	oldStdin := os.Stdin

	defer func() { os.Stdin = oldStdin }()

	// Create a pipe to simulate stdin.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdin = r

	// Write test data.
	testPatch := "*** Begin Patch\n*** End Patch"

	go func() {
		_, _ = w.Write([]byte(testPatch))
		w.Close()
	}()

	// Test with empty patchFile (reads from stdin).
	result, err := readPatchInput("")
	if err != nil {
		t.Errorf("readPatchInput() error = %v", err)
	}

	if result != testPatch {
		t.Errorf("readPatchInput() = %q, want %q", result, testPatch)
	}
}

// TestReadPatchInput_File tests reading patch from file.
func TestReadPatchInput_File(t *testing.T) {
	t.Parallel()

	// Create temp file.
	tmpDir := t.TempDir()
	patchFile := filepath.Join(tmpDir, "test.patch")
	testContent := "*** Begin Patch\n*** End Patch"

	err := os.WriteFile(patchFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test.
	result, err := readPatchInput(patchFile)
	if err != nil {
		t.Errorf("readPatchInput() error = %v", err)
	}

	if result != testContent {
		t.Errorf("readPatchInput() = %q, want %q", result, testContent)
	}
}

// TestReadPatchInput_FileNotFound tests error on missing file.
func TestReadPatchInput_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := readPatchInput("/nonexistent/file.patch")
	if err == nil {
		t.Error("readPatchInput() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "open file") {
		t.Errorf("readPatchInput() error = %v, want 'open file' error", err)
	}
}

// TestFormatParseError tests parse error formatting.
func TestFormatParseError(t *testing.T) {
	t.Parallel()

	testErr := &patchapply.Error{
		Op:   "Parse",
		Path: "",
		Line: 5,
		Err:  errInvalidOperation,
	}

	result := formatParseError(testErr)
	resultStr := result.Error()

	// Check key elements.
	if !strings.Contains(resultStr, "invalid patch syntax") {
		t.Errorf("formatParseError() missing 'invalid patch syntax'")
	}

	if !strings.Contains(resultStr, "Hint") {
		t.Errorf("formatParseError() missing 'Hint'")
	}

	if !strings.Contains(resultStr, "Begin Patch") {
		t.Errorf("formatParseError() missing format example")
	}
}

// TestFormatApplyError tests apply error formatting.
func TestFormatApplyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		wantErr  bool
		contains []string
	}{
		{
			name: "structured error",
			err: &patchapply.Error{
				Op:   "Update",
				Path: "test.go",
				Line: 10,
				Err:  patchapply.ErrContextNotFound,
			},
			wantErr:  true,
			contains: []string{"failed to apply patch", "Update", "test.go", "Hint"},
		},
		{
			name:     "generic error",
			err:      patchapply.ErrEmptyWorkspace,
			wantErr:  true,
			contains: []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := formatApplyError(tt.err)
			if result == nil {
				if tt.wantErr {
					t.Error("formatApplyError() expected error, got nil")
				}

				return
			}

			resultStr := result.Error()
			for _, want := range tt.contains {
				if !strings.Contains(resultStr, want) {
					t.Errorf("formatApplyError() = %q, want to contain %q", resultStr, want)
				}
			}
		})
	}
}

// TestGetHintForError tests error hint generation.
func TestGetHintForError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *patchapply.Error
		contains string
	}{
		{
			name: "context not found",
			err: &patchapply.Error{
				Err: patchapply.ErrContextNotFound,
			},
			contains: "context may have changed",
		},
		{
			name: "path outside workspace",
			err: &patchapply.Error{
				Err: patchapply.ErrPathOutsideWorkspace,
			},
			contains: "relative paths",
		},
		{
			name: "file exists",
			err: &patchapply.Error{
				Err: patchapply.ErrFileExists,
			},
			contains: "--force",
		},
		{
			name: "file not found",
			err: &patchapply.Error{
				Err: patchapply.ErrFileNotFound,
			},
			contains: "file exists",
		},
		{
			name: "unknown error",
			err: &patchapply.Error{
				Err: &patchapply.Error{},
			},
			contains: "Check the error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hint := getHintForError(tt.err)
			if !strings.Contains(hint, tt.contains) {
				t.Errorf("getHintForError() = %q, want to contain %q", hint, tt.contains)
			}
		})
	}
}

// TestApplyPatch_Success tests successful patch application.
func TestApplyPatch_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create test file.
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("line 1\nline 2\nline 3\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create patch.
	patchText := `*** Begin Patch
*** Update File: test.txt
@@
 line 1
-line 2
+line 2 modified
 line 3
*** End Patch`

	patchFile := filepath.Join(tmpDir, "test.patch")
	err = os.WriteFile(patchFile, []byte(patchText), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Run command via cobra.
	cmd := newApplyPatchCmd()
	cmd.SetArgs([]string{"-f", patchFile, "-w", tmpDir})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("runApplyPatch() error = %v", err)
	}

	// Verify file was modified.
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := "line 1\nline 2 modified\nline 3\n"
	if string(content) != expected {
		t.Errorf("file content = %q, want %q", string(content), expected)
	}
}

// TestApplyPatch_DryRun tests dry-run mode.
func TestApplyPatch_DryRun(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create test file.
	testFile := filepath.Join(tmpDir, "test.txt")

	originalContent := "original content\n"
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create patch.
	patchText := `*** Begin Patch
*** Add File: new.txt
+new content
*** End Patch`

	patchFile := filepath.Join(tmpDir, "test.patch")
	err = os.WriteFile(patchFile, []byte(patchText), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Run command via cobra.
	cmd := newApplyPatchCmd()
	cmd.SetArgs([]string{"-f", patchFile, "-w", tmpDir, "--dry-run"})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("runApplyPatch() error = %v", err)
	}

	// Verify new file was NOT created.
	newFile := filepath.Join(tmpDir, "new.txt")
	_, err = os.Stat(newFile)
	if !os.IsNotExist(err) {
		t.Error("dry-run should not create files")
	}

	// Verify original file unchanged.
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != originalContent {
		t.Error("dry-run should not modify files")
	}
}

// TestApplyPatch_ParseError tests invalid patch syntax.
func TestApplyPatch_ParseError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create invalid patch.
	patchText := `*** Begin Patch
*** Invalid Operation: test.txt
*** End Patch`

	patchFile := filepath.Join(tmpDir, "invalid.patch")
	err := os.WriteFile(patchFile, []byte(patchText), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Run command via cobra (should fail).
	cmd := newApplyPatchCmd()
	cmd.SetArgs([]string{"-f", patchFile, "-w", tmpDir})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err = cmd.Execute()
	if err == nil {
		t.Error("runApplyPatch() expected error for invalid patch")
	}

	if !strings.Contains(err.Error(), "invalid patch syntax") {
		t.Errorf("runApplyPatch() error = %v, want 'invalid patch syntax'", err)
	}
}

// TestApplyPatch_PathTraversal tests path traversal rejection.
func TestApplyPatch_PathTraversal(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create patch with path traversal.
	patchText := `*** Begin Patch
*** Add File: ../../etc/passwd
+malicious content
*** End Patch`

	patchFile := filepath.Join(tmpDir, "malicious.patch")
	err := os.WriteFile(patchFile, []byte(patchText), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Run command via cobra (should fail).
	cmd := newApplyPatchCmd()
	cmd.SetArgs([]string{"-f", patchFile, "-w", tmpDir})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err = cmd.Execute()
	if err == nil {
		t.Error("runApplyPatch() should reject path traversal")
	}

	if !strings.Contains(err.Error(), "path") {
		t.Errorf("runApplyPatch() error = %v, want path-related error", err)
	}
}

// TestApplyPatch_ForceOverwrite tests force mode.
func TestApplyPatch_ForceOverwrite(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create existing file.
	testFile := filepath.Join(tmpDir, "existing.txt")
	err := os.WriteFile(testFile, []byte("existing content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create patch to overwrite.
	patchText := `*** Begin Patch
*** Add File: existing.txt
+new content
*** End Patch`

	patchFile := filepath.Join(tmpDir, "test.patch")
	err = os.WriteFile(patchFile, []byte(patchText), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("without force", func(t *testing.T) {
		t.Parallel()

		cmd := newApplyPatchCmd()
		cmd.SetArgs([]string{"-f", patchFile, "-w", tmpDir})
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true

		// Should fail.
		err = cmd.Execute()
		if err == nil {
			t.Error("runApplyPatch() should fail without --force")
		}
	})

	t.Run("with force", func(t *testing.T) {
		t.Parallel()

		// Use a separate tmpDir to avoid sharing filesystem state with "without force".
		forceDir := t.TempDir()
		forceFile := filepath.Join(forceDir, "existing.txt")
		forceErr := os.WriteFile(forceFile, []byte("existing content"), 0644)
		if forceErr != nil {
			t.Fatal(forceErr)
		}

		forcePatchFile := filepath.Join(forceDir, "test.patch")
		forceErr = os.WriteFile(forcePatchFile, []byte(patchText), 0644)
		if forceErr != nil {
			t.Fatal(forceErr)
		}

		cmd := newApplyPatchCmd()
		cmd.SetArgs([]string{"-f", forcePatchFile, "-w", forceDir, "--force"})
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true

		// Should succeed.
		forceErr = cmd.Execute()
		if forceErr != nil {
			t.Errorf("runApplyPatch() with --force error = %v", forceErr)
		}

		// Verify file was overwritten.
		content, readErr := os.ReadFile(forceFile)
		if readErr != nil {
			t.Fatal(readErr)
		}
		// Note: patch adds content as-is, no automatic trailing newline.
		if string(content) != "new content" {
			t.Errorf("file content = %q, want %q", string(content), "new content")
		}
	})
}

// TestPrintResults tests result output formatting.
func TestPrintResults(t *testing.T) {
	t.Parallel()

	result := &patchapply.ApplyResult{
		FilesCreated: []string{"new1.txt", "new2.txt"},
		FilesUpdated: []string{"updated.txt"},
		FilesDeleted: []string{"deleted.txt"},
		FilesMoved:   map[string]string{"old.txt": "new.txt"},
		DryRun:       false,
	}

	// Test with verbose=false. Just ensure it doesn't panic.
	printResults(result, false)

	// Test with verbose=true.
	printResults(result, true)
}

// TestRunDryRun tests dry-run output.
func TestRunDryRun(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create test files that the patch will operate on.
	err := os.WriteFile(filepath.Join(tmpDir, "old.txt"), []byte("old content\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "existing.txt"), []byte("existing content\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create applier.
	applier, err := patchapply.NewApplier(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	applier.SetDryRun(true)

	// Create test patch.
	patch := &patchapply.Patch{
		Operations: []patchapply.FileOperation{
			&patchapply.AddFile{
				FilePath: "new.txt",
				Lines:    []string{"line 1", "line 2"},
			},
			&patchapply.DeleteFile{
				FilePath: "old.txt",
			},
			&patchapply.UpdateFile{
				FilePath: "existing.txt",
				Hunks: []patchapply.Hunk{
					{
						Header: "",
						Changes: []patchapply.LineChange{
							{Type: patchapply.LineContext, Text: "existing content"},
							{Type: patchapply.LineInsert, Text: "new line"},
						},
					},
				},
			},
		},
	}

	// Run dry-run (should not error).
	err = runDryRun(applier, patch)
	if err != nil {
		t.Errorf("runDryRun() error = %v", err)
	}
}

// integrationTestCase defines a single integration test case for apply-patch.
type integrationTestCase struct {
	name       string
	setup      func(dir string) error
	patchText  string
	wantErr    bool
	errContain string
	verify     func(dir string) error
}

// getIntegrationTestCases returns all integration test cases.
func getIntegrationTestCases() []integrationTestCase {
	return []integrationTestCase{
		{
			name: "add new file",
			setup: func(_ string) error {
				return nil
			},
			patchText: "*** Begin Patch\n*** Add File: hello.txt\n+Hello World\n*** End Patch",
			wantErr:   false,
			verify: func(dir string) error {
				content, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
				if err != nil {
					return fmt.Errorf("reading hello.txt: %w", err)
				}

				if string(content) != "Hello World" {
					return errContentMismatch
				}

				return nil
			},
		},
		{
			name: "delete file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "delete-me.txt"), []byte("content"), 0644)
			},
			patchText: "*** Begin Patch\n*** Delete File: delete-me.txt\n*** End Patch",
			wantErr:   false,
			verify: func(dir string) error {
				_, err := os.Stat(filepath.Join(dir, "delete-me.txt"))
				if !os.IsNotExist(err) {
					return errFileShouldBeDeleted
				}

				return nil
			},
		},
		{
			name: "update file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "update.txt"),
					[]byte("line 1\nline 2\nline 3\n"), 0644)
			},
			patchText: "*** Begin Patch\n*** Update File: update.txt\n@@\n line 1\n-line 2\n+line 2 updated\n line 3\n*** End Patch",
			wantErr:   false,
			verify: func(dir string) error {
				content, err := os.ReadFile(filepath.Join(dir, "update.txt"))
				if err != nil {
					return fmt.Errorf("reading update.txt: %w", err)
				}

				if string(content) != "line 1\nline 2 updated\nline 3\n" {
					return errContentMismatch2
				}

				return nil
			},
		},
		{
			name: "context not found",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "mismatch.txt"),
					[]byte("different content\n"), 0644)
			},
			patchText:  "*** Begin Patch\n*** Update File: mismatch.txt\n@@\n expected line\n-old line\n+new line\n*** End Patch",
			wantErr:    true,
			errContain: "context",
		},
	}
}

// runIntegrationTestCase executes a single integration test case.
func runIntegrationTestCase(t *testing.T, tt integrationTestCase) {
	t.Helper()
	t.Parallel()

	tmpDir := t.TempDir()

	if tt.setup != nil {
		err := tt.setup(tmpDir)
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}
	}

	patchFile := filepath.Join(tmpDir, "test.patch")
	err := os.WriteFile(patchFile, []byte(tt.patchText), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd := newApplyPatchCmd()
	cmd.SetArgs([]string{"-f", patchFile, "-w", tmpDir})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err = cmd.Execute()

	if (err != nil) != tt.wantErr {
		t.Errorf("runApplyPatch() error = %v, wantErr %v", err, tt.wantErr)
	}

	if tt.wantErr && tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
		t.Errorf("runApplyPatch() error = %v, want to contain %q", err, tt.errContain)
	}

	if !tt.wantErr && tt.verify != nil {
		verifyErr := tt.verify(tmpDir)
		if verifyErr != nil {
			t.Errorf("verification failed: %v", verifyErr)
		}
	}
}

// TestApplyPatch_Integration tests full end-to-end scenarios.
func TestApplyPatch_Integration(t *testing.T) {
	t.Parallel()

	for _, tt := range getIntegrationTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			runIntegrationTestCase(t, tt)
		})
	}
}
