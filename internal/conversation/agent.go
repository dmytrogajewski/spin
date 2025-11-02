package conversation

import (
	"fmt"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/task"
)

// buildAgent constructs a fully configured agent with all services and integrations.
func (b *Builder) buildAgent(exec *agent.Executor, env *agent.Environment) (*agent.Agent, error) {
	validator := security.NewValidator()

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
	securitySvc := security.NewSecurityService(validator, approvalSvc)

	// Detection
	var (
		cycleDetector   detection.CycleDetector
		patternDetector detection.PatternDetector
	)
	if cfg := b.cfg; cfg != nil && cfg.CycleDetection.Enabled {
		c := cycle.Config{
			WindowSize:       cfg.CycleDetection.WindowSize,
			SimilarityThresh: cfg.CycleDetection.SimilarityThresh,
			ToolRepeatLimit:  cfg.CycleDetection.ToolRepeatLimit,
			ErrorRepeatLimit: cfg.CycleDetection.ErrorRepeatLimit,
			Enabled:          true,
		}
		cycleDetector = cycle.NewDetector(c)
		patternDetector = cycle.NewPatternDetector(c)
	} else {
		cycleDetector = cycle.NewDetector(cycle.Config{Enabled: false})
	}
	detectionSvc := detection.NewDetectionService(cycleDetector, patternDetector)

	// Orchestration
	toolReg := b.buildToolRegistry(exec, validator, env)
	taskReg := b.taskRegistry
	if taskReg == nil {
		taskReg = b.buildDefaultTaskRegistry()
	}
	toolExec := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolReg,
		Validator:       validator,
		ApprovalService: approvalSvc,
		Emitter:         b.emitter,
		WorkDir:         env.WorkDir,
	})
	orchestrationSvc := orchestration.NewOrchestrationService(toolExec, toolReg, taskReg)

	// Agent options
	opts := b.buildAgentOptions()

	// ACE (optional)
	if cfg := b.cfg; cfg != nil && cfg.ACEEnabled {
		defaultAgentCfg := agent.DefaultConfig()
		aceCfg := &agent.ACEConfig{
			Enabled:        true,
			PlaybookPath:   cfg.ACEPlaybookPath,
			TrajectoryPath: cfg.ACETrajectoryPath,
			Retrieval: agent.ACERetrievalConfig{
				TopK:     cfg.ACETopK,
				MinScore: cfg.ACEMinScore,
			},
			ItemizedLearning: agent.ACEItemizedLearningConfig{
				Enabled:       true,
				ParseFeedback: true,
				UpdateAsync:   false,
			},
			Generation: agent.ACEGenerationConfig{
				Enabled:     true,
				AutoReflect: true,
			},
			Adapter: defaultAgentCfg.ACE.Adapter,
			Refine:  defaultAgentCfg.ACE.Refine,
		}
		aceSvc, err := agent.NewACEService(aceCfg, env.WorkDir, b.llm, cfg.Model)
		if err != nil {
			if b.logger != nil {
				b.logger.Warn("ACE init failed, continuing", "err", err)
			}
		} else {
			opts = append(opts, agent.WithACEService(aceSvc))
			if b.logger != nil {
				b.logger.Info("ACE enabled", "playbook", cfg.ACEPlaybookPath, "model", cfg.Model)
			}
		}
	}

	ag, err := agent.NewAgent(b.llm, securitySvc, detectionSvc, orchestrationSvc, env, b.emitter, opts...)
	if err != nil {
		return nil, fmt.Errorf("agent: %w", err)
	}
	return ag, nil
}

// buildDefaultTaskRegistry creates a task registry with standard task types.
func (b *Builder) buildDefaultTaskRegistry() *orchestration.Registry {
	r := orchestration.NewRegistry()
	_ = r.Register("regular", task.NewRegular())
	_ = r.Register("review", task.NewReview())
	_ = r.Register("compact", task.NewCompact())
	_ = r.Register("planning", task.NewPlanning())
	_ = r.SetDefault("regular")
	if b.logger != nil {
		b.logger.Debug("default task registry created")
	}
	return r
}

// buildAgentOptions constructs agent options from configuration.
func (b *Builder) buildAgentOptions() []agent.AgentOption {
	opts := []agent.AgentOption{
		agent.WithRequireApproval(true),
	}
	if cfg := b.cfg; cfg != nil {
		if cfg.MaxTurns > 0 {
			opts = append(opts, agent.WithMaxTurns(cfg.MaxTurns))
		}
		if cfg.Timeout > 0 {
			opts = append(opts, agent.WithAgentTimeout(cfg.Timeout))
		}
		if cfg.Temperature > 0 {
			opts = append(opts, agent.WithTemperature(cfg.Temperature))
		}
		if cfg.MaxTokens > 0 {
			opts = append(opts, agent.WithMaxTokens(cfg.MaxTokens))
		}
	}
	return opts
}
