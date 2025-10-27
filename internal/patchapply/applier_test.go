package patchapply

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper functions

func createTempWorkspace(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "applier-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func createFile(t *testing.T, workspace, path, content string) {
	t.Helper()
	fullPath := filepath.Join(workspace, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create parent dirs: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
}

func readFile(t *testing.T, workspace, path string) string {
	t.Helper()
	fullPath := filepath.Join(workspace, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	return string(content)
}

func fileExists(workspace, path string) bool {
	fullPath := filepath.Join(workspace, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// Test: Applier Creation

func TestNewApplier(t *testing.T) {
	workspace := createTempWorkspace(t)

	tests := []struct {
		name    string
		root    string
		wantErr bool
	}{
		{"valid workspace", workspace, false},
		{"empty root", "", true},
		{"nonexistent dir", filepath.Join(workspace, "nonexistent"), false}, // OK, can be created
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applier, err := NewApplier(tt.root)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewApplier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && applier == nil {
				t.Error("NewApplier() returned nil applier")
			}
		})
	}
}

// Test: Path Validation

func TestApplier_ValidatePath(t *testing.T) {
	workspace := createTempWorkspace(t)
	applier, err := NewApplier(workspace)
	if err != nil {
		t.Fatalf("NewApplier() failed: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid relative", "src/main.go", false},
		{"valid nested", "internal/handler/user.go", false},
		{"absolute path", "/etc/passwd", true},
		{"traversal", "../../../etc/passwd", true},
		{"hidden traversal", "foo/../../../bar", true},
		{"empty path", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a dummy Add operation to test path validation
			patch := &Patch{
				Operations: []FileOperation{
					&AddFile{FilePath: tt.path, Lines: []string{"test"}},
				},
			}

			err := applier.ValidatePatch(patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

// Test: Add File Operation

func TestApplier_AddFile(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		lines          []string
		existingFile   string
		forceOverwrite bool
		wantErr        bool
		errIs          error
	}{
		{
			name:     "add new file",
			filePath: "test.txt",
			lines:    []string{"line1", "line2"},
			wantErr:  false,
		},
		{
			name:     "add nested file",
			filePath: "src/handler/user.go",
			lines:    []string{"package handler", "", "func User() {}"},
			wantErr:  false,
		},
		{
			name:         "add existing file without force",
			filePath:     "test.txt",
			lines:        []string{"new content"},
			existingFile: "old content",
			wantErr:      true,
			errIs:        ErrFileExists,
		},
		{
			name:           "add existing file with force",
			filePath:       "test.txt",
			lines:          []string{"new content"},
			existingFile:   "old content",
			forceOverwrite: true,
			wantErr:        false,
		},
		{
			name:     "add empty file",
			filePath: "empty.txt",
			lines:    []string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := createTempWorkspace(t)
			applier, _ := NewApplier(workspace)
			applier.SetForceOverwrite(tt.forceOverwrite)

			// Create existing file if specified
			if tt.existingFile != "" {
				createFile(t, workspace, tt.filePath, tt.existingFile)
			}

			patch := &Patch{
				Operations: []FileOperation{
					&AddFile{FilePath: tt.filePath, Lines: tt.lines},
				},
			}

			result, err := applier.Apply(patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply(AddFile) error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("error = %v, want %v", err, tt.errIs)
				}
				return
			}

			// Verify file created
			if !fileExists(workspace, tt.filePath) {
				t.Errorf("file %q not created", tt.filePath)
				return
			}

			// Verify content
			expectedContent := strings.Join(tt.lines, "\n")
			actualContent := readFile(t, workspace, tt.filePath)
			if actualContent != expectedContent {
				t.Errorf("file content = %q, want %q", actualContent, expectedContent)
			}

			// Verify result
			if len(result.FilesCreated) != 1 || result.FilesCreated[0] != tt.filePath {
				t.Errorf("result.FilesCreated = %v, want [%s]", result.FilesCreated, tt.filePath)
			}
		})
	}
}

// Test: Delete File Operation

func TestApplier_DeleteFile(t *testing.T) {
	tests := []struct {
		name         string
		filePath     string
		existingFile string
		wantErr      bool
		errIs        error
	}{
		{
			name:         "delete existing file",
			filePath:     "test.txt",
			existingFile: "content",
			wantErr:      false,
		},
		{
			name:     "delete nonexistent file",
			filePath: "missing.txt",
			wantErr:  true,
			errIs:    ErrFileNotFound,
		},
		{
			name:         "delete nested file",
			filePath:     "src/handler/old.go",
			existingFile: "package handler",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := createTempWorkspace(t)
			applier, _ := NewApplier(workspace)

			// Create existing file if specified
			if tt.existingFile != "" {
				createFile(t, workspace, tt.filePath, tt.existingFile)
			}

			patch := &Patch{
				Operations: []FileOperation{
					&DeleteFile{FilePath: tt.filePath},
				},
			}

			result, err := applier.Apply(patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply(DeleteFile) error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("error = %v, want %v", err, tt.errIs)
				}
				return
			}

			// Verify file deleted
			if fileExists(workspace, tt.filePath) {
				t.Errorf("file %q still exists after delete", tt.filePath)
			}

			// Verify result
			if len(result.FilesDeleted) != 1 || result.FilesDeleted[0] != tt.filePath {
				t.Errorf("result.FilesDeleted = %v, want [%s]", result.FilesDeleted, tt.filePath)
			}
		})
	}
}

// Test: Update File Operation

func TestApplier_UpdateFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		original string
		hunks    []Hunk
		expected string
		wantErr  bool
		errIs    error
	}{
		{
			name:     "simple replace",
			filePath: "test.txt",
			original: "line1\nline2\nline3",
			hunks: []Hunk{
				{
					Header: "",
					Changes: []LineChange{
						{Type: LineContext, Text: "line1"},
						{Type: LineDelete, Text: "line2"},
						{Type: LineInsert, Text: "newline"},
						{Type: LineContext, Text: "line3"},
					},
				},
			},
			expected: "line1\nnewline\nline3",
			wantErr:  false,
		},
		{
			name:     "insert at beginning",
			filePath: "test.txt",
			original: "line1\nline2",
			hunks: []Hunk{
				{
					Header: "",
					Changes: []LineChange{
						{Type: LineInsert, Text: "newline"},
						{Type: LineContext, Text: "line1"},
					},
				},
			},
			expected: "newline\nline1\nline2",
			wantErr:  false,
		},
		{
			name:     "delete line",
			filePath: "test.txt",
			original: "line1\nline2\nline3",
			hunks: []Hunk{
				{
					Header: "",
					Changes: []LineChange{
						{Type: LineContext, Text: "line1"},
						{Type: LineDelete, Text: "line2"},
						{Type: LineContext, Text: "line3"},
					},
				},
			},
			expected: "line1\nline3",
			wantErr:  false,
		},
		{
			name:     "context not found",
			filePath: "test.txt",
			original: "line1\nline2\nline3",
			hunks: []Hunk{
				{
					Header: "",
					Changes: []LineChange{
						{Type: LineContext, Text: "nonexistent"},
						{Type: LineDelete, Text: "line2"},
					},
				},
			},
			wantErr: true,
			errIs:   ErrContextNotFound,
		},
		{
			name:     "update nonexistent file",
			filePath: "missing.txt",
			original: "",
			hunks: []Hunk{
				{
					Changes: []LineChange{
						{Type: LineContext, Text: "line1"},
					},
				},
			},
			wantErr: true,
			errIs:   ErrFileNotFound,
		},
		{
			name:     "multiple hunks",
			filePath: "test.txt",
			original: "a\nb\nc\nd\ne",
			hunks: []Hunk{
				{
					Changes: []LineChange{
						{Type: LineContext, Text: "a"},
						{Type: LineDelete, Text: "b"},
						{Type: LineInsert, Text: "B"},
					},
				},
				{
					Changes: []LineChange{
						{Type: LineContext, Text: "d"},
						{Type: LineDelete, Text: "e"},
						{Type: LineInsert, Text: "E"},
					},
				},
			},
			expected: "a\nB\nc\nd\nE",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := createTempWorkspace(t)
			applier, _ := NewApplier(workspace)

			// Create original file if content specified
			if tt.original != "" {
				createFile(t, workspace, tt.filePath, tt.original)
			}

			patch := &Patch{
				Operations: []FileOperation{
					&UpdateFile{
						FilePath: tt.filePath,
						Hunks:    tt.hunks,
					},
				},
			}

			result, err := applier.Apply(patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply(UpdateFile) error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("error = %v, want %v", err, tt.errIs)
				}
				return
			}

			// Verify content
			actualContent := readFile(t, workspace, tt.filePath)
			if actualContent != tt.expected {
				t.Errorf("file content = %q, want %q", actualContent, tt.expected)
			}

			// Verify result
			if len(result.FilesUpdated) != 1 || result.FilesUpdated[0] != tt.filePath {
				t.Errorf("result.FilesUpdated = %v, want [%s]", result.FilesUpdated, tt.filePath)
			}
		})
	}
}

// Test: Move File Operation

func TestApplier_MoveFile(t *testing.T) {
	tests := []struct {
		name     string
		oldPath  string
		newPath  string
		original string
		wantErr  bool
	}{
		{
			name:     "simple rename",
			oldPath:  "old.txt",
			newPath:  "new.txt",
			original: "content",
			wantErr:  false,
		},
		{
			name:     "move to subdirectory",
			oldPath:  "file.txt",
			newPath:  "subdir/file.txt",
			original: "content",
			wantErr:  false,
		},
		{
			name:     "rename and move",
			oldPath:  "src/old.go",
			newPath:  "internal/handler/new.go",
			original: "package handler",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := createTempWorkspace(t)
			applier, _ := NewApplier(workspace)

			// Create original file
			createFile(t, workspace, tt.oldPath, tt.original)

			patch := &Patch{
				Operations: []FileOperation{
					&UpdateFile{
						FilePath: tt.oldPath,
						NewPath:  tt.newPath,
						Hunks:    []Hunk{}, // No content changes
					},
				},
			}

			result, err := applier.Apply(patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply(Move) error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify old file deleted
			if fileExists(workspace, tt.oldPath) {
				t.Errorf("old file %q still exists", tt.oldPath)
			}

			// Verify new file created
			if !fileExists(workspace, tt.newPath) {
				t.Errorf("new file %q not created", tt.newPath)
			}

			// Verify content preserved
			actualContent := readFile(t, workspace, tt.newPath)
			if actualContent != tt.original {
				t.Errorf("file content = %q, want %q", actualContent, tt.original)
			}

			// Verify result
			if result.FilesMoved == nil || result.FilesMoved[tt.oldPath] != tt.newPath {
				t.Errorf("result.FilesMoved = %v, want {%s: %s}", result.FilesMoved, tt.oldPath, tt.newPath)
			}
		})
	}
}

// Test: Dry-Run Mode

func TestApplier_DryRun(t *testing.T) {
	workspace := createTempWorkspace(t)
	applier, _ := NewApplier(workspace)
	applier.SetDryRun(true)

	patch := &Patch{
		Operations: []FileOperation{
			&AddFile{FilePath: "test.txt", Lines: []string{"content"}},
		},
	}

	result, err := applier.Apply(patch)
	if err != nil {
		t.Errorf("Apply() in dry-run failed: %v", err)
		return
	}

	// Verify no files created
	if fileExists(workspace, "test.txt") {
		t.Error("file created in dry-run mode")
	}

	// Verify result indicates dry-run
	if !result.DryRun {
		t.Error("result.DryRun = false, want true")
	}
}

// Test: Atomic Rollback

func TestApplier_AtomicRollback(t *testing.T) {
	workspace := createTempWorkspace(t)
	applier, _ := NewApplier(workspace)

	// Create existing file for context
	createFile(t, workspace, "existing.txt", "line1\nline2\nline3")

	// Patch with multiple operations, last one will fail
	patch := &Patch{
		Operations: []FileOperation{
			&AddFile{FilePath: "new1.txt", Lines: []string{"content1"}},
			&AddFile{FilePath: "new2.txt", Lines: []string{"content2"}},
			&UpdateFile{
				FilePath: "existing.txt",
				Hunks: []Hunk{
					{
						Changes: []LineChange{
							{Type: LineContext, Text: "nonexistent"}, // Will fail
							{Type: LineInsert, Text: "new"},
						},
					},
				},
			},
		},
	}

	_, err := applier.Apply(patch)
	if err == nil {
		t.Error("Apply() should have failed")
		return
	}

	// Verify rollback: new files should not exist
	if fileExists(workspace, "new1.txt") {
		t.Error("new1.txt exists after rollback")
	}
	if fileExists(workspace, "new2.txt") {
		t.Error("new2.txt exists after rollback")
	}

	// Verify existing file unchanged
	content := readFile(t, workspace, "existing.txt")
	if content != "line1\nline2\nline3" {
		t.Errorf("existing file modified after rollback: %q", content)
	}
}

// Test: Multiple Operations

func TestApplier_MultipleOperations(t *testing.T) {
	workspace := createTempWorkspace(t)
	applier, _ := NewApplier(workspace)

	// Create existing files
	createFile(t, workspace, "old.txt", "old content")
	createFile(t, workspace, "update.txt", "line1\nline2\nline3")

	patch := &Patch{
		Operations: []FileOperation{
			&AddFile{FilePath: "new.txt", Lines: []string{"new content"}},
			&DeleteFile{FilePath: "old.txt"},
			&UpdateFile{
				FilePath: "update.txt",
				Hunks: []Hunk{
					{
						Changes: []LineChange{
							{Type: LineContext, Text: "line1"},
							{Type: LineDelete, Text: "line2"},
							{Type: LineInsert, Text: "LINE2"},
							{Type: LineContext, Text: "line3"},
						},
					},
				},
			},
		},
	}

	result, err := applier.Apply(patch)
	if err != nil {
		t.Fatalf("Apply() failed: %v", err)
	}

	// Verify all operations
	if len(result.FilesCreated) != 1 || result.FilesCreated[0] != "new.txt" {
		t.Errorf("FilesCreated = %v, want [new.txt]", result.FilesCreated)
	}
	if len(result.FilesDeleted) != 1 || result.FilesDeleted[0] != "old.txt" {
		t.Errorf("FilesDeleted = %v, want [old.txt]", result.FilesDeleted)
	}
	if len(result.FilesUpdated) != 1 || result.FilesUpdated[0] != "update.txt" {
		t.Errorf("FilesUpdated = %v, want [update.txt]", result.FilesUpdated)
	}

	// Verify file states
	if !fileExists(workspace, "new.txt") {
		t.Error("new.txt not created")
	}
	if fileExists(workspace, "old.txt") {
		t.Error("old.txt not deleted")
	}

	content := readFile(t, workspace, "update.txt")
	if content != "line1\nLINE2\nline3" {
		t.Errorf("update.txt content = %q, want %q", content, "line1\nLINE2\nline3")
	}
}

// Test: ValidatePatch

func TestApplier_ValidatePatch(t *testing.T) {
	workspace := createTempWorkspace(t)
	applier, _ := NewApplier(workspace)

	tests := []struct {
		name    string
		patch   *Patch
		wantErr bool
	}{
		{
			name: "valid patch",
			patch: &Patch{
				Operations: []FileOperation{
					&AddFile{FilePath: "test.txt", Lines: []string{"content"}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid path",
			patch: &Patch{
				Operations: []FileOperation{
					&AddFile{FilePath: "/etc/passwd", Lines: []string{"content"}},
				},
			},
			wantErr: true,
		},
		{
			name: "empty operations",
			patch: &Patch{
				Operations: []FileOperation{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applier.ValidatePatch(tt.patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test: Error Messages

func TestApplier_ErrorMessages(t *testing.T) {
	workspace := createTempWorkspace(t)
	applier, _ := NewApplier(workspace)

	tests := []struct {
		name        string
		patch       *Patch
		wantErrType error
		checkMsg    func(string) bool
	}{
		{
			name: "path traversal",
			patch: &Patch{
				Operations: []FileOperation{
					&AddFile{FilePath: "../../etc/passwd", Lines: []string{}},
				},
			},
			wantErrType: ErrPathOutsideWorkspace,
			checkMsg:    func(msg string) bool { return strings.Contains(msg, "../../etc/passwd") },
		},
		{
			name: "file not found",
			patch: &Patch{
				Operations: []FileOperation{
					&DeleteFile{FilePath: "nonexistent.txt"},
				},
			},
			wantErrType: ErrFileNotFound,
			checkMsg:    func(msg string) bool { return strings.Contains(msg, "nonexistent.txt") },
		},
		{
			name: "context not found",
			patch: &Patch{
				Operations: []FileOperation{
					&UpdateFile{
						FilePath: "test.txt",
						Hunks: []Hunk{
							{
								Changes: []LineChange{
									{Type: LineContext, Text: "missing"},
								},
							},
						},
					},
				},
			},
			wantErrType: ErrContextNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test.txt for update test
			if tt.name == "context not found" {
				createFile(t, workspace, "test.txt", "some content")
			}

			_, err := applier.Apply(tt.patch)
			if err == nil {
				t.Error("Apply() should have failed")
				return
			}

			if !errors.Is(err, tt.wantErrType) {
				t.Errorf("error type = %v, want %v", err, tt.wantErrType)
			}

			if tt.checkMsg != nil && !tt.checkMsg(err.Error()) {
				t.Errorf("error message doesn't contain expected info: %v", err)
			}
		})
	}
}

// Benchmark: Apply Small Patch

func BenchmarkApplier_SmallPatch(b *testing.B) {
	workspace := createTempWorkspace(&testing.T{})
	applier, _ := NewApplier(workspace)

	// Create test file
	createFile(&testing.T{}, workspace, "test.go", strings.Repeat("line\n", 100))

	patch := &Patch{
		Operations: []FileOperation{
			&UpdateFile{
				FilePath: "test.go",
				Hunks: []Hunk{
					{
						Changes: []LineChange{
							{Type: LineContext, Text: "line"},
							{Type: LineInsert, Text: "new line"},
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applier.Apply(patch)
	}
}
