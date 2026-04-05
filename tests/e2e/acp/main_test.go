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
	if buildErr := doBuildSpinBinary(); buildErr != nil {
		fmt.Fprintf(os.Stderr, "%v\n", buildErr)
		os.Exit(1)
	}
}

func doBuildSpinBinary() error {
	// Use file lock to prevent concurrent builds from parallel test packages.
	lockPath := binPath + ".lock"

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	defer lockFile.Close()

	// Acquire exclusive lock (blocks until available).
	fd := int(lockFile.Fd())
	if flockErr := syscall.Flock(fd, syscall.LOCK_EX); flockErr != nil {
		return fmt.Errorf("failed to acquire build lock: %w", flockErr)
	}

	defer func() { _ = syscall.Flock(fd, syscall.LOCK_UN) }()

	// Check if binary was already built by another package while we waited.
	// Use 5-minute window to avoid unnecessary rebuilds during parallel test runs.
	if info, statErr := os.Stat(binPath); statErr == nil {
		if time.Since(info.ModTime()) < 5*time.Minute {
			fmt.Fprintln(os.Stdout, "Using recently built spin binary for ACP e2e tests")

			return nil
		}
	}

	// Build to a temp file and atomically rename to avoid running a half-written binary.
	fmt.Fprintln(os.Stdout, "Building spin binary for ACP e2e tests (with e2e_llm_test tag)...")

	tmpBin := binPath + ".tmp"

	cmd := exec.CommandContext(context.Background(), "go", "build", "-tags", "e2e_llm_test", "-o", tmpBin, "../../../cmd/spin")

	output, buildErr := cmd.CombinedOutput()
	if buildErr != nil {
		os.Remove(tmpBin)

		return fmt.Errorf("failed to build binary: %w\n%s", buildErr, output)
	}

	if renameErr := os.Rename(tmpBin, binPath); renameErr != nil {
		os.Remove(tmpBin)

		return fmt.Errorf("failed to install binary: %w", renameErr)
	}

	return nil
}
