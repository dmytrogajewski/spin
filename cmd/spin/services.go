package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/runtime"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
)

// ProtocolServices holds the protocol services (Git, Shell, MCP).
type ProtocolServices struct {
	Git   *git.Service
	Shell *shell.Service
	MCP   *mcp.Service
}

// createServices creates Git, Shell, and MCP services based on config.
// Returns services and cleanup function for error handling.
// The cleanup function closes all created services in reverse order.
func createServices(ctx context.Context, cfg *config.ConfigV2, workDir string, logger *slog.Logger) (*ProtocolServices, func(), error) {
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

		gitSvc, err = git.NewService(true, workDir, logger)
		if err != nil {
			cleanup()

			return nil, nil, fmt.Errorf("create git service: %w", err)
		}
	}

	if cfg.Protocol.EnableShell {
		var err error

		shellSvc, err = shell.NewService(true, workDir, logger, cfg.Protocol.ShellTimeout)
		if err != nil {
			cleanup()

			return nil, nil, fmt.Errorf("create shell service: %w", err)
		}
	}

	logger.Debug("MCP config check", "enable_mcp", cfg.Protocol.EnableMCP, "servers_count", len(cfg.Protocol.MCPServers))

	if cfg.Protocol.EnableMCP && len(cfg.Protocol.MCPServers) > 0 {
		registryManager := mcp.NewDefaultRegistryManager(logger)

		for _, srv := range cfg.Protocol.MCPServers {
			logger.Debug("creating MCP registry", "name", srv.Name, "transport", srv.Transport, "dynamic_loadout", srv.DynamicLoadout)

			registry, err := createMCPRegistry(srv, logger)
			if err != nil {
				logger.Warn("failed to create MCP registry", "name", srv.Name, "err", err)

				continue
			}

			err = registryManager.Register(registry)
			if err != nil {
				logger.Warn("failed to register MCP registry", "name", srv.Name, "err", err)

				continue
			}

			logger.Debug("MCP registry registered", "name", srv.Name)
		}

		// Initialize all registries.
		for _, reg := range registryManager.All() {
			err := reg.Initialize(ctx)
			if err != nil {
				logger.Warn("failed to initialize MCP registry", "name", reg.Name(), "err", err)
			} else {
				logger.Debug("MCP registry initialized", "name", reg.Name())
			}
		}

		mcpSvc = mcp.NewService(registryManager)
		logger.Info("MCP service created", "registry_count", registryManager.RegistryCount())
	}

	return &ProtocolServices{
		Git:   gitSvc,
		Shell: shellSvc,
		MCP:   mcpSvc,
	}, cleanup, nil
}

// createBuiltinRuntime creates a builtin runtime with all required dependencies.
// This is shared between TUI and EXEC modes to ensure consistent runtime setup.
func createBuiltinRuntime(
	workDir string,
	emitter *events.EventEmitter,
	storage session.Storage,
	sessionID string,
	approvalHandler security.ApprovalHandler,
	services *ProtocolServices,
	ui ports.UI,
	logger *slog.Logger,
	cfg *config.ConfigV2,
) (*runtime.BuiltinRuntime, error) {
	// Build agent components needed for runtime.
	agentBuilder := agent.NewBuilder().
		WithConfig(cfg).
		WithWorkingDir(workDir).
		WithEmitter(emitter).
		WithApprovalHandler(approvalHandler)

		// Build security service and executor.
	securitySvc := agentBuilder.BuildSecurityService()
	exec := agentBuilder.BuildExecutor()
	validator := securitySvc.Validator()

	// Create builtin runtime.
	return runtime.NewBuiltinRuntime(runtime.BuiltinRuntimeConfig{
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

// createMCPRegistry creates an MCPRegistry based on server configuration.
func createMCPRegistry(srv config.MCPServerConfigV2, logger *slog.Logger) (mcp.MCPRegistry, error) {
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
		return nil, fmt.Errorf("unsupported transport: %s", transport)
	}
}

// hasDynamicRegistries checks if any MCP server has dynamic_loadout enabled.
func hasDynamicRegistries(cfg *config.ConfigV2) bool {
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
func createToolSelector(mcpSvc *mcp.Service, coreRegistry *tools.Registry, emitter *events.EventEmitter, cfg *config.ConfigV2, logger *slog.Logger) *agent.ToolSelector {
	if mcpSvc == nil {
		logger.Debug("tool selector: MCP service is nil")

		return nil
	}

	if !hasDynamicRegistries(cfg) {
		logger.Debug("tool selector: no dynamic registries configured")

		return nil
	}

	logger.Info("tool selector: creating with dynamic registries")

	return agent.NewToolSelector(
		mcpSvc,
		coreRegistry,
		emitter,
		agent.DefaultToolSelectionConfig(),
		logger,
	)
}
