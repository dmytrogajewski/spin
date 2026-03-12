package conversation

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// buildAgent constructs a fully configured agent with all services and integrations.
func (b *Builder) buildAgent(exec *agent.Executor, env *agent.Environment) (*agent.Agent, error) {
	agentBuilder := agent.NewBuilder().
		WithConfig(b.cfg).
		WithProvider(b.llm).
		WithWorkingDir(b.workDir).
		WithEmitter(b.emitter).
		WithRuntime(b.runtime)

	detectionSvc := agentBuilder.BuildDetectionService()
	securitySvc := agentBuilder.BuildSecurityService()

	toolReg := b.buildOrRegisterTools(exec, securitySvc, env)

	if err := b.registerMemoryTools(toolReg); err != nil {
		b.logWarn("memory tools registration failed", "err", err)
	}

	toolRuntime := agent.NewToolRuntime(agent.ToolRuntimeConfig{
		Registry:        toolReg,
		Validator:       securitySvc.Validator(),
		ApprovalService: securitySvc.ApprovalService(),
		Emitter:         b.emitter,
		WorkDir:         env.WorkDir,
	})

	planningSvc := agentBuilder.BuildPlanningService()
	opts := agentBuilder.BuildOptions()
	opts = b.appendACEOptions(agentBuilder, opts)
	opts = b.appendAgentsMDOptions(agentBuilder, opts)
	opts = b.appendToolSelectorOptions(toolReg, opts)

	ag, err := agent.NewAgent(b.llm, securitySvc, detectionSvc, toolRuntime, planningSvc, env, b.emitter, opts...)
	if err != nil {
		return nil, fmt.Errorf("agent: %w", err)
	}

	return ag, nil
}

// buildOrRegisterTools creates the tool registry, using runtime registration if available.
func (b *Builder) buildOrRegisterTools(exec *agent.Executor, securitySvc *security.Service, env *agent.Environment) *tools.Registry {
	if b.runtime != nil {
		toolReg := tools.NewRegistry()
		b.runtime.RegisterTools(toolReg)
		_ = b.registerIntegrationTools(toolReg)
		return toolReg
	}
	return b.buildToolRegistry(exec, securitySvc, env)
}

// appendACEOptions adds ACE-related agent options if ACE is enabled.
func (b *Builder) appendACEOptions(agentBuilder *agent.Builder, opts []agent.Option) []agent.Option {
	if b.cfg == nil || !b.cfg.ACE.Enabled {
		return opts
	}

	aceSvc, err := agentBuilder.BuildACEService()
	if err != nil {
		b.logWarn("ACE init failed, continuing", "err", err)
		return opts
	}

	opts = append(opts, agent.WithACEService(aceSvc))
	aceConfig := agent.ConvertACEConfig(&b.cfg.ACE)
	opts = append(opts, agent.WithACEConfig(aceConfig))
	b.logInfo("ACE enabled", "playbook", b.cfg.ACE.PlaybookPath, "model", b.cfg.LLM.Model)

	return opts
}

// appendAgentsMDOptions adds AGENTS.md-related agent options if enabled.
func (b *Builder) appendAgentsMDOptions(agentBuilder *agent.Builder, opts []agent.Option) []agent.Option {
	if b.cfg == nil || !b.cfg.AgentsMD.Enabled {
		return opts
	}

	gitRoot := b.resolveGitRoot()
	agentsMDSvc := agentBuilder.BuildAgentsMDService(gitRoot)
	if agentsMDSvc == nil {
		return opts
	}

	if err := agentsMDSvc.Load(context.Background()); err != nil {
		b.logWarn("failed to load AGENTS.md", "error", err)
		return opts
	}

	if agentsMDSvc.IsLoaded() {
		opts = append(opts, agent.WithAgentsMDService(agentsMDSvc))
		b.logInfo("AGENTS.md loaded", "path", agentsMDSvc.Path())
	}

	return opts
}

// resolveGitRoot returns the git repository root, or empty string if unavailable.
func (b *Builder) resolveGitRoot() string {
	if b.gitService == nil || !b.gitService.IsRepository() {
		return ""
	}
	if repo := b.gitService.GetIntegration().GetRepository(); repo != nil {
		return repo.Root()
	}
	return ""
}

// appendToolSelectorOptions adds dynamic tool selector options if configured.
func (b *Builder) appendToolSelectorOptions(toolReg *tools.Registry, opts []agent.Option) []agent.Option {
	if b.toolSelector == nil {
		return opts
	}

	b.toolSelector.SetRuntimeRegistry(toolReg)
	opts = append(opts, agent.WithToolSelector(b.toolSelector))
	b.logInfo("dynamic tool selection enabled")

	return opts
}

// logWarn logs a warning message if logger is available.
func (b *Builder) logWarn(msg string, args ...any) {
	if b.logger != nil {
		b.logger.Warn(msg, args...)
	}
}

// logInfo logs an info message if logger is available.
func (b *Builder) logInfo(msg string, args ...any) {
	if b.logger != nil {
		b.logger.Info(msg, args...)
	}
}
