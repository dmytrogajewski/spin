// Package main provides the spin CLI application entry point.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/agent"
	agentexec "github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/builder"
	"github.com/dmytrogajewski/spin/internal/mcp"
	acppkg "github.com/dmytrogajewski/spin/internal/protocol/acp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// newACPCmd creates the ACP server command.
func newACPCmd() *cobra.Command {
	var (
		workDir string
		apiKey  string
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Only pass flag values if they were explicitly set.
			providerType, _ := cmd.Flags().GetString("provider")
			baseURL, _ := cmd.Flags().GetString("base-url")
			model, _ := cmd.Flags().GetString("model")

			// Check if flags were explicitly set by user.
			flagOverrides := config.FlagOverrides{}
			if cmd.Flags().Changed("provider") {
				flagOverrides.Provider = providerType
			}

			if cmd.Flags().Changed("model") {
				flagOverrides.Model = model
			}

			if cmd.Flags().Changed("base-url") {
				flagOverrides.BaseURL = baseURL
			}

			return runACPServer(cmd, workDir, flagOverrides, apiKey)
		},
	}

	// Server-specific flags (empty defaults - config file values take precedence).
	cmd.Flags().StringVar(&workDir, "workspace", ".", "Workspace directory path")
	cmd.Flags().String("provider", "", "LLM provider type (ollama, openai)")
	cmd.Flags().String("base-url", "", "Provider base URL")
	cmd.Flags().String("model", "", "Model name")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key (for cloud providers)")

	return cmd
}

// acpInfra holds the infrastructure components for the ACP server.
type acpInfra struct {
	emitter  *events.EventEmitter
	storage  session.Storage
	provider llm.Provider
	services *ProtocolServices
	cleanup  func()
	logger   *slog.Logger
}

// createACPInfra creates the infrastructure components for the ACP server.
func createACPInfra(ctx context.Context, cmd *cobra.Command, workDir string, flagOverrides config.FlagOverrides, apiKey string) (*config.V2, *acpInfra, error) {
	authMgr := createAuthManager()

	cfg, err := config.Load(config.Source{
		File:    flagConfigFile(cmd),
		Flags:   flagOverrides,
		WorkDir: workDir,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	if agentsMD := flagAgentsMD(cmd); agentsMD != "" {
		cfg.AgentsMD.Path = agentsMD
	}

	provider, err := buildProviderForACP(ctx, cfg, authMgr, cfg.LLM.Provider, cfg.LLM.BaseURL, cfg.LLM.Model, apiKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create provider: %w", err)
	}

	logger := slog.Default()

	services, cleanup, err := createServices(ctx, cfg, workDir, logger)
	if err != nil {
		provider.Close()
		return nil, nil, fmt.Errorf("failed to create services: %w", err)
	}

	bufferSize := 100
	if cfg.Agent.StreamBuffer > 0 {
		bufferSize = cfg.Agent.StreamBuffer
	}

	storageDir := cfg.Agent.SessionDir
	if storageDir == "" {
		storageDir = "~/.spin/sessions"
	}

	storage, err := session.NewFileStorage(storageDir)
	if err != nil {
		cleanup()
		provider.Close()
		return nil, nil, fmt.Errorf("create session storage: %w", err)
	}

	return cfg, &acpInfra{
		emitter:  events.NewEventEmitter(bufferSize),
		storage:  storage,
		provider: provider,
		services: services,
		cleanup:  cleanup,
		logger:   logger,
	}, nil
}

// wireACPAgent configures all the connections between ACP components.
func wireACPAgent(
	acpAgent *acppkg.SpinACPAgent,
	acpRuntime *agentexec.ACPRuntime,
	coreAgent *agent.Agent,
	convManager *conversation.Manager,
	histStorage history.Storage,
) *acp.AgentSideConnection {
	acpAgent.SetConversationManager(convManager)
	acpAgent.SetHistoryStorage(histStorage)

	acpRuntime.SetACPAgent(acpAgent)
	acpAgent.SetACPRuntime(acpRuntime)

	acpApprovalHandler := acppkg.NewApprovalHandler(acpAgent, 60*time.Second)
	acpRuntime.SetApprovalHandler(acpApprovalHandler.HandleApprovalRequest)
	acpAgent.SetApprovalHandler(acpApprovalHandler)
	acpAgent.SetApprovalService(coreAgent.GetSecurityService().ApprovalService())

	conn := acp.NewAgentSideConnection(acpAgent, os.Stdout, os.Stdin)
	acpAgent.SetConnection(conn)

	terminalClient := acppkg.NewTerminalClient(conn)
	acpRuntime.SetTerminalClient(terminalClient)

	filesystemClient := acppkg.NewFilesystemClient(conn)
	acpRuntime.SetFilesystemClient(filesystemClient)

	return conn
}

// createACPConversationManager creates the conversation manager for the ACP server.
func createACPConversationManager(cfg *config.V2, coreAgent *agent.Agent, infra *acpInfra) (*conversation.Manager, history.Storage, error) {
	historyDir := cfg.Agent.SessionDir
	if historyDir == "" {
		historyDir = "~/.spin/sessions"
	}

	histStorage, err := history.NewFileStorage(historyDir)
	if err != nil {
		return nil, nil, fmt.Errorf("create history storage: %w", err)
	}

	maxTokens := getHistoryMaxTokens(cfg, infra.provider)

	convFactory := func(_ context.Context, sessionID string, sessWorkDir string) (*conversation.Conversation, error) {
		return conversation.NewFromAgent(conversation.NewFromAgentConfig{
			Agent:     coreAgent,
			Emitter:   infra.emitter,
			WorkDir:   sessWorkDir,
			ID:        sessionID,
			MaxTokens: maxTokens,
		})
	}

	convManager, err := conversation.NewManager(conversation.ManagerConfig{
		Factory:        convFactory,
		Storage:        infra.storage,
		HistoryStorage: histStorage,
		Logger:         infra.logger,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create conversation manager: %w", err)
	}

	return convManager, histStorage, nil
}

// runACPServer starts the ACP server.
func runACPServer(cmd *cobra.Command, workDir string, flagOverrides config.FlagOverrides, apiKey string) error {
	var err error

	workDir, err = filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("resolve workspace path: %w", err)
	}

	ctx := context.Background()

	cfg, infra, err := createACPInfra(ctx, cmd, workDir, flagOverrides, apiKey)
	if err != nil {
		return err
	}
	defer infra.provider.Close()
	defer infra.cleanup()

	acpRuntime, err := agentexec.NewACP(agentexec.ACPConfig{
		WorkDir:      workDir,
		Emitter:      infra.emitter,
		Storage:      infra.storage,
		ShellService: infra.services.Shell,
		GitService:   infra.services.Git,
		Logger:       infra.logger,
	})
	if err != nil {
		return fmt.Errorf("create ACP runtime: %w", err)
	}

	coreAgent, err := buildCoreAgent(ctx, cfg, infra.provider, workDir, infra.emitter, acpRuntime)
	if err != nil {
		return fmt.Errorf("build core agent: %w", err)
	}

	mcpService := mcp.NewService(mcp.NewDefaultRegistryManager(infra.logger))

	acpAgent, err := acppkg.NewSpinACPAgentWithStorage(coreAgent, mcpService, infra.emitter, infra.storage)
	if err != nil {
		return fmt.Errorf("create ACP protocol adapter: %w", err)
	}

	convManager, histStorage, err := createACPConversationManager(cfg, coreAgent, infra)
	if err != nil {
		return err
	}

	conn := wireACPAgent(acpAgent, acpRuntime, coreAgent, convManager, histStorage)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	setupACPServerSignalHandling(cancel)
	logACPServerStart(cfg.LLM.Provider, cfg.LLM.Model, workDir)

	select {
	case <-conn.Done():
		log.Println("ACP client disconnected")
	case <-ctx.Done():
		log.Println("ACP server shutting down")
	}

	return nil
}

// getHistoryMaxTokens determines appropriate max tokens for history based on LLM context window.
// Priority order:
//  1. Config context_window override (if set - for custom/fine-tuned models)
//  2. Provider's auto-detected context window (from Capabilities)
//  3. Default of 8192 tokens
//
// Note: LLM.MaxTokens is intentionally NOT used here - it's for generation limit,
// not context window. Providers should report context window via Capabilities().
func getHistoryMaxTokens(cfg *config.V2, provider llm.Provider) int {
	const (
		defaultTokens = 8192
		minTokens     = 2048
		maxTokens     = 128000 // Cap to prevent excessive memory usage.
	)

	var contextWindow int

	// Priority 1: Config override for custom/fine-tuned models.
	if cfg != nil && cfg.LLM.ContextWindow > 0 {
		contextWindow = cfg.LLM.ContextWindow
	}

	// Priority 2: Provider's auto-detected context window (primary mechanism).
	if contextWindow == 0 && provider != nil {
		caps := provider.Capabilities()
		if caps.ContextWindow > 0 {
			contextWindow = caps.ContextWindow
		}
	}

	// Priority 3: Default.
	if contextWindow == 0 {
		return defaultTokens
	}

	// Use 75% of context window for history (leave room for responses).
	historyTokens := min(
		// Apply constraints.
		int(float64(contextWindow)*0.75), maxTokens)

	if historyTokens < minTokens {
		historyTokens = defaultTokens
	}

	return historyTokens
}

// buildProviderForACP creates and configures an LLM provider for the ACP server.
func buildProviderForACP(ctx context.Context, cfg *config.V2, authMgr *auth.Manager, providerType, baseURL, model, apiKey string) (llm.Provider, error) {
	extra, ok, err := createProviderForACPExtra(providerType, baseURL, model, apiKey)
	if err != nil {
		return nil, err
	} else if ok {
		return extra, nil
	}

	b := builder.NewBuilder(cfg, authMgr)

	return b.Build(ctx)
}

// buildCoreAgent constructs the core agent with all required services and dependencies.
func buildCoreAgent(
	ctx context.Context,
	cfg *config.V2,
	provider llm.Provider,
	workDir string,
	emitter *events.EventEmitter,
	rt agentexec.Runtime,
) (*agent.Agent, error) {
	agentBuilder := agent.NewBuilder().
		WithConfig(cfg).
		WithProvider(provider).
		WithWorkingDir(workDir).
		WithEmitter(emitter).
		WithRuntime(rt)

	environment := agentBuilder.BuildEnvironment(ctx)
	securityService := agentBuilder.BuildSecurityService()
	detectionService := agentBuilder.BuildDetectionService()
	planningService := agentBuilder.BuildPlanningService()
	executor := agentBuilder.BuildExecutor()
	toolExecutor := agent.NewToolExecutorAdapter(executor)

	if acpRT, ok := rt.(*agentexec.ACPRuntime); ok {
		acpRT.SetExecutor(toolExecutor)
		acpRT.SetValidator(agentexec.NewValidatorAdapter(securityService.Validator()))
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

	opts := agentBuilder.BuildOptions()

	if cfg != nil && cfg.ACE.Enabled {
		aceSvc, err := agentBuilder.BuildACEService(ctx)
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
