package main

import (
	"testing"
)

func TestNewMCPCmd(t *testing.T) {
	cmd := newMCPCmd()
	if cmd == nil {
		t.Errorf("newMCPCmd() returned nil")
	}

	if cmd.Use != "mcp" {
		t.Errorf("newMCPCmd().Use = %v, want %v", cmd.Use, "mcp")
	}

	if cmd.Short != "Manage MCP (Model Context Protocol) servers" {
		t.Errorf("newMCPCmd().Short = %v, want %v", cmd.Short, "Manage MCP (Model Context Protocol) servers")
	}
}

func TestNewMCPAddCmd(t *testing.T) {
	cmd := newMCPAddCmd()
	if cmd == nil {
		t.Errorf("newMCPAddCmd() returned nil")
	}

	if cmd.Use != "add <name> <command> [args...]" {
		t.Errorf("newMCPAddCmd().Use = %v, want %v", cmd.Use, "add <name> <command> [args...]")
	}

	if cmd.Short != "Add a new MCP server" {
		t.Errorf("newMCPAddCmd().Short = %v, want %v", cmd.Short, "Add a new MCP server")
	}
}

func TestNewMCPListCmd(t *testing.T) {
	cmd := newMCPListCmd()
	if cmd == nil {
		t.Errorf("newMCPListCmd() returned nil")
	}

	if cmd.Use != "list" {
		t.Errorf("newMCPListCmd().Use = %v, want %v", cmd.Use, "list")
	}

	if cmd.Short != "List all configured MCP servers" {
		t.Errorf("newMCPListCmd().Short = %v, want %v", cmd.Short, "List all configured MCP servers")
	}
}

func TestNewMCPGetCmd(t *testing.T) {
	cmd := newMCPGetCmd()
	if cmd == nil {
		t.Errorf("newMCPGetCmd() returned nil")
	}

	if cmd.Use != "get <name>" {
		t.Errorf("newMCPGetCmd().Use = %v, want %v", cmd.Use, "get <name>")
	}

	if cmd.Short != "Show details of an MCP server" {
		t.Errorf("newMCPGetCmd().Short = %v, want %v", cmd.Short, "Show details of an MCP server")
	}
}

func TestNewMCPRemoveCmd(t *testing.T) {
	cmd := newMCPRemoveCmd()
	if cmd == nil {
		t.Errorf("newMCPRemoveCmd() returned nil")
	}

	if cmd.Use != "remove <name>" {
		t.Errorf("newMCPRemoveCmd().Use = %v, want %v", cmd.Use, "remove <name>")
	}

	if cmd.Short != "Remove an MCP server" {
		t.Errorf("newMCPRemoveCmd().Short = %v, want %v", cmd.Short, "Remove an MCP server")
	}
}

func TestMCPCmdSubcommands(t *testing.T) {
	cmd := newMCPCmd()

	// Test that all expected subcommands exist
	expectedSubcommands := []string{
		"add",
		"list",
		"get",
		"remove",
	}

	subcommands := cmd.Commands()
	if len(subcommands) != len(expectedSubcommands) {
		t.Errorf("MCP command has %d subcommands, want %d", len(subcommands), len(expectedSubcommands))
	}

	for _, expected := range expectedSubcommands {
		found := false
		for _, subcmd := range subcommands {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("MCP subcommand %s not found", expected)
		}
	}
}

func TestMCPCmdHelp(t *testing.T) {
	cmd := newMCPCmd()

	// Test that help text is properly set
	if cmd.Long == "" {
		t.Errorf("MCP command Long description is empty")
	}

	if cmd.Short == "" {
		t.Errorf("MCP command Short description is empty")
	}
}

func TestMCPCmdExamples(t *testing.T) {
	cmd := newMCPCmd()

	// Test that examples are included in help text
	helpText := cmd.Long
	expectedExamples := []string{
		"spin mcp add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace",
		"spin mcp list",
		"spin mcp get filesystem",
		"spin mcp remove filesystem",
	}

	for _, example := range expectedExamples {
		if !contains(helpText, example) {
			t.Errorf("Help text missing example: %s", example)
		}
	}
}

func TestMCPAddCmdFlags(t *testing.T) {
	cmd := newMCPAddCmd()

	// MCP add command currently has no flags
	// Flags can be added in the future as needed
	if cmd.Flags() == nil {
		t.Errorf("newMCPAddCmd().Flags() returned nil")
	}
}

func TestMCPAddCmdDefaultValues(t *testing.T) {
	cmd := newMCPAddCmd()

	// MCP add command currently has no flags with default values
	// This test is a placeholder for future flag additions
	if cmd == nil {
		t.Errorf("newMCPAddCmd() returned nil")
	}
}

func TestMCPListCmdFlags(t *testing.T) {
	cmd := newMCPListCmd()

	// Test that expected flags exist
	expectedFlags := []string{
		"format",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Flag %s not found", flagName)
		}
	}
}

func TestMCPListCmdDefaultValues(t *testing.T) {
	cmd := newMCPListCmd()

	// Test default values
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil || formatFlag.DefValue != "table" {
		t.Errorf("format flag default = %v, want %v", formatFlag.DefValue, "table")
	}
}

func TestMCPGetCmdFlags(t *testing.T) {
	cmd := newMCPGetCmd()

	// Test that expected flags exist
	expectedFlags := []string{
		"format",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Flag %s not found", flagName)
		}
	}
}

func TestMCPGetCmdDefaultValues(t *testing.T) {
	cmd := newMCPGetCmd()

	// Test default values
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil || formatFlag.DefValue != "text" {
		t.Errorf("format flag default = %v, want %v", formatFlag.DefValue, "text")
	}
}

// Helper function to check if a string contains a substring

