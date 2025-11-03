package agent

import (
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Builder constructs Agent instances with all dependencies.
type Builder struct {
	config          *Config
	provider        llm.Provider
	workingDir      string
	emitter         *events.EventEmitter
	approvalHandler security.ApprovalHandler
}

// NewBuilder creates a new agent builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// WithConfig sets the agent configuration.
func (b *Builder) WithConfig(cfg *Config) *Builder {
	b.config = cfg
	return b
}

// WithProvider sets the LLM provider.
func (b *Builder) WithProvider(provider llm.Provider) *Builder {
	b.provider = provider
	return b
}

// WithWorkingDir sets the working directory.
func (b *Builder) WithWorkingDir(dir string) *Builder {
	b.workingDir = dir
	return b
}

// WithEmitter sets the event emitter.
func (b *Builder) WithEmitter(emitter *events.EventEmitter) *Builder {
	b.emitter = emitter
	return b
}

// WithApprovalHandler sets the approval handler.
func (b *Builder) WithApprovalHandler(handler security.ApprovalHandler) *Builder {
	b.approvalHandler = handler
	return b
}

// buildExecutor creates an Executor with appropriate options based on configuration.
func (b *Builder) buildExecutor() *Executor {
	validator := security.NewValidator()
	opts := []ExecutorOption{
		WithValidator(validator),
	}

	if b.approvalHandler != nil {
		opts = append(opts,
			WithApprovalService(security.NewApprovalService(b.approvalHandler, b.emitter, validator)),
		)
	}

	if cfg := b.config; cfg != nil {
		if cfg.Timeout > 0 {
			opts = append(opts, WithTimeout(cfg.Timeout))
		}
		if cfg.CacheCommands {
			cache := NewCommandCache(5*time.Minute, 10*1024*1024) // 5m TTL, 10MB cap
			opts = append(opts, WithCache(cache))
		}
	}

	exec, err := NewExecutor(b.workingDir, opts...)
	if err != nil {
		// In builder pattern, we panic on invalid configuration
		// This should never happen with valid builder state
		panic("failed to create executor: " + err.Error())
	}
	return exec
}

// buildEnvironment gathers environment information for the working directory.
func (b *Builder) buildEnvironment() *Environment {
	opts := b.buildEnvironmentOptions()
	env, err := GatherEnvironment(b.workingDir, opts...)
	if err != nil {
		// In builder pattern, we panic on invalid configuration
		panic("failed to gather environment: " + err.Error())
	}
	return env
}

// buildEnvironmentOptions constructs environment options from configuration.
func (b *Builder) buildEnvironmentOptions() []EnvironmentOption {
	if b.config == nil {
		return nil
	}
	var opts []EnvironmentOption
	if b.config.MaxFiles > 0 {
		opts = append(opts, WithMaxFiles(b.config.MaxFiles))
	}
	if b.config.MaxDepth > 0 {
		opts = append(opts, WithMaxDepth(b.config.MaxDepth))
	}
	if b.config.SkipGit {
		opts = append(opts, WithSkipGit(true))
	}
	return opts
}

// Build constructs a fully configured Agent with all dependencies.
func (b *Builder) Build() (*Agent, error) {
	// Validate required fields
	if b.config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if b.provider == nil {
		return nil, fmt.Errorf("provider is required")
	}
	if b.workingDir == "" {
		return nil, fmt.Errorf("workingDir is required")
	}
	if b.emitter == nil {
		return nil, fmt.Errorf("emitter is required")
	}

	// Build core components
	executor := b.buildExecutor()
	env := b.buildEnvironment()

	// Build services
	securitySvc := b.buildSecurityService()
	detectionSvc := b.buildDetectionService()
	orchestrationSvc := b.buildOrchestrationService(executor, env)

	// Build agent options
	opts := b.buildAgentOptions()

	// Build ACE service if enabled
	if b.config != nil && b.config.ACE.Enabled {
		aceSvc, err := b.buildACEService()
		if err == nil {
			opts = append(opts, WithACEService(aceSvc))
		}
	}

	// Create agent
	return NewAgent(b.provider, securitySvc, detectionSvc, orchestrationSvc, env, b.emitter, opts...)
}

// buildSecurityService creates security service with approval handling.
func (b *Builder) buildSecurityService() *security.SecurityService {
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

	return security.NewSecurityService(validator, approvalSvc)
}

// buildDetectionService creates detection service with cycle detection.
func (b *Builder) buildDetectionService() *detection.DetectionService {
	var (
		cycleDetector   detection.CycleDetector
		patternDetector detection.PatternDetector
	)

	if b.config != nil && b.config.CycleDetection.Enabled {
		c := cycle.Config{
			WindowSize:       b.config.CycleDetection.WindowSize,
			SimilarityThresh: b.config.CycleDetection.SimilarityThresh,
			ToolRepeatLimit:  b.config.CycleDetection.ToolRepeatLimit,
			ErrorRepeatLimit: b.config.CycleDetection.ErrorRepeatLimit,
			Enabled:          true,
		}
		cycleDetector = cycle.NewDetector(c)
		patternDetector = cycle.NewPatternDetector(c)
	} else {
		cycleDetector = cycle.NewDetector(cycle.Config{Enabled: false})
	}

	return detection.NewDetectionService(cycleDetector, patternDetector)
}

// buildOrchestrationService creates orchestration service.
// Note: This is a simplified version. Full tool registry setup should be done by caller.
func (b *Builder) buildOrchestrationService(exec *Executor, env *Environment) *orchestration.OrchestrationService {
	// For now, return a basic orchestration service
	// The conversation layer will handle full tool/task registry setup
	validator := security.NewValidator()
	approvalSvc := security.NewApprovalService(b.approvalHandler, b.emitter, validator)

	toolReg := tools.NewRegistry()
	taskReg := orchestration.NewRegistry()

	toolExec := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolReg,
		Validator:       validator,
		ApprovalService: approvalSvc,
		Emitter:         b.emitter,
		WorkDir:         env.WorkDir,
	})

	return orchestration.NewOrchestrationService(toolExec, toolReg, taskReg)
}

// buildAgentOptions constructs agent options from configuration.
func (b *Builder) buildAgentOptions() []AgentOption {
	opts := []AgentOption{
		WithRequireApproval(true),
	}

	if b.config == nil {
		return opts
	}

	if b.config.MaxTurns > 0 {
		opts = append(opts, WithMaxTurns(b.config.MaxTurns))
	}
	if b.config.Timeout > 0 {
		opts = append(opts, WithAgentTimeout(b.config.Timeout))
	}
	if b.config.Temperature > 0 {
		opts = append(opts, WithTemperature(b.config.Temperature))
	}
	if b.config.MaxTokens > 0 {
		opts = append(opts, WithMaxTokens(b.config.MaxTokens))
	}

	return opts
}

// buildACEService creates ACE service if enabled.
func (b *Builder) buildACEService() (*ACEService, error) {
	if b.config == nil || !b.config.ACE.Enabled {
		return nil, fmt.Errorf("ACE not enabled")
	}

	return NewACEService(&b.config.ACE, b.workingDir, b.provider, b.config.Model)
}
