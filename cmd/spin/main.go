package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Check for special binary names (symlinks)
	if exitCode := handleSpecialBinaryName(); exitCode >= 0 {
		os.Exit(exitCode)
	}

	// Check for internal flags (used for subprocess execution)
	if exitCode := handleInternalFlags(); exitCode >= 0 {
		os.Exit(exitCode)
	}

	// Normal CLI execution
	if err := execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// handleSpecialBinaryName checks for special binary names and returns exit code or -1.
func handleSpecialBinaryName() int {
	arg0 := os.Args[0]
	baseName := filepath.Base(arg0)

	switch baseName {
	case "spin-apply-patch":
		return runApplyPatchMode()
	case "spin-sandbox":
		return runSandboxMode()
	}
	return -1
}

// handleInternalFlags checks for internal flags and returns exit code or -1.
func handleInternalFlags() int {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--spin-run-as-apply-patch":
			return runApplyPatchMode()
		case "--spin-run-as-sandbox":
			return runSandboxMode()
		}
	}
	return -1
}

// execute runs the root command.
func execute() error {
	cmd := newRootCmd()
	return cmd.Execute()
}

// runApplyPatchMode is now implemented in apply_patch.go

// runSandboxMode runs the sandbox test mode.
// This will be implemented when sandbox functionality is added.
func runSandboxMode() int {
	fmt.Fprintln(os.Stderr, "Sandbox mode not yet implemented")
	return 1
}
