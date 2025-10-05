package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmytrogajewski/spin/internal/appserver"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/ollama"
	"github.com/dmytrogajewski/spin/internal/llm/openai"
	"github.com/dmytrogajewski/spin/internal/version"
	"github.com/spf13/cobra"
)

// newServeCmd creates the serve command for JSON-RPC app server mode.
func newServeCmd() *cobra.Command {
	var (
		workDir      string
		providerType string
		baseURL      string
		model        string
		apiKey       string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start JSON-RPC app server",
		Long: `Start Spin as a JSON-RPC server for IDE integration.

The server communicates via stdin/stdout using JSON-RPC protocol.
This mode is designed for editors and IDEs that want to integrate
Spin as a language server or coding assistant.

Examples:
  spin serve
  spin serve --provider openai --model gpt-4
  spin serve --workspace /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(workDir, providerType, baseURL, model, apiKey)
		},
	}

	// Server-specific flags
	cmd.Flags().StringVar(&workDir, "workspace", ".", "Workspace directory path")
	cmd.Flags().StringVar(&providerType, "provider", "ollama", "LLM provider type (ollama, openai)")
	cmd.Flags().StringVar(&baseURL, "base-url", "http://localhost:11434", "Provider base URL")
	cmd.Flags().StringVar(&model, "model", "codellama:13b", "Model name")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key (for cloud providers)")

	return cmd
}

// runServer starts the JSON-RPC server.
func runServer(workDir, providerType, baseURL, model, apiKey string) error {
	// Create LLM provider
	var provider llm.Provider
	var err error

	switch providerType {
	case "ollama":
		provider, err = ollama.NewProvider(ollama.Config{
			BaseURL: baseURL,
			Model:   model,
		})

	case "openai":
		provider, err = openai.NewProvider(openai.Config{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   model,
		})

	default:
		return fmt.Errorf("unknown provider type: %s", providerType)
	}

	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	defer provider.Close()

	// Create app server
	server, err := appserver.New(appserver.Config{
		WorkspacePath: workDir,
		Version:       version.ShortVersion(),
		Provider:      provider,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nShutting down server...")
		cancel()
	}()

	// Start server on stdin/stdout
	log.Println("Starting JSON-RPC server on stdin/stdout...")
	log.Printf("Provider: %s, Model: %s", providerType, model)
	log.Printf("Workspace: %s", workDir)

	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		if err != context.Canceled {
			return fmt.Errorf("server error: %w", err)
		}
	}

	log.Println("Server stopped")
	return nil
}
