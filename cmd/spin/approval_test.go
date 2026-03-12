package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/security"
)

func TestApproval_List_Empty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spin.yaml")

	policyPath := filepath.Join(tmpDir, "policies.json")
	err := os.WriteFile(configPath, []byte("version: \"2.0\"\nsecurity:\n  policy_file: "+policyPath+"\n"), 0644)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	root := newRootCmd()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config-file", configPath, "approval", "list", "--scope", "global"})

	err = root.Execute()
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !strings.Contains(out.String(), "No policies found.") {
		t.Fatalf("expected 'No policies found.' got: %s", out.String())
	}
}

func TestApproval_List_WithData(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spin.yaml")

	policyPath := filepath.Join(tmpDir, "policies.json")
	err := os.WriteFile(configPath, []byte("version: \"2.0\"\nsecurity:\n  policy_file: "+policyPath+"\n  approval_persistence_enabled: true\n"), 0644)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	store, err := security.NewFilePolicyStore(policyPath, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ctx := context.Background()
	key := security.NewPolicyKey("/bin/echo", []string{"hello"}, tmpDir)

	p := security.Policy{
		Version:   "1",
		Scope:     security.ScopeGlobal,
		Key:       key,
		Decision:  security.DecisionAllow,
		CreatedAt: time.Now(),
	}
	err = store.Save(ctx, p)
	if err != nil {
		t.Fatalf("save policy: %v", err)
	}

	root := newRootCmd()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config-file", configPath, "approval", "list", "--scope", "global"})

	err = root.Execute()
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "[global] /bin/echo hello") {
		t.Fatalf("expected list output to include policy, got: %s", got)
	}
}

func TestApproval_Revoke_NonExistent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spin.yaml")

	policyPath := filepath.Join(tmpDir, "policies.json")
	err := os.WriteFile(configPath, []byte("version: \"2.0\"\nsecurity:\n  policy_file: "+policyPath+"\n"), 0644)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	root := newRootCmd()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{
		"--config-file", configPath,
		"approval", "revoke",
		"--scope", "global",
		"--program", "/bin/echo",
		"--arg", "hello",
		"--workdir", tmpDir,
	})

	err = root.Execute()
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !strings.Contains(out.String(), "No matching policy found.") {
		t.Fatalf("expected no match message, got: %s", out.String())
	}
}

func TestApproval_Clear_Empty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spin.yaml")

	policyPath := filepath.Join(tmpDir, "policies.json")
	err := os.WriteFile(configPath, []byte("version: \"2.0\"\nsecurity:\n  policy_file: "+policyPath+"\n"), 0644)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	root := newRootCmd()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config-file", configPath, "approval", "clear", "--scope", "global"})

	err = root.Execute()
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !strings.Contains(out.String(), "Cleared 0 policies.") {
		t.Fatalf("expected cleared 0, got: %s", out.String())
	}
}
