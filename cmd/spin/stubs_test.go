package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecCommand(t *testing.T) {
	t.Parallel()

	cmd := newExecCmd()
	if !strings.HasPrefix(cmd.Use, "exec") {
		t.Errorf("Exec command Use = %s, should start with 'exec'", cmd.Use)
	}
}

func TestConfigCommand(t *testing.T) {
	t.Parallel()

	cmd := newConfigCmd()
	if cmd.Use != "config" {
		t.Errorf("Config command Use = %s, want 'config'", cmd.Use)
	}

	// Check subcommands.
	subcommands := []string{"show", "validate", "edit", "path"}
	for _, subcmd := range subcommands {
		found := false

		for _, c := range cmd.Commands() {
			if c.Name() == subcmd {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("Config command should have '%s' subcommand", subcmd)
		}
	}
}

func TestMCPCommand(t *testing.T) {
	t.Parallel()

	cmd := newMCPCmd()
	if cmd.Use != "mcp" {
		t.Errorf("MCP command Use = %s, want 'mcp'", cmd.Use)
	}

	// Check subcommands.
	subcommands := []string{"registry", "search", "list"}
	for _, subcmd := range subcommands {
		found := false

		for _, c := range cmd.Commands() {
			if c.Name() == subcmd {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("MCP command should have '%s' subcommand", subcmd)
		}
	}
}

func TestDebugCommand(t *testing.T) {
	t.Parallel()

	cmd := newDebugCmd()
	if cmd.Use != "debug" {
		t.Errorf("Debug command Use = %s, want 'debug'", cmd.Use)
	}

	// Check subcommands.
	subcommands := []string{"sandbox", "landlock"}
	for _, subcmd := range subcommands {
		found := false

		for _, c := range cmd.Commands() {
			if c.Name() == subcmd {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("Debug command should have '%s' subcommand", subcmd)
		}
	}
}

func TestStubCommands_NotImplemented(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		// Only test truly unimplemented commands.
		{"debug landlock", []string{"debug", "landlock", "ls"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rootCmd := newRootCmd()
			rootCmd.SetArgs(tt.args)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)

			err := rootCmd.Execute()
			if err == nil {
				t.Errorf("%s should return 'not implemented' error", tt.name)

				return
			}

			if !strings.Contains(err.Error(), "not implemented") {
				t.Errorf("%s error should mention 'not implemented', got: %v", tt.name, err)
			}
		})
	}
}

func TestAllCommandsAvailable(t *testing.T) {
	t.Parallel()

	rootCmd := newRootCmd()

	expectedCommands := []string{"version", "completion", "exec", "config", "mcp", "debug", "plugin"}
	for _, cmdName := range expectedCommands {
		found := false

		for _, c := range rootCmd.Commands() {
			if c.Name() == cmdName {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("Root command should have '%s' command", cmdName)
		}
	}
}
