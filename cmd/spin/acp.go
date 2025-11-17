package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/ollama"
	"github.com/dmytrogajewski/spin/internal/llm/openai"
	"github.com/dmytrogajewski/spin/internal/mcp"
	acppkg "github.com/dmytrogajewski/spin/internal/protocol/acp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/spf13/cobra"
)

// newACPCmd creates the ACP server command.
func newACPCmd() *cobra.Command {
	var (
		workDir      string
		providerType string
		baseURL      string
		model        string
		apiKey       string
	)

	cmd := &cobra.Command{
		Use:   "acp",
		Short: "Start ACP (Agent Client Protocol) server",
		Long: `Start Spin as an ACP (Agent Client Protocol) server.

The server communicates via stdin/stdout using ACP protocol.
This mode is designed for ACP-compatible clients that want to integrate
Spin as an agent.

Examples:
  spin acp
  spin acp --provider openai --model gpt-4
  spin acp --workspace /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runACPServer(workDir, providerType, baseURL, model, apiKey)
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

// runACPServer starts the ACP server.
func runACPServer(workDir, providerType, baseURL, model, apiKey string) error {
	provider, err := createProviderForACP(providerType, baseURL, model, apiKey)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	defer provider.Close()

	// Load config for ACP mode
	cfg, err := loadConfigForMode("", 0, workDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create conversation using unified builder with temporary auto-approve handler
	// We'll update the approval handler after creating the ACP agent
	ctx := context.Background()
	conv, err := createACPConversation(ctx, workDir, provider, cfg)
	if err != nil {
		return fmt.Errorf("failed to create ACP conversation: %w", err)
	}

	// Extract agent and emitter from conversation
	agentInstance := conv.GetAgent()
	emitter := conv.GetEmitter()

	// Create session storage for persistence
	storageDir := cfg.Agent.SessionDir
	if storageDir == "" {
		// Default to ~/.spin/sessions if not configured
		storageDir = "~/.spin/sessions"
	}
	storage, err := session.NewFileStorage(storageDir)
	if err != nil {
		return fmt.Errorf("failed to create session storage: %w", err)
	}

	// Create MCP manager (separate from conversation's MCP service)
	// The conversation's MCP service is for tool integration, this is for ACP protocol
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())

	// Create ACP agent with storage
	acpAgent, err := acppkg.NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	if err != nil {
		return fmt.Errorf("failed to create ACP agent: %w", err)
	}

	// Create ACP approval handler
	approvalHandler := acppkg.NewACPApprovalHandler(acpAgent, 60*time.Second)

	// Update approval service with ACP handler using unified builder logic
	// This ensures ACP mode uses the same policy store logic as other modes (TUI/Exec)
	agentBuilder := agent.NewBuilder().
		WithConfig(cfg).
		WithWorkingDir(workDir).
		WithEmitter(emitter).
		WithApprovalHandler(approvalHandler.HandleApprovalRequest)

	// Build a new security service using unified builder (handles policy store config properly)
	newSecurityService := agentBuilder.BuildSecurityService()
	newApprovalService := newSecurityService.ApprovalService()

	// Update the agent's approval service
	// The agent was created by conversation.Builder, so we update it with the ACP handler
	agentInstance.SetApprovalService(newApprovalService)

	// Wire the approval service + handler into the ACP adapter
	acpAgent.SetApprovalHandler(approvalHandler)
	acpAgent.SetApprovalService(newApprovalService)

	// Create ACP connection (starts automatically)
	conn := acp.NewAgentSideConnection(acpAgent, os.Stdout, os.Stdin)

	// Set connection on agent for sending notifications
	acpAgent.SetConnection(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupACPServerSignalHandling(cancel)
	logACPServerStart(providerType, model, workDir)

	// Wait for connection to finish (blocks until peer disconnects or context cancelled)
	select {
	case <-conn.Done():
		log.Println("ACP client disconnected")
	case <-ctx.Done():
		log.Println("ACP server shutting down")
	}

	return nil
}

// createProviderForACP creates an LLM provider for ACP server.
func createProviderForACP(providerType, baseURL, model, apiKey string) (llm.Provider, error) {
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
		if extra, ok, err := createProviderForACPExtra(providerType, baseURL, model, apiKey); err != nil {
			return nil, err
		} else if ok {
			return extra, nil
		}
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// createACPConversation creates a conversation for ACP mode using conversation.Builder.
// Uses a temporary auto-approve handler initially, which will be updated after ACP agent creation.
// Uses unified protocol services setup like TUI/EXEC modes.
func createACPConversation(ctx context.Context, workDir string, provider llm.Provider, cfg *config.ConfigV2) (*conversation.Conversation, error) {
	// Use temporary auto-approve handler - will be updated after ACP agent is created
	approvalHandler := createAutoApproveHandler()

	// Create services based on configuration (unified with TUI/EXEC)
	logger := slog.Default()
	protocolServices, cleanup, err := createProtocolServices(cfg, workDir, logger)
	if err != nil {
		return nil, err
	}

	// Build conversation with services (unified with TUI/EXEC)
	builder := conversation.NewBuilder(cfg, workDir).
		WithLLM(provider).
		WithApprovalHandler(approvalHandler)

	if protocolServices.Git != nil {
		builder = builder.WithGit(protocolServices.Git)
	}
	if protocolServices.Shell != nil {
		builder = builder.WithShell(protocolServices.Shell)
	}
	if protocolServices.MCP != nil {
		builder = builder.WithMCP(protocolServices.MCP)
	}

	conv, err := builder.Build(ctx)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("build conversation: %w", err)
	}

	return conv, nil
}

// setupACPServerSignalHandling sets up signal handling for graceful shutdown.
func setupACPServerSignalHandling(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nShutting down ACP server...")
		cancel()
	}()
}

// logACPServerStart logs server startup information.
func logACPServerStart(providerType, model, workDir string) {
	log.Println("Starting ACP server on stdin/stdout...")
	log.Printf("Provider: %s, Model: %s", providerType, model)
	log.Printf("Workspace: %s", workDir)
}
