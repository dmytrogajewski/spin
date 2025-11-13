package agent

import (
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/config"
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
	unifiedConfig   *config.ConfigV2 // Unified config from config package (V2)
	provider        llm.Provider
	workingDir      string
	emitter         *events.EventEmitter
	approvalHandler security.ApprovalHandler
}

// NewBuilder creates a new agent builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// WithUnifiedConfig sets the unified configuration from config package (V2).
// This eliminates the need for config type conversions.
func (b *Builder) WithUnifiedConfig(cfg *config.ConfigV2) *Builder {
	b.unifiedConfig = cfg
	return b
}

// getTimeout returns timeout from unified config.
func (b *Builder) getTimeout() time.Duration {
	if b.unifiedConfig != nil {
		return b.unifiedConfig.Agent.Timeout
	}
	return 0
}

// getCacheCommands returns cache setting from unified config.
func (b *Builder) getCacheCommands() bool {
	if b.unifiedConfig != nil {
		return b.unifiedConfig.Agent.CacheCommands
	}
	return false
}

// getMaxFiles returns max files from unified config.
func (b *Builder) getMaxFiles() int {
	if b.unifiedConfig != nil {
		return b.unifiedConfig.Agent.MaxFiles
	}
	return 0
}

// getMaxDepth returns max depth from unified config.
func (b *Builder) getMaxDepth() int {
	if b.unifiedConfig != nil {
		return b.unifiedConfig.Agent.MaxDepth
	}
	return 0
}

// getSkipGit returns skip git from unified config.
func (b *Builder) getSkipGit() bool {
	if b.unifiedConfig != nil {
		return b.unifiedConfig.Agent.SkipGit
	}
	return false
}

// getMaxTurns returns max turns from unified config.
func (b *Builder) getMaxTurns() int {
	if b.unifiedConfig != nil {
		return b.unifiedConfig.Agent.MaxTurns
	}
	return 0
}

// getTemperature returns temperature from unified config.
func (b *Builder) getTemperature() float64 {
	if b.unifiedConfig != nil {
		return b.unifiedConfig.LLM.Temperature
	}
	return 0
}

// getMaxTokens returns max tokens from unified config.
func (b *Builder) getMaxTokens() int {
	if b.unifiedConfig != nil {
		return b.unifiedConfig.LLM.MaxTokens
	}
	return 0
}

// getModel returns model from unified config.
func (b *Builder) getModel() string {
	if b.unifiedConfig != nil {
		return b.unifiedConfig.LLM.Model
	}
	return ""
}

// isCycleDetectionEnabled returns whether cycle detection is enabled.
func (b *Builder) isCycleDetectionEnabled() bool {
	if b.unifiedConfig != nil {
		return b.unifiedConfig.Agent.CycleDetection.Enabled
	}
	return false
}

// getCycleDetectionConfig returns cycle detection config from unified config.
func (b *Builder) getCycleDetectionConfig() cycle.Config {
	if b.unifiedConfig != nil {
		return cycle.Config{
			WindowSize:       b.unifiedConfig.Agent.CycleDetection.WindowSize,
			SimilarityThresh: b.unifiedConfig.Agent.CycleDetection.SimilarityThresh,
			ToolRepeatLimit:  b.unifiedConfig.Agent.CycleDetection.ToolRepeatLimit,
			ErrorRepeatLimit: b.unifiedConfig.Agent.CycleDetection.ErrorRepeatLimit,
			Enabled:          true,
		}
	}
	return cycle.Config{Enabled: false}
}

// getACEConfig returns ACE config from unified config.
// Converts ConfigV2 ACE config to nested ACEConfig.
func (b *Builder) getACEConfig() *ACEConfig {
	if b.unifiedConfig != nil {
		// Convert V2 config to nested ACEConfig
		return &ACEConfig{
			Enabled:        b.unifiedConfig.ACE.Enabled,
			PlaybookPath:   b.unifiedConfig.ACE.PlaybookPath,
			TrajectoryPath: b.unifiedConfig.ACE.TrajectoryPath,
			Retrieval: ACERetrievalConfig{
				TopK:               b.unifiedConfig.ACE.TopK,
				MinScore:           b.unifiedConfig.ACE.MinScore,
				ProgressiveContext: DefaultProgressiveContextConfig(),
			},
			ItemizedLearning: ACEItemizedLearningConfig{
				Enabled:       true,
				ParseFeedback: true,
				UpdateAsync:   true,
			},
			Generation: ACEGenerationConfig{
				Enabled:     true,
				AutoReflect: true,
			},
			Adapter: ACEAdapterConfig{
				Enabled:          true,
				UtilityThreshold: 0.1,
				MaxMemorySize:    1000,
			},
			Refine: ACERefineConfig{
				Enabled:         true,
				Mode:            "proactive",
				MaxBullets:      1000,
				MaxTokens:       500000,
				MinUtilityScore: 0.1,
				CheckInterval:   100,
			},
		}
	}
	return nil
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

// BuildExecutor creates an Executor with appropriate options based on configuration.
// This is a public helper for use by conversation package.
func (b *Builder) BuildExecutor() *Executor {
	return b.buildExecutor()
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

	// Use unified config helpers
	if timeout := b.getTimeout(); timeout > 0 {
		opts = append(opts, WithTimeout(timeout))
	}
	if b.getCacheCommands() {
		cache := NewCommandCache(5*time.Minute, 10*1024*1024) // 5m TTL, 10MB cap
		opts = append(opts, WithCache(cache))
	}

	exec, err := NewExecutor(b.workingDir, opts...)
	if err != nil {
		// In builder pattern, we panic on invalid configuration
		// This should never happen with valid builder state
		panic("failed to create executor: " + err.Error())
	}
	return exec
}

// BuildEnvironment gathers environment information for the working directory.
// This is a public helper for use by conversation package.
func (b *Builder) BuildEnvironment() *Environment {
	return b.buildEnvironment()
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
	var opts []EnvironmentOption
	if maxFiles := b.getMaxFiles(); maxFiles > 0 {
		opts = append(opts, WithMaxFiles(maxFiles))
	}
	if maxDepth := b.getMaxDepth(); maxDepth > 0 {
		opts = append(opts, WithMaxDepth(maxDepth))
	}
	if b.getSkipGit() {
		opts = append(opts, WithSkipGit(true))
	}
	return opts
}

// BuildSecurityService creates security service with approval handling.
// This is a public helper for use by conversation package.
func (b *Builder) BuildSecurityService() *security.SecurityService {
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

// BuildDetectionService creates detection service with cycle detection.
// This is a public helper for use by conversation package.
func (b *Builder) BuildDetectionService() *detection.DetectionService {
	var (
		cycleDetector   detection.CycleDetector
		patternDetector detection.PatternDetector
	)

	if b.isCycleDetectionEnabled() {
		c := b.getCycleDetectionConfig()
		cycleDetector = cycle.NewDetector(c)
		patternDetector = cycle.NewPatternDetector(c)
	} else {
		cycleDetector = cycle.NewDetector(cycle.Config{Enabled: false})
	}

	return detection.NewDetectionService(cycleDetector, patternDetector)
}

// BuildAgentOptions constructs agent options from configuration.
// This is a public helper for use by conversation package.
func (b *Builder) BuildAgentOptions() []AgentOption {
	opts := []AgentOption{
		WithRequireApproval(true),
	}

	if maxTurns := b.getMaxTurns(); maxTurns > 0 {
		opts = append(opts, WithMaxTurns(maxTurns))
	}
	if timeout := b.getTimeout(); timeout > 0 {
		opts = append(opts, WithAgentTimeout(timeout))
	}
	if temp := b.getTemperature(); temp > 0 {
		opts = append(opts, WithTemperature(temp))
	}
	if maxTokens := b.getMaxTokens(); maxTokens > 0 {
		opts = append(opts, WithMaxTokens(maxTokens))
	}

	return opts
}

// BuildACEService creates ACE service if enabled.
// This is a public helper for use by conversation package.
func (b *Builder) BuildACEService() (*ACEService, error) {
	aceConfig := b.getACEConfig()
	if aceConfig == nil || !aceConfig.Enabled {
		return nil, fmt.Errorf("ACE not enabled")
	}

	return NewACEService(aceConfig, b.workingDir, b.provider, b.getModel(), b.getMaxTokens())
}

// Build constructs a fully configured Agent with all dependencies.
// This method wires together all the builder components to create a complete agent.
func (b *Builder) Build() (*Agent, error) {
	// Validate required fields
	if b.unifiedConfig == nil {
		return nil, fmt.Errorf("config is required (use WithUnifiedConfig)")
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
	securitySvc := b.BuildSecurityService()
	detectionSvc := b.BuildDetectionService()
	orchestrationSvc := b.buildOrchestrationService(executor, env)

	// Build agent options
	opts := b.BuildAgentOptions()

	// Build ACE service if enabled
	if b.unifiedConfig != nil && b.unifiedConfig.ACE.Enabled {
		aceSvc, err := b.BuildACEService()
		if err == nil {
			opts = append(opts, WithACEService(aceSvc))
			// Also set the ACE config so the agent can access it
			aceConfig := b.getACEConfig()
			if aceConfig != nil {
				opts = append(opts, WithACEConfig(aceConfig))
			}
		}
	}

	// Create agent
	return NewAgent(b.provider, securitySvc, detectionSvc, orchestrationSvc, env, b.emitter, opts...)
}

// buildOrchestrationService creates orchestration service.
// Note: This creates a basic orchestration service. The conversation layer
// can build a more complete tool registry with integration-specific tools.
func (b *Builder) buildOrchestrationService(exec *Executor, env *Environment) *orchestration.OrchestrationService {
	validator := security.NewValidator()
	approvalSvc := security.NewApprovalService(b.approvalHandler, b.emitter, validator)

	// Create registry with builtins
	toolReg := tools.NewRegistryWithBuiltins()

	toolExec := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolReg,
		Validator:       validator,
		ApprovalService: approvalSvc,
		Emitter:         b.emitter,
		WorkDir:         env.WorkDir,
	})

	return orchestration.NewOrchestrationService(toolExec, toolReg)
}
