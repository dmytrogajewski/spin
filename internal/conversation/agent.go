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
	// Use agent.Builder helper methods for service construction
	// Runtime provides: approval handler, tool registration, notifications
	// Emitter is passed from cmd layer and shared with runtime
	agentBuilder := agent.NewBuilder().
		WithConfig(b.cfg).
		WithProvider(b.llm).
		WithWorkingDir(b.workDir).
		WithEmitter(b.emitter).
		WithRuntime(b.runtime) // Runtime ALWAYS provides approval handler

	// Build detection using builder helper
	detectionSvc := agentBuilder.BuildDetectionService()

	// Shared validator + approval service for security + runtime
	// Use agent.Builder helper to build security service (uses runtime's approval handler if set)
	securitySvc := agentBuilder.BuildSecurityService()

	// Extract ApprovalService and Validator from SecurityService for ToolRuntime
	var approvalSvc *security.ApprovalService = securitySvc.ApprovalService()
	var runtimeValidator *security.Validator = securitySvc.Validator()

	// Build tool registry - use runtime's tool registration if available, otherwise build from integrations
	var toolReg *tools.Registry
	if b.runtime != nil {
		// Use runtime's tool registration
		toolReg = tools.NewRegistry()
		b.runtime.RegisterTools(toolReg)
		// Also register integration tools (MCP, Git)
		_ = b.registerIntegrationTools(toolReg)
	} else {
		// Fall back to conversation-level tool registry building
		toolReg = b.buildToolRegistry(exec, securitySvc, env)
	}

	// Register memory tools if memory service is available
	if err := b.registerMemoryTools(toolReg); err != nil {
		if b.logger != nil {
			b.logger.Warn("memory tools registration failed", "err", err)
		}
	}

	toolRuntime := agent.NewToolRuntime(agent.ToolRuntimeConfig{
		Registry:        toolReg,
		Validator:       runtimeValidator, // *security.Validator
		ApprovalService: approvalSvc,      // *security.ApprovalService
		Emitter:         b.emitter,
		WorkDir:         env.WorkDir,
	})

	// Build PlanningService using builder helper
	planningSvc := agentBuilder.BuildPlanningService()

	// Agent options using builder helper
	opts := agentBuilder.BuildAgentOptions()

	// ACE service using builder helper
	if b.cfg != nil && b.cfg.ACE.Enabled {
		aceSvc, err := agentBuilder.BuildACEService()
		if err != nil {
			if b.logger != nil {
				b.logger.Warn("ACE init failed, continuing", "err", err)
			}
		} else {
			opts = append(opts, agent.WithACEService(aceSvc))
			// Also pass ACE config to agent so it can emit events
			// Convert ConfigV2 ACE to agent.ACEConfig
			aceConfig := agent.ConvertACEConfig(&b.cfg.ACE)
			opts = append(opts, agent.WithACEConfig(aceConfig))
			if b.logger != nil {
				b.logger.Info("ACE enabled", "playbook", b.cfg.ACE.PlaybookPath, "model", b.cfg.LLM.Model)
			}
		}
	}

	// AGENTS.md service using builder helper
	if b.cfg != nil && b.cfg.AgentsMD.Enabled {
		// Get git root for discovery
		gitRoot := ""
		if b.gitService != nil && b.gitService.IsRepository() {
			if repo := b.gitService.GetIntegration().GetRepository(); repo != nil {
				gitRoot = repo.Root()
			}
		}
		agentsMDSvc := agentBuilder.BuildAgentsMDService(gitRoot)
		if agentsMDSvc != nil {
			// Load AGENTS.md content (errors are logged but don't fail startup)
			if err := agentsMDSvc.Load(context.Background()); err != nil {
				if b.logger != nil {
					b.logger.Warn("failed to load AGENTS.md", "error", err)
				}
			} else if agentsMDSvc.IsLoaded() {
				opts = append(opts, agent.WithAgentsMDService(agentsMDSvc))
				if b.logger != nil {
					b.logger.Info("AGENTS.md loaded", "path", agentsMDSvc.Path())
				}
			}
		}
	}

	// Dynamic tool selector (for MCP registries with dynamic_loadout)
	if b.toolSelector != nil {
		// Set the runtime registry so dynamically loaded tools can be registered
		b.toolSelector.SetRuntimeRegistry(toolReg)
		opts = append(opts, agent.WithToolSelector(b.toolSelector))
		if b.logger != nil {
			b.logger.Info("dynamic tool selection enabled")
		}
	}

	// Create agent
	ag, err := agent.NewAgent(b.llm, securitySvc, detectionSvc, toolRuntime, planningSvc, env, b.emitter, opts...)
	if err != nil {
		return nil, fmt.Errorf("agent: %w", err)
	}
	return ag, nil
}
