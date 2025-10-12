package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// ApplyPatchOptions configures patch application behavior
type ApplyPatchOptions struct {
	// DryRun validates the patch without applying changes
	DryRun bool
	// Force allows overwriting existing files
	Force bool
}

// ApplyPatchResult contains the result of a patch application
type ApplyPatchResult struct {
	Success       bool
	Message       string
	FilesModified []string
	Error         *PatchError
}

// PatchError provides detailed error information for patch failures
type PatchError struct {
	Message  string
	FilePath string
	Line     int
	Reason   string
}

// Error implements the error interface
func (e *PatchError) Error() string {
	if e.FilePath != "" && e.Line > 0 {
		return fmt.Sprintf("%s (file: %s, line: %d): %s", e.Message, e.FilePath, e.Line, e.Reason)
	}
	if e.FilePath != "" {
		return fmt.Sprintf("%s (file: %s): %s", e.Message, e.FilePath, e.Reason)
	}
	return fmt.Sprintf("%s: %s", e.Message, e.Reason)
}

// ApplyPatch applies a Git unified diff patch to the repository working tree.
//
// The patch must be in standard Git unified diff format (output of `git diff`).
// This method uses the go-gitdiff library to parse and apply patches.
//
// Example:
//
//	patchText := `diff --git a/file.txt b/file.txt
//	--- a/file.txt
//	+++ b/file.txt
//	@@ -1 +1 @@
//	-old line
//	+new line
//	`
//
//	result, err := repo.ApplyPatch(ctx, patchText, ApplyPatchOptions{})
//	if err != nil {
//	    return err
//	}
//
//	if !result.Success {
//	    fmt.Printf("Patch failed: %s\n", result.Error)
//	}
func (r *Repository) ApplyPatch(ctx context.Context, patchText string, opts ApplyPatchOptions) (*ApplyPatchResult, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Parse the Git diff using go-gitdiff
	files, _, err := gitdiff.Parse(strings.NewReader(patchText))
	if err != nil {
		return &ApplyPatchResult{
			Success: false,
			Message: "Failed to parse patch",
			Error: &PatchError{
				Message: "Patch parsing failed",
				Reason:  err.Error(),
			},
		}, nil
	}

	if len(files) == 0 {
		return &ApplyPatchResult{
			Success: true,
			Message: "Empty patch (no changes)",
		}, nil
	}

	// Apply each file patch
	filesModified := make([]string, 0, len(files))
	for _, file := range files {
		filePath := file.NewName
		if filePath == "" {
			filePath = file.OldName
		}

		if err := r.applyFilePatch(ctx, file, opts); err != nil {
			// Check if it's an ApplyError with line information
			if applyErr, ok := err.(*gitdiff.ApplyError); ok {
				return &ApplyPatchResult{
					Success:       false,
					Message:       "Patch application failed",
					FilesModified: filesModified,
					Error: &PatchError{
						Message:  "Failed to apply patch",
						FilePath: filePath,
						Line:     int(applyErr.Line),
						Reason:   applyErr.Unwrap().Error(),
					},
				}, nil
			}

			return &ApplyPatchResult{
				Success:       false,
				Message:       "Patch application failed",
				FilesModified: filesModified,
				Error: &PatchError{
					Message:  "Failed to apply patch",
					FilePath: filePath,
					Reason:   err.Error(),
				},
			}, nil
		}
		filesModified = append(filesModified, filePath)
	}

	message := "Patch applied successfully"
	if opts.DryRun {
		message = "Patch can be applied (dry-run)"
	}

	return &ApplyPatchResult{
		Success:       true,
		Message:       message,
		FilesModified: filesModified,
	}, nil
}

// applyFilePatch applies a single file patch using go-gitdiff
func (r *Repository) applyFilePatch(ctx context.Context, file *gitdiff.File, opts ApplyPatchOptions) error {
	oldPath := filepath.Join(r.root, file.OldName)
	newPath := filepath.Join(r.root, file.NewName)

	// Handle file deletion
	if file.IsDelete {
		if _, err := os.Stat(oldPath); os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", file.OldName)
		}
		if opts.DryRun {
			return nil
		}
		return os.Remove(oldPath)
	}

	// Handle new file creation
	if file.IsNew {
		if !opts.Force {
			if _, err := os.Stat(newPath); err == nil {
				return fmt.Errorf("file already exists: %s", file.NewName)
			}
		}

		if opts.DryRun {
			return nil
		}

		// Create directory if needed
		dir := filepath.Dir(newPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}

		// Apply the patch to create new file
		var output bytes.Buffer
		if err := gitdiff.Apply(&output, &emptyReaderAt{}, file); err != nil {
			return fmt.Errorf("apply new file patch: %w", err)
		}

		return os.WriteFile(newPath, output.Bytes(), 0644)
	}

	// Handle file modification or rename
	srcPath := oldPath
	if file.IsRename {
		srcPath = oldPath
	}

	// Read source file
	srcContent, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	// Apply patch
	var output bytes.Buffer
	src := bytes.NewReader(srcContent)
	if err := gitdiff.Apply(&output, src, file); err != nil {
		return err
	}

	if opts.DryRun {
		return nil
	}

	// Handle rename
	if file.IsRename && oldPath != newPath {
		// Create target directory if needed
		dir := filepath.Dir(newPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}

		// Remove old file
		if err := os.Remove(oldPath); err != nil {
			return fmt.Errorf("remove old file: %w", err)
		}
	}

	// Write modified content
	return os.WriteFile(newPath, output.Bytes(), 0644)
}

// emptyReaderAt provides an empty reader for new file creation
type emptyReaderAt struct{}

func (e *emptyReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, io.EOF
}
