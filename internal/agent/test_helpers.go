package agent

import (
	"fmt"
	"testing"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// createTestAgentWithServices creates a fully configured agent for testing.
// This helper consolidates agent creation across all test files.
func createTestAgentWithServices(t *testing.T) *Agent {
	t.Helper()

	llmProvider := llm.NewMockProvider("test")
	validator := security.NewValidator()
	workDir := t.TempDir()
	env := &Environment{WorkDir: workDir}
	emitter := events.NewEventEmitter(100)

	// Build SecurityService
	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	// Build DetectionService
	cycleConfig := cycle.Config{
		WindowSize:       3,
		SimilarityThresh: 0.8,
		ToolRepeatLimit:  3,
		ErrorRepeatLimit: 2,
		Enabled:          true,
	}
	cycleDetector := cycle.NewDetector(cycleConfig)
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	// Build tool registry with built-in tools
	toolRegistry := tools.NewRegistry()
	executor, err := NewExecutor(workDir)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	_ = toolRegistry.Register(tools.NewReadFileTool())
	_ = toolRegistry.Register(tools.NewWriteFileTool())
	_ = toolRegistry.Register(tools.NewListDirectoryTool())
	_ = toolRegistry.Register(tools.NewExecuteCommandTool(executor, validator))
	_ = toolRegistry.Register(tools.NewGetContextTool(env))
	_ = toolRegistry.Register(tools.NewApplyPatchTool(workDir))
	_ = toolRegistry.Register(tools.NewFileSearchTool(workDir))
	_ = toolRegistry.Register(tools.NewGitContextTool(workDir))

	// Build task registry (using orchestration.Registry, not task.Registry)
	taskRegistry := orchestration.NewRegistry()
	_ = taskRegistry.Register("regular", task.NewRegular())
	_ = taskRegistry.Register("review", task.NewReview())
	_ = taskRegistry.Register("compact", task.NewCompact())
	_ = taskRegistry.Register("planning", task.NewPlanning())
	_ = taskRegistry.SetDefault("regular")

	// Build OrchestrationService
	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         workDir,
	})
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	// Create agent
	agent, err := NewAgent(llmProvider, securityService, detectionService, orchestrationService, env, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	return agent
}

// newAgentForTest is a test-only wrapper that provides the old NewAgent signature
// but builds services internally. This allows existing tests to continue working.
// USE THIS IN TESTS INSTEAD OF NewAgent TO AVOID UPDATING EVERY TEST FILE.
func newAgentForTest(
	provider llm.Provider,
	executor *Executor,
	validator *security.Validator,
	environment *Environment,
	emitter *events.EventEmitter,
	opts ...AgentOption,
) (*Agent, error) {
	// Build SecurityService
	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	// Build DetectionService
	cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	// Build tool registry
	toolRegistry := tools.NewRegistry()
	_ = toolRegistry.Register(tools.NewReadFileTool())
	_ = toolRegistry.Register(tools.NewWriteFileTool())
	_ = toolRegistry.Register(tools.NewListDirectoryTool())
	_ = toolRegistry.Register(tools.NewExecuteCommandTool(executor, validator))
	_ = toolRegistry.Register(tools.NewGetContextTool(environment))
	_ = toolRegistry.Register(tools.NewApplyPatchTool(environment.WorkDir))
	_ = toolRegistry.Register(tools.NewFileSearchTool(environment.WorkDir))
	_ = toolRegistry.Register(tools.NewGitContextTool(environment.WorkDir))

	// Build task registry (using orchestration.Registry, not task.Registry)
	taskRegistry := orchestration.NewRegistry()
	_ = taskRegistry.Register("regular", task.NewRegular())
	_ = taskRegistry.Register("review", task.NewReview())
	_ = taskRegistry.Register("compact", task.NewCompact())
	_ = taskRegistry.Register("planning", task.NewPlanning())
	_ = taskRegistry.SetDefault("regular")

	// Build OrchestrationService
	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         environment.WorkDir,
	})
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	// Create agent with services using the real NewAgent function
	agent := &Agent{
		llm:           provider,
		security:      securityService,
		detection:     detectionService,
		orchestration: orchestrationService,
		context:       environment,
		emitter:       emitter,
		config:        DefaultConfig(),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(agent); err != nil {
			return nil, fmt.Errorf("applying option: %w", err)
		}
	}

	return agent, nil
}
