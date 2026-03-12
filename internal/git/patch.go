package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ApplyPatch applies a git patch to the repository.
func (r *Repository) ApplyPatch(ctx context.Context, patchText string, opts ApplyPatchOptions) (*ApplyPatchResult, error) {
	// Check context.
	err := ctx.Err()
	if err != nil {
		return nil, fmt.Errorf("apply patch: %w", err)
	}

	// Handle empty patch.
	if strings.TrimSpace(patchText) == "" {
		return &ApplyPatchResult{
			Success: true,
			Message: "Empty patch - nothing to apply",
		}, nil
	}

	// Build git apply command.
	args := []string{"apply"}
	if opts.DryRun {
		args = append(args, "--check")
	}

	args = append(args, "-")

	// Create command.
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.root
	cmd.Stdin = strings.NewReader(patchText)

	// Capture output.
	var stdout, stderr strings.Builder

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command.
	err = cmd.Run()
	if err != nil {
		// Check if it's a context error.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("apply patch context canceled: %w", ctx.Err())
		}

		// Patch failed to apply.
		return &ApplyPatchResult{
			Success: false,
			Message: fmt.Sprintf("failed to apply patch: %s", stderr.String()),
		}, fmt.Errorf("git apply failed: %w", err)
	}

	// Success.
	message := "patch applied successfully"
	if opts.DryRun {
		message = "patch would apply successfully (dry-run)"
	}

	return &ApplyPatchResult{
		Success: true,
		Message: message,
	}, nil
}
