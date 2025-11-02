package conversation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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

	// Build executor
	exec, err := b.buildExecutor(b.workDir)
	if err != nil {
		return nil, fmt.Errorf("build executor: %w", err)
	}

	// Gather environment
	env, err := b.gatherEnvironmentContext(b.workDir)
	if err != nil {
		return nil, fmt.Errorf("gather environment: %w", err)
	}

	// Build agent
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
