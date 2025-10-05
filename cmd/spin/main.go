package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Check for special binary names (symlinks)
	arg0 := os.Args[0]
	baseName := filepath.Base(arg0)

	switch baseName {
	case "spin-apply-patch":
		os.Exit(runApplyPatchMode())
	case "spin-sandbox":
		os.Exit(runSandboxMode())
	}

	// Check for internal flags (used for subprocess execution)
	for _, arg := range os.Args[1:] {
		if arg == "--spin-run-as-apply-patch" {
			os.Exit(runApplyPatchMode())
		}
		if arg == "--spin-run-as-sandbox" {
			os.Exit(runSandboxMode())
		}
	}

	// Normal CLI execution
	if err := execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// execute runs the root command.
func execute() error {
	cmd := newRootCmd()
	return cmd.Execute()
}

// runApplyPatchMode runs the apply-patch special mode.
// This will be implemented when patch functionality is added.
func runApplyPatchMode() int {
	fmt.Fprintln(os.Stderr, "Apply patch mode not yet implemented")
	return 1
}

// runSandboxMode runs the sandbox test mode.
// This will be implemented when sandbox functionality is added.
func runSandboxMode() int {
	fmt.Fprintln(os.Stderr, "Sandbox mode not yet implemented")
	return 1
}
