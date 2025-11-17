package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
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
	cmd := newRootCmd()

	// Check that global flags are registered
	flags := []string{"model", "provider", "sandbox", "cd", "config", "config-file", "mode"}

	for _, flagName := range flags {
		flag := cmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Global flag --%s not found", flagName)
		}
	}
}

func TestGlobalFlags_Parsing(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantModel    string
		wantProvider string
		wantSandbox  string
	}{
		{
			name:         "model flag",
			args:         []string{"--model", "llama3.1", "--help"},
			wantModel:    "llama3.1",
			wantProvider: "",
			wantSandbox:  "",
		},
		{
			name:         "provider flag",
			args:         []string{"--provider", "ollama", "--help"},
			wantModel:    "",
			wantProvider: "ollama",
			wantSandbox:  "",
		},
		{
			name:         "sandbox flag",
			args:         []string{"--sandbox", "workspace-write", "--help"},
			wantModel:    "",
			wantProvider: "",
			wantSandbox:  "workspace-write",
		},
		{
			name:         "multiple flags",
			args:         []string{"--model", "mixtral", "--provider", "lmstudio", "--sandbox", "read-only", "--help"},
			wantModel:    "mixtral",
			wantProvider: "lmstudio",
			wantSandbox:  "read-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRootCmd()
			cmd.SetArgs(tt.args)

			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			// Execute should not error on --help
			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			// Check flag values
			if tt.wantModel != "" {
				model, _ := cmd.Flags().GetString("model")
				if model != tt.wantModel {
					t.Errorf("model = %s, want %s", model, tt.wantModel)
				}
			}

			if tt.wantProvider != "" {
				provider, _ := cmd.Flags().GetString("provider")
				if provider != tt.wantProvider {
					t.Errorf("provider = %s, want %s", provider, tt.wantProvider)
				}
			}

			if tt.wantSandbox != "" {
				sandbox, _ := cmd.Flags().GetString("sandbox")
				if sandbox != tt.wantSandbox {
					t.Errorf("sandbox = %s, want %s", sandbox, tt.wantSandbox)
				}
			}
		})
	}
}

func TestRootCommand_Version(t *testing.T) {
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
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--invalid-flag"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid flag, got nil")
	}

	output := out.String()
	if !strings.Contains(output, "unknown flag") {
		t.Errorf("Error output should mention unknown flag, got: %s", output)
	}
}

func TestConfigOverrides(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--config", "key1=value1", "--config", "key2=value2", "--help"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	config, _ := cmd.Flags().GetStringSlice("config")
	if len(config) != 2 {
		t.Errorf("config overrides count = %d, want 2", len(config))
	}
	if config[0] != "key1=value1" {
		t.Errorf("config[0] = %s, want 'key1=value1'", config[0])
	}
	if config[1] != "key2=value2" {
		t.Errorf("config[1] = %s, want 'key2=value2'", config[1])
	}
}

// TestTaskModeFlag tests the --mode flag functionality
func TestTaskModeFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantMode string
	}{
		{
			name:     "default mode",
			args:     []string{"--help"},
			wantMode: "regular",
		},
		{
			name:     "explicit regular mode",
			args:     []string{"--mode", "regular", "--help"},
			wantMode: "regular",
		},
		{
			name:     "review mode",
			args:     []string{"--mode", "review", "--help"},
			wantMode: "review",
		},
		{
			name:     "compact mode",
			args:     []string{"--mode", "compact", "--help"},
			wantMode: "compact",
		},
		{
			name:     "planning mode",
			args:     []string{"--mode", "planning", "--help"},
			wantMode: "planning",
		},
		{
			name:     "short flag",
			args:     []string{"-m", "compact", "--help"},
			wantMode: "compact",
		},
		{
			name:     "short flag review",
			args:     []string{"-m", "review", "--help"},
			wantMode: "review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flagTaskMode to default
			flagTaskMode = "regular"

			cmd := newRootCmd()
			cmd.SetArgs(tt.args)

			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			// Execute should not error with --help
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v, output: %s", err, out.String())
			}

			// Check flag value
			mode, err := cmd.Flags().GetString("mode")
			if err != nil {
				t.Fatalf("Failed to get mode flag: %v", err)
			}

			if mode != tt.wantMode {
				t.Errorf("mode flag = %q, want %q", mode, tt.wantMode)
			}
		})
	}
}

// TestTaskModeFlagHelp verifies the mode flag appears in help output
func TestTaskModeFlagHelp(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--help"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := out.String()

	// Check that mode flag is documented
	if !strings.Contains(output, "--mode") && !strings.Contains(output, "-m") {
		t.Error("Help output should contain '--mode' or '-m' flag")
	}

	// Check that valid modes are mentioned in help
	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		if !strings.Contains(output, mode) {
			t.Errorf("Help output should mention mode %q", mode)
		}
	}
}

// TestTaskModeFlagDefault verifies the default value
func TestTaskModeFlagDefault(t *testing.T) {
	cmd := newRootCmd()

	flag := cmd.PersistentFlags().Lookup("mode")
	if flag == nil {
		t.Fatal("--mode flag not found")
	}

	if flag.DefValue != "regular" {
		t.Errorf("mode flag default = %q, want %q", flag.DefValue, "regular")
	}
}

// TestTaskModeFlagShorthand verifies the short form
func TestTaskModeFlagShorthand(t *testing.T) {
	cmd := newRootCmd()

	flag := cmd.PersistentFlags().Lookup("mode")
	if flag == nil {
		t.Fatal("--mode flag not found")
	}

	if flag.Shorthand != "m" {
		t.Errorf("mode flag shorthand = %q, want %q", flag.Shorthand, "m")
	}
}
