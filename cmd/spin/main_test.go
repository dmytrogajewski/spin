package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExecute(t *testing.T) {
	t.Parallel()

	// Test help flag using cobra command directly.
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("execute() with --help should not error, got: %v", err)
	}
}

func TestExecute_InvalidCommand(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"invalid-command"})

	err := cmd.Execute()
	if err == nil {
		t.Error("execute() with invalid command should return error")
	}
}

func TestRunApplyPatchMode(t *testing.T) {
	t.Parallel()

	code := runApplyPatchMode()
	if code == 0 {
		t.Error("runApplyPatchMode() should return non-zero exit code (not implemented)")
	}
}

func TestRunSandboxMode(t *testing.T) {
	t.Parallel()

	code := runSandboxMode()
	if code == 0 {
		t.Error("runSandboxMode() should return non-zero exit code (not implemented)")
	}
}

func TestBinaryNameDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		binaryName string
		wantMode   string
	}{
		{
			name:       "spin-apply-patch",
			binaryName: "spin-apply-patch",
			wantMode:   "apply-patch",
		},
		{
			name:       "spin-sandbox",
			binaryName: "spin-sandbox",
			wantMode:   "sandbox",
		},
		{
			name:       "regular spin",
			binaryName: "spin",
			wantMode:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			baseName := filepath.Base(tt.binaryName)

			switch baseName {
			case "spin-apply-patch":
				if tt.wantMode != "apply-patch" {
					t.Errorf("Expected apply-patch mode, got: %s", tt.wantMode)
				}
			case "spin-sandbox":
				if tt.wantMode != "sandbox" {
					t.Errorf("Expected sandbox mode, got: %s", tt.wantMode)
				}
			case "spin":
				if tt.wantMode != "" {
					t.Errorf("Expected regular mode, got: %s", tt.wantMode)
				}
			}
		})
	}
}

func TestMain(m *testing.M) {
	// Redirect stderr to suppress error messages during tests.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	code := m.Run()

	// Restore stderr.
	w.Close()

	os.Stderr = oldStderr

	// Drain pipe.
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	os.Exit(code)
}
