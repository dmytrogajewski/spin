package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/patchapply"
)

// newApplyPatchCmd returns the apply-patch command for inclusion in root.
func newApplyPatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply-patch",
		Short: "Apply a Spin patch to the workspace",
		Long: `Apply a Spin patch from stdin or file.

Examples:
  # Apply from stdin
  cat changes.patch | spin apply-patch

  # Apply from file
  spin apply-patch -f changes.patch

  # Dry-run mode
  spin apply-patch --dry-run -f changes.patch

  # Custom workspace
  spin apply-patch -w /path/to/project -f changes.patch

  # Force overwrite existing files
  spin apply-patch --force -f changes.patch`,
		RunE: runApplyPatch,
	}

	cmd.Flags().StringP("file", "f", "", "Patch file (default: stdin)")
	cmd.Flags().StringP("workspace", "w", ".", "Workspace directory")
	cmd.Flags().Bool("dry-run", false, "Validate without applying")
	cmd.Flags().Bool("force", false, "Force overwrite existing files")
	cmd.Flags().BoolP("verbose", "v", false, "Verbose output")

	return cmd
}

// runApplyPatch executes the apply-patch command.
func runApplyPatch(cmd *cobra.Command, _ []string) error {
	patchFile, _ := cmd.Flags().GetString("file")
	workspace, _ := cmd.Flags().GetString("workspace")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Read patch text.
	patchText, err := readPatchInput(patchFile)
	if err != nil {
		return fmt.Errorf("read patch: %w", err)
	}

	// Parse patch.
	parser := patchapply.NewParser(patchText)

	patch, err := parser.Parse()
	if err != nil {
		return formatParseError(err)
	}

	// Resolve workspace.
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
	}

	// Create applier.
	applier, err := patchapply.NewApplier(absWorkspace)
	if err != nil {
		return fmt.Errorf("create applier: %w", err)
	}

	// Configure applier.
	applier.SetDryRun(dryRun)
	applier.SetForceOverwrite(force)

	// Apply or validate.
	if dryRun {
		return runDryRun(applier, patch)
	}

	result, err := applier.Apply(cmd.Context(), patch)
	if err != nil {
		return formatApplyError(err)
	}

	// Output results.
	printResults(result, verbose)

	return nil
}

// readPatchInput reads patch from file or stdin.
func readPatchInput(patchFile string) (string, error) {
	var reader io.Reader

	if patchFile != "" {
		f, err := os.Open(patchFile)
		if err != nil {
			return "", fmt.Errorf("open file: %w", err)
		}
		defer f.Close()

		reader = f
	} else {
		reader = os.Stdin
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}

	return string(data), nil
}

// runDryRun performs dry-run validation.
func runDryRun(applier *patchapply.Applier, patch *patchapply.Patch) error {
	err := applier.ValidatePatch(patch)
	if err != nil {
		return formatApplyError(err)
	}

	fmt.Fprintln(os.Stdout, "[DRY RUN] Would apply the following changes:")

	for _, op := range patch.Operations {
		switch patchOp := op.(type) {
		case *patchapply.AddFile:
			fmt.Fprintf(os.Stdout, "  Would create: %s (%d lines)\n", patchOp.FilePath, len(patchOp.Lines))
		case *patchapply.DeleteFile:
			fmt.Fprintf(os.Stdout, "  Would delete: %s\n", patchOp.FilePath)
		case *patchapply.UpdateFile:
			if patchOp.NewPath != "" {
				fmt.Fprintf(os.Stdout, "  Would move: %s → %s (%d hunks)\n", patchOp.FilePath, patchOp.NewPath, len(patchOp.Hunks))
			} else {
				fmt.Fprintf(os.Stdout, "  Would update: %s (%d hunks)\n", patchOp.FilePath, len(patchOp.Hunks))
			}
		}
	}

	return nil
}

// printResults prints successful application results.
func printResults(result *patchapply.ApplyResult, verbose bool) {
	fmt.Fprintln(os.Stdout, "✓ Applied patch successfully")

	printFileList("Created", result.FilesCreated, verbose)
	printFileList("Updated", result.FilesUpdated, verbose)
	printFileList("Deleted", result.FilesDeleted, verbose)
	printFileMovedList(result.FilesMoved, verbose)
}

// printFileList prints a summary and optional detail list of files.
func printFileList(label string, files []string, verbose bool) {
	if len(files) == 0 {
		return
	}

	fmt.Fprintf(os.Stdout, "  %s: %d files\n", label, len(files))

	if !verbose {
		return
	}

	for _, f := range files {
		fmt.Fprintf(os.Stdout, "    - %s\n", f)
	}
}

// printFileMovedList prints moved files summary and details.
func printFileMovedList(files map[string]string, verbose bool) {
	if len(files) == 0 {
		return
	}

	fmt.Fprintf(os.Stdout, "  Moved: %d files\n", len(files))

	if !verbose {
		return
	}

	for old, newPath := range files {
		fmt.Fprintf(os.Stdout, "    - %s → %s\n", old, newPath)
	}
}

// formatParseError formats parse errors with helpful hints.
func formatParseError(err error) error {
	return fmt.Errorf(`error: invalid patch syntax
%w

Hint: Check the patch format specification.
Expected format:
  *** Begin Patch
  *** Add File: path/to/file.txt
  +content line
  *** End Patch`, err)
}

// formatApplyError formats application errors with context.
func formatApplyError(err error) error {
	// Extract structured error if available.
	var applyErr *patchapply.Error
	if errors.As(err, &applyErr) {
		return fmt.Errorf(`error: failed to apply patch
  %s operation on %s (line %d)
  %w

Hint: %s`,
			applyErr.Op,
			applyErr.Path,
			applyErr.Line,
			applyErr.Err,
			getHintForError(applyErr))
	}

	return fmt.Errorf("error: %w", err)
}

// getHintForError provides helpful hints for common errors.
func getHintForError(err *patchapply.Error) string {
	switch {
	case errors.Is(err.Err, patchapply.ErrContextNotFound):
		return "The context may have changed. Update the patch with current file content."
	case errors.Is(err.Err, patchapply.ErrPathOutsideWorkspace):
		return "Use relative paths within the workspace only."
	case errors.Is(err.Err, patchapply.ErrFileExists):
		return "Use --force to overwrite existing files."
	case errors.Is(err.Err, patchapply.ErrFileNotFound):
		return "Ensure the file exists before updating."
	default:
		return "Check the error message above for details."
	}
}

// runApplyPatchMode is called from main.go for special execution modes.
func runApplyPatchMode() int {
	cmd := newApplyPatchCmd()
	cmd.Use = binaryApplyPatch
	cmd.Short = "Apply a Spin patch (standalone mode)"
	cmd.Long = `Apply a Spin patch from stdin or file.

This is the standalone version of the apply-patch command.
Can be invoked as:
  - spin-apply-patch (symlink)
  - spin apply-patch (subcommand)
  - spin --spin-run-as-apply-patch (internal)

Examples:
  # Apply from stdin
  cat changes.patch | spin-apply-patch

  # Apply from file
  spin-apply-patch -f changes.patch

  # Dry-run mode
  spin-apply-patch --dry-run -f changes.patch`

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)

		return 1
	}

	return 0
}
