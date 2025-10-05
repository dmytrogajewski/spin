package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecCommand(t *testing.T) {
	cmd := newExecCmd()
	if !strings.HasPrefix(cmd.Use, "exec") {
		t.Errorf("Exec command Use = %s, should start with 'exec'", cmd.Use)
	}
}

func TestConfigCommand(t *testing.T) {
	cmd := newConfigCmd()
	if cmd.Use != "config" {
		t.Errorf("Config command Use = %s, want 'config'", cmd.Use)
	}

	// Check subcommands
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
	cmd := newMCPCmd()
	if cmd.Use != "mcp" {
		t.Errorf("MCP command Use = %s, want 'mcp'", cmd.Use)
	}

	// Check subcommands
	subcommands := []string{"add", "list", "get", "remove"}
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
	cmd := newDebugCmd()
	if cmd.Use != "debug" {
		t.Errorf("Debug command Use = %s, want 'debug'", cmd.Use)
	}

	// Check subcommands
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
	tests := []struct {
		name string
		args []string
	}{
		// Note: "exec" removed - now implemented in Phase 2.1
		{"config show", []string{"config", "show"}},
		{"config validate", []string{"config", "validate"}},
		{"config edit", []string{"config", "edit"}},
		{"config path", []string{"config", "path"}},
		{"mcp add", []string{"mcp", "add", "test", "cmd"}},
		{"mcp list", []string{"mcp", "list"}},
		{"mcp get", []string{"mcp", "get", "test"}},
		{"mcp remove", []string{"mcp", "remove", "test"}},
		{"debug sandbox", []string{"debug", "sandbox", "ls"}},
		{"debug landlock", []string{"debug", "landlock", "ls"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := newRootCmd()
			rootCmd.SetArgs(tt.args)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)

			err := rootCmd.Execute()
			if err == nil {
				t.Errorf("%s should return 'not implemented' error", tt.name)
			}

			if !strings.Contains(err.Error(), "not yet implemented") {
				t.Errorf("%s error should mention 'not yet implemented', got: %v", tt.name, err)
			}
		})
	}
}

func TestAllCommandsAvailable(t *testing.T) {
	rootCmd := newRootCmd()

	expectedCommands := []string{"version", "completion", "exec", "config", "mcp", "debug"}
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
