package conversation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/llm"
	mcppkg "github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Builder constructs a Conversation instance with all its dependencies.
// This pattern follows the service injection approach used in the tools package.
type Builder struct {
	// Core configuration
	cfg     *config.Config
	workDir string

	// Services (injected from application layer)
	gitService   *gitpkg.Service
	shellService *shellpkg.Service
	mcpService   *mcppkg.Service

	// Optional overrides
	llm             llm.Provider
	emitter         *events.EventEmitter
	storage         session.Storage
	toolRegistry    *tools.Registry
	taskRegistry    *orchestration.Registry
	approvalHandler security.ApprovalHandler
	logger          *slog.Logger

	// Managed resources
	authManager *auth.Manager
}

// NewBuilder creates a new Conversation builder with the given configuration and working directory.
func NewBuilder(cfg *config.Config, workDir string) *Builder {
	return &Builder{
		cfg:     cfg,
		workDir: workDir,
	}
}

// WithGit sets the Git service.
func (b *Builder) WithGit(service *gitpkg.Service) *Builder {
	b.gitService = service
	return b
}

// WithShell sets the Shell service.
func (b *Builder) WithShell(service *shellpkg.Service) *Builder {
	b.shellService = service
	return b
}

// WithMCP sets the MCP service.
func (b *Builder) WithMCP(service *mcppkg.Service) *Builder {
	b.mcpService = service
	return b
}

// WithLLM sets a custom LLM provider.
func (b *Builder) WithLLM(provider llm.Provider) *Builder {
	b.llm = provider
	return b
}

// WithToolRegistry sets a custom tool registry.
func (b *Builder) WithToolRegistry(registry *tools.Registry) *Builder {
	b.toolRegistry = registry
	return b
}

// WithApprovalHandler sets a custom approval handler.
func (b *Builder) WithApprovalHandler(handler security.ApprovalHandler) *Builder {
	b.approvalHandler = handler
	return b
}

// Build constructs and returns a fully initialized Conversation.
func (b *Builder) Build(ctx context.Context) (*Conversation, error) {
	// Validate configuration
	if err := b.validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	logger := b.getLogger()
	logger.Info("building conversation", "work_dir", b.workDir)

	// Initialize core dependencies
	if err := b.initializeCoreDependencies(); err != nil {
		return nil, fmt.Errorf("initialize core dependencies: %w", err)
	}

	// Build executor using agent package helper
	exec := agent.NewBuilder().
		WithConfig(b.convertToAgentConfig()).
		WithWorkingDir(b.workDir).
		WithEmitter(b.emitter).
		WithApprovalHandler(b.approvalHandler).
		BuildExecutor()

	// Gather environment using agent package helper
	env := agent.NewBuilder().
		WithConfig(b.convertToAgentConfig()).
		WithWorkingDir(b.workDir).
		BuildEnvironment()

	// Enrich environment with Git/Shell context (conversation-level concern)
	b.enrichEnvironmentWithIntegrations(env)

	// Build agent (orchestration handled by conversation)
	agentInstance, err := b.buildAgent(exec, env)
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	// Create session
	sess := session.NewSession(b.workDir)
	logger.Info("session created", "session_id", sess.ID)

	// Create history
	hist := b.createHistory()

	// Build the Conversation
	conv := &Conversation{
		gitService:   b.gitService,
		shellService: b.shellService,
		mcpService:   b.mcpService,
		agent:        agentInstance,
		history:      hist,
		emitter:      b.emitter,
		taskMode:     "regular",
		sessionID:    sess.ID,
		workDir:      b.workDir,
	}

	// Attach JSONL event logger if debug mode
	if b.cfg != nil && b.cfg.Debug {
		b.attachJSONLEventLogger(ctx, sess.ID)
	}

	logger.Info("conversation built successfully", "session_id", sess.ID)
	return conv, nil
}

// validate ensures the configuration is valid and required fields are set.
func (b *Builder) validate() error {
	if b.cfg == nil {
		return errors.New("config cannot be nil")
	}
	if err := b.cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if b.workDir == "" {
		return errors.New("workDir cannot be empty")
	}
	return nil
}

// getLogger returns the configured logger or creates a default one.
func (b *Builder) getLogger() *slog.Logger {
	if b.logger != nil {
		return b.logger
	}
	return slog.Default()
}

// initializeCoreDependencies sets up LLM, event emitter, storage, and auth.
func (b *Builder) initializeCoreDependencies() error {
	// LLM provider
	if b.llm == nil {
		b.llm = llm.NewMockProvider("default")
	}

	// Event emitter
	if b.emitter == nil {
		bufferSize := 100
		if b.cfg.StreamBuffer > 0 {
			bufferSize = b.cfg.StreamBuffer
		}
		b.emitter = events.NewEventEmitter(bufferSize)
	}

	// Session storage
	if b.storage == nil {
		fs, err := session.NewFileStorage(b.cfg.SessionDir)
		if err != nil {
			return fmt.Errorf("initialize storage: %w", err)
		}
		b.storage = fs
	}

	// Auth manager
	keystore := auth.NewKeystore()
	b.authManager = auth.NewManager(keystore)

	return nil
}

// convertToAgentConfig converts config.Config to agent.Config
func (b *Builder) convertToAgentConfig() *agent.Config {
	if b.cfg == nil {
		return agent.DefaultConfig()
	}

	return &agent.Config{
		Provider:        b.cfg.Provider,
		Model:           b.cfg.Model,
		ProviderConfig:  b.cfg.ProviderConfig,
		Temperature:     b.cfg.Temperature,
		MaxTurns:        b.cfg.MaxTurns,
		Timeout:         b.cfg.Timeout,
		WorkDir:         b.cfg.WorkDir,
		MaxTokens:       b.cfg.MaxTokens,
		RequireApproval: b.cfg.RequireApproval,
		SandboxMode:     b.cfg.SandboxMode,
		PolicyFile:      b.cfg.PolicyFile,
		AllowedCommands: b.cfg.AllowedCommands,
		EnableMCP:       b.cfg.EnableMCP,
		MCPServers:      convertMCPServers(b.cfg.MCPServers),
		EnableGit:       b.cfg.EnableGit,
		EnableShell:     b.cfg.EnableShell,
		StreamBuffer:    b.cfg.StreamBuffer,
		CacheCommands:   b.cfg.CacheCommands,
		MaxFiles:        b.cfg.MaxFiles,
		MaxDepth:        b.cfg.MaxDepth,
		SkipGit:         b.cfg.SkipGit,
		SessionDir:      b.cfg.SessionDir,
		HistoryLimit:    b.cfg.HistoryLimit,
		LogLevel:        b.cfg.LogLevel,
		LogFormat:       b.cfg.LogFormat,
		Debug:           b.cfg.Debug,
		CycleDetection:  convertCycleDetection(b.cfg.CycleDetection),
		ACE:             convertACEConfigFromFlat(b.cfg),
	}
}

func convertMCPServers(servers []config.MCPServerConfig) []agent.MCPServerConfig {
	result := make([]agent.MCPServerConfig, len(servers))
	for i, s := range servers {
		result[i] = agent.MCPServerConfig{
			Command: s.Command,
			Args:    s.Args,
			Env:     s.Env,
		}
	}
	return result
}

func convertCycleDetection(cfg config.CycleDetectionConfig) agent.CycleDetectionConfig {
	return agent.CycleDetectionConfig{
		Enabled:          cfg.Enabled,
		WindowSize:       cfg.WindowSize,
		SimilarityThresh: cfg.SimilarityThresh,
		ToolRepeatLimit:  cfg.ToolRepeatLimit,
		ErrorRepeatLimit: cfg.ErrorRepeatLimit,
	}
}

func convertACEConfigFromFlat(cfg *config.Config) agent.ACEConfig {
	// config.Config has flat ACE fields, convert to nested agent.ACEConfig
	defaultACE := agent.DefaultConfig().ACE

	return agent.ACEConfig{
		Enabled:        cfg.ACEEnabled,
		PlaybookPath:   cfg.ACEPlaybookPath,
		TrajectoryPath: cfg.ACETrajectoryPath,
		Retrieval: agent.ACERetrievalConfig{
			TopK:     cfg.ACETopK,
			MinScore: cfg.ACEMinScore,
			ProgressiveContext: agent.ProgressiveContextConfig{
				Enabled:    true,
				CacheTTL:   10,
				MaxBullets: 50,
			},
		},
		ItemizedLearning: agent.ACEItemizedLearningConfig{
			Enabled:       true,
			ParseFeedback: true,
			UpdateAsync:   false,
		},
		Generation: agent.ACEGenerationConfig{
			Enabled:     true,
			AutoReflect: true,
		},
		Adapter: defaultACE.Adapter,
		Refine:  defaultACE.Refine,
	}
}

// enrichEnvironmentWithIntegrations adds context from Git and Shell integrations.
func (b *Builder) enrichEnvironmentWithIntegrations(env *agent.Environment) {
	if b.gitService != nil && b.gitService.IsRepository() {
		b.addGitContext(env)
	}
	if b.shellService != nil && b.shellService.IsEnabled() {
		b.addShellContext(env)
	}
}

// addGitContext enriches environment with Git repository information.
func (b *Builder) addGitContext(env *agent.Environment) {
	info := b.gitService.GetContextInfo()
	set := func(k, v string) { env.Environment[k] = v }

	set("git_enabled", boolString(info.GitEnabled))
	set("is_repo", boolString(info.IsRepo))
	if !info.IsRepo {
		if b.logger != nil {
			b.logger.Debug("git context: not a repository")
		}
		return
	}

	if info.Branch != "" {
		set("branch", info.Branch)
	}
	if info.Remote != "" {
		set("remote", info.Remote)
	}
	if info.Commit != "" {
		set("commit", info.Commit)
	}
	set("is_clean", boolString(info.IsClean))

	if info.ModifiedFiles > 0 {
		set("modified_files", fmt.Sprintf("%d", info.ModifiedFiles))
	}
	if info.UntrackedFiles > 0 {
		set("untracked_files", fmt.Sprintf("%d", info.UntrackedFiles))
	}
	if info.Ahead > 0 {
		set("ahead", fmt.Sprintf("%d", info.Ahead))
	}
	if info.Behind > 0 {
		set("behind", fmt.Sprintf("%d", info.Behind))
	}
	if info.Detached {
		set("detached", "true")
	}

	if b.logger != nil {
		b.logger.Debug("git context added", "branch", info.Branch, "clean", info.IsClean)
	}
}

// addShellContext enriches environment with Shell context information.
func (b *Builder) addShellContext(env *agent.Environment) {
	info := b.shellService.GetContextInfo()
	set := func(k, v string) { env.Environment[k] = v }

	set("shell_enabled", boolString(info.ShellEnabled))
	if !info.ShellEnabled {
		return
	}
	if info.Shell != "" {
		set("shell", info.Shell)
	}
	if info.ShellPath != "" {
		set("shell_path", info.ShellPath)
	}
	if b.logger != nil {
		b.logger.Debug("shell context added", "shell", info.Shell)
	}
}

// boolString converts a boolean to "true" or "false" string.
func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
