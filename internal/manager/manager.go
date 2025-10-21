package manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/history/compress"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Manager coordinates conversation lifecycle and state management.
// It serves as the main entry point for creating and managing conversations.
type Manager struct {
	cfg              *Config
	llm              llm.Provider
	emitter          *events.EventEmitter
	storage          session.Storage
	toolRegistry     *tools.Registry
	taskRegistry     *orchestration.Registry // Task registry for all conversations
	approvalHandler  security.ApprovalHandler
	authManager      *auth.Manager           // Credential management
	mcpManager       *mcp.MCPManager         // MCP server manager
	gitIntegration   *git.GitIntegration     // Git integration
	shellIntegration *shell.ShellIntegration // Shell integration
	logger           *slog.Logger            // Logger for manager operations
}

// Functional options
type ManagerOption func(*Manager) error

// WithLLM sets the LLM provider for the manager
func WithLLM(provider llm.Provider) ManagerOption {
	return func(m *Manager) error {
		m.llm = provider
		return nil
	}
}

// WithManagerToolRegistry sets a custom tool registry for all conversations created by this manager
func WithManagerToolRegistry(registry *tools.Registry) ManagerOption {
	return func(m *Manager) error {
		m.toolRegistry = registry
		return nil
	}
}

// WithManagerApprovalHandler sets the approval handler for all agents created by this manager
func WithManagerApprovalHandler(handler security.ApprovalHandler) ManagerOption {
	return func(m *Manager) error {
		m.approvalHandler = handler
		return nil
	}
}

// getLogger returns a logger from manager or falls back to default.
func (m *Manager) getLogger(ctx context.Context) *slog.Logger {
	if m.logger != nil {
		return m.logger
	}
	return slog.Default()
}

// withContext creates a logger with context fields extracted from ctx.
// This is a local helper function since logger.withContext is not exported.
func withContext(ctx context.Context) *slog.Logger {
	return slog.Default()
}

// buildExecutor creates a configured executor for command execution.
func (m *Manager) buildExecutor(workDir string, logger *slog.Logger) (*agent.Executor, error) {
	validator := security.NewValidator()

	// Build approval service if handler configured
	var approvalService *security.ApprovalService
	if m.approvalHandler != nil {
		approvalService = security.NewApprovalService(m.approvalHandler, m.emitter, validator)
	}

	// Build executor options
	opts := m.buildExecutorOptions(validator, approvalService, logger)

	executor, err := agent.NewExecutor(workDir, opts...)
	if err != nil {
		logger.Error("failed to create executor", "error", err, "work_dir", workDir)
		return nil, err
	}

	return executor, nil
}

// buildExecutorOptions creates executor configuration options.
func (m *Manager) buildExecutorOptions(validator *security.Validator, approvalService *security.ApprovalService, logger *slog.Logger) []agent.ExecutorOption {
	var opts []agent.ExecutorOption

	// Add validator
	opts = append(opts, agent.WithValidator(validator))

	// Add approval service if available
	if approvalService != nil {
		opts = append(opts, agent.WithApprovalService(approvalService))
	}

	// Apply configuration options
	if m.cfg != nil {
		if m.cfg.Timeout > 0 {
			opts = append(opts, agent.WithTimeout(m.cfg.Timeout))
		}
		if m.cfg.CacheCommands {
			cache := agent.NewCommandCache(5*time.Minute, 10*1024*1024) // 5 min TTL, 10MB max
			opts = append(opts, agent.WithCache(cache))
			logger.Debug("enabled command caching")
		}
		if m.cfg.SandboxMode != "" {
			logger.Debug("sandbox mode configured", "mode", m.cfg.SandboxMode)
		}
	}

	return opts
}

// gatherEnvironmentContext collects environment information for the agent.
func (m *Manager) gatherEnvironmentContext(workDir string, logger *slog.Logger) (*agent.Environment, error) {
	opts := m.buildEnvironmentOptions()

	ctxEnv, err := agent.GatherEnvironment(workDir, opts...)
	if err != nil {
		logger.Error("failed to gather environment", "error", err, "work_dir", workDir)
		return nil, err
	}

	// Enrich with integration contexts
	m.enrichEnvironmentWithIntegrations(ctxEnv, logger)

	return ctxEnv, nil
}

// buildEnvironmentOptions creates environment gathering options from config.
func (m *Manager) buildEnvironmentOptions() []agent.EnvironmentOption {
	var opts []agent.EnvironmentOption

	if m.cfg != nil {
		if m.cfg.MaxFiles > 0 {
			opts = append(opts, agent.WithMaxFiles(m.cfg.MaxFiles))
		}
		if m.cfg.MaxDepth > 0 {
			opts = append(opts, agent.WithMaxDepth(m.cfg.MaxDepth))
		}
		if m.cfg.SkipGit {
			opts = append(opts, agent.WithSkipGit(true))
		}
	}

	return opts
}

// enrichEnvironmentWithIntegrations adds context from active integrations.
func (m *Manager) enrichEnvironmentWithIntegrations(env *agent.Environment, logger *slog.Logger) {
	if m.gitIntegration != nil && m.gitIntegration.IsRepository() {
		m.addGitContext(env, logger)
	}

	if m.shellIntegration != nil && m.shellIntegration.IsEnabled() {
		m.addShellContext(env, logger)
	}
}

// addGitContext merges git information into environment.
func (m *Manager) addGitContext(env *agent.Environment, logger *slog.Logger) {
	gitInfo := m.gitIntegration.GetContextInfo()
	for key, value := range gitInfo {
		if strValue, ok := value.(string); ok {
			env.Environment[key] = strValue
		}
	}
	logger.Debug("added Git context", "git_info", gitInfo)
}

// addShellContext merges shell information into environment.
func (m *Manager) addShellContext(env *agent.Environment, logger *slog.Logger) {
	shellInfo := m.shellIntegration.GetContextInfo()
	for key, value := range shellInfo {
		if strValue, ok := value.(string); ok {
			env.Environment[key] = strValue
		}
	}
	logger.Debug("added Shell context", "shell_info", shellInfo)
}

// registerIntegrationTools registers tools from all active integrations.
func (m *Manager) registerIntegrationTools(logger *slog.Logger) error {
	// Ensure tool registry exists
	if m.toolRegistry == nil {
		m.toolRegistry = tools.NewRegistry()
	}

	// Register MCP tools
	if err := m.registerMCPTools(logger); err != nil {
		return fmt.Errorf("register MCP tools: %w", err)
	}

	// Register Git tools
	if err := m.registerGitTools(logger); err != nil {
		return fmt.Errorf("register Git tools: %w", err)
	}

	// Register Shell tools
	if err := m.registerShellTools(logger); err != nil {
		return fmt.Errorf("register Shell tools: %w", err)
	}

	return nil
}

// registerMCPTools registers tools from MCP manager.
func (m *Manager) registerMCPTools(logger *slog.Logger) error {
	if m.mcpManager == nil {
		return nil
	}

	mcpTools := m.mcpManager.GetTools()
	if len(mcpTools) == 0 {
		return nil
	}

	for _, tool := range mcpTools {
		if err := m.toolRegistry.Register(tool); err != nil {
			logger.Warn("failed to register MCP tool", "tool", tool.Name(), "error", err)
		} else {
			logger.Debug("registered MCP tool", "tool", tool.Name())
		}
	}

	logger.Info("registered MCP tools", "count", len(mcpTools))
	return nil
}

// registerGitTools registers Git operation tool if Git integration is active.
func (m *Manager) registerGitTools(logger *slog.Logger) error {
	if m.gitIntegration == nil || !m.gitIntegration.IsRepository() {
		return nil
	}

	gitTool := tools.NewGitOperationTool(m.gitIntegration)
	if err := m.toolRegistry.Register(gitTool); err != nil {
		logger.Warn("failed to register Git operation tool", "error", err)
		return err
	}

	logger.Debug("registered Git operation tool")
	return nil
}

// registerShellTools registers Shell operation tool if Shell integration is active.
func (m *Manager) registerShellTools(logger *slog.Logger) error {
	if m.shellIntegration == nil || !m.shellIntegration.IsEnabled() {
		return nil
	}

	shellTool := shell.NewShellOperationTool(m.shellIntegration)
	if err := m.toolRegistry.Register(shellTool); err != nil {
		logger.Warn("failed to register Shell operation tool", "error", err)
		return err
	}

	logger.Debug("registered Shell operation tool")
	return nil
}

// buildAgent creates a configured agent with service-based architecture.
func (m *Manager) buildAgent(executor *agent.Executor, ctxEnv *agent.Environment, logger *slog.Logger) (*agent.Agent, error) {
	// Build SecurityService
	validator := security.NewValidator()
	var approvalService *security.ApprovalService
	if m.approvalHandler != nil {
		approvalService = security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{
			Handler:         m.approvalHandler,
			Emitter:         m.emitter,
			Validator:       validator,
			ApprovalTimeout: m.cfg.ApprovalTimeout,
		})
	} else {
		approvalService = security.NewApprovalService(nil, m.emitter, validator)
	}
	securityService := security.NewSecurityService(validator, approvalService)

	// Build DetectionService
	var cycleDetector *cycle.Detector
	var patternDetector *cycle.PatternDetector
	if m.cfg != nil && m.cfg.CycleDetection.Enabled {
		cycleConfig := cycle.Config{
			WindowSize:       m.cfg.CycleDetection.WindowSize,
			SimilarityThresh: m.cfg.CycleDetection.SimilarityThresh,
			ToolRepeatLimit:  m.cfg.CycleDetection.ToolRepeatLimit,
			ErrorRepeatLimit: m.cfg.CycleDetection.ErrorRepeatLimit,
			Enabled:          m.cfg.CycleDetection.Enabled,
		}
		cycleDetector = cycle.NewDetector(cycleConfig)
		patternDetector = cycle.NewPatternDetector(cycleConfig)
		logger.Debug("enabled cycle detection")
	} else {
		cycleDetector = cycle.NewDetector(cycle.Config{Enabled: false})
		patternDetector = nil
	}
	detectionService := detection.NewDetectionService(cycleDetector, patternDetector)

	// Build tool registry with built-in and integration tools
	toolRegistry := m.buildToolRegistry(executor, validator, ctxEnv, logger)

	// Build task registry (use manager's registry or create default)
	taskRegistry := m.taskRegistry
	if taskRegistry == nil {
		taskRegistry = m.buildDefaultTaskRegistry(logger)
	}

	// Build ToolExecutor
	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         m.emitter,
		WorkDir:         ctxEnv.WorkDir,
	})

	// Build OrchestrationService
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	// Build agent options (only config options, no more registry/handler options)
	opts := m.buildAgentOptions(logger)

	// Create agent with services
	agentInstance, err := agent.NewAgent(m.llm, securityService, detectionService, orchestrationService, ctxEnv, m.emitter, opts...)
	if err != nil {
		logger.Error("failed to create agent", "error", err)
		return nil, err
	}

	return agentInstance, nil
}

// buildToolRegistry creates a tool registry with built-in and integration tools.
func (m *Manager) buildToolRegistry(executor *agent.Executor, validator *security.Validator, ctxEnv *agent.Environment, logger *slog.Logger) *tools.Registry {
	// Start with manager's tool registry if provided
	var registry *tools.Registry
	if m.toolRegistry != nil {
		registry = m.toolRegistry
		logger.Debug("using custom tool registry", "tool_count", len(registry.ListSchemas()))
	} else {
		registry = tools.NewRegistry()
	}

	// Register built-in tools
	_ = registry.Register(tools.NewReadFileTool())
	_ = registry.Register(tools.NewWriteFileTool())
	_ = registry.Register(tools.NewListDirectoryTool())
	_ = registry.Register(tools.NewExecuteCommandTool(executor, validator))
	_ = registry.Register(tools.NewGetContextTool(ctxEnv))
	_ = registry.Register(tools.NewApplyPatchTool(ctxEnv.WorkDir))
	_ = registry.Register(tools.NewFileSearchTool(ctxEnv.WorkDir))
	_ = registry.Register(tools.NewGitContextTool(ctxEnv.WorkDir))

	// Register integration tools
	if err := m.registerIntegrationTools(logger); err != nil {
		logger.Warn("failed to register integration tools", "error", err)
	}

	return registry
}

// buildDefaultTaskRegistry creates a default task registry with built-in modes.
func (m *Manager) buildDefaultTaskRegistry(logger *slog.Logger) *orchestration.Registry {
	taskRegistry := orchestration.NewRegistry()
	_ = taskRegistry.Register("regular", task.NewRegular())
	_ = taskRegistry.Register("review", task.NewReview())
	_ = taskRegistry.Register("compact", task.NewCompact())
	_ = taskRegistry.Register("planning", task.NewPlanning())
	_ = taskRegistry.SetDefault("regular")
	logger.Debug("created default task registry")
	return taskRegistry
}

// buildAgentOptions creates agent configuration options (config only, no services).
func (m *Manager) buildAgentOptions(logger *slog.Logger) []agent.AgentOption {
	var opts []agent.AgentOption

	// Enable approval for dangerous commands
	opts = append(opts, agent.WithRequireApproval(true))

	// Apply configuration options from Manager config
	if m.cfg != nil {
		if m.cfg.MaxTurns > 0 {
			opts = append(opts, agent.WithMaxTurns(m.cfg.MaxTurns))
		}
		if m.cfg.Timeout > 0 {
			opts = append(opts, agent.WithAgentTimeout(m.cfg.Timeout))
		}
		if m.cfg.Temperature > 0 {
			opts = append(opts, agent.WithTemperature(m.cfg.Temperature))
		}
		if m.cfg.MaxTokens > 0 {
			opts = append(opts, agent.WithMaxTokens(m.cfg.MaxTokens))
		}
	}

	return opts
}

// createHistory creates a conversation history with compression support.
func (m *Manager) createHistory() *history.History {
	hist := history.NewHistoryWithDefaults()

	if m.llm != nil {
		// Use composite compressor: LLM summarization (primary) + hybrid (fallback)
		adapter := history.NewLLMProviderAdapter(m.llm)
		compressor := compress.NewDefaultLLMWithHybridFallback(adapter)
		hist.SetCompressor(compressor)
	}

	// Set event emitter for compression notifications
	hist.SetEventEmitter(m.emitter)

	_ = hist.AddSystemMessage("You are a helpful AI coding assistant.")

	return hist
}

// NewManager creates a new Manager
func NewManager(cfg *Config, opts ...ManagerOption) (*Manager, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	m := &Manager{cfg: cfg}
	for _, opt := range opts {
		if err := opt(m); err != nil {
			return nil, err
		}
	}
	if m.llm == nil {
		m.llm = llm.NewMockProvider("default")
	}
	if m.emitter == nil {
		m.emitter = events.NewEventEmitter(100) // Default buffer size
	}
	if m.storage == nil {
		fs, err := session.NewFileStorage(cfg.SessionDir)
		if err != nil {
			return nil, fmt.Errorf("initialize storage: %w", err)
		}
		m.storage = fs
	}

	// Initialize MCP manager if MCP is enabled
	if cfg.EnableMCP {
		ctx := context.Background()
		logger := m.getLogger(ctx)
		// Convert manager.Config to mcp.Config
		mcpConfig := &mcp.Config{
			EnableMCP:  cfg.EnableMCP,
			MCPServers: make([]mcp.MCPServerConfig, len(cfg.MCPServers)),
		}
		for i, srv := range cfg.MCPServers {
			mcpConfig.MCPServers[i] = mcp.MCPServerConfig{
				Name:    srv.Name,
				Command: srv.Command,
				Args:    srv.Args,
				Env:     srv.Env,
			}
		}
		mcpManager := mcp.NewMCPManager(mcpConfig, logger)
		if err := mcpManager.Initialize(context.Background()); err != nil {
			// Log error but don't fail manager creation
			logger.Error("Failed to initialize MCP manager", "error", err)
		} else {
			m.mcpManager = mcpManager
		}
	}

	// Initialize Git integration if enabled
	if cfg.EnableGit {
		ctx := context.Background()
		logger := m.getLogger(ctx)
		gitIntegration := git.NewGitIntegration(true, cfg.WorkDir, logger)
		if err := gitIntegration.Initialize(context.Background()); err != nil {
			// Log error but don't fail manager creation
			logger.Error("Failed to initialize Git integration", "error", err)
		} else {
			m.gitIntegration = gitIntegration
		}
	}

	// Initialize Shell integration if enabled
	if cfg.EnableShell {
		ctx := context.Background()
		logger := m.getLogger(ctx)
		shellIntegration := shell.NewShellIntegration(true, cfg.WorkDir, logger)
		if err := shellIntegration.Initialize(context.Background()); err != nil {
			// Log error but don't fail manager creation
			logger.Error("Failed to initialize Shell integration", "error", err)
		} else {
			m.shellIntegration = shellIntegration
		}
	}

	// Initialize auth manager with platform-specific keystore
	keystore := auth.NewKeystore()
	m.authManager = auth.NewManager(keystore)

	return m, nil
}

// NewConversation starts a new conversation for the given workDir
func (m *Manager) NewConversation(ctx context.Context, workDir string) (*conversation.Conversation, error) {
	loggerInstance := withContext(ctx)
	loggerInstance.Info("creating new conversation", "work_dir", workDir)

	if workDir == "" {
		workDir = m.cfg.WorkDir
	}

	// Step 1: Build executor
	executor, err := m.buildExecutor(workDir, loggerInstance)
	if err != nil {
		return nil, err
	}

	// Step 2: Gather environment
	ctxEnv, err := m.gatherEnvironmentContext(workDir, loggerInstance)
	if err != nil {
		return nil, err
	}

	// Step 3: Build agent
	agentInstance, err := m.buildAgent(executor, ctxEnv, loggerInstance)
	if err != nil {
		return nil, err
	}

	// Step 4: Create session
	sess := session.NewSession(workDir)
	loggerInstance.Info("session created", "session_id", sess.ID)

	// Step 5: Create history
	hist := m.createHistory()

	// Step 6: Create conversation with session ID
	conv := conversation.NewConversation(agentInstance, hist, m.emitter, sess.ID)
	loggerInstance.Info("conversation created successfully", "session_id", sess.ID)
	return conv, nil
}

// Close closes the manager and cleans up resources.
func (m *Manager) Close() error {
	var err error

	// Close MCP manager if available
	if m.mcpManager != nil {
		if closeErr := m.mcpManager.Close(); closeErr != nil {
			err = closeErr
		}
	}

	// Close Git integration if available
	if m.gitIntegration != nil {
		if closeErr := m.gitIntegration.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
	}

	// Close Shell integration if available
	if m.shellIntegration != nil {
		if closeErr := m.shellIntegration.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
	}

	// Note: Storage interface doesn't have Close method

	// Close event emitter if available
	if m.emitter != nil {
		m.emitter.Close()
	}

	return err
}
