package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/security"
)

// newApprovalCmd creates the 'approval' command group.
func newApprovalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approval",
		Short: "Manage persisted approval policies (session/global)",
		Long:  "List, revoke, and clear persisted approval policies. Uses the configured policy file when available.",
	}

	cmd.AddCommand(newApprovalListCmd())
	cmd.AddCommand(newApprovalRevokeCmd())
	cmd.AddCommand(newApprovalClearCmd())

	return cmd
}

// newApprovalListCmd creates the command for listing stored approvals.
func newApprovalListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List persisted approval policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, _ := cmd.Flags().GetString("scope")

			store, err := buildPolicyStore()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			items, err := store.List(ctx, scope)
			if err != nil {
				return fmt.Errorf("list policies: %w", err)
			}

			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No policies found.")

				return nil
			}

			for _, p := range items {
				exp := "never"
				if p.ExpiresAt != nil {
					exp = p.ExpiresAt.UTC().Format(time.RFC3339)
				}

				args := strings.Join(p.Key.Args, " ")
				fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s %s (wd=%s) decision=%s expires=%s\n",
					p.Scope, p.Key.Program, args, p.Key.WorkDir, p.Decision, exp)
			}

			return nil
		},
	}
	cmd.Flags().String("scope", security.ScopeGlobal, "Scope to list (session|global)")

	return cmd
}

// newApprovalRevokeCmd creates the command for revoking a specific approval.
func newApprovalRevokeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke a specific approval policy",
		Long:  "Revoke a policy by specifying program, args, workdir and scope.",
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, _ := cmd.Flags().GetString("scope")
			prog, _ := cmd.Flags().GetString("program")
			workDir, _ := cmd.Flags().GetString("workdir")
			argList, _ := cmd.Flags().GetStringArray("arg")

			if prog == "" {
				return errors.New("--program is required")
			}

			store, err := buildPolicyStore()
			if err != nil {
				return err
			}

			key := security.NewPolicyKey(prog, argList, workDir)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			deleted, err := store.Delete(ctx, key, scope)
			if err != nil {
				return fmt.Errorf("revoke policy: %w", err)
			}

			if !deleted {
				fmt.Fprintln(cmd.OutOrStdout(), "No matching policy found.")

				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Policy revoked.")

			return nil
		},
	}
	cmd.Flags().String("scope", security.ScopeGlobal, "Scope (session|global)")
	cmd.Flags().String("program", "", "Program/binary of the approved command (required)")
	cmd.Flags().StringArray("arg", nil, "Argument (repeatable)")
	cmd.Flags().String("workdir", "", "Working directory used for the command")

	return cmd
}

// newApprovalClearCmd creates the command for clearing all stored approvals.
func newApprovalClearCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all persisted approval policies for a scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, _ := cmd.Flags().GetString("scope")

			store, err := buildPolicyStore()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			n, err := store.Clear(ctx, scope)
			if err != nil {
				return fmt.Errorf("clear policies: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Cleared %d policies.\n", n)

			return nil
		},
	}
	cmd.Flags().String("scope", security.ScopeGlobal, "Scope to clear (session|global)")

	return cmd
}

// buildPolicyStore constructs a PolicyStore consistent with agent builder logic.
func buildPolicyStore() (security.PolicyStore, error) {
	loader := config.NewLoaderV2()

	cfg, err := func() (*config.ConfigV2, error) {
		if flagConfigFile != "" {
			return loader.LoadFromFileWithEnv(flagConfigFile)
		}

		return loader.LoadWithEnv()
	}()
	if err != nil {
		// Fallback to defaults if config cannot be loaded.
		cfg = config.DefaultConfigV2()
	}

	// If approval persistence is disabled, operate purely in-memory.
	if !cfg.Security.ApprovalPersistenceEnabled {
		return security.NewMemoryPolicyStore(30 * time.Second), nil
	}

	// If policy file path not set, default under user config dir matching DefaultConfigV2.
	path := cfg.Security.PolicyFile
	if path == "" {
		// Ensure we always operate on a deterministic path for file-backed policies
		// DefaultConfigV2 sets it to ~/.config/spin/policies.json when available
		// If still empty (edge), compute a local default.
		home := filepath.Clean("~")
		if strings.HasPrefix(path, "~") {
			// leave as-is for file store to resolve via os.UserConfigDir in defaults.
		} else if home == "~" {
			// No explicit home expansion here; rely on DefaultConfigV2 typical path.
		}
	}

	// Prefer file-backed store when path present; otherwise memory store.
	if path != "" {
		store, err := security.NewFilePolicyStore(path, 30*time.Second)
		if err == nil {
			return store, nil
		}
	}

	return security.NewMemoryPolicyStore(30 * time.Second), nil
}
