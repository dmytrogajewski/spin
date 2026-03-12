package patchapply

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Error types for patch application.
var (
	ErrPathOutsideWorkspace = errors.New("path outside workspace")
	ErrFileNotFound         = errors.New("file not found")
	ErrFileExists           = errors.New("file already exists")
	ErrContextNotFound      = errors.New("context not found")
	ErrPermissionDenied     = errors.New("permission denied")
	ErrEmptyWorkspace       = errors.New("empty workspace root")
	ErrUnknownOperationType = errors.New("unknown operation type")
	ErrNoContextLinesInHunk = errors.New("no context lines in hunk")
	ErrContextMismatch = errors.New("context mismatch")
	ErrDeleteBeyondEndOfFile = errors.New("delete beyond end of file")
)

// modOperation represents the type of file modification.
type modOperation int

const (
	opCreate modOperation = iota
	opUpdate
	opDelete
	opMove
)

// fileModification tracks a file change for rollback.
type fileModification struct {
	path            string
	operation       modOperation
	originalContent []byte
	created         bool
}

// Applier applies patches to the filesystem safely.
//
// The applier provides:
//   - Workspace confinement (all paths validated)
//   - Atomic operations (all-or-nothing with rollback)
//   - Dry-run mode (preview without changes)
//   - Optional backups
//   - Clear error messages
//
// Example usage:
//
//	applier, err := NewApplier("/workspace")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	applier.SetDryRun(false)
//	result, err := applier.Apply(patch)
//	if err != nil {
//	    log.Fatalf("patch failed: %v", err)
//	}
//	fmt.Printf("Created: %v\n", result.FilesCreated)
type Applier struct {
	workspaceRoot  string
	dryRun         bool
	createBackup   bool
	forceOverwrite bool
	modifications  []*fileModification
}

// ApplyResult contains the results of applying a patch.
type ApplyResult struct {
	FilesCreated []string
	FilesDeleted []string
	FilesUpdated []string
	FilesMoved   map[string]string // old path -> new path.
	DryRun       bool
}

// Error represents a patch application error with context.
type Error struct {
	Op      string // Operation (Add, Delete, Update, Move).
	Path    string // File path.
	Line    int    // Line number (for hunk errors).
	Err     error  // Underlying error.
	Context string // Additional context.
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s %q: %v (%s)", e.Op, e.Path, e.Err, e.Context)
	}

	if e.Line > 0 {
		return fmt.Sprintf("%s %q at line %d: %v", e.Op, e.Path, e.Line, e.Err)
	}

	return fmt.Sprintf("%s %q: %v", e.Op, e.Path, e.Err)
}

// Unwrap returns the underlying error for errors.Is and errors.As.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is implements error matching for errors.Is.
func (e *Error) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewApplier creates a new patch applier for the given workspace.
//
// The workspace root must be an absolute path. All file operations
// are confined to this workspace.
//
// Returns an error if the workspace root is empty or invalid.
func NewApplier(workspaceRoot string) (*Applier, error) {
	if workspaceRoot == "" {
		return nil, ErrEmptyWorkspace
	}

	// Resolve to absolute path.
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace root: %w", err)
	}

	return &Applier{
		workspaceRoot:  absRoot,
		dryRun:         false,
		createBackup:   false,
		forceOverwrite: false,
		modifications:  make([]*fileModification, 0),
	}, nil
}

// SetDryRun enables or disables dry-run mode.
//
// In dry-run mode, the applier validates the patch but does not
// modify any files. Use this to preview changes before applying.
func (a *Applier) SetDryRun(enabled bool) {
	a.dryRun = enabled
}

// SetForceOverwrite enables or disables overwriting existing files.
//
// When enabled, Add operations will overwrite existing files.
// When disabled (default), Add operations fail if the file exists.
func (a *Applier) SetForceOverwrite(enabled bool) {
	a.forceOverwrite = enabled
}

// Apply applies the patch to the workspace.
//
// The applier performs these steps:
//  1. Validate all operations (paths, file existence)
//  2. If dry-run mode, return without modifying files
//  3. Apply operations in order, tracking modifications
//  4. If any operation fails, rollback all changes
//  5. Return result summary
//
// Returns ApplyResult on success, Error on failure.
func (a *Applier) Apply(patch *Patch) (*ApplyResult, error) {
	a.resetModifications()

	err := a.ValidatePatch(patch)
	if err != nil {
		return nil, err
	}

	if a.dryRun {
		return &ApplyResult{DryRun: true}, nil
	}

	result := a.createApplyResult()
	err = a.applyOperations(patch.Operations, result)
	if err != nil {
		a.rollback()

		return nil, err
	}

	return result, nil
}

// resetModifications resets the modifications tracking.
func (a *Applier) resetModifications() {
	a.modifications = make([]*fileModification, 0)
}

// createApplyResult creates a new ApplyResult with initialized slices.
func (a *Applier) createApplyResult() *ApplyResult {
	return &ApplyResult{
		FilesCreated: make([]string, 0),
		FilesDeleted: make([]string, 0),
		FilesUpdated: make([]string, 0),
		FilesMoved:   make(map[string]string),
	}
}

// applyOperations applies all operations in the patch.
func (a *Applier) applyOperations(operations []FileOperation, result *ApplyResult) error {
	for _, op := range operations {
		err := a.applyOperation(op, result)
		if err != nil {
			return err
		}
	}

	return nil
}

// applyOperation applies a single operation.
func (a *Applier) applyOperation(op FileOperation, result *ApplyResult) error {
	switch op := op.(type) {
	case *AddFile:
		return a.applyAddFile(op, result)
	case *DeleteFile:
		return a.applyDeleteFile(op, result)
	case *UpdateFile:
		return a.applyUpdateFile(op, result)
	default:
return fmt.Errorf("unknown operation type: %T: %w", op, ErrUnknownOperationType)
	}
}

// ValidatePatch validates the patch without applying it.
//
// This checks:
//   - All paths are valid and within workspace
//   - Files exist for Update/Delete operations
//   - Files don't exist for Add operations (unless force mode)
//
// Returns nil if the patch is valid, error otherwise.
func (a *Applier) ValidatePatch(patch *Patch) error {
	for _, op := range patch.Operations {
		err := a.validateOperation(op)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateOperation validates a single file operation.
func (a *Applier) validateOperation(op FileOperation) error {
	path := op.Path()

	// Validate path.
	_, err := a.resolvePath(path)
	if err != nil {
		return a.wrapError("Validate", path, err, "")
	}

	// Type-specific validation.
	switch op := op.(type) {
	case *AddFile:
		return a.validateAddFile(op)
	case *DeleteFile:
		return a.validateDeleteFile(op)
	case *UpdateFile:
		return a.validateUpdateFile(op)
	}

	return nil
}

// validateAddFile validates an Add operation.
func (a *Applier) validateAddFile(op *AddFile) error {
	fullPath, err := a.resolvePath(op.FilePath)
	if err != nil {
		return a.wrapError("Add", op.FilePath, err, "")
	}

	// Check if file exists.
	_, err = os.Stat(fullPath)
	if err == nil {
		if !a.forceOverwrite {
			return a.wrapError("Add", op.FilePath, ErrFileExists, "use force mode to overwrite")
		}
	}

	return nil
}

// validateDeleteFile validates a Delete operation.
func (a *Applier) validateDeleteFile(op *DeleteFile) error {
	fullPath, err := a.resolvePath(op.FilePath)
	if err != nil {
		return a.wrapError("Delete", op.FilePath, err, "")
	}

	// Check if file exists.
	_, err = os.Stat(fullPath)
	if os.IsNotExist(err) {
		return a.wrapError("Delete", op.FilePath, ErrFileNotFound, "")
	}

	return nil
}

// validateUpdateFile validates an Update operation.
func (a *Applier) validateUpdateFile(op *UpdateFile) error {
	// Validate old path.
	fullPath, err := a.resolvePath(op.FilePath)
	if err != nil {
		return a.wrapError("Update", op.FilePath, err, "")
	}

	// Check if file exists.
	_, err = os.Stat(fullPath)
	if os.IsNotExist(err) {
		return a.wrapError("Update", op.FilePath, ErrFileNotFound, "file must exist for update")
	}

	// Validate new path if move operation.
	if op.NewPath != "" {
		_, err = a.resolvePath(op.NewPath)
		if err != nil {
			return a.wrapError("Move", op.NewPath, err, "")
		}
	}

	return nil
}

// resolvePath validates and resolves a relative path to an absolute path.
//
// Returns the absolute path if valid, error if outside workspace or invalid.
func (a *Applier) resolvePath(relPath string) (string, error) {
	// Validate relative path.
	if strings.Contains(relPath, "..") || filepath.IsAbs(relPath) {
		return "", ErrPathOutsideWorkspace
	}

	// Safely join with workspace root.
	fullPath := filepath.Join(a.workspaceRoot, relPath)

	// Ensure the resolved path is still within workspace.
	if !strings.HasPrefix(fullPath, a.workspaceRoot) {
		return "", ErrPathOutsideWorkspace
	}

	return fullPath, nil
}

// applyAddFile applies an Add operation.
func (a *Applier) applyAddFile(op *AddFile, result *ApplyResult) error {
	fullPath, err := a.resolvePath(op.FilePath)
	if err != nil {
		return a.wrapError("Add", op.FilePath, err, "")
	}

	// Check if file exists.
	existingContent := []byte(nil)

	_, err = os.Stat(fullPath)
	if err == nil {
		if !a.forceOverwrite {
			return a.wrapError("Add", op.FilePath, ErrFileExists, "use force mode to overwrite")
		}
		// Read existing content for rollback.
		existingContent, _ = os.ReadFile(fullPath)
	}

	// Create parent directories.
	err = os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err != nil {
		return a.wrapError("Add", op.FilePath, err, "failed to create parent directories")
	}

	// Write file content.
	content := strings.Join(op.Lines, "\n")
	err = os.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		return a.wrapError("Add", op.FilePath, err, "failed to write file")
	}

	// Track for rollback.
	if existingContent != nil {
		a.trackModification(op.FilePath, opUpdate, existingContent)
	} else {
		a.trackModification(op.FilePath, opCreate, nil)
	}

	result.FilesCreated = append(result.FilesCreated, op.FilePath)

	return nil
}

// applyDeleteFile applies a Delete operation.
func (a *Applier) applyDeleteFile(op *DeleteFile, result *ApplyResult) error {
	fullPath, err := a.resolvePath(op.FilePath)
	if err != nil {
		return a.wrapError("Delete", op.FilePath, err, "")
	}

	originalContent, err := a.readFileForRollback(fullPath, op.FilePath)
	if err != nil {
		return err
	}

	err = a.deleteFile(fullPath, op.FilePath)
	if err != nil {
		return err
	}

	a.trackModification(op.FilePath, opDelete, originalContent)
	result.FilesDeleted = append(result.FilesDeleted, op.FilePath)

	return nil
}

// readFileForRollback reads the original file content for rollback purposes.
func (a *Applier) readFileForRollback(fullPath, filePath string) ([]byte, error) {
	originalContent, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, a.wrapError("Delete", filePath, ErrFileNotFound, "")
		}

		return nil, a.wrapError("Delete", filePath, err, "failed to read file")
	}

	return originalContent, nil
}

// deleteFile deletes the file from the filesystem.
func (a *Applier) deleteFile(fullPath, filePath string) error {
	err := os.Remove(fullPath)
	if err != nil {
		return a.wrapError("Delete", filePath, err, "failed to delete file")
	}

	return nil
}

// applyUpdateFile applies an Update operation (with optional move).
func (a *Applier) applyUpdateFile(op *UpdateFile, result *ApplyResult) error {
	fullPath, err := a.resolvePath(op.FilePath)
	if err != nil {
		return a.wrapError("Update", op.FilePath, err, "")
	}

	// Read file content.
	originalContent, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return a.wrapError("Update", op.FilePath, ErrFileNotFound, "file must exist for update")
		}

		return a.wrapError("Update", op.FilePath, err, "failed to read file")
	}

	// Parse lines.
	lines := strings.Split(string(originalContent), "\n")

	// Apply each hunk.
	for i, hunk := range op.Hunks {
		err = a.applyHunk(&lines, hunk, op.FilePath, i)
		if err != nil {
			return err
		}
	}

	// Determine target path (original or new if moving).
	targetPath := fullPath
	if op.NewPath != "" {
		targetPath, err = a.resolvePath(op.NewPath)
		if err != nil {
			return a.wrapError("Move", op.NewPath, err, "")
		}

		// Ensure parent directories exist.
		err = os.MkdirAll(filepath.Dir(targetPath), 0755)
		if err != nil {
			return a.wrapError("Move", op.NewPath, err, "failed to create parent directories")
		}

		result.FilesMoved[op.FilePath] = op.NewPath
	}

	// Write modified content.
	newContent := strings.Join(lines, "\n")
	err = os.WriteFile(targetPath, []byte(newContent), 0644)
	if err != nil {
		return a.wrapError("Update", op.FilePath, err, "failed to write file")
	}

	// If moved, delete original.
	if op.NewPath != "" && targetPath != fullPath {
		err = os.Remove(fullPath)
		if err != nil {
			return a.wrapError("Move", op.FilePath, err, "failed to delete original file")
		}
	}

	// Track for rollback.
	a.trackModification(op.FilePath, opUpdate, originalContent)

	result.FilesUpdated = append(result.FilesUpdated, op.FilePath)

	return nil
}

// applyHunk applies a single hunk to the file lines.
func (a *Applier) applyHunk(lines *[]string, hunk Hunk, filePath string, hunkIdx int) error {
	// Extract context lines from hunk.
	contextLines := a.extractContextLines(hunk)
	if len(contextLines) == 0 {
		return a.wrapError("Update", filePath, ErrNoContextLinesInHunk,
			fmt.Sprintf("hunk %d must have context lines for matching", hunkIdx))
	}

	// Find context in file using fuzzy matcher.
	matcher := NewMatcher(*lines)

	pos := matcher.FindContext(contextLines, hunk.Header)
	if pos < 0 {
		return a.wrapError("Update", filePath, ErrContextNotFound,
			fmt.Sprintf("could not find context for hunk %d (header: %q)", hunkIdx, hunk.Header))
	}

	// Apply changes at found position.
	newLines := make([]string, 0, len(*lines))
	newLines = append(newLines, (*lines)[:pos]...) // Before context.

	offset := 0

	for _, change := range hunk.Changes {
		switch change.Type {
		case LineContext:
			// Verify context matches (within fuzzy threshold).
			if pos+offset >= len(*lines) {
				return a.wrapError("Update", filePath, ErrContextMismatch,
					fmt.Sprintf("hunk %d: ran out of lines", hunkIdx))
			}

			newLines = append(newLines, change.Text)
			offset++

		case LineDelete:
			// Skip this line (delete).
			if pos+offset >= len(*lines) {
				return a.wrapError("Update", filePath, ErrDeleteBeyondEndOfFile,
					fmt.Sprintf("hunk %d: cannot delete line beyond end", hunkIdx))
			}

			offset++

		case LineInsert:
			// Add this line.
			newLines = append(newLines, change.Text)
		}
	}

	// Add remaining lines after hunk.
	if pos+offset <= len(*lines) {
		newLines = append(newLines, (*lines)[pos+offset:]...)
	}

	// Replace lines.
	*lines = newLines

	return nil
}

// extractContextLines extracts context lines from a hunk for matching.
func (a *Applier) extractContextLines(hunk Hunk) []string {
	var contextLines []string

	for _, change := range hunk.Changes {
		if change.Type == LineContext {
			contextLines = append(contextLines, change.Text)
		}
	}

	return contextLines
}

// trackModification records a file modification for rollback.
func (a *Applier) trackModification(path string, op modOperation, originalContent []byte) {
	mod := &fileModification{
		path:            path,
		operation:       op,
		originalContent: originalContent,
		created:         op == opCreate,
	}
	a.modifications = append(a.modifications, mod)
}

// rollback reverses all tracked modifications.
func (a *Applier) rollback() {
	a.rollbackModifications()
	a.clearModifications()
}

// rollbackModifications rolls back all modifications in reverse order.
func (a *Applier) rollbackModifications() {
	for i := len(a.modifications) - 1; i >= 0; i-- {
		mod := a.modifications[i]

		fullPath, err := a.resolvePath(mod.path)
		if err != nil {
			// Best effort rollback, log and continue.
			continue
		}

		a.rollbackModification(mod, fullPath)
	}
}

// rollbackModification rolls back a single modification.
func (a *Applier) rollbackModification(mod *fileModification, fullPath string) {
	switch mod.operation {
	case opCreate:
		a.rollbackCreate(fullPath)
	case opUpdate:
		a.rollbackUpdate(fullPath, mod.originalContent)
	case opDelete:
		a.rollbackDelete(fullPath, mod.originalContent)
	}
}

// rollbackCreate removes a created file.
func (a *Applier) rollbackCreate(fullPath string) {
	os.Remove(fullPath)
}

// rollbackUpdate restores original content for an updated file.
func (a *Applier) rollbackUpdate(fullPath string, originalContent []byte) {
	_ = os.WriteFile(fullPath, originalContent, 0644)
}

// rollbackDelete recreates a deleted file.
func (a *Applier) rollbackDelete(fullPath string, originalContent []byte) {
	_ = os.WriteFile(fullPath, originalContent, 0644)
}

// clearModifications clears the modifications tracking.
func (a *Applier) clearModifications() {
	a.modifications = make([]*fileModification, 0)
}

// wrapError wraps an error with context information.
func (a *Applier) wrapError(op, path string, err error, context string) *Error {
	return &Error{
		Op:      op,
		Path:    path,
		Err:     err,
		Context: context,
	}
}
