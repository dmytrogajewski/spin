package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/dmytrogajewski/spin/internal/auth"
	termx "github.com/dmytrogajewski/spin/internal/ui/term"
)

// ErrAPIKeyCannotBeEmpty is a sentinel error.
var ErrAPIKeyCannotBeEmpty = errors.New("api key cannot be empty")

// newAuthCmd creates the auth management command.
func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication credentials",
		Long: `Manage authentication credentials for LLM providers.

Credentials are stored securely using platform-specific keystores:
  • Linux: Secret Service (libsecret)
  • macOS: Keychain
  • Windows: Credential Manager

Examples:
  # Store a credential
  spin auth login openai

  # List all stored credentials
  spin auth list

  # Delete a credential
  spin auth logout openai`,
	}

	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthListCmd())

	return cmd
}

// newAuthLoginCmd creates the command for logging in to an LLM provider.
func newAuthLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login <provider>",
		Short: "Store authentication credential for a provider",
		Long: `Store authentication credential for an LLM provider.

The credential is stored securely in your system's keystore.
Supported providers: openai, anthropic, openai-compatible

Examples:
  # Store OpenAI API key
  spin auth login openai

  # Store Anthropic API key
  spin auth login anthropic

  # Store with explicit key (non-interactive)
  spin auth login openai --key sk-...`,
		Args: cobra.ExactArgs(1),
		RunE: runAuthLogin,
	}
	cmd.Flags().String("key", "", "API key (if not provided, will prompt securely)")

	return cmd
}

// newAuthLogoutCmd creates the command for logging out from an LLM provider.
func newAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout <provider>",
		Short: "Remove authentication credential for a provider",
		Long: `Remove authentication credential for an LLM provider.

This deletes the credential from your system's keystore.

Examples:
  # Remove OpenAI credential
  spin auth logout openai

  # Remove Anthropic credential
  spin auth logout anthropic`,
		Args: cobra.ExactArgs(1),
		RunE: runAuthLogout,
	}

	return cmd
}

// newAuthListCmd creates the command for listing authenticated providers.
func newAuthListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all stored credentials",
		Long: `List all providers with stored authentication credentials.

Credential values are not displayed for safety.

Note: On Linux, the Secret Service API does not support listing credentials.
Use specific provider names with 'spin auth login' instead.

Examples:
  # List all stored credentials (macOS, Windows)
  spin auth list`,
		RunE: runAuthList,
	}

	return cmd
}

// runAuthLogin implements the 'auth login' command.
func runAuthLogin(cmd *cobra.Command, args []string) error {
	provider := args[0]
	keyFlag, _ := cmd.Flags().GetString("key")

	// Get API key.
	var apiKey string
	if keyFlag != "" {
		apiKey = keyFlag
	} else {
		// Prompt for key securely.
		var err error

		apiKey, err = promptForAPIKey(cmd, provider)
		if err != nil {
			return err
		}
	}

	if apiKey == "" {
		return ErrAPIKeyCannotBeEmpty
	}

	// Create auth manager.
	authMgr := createAuthManager()

	// Store credential.
	cred := auth.Credential{
		Type:  auth.CredentialTypeAPIKey,
		Value: apiKey,
	}

	ctx := context.Background()

	err := authMgr.SetCredential(ctx, provider, cred)
	if err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Credential stored for %s\n", provider)

	return nil
}

// runAuthLogout implements the 'auth logout' command.
func runAuthLogout(cmd *cobra.Command, args []string) error {
	provider := args[0]

	// Create auth manager.
	authMgr := createAuthManager()

	// Delete credential.
	ctx := context.Background()

	err := authMgr.DeleteCredential(ctx, provider)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Credential removed for %s\n", provider)

	return nil
}

// runAuthList implements the 'auth list' command.
func runAuthList(cmd *cobra.Command, _ []string) error {
	// Create auth manager.
	authMgr := createAuthManager()

	// List providers.
	ctx := context.Background()

	providers, err := authMgr.ListProviders(ctx)
	if err != nil {
		// Check if this is the "list not supported" error on Linux.
		if strings.Contains(err.Error(), "list not supported") {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Listing credentials is not supported on Linux Secret Service.\n\n")
			fmt.Fprintf(cmd.ErrOrStderr(), "You can still use credentials by provider name:\n")
			fmt.Fprintf(cmd.ErrOrStderr(), "  • spin auth login <provider>  - Store credential\n")
			fmt.Fprintf(cmd.ErrOrStderr(), "  • spin auth logout <provider> - Remove credential\n\n")
			fmt.Fprintf(cmd.ErrOrStderr(), "Supported providers: openai, anthropic, openai-compatible\n")

			return nil
		}

		return fmt.Errorf("failed to list credentials: %w", err)
	}

	if len(providers) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No stored credentials\n")
		fmt.Fprintf(cmd.OutOrStdout(), "\nTo store a credential, run:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  spin auth login <provider>\n")

		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Stored credentials:\n")

	for _, provider := range providers {
		fmt.Fprintf(cmd.OutOrStdout(), "  • %s\n", provider)
	}

	return nil
}

// promptForAPIKey prompts the user to enter an API key securely.
func promptForAPIKey(cmd *cobra.Command, provider string) (string, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "Enter API key for %s: ", provider)

	// Check if stdin is a terminal.
	fd := termx.SafeFd(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		// Read password without echo.
		keyBytes, err := term.ReadPassword(fd)

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		if err != nil {
			return "", fmt.Errorf("failed to read API key: %w", err)
		}

		return string(keyBytes), nil
	}

	// Fallback: read from non-terminal stdin.
	reader := bufio.NewReader(os.Stdin)

	key, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read API key: %w", err)
	}

	return strings.TrimSpace(key), nil
}

// init registers syscall for term package on Unix systems.
func init() {
	// Ensure term package can access syscall.Stdin.
	_ = syscall.Stdin
}
