package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// configShowTestCase defines a test case for config show.
type configShowTestCase struct {
	name       string
	configYAML string
	format     string
	wantErr    bool
	wantOutput []string
}

// runConfigShowTestCase runs a single config show test case.
func runConfigShowTestCase(t *testing.T, tt configShowTestCase) {
	t.Helper()
	t.Parallel()

	tmpDir := t.TempDir()

	configFile := filepath.Join(tmpDir, "spin.yaml")

	err := os.WriteFile(configFile, []byte(tt.configYAML), 0o600)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	var stdout, stderr bytes.Buffer

	root := newRootCmd()
	root.SetArgs([]string{"--config-file", configFile, "config", "show", "--format", tt.format})
	root.SetOut(&stdout)
	root.SetErr(&stderr)

	err = root.Execute()

	if (err != nil) != tt.wantErr {
		t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)

		return
	}

	if !tt.wantErr {
		output := stdout.String()
		for _, want := range tt.wantOutput {
			if !strings.Contains(output, want) {
				t.Errorf("Output missing expected string %q\nGot: %s", want, output)
			}
		}
	}
}

func TestConfigShow(t *testing.T) {
	t.Parallel()

	tests := []configShowTestCase{
		{
			name:       "text format with valid config",
			configYAML: "version: \"2.0\"\nllm:\n  provider: openai\n  model: gpt-4o\nsecurity:\n  sandbox_mode: workspace-only\n",
			format:     "text",
			wantErr:    false,
			wantOutput: []string{"llm:", "provider: openai", "model: gpt-4o", "security:"},
		},
		{
			name:       "json format with valid config",
			configYAML: "version: \"2.0\"\nllm:\n  provider: openai\n  model: gpt-4o\n",
			format:     "json",
			wantErr:    false,
			wantOutput: []string{`"LLM"`, `"Provider"`, `"openai"`},
		},
		{
			name:       "yaml format with valid config",
			configYAML: "version: \"2.0\"\nllm:\n  provider: openai\n  model: gpt-4o\n",
			format:     "yaml",
			wantErr:    false,
			wantOutput: []string{"llm:", "provider: openai"},
		},
		{
			name:       "redacts sensitive values",
			configYAML: "version: \"2.0\"\nllm:\n  provider: openai\n  model: gpt-4o\n  api_key: sk-secret123\n",
			format:     "text",
			wantErr:    false,
			wantOutput: []string{"api_key:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runConfigShowTestCase(t, tt)
		})
	}
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configYAML string
		wantErr    bool
		wantOutput []string
	}{
		{
			name: "valid config",
			configYAML: `version: "2.0"
llm:
  provider: openai
  model: gpt-4o
  base_url: https://api.openai.com/v1
  timeout: 60s
security:
  sandbox_mode: workspace-only
`,
			wantErr: false,
			wantOutput: []string{
				"✓ Configuration V2 is valid",
			},
		},
		{
			name: "invalid yaml syntax",
			configYAML: `version: "2.0"
llm:
  provider: openai
  model: [unclosed bracket
`,
			wantErr: true,
			wantOutput: []string{
				"invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp config file.
			tmpDir := t.TempDir()

			configFile := filepath.Join(tmpDir, "spin.yaml")

			err := os.WriteFile(configFile, []byte(tt.configYAML), 0o600)
			if err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			// Capture output.
			var stdout, stderr bytes.Buffer

			// Create root command with config subcommand.
			root := newRootCmd()
			root.SetArgs([]string{"--config-file", configFile, "config", "validate"})
			root.SetOut(&stdout)
			root.SetErr(&stderr)

			// Execute.
			err = root.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v\nStdout: %s\nStderr: %s",
					err, tt.wantErr, stdout.String(), stderr.String())

				return
			}

			output := stdout.String() + stderr.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing expected string %q\nGot: %s", want, output)
				}
			}
		})
	}
}

// configPathTestCase defines a test case for config path.
type configPathTestCase struct {
	name       string
	setupFile  bool
	wantErr    bool
	wantOutput []string
}

// runConfigPathTestCase runs a single config path test case.
func runConfigPathTestCase(t *testing.T, tt configPathTestCase) {
	t.Helper()
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	if tt.setupFile {
		configYAML := "version: \"2.0\"\nllm:\n  provider: openai\n  model: gpt-4o\n"

		err := os.WriteFile(configFile, []byte(configYAML), 0o600)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}
	}

	var stdout, stderr bytes.Buffer

	cfgPath := configFile
	if !tt.setupFile {
		cfgPath = filepath.Join(tmpDir, "nonexistent.yaml")
	}

	root := newRootCmd()
	root.SetArgs([]string{"--config-file", cfgPath, "config", "path"})
	root.SetOut(&stdout)
	root.SetErr(&stderr)

	err := root.Execute()

	if (err != nil) != tt.wantErr {
		t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)

		return
	}

	output := stdout.String() + stderr.String()
	for _, want := range tt.wantOutput {
		if !strings.Contains(output, want) {
			t.Errorf("Output missing expected string %q\nGot: %s", want, output)
		}
	}
}

func TestConfigPath(t *testing.T) {
	t.Parallel()

	tests := []configPathTestCase{
		{
			name:       "config file exists",
			setupFile:  true,
			wantErr:    false,
			wantOutput: []string{"spin.yaml"},
		},
		{
			name:       "config file not found",
			setupFile:  false,
			wantErr:    true,
			wantOutput: []string{"No configuration file found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runConfigPathTestCase(t, tt)
		})
	}
}

func TestConfigPathShowAll(t *testing.T) {
	t.Parallel()

	// Create temp dir.
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "nonexistent.yaml")

	// Capture output.
	var stdout, stderr bytes.Buffer

	// Create root command with config subcommand.
	root := newRootCmd()
	root.SetArgs([]string{"--config-file", configFile, "config", "path", "--all"})
	root.SetOut(&stdout)
	root.SetErr(&stderr)

	// Execute.
	err := root.Execute()
	if err == nil {
		t.Error("Expected error for nonexistent config, got nil")
	}

	output := stdout.String() + stderr.String()
	expectedStrings := []string{
		"No configuration file found",
		"Search paths:",
	}

	for _, want := range expectedStrings {
		if !strings.Contains(output, want) {
			t.Errorf("Output missing expected string %q\nGot: %s", want, output)
		}
	}
}

func TestConfigEdit(t *testing.T) {
	t.Run("fails when no editor found", func(t *testing.T) {
		// Clear all editor env vars (cannot use t.Parallel — modifies process-global env).
		t.Setenv("EDITOR", "")
		t.Setenv("VISUAL", "")
		t.Setenv("PATH", "/nonexistent") // Make sure no editors in path.

		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "spin.yaml")

		// Create minimal config.
		configYAML := `version: "2.0"
llm:
  provider: openai
  model: gpt-4o
`

		err := os.WriteFile(configFile, []byte(configYAML), 0o600)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		// Create root command with config subcommand.
		root := newRootCmd()
		root.SetArgs([]string{"--config-file", configFile, "config", "edit"})

		// Execute - should fail with no editor.
		err = root.Execute()
		if err == nil {
			t.Error("Expected error when no editor found, got nil")
		}

		if !strings.Contains(err.Error(), "editor") {
			t.Errorf("Error should mention editor, got: %v", err)
		}
	})
}

func TestGetEditor(t *testing.T) {
	// Cannot use t.Parallel — subtests modify process-global env vars.
	tests := []struct {
		name         string
		editorEnv    string
		visualEnv    string
		wantNonEmpty bool
	}{
		{
			name:         "uses EDITOR",
			editorEnv:    "vim",
			wantNonEmpty: true,
		},
		{
			name:         "uses VISUAL when EDITOR not set",
			visualEnv:    "emacs",
			wantNonEmpty: true,
		},
		{
			name:         "falls back to vi",
			wantNonEmpty: true, // vi should exist on most systems.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test env vars (t.Setenv handles save/restore automatically).
			t.Setenv("EDITOR", tt.editorEnv)
			t.Setenv("VISUAL", tt.visualEnv)

			editor := getEditor()

			if tt.wantNonEmpty && editor == "" {
				t.Error("getEditor() returned empty, want non-empty")
			}
		})
	}
}

func TestGetConfigSearchPaths(t *testing.T) {
	t.Parallel()

	paths := getConfigSearchPaths()

	if len(paths) == 0 {
		t.Error("getConfigSearchPaths() returned empty slice")
	}

	// Should contain at least current dir and home dir.
	foundCurrent := false
	foundHome := false

	for _, path := range paths {
		if strings.Contains(path, "./spin.yaml") || strings.Contains(path, "spin.yaml") {
			foundCurrent = true
		}

		if strings.Contains(path, ".spin") || strings.Contains(path, "HOME") {
			foundHome = true
		}
	}

	if !foundCurrent {
		t.Error("Search paths missing current directory")
	}

	if !foundHome {
		t.Error("Search paths missing home directory")
	}
}
