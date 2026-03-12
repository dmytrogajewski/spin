package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/patchapply"
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
		w.Write([]byte(testPatch))
		w.Close()
	}()

	// Test.
	result, err := readPatchInput()
	if err != nil {
		t.Errorf("readPatchInput() error = %v", err)
	}

	if result != testPatch {
		t.Errorf("readPatchInput() = %q, want %q", result, testPatch)
	}
}

// TestReadPatchInput_File tests reading patch from file.
func TestReadPatchInput_File(t *testing.T) {
	// Create temp file.
	tmpDir := t.TempDir()
	patchFile := filepath.Join(tmpDir, "test.patch")
	testContent := "*** Begin Patch\n*** End Patch"

	err := os.WriteFile(patchFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Set flag.
	applyPatchFile = patchFile

	defer func() { applyPatchFile = "" }()

	// Test.
	result, err := readPatchInput()
	if err != nil {
		t.Errorf("readPatchInput() error = %v", err)
	}

	if result != testContent {
		t.Errorf("readPatchInput() = %q, want %q", result, testContent)
	}
}

// TestReadPatchInput_FileNotFound tests error on missing file.
func TestReadPatchInput_FileNotFound(t *testing.T) {
	applyPatchFile = "/nonexistent/file.patch"

	defer func() { applyPatchFile = "" }()

	_, err := readPatchInput()
	if err == nil {
		t.Error("readPatchInput() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "open file") {
		t.Errorf("readPatchInput() error = %v, want 'open file' error", err)
	}
}

// TestFormatParseError tests parse error formatting.
func TestFormatParseError(t *testing.T) {
	testErr := &patchapply.Error{
		Op:   "Parse",
		Path: "",
		Line: 5,
		Err:  errors.New("invalid operation"),
	}

	result := formatParseError(testErr)
	resultStr := result.Error()

	// Check key elements.
	if !strings.Contains(resultStr, "Invalid patch syntax") {
		t.Errorf("formatParseError() missing 'Invalid patch syntax'")
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
			contains: []string{"Failed to apply patch", "Update", "test.go", "Hint"},
		},
		{
			name:     "generic error",
			err:      patchapply.ErrEmptyWorkspace,
			wantErr:  true,
			contains: []string{"Error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			hint := getHintForError(tt.err)
			if !strings.Contains(hint, tt.contains) {
				t.Errorf("getHintForError() = %q, want to contain %q", hint, tt.contains)
			}
		})
	}
}

// TestApplyPatch_Success tests successful patch application.
func TestApplyPatch_Success(t *testing.T) {
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

	// Set flags.
	applyPatchFile = patchFile
	applyPatchWorkspace = tmpDir
	applyPatchDryRun = false
	applyPatchForce = false
	applyPatchVerbose = false

	defer func() {
		applyPatchFile = ""
		applyPatchWorkspace = ""
		applyPatchDryRun = false
		applyPatchForce = false
		applyPatchVerbose = false
	}()

	// Run command.
	err = runApplyPatch(nil, nil)
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

	// Set flags.
	applyPatchFile = patchFile
	applyPatchWorkspace = tmpDir
	applyPatchDryRun = true

	defer func() {
		applyPatchFile = ""
		applyPatchWorkspace = ""
		applyPatchDryRun = false
	}()

	// Run command.
	err = runApplyPatch(nil, nil)
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

	// Set flags.
	applyPatchFile = patchFile
	applyPatchWorkspace = tmpDir

	defer func() {
		applyPatchFile = ""
		applyPatchWorkspace = ""
	}()

	// Run command (should fail).
	err = runApplyPatch(nil, nil)
	if err == nil {
		t.Error("runApplyPatch() expected error for invalid patch")
	}

	if !strings.Contains(err.Error(), "Invalid patch syntax") {
		t.Errorf("runApplyPatch() error = %v, want 'Invalid patch syntax'", err)
	}
}

// TestApplyPatch_PathTraversal tests path traversal rejection.
func TestApplyPatch_PathTraversal(t *testing.T) {
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

	// Set flags.
	applyPatchFile = patchFile
	applyPatchWorkspace = tmpDir

	defer func() {
		applyPatchFile = ""
		applyPatchWorkspace = ""
	}()

	// Run command (should fail).
	err = runApplyPatch(nil, nil)
	if err == nil {
		t.Error("runApplyPatch() should reject path traversal")
	}

	if !strings.Contains(err.Error(), "path") {
		t.Errorf("runApplyPatch() error = %v, want path-related error", err)
	}
}

// TestApplyPatch_ForceOverwrite tests force mode.
func TestApplyPatch_ForceOverwrite(t *testing.T) {
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
		// Set flags (no force).
		applyPatchFile = patchFile
		applyPatchWorkspace = tmpDir
		applyPatchForce = false

		defer func() {
			applyPatchFile = ""
			applyPatchWorkspace = ""
			applyPatchForce = false
		}()

		// Should fail.
		err := runApplyPatch(nil, nil)
		if err == nil {
			t.Error("runApplyPatch() should fail without --force")
		}
	})

	t.Run("with force", func(t *testing.T) {
		// Set flags (with force).
		applyPatchFile = patchFile
		applyPatchWorkspace = tmpDir
		applyPatchForce = true

		defer func() {
			applyPatchFile = ""
			applyPatchWorkspace = ""
			applyPatchForce = false
		}()

		// Should succeed.
		err := runApplyPatch(nil, nil)
		if err != nil {
			t.Errorf("runApplyPatch() with --force error = %v", err)
		}

		// Verify file was overwritten.
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}
		// Note: patch adds content as-is, no automatic trailing newline.
		if string(content) != "new content" {
			t.Errorf("file content = %q, want %q", string(content), "new content")
		}
	})
}

// TestPrintResults tests result output formatting.
func TestPrintResults(t *testing.T) {
	result := &patchapply.ApplyResult{
		FilesCreated: []string{"new1.txt", "new2.txt"},
		FilesUpdated: []string{"updated.txt"},
		FilesDeleted: []string{"deleted.txt"},
		FilesMoved:   map[string]string{"old.txt": "new.txt"},
		DryRun:       false,
	}

	// Test with verbose=false.
	applyPatchVerbose = false
	// Just ensure it doesn't panic.
	printResults(result)

	// Test with verbose=true.
	applyPatchVerbose = true

	printResults(result)

	applyPatchVerbose = false
}

// TestRunDryRun tests dry-run output.
func TestRunDryRun(t *testing.T) {
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

// TestApplyPatch_Integration tests full end-to-end scenarios.
func TestApplyPatch_Integration(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(dir string) error
		patchText  string
		wantErr    bool
		errContain string
		verify     func(dir string) error
	}{
		{
			name: "add new file",
			setup: func(dir string) error {
				return nil
			},
			patchText: `*** Begin Patch
*** Add File: hello.txt
+Hello World
*** End Patch`,
			wantErr: false,
			verify: func(dir string) error {
				content, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
				if err != nil {
					return err
				}
				// Note: patch adds content as-is, no automatic trailing newline.
				if string(content) != "Hello World" {
					return errors.New("content mismatch")
				}

				return nil
			},
		},
		{
			name: "delete file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "delete-me.txt"), []byte("content"), 0644)
			},
			patchText: `*** Begin Patch
*** Delete File: delete-me.txt
*** End Patch`,
			wantErr: false,
			verify: func(dir string) error {
				_, err := os.Stat(filepath.Join(dir, "delete-me.txt"))
				if !os.IsNotExist(err) {
					return errors.New("file should be deleted")
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
			patchText: `*** Begin Patch
*** Update File: update.txt
@@
 line 1
-line 2
+line 2 updated
 line 3
*** End Patch`,
			wantErr: false,
			verify: func(dir string) error {
				content, err := os.ReadFile(filepath.Join(dir, "update.txt"))
				if err != nil {
					return err
				}

				expected := "line 1\nline 2 updated\nline 3\n"
				if string(content) != expected {
					return errors.New("content mismatch")
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
			patchText: `*** Begin Patch
*** Update File: mismatch.txt
@@
 expected line
-old line
+new line
*** End Patch`,
			wantErr:    true,
			errContain: "context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Setup.
			if tt.setup != nil {
				err := tt.setup(tmpDir)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			// Write patch.
			patchFile := filepath.Join(tmpDir, "test.patch")
			err := os.WriteFile(patchFile, []byte(tt.patchText), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Set flags.
			applyPatchFile = patchFile
			applyPatchWorkspace = tmpDir

			defer func() {
				applyPatchFile = ""
				applyPatchWorkspace = ""
			}()

			// Run.
			err = runApplyPatch(nil, nil)

			// Check error.
			if (err != nil) != tt.wantErr {
				t.Errorf("runApplyPatch() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
				t.Errorf("runApplyPatch() error = %v, want to contain %q", err, tt.errContain)
			}

			// Verify.
			if !tt.wantErr && tt.verify != nil {
				err := tt.verify(tmpDir)
				if err != nil {
					t.Errorf("verification failed: %v", err)
				}
			}
		})
	}
}
