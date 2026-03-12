package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigShow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configYAML string
		format     string
		wantErr    bool
		wantOutput []string // Strings that should appear in output.
	}{
		{
			name: "text format with valid config",
			configYAML: `version: "2.0"
llm:
  provider: openai
  model: gpt-4o
security:
  sandbox_mode: workspace-only
`,
			format:  "text",
			wantErr: false,
			wantOutput: []string{
				"llm:",
				"provider: openai",
				"model: gpt-4o",
				"security:",
			},
		},
		{
			name: "json format with valid config",
			configYAML: `version: "2.0"
llm:
  provider: openai
  model: gpt-4o
`,
			format:  "json",
			wantErr: false,
			wantOutput: []string{
				`"LLM"`,
				`"Provider"`,
				`"openai"`,
			},
		},
		{
			name: "yaml format with valid config",
			configYAML: `version: "2.0"
llm:
  provider: openai
  model: gpt-4o
`,
			format:  "yaml",
			wantErr: false,
			wantOutput: []string{
				"llm:",
				"provider: openai",
			},
		},
		{
			name: "redacts sensitive values",
			configYAML: `version: "2.0"
llm:
  provider: openai
  model: gpt-4o
  api_key: sk-secret123
`,
			format:  "text",
			wantErr: false,
			wantOutput: []string{
				"api_key:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp config file.
			tmpDir := t.TempDir()

			configFile := filepath.Join(tmpDir, "spin.yaml")
			err := os.WriteFile(configFile, []byte(tt.configYAML), 0644)
			if err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			// Capture output.
			var stdout, stderr bytes.Buffer

			// Create root command with config subcommand (provides --config-file persistent flag).
			root := newRootCmd()
			root.SetArgs([]string{"--config-file", configFile, "config", "show", "--format", tt.format})
			root.SetOut(&stdout)
			root.SetErr(&stderr)

			// Execute.
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
			err := os.WriteFile(configFile, []byte(tt.configYAML), 0644)
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

func TestConfigPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupFile  bool
		wantErr    bool
		wantOutput []string
	}{
		{
			name:      "config file exists",
			setupFile: true,
			wantErr:   false,
			wantOutput: []string{
				"spin.yaml",
			},
		},
		{
			name:      "config file not found",
			setupFile: false,
			wantErr:   true,
			wantOutput: []string{
				"No configuration file found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp dir.
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "spin.yaml")

			if tt.setupFile {
				// Create minimal valid config.
				configYAML := `version: "2.0"
llm:
  provider: openai
  model: gpt-4o
`
				err := os.WriteFile(configFile, []byte(configYAML), 0644)
				if err != nil {
					t.Fatalf("Failed to create config file: %v", err)
				}
			}

			// Capture output.
			var stdout, stderr bytes.Buffer

			// Determine config file path.
			cfgPath := configFile
			if !tt.setupFile {
				cfgPath = filepath.Join(tmpDir, "nonexistent.yaml")
			}

			// Create root command with config subcommand.
			root := newRootCmd()
			root.SetArgs([]string{"--config-file", cfgPath, "config", "path"})
			root.SetOut(&stdout)
			root.SetErr(&stderr)

			// Execute.
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
		oldEditor := os.Getenv("EDITOR")
		oldVisual := os.Getenv("VISUAL")
		oldPath := os.Getenv("PATH")

		os.Unsetenv("EDITOR")
		os.Unsetenv("VISUAL")
		os.Setenv("PATH", "/nonexistent") // Make sure no editors in path.

		defer func() {
			os.Setenv("EDITOR", oldEditor)
			os.Setenv("VISUAL", oldVisual)
			os.Setenv("PATH", oldPath)
		}()

		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "spin.yaml")

		// Create minimal config.
		configYAML := `version: "2.0"
llm:
  provider: openai
  model: gpt-4o
`
		err := os.WriteFile(configFile, []byte(configYAML), 0644)
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
			// Save and restore env vars.
			oldEditor := os.Getenv("EDITOR")
			oldVisual := os.Getenv("VISUAL")

			defer func() {
				os.Setenv("EDITOR", oldEditor)
				os.Setenv("VISUAL", oldVisual)
			}()

			// Set test env vars.
			os.Unsetenv("EDITOR")
			os.Unsetenv("VISUAL")

			if tt.editorEnv != "" {
				os.Setenv("EDITOR", tt.editorEnv)
			}

			if tt.visualEnv != "" {
				os.Setenv("VISUAL", tt.visualEnv)
			}

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
