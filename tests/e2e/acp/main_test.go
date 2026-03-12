package acp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

const skipBuildEnv = "SPIN_E2E_SKIP_BUILD"

func TestMain(m *testing.M) {
	if shouldSkipBuild() {
		_, err := os.Stat(binPath)
		if err == nil {
			fmt.Fprintln(os.Stdout, "Using existing spin binary for ACP e2e tests")
		} else {
			fmt.Fprintln(os.Stdout, "Pre-built spin binary not found, rebuilding...")
			buildSpinBinary()
		}
	} else {
		buildSpinBinary()
	}

	os.Exit(m.Run())
}

func shouldSkipBuild() bool {
	return os.Getenv(skipBuildEnv) == "1"
}

func buildSpinBinary() {
	// Use file lock to prevent concurrent builds from parallel test packages.
	lockPath := binPath + ".lock"

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create lock file: %v\n", err)
		os.Exit(1)
	}
	defer lockFile.Close()

	// Acquire exclusive lock (blocks until available).
	fd := int(lockFile.Fd()) //nolint:gosec // fd is always a small positive int
	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to acquire build lock: %v\n", err)
		os.Exit(1)
	}

	defer func() { _ = syscall.Flock(fd, syscall.LOCK_UN) }()

	// Check if binary was already built by another package while we waited.
	if info, statErr := os.Stat(binPath); statErr == nil {
		if time.Since(info.ModTime()) < 30*time.Second {
			fmt.Fprintln(os.Stdout, "Using recently built spin binary for ACP e2e tests")

			return
		}
	}

	fmt.Fprintln(os.Stdout, "Building spin binary for ACP e2e tests (with e2e_llm_test tag)...")

	cmd := exec.CommandContext(context.Background(), "go", "build", "-tags", "e2e_llm_test", "-o", binPath, "../../../cmd/spin")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary: %v\n%s\n", err, output)
		os.Exit(1)
	}
}
