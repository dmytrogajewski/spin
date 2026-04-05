package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
)

var (
	// ErrUnsupportedTransport is a sentinel error.
	ErrUnsupportedTransport = errors.New("unsupported transport")
	// ErrNoSessionDir is returned when no session directory is configured.
	ErrNoSessionDir = errors.New("no session directory configured")
)

// createSessionStorage creates a session storage if a directory is configured.
func createSessionStorage(sessionDir string) (session.Storage, error) {
	if sessionDir == "" {
		return nil, ErrNoSessionDir
	}

	storage, err := session.NewFileStorage(sessionDir)
	if err != nil {
		return nil, fmt.Errorf("create session storage: %w", err)
	}

	return storage, nil
}

// ProtocolServices holds the protocol services (Git, Shell, MCP).
type ProtocolServices struct {
	Git   *git.Service
	Shell *shell.Service
	MCP   *mcp.Service
}

// createServices creates Git, Shell, and MCP services based on config.
// Returns services and cleanup function for error handling.
// The cleanup function closes all created services in reverse order.
func createServices(ctx context.Context, cfg *config.V2, workDir string, logger *slog.Logger) (*ProtocolServices, func(), error) {
	var (
		gitSvc   *git.Service
		shellSvc *shell.Service
		mcpSvc   *mcp.Service
	)

	cleanup := func() {
		if mcpSvc != nil {
			mcpSvc.Close()
		}

		if shellSvc != nil {
			shellSvc.Close()
		}

		if gitSvc != nil {
			gitSvc.Close()
		}
	}

	if cfg.Protocol.EnableGit {
		var err error

		gitSvc, err = git.NewService(ctx, true, workDir, logger)
		if err != nil {
			cleanup()

			return nil, nil, fmt.Errorf("create git service: %w", err)
		}
	}

	if cfg.Protocol.EnableShell {
		var err error

		shellSvc, err = shell.NewService(ctx, true, workDir, logger, cfg.Protocol.ShellTimeout)
		if err != nil {
			cleanup()

			return nil, nil, fmt.Errorf("create shell service: %w", err)
		}
	}

	logger.DebugContext(ctx, "MCP config check", "enable_mcp", cfg.Protocol.EnableMCP, "servers_count", len(cfg.Protocol.MCPServers))

	mcpSvc = createMCPService(ctx, cfg, logger)

	return &ProtocolServices{
		Git:   gitSvc,
		Shell: shellSvc,
		MCP:   mcpSvc,
	}, cleanup, nil
}

// createMCPService creates the MCP service if MCP is enabled and servers are configured.
func createMCPService(ctx context.Context, cfg *config.V2, logger *slog.Logger) *mcp.Service {
	if !cfg.Protocol.EnableMCP || len(cfg.Protocol.MCPServers) == 0 {
		return nil
	}

	registryManager := mcp.NewDefaultRegistryManager(logger)

	for _, srv := range cfg.Protocol.MCPServers {
		logger.DebugContext(ctx, "creating MCP registry",
			"name", srv.Name, "transport", srv.Transport,
			"dynamic_loadout", srv.DynamicLoadout)

		registerMCPServer(ctx, registryManager, srv, logger)
	}

	initializeMCPRegistries(ctx, registryManager, logger)

	logger.InfoContext(ctx, "MCP service created", "registry_count", registryManager.RegistryCount())

	return mcp.NewService(registryManager)
}

// registerMCPServer creates and registers a single MCP server.
func registerMCPServer(
	ctx context.Context, registryManager *mcp.DefaultRegistryManager,
	srv config.MCPServerConfigV2, logger *slog.Logger,
) {
	registry, err := createMCPRegistry(srv, logger)
	if err != nil {
		logger.WarnContext(ctx, "failed to create MCP registry", "name", srv.Name, "err", err)

		return
	}

	err = registryManager.Register(registry)
	if err != nil {
		logger.WarnContext(ctx, "failed to register MCP registry", "name", srv.Name, "err", err)

		return
	}

	logger.DebugContext(ctx, "MCP registry registered", "name", srv.Name)
}

// initializeMCPRegistries initializes all registered MCP registries.
func initializeMCPRegistries(ctx context.Context, registryManager *mcp.DefaultRegistryManager, logger *slog.Logger) {
	for _, reg := range registryManager.All() {
		err := reg.Initialize(ctx)
		if err != nil {
			logger.WarnContext(ctx, "failed to initialize MCP registry", "name", reg.Name(), "err", err)
		} else {
			logger.DebugContext(ctx, "MCP registry initialized", "name", reg.Name())
		}
	}
}

// createBuiltinRuntime creates a builtin runtime with all required dependencies.
// This is shared between TUI and EXEC modes to ensure consistent runtime setup.
func createBuiltinRuntime(
	ctx context.Context,
	workDir string,
	emitter *events.EventEmitter,
	storage session.Storage,
	sessionID string,
	approvalHandler safety.ApprovalHandler,
	services *ProtocolServices,
	ui ports.UI,
	logger *slog.Logger,
	cfg *config.V2,
) (*executor.BuiltinRuntime, error) {
	// Build agent components needed for executor.
	agentBuilder := agent.NewBuilder().
		WithConfig(cfg).
		WithWorkingDir(workDir).
		WithEmitter(emitter).
		WithApprovalHandler(approvalHandler)

		// Build security service and executor.
	securitySvc := agentBuilder.BuildSecurityService(ctx)
	exec := agentBuilder.BuildExecutor(ctx)
	validator := securitySvc.Validator()

	// Create builtin executor.
	return executor.NewBuiltinRuntime(executor.BuiltinRuntimeConfig{
		WorkDir:         workDir,
		Emitter:         emitter,
		Storage:         storage,
		SessionID:       sessionID,
		Executor:        agent.NewExecutorRuntimeAdapter(exec),
		Validator:       validator,
		ShellService:    services.Shell,
		GitService:      services.Git,
		UI:              ui,
		ApprovalHandler: approvalHandler,
		Logger:          logger,
	})
}

// createMCPRegistry creates an Registry based on server configuration.
func createMCPRegistry(srv config.MCPServerConfigV2, logger *slog.Logger) (mcp.Registry, error) {
	transport := mcp.TransportType(srv.Transport)
	if transport == "" {
		transport = mcp.TransportStdio
	}

	switch transport {
	case mcp.TransportStdio:
		return mcp.NewLocalRegistry(mcp.LocalRegistryConfig{
			Name:    srv.Name,
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
			Logger:  logger,
		})

	case mcp.TransportSSE, mcp.TransportStreamableHTTP:
		var oauth *mcp.OAuthConfig
		if srv.OAuth != nil {
			oauth = &mcp.OAuthConfig{
				ClientID:     srv.OAuth.ClientID,
				ClientSecret: srv.OAuth.ClientSecret,
				RedirectURL:  srv.OAuth.RedirectURL,
				Scopes:       srv.OAuth.Scopes,
			}
		}

		return mcp.NewRemoteRegistry(mcp.RemoteRegistryConfig{
			Name:      srv.Name,
			Transport: transport,
			URL:       srv.URL,
			Headers:   srv.Headers,
			OAuth:     oauth,
			Logger:    logger,
		})

	case mcp.TransportSmithery:
		return mcp.NewSmitheryRegistry(mcp.SmitheryRegistryConfig{
			Name:      srv.Name,
			APIKey:    srv.SmitheryAPIKey,
			MCPURL:    srv.URL,
			Namespace: srv.SmitheryNamespace,
			Logger:    logger,
		})

	default:
		return nil, fmt.Errorf("unsupported transport: %s: %w", transport, ErrUnsupportedTransport)
	}
}

// hasDynamicRegistries checks if any MCP server has dynamic_loadout enabled.
func hasDynamicRegistries(cfg *config.V2) bool {
	if cfg == nil || !cfg.Protocol.EnableMCP {
		return false
	}

	for _, srv := range cfg.Protocol.MCPServers {
		if srv.DynamicLoadout {
			return true
		}
	}

	return false
}

// createToolSelector creates a ToolSelector if dynamic registries are configured.
func createToolSelector(
	ctx context.Context, mcpSvc *mcp.Service,
	coreRegistry *tools.Registry, emitter *events.EventEmitter,
	cfg *config.V2, logger *slog.Logger,
) *tool.Selector {
	if mcpSvc == nil {
		logger.DebugContext(ctx, "tool selector: MCP service is nil")

		return nil
	}

	if !hasDynamicRegistries(cfg) {
		logger.DebugContext(ctx, "tool selector: no dynamic registries configured")

		return nil
	}

	logger.InfoContext(ctx, "tool selector: creating with dynamic registries")

	return tool.NewSelector(
		mcpSvc,
		coreRegistry,
		emitter,
		tool.DefaultSelectionConfig(),
		logger,
	)
}
