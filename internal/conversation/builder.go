package conversation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/caller"
	"github.com/dmytrogajewski/spin/internal/agent/child"
	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/agent/frame"
	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/agent/harness/bridge"
	acemw "github.com/dmytrogajewski/spin/internal/agent/middleware/ace"
	snapshotmw "github.com/dmytrogajewski/spin/internal/agent/middleware/snapshot"
	"github.com/dmytrogajewski/spin/internal/agent/prompt"
	"github.com/dmytrogajewski/spin/internal/agent/scaffold"
	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/agent/tasks"
	"github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/contexteng/adapter"
	"github.com/dmytrogajewski/spin/internal/contexteng/compactor"
	"github.com/dmytrogajewski/spin/internal/contexteng/observation"
	"github.com/dmytrogajewski/spin/internal/contexteng/reminder"
	"github.com/dmytrogajewski/spin/internal/contexteng/retrieval"
	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/llm"
	llmcache "github.com/dmytrogajewski/spin/internal/llm/cache"
	"github.com/dmytrogajewski/spin/internal/lsp"
	mcppkg "github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/plugins"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
	"github.com/dmytrogajewski/spin/internal/session"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/skills"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/undo"
	"github.com/dmytrogajewski/spin/pkg/tokenizer"
)

// ErrSubagentSpawnNotSupported indicates subagent spawning requires per-subagent harness setup.
var ErrSubagentSpawnNotSupported = errors.New("subagent spawn requires per-subagent harness setup")

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
	lspManager   *lsp.Manager

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
	logger := b.getLogger()
	logger.InfoContext(ctx, "building conversation", "work_dir", b.workDir)

	// Phase 1: validate and initialize core dependencies.
	if err := b.initBuildPrerequisites(ctx); err != nil {
		return nil, err
	}

	// Phase 2: build executor and environment.
	exec, env := b.buildExecutorAndEnvironment(ctx)

	// Phase 3: session, memory, agent, harness.
	sess := session.NewSession(b.workDir)
	logger.InfoContext(ctx, "session created", "session_id", sess.ID)

	if err := b.initializeMemory(ctx, sess.ID); err != nil {
		return nil, fmt.Errorf("initialize memory: %w", err)
	}

	result, err := b.buildAgent(ctx, exec, env)
	if err != nil {
		return nil, fmt.Errorf("build agent: %w", err)
	}

	harnessExec, err := b.buildHarnessExecutor(
		ctx, result.toolReg, result.toolRuntime, result.aceService, result.aceConfig, env, result.hookRunner,
	)
	if err != nil {
		return nil, fmt.Errorf("build harness executor: %w", err)
	}

	// Phase 4: assemble conversation.
	conv := b.assembleConversation(ctx, sess, result, harnessExec, logger)

	logger.InfoContext(ctx, "conversation built successfully", "session_id", conv.id)

	return conv, nil
}

// initBuildPrerequisites validates config and initializes core dependencies.
func (b *Builder) initBuildPrerequisites(ctx context.Context) error {
	if err := b.validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := b.initializeCoreDependencies(); err != nil {
		return fmt.Errorf("initialize core dependencies: %w", err)
	}

	b.lspManager = lsp.NewManager("file://"+b.workDir, lsp.DefaultServerFactory)
	b.initProviderCache(ctx, b.getLogger())
	b.attachPluginMCP(ctx)

	return nil
}

func (b *Builder) pluginPaths() []string {
	if b.cfg == nil {
		return nil
	}

	return b.cfg.Plugins.Paths
}

func (b *Builder) skillCatalog() skills.Catalog {
	return plugins.DiscoverCatalog(b.workDir, b.pluginPaths())
}

func (b *Builder) attachPluginMCP(ctx context.Context) {
	if b.mcpService == nil {
		return
	}

	_ = plugins.AttachMCP(ctx, b.mcpService, b.discoverPlugins().Plugins)
}

func (b *Builder) pluginHookScripts() []hooks.PluginScript {
	return plugins.HookScripts(b.discoverPlugins().Plugins)
}

func (b *Builder) hooksGlobalDir() string {
	return hooks.DefaultGlobalDir()
}

func (b *Builder) discoverPlugins() plugins.Result {
	opts := skills.OptionsFor(b.workDir)

	return plugins.Discover(plugins.DiscoverOptions{
		WorkDir:    b.workDir,
		HomeDir:    opts.HomeDir,
		ExtraPaths: b.pluginPaths(),
	})
}

// buildExecutorAndEnvironment creates the executor and gathers environment info.
func (b *Builder) buildExecutorAndEnvironment(ctx context.Context) (*agent.Executor, *agent.Environment) {
	exec := agent.NewBuilder().
		WithConfig(b.cfg).
		WithWorkingDir(b.workDir).
		WithEmitter(b.emitter).
		WithRuntime(b.runtime).
		BuildExecutor(ctx)

	env := agent.NewBuilder().
		WithConfig(b.cfg).
		WithWorkingDir(b.workDir).
		BuildEnvironment(ctx)

	b.enrichEnvironmentWithIntegrations(ctx, env)

	return exec, env
}

// assembleConversation creates the final Conversation struct with all components.
func (b *Builder) assembleConversation(
	ctx context.Context,
	sess *session.Session,
	result *agentBuildResult,
	harnessExec *bridge.TurnExecutor,
	logger *slog.Logger,
) *Conversation {
	convID := sess.ID
	sessionIdx := b.initSessionIndex(ctx, convID, sess, logger)
	transcriptWriter := b.initTranscriptWriter(ctx, convID, logger)
	taskReg := tasks.Restore(sess)
	mgr := b.createSubagentManager(result.hookRunner)
	tools.RegisterAgentTaskTools(result.toolReg, taskReg)
	b.registerNavigate(result.toolReg, sessionIdx)

	b.fireSessionStartHook(ctx, result.hookRunner, convID)

	conv := &Conversation{
		gitService:        b.gitService,
		shellService:      b.shellService,
		mcpService:        b.mcpService,
		memoryService:     b.memoryService,
		agent:             result.agent,
		history:           b.createHistory(ctx),
		emitter:           b.emitter,
		taskMode:          "regular",
		id:                convID,
		workDir:           b.workDir,
		sessionDir:        b.resolveSessionDir(),
		harnessExecutor:   harnessExec,
		subagentManager:   mgr,
		taskRegistry:      taskReg,
		retrievalPipeline: retrieval.NewPipeline(retrieval.NewBulletSource()),
		lspManager:        b.lspManager,
		hookRunner:        result.hookRunner,
		transcriptWriter:  transcriptWriter,
		sessionIndex:      sessionIdx,
	}

	if tm, ok := b.runtime.(shellTaskProvider); ok {
		conv.shellTasks = tools.AsShellSource(tm.TaskManager())
	}

	if b.cfg != nil && b.cfg.Agent.Debug {
		b.attachJSONLEventLogger(ctx, convID)
	}

	return conv
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

// initSessionIndex creates a session index and updates it with the new session (best-effort).
func (b *Builder) initSessionIndex(ctx context.Context, convID string, sess *session.Session, logger *slog.Logger) *session.Index {
	sessionIndexPath := filepath.Join(b.resolveSessionDir(), "index.json")

	sessionIdx, idxErr := session.NewSessionIndex(ctx, sessionIndexPath, nil,
		session.WithRebuildCallback(func() {
			logger.InfoContext(ctx, "session index rebuilt from scratch")
		}),
	)
	if idxErr != nil {
		logger.DebugContext(ctx, "session index creation failed, continuing without index", "error", idxErr)
	}

	if sessionIdx != nil {
		_ = sessionIdx.Update(ctx, session.IndexEntry{
			ID:           convID,
			WorkDir:      b.workDir,
			LastModified: sess.CreatedAt,
		})
	}

	return sessionIdx
}

// initTranscriptWriter creates a transcript writer for JSONL persistence (best-effort).
func (b *Builder) initTranscriptWriter(ctx context.Context, convID string, logger *slog.Logger) *session.TranscriptWriter {
	transcriptPath := session.TranscriptPath(b.resolveSessionDir(), convID)

	if mkdirErr := os.MkdirAll(filepath.Dir(transcriptPath), 0o750); mkdirErr != nil {
		return nil
	}

	tw, twErr := session.NewTranscriptWriter(transcriptPath)
	if twErr != nil {
		logger.DebugContext(ctx, "transcript writer creation failed, continuing without persistence", "error", twErr)

		return nil
	}

	return tw
}

// fireSessionStartHook fires the SESSION_START hook (non-blocking, best-effort).
func (b *Builder) fireSessionStartHook(ctx context.Context, hookRunner *hooks.Runner, convID string) {
	if hookRunner == nil {
		return
	}

	evtCtx := hooks.EventContext{
		SessionID: convID,
		WorkDir:   b.workDir,
	}
	hookRunner.Execute(ctx, hooks.EventSessionStart, evtCtx)
}

// createSubagentManager creates a subagent manager with the process executor.
func (b *Builder) createSubagentManager(runner *hooks.Runner) *subagent.Manager {
	mgr := subagent.NewManager(b.processExecutor(runner), subagent.DefaultMaxConcurrent)
	mgr.SetBackgroundStarter(child.ImmediateStarter(child.ResolveBinary(), b.workDir, runner))
	b.registerConfigSubagents(mgr)

	return mgr
}

func (b *Builder) processExecutor(runner *hooks.Runner) subagent.Executor {
	return child.NewExecutor(child.ResolveBinary(), b.workDir, b.emitter, runner)
}

func (b *Builder) registerConfigSubagents(mgr *subagent.Manager) {
	if b.cfg == nil {
		return
	}

	for name, sa := range b.cfg.Subagents {
		_ = mgr.Register(overlaySpec(mgr.Spec(name), name, sa))
	}
}

func overlaySpec(existing *subagent.Spec, name string, sa config.SubagentConfigV2) *subagent.Spec {
	spec := &subagent.Spec{Name: name, Description: name}

	if existing != nil {
		copied := *existing
		spec = &copied
	}

	spec.ModelOverride = sa.Model
	spec.MaxIterations = sa.EffectiveMaxIterations()

	return spec
}

// getLogger returns the configured logger or creates a default one.
func (b *Builder) getLogger() *slog.Logger {
	if b.logger != nil {
		return b.logger
	}

	return slog.Default()
}

type shellTaskProvider interface {
	TaskManager() *executor.TaskManagerAdapter
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

// enrichEnvironment sets non-empty values from fields into the environment map.
func enrichEnvironment(env *agent.Environment, fields map[string]string) {
	for k, v := range fields {
		if v != "" {
			env.Environment[k] = v
		}
	}
}

// addGitContext enriches environment with Git repository information.
func (b *Builder) addGitContext(ctx context.Context, env *agent.Environment) {
	info := b.gitService.GetContextInfo()

	enrichEnvironment(env, map[string]string{
		"git_enabled": strconv.FormatBool(info.GitEnabled),
		"is_repo":     strconv.FormatBool(info.IsRepo),
	})

	if !info.IsRepo {
		if b.logger != nil {
			b.logger.DebugContext(ctx, "git context: not a repository")
		}

		return
	}

	enrichEnvironment(env, map[string]string{
		"branch":          info.Branch,
		"remote":          info.Remote,
		"commit":          info.Commit,
		"is_clean":        strconv.FormatBool(info.IsClean),
		"modified_files":  nonZeroItoa(info.ModifiedFiles),
		"untracked_files": nonZeroItoa(info.UntrackedFiles),
		"ahead":           nonZeroItoa(info.Ahead),
		"behind":          nonZeroItoa(info.Behind),
	})

	if info.Detached {
		env.Environment["detached"] = "true"
	}

	if b.logger != nil {
		b.logger.DebugContext(ctx, "git context added", "branch", info.Branch, "clean", info.IsClean)
	}
}

// nonZeroItoa returns the string representation of n, or "" if n is zero.
func nonZeroItoa(n int) string {
	if n == 0 {
		return ""
	}

	return strconv.Itoa(n)
}

// buildHarnessExecutor constructs the experimental harness executor from shared components.
func (b *Builder) buildHarnessExecutor(
	ctx context.Context,
	toolReg *tools.Registry,
	toolRuntime *tool.Runtime,
	aceSvc *ace.Service,
	aceConfig *ace.Config,
	env *agent.Environment,
	hookRunner *hooks.Runner,
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

	systemPrompt := b.composeSystemPrompt(env)

	// Compile scaffold.Spec via Factory (replaces manual construction).
	factory, factoryErr := scaffold.NewFactory(b.cfg, toolReg, nil)
	if factoryErr != nil {
		return nil, fmt.Errorf("scaffold factory: %w", factoryErr)
	}

	spec, compileErr := factory.Compile(scaffold.AgentTypeMain)
	if compileErr != nil {
		return nil, fmt.Errorf("compile main spec: %w", compileErr)
	}

	// Override Factory's default system prompt with Composer output (richer).
	spec.SystemPrompt = systemPrompt

	// Create undo service for snapshot middleware.
	undoSvc := b.createUndoService(ctx, logger)

	// Build guards, middleware chain, and harness options.
	guards := b.buildHarnessGuards()
	middlewares := b.buildHarnessMiddlewares(aceSvc, aceConfig, undoSvc, logger)

	harnessOpts := b.buildHarnessOpts()
	if hookRunner != nil {
		harnessOpts = append(harnessOpts, harness.WithHookRunner(hookRunner))
	}

	harnessExec, err := bridge.BuildExecutor(bridge.Config{
		Spec:        spec,
		LLMCaller:   llmCaller,
		Registry:    toolReg,
		Runtime:     toolRuntime,
		Logger:      logger,
		Guards:      guards,
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
	undoSvc *undo.Service,
	logger *slog.Logger,
) []harness.Middleware {
	var middlewares []harness.Middleware

	if aceSvc != nil && aceConfig != nil {
		inner := acemw.New(aceSvc, aceConfig, b.emitter, logger)
		middlewares = append(middlewares, acemw.NewHarnessAdapter(inner))
	}

	// Snapshot middleware captures working-tree state after each execution phase.
	if undoSvc != nil {
		middlewares = append(middlewares, snapshotmw.NewMiddleware(undoSvc, logger))
	}

	return middlewares
}

// createUndoService creates the undo service with snapshot support.
// Returns nil if snapshot initialization fails (best-effort).
func (b *Builder) createUndoService(ctx context.Context, logger *slog.Logger) *undo.Service {
	opLog := undo.NewOperationLog()
	snapshotMgr := undo.NewSnapshotManager(b.workDir)

	if err := snapshotMgr.Init(ctx); err != nil {
		logger.DebugContext(ctx, "snapshot manager init failed, undo service without snapshots",
			"error", err)

		return undo.NewService(opLog, nil)
	}

	return undo.NewService(opLog, snapshotMgr)
}

// buildHarnessGuards creates the guard chain for the harness executor.
func (b *Builder) buildHarnessGuards() []harness.Guard {
	guards := make([]harness.Guard, 0, 1)

	// Doom-loop detection: halt when the same tool call repeats excessively.
	doomGuard := harness.NewDoomLoopGuard(harness.DefaultWindowSize, harness.DefaultThreshold)
	doomGuard.SetEmitter(b.emitter)
	guards = append(guards, doomGuard)

	return guards
}

// maxHarnessOpts is the expected maximum number of harness options.
const maxHarnessOpts = 5

// buildHarnessOpts creates harness options for contexteng adapters.
func (b *Builder) buildHarnessOpts() []harness.Option {
	opts := make([]harness.Option, 0, maxHarnessOpts)

	// Wire compactor adapter with LLM context window from config.
	if b.cfg.LLM.ContextWindow > 0 {
		tok := &tokenizer.SimpleTokenizer{}

		var compactorOpts []compactor.Option
		if b.cfg.LLM.CompactorWarning > 0 || b.cfg.LLM.CompactorObserve > 0 || b.cfg.LLM.CompactorPrune > 0 {
			compactorOpts = append(compactorOpts, compactor.WithThresholds(
				b.cfg.LLM.CompactorWarning,
				b.cfg.LLM.CompactorObserve,
				b.cfg.LLM.CompactorPrune,
			))
		}

		if b.cfg.LLM.CompactorRecentProtected > 0 {
			compactorOpts = append(compactorOpts, compactor.WithRecentProtected(b.cfg.LLM.CompactorRecentProtected))
		}

		comp := compactor.NewCompactor(tok, b.cfg.LLM.ContextWindow, compactorOpts...)
		opts = append(opts, harness.WithCompactor(
			adapter.NewCompactorAdapter(comp),
		))
	}

	// Wire observation summarizer (always available, no config dependency).
	obs := observation.NewSummarizer()
	// Wire reminder injector with all default detectors.
	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	// Wire event emitter so harness phases can emit structured events.
	opts = append(opts,
		harness.WithObservationSummarizer(adapter.NewObservationAdapter(obs)),
		harness.WithReminderInjector(adapter.NewReminderAdapter(inj)),
		harness.WithEmitter(b.emitter),
	)

	return opts
}

// initProviderCache creates a provider cache and loads/caches model capabilities.
// If the config doesn't set a context window, the cache is used to populate it.
func (b *Builder) initProviderCache(ctx context.Context, logger *slog.Logger) {
	cacheDir := filepath.Join(b.cfg.Agent.SessionDir, "..", "cache")

	provCache, cacheErr := llmcache.NewProviderCache(cacheDir, nil, llmcache.WithTimeFunc(time.Now))
	if cacheErr != nil {
		logger.DebugContext(ctx, "provider cache init failed, continuing without cache", "error", cacheErr)

		return
	}

	providerName := b.cfg.LLM.Provider
	modelName := b.cfg.LLM.Model

	// Load cached data for the current provider.
	if loadErr := provCache.Load(providerName); loadErr != nil {
		logger.DebugContext(ctx, "provider cache load failed", "provider", providerName, "error", loadErr)
	}

	// Use cached context length if config doesn't override.
	if b.cfg.LLM.ContextWindow == 0 {
		ctxLen := provCache.ContextLength(ctx, providerName, modelName)
		if ctxLen > 0 {
			b.cfg.LLM.ContextWindow = ctxLen
			logger.InfoContext(ctx, "context window set from provider cache", "context_window", ctxLen)
		}
	}

	// Cache current provider capabilities from LLM if available.
	if b.llm != nil {
		caps := b.llm.Capabilities()
		_ = provCache.Put(ctx, providerName, modelName, llmcache.ModelCapabilities{
			ModelID:         modelName,
			Provider:        providerName,
			ContextLength:   caps.ContextWindow,
			Streaming:       caps.Streaming,
			FunctionCalling: caps.FunctionCalling,
		})
	}
}

// composeSystemPrompt builds the Composer output used by TUI and exec.
func (b *Builder) composeSystemPrompt(env *agent.Environment) string {
	composer := prompt.NewComposer()
	for _, section := range prompt.DefaultRegularSections() {
		composer.AddSection(section)
	}

	if agentsMD := b.resolveAgentsMDContent(); agentsMD != "" {
		composer.AddSection(prompt.ProjectInstructionsSection(agentsMD))
	}

	prompt.ApplyCatalog(composer, b.skillCatalog())
	prompt.ApplyTaskFrame(composer, frame.FromMode(agent.ModeRegular))
	composer.SetVar("WORK_DIR", b.workDir)

	if env != nil && env.ProjectType != "" {
		composer.SetVar("PROJECT_TYPE", env.ProjectType)
	}

	return composer.Compose(env)
}

// resolveAgentsMDContent reads AGENTS.md from the work directory if it exists.
// Returns empty string if not found or unreadable.
func (b *Builder) resolveAgentsMDContent() string {
	agentsMDPath := filepath.Join(b.workDir, "AGENTS.md")

	content, err := os.ReadFile(agentsMDPath)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(content))
}

// addShellContext enriches environment with Shell context information.
func (b *Builder) addShellContext(ctx context.Context, env *agent.Environment) {
	info := b.shellService.GetContextInfo()

	enrichEnvironment(env, map[string]string{
		"shell_enabled": strconv.FormatBool(info.ShellEnabled),
	})

	if !info.ShellEnabled {
		return
	}

	enrichEnvironment(env, map[string]string{
		"shell":      info.Shell,
		"shell_path": info.ShellPath,
	})

	if b.logger != nil {
		b.logger.DebugContext(ctx, "shell context added", "shell", info.Shell)
	}
}
