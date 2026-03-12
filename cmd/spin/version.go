package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/version"
)

var (
	versionVerbose bool
	versionShort   bool
)

// newVersionCmd creates the version command.
func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the version, build information, and Go version for Spin.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if versionShort {
				fmt.Fprintln(cmd.OutOrStdout(), version.ShortVersion())

				return nil
			}

			if versionVerbose {
				info := version.GetVersionInfo()
				fmt.Fprintf(cmd.OutOrStdout(), "spin version %s\n", info.Version)
				fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n", info.Commit)
				fmt.Fprintf(cmd.OutOrStdout(), "  built: %s\n", info.BuildDate)
				fmt.Fprintf(cmd.OutOrStdout(), "  go: %s\n", info.GoVersion)

				return nil
			}

			// Default: full version string.
			fmt.Fprintln(cmd.OutOrStdout(), version.String())

			return nil
		},
	}

	cmd.Flags().BoolVarP(&versionVerbose, "verbose", "v", false, "Show verbose version information")
	cmd.Flags().BoolVarP(&versionShort, "short", "s", false, "Show only the version number")

	return cmd
}
