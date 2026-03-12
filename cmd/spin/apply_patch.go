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

// applyPatchCmd represents the apply-patch subcommand.
var applyPatchCmd = &cobra.Command{
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

// Flags.
var (
	applyPatchFile      string
	applyPatchWorkspace string
	applyPatchDryRun    bool
	applyPatchForce     bool
	applyPatchVerbose   bool
)

func init() {
	applyPatchCmd.Flags().StringVarP(&applyPatchFile, "file", "f", "", "Patch file (default: stdin)")
	applyPatchCmd.Flags().StringVarP(&applyPatchWorkspace, "workspace", "w", ".", "Workspace directory")
	applyPatchCmd.Flags().BoolVar(&applyPatchDryRun, "dry-run", false, "Validate without applying")
	applyPatchCmd.Flags().BoolVar(&applyPatchForce, "force", false, "Force overwrite existing files")
	applyPatchCmd.Flags().BoolVarP(&applyPatchVerbose, "verbose", "v", false, "Verbose output")
}

// newApplyPatchCmd returns the apply-patch command for inclusion in root.
func newApplyPatchCmd() *cobra.Command {
	return applyPatchCmd
}

// runApplyPatch executes the apply-patch command.
func runApplyPatch(_ *cobra.Command, _ []string) error {
	// Read patch text.
	patchText, err := readPatchInput()
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
	workspace, err := filepath.Abs(applyPatchWorkspace)
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
	}

	// Create applier.
	applier, err := patchapply.NewApplier(workspace)
	if err != nil {
		return fmt.Errorf("create applier: %w", err)
	}

	// Configure applier.
	applier.SetDryRun(applyPatchDryRun)
	applier.SetForceOverwrite(applyPatchForce)

	// Apply or validate.
	if applyPatchDryRun {
		return runDryRun(applier, patch)
	}

	result, err := applier.Apply(patch)
	if err != nil {
		return formatApplyError(err)
	}

	// Output results.
	printResults(result)

	return nil
}

// readPatchInput reads patch from file or stdin.
func readPatchInput() (string, error) {
	var reader io.Reader

	if applyPatchFile != "" {
		f, err := os.Open(applyPatchFile)
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
		switch v := op.(type) {
		case *patchapply.AddFile:
			fmt.Fprintf(os.Stdout, "  Would create: %s (%d lines)\n", v.FilePath, len(v.Lines))
		case *patchapply.DeleteFile:
			fmt.Fprintf(os.Stdout, "  Would delete: %s\n", v.FilePath)
		case *patchapply.UpdateFile:
			if v.NewPath != "" {
				fmt.Fprintf(os.Stdout, "  Would move: %s → %s (%d hunks)\n", v.FilePath, v.NewPath, len(v.Hunks))
			} else {
				fmt.Fprintf(os.Stdout, "  Would update: %s (%d hunks)\n", v.FilePath, len(v.Hunks))
			}
		}
	}

	return nil
}

// printResults prints successful application results.
func printResults(result *patchapply.ApplyResult) {
	fmt.Fprintln(os.Stdout, "✓ Applied patch successfully")

	if len(result.FilesCreated) > 0 {
		fmt.Fprintf(os.Stdout, "  Created: %d files\n", len(result.FilesCreated))

		if applyPatchVerbose {
			for _, f := range result.FilesCreated {
				fmt.Fprintf(os.Stdout, "    - %s\n", f)
			}
		}
	}

	if len(result.FilesUpdated) > 0 {
		fmt.Fprintf(os.Stdout, "  Updated: %d files\n", len(result.FilesUpdated))

		if applyPatchVerbose {
			for _, f := range result.FilesUpdated {
				fmt.Fprintf(os.Stdout, "    - %s\n", f)
			}
		}
	}

	if len(result.FilesDeleted) > 0 {
		fmt.Fprintf(os.Stdout, "  Deleted: %d files\n", len(result.FilesDeleted))

		if applyPatchVerbose {
			for _, f := range result.FilesDeleted {
				fmt.Fprintf(os.Stdout, "    - %s\n", f)
			}
		}
	}

	if len(result.FilesMoved) > 0 {
		fmt.Fprintf(os.Stdout, "  Moved: %d files\n", len(result.FilesMoved))

		if applyPatchVerbose {
			for old, new := range result.FilesMoved {
				fmt.Fprintf(os.Stdout, "    - %s → %s\n", old, new)
			}
		}
	}
}

// formatParseError formats parse errors with helpful hints.
func formatParseError(err error) error {
	return fmt.Errorf(`Error: Invalid patch syntax
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
		return fmt.Errorf(`Error: Failed to apply patch
  %s operation on %s (line %d)
  %w

Hint: %s`,
			applyErr.Op,
			applyErr.Path,
			applyErr.Line,
			applyErr.Err,
			getHintForError(applyErr))
	}

	return fmt.Errorf("Error: %w", err)
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
	// Create a minimal cobra command for special mode.
	cmd := &cobra.Command{
		Use:   "spin-apply-patch",
		Short: "Apply a Spin patch (standalone mode)",
		Long: `Apply a Spin patch from stdin or file.

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
  spin-apply-patch --dry-run -f changes.patch`,
		RunE: runApplyPatch,
	}

	// Add flags.
	cmd.Flags().StringVarP(&applyPatchFile, "file", "f", "", "Patch file (default: stdin)")
	cmd.Flags().StringVarP(&applyPatchWorkspace, "workspace", "w", ".", "Workspace directory")
	cmd.Flags().BoolVar(&applyPatchDryRun, "dry-run", false, "Validate without applying")
	cmd.Flags().BoolVar(&applyPatchForce, "force", false, "Force overwrite existing files")
	cmd.Flags().BoolVarP(&applyPatchVerbose, "verbose", "v", false, "Verbose output")

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)

		return 1
	}

	return 0
}
