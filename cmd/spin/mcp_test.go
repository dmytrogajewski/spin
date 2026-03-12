package main

import (
	"strings"
	"testing"
)

func TestNewMCPCmd(t *testing.T) {
	t.Parallel()

	cmd := newMCPCmd()
	if cmd == nil {
		t.Fatal("newMCPCmd() returned nil")
	}

	if cmd.Use != "mcp" {
		t.Errorf("newMCPCmd().Use = %v, want %v", cmd.Use, "mcp")
	}

	if cmd.Short != "Manage MCP (Model Context Protocol) registries and tools" {
		t.Errorf("newMCPCmd().Short = %v, want %v", cmd.Short, "Manage MCP (Model Context Protocol) registries and tools")
	}
}

func TestNewMCPRegistryListCmd(t *testing.T) {
	t.Parallel()

	cmd := newMCPRegistryListCmd()
	if cmd == nil {
		t.Fatal("newMCPRegistryListCmd() returned nil")
	}

	if cmd.Use != "list" {
		t.Errorf("Use = %v, want %v", cmd.Use, "list")
	}

	if cmd.Short != "List all configured registries" {
		t.Errorf("Short = %v, want %v", cmd.Short, "List all configured registries")
	}
}

func TestNewMCPRegistryGetCmd(t *testing.T) {
	t.Parallel()

	cmd := newMCPRegistryGetCmd()
	if cmd == nil {
		t.Fatal("newMCPRegistryGetCmd() returned nil")
	}

	if cmd.Use != "get <name>" {
		t.Errorf("Use = %v, want %v", cmd.Use, "get <name>")
	}

	if cmd.Short != "Show details of a registry" {
		t.Errorf("Short = %v, want %v", cmd.Short, "Show details of a registry")
	}
}

func TestNewMCPRegistryRemoveCmd(t *testing.T) {
	t.Parallel()

	cmd := newMCPRegistryRemoveCmd()
	if cmd == nil {
		t.Fatal("newMCPRegistryRemoveCmd() returned nil")
	}

	if cmd.Use != "remove <name>" {
		t.Errorf("Use = %v, want %v", cmd.Use, "remove <name>")
	}

	if cmd.Short != "Remove a registry" {
		t.Errorf("Short = %v, want %v", cmd.Short, "Remove a registry")
	}
}

func TestMCPCmdSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newMCPCmd()

	expectedSubcommands := []string{
		"registry",
		"search",
		"list",
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
	t.Parallel()

	cmd := newMCPCmd()

	if cmd.Long == "" {
		t.Errorf("MCP command Long description is empty")
	}

	if cmd.Short == "" {
		t.Errorf("MCP command Short description is empty")
	}
}

func TestMCPCmdExamples(t *testing.T) {
	t.Parallel()

	cmd := newMCPCmd()

	helpText := cmd.Long
	expectedExamples := []string{
		"spin mcp registry local add filesystem",
		"spin mcp registry list",
		"spin mcp search github",
		"spin mcp list",
	}

	for _, example := range expectedExamples {
		if !strings.Contains(helpText, example) {
			t.Errorf("Help text missing example: %s", example)
		}
	}
}

func TestMCPRegistryListCmdFlags(t *testing.T) {
	t.Parallel()

	cmd := newMCPRegistryListCmd()

	flag := cmd.Flags().Lookup("format")
	if flag == nil {
		t.Errorf("Flag 'format' not found")
	}
}

func TestMCPRegistryListCmdDefaultValues(t *testing.T) {
	t.Parallel()

	cmd := newMCPRegistryListCmd()

	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil || formatFlag.DefValue != "table" {
		t.Errorf("format flag default = %v, want %v", formatFlag.DefValue, "table")
	}
}

func TestMCPRegistryGetCmdFlags(t *testing.T) {
	t.Parallel()

	cmd := newMCPRegistryGetCmd()

	flag := cmd.Flags().Lookup("format")
	if flag == nil {
		t.Errorf("Flag 'format' not found")
	}
}

func TestMCPRegistryGetCmdDefaultValues(t *testing.T) {
	t.Parallel()

	cmd := newMCPRegistryGetCmd()

	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil || formatFlag.DefValue != "text" {
		t.Errorf("format flag default = %v, want %v", formatFlag.DefValue, "text")
	}
}

func TestMCPRegistrySubcommands(t *testing.T) {
	t.Parallel()

	cmd := newMCPRegistryCmd()

	expectedSubcommands := []string{
		"local",
		"remote",
		"smithery",
		"list",
		"get",
		"remove",
	}

	subcommands := cmd.Commands()
	if len(subcommands) != len(expectedSubcommands) {
		t.Errorf("registry command has %d subcommands, want %d", len(subcommands), len(expectedSubcommands))
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
			t.Errorf("registry subcommand %s not found", expected)
		}
	}
}
