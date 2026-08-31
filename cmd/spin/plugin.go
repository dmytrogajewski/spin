package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/plugins"
)

const (
	pluginValidateUse = "validate <dir>"
	reportOK          = "ok"
	errWriteReport    = "write report: %w"
)

// newPluginCmd creates the plugin command with validate.
func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Validate Agent Plugins 1.0 packages",
		Long:  `Validate an Agent Plugins 1.0 directory without starting MCP or merging skills.`,
	}

	cmd.AddCommand(newPluginValidateCmd())

	return cmd
}

func newPluginValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   pluginValidateUse,
		Short: "Validate plugin.json and contained skills",
		Long: `Validate a plugin directory against the Agent Plugins 1.0 closed schema.

Checks required $schema and name, reports unknown top-level fields, contains
plugin-relative paths, and lists immediate skills/ children. Does not start
MCP servers or merge skills into the catalog.

Examples:
  spin plugin validate ./my-plugin`,
		Args: cobra.ExactArgs(1),
		RunE: runPluginValidate,
	}
}

func runPluginValidate(cmd *cobra.Command, args []string) error {
	plugin, err := plugins.Load(args[0])
	if err != nil {
		return fmt.Errorf("validate plugin: %w", err)
	}

	return writeValidateReport(cmd.OutOrStdout(), plugin)
}

func writeValidateReport(out io.Writer, plugin plugins.Plugin) error {
	if _, err := fmt.Fprintf(out, "plugin: %s\n", plugin.Manifest.Name); err != nil {
		return fmt.Errorf(errWriteReport, err)
	}

	if _, err := fmt.Fprintf(out, "schema: %s\n", plugin.Manifest.Schema); err != nil {
		return fmt.Errorf(errWriteReport, err)
	}

	if _, err := fmt.Fprintf(out, "skills: %d\n", len(plugin.Skills)); err != nil {
		return fmt.Errorf(errWriteReport, err)
	}

	for _, skill := range plugin.Skills {
		if _, err := fmt.Fprintf(out, "  - %s\n", skill.Name); err != nil {
			return fmt.Errorf(errWriteReport, err)
		}
	}

	if err := writeValidateWarnings(out, plugin.Warnings); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(out, "%s\n", reportOK); err != nil {
		return fmt.Errorf(errWriteReport, err)
	}

	return nil
}

func writeValidateWarnings(out io.Writer, warnings []string) error {
	if len(warnings) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(out, "warnings:"); err != nil {
		return fmt.Errorf(errWriteReport, err)
	}

	for _, warning := range warnings {
		if _, err := fmt.Fprintf(out, "  - %s\n", warning); err != nil {
			return fmt.Errorf(errWriteReport, err)
		}
	}

	return nil
}
