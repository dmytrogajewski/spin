package conversation

import (
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// registerIntegrationTools registers tools from MCP and Git integrations.
func (b *Builder) registerIntegrationTools(registry *tools.Registry) error {
	if registry == nil {
		return fmt.Errorf("tool registry is nil")
	}

	if b.mcpService != nil {
		if err := b.registerMCPTools(registry); err != nil {
			return fmt.Errorf("mcp tools: %w", err)
		}
	}
	if b.gitService != nil {
		if err := b.registerGitTools(registry); err != nil {
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
		if err := registry.Register(t); err != nil {
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
func (b *Builder) buildToolRegistry(exec *agent.Executor, validator *security.Validator, env *agent.Environment) *tools.Registry {
	registry := b.toolRegistry
	if registry == nil {
		registry = tools.NewRegistryWithBuiltins()
		if b.logger != nil {
			b.logger.Debug("created tool registry with builtins")
		}
	}

	var (
		validatorAdapt tools.CommandValidator
		shellCtxAdapt  tools.ShellContext
		execAdapt      tools.CommandExecutor
	)

	if validator != nil {
		validatorAdapt = &validatorAdapter{validator: validator}
	}
	if b.shellService != nil {
		shellCtxAdapt = &shellContextAdapter{shellCtx: b.shellService.GetContext()}
	}
	if exec != nil {
		execAdapt = &executorAdapter{executor: exec}
	}

	// Replace builtin tools with configured versions
	_ = registry.RegisterOrReplace(tools.NewShellCommandTool(validatorAdapt, shellCtxAdapt, execAdapt))
	_ = registry.RegisterOrReplace(tools.NewGetContextTool(env))
	_ = registry.RegisterOrReplace(tools.NewApplyPatchTool(env.WorkDir))
	_ = registry.RegisterOrReplace(tools.NewFileSearchTool(env.WorkDir))
	_ = registry.RegisterOrReplace(tools.NewGitContextTool(env.WorkDir))

	if err := b.registerIntegrationTools(registry); err != nil {
		if b.logger != nil {
			b.logger.Warn("integration tools registration failed", "err", err)
		}
	}
	return registry
}
