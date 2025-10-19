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
	provider, err := createProvider(providerType, baseURL, model, apiKey)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	defer provider.Close()

	server, err := createAppServer(workDir, provider)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupServerSignalHandling(cancel)
	logServerStart(providerType, model, workDir)

	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		if err != context.Canceled {
			return fmt.Errorf("server error: %w", err)
		}
	}

	log.Println("Server stopped")
	return nil
}

// createProvider creates an LLM provider based on the provider type.
func createProvider(providerType, baseURL, model, apiKey string) (llm.Provider, error) {
	switch providerType {
	case "ollama":
		return ollama.NewProvider(ollama.Config{
			BaseURL: baseURL,
			Model:   model,
			Timeout: llm.DefaultTimeout,
		})
	case "openai":
		return openai.NewProvider(openai.Config{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   model,
			Timeout: llm.DefaultTimeout,
		})
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// createAppServer creates the application server.
func createAppServer(workDir string, provider llm.Provider) (*appserver.Server, error) {
	return appserver.New(appserver.Config{
		WorkspacePath: workDir,
		Version:       version.ShortVersion(),
		Provider:      provider,
	})
}

// setupServerSignalHandling sets up signal handling for graceful shutdown.
func setupServerSignalHandling(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nShutting down server...")
		cancel()
	}()
}

// logServerStart logs server startup information.
func logServerStart(providerType, model, workDir string) {
	log.Println("Starting JSON-RPC server on stdin/stdout...")
	log.Printf("Provider: %s, Model: %s", providerType, model)
	log.Printf("Workspace: %s", workDir)
}
