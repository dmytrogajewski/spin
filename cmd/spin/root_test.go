package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()

	if cmd.Use != "spin" {
		t.Errorf("Root command Use = %s, want 'spin'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Root command Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Root command Long description should not be empty")
	}
}

func TestRootCommand_Help(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--help"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "spin") {
		t.Error("Help output should contain 'spin'")
	}

	if !strings.Contains(output, "Usage:") {
		t.Error("Help output should contain 'Usage:'")
	}
}

func TestGlobalFlags(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()

	// Check that global flags are registered.
	flags := []string{"model", "provider", "sandbox", "cd", "config", "config-file", "mode"}

	for _, flagName := range flags {
		flag := cmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Global flag --%s not found", flagName)
		}
	}
}

func TestGlobalFlags_Parsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		flagName  string
		flagValue string
	}{
		{
			name:      "model flag",
			flagName:  "model",
			flagValue: "llama3.1",
		},
		{
			name:      "provider flag",
			flagName:  "provider",
			flagValue: "ollama",
		},
		{
			name:      "sandbox flag",
			flagName:  "sandbox",
			flagValue: "workspace-write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := newRootCmd()

			// Verify the flag exists and can be set.
			flag := cmd.PersistentFlags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag --%s not found", tt.flagName)
			}

			// Verify parsing works without executing (avoids global variable races).
			err := cmd.PersistentFlags().Set(tt.flagName, tt.flagValue)
			if err != nil {
				t.Fatalf("failed to set flag --%s: %v", tt.flagName, err)
			}

			if !cmd.PersistentFlags().Changed(tt.flagName) {
				t.Errorf("flag --%s should be marked as changed after Set()", tt.flagName)
			}
		})
	}
}

func TestRootCommand_Version(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--version"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("--version failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "version") {
		t.Errorf("Version output should contain 'version', got: %s", output)
	}
}

func TestRootCommand_InvalidFlag(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--invalid-flag"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid flag, got nil")
	}

	errMsg := err.Error()

	output := out.String()
	if !strings.Contains(output, "unknown flag") && !strings.Contains(errMsg, "unknown flag") {
		t.Errorf("Error output should mention unknown flag, got output: %q, err: %q", output, errMsg)
	}
}

func TestConfigOverrides(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()

	// Verify the config flag exists and supports multiple values.
	flag := cmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("config flag not found")
	}

	if flag.Shorthand != "c" {
		t.Errorf("config flag shorthand = %q, want %q", flag.Shorthand, "c")
	}

	if flag.DefValue != "[]" {
		t.Errorf("config flag default = %q, want %q", flag.DefValue, "[]")
	}

	// Verify the flag type supports string slices.
	if flag.Value.Type() != "stringSlice" {
		t.Errorf("config flag type = %q, want %q", flag.Value.Type(), "stringSlice")
	}
}

// TestTaskModeFlag tests the --mode flag functionality.
func TestTaskModeFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		flagName  string
		shorthand string
		value     string
	}{
		{
			name:     "explicit regular mode",
			flagName: "mode",
			value:    "regular",
		},
		{
			name:     "review mode",
			flagName: "mode",
			value:    "review",
		},
		{
			name:     "compact mode",
			flagName: "mode",
			value:    "compact",
		},
		{
			name:     "planning mode",
			flagName: "mode",
			value:    "planning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := newRootCmd()

			// Verify the mode flag can be set without executing (avoids global variable races).
			err := cmd.PersistentFlags().Set(tt.flagName, tt.value)
			if err != nil {
				t.Fatalf("failed to set flag --%s=%s: %v", tt.flagName, tt.value, err)
			}

			if !cmd.PersistentFlags().Changed(tt.flagName) {
				t.Errorf("flag --%s should be marked as changed", tt.flagName)
			}
		})
	}
}

// TestTaskModeFlagHelp verifies the mode flag appears in help output.
func TestTaskModeFlagHelp(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"--help"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := out.String()

	// Check that mode flag is documented.
	if !strings.Contains(output, "--mode") && !strings.Contains(output, "-m") {
		t.Error("Help output should contain '--mode' or '-m' flag")
	}

	// Check that valid modes are mentioned in help.
	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		if !strings.Contains(output, mode) {
			t.Errorf("Help output should mention mode %q", mode)
		}
	}
}

// TestTaskModeFlagDefault verifies the default value.
func TestTaskModeFlagDefault(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()

	flag := cmd.PersistentFlags().Lookup("mode")
	if flag == nil {
		t.Fatal("--mode flag not found")
	}

	if flag.DefValue != "regular" {
		t.Errorf("mode flag default = %q, want %q", flag.DefValue, "regular")
	}
}

// TestTaskModeFlagShorthand verifies the short form.
func TestTaskModeFlagShorthand(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()

	flag := cmd.PersistentFlags().Lookup("mode")
	if flag == nil {
		t.Fatal("--mode flag not found")
	}

	if flag.Shorthand != "m" {
		t.Errorf("mode flag shorthand = %q, want %q", flag.Shorthand, "m")
	}
}
