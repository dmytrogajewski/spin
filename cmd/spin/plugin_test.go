package main

// Journey: specs/journeys/JOURNEY-004-parse-agent-plugins.md.

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

const (
	pluginTestdata      = "../../internal/plugins/testdata"
	pluginValidFixture  = "valid-plugin"
	pluginUnknownFix    = "unknown-field"
	pluginNestedFix     = "nested-skill"
	wantValidateOK      = "ok"
	wantPluginNameValid = "valid-plugin"
	wantSkillSummarize  = "summarize"
)

func TestNewPluginCmd(t *testing.T) {
	t.Parallel()

	cmd := newPluginCmd()
	if cmd.Use != "plugin" {
		t.Errorf("Use = %q, want plugin", cmd.Use)
	}

	if len(cmd.Commands()) != 1 {
		t.Errorf("expected 1 subcommand, got %d", len(cmd.Commands()))
	}

	if cmd.Commands()[0].Name() != "validate" {
		t.Errorf("subcommand = %q, want validate", cmd.Commands()[0].Name())
	}
}

func TestPluginValidate_Valid(t *testing.T) {
	t.Parallel()

	out, err := runPluginValidateCmd(filepath.Join(pluginTestdata, pluginValidFixture))
	if err != nil {
		t.Fatalf("validate valid plugin: %v", err)
	}

	if !strings.Contains(out, "plugin: "+wantPluginNameValid) {
		t.Errorf("output missing plugin name: %s", out)
	}

	if !strings.Contains(out, wantSkillSummarize) {
		t.Errorf("output missing skill: %s", out)
	}

	if !strings.Contains(out, wantValidateOK) {
		t.Errorf("output missing ok: %s", out)
	}
}

func TestPluginValidate_UnknownFieldWarning(t *testing.T) {
	t.Parallel()

	out, err := runPluginValidateCmd(filepath.Join(pluginTestdata, pluginUnknownFix))
	if err != nil {
		t.Fatalf("validate unknown-field plugin: %v", err)
	}

	if !strings.Contains(out, "unknown field") {
		t.Errorf("expected unknown-field warning, got: %s", out)
	}

	if !strings.Contains(out, wantValidateOK) {
		t.Errorf("unknown field should still be ok, got: %s", out)
	}
}

func TestPluginValidate_NestedSkillIgnored(t *testing.T) {
	t.Parallel()

	out, err := runPluginValidateCmd(filepath.Join(pluginTestdata, pluginNestedFix))
	if err != nil {
		t.Fatalf("validate nested-skill plugin: %v", err)
	}

	if !strings.Contains(out, "deploy") {
		t.Errorf("expected deploy skill, got: %s", out)
	}

	if strings.Contains(out, "  - nested") {
		t.Errorf("nested skill must be ignored, got: %s", out)
	}
}

func TestPluginValidate_MissingManifest(t *testing.T) {
	t.Parallel()

	_, err := runPluginValidateCmd(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing plugin.json")
	}

	if !strings.Contains(err.Error(), "plugin.json not found") {
		t.Errorf("error = %v, want missing plugin.json", err)
	}
}

func TestRootHasPluginCommand(t *testing.T) {
	t.Parallel()

	rootCmd := newRootCmd()

	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "plugin" {
			found = true

			break
		}
	}

	if !found {
		t.Fatal("root command should have plugin")
	}
}

func runPluginValidateCmd(dir string) (string, error) {
	cmd := newPluginValidateCmd()
	cmd.SetArgs([]string{dir})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()

	return out.String(), err
}
