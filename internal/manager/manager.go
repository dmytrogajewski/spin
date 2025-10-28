package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/history"
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
	authManager      *auth.Manager       // Credential management
	mcpManager       *mcp.MCPManager     // MCP server manager
	gitIntegration   *git.GitIntegration // Git integration
	shellIntegration *shell.Context      // Shell context
	logger           *slog.Logger        // Logger for manager operations
}

// Functional options
type ManagerOption func(*Manager) error

// WithLLM sets the LLM provider for the manager.
// This option works with both NewManager and Builder.
func WithLLM(provider llm.Provider) ManagerOption {
	return func(m *Manager) error {
		m.llm = provider
		return nil
	}
}

// WithManagerToolRegistry sets a custom tool registry for all conversations created by this manager.
// This option works with both NewManager and Builder.
func WithManagerToolRegistry(registry *tools.Registry) ManagerOption {
	return func(m *Manager) error {
		m.toolRegistry = registry
		return nil
	}
}

// WithManagerApprovalHandler sets the approval handler for all agents created by this manager.
// This option works with both NewManager and Builder.
func WithManagerApprovalHandler(handler security.ApprovalHandler) ManagerOption {
	return func(m *Manager) error {
		m.approvalHandler = handler
		return nil
	}
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

	// Add git enabled status
	if gitInfo.GitEnabled {
		env.Environment["git_enabled"] = "true"
	} else {
		env.Environment["git_enabled"] = "false"
	}

	// Add repository status
	if gitInfo.IsRepo {
		env.Environment["is_repo"] = "true"

		// Add repository details
		if gitInfo.Branch != "" {
			env.Environment["branch"] = gitInfo.Branch
		}
		if gitInfo.Remote != "" {
			env.Environment["remote"] = gitInfo.Remote
		}
		if gitInfo.Commit != "" {
			env.Environment["commit"] = gitInfo.Commit
		}

		// Add working directory status
		if gitInfo.IsClean {
			env.Environment["is_clean"] = "true"
		} else {
			env.Environment["is_clean"] = "false"
		}

		// Add file counts (only if non-zero for cleaner output)
		if gitInfo.ModifiedFiles > 0 {
			env.Environment["modified_files"] = fmt.Sprintf("%d", gitInfo.ModifiedFiles)
		}
		if gitInfo.UntrackedFiles > 0 {
			env.Environment["untracked_files"] = fmt.Sprintf("%d", gitInfo.UntrackedFiles)
		}

		// Add remote sync status (only if non-zero)
		if gitInfo.Ahead > 0 {
			env.Environment["ahead"] = fmt.Sprintf("%d", gitInfo.Ahead)
		}
		if gitInfo.Behind > 0 {
			env.Environment["behind"] = fmt.Sprintf("%d", gitInfo.Behind)
		}

		// Add detached status
		if gitInfo.Detached {
			env.Environment["detached"] = "true"
		}
	} else {
		env.Environment["is_repo"] = "false"
	}

	logger.Debug("added Git context", "git_info", gitInfo)
}

// addShellContext merges shell information into environment.
func (m *Manager) addShellContext(env *agent.Environment, logger *slog.Logger) {
	shellInfo := m.shellIntegration.GetContextInfo()
	if shellInfo.ShellEnabled {
		env.Environment["shell_enabled"] = "true"
		if shellInfo.Shell != "" {
			env.Environment["shell"] = shellInfo.Shell
		}
		if shellInfo.ShellPath != "" {
			env.Environment["shell_path"] = shellInfo.ShellPath
		}
	} else {
		env.Environment["shell_enabled"] = "false"
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

// buildAgent creates a configured agent with service-based architecture.
func (m *Manager) buildAgent(executor *agent.Executor, ctxEnv *agent.Environment, logger *slog.Logger) (*agent.Agent, error) {
	// Build SecurityService
	validator := security.NewValidator()
	var approvalService *security.ApprovalService
	if m.approvalHandler != nil {
		approvalService = security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{
			Handler:   m.approvalHandler,
			Emitter:   m.emitter,
			Validator: validator,
		})
	} else {
		approvalService = security.NewApprovalService(nil, m.emitter, validator)
	}
	securityService := security.NewSecurityService(validator, approvalService)

	// Build DetectionService
	var cycleDetector detection.CycleDetector
	var patternDetector detection.PatternDetector
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

// validatorAdapter adapts security.Validator to tools.CommandValidator.
type validatorAdapter struct {
	validator *security.Validator
}

func (a *validatorAdapter) Classify(cmd tools.CommandInfo) (tools.ValidationResult, error) {
	// Convert CommandInfo to *security.Command
	secCmd := &security.Command{
		Program: cmd.GetProgram(),
		Args:    cmd.GetArgs(),
		Raw:     cmd.GetRaw(),
		WorkDir: cmd.GetWorkDir(),
	}
	return a.validator.Classify(secCmd)
}

// shellContextAdapter adapts shell.Context to tools.ShellContext.
type shellContextAdapter struct {
	shellCtx *shell.Context
}

func (a *shellContextAdapter) GetWorkingDirectory() string {
	return a.shellCtx.GetWorkingDirectory()
}

func (a *shellContextAdapter) GetEnvironmentVars() map[string]string {
	return a.shellCtx.GetEnvironmentVars()
}

func (a *shellContextAdapter) GetContextInfo() tools.ShellContextInfo {
	return a.shellCtx.GetContextInfo()
}

func (a *shellContextAdapter) IsShellCommand(command string) bool {
	return a.shellCtx.IsShellCommand(command)
}

// executorAdapter adapts agent.Executor to tools.CommandExecutor.
type executorAdapter struct {
	executor *agent.Executor
}

func (a *executorAdapter) Execute(ctx context.Context, cmd tools.CommandInfo, opts interface{}) (tools.ExecutionResult, error) {
	// Convert CommandInfo to *security.Command
	secCmd := &security.Command{
		Program: cmd.GetProgram(),
		Args:    cmd.GetArgs(),
		Raw:     cmd.GetRaw(),
		WorkDir: cmd.GetWorkDir(),
	}
	return a.executor.Execute(ctx, secCmd, nil)
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

	// Create adapters for shell command tool
	var validatorAdapt tools.CommandValidator
	if validator != nil {
		validatorAdapt = &validatorAdapter{validator: validator}
	}

	var shellCtxAdapt tools.ShellContext
	if m.shellIntegration != nil {
		shellCtxAdapt = &shellContextAdapter{shellCtx: m.shellIntegration}
	}

	var executorAdapt tools.CommandExecutor
	if executor != nil {
		executorAdapt = &executorAdapter{executor: executor}
	}

	// Register built-in tools
	_ = registry.Register(tools.NewReadFileTool())
	_ = registry.Register(tools.NewWriteFileTool())
	_ = registry.Register(tools.NewListDirectoryTool())
	_ = registry.Register(tools.NewShellCommandTool(validatorAdapt, shellCtxAdapt, executorAdapt))
	_ = registry.Register(tools.NewGetContextTool(ctxEnv))
	_ = registry.Register(tools.NewApplyPatchTool(ctxEnv.WorkDir))
	_ = registry.Register(tools.NewFileSearchTool(ctxEnv.WorkDir))
	_ = registry.Register(tools.NewGitContextTool(ctxEnv.WorkDir))

	// Register integration tools (MCP, Git)
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

// createHistory creates a conversation history.
func (m *Manager) createHistory() *history.History {
	hist := history.NewHistoryWithDefaults()

	// Set event emitter for notifications
	hist.SetEventEmitter(m.emitter)

	_ = hist.AddSystemMessage("You are a helpful AI coding assistant.")

	return hist
}

// NewManager creates a new Manager using the Builder pattern.
// This is a convenience function that wraps the Builder for backward compatibility.
//
// For more control over initialization, use NewBuilder() directly:
//
//	mgr, err := manager.NewBuilder(cfg).
//	    WithLLM(provider).
//	    WithApprovalHandler(handler).
//	    Build(ctx)
func NewManager(cfg *Config, opts ...ManagerOption) (*Manager, error) {
	// Create builder with config
	builder := NewBuilder(cfg)

	// Apply functional options via temporary manager
	// This preserves backward compatibility with existing code
	tempMgr := &Manager{cfg: cfg}
	for _, opt := range opts {
		if err := opt(tempMgr); err != nil {
			return nil, err
		}
	}

	// Transfer values from temp manager to builder
	if tempMgr.llm != nil {
		builder.WithLLM(tempMgr.llm)
	}
	if tempMgr.toolRegistry != nil {
		builder.WithToolRegistry(tempMgr.toolRegistry)
	}
	if tempMgr.approvalHandler != nil {
		builder.WithApprovalHandler(tempMgr.approvalHandler)
	}
	if tempMgr.emitter != nil {
		builder.WithEventEmitter(tempMgr.emitter)
	}
	if tempMgr.storage != nil {
		builder.WithStorage(tempMgr.storage)
	}
	if tempMgr.taskRegistry != nil {
		builder.WithTaskRegistry(tempMgr.taskRegistry)
	}
	if tempMgr.logger != nil {
		builder.WithLogger(tempMgr.logger)
	}

	// Build and return manager
	// Note: Manager initialization doesn't need context propagation since it's immediate setup
	// The context from NewConversation is used for actual operations
	return builder.Build(context.Background())
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

	// Optional: attach per-session JSONL event logger when debug enabled
	if m.cfg != nil && m.cfg.Debug {
		// Determine session directory
		sessBase := m.cfg.SessionDir
		if sessBase == "" {
			// Fallback default
			home, _ := os.UserHomeDir()
			sessBase = filepath.Join(home, ".spin", "sessions")
		} else if sessBase[:1] == "~" {
			if home, err := os.UserHomeDir(); err == nil {
				sessBase = filepath.Join(home, sessBase[1:])
			}
		}
		sessDir := filepath.Join(sessBase, sess.ID)
		_ = os.MkdirAll(sessDir, 0o755)
		logPath := filepath.Join(sessDir, "events.jsonl")

		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			// Subscribe to events and write JSON lines with session_id
			subID, ch, subErr := m.emitter.Subscribe()
			if subErr == nil {
				go func() {
					defer func() {
						m.emitter.Unsubscribe(subID)
						_ = f.Close()
					}()
					enc := json.NewEncoder(f)
					for {
						select {
						case ev, ok := <-ch:
							if !ok {
								return
							}
							// Wrap event with session metadata
							record := map[string]any{
								"session_id": sess.ID,
								"timestamp":  ev.Timestamp.Format(time.RFC3339Nano),
								"type":       ev.Type.String(),
								"data":       ev.Data,
							}
							// Best-effort write; ignore individual write errors
							_ = enc.Encode(record)
						case <-ctx.Done():
							return
						}
					}
				}()
			} else {
				_ = f.Close()
			}
		}
	}
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
