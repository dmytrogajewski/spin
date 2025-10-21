package conversation

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// setupTestConv creates a test conversation with all dependencies
func setupTestConv(t *testing.T) *Conversation {
	t.Helper()

	llmProvider := llm.NewMockProvider("ok")
	validator := security.NewValidator()
	workDir := t.TempDir()

	executor, err := agent.NewExecutor(workDir)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	env := &agent.Environment{WorkDir: workDir}
	emitter := events.NewEventEmitter(100)

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
	agentInstance, err := agent.NewAgent(llmProvider, securityService, detectionService, orchestrationService, env, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	hist := history.NewHistoryWithDefaults()
	return NewConversation(agentInstance, hist, emitter, "test-session-id")
}
