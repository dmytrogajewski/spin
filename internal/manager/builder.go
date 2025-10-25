package manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Builder constructs a Manager instance with all its dependencies.
// This pattern separates configuration validation, dependency initialization,
// and integration setup from the Manager's operational logic.
type Builder struct {
	// Core configuration
	cfg *Config

	// Optional overrides
	llm             llm.Provider
	emitter         *events.EventEmitter
	storage         session.Storage
	toolRegistry    *tools.Registry
	taskRegistry    *orchestration.Registry
	approvalHandler security.ApprovalHandler
	logger          *slog.Logger

	// Managed resources (initialized during Build)
	authManager      *auth.Manager
	mcpManager       *mcp.MCPManager
	gitIntegration   *git.GitIntegration
	shellIntegration *shell.ShellIntegration
}

// NewBuilder creates a new Manager builder with the given configuration.
func NewBuilder(cfg *Config) *Builder {
	return &Builder{
		cfg: cfg,
	}
}

// WithLLM sets a custom LLM provider.
func (b *Builder) WithLLM(provider llm.Provider) *Builder {
	b.llm = provider
	return b
}

// WithEventEmitter sets a custom event emitter.
func (b *Builder) WithEventEmitter(emitter *events.EventEmitter) *Builder {
	b.emitter = emitter
	return b
}

// WithStorage sets a custom session storage.
func (b *Builder) WithStorage(storage session.Storage) *Builder {
	b.storage = storage
	return b
}

// WithToolRegistry sets a custom tool registry.
func (b *Builder) WithToolRegistry(registry *tools.Registry) *Builder {
	b.toolRegistry = registry
	return b
}

// WithTaskRegistry sets a custom task registry.
func (b *Builder) WithTaskRegistry(registry *orchestration.Registry) *Builder {
	b.taskRegistry = registry
	return b
}

// WithApprovalHandler sets a custom approval handler.
func (b *Builder) WithApprovalHandler(handler security.ApprovalHandler) *Builder {
	b.approvalHandler = handler
	return b
}

// WithLogger sets a custom logger.
func (b *Builder) WithLogger(logger *slog.Logger) *Builder {
	b.logger = logger
	return b
}

// Build constructs and returns a fully initialized Manager.
// This method orchestrates all initialization steps in a clear, testable way.
func (b *Builder) Build(ctx context.Context) (*Manager, error) {
	// Validate configuration
	if err := b.validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	logger := b.getLogger()
	logger.Info("building manager", "provider", b.cfg.Provider, "model", b.cfg.Model)

	// Initialize core dependencies
	if err := b.initializeCoreDependencies(); err != nil {
		return nil, fmt.Errorf("initialize core dependencies: %w", err)
	}

	// Initialize integrations
	if err := b.initializeIntegrations(ctx, logger); err != nil {
		return nil, fmt.Errorf("initialize integrations: %w", err)
	}

	// Build the Manager
	mgr := &Manager{
		cfg:              b.cfg,
		llm:              b.llm,
		emitter:          b.emitter,
		storage:          b.storage,
		toolRegistry:     b.toolRegistry,
		taskRegistry:     b.taskRegistry,
		approvalHandler:  b.approvalHandler,
		authManager:      b.authManager,
		mcpManager:       b.mcpManager,
		gitIntegration:   b.gitIntegration,
		shellIntegration: b.shellIntegration,
		logger:           b.logger,
	}

	logger.Info("manager built successfully")
	return mgr, nil
}

// validate ensures the configuration is valid and required fields are set.
func (b *Builder) validate() error {
	if b.cfg == nil {
		return errors.New("config cannot be nil")
	}
	if err := b.cfg.Validate(); err != nil {
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

// initializeIntegrations sets up MCP, Git, and Shell integrations based on config.
func (b *Builder) initializeIntegrations(ctx context.Context, logger *slog.Logger) error {
	// Initialize MCP if enabled
	if b.cfg.EnableMCP {
		if err := b.initializeMCP(ctx, logger); err != nil {
			// Log but don't fail - MCP is optional
			logger.Error("failed to initialize MCP manager", "error", err)
		}
	}

	// Initialize Git if enabled
	if b.cfg.EnableGit {
		if err := b.initializeGit(ctx, logger); err != nil {
			// Log but don't fail - Git is optional
			logger.Error("failed to initialize Git integration", "error", err)
		}
	}

	// Initialize Shell if enabled
	if b.cfg.EnableShell {
		if err := b.initializeShell(ctx, logger); err != nil {
			// Log but don't fail - Shell is optional
			logger.Error("failed to initialize Shell integration", "error", err)
		}
	}

	return nil
}

// initializeMCP sets up the MCP manager with configured servers.
func (b *Builder) initializeMCP(ctx context.Context, logger *slog.Logger) error {
	mcpConfig := &mcp.Config{
		EnableMCP:  b.cfg.EnableMCP,
		MCPServers: make([]mcp.MCPServerConfig, len(b.cfg.MCPServers)),
	}

	for i, srv := range b.cfg.MCPServers {
		mcpConfig.MCPServers[i] = mcp.MCPServerConfig{
			Name:    srv.Name,
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
		}
	}

	mcpManager := mcp.NewMCPManager(mcpConfig, logger)
	if err := mcpManager.Initialize(ctx); err != nil {
		return fmt.Errorf("mcp manager initialization: %w", err)
	}

	b.mcpManager = mcpManager
	logger.Info("MCP manager initialized", "servers", len(b.cfg.MCPServers))
	return nil
}

// initializeGit sets up the Git integration.
func (b *Builder) initializeGit(ctx context.Context, logger *slog.Logger) error {
	gitIntegration := git.NewGitIntegration(true, b.cfg.WorkDir, logger)
	if err := gitIntegration.Initialize(ctx); err != nil {
		return fmt.Errorf("git integration initialization: %w", err)
	}

	b.gitIntegration = gitIntegration
	logger.Info("Git integration initialized", "work_dir", b.cfg.WorkDir)
	return nil
}

// initializeShell sets up the Shell integration.
func (b *Builder) initializeShell(ctx context.Context, logger *slog.Logger) error {
	shellIntegration := shell.NewShellIntegration(
		true,
		b.cfg.WorkDir,
		logger,
		b.cfg.ShellTimeout,
	)
	if err := shellIntegration.Initialize(ctx); err != nil {
		return fmt.Errorf("shell integration initialization: %w", err)
	}

	b.shellIntegration = shellIntegration
	logger.Info("Shell integration initialized",
		"work_dir", b.cfg.WorkDir,
		"timeout", b.cfg.ShellTimeout)
	return nil
}
