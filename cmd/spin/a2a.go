package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/agent/child"
	"github.com/dmytrogajewski/spin/internal/agent/subagent"
)

const (
	flagA2ASpec   = "spec"
	flagA2AStdio  = "stdio"
	flagA2AListen = "listen"
)

// newA2ACmd creates the local A2A child server command.
func newA2ACmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "a2a",
		Short: "Serve a local A2A child over stdio or a Unix socket",
		Long: `Start spin as a local A2A child process.

The first framed stdout (or socket) message is the Agent Card built from a
builtin subagent Spec. Further lines are NDJSON-RPC (message/send, tasks/*).
Child logs go to stderr, never the RPC stream.

Examples:
  spin a2a --spec explorer --stdio
  spin a2a --spec explorer --listen unix:///tmp/spin-a2a.sock`,
		RunE: runA2A,
	}

	cmd.Flags().String(flagA2ASpec, "", "builtin spec (explorer, planner, reviewer, ask_user)")
	cmd.Flags().Bool(flagA2AStdio, true, "serve NDJSON-RPC on stdin/stdout (default)")
	cmd.Flags().String(flagA2AListen, "", "alternate binding unix://PATH")
	_ = cmd.MarkFlagRequired(flagA2ASpec)

	return cmd
}

func runA2A(cmd *cobra.Command, _ []string) error {
	cleanupPid := installChildPid()
	defer cleanupPid()

	specName, _ := cmd.Flags().GetString(flagA2ASpec)
	listen, _ := cmd.Flags().GetString(flagA2AListen)

	spec, err := subagent.Lookup(specName)
	if err != nil {
		return fmt.Errorf("a2a: %w", err)
	}

	server, err := child.NewServer(spec)
	if err != nil {
		return fmt.Errorf("a2a server: %w", err)
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if listen != "" {
		if serveErr := server.ListenAndServe(ctx, listen); serveErr != nil {
			return fmt.Errorf("a2a listen: %w", serveErr)
		}

		return nil
	}

	if serveErr := server.Serve(ctx, cmd.InOrStdin(), cmd.OutOrStdout()); serveErr != nil {
		return fmt.Errorf("a2a serve: %w", serveErr)
	}

	return nil
}

func installChildPid() func() {
	dir := child.RuntimeDir()
	pid := os.Getpid()
	_ = child.WritePidFile(dir, pid)

	return func() {
		_ = os.Remove(child.PidPath(dir, pid))
	}
}
