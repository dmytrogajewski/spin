package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/appinfo"
)

// newVersionCmd creates the version command.
func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the version, build information, and Go version for Spin.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			short, _ := cmd.Flags().GetBool("short")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if short {
				fmt.Fprintln(cmd.OutOrStdout(), appinfo.ShortVersion())

				return nil
			}

			if verbose {
				info := appinfo.GetInfo()
				fmt.Fprintf(cmd.OutOrStdout(), "spin version %s\n", info.Version)
				fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n", info.Commit)
				fmt.Fprintf(cmd.OutOrStdout(), "  built: %s\n", info.BuildDate)
				fmt.Fprintf(cmd.OutOrStdout(), "  go: %s\n", info.GoVersion)

				return nil
			}

			// Default: full version string.
			fmt.Fprintln(cmd.OutOrStdout(), appinfo.String())

			return nil
		},
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show verbose version information")
	cmd.Flags().BoolP("short", "s", false, "Show only the version number")

	return cmd
}
