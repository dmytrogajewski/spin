package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	cmd := newVersionCmd()

	if cmd.Use != "version" {
		t.Errorf("Version command Use = %s, want 'version'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Version command Short description should not be empty")
	}
}

func TestVersionCommand_Execute(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.SetArgs([]string{"version"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "version") {
		t.Errorf("Version output should contain 'version', got: %s", output)
	}
}

func TestVersionCommand_WithVerbose(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.SetArgs([]string{"version", "--verbose"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version --verbose failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "version") {
		t.Error("Version output should contain 'version'")
	}

	if !strings.Contains(output, "commit") {
		t.Error("Verbose version output should contain 'commit'")
	}

	if !strings.Contains(output, "built") {
		t.Error("Verbose version output should contain 'built'")
	}
}

func TestVersionCommand_Short(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.SetArgs([]string{"version", "--short"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version --short failed: %v", err)
	}

	output := strings.TrimSpace(out.String())
	// Should just be the version number.
	if strings.Contains(output, "commit") || strings.Contains(output, "built") {
		t.Errorf("Short version should not contain commit or build info, got: %s", output)
	}
}
