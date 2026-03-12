package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApprovalCLI_ListAndClear_Empty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spin.yaml")
	policyPath := filepath.Join(tmpDir, "policies.json")

	cfg := "version: \"2.0\"\nsecurity:\n  policy_file: " + policyPath + "\n"
	err := writeFile(configPath, cfg)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	// list should say no policies.
	stdout, stderr, err := runSpin(t, "--config-file", configPath, "approval", "list", "--scope", "global")
	if err != nil {
		t.Fatalf("approval list failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "No policies found.") {
		t.Fatalf("expected no policies, got: %s", stdout)
	}

	// clear should clear 0.
	stdout, stderr, err = runSpin(t, "--config-file", configPath, "approval", "clear", "--scope", "global")
	if err != nil {
		t.Fatalf("approval clear failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Cleared 0 policies.") {
		t.Fatalf("expected cleared 0, got: %s", stdout)
	}
}

func TestApprovalCLI_Revoke_NonExistent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spin.yaml")
	policyPath := filepath.Join(tmpDir, "policies.json")

	cfg := "version: \"2.0\"\nsecurity:\n  policy_file: " + policyPath + "\n"
	err := writeFile(configPath, cfg)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout, stderr, err := runSpin(t,
		"--config-file", configPath,
		"approval", "revoke",
		"--scope", "global",
		"--program", "/bin/echo",
		"--arg", "hello",
		"--workdir", tmpDir,
	)
	if err != nil {
		t.Fatalf("approval revoke failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "No matching policy found.") {
		t.Fatalf("expected no match message, got: %s", stdout)
	}
}

// writeFile is a tiny helper to reduce boilerplate in e2e tests.
func writeFile(path, content string) error {
	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}

	return nil
}
