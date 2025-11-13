package conversation

import (
	"fmt"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
)

// buildAgent constructs a fully configured agent with all services and integrations.
func (b *Builder) buildAgent(exec *agent.Executor, env *agent.Environment) (*agent.Agent, error) {
	// Use agent.Builder helper methods for service construction
	agentBuilder := agent.NewBuilder().
		WithUnifiedConfig(b.cfg).
		WithProvider(b.llm).
		WithWorkingDir(b.workDir).
		WithEmitter(b.emitter).
		WithApprovalHandler(b.approvalHandler)

	// Build services using agent.Builder helpers
	securitySvc := agentBuilder.BuildSecurityService()
	detectionSvc := agentBuilder.BuildDetectionService()

	// Build tool registry at conversation level (with integrations)
	validator := security.NewValidator()
	toolReg := b.buildToolRegistry(exec, validator, env)

	// Build orchestration with conversation's tool registry
	var approvalSvc *security.ApprovalService
	if b.approvalHandler != nil {
		approvalSvc = security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{
			Handler:   b.approvalHandler,
			Emitter:   b.emitter,
			Validator: validator,
		})
	} else {
		approvalSvc = security.NewApprovalService(nil, b.emitter, validator)
	}

	toolExec := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolReg,
		Validator:       validator,
		ApprovalService: approvalSvc,
		Emitter:         b.emitter,
		WorkDir:         env.WorkDir,
	})
	orchestrationSvc := orchestration.NewOrchestrationService(toolExec, toolReg)

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

	// Create agent
	ag, err := agent.NewAgent(b.llm, securitySvc, detectionSvc, orchestrationSvc, env, b.emitter, opts...)
	if err != nil {
		return nil, fmt.Errorf("agent: %w", err)
	}
	return ag, nil
}
