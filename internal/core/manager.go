package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/core/cycle"
	"github.com/dmytrogajewski/spin/internal/core/history"
	"github.com/dmytrogajewski/spin/internal/core/history/compress"
	"github.com/dmytrogajewski/spin/internal/core/task"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Manager coordinates conversation lifecycle and state management.
// It serves as the main entry point for creating and managing conversations.
type Manager struct {
	cfg              *Config
	llm              llm.Provider
	emitter          *EventEmitter
	storage          session.Storage
	toolRegistry     *tools.Registry
	taskRegistry     *task.Registry // Task registry for all conversations
	approvalHandler  ApprovalHandler
	mcpManager       *MCPManager       // MCP server manager
	gitIntegration   *GitIntegration   // Git integration
	shellIntegration *ShellIntegration // Shell integration
	logger           *slog.Logger      // Logger for manager operations
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
func WithManagerApprovalHandler(handler ApprovalHandler) ManagerOption {
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

// buildExecutor creates a configured executor for command execution.
func (m *Manager) buildExecutor(workDir string, logger *slog.Logger) (*Executor, error) {
	validator := NewValidator()

	// Build approval service if handler configured
	var approvalService *ApprovalService
	if m.approvalHandler != nil {
		approvalService = NewApprovalService(m.approvalHandler)
	}

	// Build executor options
	opts := m.buildExecutorOptions(validator, approvalService, logger)

	executor, err := NewExecutor(workDir, opts...)
	if err != nil {
		logger.Error("failed to create executor", "error", err, "work_dir", workDir)
		return nil, err
	}

	return executor, nil
}

// buildExecutorOptions creates executor configuration options.
func (m *Manager) buildExecutorOptions(validator *Validator, approvalService *ApprovalService, logger *slog.Logger) []ExecutorOption {
	var opts []ExecutorOption

	// Add validator
	opts = append(opts, WithValidator(validator))

	// Add approval service if available
	if approvalService != nil {
		opts = append(opts, WithApprovalService(approvalService))
	}

	// Apply configuration options
	if m.cfg != nil {
		if m.cfg.Timeout > 0 {
			opts = append(opts, WithTimeout(m.cfg.Timeout))
		}
		if m.cfg.CacheCommands {
			cache := NewCommandCache(5*time.Minute, 10*1024*1024) // 5 min TTL, 10MB max
			opts = append(opts, WithCache(cache))
			logger.Debug("enabled command caching")
		}
		if m.cfg.SandboxMode != "" {
			logger.Debug("sandbox mode configured", "mode", m.cfg.SandboxMode)
		}
	}

	return opts
}

// gatherEnvironmentContext collects environment information for the agent.
func (m *Manager) gatherEnvironmentContext(workDir string, logger *slog.Logger) (*Environment, error) {
	opts := m.buildEnvironmentOptions()

	ctxEnv, err := GatherEnvironment(workDir, opts...)
	if err != nil {
		logger.Error("failed to gather environment", "error", err, "work_dir", workDir)
		return nil, err
	}

	// Enrich with integration contexts
	m.enrichEnvironmentWithIntegrations(ctxEnv, logger)

	return ctxEnv, nil
}

// buildEnvironmentOptions creates environment gathering options from config.
func (m *Manager) buildEnvironmentOptions() []EnvironmentOption {
	var opts []EnvironmentOption

	if m.cfg != nil {
		if m.cfg.MaxFiles > 0 {
			opts = append(opts, WithMaxFiles(m.cfg.MaxFiles))
		}
		if m.cfg.MaxDepth > 0 {
			opts = append(opts, WithMaxDepth(m.cfg.MaxDepth))
		}
		if m.cfg.SkipGit {
			opts = append(opts, WithSkipGit(true))
		}
	}

	return opts
}

// enrichEnvironmentWithIntegrations adds context from active integrations.
func (m *Manager) enrichEnvironmentWithIntegrations(env *Environment, logger *slog.Logger) {
	if m.gitIntegration != nil && m.gitIntegration.IsRepository() {
		m.addGitContext(env, logger)
	}

	if m.shellIntegration != nil && m.shellIntegration.IsEnabled() {
		m.addShellContext(env, logger)
	}
}

// addGitContext merges git information into environment.
func (m *Manager) addGitContext(env *Environment, logger *slog.Logger) {
	gitInfo := m.gitIntegration.GetContextInfo()
	for key, value := range gitInfo {
		if strValue, ok := value.(string); ok {
			env.Environment[key] = strValue
		}
	}
	logger.Debug("added Git context", "git_info", gitInfo)
}

// addShellContext merges shell information into environment.
func (m *Manager) addShellContext(env *Environment, logger *slog.Logger) {
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

	gitTool := NewGitOperationTool(m.gitIntegration)
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

	shellTool := NewShellOperationTool(m.shellIntegration)
	if err := m.toolRegistry.Register(shellTool); err != nil {
		logger.Warn("failed to register Shell operation tool", "error", err)
		return err
	}

	logger.Debug("registered Shell operation tool")
	return nil
}

// buildAgent creates a configured agent for conversation.
func (m *Manager) buildAgent(executor *Executor, ctxEnv *Environment, logger *slog.Logger) (*Agent, error) {
	validator := NewValidator()

	// Register integration tools
	if err := m.registerIntegrationTools(logger); err != nil {
		logger.Error("failed to register integration tools", "error", err)
		return nil, err
	}

	// Build agent options
	opts := m.buildAgentOptions(logger)

	agent, err := NewAgent(m.llm, executor, validator, ctxEnv, m.emitter, opts...)
	if err != nil {
		logger.Error("failed to create agent", "error", err)
		return nil, err
	}

	return agent, nil
}

// buildAgentOptions creates agent configuration options.
func (m *Manager) buildAgentOptions(logger *slog.Logger) []AgentOption {
	var opts []AgentOption

	// Enable approval for dangerous commands
	opts = append(opts, WithRequireApproval(true))

	// Apply configuration options from Manager config
	if m.cfg != nil {
		if m.cfg.MaxTurns > 0 {
			opts = append(opts, WithMaxTurns(m.cfg.MaxTurns))
		}
		if m.cfg.Timeout > 0 {
			opts = append(opts, WithAgentTimeout(m.cfg.Timeout))
		}
		if m.cfg.Temperature > 0 {
			opts = append(opts, WithTemperature(m.cfg.Temperature))
		}
		if m.cfg.MaxTokens > 0 {
			opts = append(opts, WithMaxTokens(m.cfg.MaxTokens))
		}
	}

	// Wire approval handler if configured
	if m.approvalHandler != nil {
		opts = append(opts, WithApprovalHandler(m.approvalHandler))
	}

	// Wire cycle detection if enabled
	if m.cfg != nil && m.cfg.CycleDetection.Enabled {
		cycleConfig := cycle.Config{
			WindowSize:       m.cfg.CycleDetection.WindowSize,
			SimilarityThresh: m.cfg.CycleDetection.SimilarityThresh,
			ToolRepeatLimit:  m.cfg.CycleDetection.ToolRepeatLimit,
			ErrorRepeatLimit: m.cfg.CycleDetection.ErrorRepeatLimit,
			Enabled:          m.cfg.CycleDetection.Enabled,
		}
		patternDetector := cycle.NewPatternDetector(cycleConfig)
		opts = append(opts, WithPatternDetector(patternDetector))
		logger.Debug("enabled cycle detection")
	}

	// Add tool registry if configured
	if m.toolRegistry != nil {
		opts = append(opts, WithToolRegistry(m.toolRegistry))
		logger.Debug("using custom tool registry", "tool_count", len(m.toolRegistry.ListSchemas()))
	}

	// Pass task registry if configured
	if m.taskRegistry != nil {
		opts = append(opts, WithTaskRegistry(m.taskRegistry))
		logger.Debug("using custom task registry", "task_count", len(m.taskRegistry.List()))
	}

	return opts
}

// createHistory creates a conversation history with compression support.
func (m *Manager) createHistory() *History {
	hist := NewHistoryWithDefaults()

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
		m.emitter = NewEventEmitter(DefaultEventBufferSize)
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
		mcpManager := NewMCPManager(cfg, logger)
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
		gitIntegration := NewGitIntegration(true, cfg.WorkDir, logger)
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
		shellIntegration := NewShellIntegration(true, cfg.WorkDir, logger)
		if err := shellIntegration.Initialize(context.Background()); err != nil {
			// Log error but don't fail manager creation
			logger.Error("Failed to initialize Shell integration", "error", err)
		} else {
			m.shellIntegration = shellIntegration
		}
	}

	return m, nil
}

// NewConversation starts a new conversation for the given workDir
func (m *Manager) NewConversation(ctx context.Context, workDir string) (*Conversation, error) {
	logger := withContext(ctx)
	logger.Info("creating new conversation", "work_dir", workDir)

	if workDir == "" {
		workDir = m.cfg.WorkDir
	}

	// Step 1: Build executor
	executor, err := m.buildExecutor(workDir, logger)
	if err != nil {
		return nil, err
	}

	// Step 2: Gather environment
	ctxEnv, err := m.gatherEnvironmentContext(workDir, logger)
	if err != nil {
		return nil, err
	}

	// Step 3: Build agent
	agent, err := m.buildAgent(executor, ctxEnv, logger)
	if err != nil {
		return nil, err
	}

	// Step 4: Create history
	hist := m.createHistory()

	// Step 5: Create conversation
	conv := NewConversation(agent, hist, m.emitter)
	logger.Info("conversation created successfully")
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
