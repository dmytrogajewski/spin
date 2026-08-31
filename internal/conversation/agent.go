package conversation

import (
	"context"
	"fmt"
	osexec "os/exec"
	"path/filepath"

	"github.com/dmytrogajewski/spin/internal/ace"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/agentsmd"
	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// agentBuildResult holds the outputs of buildAgent for downstream consumers.
type agentBuildResult struct {
	agent       *agent.Agent
	toolRuntime *tool.Runtime
	toolReg     *tools.Registry
	aceService  *ace.Service
	aceConfig   *ace.Config
	hookRunner  *hooks.Runner
}

// buildAgent constructs a fully configured agent with all services and integrations.
func (b *Builder) buildAgent(ctx context.Context, exec *agent.Executor, env *agent.Environment) (*agentBuildResult, error) {
	agentBuilder := agent.NewBuilder().
		WithConfig(b.cfg).
		WithProvider(b.llm).
		WithWorkingDir(b.workDir).
		WithEmitter(b.emitter).
		WithRuntime(b.runtime)

	detectionSvc := agentBuilder.BuildDetectionService()
	securitySvc := agentBuilder.BuildSecurityService(ctx)

	toolReg := b.buildOrRegisterTools(exec, securitySvc, env)

	if err := b.registerMemoryTools(toolReg); err != nil {
		b.logWarn("memory tools registration failed", "err", err)
	}

	hookRunner := hooks.NewRunner(hooks.Config{
		GlobalDir:     b.hooksGlobalDir(),
		ProjectDir:    filepath.Join(b.workDir, ".spin", "hooks"),
		Logger:        b.getLogger(),
		PluginScripts: b.pluginHookScripts(),
	})

	toolRuntime := tool.NewRuntime(tool.RuntimeConfig{
		Registry:        toolReg,
		Validator:       securitySvc.Validator(),
		ApprovalService: securitySvc.ApprovalService(),
		Emitter:         b.emitter,
		WorkDir:         env.WorkDir,
		HookRunner:      hookRunner,
		CompactEnabled:  b.cfg.Compact.Active(),
		CompactBackend:  b.cfg.Compact.Backend,
		LookPath:        osexec.LookPath,
	})

	opts := agentBuilder.BuildOptions()

	// Build optional services for component wiring.
	aceSvc, aceConfig := b.buildACEServices(ctx, agentBuilder)
	if aceSvc != nil {
		opts = append(opts, agent.WithACEService(aceSvc), agent.WithACEConfig(aceConfig))
	}

	agentsMDSvc := b.buildAgentsMDService(ctx, agentBuilder)
	if agentsMDSvc != nil {
		opts = append(opts, agent.WithAgentsMDService(agentsMDSvc))
	}

	opts = b.appendToolSelectorOptions(toolReg, opts)

	ag, err := agent.NewAgent(b.llm, securitySvc, detectionSvc, toolRuntime, env, b.emitter, opts...)
	if err != nil {
		return nil, fmt.Errorf("agent: %w", err)
	}

	return &agentBuildResult{
		agent:       ag,
		toolRuntime: toolRuntime,
		toolReg:     toolReg,
		aceService:  aceSvc,
		aceConfig:   aceConfig,
		hookRunner:  hookRunner,
	}, nil
}

// buildOrRegisterTools creates the tool registry, using runtime registration if available.
func (b *Builder) buildOrRegisterTools(exec *agent.Executor, securitySvc *safety.Service, env *agent.Environment) *tools.Registry {
	if b.runtime != nil {
		toolReg := tools.NewRegistry()
		b.runtime.RegisterTools(toolReg)
		_ = b.registerIntegrationTools(toolReg)

		return toolReg
	}

	return b.buildToolRegistry(exec, securitySvc, env)
}

// buildACEServices creates ACE service and config if ACE is enabled.
func (b *Builder) buildACEServices(ctx context.Context, agentBuilder *agent.Builder) (*ace.Service, *ace.Config) {
	if b.cfg == nil || !b.cfg.ACE.Enabled {
		return nil, nil
	}

	aceSvc, err := agentBuilder.BuildACEService(ctx)
	if err != nil {
		b.logWarn("ACE init failed, continuing", "err", err)

		return nil, nil
	}

	aceConfig := ace.ConvertConfig(&b.cfg.ACE)
	b.logInfo("ACE enabled", "playbook", b.cfg.ACE.PlaybookPath, "model", b.cfg.LLM.Model)

	return aceSvc, aceConfig
}

// buildAgentsMDService creates the AGENTS.md service if enabled.
func (b *Builder) buildAgentsMDService(ctx context.Context, agentBuilder *agent.Builder) *agentsmd.Service {
	if b.cfg == nil || !b.cfg.AgentsMD.Enabled {
		return nil
	}

	gitRoot := b.resolveGitRoot()

	agentsMDSvc := agentBuilder.BuildAgentsMDService(gitRoot)
	if agentsMDSvc == nil {
		return nil
	}

	if err := agentsMDSvc.Load(ctx); err != nil {
		b.logWarn("failed to load AGENTS.md", "error", err)

		return nil
	}

	if agentsMDSvc.IsLoaded() {
		b.logInfo("AGENTS.md loaded", "path", agentsMDSvc.Path())

		return agentsMDSvc
	}

	return nil
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
