package conversation

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/dmytrogajewski/spin/internal/ace"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/caller"
	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/agent/harness/bridge"
	acemw "github.com/dmytrogajewski/spin/internal/agent/middleware/ace"
	"github.com/dmytrogajewski/spin/internal/agent/prompt"
	"github.com/dmytrogajewski/spin/internal/agent/scaffold"
	"github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/contexteng/adapter"
	"github.com/dmytrogajewski/spin/internal/contexteng/compactor"
	"github.com/dmytrogajewski/spin/internal/contexteng/observation"
	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/llm"
	mcppkg "github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/session"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Builder constructs a Conversation instance with all its dependencies.
// This pattern follows the service injection approach used in the tools package.
// Runtime is REQUIRED - it provides approval handler, tool registration, and other runtime-specific behavior.
type Builder struct {
	// Core configuration.
	cfg     *config.V2
	workDir string

	// Services (injected from application layer).
	gitService   *gitpkg.Service
	shellService *shellpkg.Service
	mcpService   *mcppkg.Service

	// Required.
	runtime executor.Runtime     // Runtime provides approval handler, tools, notifications.
	emitter *events.EventEmitter // Emitter MUST match runtime's emitter.

	// Optional overrides.
	llm          llm.Provider
	storage      session.Storage
	toolRegistry *tools.Registry // Optional pre-built tool registry.
	toolSelector *tool.Selector  // Dynamic tool selection - optional.
	logger       *slog.Logger

	// Managed resources.
	authManager   *auth.Manager
	memoryService *MemoryService
}

// NewBuilder creates a new Conversation builder with required dependencies.
// Runtime and emitter are REQUIRED - they must be created by the caller (cmd layer).
// The same emitter instance must be passed to both the runtime and the builder.
func NewBuilder(cfg *config.V2, workDir string, runtime executor.Runtime, emitter *events.EventEmitter, provider llm.Provider) *Builder {
	if cfg == nil {
		panic("config cannot be nil")
	}

	if workDir == "" {
		panic("workDir cannot be empty")
	}

	if runtime == nil {
		panic("runtime cannot be nil")
	}

	if emitter == nil {
		panic("emitter cannot be nil")
	}

	if provider == nil {
		panic("provider cannot be nil")
	}

	return &Builder{
		cfg:     cfg,
		workDir: workDir,
		runtime: runtime,
		emitter: emitter,
		llm:     provider,
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

// WithToolSelector sets the dynamic tool selector.
func (b *Builder) WithToolSelector(selector *tool.Selector) *Builder {
	b.toolSelector = selector

	return b
}

// WithAuthManager sets a pre-configured auth manager, preventing the builder
// from creating one internally. This is useful in tests to avoid side-effects
// from the default keystore (e.g., D-Bus goroutine leaks on Linux).
func (b *Builder) WithAuthManager(mgr *auth.Manager) *Builder {
	b.authManager = mgr

	return b
}

// Build constructs and returns a fully initialized Conversation.
func (b *Builder) Build(ctx context.Context) (*Conversation, error) {
	// Validate configuration.
	err := b.validate()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	logger := b.getLogger()
	logger.InfoContext(ctx, "building conversation", "work_dir", b.workDir)

	// Initialize core dependencies.
	err = b.initializeCoreDependencies()
	if err != nil {
		return nil, fmt.Errorf("initialize core dependencies: %w", err)
	}

	// Build executor using agent package helper with unified config
	// Runtime provides approval handler via executor.ApprovalHandler().
	exec := agent.NewBuilder().
		WithConfig(b.cfg).
		WithWorkingDir(b.workDir).
		WithEmitter(b.emitter).
		WithRuntime(b.runtime).
		BuildExecutor(ctx)

		// Gather environment using agent package helper with unified config.
	env := agent.NewBuilder().
		WithConfig(b.cfg).
		WithWorkingDir(b.workDir).
		BuildEnvironment(ctx)

		// Enrich environment with Git/Shell context (conversation-level concern).
	b.enrichEnvironmentWithIntegrations(ctx, env)

	// Create session early (ID is needed for memory initialization).
	sess := session.NewSession(b.workDir)
	logger.InfoContext(ctx, "session created", "session_id", sess.ID)

	// Initialize memory services if configured.
	err = b.initializeMemory(ctx, sess.ID)
	if err != nil {
		return nil, fmt.Errorf("initialize memory: %w", err)
	}

	// Build agent (orchestration handled by conversation).
	result, err := b.buildAgent(ctx, exec, env)
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	// Build harness executor for the conversation execution path.
	harnessExec, err := b.buildHarnessExecutor(
		result.toolReg, result.toolRuntime, result.aceService, result.aceConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("build harness executor: %w", err)
	}

	// Create history.
	hist := b.createHistory(ctx)

	// Use session ID as conversation ID (both are UUID strings)
	// This maintains clean dependency direction: conversation doesn't depend on protocol.
	convID := sess.ID

	// Build the Conversation with unified ID.
	conv := &Conversation{
		gitService:      b.gitService,
		shellService:    b.shellService,
		mcpService:      b.mcpService,
		memoryService:   b.memoryService,
		agent:           result.agent,
		history:         hist,
		emitter:         b.emitter,
		taskMode:        "regular",
		id:              convID,
		workDir:         b.workDir,
		harnessExecutor: harnessExec,
	}

	// Attach JSONL event logger if debug mode.
	if b.cfg != nil && b.cfg.Agent.Debug {
		b.attachJSONLEventLogger(ctx, convID)
	}

	logger.InfoContext(ctx, "conversation built successfully", "session_id", convID)

	return conv, nil
}

// validate ensures the configuration is valid.
// Required fields are already validated in NewBuilder constructor.
func (b *Builder) validate() error {
	err := b.cfg.Validate()
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
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

// initializeCoreDependencies sets up optional dependencies (storage, auth).
// Required dependencies (runtime, emitter, provider) are passed to constructor.
func (b *Builder) initializeCoreDependencies() error {
	// Session storage (optional - can use default).
	if b.storage == nil {
		// Use default session directory if not configured.
		sessionDir := b.cfg.Agent.SessionDir
		if sessionDir == "" {
			sessionDir = "~/.spin/sessions"
		}

		fs, err := session.NewFileStorage(sessionDir)
		if err != nil {
			return fmt.Errorf("initialize storage: %w", err)
		}

		b.storage = fs
	}

	// Auth manager (internal resource) - only create if not already set.
	if b.authManager == nil {
		keystore := auth.NewKeystore()
		b.authManager = auth.NewManager(keystore)
	}

	return nil
}

// enrichEnvironmentWithIntegrations adds context from Git and Shell integrations.
func (b *Builder) enrichEnvironmentWithIntegrations(ctx context.Context, env *agent.Environment) {
	if b.gitService != nil && b.gitService.IsRepository() {
		b.addGitContext(ctx, env)
	}

	if b.shellService != nil && b.shellService.IsEnabled() {
		b.addShellContext(ctx, env)
	}
}

// addGitContext enriches environment with Git repository information.
func (b *Builder) addGitContext(ctx context.Context, env *agent.Environment) {
	info := b.gitService.GetContextInfo()
	set := func(k, v string) { env.Environment[k] = v }

	set("git_enabled", strconv.FormatBool(info.GitEnabled))
	set("is_repo", strconv.FormatBool(info.IsRepo))

	if !info.IsRepo {
		if b.logger != nil {
			b.logger.DebugContext(ctx, "git context: not a repository")
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

	set("is_clean", strconv.FormatBool(info.IsClean))

	if info.ModifiedFiles > 0 {
		set("modified_files", strconv.Itoa(info.ModifiedFiles))
	}

	if info.UntrackedFiles > 0 {
		set("untracked_files", strconv.Itoa(info.UntrackedFiles))
	}

	if info.Ahead > 0 {
		set("ahead", strconv.Itoa(info.Ahead))
	}

	if info.Behind > 0 {
		set("behind", strconv.Itoa(info.Behind))
	}

	if info.Detached {
		set("detached", "true")
	}

	if b.logger != nil {
		b.logger.DebugContext(ctx, "git context added", "branch", info.Branch, "clean", info.IsClean)
	}
}

// buildHarnessExecutor constructs the experimental harness executor from shared components.
func (b *Builder) buildHarnessExecutor(
	toolReg *tools.Registry,
	toolRuntime *tool.Runtime,
	aceSvc *ace.Service,
	aceConfig *ace.Config,
) (*bridge.TurnExecutor, error) {
	logger := b.getLogger()

	// Create a dedicated LLMCaller for the harness bridge with router support.
	pb := prompt.New(b.llm, logger)
	router := llm.NewSingleProviderRouter(b.llm)

	llmCaller := caller.New(caller.Config{
		Router:        router,
		Role:          llm.RoleAction,
		PromptBuilder: pb,
		Emitter:       b.emitter,
		Logger:        logger,
		Temperature:   b.cfg.LLM.Temperature,
		MaxTokens:     b.cfg.LLM.MaxTokens,
	})

	// Build a minimal scaffold.Spec from the tool registry.
	spec := &scaffold.Spec{
		ToolSchemas: toolReg.ListSchemas(),
		Config: scaffold.SpecConfig{
			MaxTurns: b.cfg.Agent.MaxTurns,
		},
	}

	// Build middleware chain.
	middlewares := b.buildHarnessMiddlewares(aceSvc, aceConfig, logger)

	harnessOpts := b.buildHarnessOpts()

	harnessExec, err := bridge.BuildExecutor(bridge.Config{
		Spec:        spec,
		LLMCaller:   llmCaller,
		Registry:    toolReg,
		Runtime:     toolRuntime,
		Logger:      logger,
		Middlewares: middlewares,
		HarnessOpts: harnessOpts,
	})
	if err != nil {
		return nil, fmt.Errorf("bridge build: %w", err)
	}

	return bridge.NewTurnExecutor(harnessExec), nil
}

// buildHarnessMiddlewares constructs the middleware chain for the harness executor.
func (b *Builder) buildHarnessMiddlewares(
	aceSvc *ace.Service,
	aceConfig *ace.Config,
	logger *slog.Logger,
) []harness.Middleware {
	var middlewares []harness.Middleware

	if aceSvc != nil && aceConfig != nil {
		inner := acemw.New(aceSvc, aceConfig, b.emitter, logger)
		middlewares = append(middlewares, acemw.NewHarnessAdapter(inner))
	}

	return middlewares
}

// buildHarnessOpts creates harness options for contexteng adapters.
func (b *Builder) buildHarnessOpts() []harness.Option {
	var opts []harness.Option

	// Wire compactor adapter with LLM context window from config.
	if b.cfg.LLM.ContextWindow > 0 {
		tok := &tokenizer.SimpleTokenizer{}
		comp := compactor.NewCompactor(tok, b.cfg.LLM.ContextWindow)
		opts = append(opts, harness.WithCompactor(
			adapter.NewCompactorAdapter(comp),
		))
	}

	// Wire observation summarizer (always available, no config dependency).
	obs := observation.NewSummarizer()
	opts = append(opts, harness.WithObservationSummarizer(
		adapter.NewObservationAdapter(obs),
	))

	return opts
}

// addShellContext enriches environment with Shell context information.
func (b *Builder) addShellContext(ctx context.Context, env *agent.Environment) {
	info := b.shellService.GetContextInfo()
	set := func(k, v string) { env.Environment[k] = v }

	set("shell_enabled", strconv.FormatBool(info.ShellEnabled))

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
		b.logger.DebugContext(ctx, "shell context added", "shell", info.Shell)
	}
}
