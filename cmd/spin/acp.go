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
	"github.com/dmytrogajewski/spin/internal/agent/runtime"
	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/builder"
	"github.com/dmytrogajewski/spin/internal/mcp"
	acppkg "github.com/dmytrogajewski/spin/internal/protocol/acp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
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
	authMgr := createAuthManager()

	cfg, err := config.Load(config.Source{
		File: flagConfigFile,
		Flags: config.FlagOverrides{
			Provider: providerType,
			Model:    model,
			BaseURL:  baseURL,
		},
		WorkDir: workDir,
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx := context.Background()
	provider, err := buildProviderForACP(
		ctx,
		cfg,
		authMgr,
		cfg.LLM.Provider,
		cfg.LLM.BaseURL,
		cfg.LLM.Model,
		apiKey,
	)

	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	defer provider.Close()

	logger := slog.Default()
	protocolServices, cleanup, err := createServices(cfg, workDir, logger)

	if err != nil {
		return fmt.Errorf("failed to create services: %w", err)
	}

	defer cleanup()

	bufferSize := 100

	if cfg.Agent.StreamBuffer > 0 {
		bufferSize = cfg.Agent.StreamBuffer
	}

	emitter := events.NewEventEmitter(bufferSize)
	storageDir := cfg.Agent.SessionDir

	if storageDir == "" {
		storageDir = "~/.spin/sessions"
	}

	storage, err := session.NewFileStorage(storageDir)

	if err != nil {
		return fmt.Errorf("create session storage: %w", err)
	}

	acpRuntime, err := runtime.NewACP(runtime.ACPConfig{
		WorkDir:      workDir,
		Emitter:      emitter,
		Storage:      storage,
		ShellService: protocolServices.Shell,
		GitService:   protocolServices.Git,
		Logger:       logger,
	})

	if err != nil {
		return fmt.Errorf("create ACP runtime: %w", err)
	}

	coreAgent, err := buildCoreAgent(cfg, provider, workDir, emitter, acpRuntime)

	if err != nil {
		return fmt.Errorf("build core agent: %w", err)
	}

	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, logger)
	acpAgent, err := acppkg.NewSpinACPAgentWithStorage(coreAgent, mcpManager, emitter, storage)

	if err != nil {
		return fmt.Errorf("create ACP protocol adapter: %w", err)
	}

	acpRuntime.SetACPAgent(acpAgent)
	acpApprovalHandler := acppkg.NewACPApprovalHandler(acpAgent, 60*time.Second)
	acpRuntime.SetApprovalHandler(acpApprovalHandler.HandleApprovalRequest)
	acpAgent.SetApprovalHandler(acpApprovalHandler)
	acpAgent.SetApprovalService(coreAgent.GetSecurityService().ApprovalService())
	conn := acp.NewAgentSideConnection(acpAgent, os.Stdout, os.Stdin)
	acpAgent.SetConnection(conn)
	terminalClient := acppkg.NewACPTerminalClient(conn)
	acpRuntime.SetTerminalClient(terminalClient)
	filesystemClient := acppkg.NewACPFilesystemClient(conn)
	acpRuntime.SetFilesystemClient(filesystemClient)
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	setupACPServerSignalHandling(cancel)
	logACPServerStart(providerType, model, workDir)

	select {
	case <-conn.Done():
		log.Println("ACP client disconnected")
	case <-ctx.Done():
		log.Println("ACP server shutting down")
	}

	return nil
}

// buildProviderForACP creates and configures an LLM provider for the ACP server.
func buildProviderForACP(ctx context.Context, cfg *config.ConfigV2, authMgr *auth.Manager, providerType, baseURL, model, apiKey string) (llm.Provider, error) {
	if extra, ok, err := createProviderForACPExtra(providerType, baseURL, model, apiKey); err != nil {
		return nil, err
	} else if ok {
		return extra, nil
	}

	b := builder.NewBuilder(cfg, authMgr)

	return b.Build(ctx)
}

// buildCoreAgent constructs the core agent with all required services and dependencies.
func buildCoreAgent(
	cfg *config.ConfigV2,
	provider llm.Provider,
	workDir string,
	emitter *events.EventEmitter,
	rt runtime.Runtime,
) (*agent.Agent, error) {
	agentBuilder := agent.NewBuilder().
		WithConfig(cfg).
		WithProvider(provider).
		WithWorkingDir(workDir).
		WithEmitter(emitter).
		WithRuntime(rt)

	environment := agentBuilder.BuildEnvironment()
	securityService := agentBuilder.BuildSecurityService()
	detectionService := agentBuilder.BuildDetectionService()
	planningService := agentBuilder.BuildPlanningService()
	executor := agentBuilder.BuildExecutor()
	toolExecutor := agent.NewToolExecutorAdapter(executor)

	if acpRT, ok := rt.(*runtime.ACPRuntime); ok {
		acpRT.SetExecutor(toolExecutor)
		acpRT.SetValidator(runtime.NewValidatorAdapter(securityService.Validator()))
	}

	toolRegistry := tools.NewRegistry()
	rt.RegisterTools(toolRegistry)

	toolRuntime := agent.NewToolRuntime(agent.ToolRuntimeConfig{
		Registry:        toolRegistry,
		Validator:       securityService.Validator(),
		ApprovalService: securityService.ApprovalService(),
		Emitter:         emitter,
		WorkDir:         workDir,
	})

	opts := agentBuilder.BuildAgentOptions()

	if cfg != nil && cfg.ACE.Enabled {
		aceSvc, err := agentBuilder.BuildACEService()

		if err == nil {
			opts = append(opts, agent.WithACEService(aceSvc))
			aceConfig := agent.ConvertACEConfig(&cfg.ACE)
			opts = append(opts, agent.WithACEConfig(aceConfig))
		}
	}

	agentInstance, err := agent.NewAgent(provider, securityService, detectionService, toolRuntime, planningService, environment, emitter, opts...)

	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	return agentInstance, nil
}

// setupACPServerSignalHandling configures signal handlers for graceful shutdown.
func setupACPServerSignalHandling(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nShutting down ACP server...")
		cancel()
	}()
}

// logACPServerStart logs the ACP server startup information.
func logACPServerStart(providerType, model, workDir string) {
	log.Println("Starting ACP server on stdin/stdout...")
	log.Printf("Provider: %s, Model: %s", providerType, model)
	log.Printf("Workspace: %s", workDir)
}
