package agent

import (
	"errors"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/agentsmd"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/planning"
	"github.com/dmytrogajewski/spin/internal/security"
)

var ErrAceNotEnabled = errors.New("ACE not enabled")

// Builder constructs Agent instances with all dependencies.
type Builder struct {
	config          *config.V2 // Config from config package (V2).
	provider        llm.Provider
	workingDir      string
	emitter         *events.EventEmitter
	approvalHandler security.ApprovalHandler
	runtime         executor.Runtime // Optional runtime for tool registration and approval.
}

// NewBuilder creates a new agent builder.
func NewBuilder() *Builder {
	return &Builder{config: config.DefaultV2()}
}

// WithConfig sets the configuration from config package (V2).
func (b *Builder) WithConfig(cfg *config.V2) *Builder {
	if cfg != nil {
		b.config = cfg
	}

	return b
}

// getTimeout returns timeout from unified config.
func (b *Builder) getTimeout() time.Duration {
	return b.config.Agent.Timeout
}

// getCacheCommands returns cache setting from unified config.
func (b *Builder) getCacheCommands() bool {
	return b.config.Agent.CacheCommands
}

// getMaxFiles returns max files from unified config.
func (b *Builder) getMaxFiles() int {
	return b.config.Agent.MaxFiles
}

// getMaxDepth returns max depth from unified config.
func (b *Builder) getMaxDepth() int {
	return b.config.Agent.MaxDepth
}

// getSkipGit returns skip git from unified config.
func (b *Builder) getSkipGit() bool {
	return b.config.Agent.SkipGit
}

// getMaxTurns returns max turns from unified config.
func (b *Builder) getMaxTurns() int {
	return b.config.Agent.MaxTurns
}

// getTemperature returns temperature from unified config.
func (b *Builder) getTemperature() float64 {
	return b.config.LLM.Temperature
}

// getMaxTokens returns max tokens from unified config.
func (b *Builder) getMaxTokens() int {
	return b.config.LLM.MaxTokens
}

// getModel returns model from unified config.
func (b *Builder) getModel() string {
	return b.config.LLM.Model
}

// isCycleDetectionEnabled returns whether cycle detection is enabled.
func (b *Builder) isCycleDetectionEnabled() bool {
	return b.config.Agent.CycleDetection.Enabled
}

// getCycleDetectionConfig returns cycle detection config from unified config.
func (b *Builder) getCycleDetectionConfig() cycle.Config {
	return cycle.Config{
		WindowSize:       b.config.Agent.CycleDetection.WindowSize,
		SimilarityThresh: b.config.Agent.CycleDetection.SimilarityThresh,
		ToolRepeatLimit:  b.config.Agent.CycleDetection.ToolRepeatLimit,
		ErrorRepeatLimit: b.config.Agent.CycleDetection.ErrorRepeatLimit,
		Enabled:          b.config.Agent.CycleDetection.Enabled,
	}
}

// getACEConfig returns ACE config from unified config.
// Converts V2 ACE config to nested ACEConfig.
func (b *Builder) getACEConfig() *ACEConfig {
	// Convert V2 config to nested ACEConfig.
	return &ACEConfig{
		Enabled:        b.config.ACE.Enabled,
		PlaybookPath:   b.config.ACE.PlaybookPath,
		TrajectoryPath: b.config.ACE.TrajectoryPath,
		Retrieval: ACERetrievalConfig{
			TopK:               b.config.ACE.TopK,
			MinScore:           b.config.ACE.MinScore,
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

// WithRuntime sets the runtime for tool registration and approval handling.
// If set, the runtime's approval handler and tool registry are used.
func (b *Builder) WithRuntime(rt executor.Runtime) *Builder {
	b.runtime = rt

	return b
}

// BuildExecutor creates an Executor with appropriate options based on configuration.
// This is a public helper for use by conversation package.
func (b *Builder) BuildExecutor() *Executor {
	return b.newExecutor()
}

// newExecutor creates an Executor with appropriate options based on configuration.
func (b *Builder) newExecutor() *Executor {
	// Build SecurityService (which includes Validator internally).
	securityService := b.BuildSecurityService()
	opts := []ExecutorOption{
		WithSecurityService(securityService),
	}

	// Extract ApprovalService from SecurityService for Executor.
	approvalService := securityService.ApprovalService()
	if approvalService != nil {
		opts = append(opts, WithApprovalService(approvalService))
	}

	// Use unified config helpers.
	if timeout := b.getTimeout(); timeout > 0 {
		opts = append(opts, WithTimeout(timeout))
	}

	if b.getCacheCommands() {
		cache := NewCommandCache(5*time.Minute, 10*1024*1024) // 5m TTL, 10MB cap.
		opts = append(opts, WithCache(cache))
	}

	exec, err := NewExecutor(b.workingDir, opts...)
	if err != nil {
		// In builder pattern, we panic on invalid configuration
		// This should never happen with valid builder state.
		panic("failed to create executor: " + err.Error())
	}

	return exec
}

// BuildEnvironment gathers environment information for the working directory.
// This is a public helper for use by conversation package.
func (b *Builder) BuildEnvironment() *Environment {
	return b.gatherEnvironment()
}

// gatherEnvironment gathers environment information for the working directory.
func (b *Builder) gatherEnvironment() *Environment {
	opts := b.buildEnvironmentOptions()

	env, err := GatherEnvironment(b.workingDir, opts...)
	if err != nil {
		// In builder pattern, we panic on invalid configuration.
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
// If a runtime is set, uses the runtime's approval handler; otherwise uses the builder's approval handler.
func (b *Builder) BuildSecurityService() *security.Service {
	validator := security.NewValidator()

	// Use runtime's approval handler if available, otherwise use builder's.
	handler := b.approvalHandler
	if b.runtime != nil {
		handler = b.runtime.ApprovalHandler()
	}

	store := b.buildPolicyStore()

	cfg := security.ApprovalServiceConfig{
		Handler:           handler,
		Emitter:           b.emitter,
		Validator:         validator,
		Store:             store,
		SessionDefaultTTL: b.config.Security.SessionPolicyTTL,
		GlobalDefaultTTL:  b.config.Security.GlobalPolicyTTL,
	}
	approvalSvc := security.NewApprovalServiceWithConfig(cfg)

	return security.NewService(validator, approvalSvc)
}

// buildPolicyStore creates the appropriate policy store based on configuration.
func (b *Builder) buildPolicyStore() security.PolicyStore {
	if !b.config.Security.ApprovalPersistenceEnabled {
		return nil
	}

	if policyPath := b.config.Security.PolicyFile; policyPath != "" {
		fs, err := security.NewFilePolicyStore(policyPath, 30*time.Second)
		if err == nil {
			return fs
		}
	}

	return security.NewMemoryPolicyStore(30 * time.Second)
}

// BuildDetectionService creates detection service with cycle detection.
// This is a public helper for use by conversation package.
func (b *Builder) BuildDetectionService() *detection.Service {
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

	return detection.NewService(cycleDetector, patternDetector)
}

// BuildPlanningService creates planning service with LLM provider.
// This is a public helper for use by conversation package.
func (b *Builder) BuildPlanningService() *planning.Service {
	return planning.NewService(b.provider)
}

// BuildOptions constructs agent options from configuration.
// This is a public helper for use by conversation package.
func (b *Builder) BuildOptions() []Option {
	opts := []Option{
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
		return nil, ErrAceNotEnabled
	}

	return NewACEService(aceConfig, b.workingDir, b.provider, b.getModel(), b.getMaxTokens())
}

// BuildAgentsMDService creates AGENTS.md service if enabled.
// This is a public helper for use by conversation package.
// gitRoot is optional; pass empty string if not in a git repository.
func (b *Builder) BuildAgentsMDService(gitRoot string) *agentsmd.Service {
	if !b.config.AgentsMD.Enabled {
		return nil
	}

	cfg := &agentsmd.Config{
		Enabled: b.config.AgentsMD.Enabled,
		Path:    b.config.AgentsMD.Path,
		MaxSize: b.config.AgentsMD.MaxSize,
	}

	return agentsmd.NewService(cfg, b.workingDir, gitRoot)
}
