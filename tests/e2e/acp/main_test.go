package acp

import (
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
			fmt.Println("Using existing spin binary for ACP e2e tests")
		} else {
			fmt.Println("Pre-built spin binary not found, rebuilding...")
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
	fmt.Println("Building spin binary for ACP e2e tests (with e2e_llm_test tag)...")

	cmd := exec.Command("go", "build", "-tags", "e2e_llm_test", "-o", binPath, "../../../cmd/spin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to build binary: %v\n%s\n", err, output)
		os.Exit(1)
	}
}
