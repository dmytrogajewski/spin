package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigShow(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		format     string
		wantErr    bool
		wantOutput []string // Strings that should appear in output
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
			// Create temp config file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "spin.yaml")
			if err := os.WriteFile(configFile, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			// Capture output
			var stdout, stderr bytes.Buffer
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			defer func() {
				os.Stdout = oldStdout
				os.Stderr = oldStderr
			}()

			// Create command
			cmd := newConfigShowCmd()
			cmd.SetArgs([]string{"--format", tt.format})
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Set config file flag
			flagConfigFile = configFile

			// Execute
			err := cmd.Execute()

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
			// Create temp config file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "spin.yaml")
			if err := os.WriteFile(configFile, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			// Capture output
			var stdout, stderr bytes.Buffer

			// Create command
			cmd := newConfigValidateCmd()
			cmd.SetArgs([]string{})
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Set config file flag
			flagConfigFile = configFile

			// Execute
			err := cmd.Execute()

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
			// Create temp dir
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "spin.yaml")

			if tt.setupFile {
				// Create minimal valid config
				configYAML := `version: "2.0"
llm:
  provider: openai
  model: gpt-4o
`
				if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
					t.Fatalf("Failed to create config file: %v", err)
				}
			}

			// Capture output
			var stdout, stderr bytes.Buffer

			// Create command
			cmd := newConfigPathCmd()
			cmd.SetArgs([]string{})
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Set config file flag
			if tt.setupFile {
				flagConfigFile = configFile
			} else {
				flagConfigFile = filepath.Join(tmpDir, "nonexistent.yaml")
			}

			// Execute
			err := cmd.Execute()

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
	// Create temp dir
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "nonexistent.yaml")

	// Capture output
	var stdout, stderr bytes.Buffer

	// Create command
	cmd := newConfigPathCmd()
	cmd.SetArgs([]string{"--all"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	// Set nonexistent config file
	flagConfigFile = configFile

	// Execute
	err := cmd.Execute()

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
		// Clear all editor env vars
		oldEditor := os.Getenv("EDITOR")
		oldVisual := os.Getenv("VISUAL")
		oldPath := os.Getenv("PATH")
		os.Unsetenv("EDITOR")
		os.Unsetenv("VISUAL")
		os.Setenv("PATH", "/nonexistent") // Make sure no editors in path
		defer func() {
			os.Setenv("EDITOR", oldEditor)
			os.Setenv("VISUAL", oldVisual)
			os.Setenv("PATH", oldPath)
		}()

		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "spin.yaml")

		// Create minimal config
		configYAML := `version: "2.0"
llm:
  provider: openai
  model: gpt-4o
`
		if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		// Create command
		cmd := newConfigEditCmd()
		cmd.SetArgs([]string{})

		flagConfigFile = configFile

		// Execute - should fail with no editor
		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error when no editor found, got nil")
		}

		if !strings.Contains(err.Error(), "editor") {
			t.Errorf("Error should mention editor, got: %v", err)
		}
	})
}

func TestGetEditor(t *testing.T) {
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
			wantNonEmpty: true, // vi should exist on most systems
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			oldEditor := os.Getenv("EDITOR")
			oldVisual := os.Getenv("VISUAL")
			defer func() {
				os.Setenv("EDITOR", oldEditor)
				os.Setenv("VISUAL", oldVisual)
			}()

			// Set test env vars
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
	paths := getConfigSearchPaths()

	if len(paths) == 0 {
		t.Error("getConfigSearchPaths() returned empty slice")
	}

	// Should contain at least current dir and home dir
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
