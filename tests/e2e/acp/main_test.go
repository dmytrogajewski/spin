package acp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
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
	fmt.Fprintln(os.Stdout, "Building spin binary for ACP e2e tests (with e2e_llm_test tag)...")

	cmd := exec.CommandContext(context.Background(), "go", "build", "-tags", "e2e_llm_test", "-o", binPath, "../../../cmd/spin")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary: %v\n%s\n", err, output)
		os.Exit(1)
	}
}
