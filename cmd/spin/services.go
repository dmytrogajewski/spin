package main

import (
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
func createServices(cfg *config.ConfigV2, workDir string, logger *slog.Logger) (*ProtocolServices, func(), error) {
	var gitSvc *git.Service
	var shellSvc *shell.Service
	var mcpSvc *mcp.Service

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

	if cfg.Protocol.EnableMCP && len(cfg.Protocol.MCPServers) > 0 {
		mcpCfg := &mcp.Config{
			EnableMCP:  true,
			MCPServers: make([]mcp.MCPServerConfig, len(cfg.Protocol.MCPServers)),
		}
		for i, srv := range cfg.Protocol.MCPServers {
			mcpCfg.MCPServers[i] = mcp.MCPServerConfig{
				Name:    srv.Name,
				Command: srv.Command,
				Args:    srv.Args,
				Env:     srv.Env,
			}
		}
		var err error
		mcpSvc, err = mcp.NewService(mcpCfg, logger)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("create mcp service: %w", err)
		}
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
	// Build agent components needed for runtime
	agentBuilder := agent.NewBuilder().
		WithConfig(cfg).
		WithWorkingDir(workDir).
		WithEmitter(emitter).
		WithApprovalHandler(approvalHandler)

	// Build security service and executor
	securitySvc := agentBuilder.BuildSecurityService()
	exec := agentBuilder.BuildExecutor()
	validator := securitySvc.Validator()

	// Create builtin runtime
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
