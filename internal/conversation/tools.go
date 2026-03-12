package conversation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

var ErrToolRegistryIsNil = errors.New("tool registry is nil")

// registerIntegrationTools registers tools from MCP and Git integrations.
func (b *Builder) registerIntegrationTools(registry *tools.Registry) error {
	if registry == nil {
		return ErrToolRegistryIsNil
	}

	if b.mcpService != nil {
		err := b.registerMCPTools(registry)
		if err != nil {
			return fmt.Errorf("mcp tools: %w", err)
		}
	}

	if b.gitService != nil {
		err := b.registerGitTools(registry)
		if err != nil {
			return fmt.Errorf("git tools: %w", err)
		}
	}

	return nil
}

// registerMCPTools registers all tools provided by MCP servers.
func (b *Builder) registerMCPTools(registry *tools.Registry) error {
	mcpTools := b.mcpService.GetTools()
	if len(mcpTools) == 0 {
		return nil
	}

	var names []string

	for _, t := range mcpTools {
		err := registry.Register(t)
		if err != nil {
			if b.logger != nil {
				b.logger.Warn("mcp tool register failed", "tool", t.Name(), "err", err)
			}

			continue
		}

		names = append(names, t.Name())
	}

	if len(names) > 0 && b.logger != nil {
		b.logger.Info("mcp tools registered", "tools", strings.Join(names, ", "))
	}

	return nil
}

// registerGitTools registers Git operation tools.
func (b *Builder) registerGitTools(registry *tools.Registry) error {
	return registry.Register(tools.NewGitOperationTool(b.gitService.GetIntegration()))
}

// buildToolRegistry constructs a complete tool registry with all standard and integration tools.
func (b *Builder) buildToolRegistry(exec *agent.Executor, securityService *security.Service, env *agent.Environment) *tools.Registry {
	registry := b.toolRegistry
	if registry == nil {
		// Use shared factory to create base registry with configured tools.
		registry = tools.NewDefaultRegistry(env.WorkDir, env)

		if b.logger != nil {
			b.logger.Debug("created tool registry with builtins")
		}
	}

	var (
		validatorAdapt tools.CommandValidator
		shellCtxAdapt  tools.ShellContext
		execAdapt      tools.CommandExecutor
	)

	if securityService != nil {
		validatorAdapt = &validatorAdapter{securityService: securityService}
	}

	if b.shellService != nil {
		shellCtxAdapt = &shellContextAdapter{shellCtx: b.shellService.GetContext()}
	}

	if exec != nil {
		execAdapt = &executorAdapter{executor: exec}
	}

	// Replace shell_command tool with configured version (factory creates it with nil params)
	// Other tools (get_context, apply_patch, file_search, git_context) are already configured
	// by the factory, but we replace them again if they need different configuration.
	if validatorAdapt != nil || shellCtxAdapt != nil || execAdapt != nil {
		_ = registry.RegisterOrReplace(tools.NewShellCommandTool(validatorAdapt, shellCtxAdapt, execAdapt))
	}

	err := b.registerIntegrationTools(registry)
	if err != nil {
		if b.logger != nil {
			b.logger.Warn("integration tools registration failed", "err", err)
		}
	}

	return registry
}
