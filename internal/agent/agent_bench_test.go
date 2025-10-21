package agent

import (
	"context"
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

// Benchmark tests for task mode performance
// See: specs/frds/FRD-20251012200000-task-mode-performance-benchmarks.md

// ============================================================================
// Task Resolution Benchmarks
// ============================================================================

// BenchmarkAgent_ResolveTaskExplicit benchmarks resolving an explicit task object.
// Expected: ~50-100 ns/op (pointer comparison, should be instant)
func BenchmarkAgent_ResolveTaskExplicit(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewRegular()
	req := &AgentRequest{Task: taskObj}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolvedTask, err := agent.resolveTask(req)
		if err != nil {
			b.Fatal(err)
		}
		if resolvedTask == nil {
			b.Fatal("expected task")
		}
	}
}

// BenchmarkAgent_ResolveTaskByName benchmarks resolving task by name (registry lookup).
// Expected: ~100-150 ns/op (map lookup + RLock)
func BenchmarkAgent_ResolveTaskByName(b *testing.B) {
	agent := newBenchAgent(b)
	req := &AgentRequest{TaskName: "review"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolvedTask, err := agent.resolveTask(req)
		if err != nil {
			b.Fatal(err)
		}
		if resolvedTask == nil {
			b.Fatal("expected task")
		}
	}
}

// BenchmarkAgent_ResolveTaskDefault benchmarks default task fallback.
// Expected: ~50-100 ns/op (return cached default)
func BenchmarkAgent_ResolveTaskDefault(b *testing.B) {
	agent := newBenchAgent(b)
	req := &AgentRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolvedTask, err := agent.resolveTask(req)
		if err != nil {
			b.Fatal(err)
		}
		if resolvedTask == nil {
			b.Fatal("expected task")
		}
	}
}

// ============================================================================
// Tool Filtering Benchmarks
// ============================================================================

// BenchmarkAgent_ToolFilteringRegular benchmarks tool filtering for regular mode (all tools).
// Expected: ~30-50 μs/op (returns all tools, minimal filtering)
func BenchmarkAgent_ToolFilteringRegular(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewRegular()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolSchemas, err := agent.BuildToolsForTask(taskObj)
		if err != nil {
			b.Fatal(err)
		}
		if len(toolSchemas) == 0 {
			b.Fatal("expected tools")
		}
	}
}

// BenchmarkAgent_ToolFilteringReview benchmarks tool filtering for review mode (read-only).
// Expected: ~40-60 μs/op (filters out write tools)
func BenchmarkAgent_ToolFilteringReview(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewReview()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolSchemas, err := agent.BuildToolsForTask(taskObj)
		if err != nil {
			b.Fatal(err)
		}
		if len(toolSchemas) == 0 {
			b.Fatal("expected tools")
		}
	}
}

// BenchmarkAgent_ToolFilteringCompact benchmarks tool filtering for compact mode (minimal).
// Expected: ~20-30 μs/op (minimal set, fast filter)
func BenchmarkAgent_ToolFilteringCompact(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewCompact()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolSchemas, err := agent.BuildToolsForTask(taskObj)
		if err != nil {
			b.Fatal(err)
		}
		if len(toolSchemas) == 0 {
			b.Fatal("expected tools")
		}
	}
}

// BenchmarkAgent_ToolFilteringPlanning benchmarks tool filtering for planning mode (context tools).
// Expected: ~20-30 μs/op (context tools only)
func BenchmarkAgent_ToolFilteringPlanning(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewPlanning()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolSchemas, err := agent.BuildToolsForTask(taskObj)
		if err != nil {
			b.Fatal(err)
		}
		if len(toolSchemas) == 0 {
			b.Fatal("expected tools")
		}
	}
}

// BenchmarkAgent_ToolFilteringScaling benchmarks tool filtering with varying tool counts.
// Expected: Linear O(n) with tool count, < 100 μs for 50 tools
func BenchmarkAgent_ToolFilteringScaling(b *testing.B) {
	toolCounts := []int{10, 25, 50, 100}

	for _, count := range toolCounts {
		b.Run(fmt.Sprintf("tools=%d", count), func(b *testing.B) {
			agent := newBenchAgentWithTools(b, count)
			taskObj := task.NewReview()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				toolSchemas, err := agent.BuildToolsForTask(taskObj)
				if err != nil {
					b.Fatal(err)
				}
				_ = toolSchemas
			}
		})
	}
}

// ============================================================================
// Mode Switching Benchmarks
// ============================================================================

// NOTE: Conversation benchmarks removed to avoid cyclic import.
// Conversation benchmarks should be in internal/conversation/conversation_bench_test.go
// The following benchmarks were moved:
// - BenchmarkConversation_GetTaskMode
// - BenchmarkConversation_SetTaskMode
// - BenchmarkConversation_SetTaskModeConcurrent

// ============================================================================
// Memory Profiling Benchmarks
// ============================================================================

// BenchmarkAgent_TaskRegistryMemory benchmarks memory overhead of task registry.
// Expected: ~5-8 KB per agent (4 tasks + metadata)
func BenchmarkAgent_TaskRegistryMemory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		agent := newBenchAgent(b)
		_ = agent
	}
}

// BenchmarkAgent_ResolveTaskAllocs benchmarks allocations in task resolution.
// Expected: 0-1 allocs/op (should be zero for cached paths)
func BenchmarkAgent_ResolveTaskAllocs(b *testing.B) {
	agent := newBenchAgent(b)
	req := &AgentRequest{TaskName: "review"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agent.resolveTask(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAgent_ToolFilteringAllocs benchmarks allocations in tool filtering.
// Expected: 1-2 allocs/op (slice allocation for filtered tools)
func BenchmarkAgent_ToolFilteringAllocs(b *testing.B) {
	agent := newBenchAgent(b)
	taskObj := task.NewReview()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolSchemas, err := agent.BuildToolsForTask(taskObj)
		if err != nil {
			b.Fatal(err)
		}
		_ = toolSchemas
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// newBenchAgent creates a minimal agent for benchmarking.
func newBenchAgent(b *testing.B) *Agent {
	b.Helper()

	// Mock LLM provider (zero logic, fast)
	mockLLM := llm.NewMockProvider("test response")

	// Real tool registry with basic tools manually registered
	workDir := b.TempDir()
	toolRegistry := tools.NewRegistry()
	// Executor and validator (needed for execute_command tool)
	executor, _ := NewExecutor(workDir)
	validator := security.NewValidator()

	// Register built-in tools
	toolRegistry.Register(tools.NewReadFileTool())
	toolRegistry.Register(tools.NewWriteFileTool())
	toolRegistry.Register(tools.NewListDirectoryTool())
	toolRegistry.Register(tools.NewExecuteCommandTool(executor, validator))
	toolRegistry.Register(tools.NewGetContextTool(nil))
	toolRegistry.Register(tools.NewFileSearchTool(workDir))
	toolRegistry.Register(tools.NewGitContextTool(workDir))

	// Environment context
	env := &Environment{
		WorkDir: workDir,
	}

	// Event emitter
	emitter := events.NewEventEmitter(100)

	// Build services for agent
	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	cycleConfig := cycle.Config{Enabled: false}
	cycleDetector := cycle.NewDetector(cycleConfig)
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	taskRegistry := buildBenchTaskRegistry()
	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         workDir,
	})
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	// Create agent
	agent, err := NewAgent(mockLLM, securityService, detectionService, orchestrationService, env, emitter)
	if err != nil {
		b.Fatal(err)
	}

	return agent
}

// buildBenchTaskRegistry creates a task registry for benchmarks
func buildBenchTaskRegistry() *orchestration.Registry {
	registry := orchestration.NewRegistry()
	_ = registry.Register("regular", task.NewRegular())
	_ = registry.Register("review", task.NewReview())
	_ = registry.Register("compact", task.NewCompact())
	_ = registry.Register("planning", task.NewPlanning())
	_ = registry.SetDefault("regular")
	return registry
}

// newBenchAgentWithTools creates agent with specified number of tools.
func newBenchAgentWithTools(b *testing.B, toolCount int) *Agent {
	b.Helper()

	// Build base registry
	toolRegistry := tools.NewRegistry()

	// Add dummy tools to reach target count
	for i := 0; i < toolCount; i++ {
		tool := &dummyTool{name: fmt.Sprintf("dummy_%d", i)}
		_ = toolRegistry.Register(tool)
	}

	// Build agent with this registry
	workDir := b.TempDir()
	mockLLM := llm.NewMockProvider("ok")
	validator := security.NewValidator()
	executor, _ := NewExecutor(workDir)

	// Register execute_command tool in the registry
	_ = toolRegistry.Register(tools.NewExecuteCommandTool(executor, validator))

	env := &Environment{WorkDir: workDir}
	emitter := events.NewEventEmitter(100)

	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	cycleConfig := cycle.Config{Enabled: false}
	cycleDetector := cycle.NewDetector(cycleConfig)
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	taskRegistry := buildBenchTaskRegistry()
	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         workDir,
	})
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	agent, _ := NewAgent(mockLLM, securityService, detectionService, orchestrationService, env, emitter)
	return agent
}

// NOTE: newBenchConversation removed to avoid cyclic import.
// Conversation benchmarks should be in internal/conversation package tests.

// ============================================================================
// Mock/Dummy Types for Benchmarking
// ============================================================================

// dummyTool is a minimal tool implementation for testing.
type dummyTool struct {
	name string
}

func (t *dummyTool) Name() string {
	return t.name
}

func (t *dummyTool) Description() string {
	return "dummy tool for benchmarking"
}

func (t *dummyTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        t.name,
			Description: "dummy tool for benchmarking",
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: map[string]tools.PropertyDefinition{},
			},
		},
	}
}

func (t *dummyTool) Execute(ctx context.Context, params map[string]interface{}) (tools.ToolResult, error) {
	return tools.ToolResult{Success: true, Output: "ok"}, nil
}
