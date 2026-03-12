package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestApprovalPersistence_SessionAndGlobalScopes validates the basic wiring
// of approval persistence configuration and CLI in an isolated workspace.
// Full ACP-driven shell flows are covered in dedicated ACP E2E suites.
func TestApprovalPersistence_SessionAndGlobalScopes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spin.yaml")
	policyPath := filepath.Join(tmpDir, "policies.json")

	cfg := `
version: "2.0"
security:
  policy_file: ` + policyPath + `
  approval_persistence_enabled: true
`

	err := writeFile(configPath, cfg)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	// For now, just assert that approval CLI works with the configured policy path.
	stdout, stderr, err := runSpin(t, "--config-file", configPath, "approval", "list", "--scope", "global")
	if err != nil {
		t.Fatalf("approval list failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "No policies found.") {
		t.Fatalf("expected initial list to be empty, got: %s", stdout)
	}
}
