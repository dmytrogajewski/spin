package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/task"
)

// TestAgent_TaskBudgetOverridesConfig verifies that a task's MaxTokens
// overrides the agent's config.MaxTokens when task.MaxTokens() > 0.
func TestAgent_TaskBudgetOverridesConfig(t *testing.T) {
	// Create agent with 4K config
	llmCapture := newCapturingLLMProvider()
	validator := security.NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := events.NewEventEmitter(100)

	agent, err := newAgentForTest(llmCapture, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Override config to 4K tokens
	agent.config.MaxTokens = 4096

	// Regular mode has 16K tokens
	regularTask := task.NewRegular()
	if regularTask.MaxTokens() != 16384 {
		t.Fatalf("expected regular task to have 16384 tokens, got %d", regularTask.MaxTokens())
	}

	// Create request with regular task
	req := &AgentRequest{
		Input: "test input",
		Task:  regularTask,
	}

	// Execute (will fail because no tools, but that's ok - we just want to capture the request)
	_, _ = agent.Execute(context.Background(), req)

	// Verify task budget was used (16K, not 4K from config)
	capturedTokens := llmCapture.GetLastMaxTokens()
	if capturedTokens != 16384 {
		t.Errorf("expected task budget 16384 to override config 4096, got %d", capturedTokens)
	}
}

// TestAgent_AgentConfigFallbackWhenTaskBudgetZero verifies that when
// a task returns 0 for MaxTokens(), the agent config is used instead.
func TestAgent_AgentConfigFallbackWhenTaskBudgetZero(t *testing.T) {
	llmCapture := newCapturingLLMProvider()
	validator := security.NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := events.NewEventEmitter(100)

	agent, err := newAgentForTest(llmCapture, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Set agent config to 8K tokens
	agent.config.MaxTokens = 8192

	// Custom task with zero budget (should fall back to agent config)
	zeroTask := &mockTaskWithBudget{
		name:      "zero-budget",
		maxTokens: 0, // Zero means no override
	}

	req := &AgentRequest{
		Input: "test input",
		Task:  zeroTask,
	}

	_, _ = agent.Execute(context.Background(), req)

	// Verify agent config was used
	capturedTokens := llmCapture.GetLastMaxTokens()
	if capturedTokens != 8192 {
		t.Errorf("expected agent config 8192 when task budget is 0, got %d", capturedTokens)
	}
}

// TestAgent_CompactModeUses4KBudget verifies that compact mode
// restricts the token budget to 4K even when agent config is higher.
func TestAgent_CompactModeUses4KBudget(t *testing.T) {
	llmCapture := newCapturingLLMProvider()
	validator := security.NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := events.NewEventEmitter(100)

	agent, err := newAgentForTest(llmCapture, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Set agent config to 16K tokens
	agent.config.MaxTokens = 16384

	// Compact mode has 4K tokens
	compactTask := task.NewCompact()
	if compactTask.MaxTokens() != 4096 {
		t.Fatalf("expected compact task to have 4096 tokens, got %d", compactTask.MaxTokens())
	}

	req := &AgentRequest{
		Input: "Quick question",
		Task:  compactTask,
	}

	_, _ = agent.Execute(context.Background(), req)

	// Verify compact budget was used (4K, not 16K from config)
	capturedTokens := llmCapture.GetLastMaxTokens()
	if capturedTokens != 4096 {
		t.Errorf("expected compact mode to restrict to 4096 tokens, got %d", capturedTokens)
	}
}

// TestAgent_AllTaskModesApplyCorrectBudgets verifies that each built-in
// task mode applies its correct token budget.
func TestAgent_AllTaskModesApplyCorrectBudgets(t *testing.T) {
	tests := []struct {
		name           string
		task           Task
		agentMaxTokens int
		wantMaxTokens  int
	}{
		{
			name:           "regular mode overrides low config",
			task:           task.NewRegular(),
			agentMaxTokens: 4096,
			wantMaxTokens:  16384, // task overrides
		},
		{
			name:           "review mode overrides low config",
			task:           task.NewReview(),
			agentMaxTokens: 4096,
			wantMaxTokens:  12288, // task overrides
		},
		{
			name:           "compact mode restricts high config",
			task:           task.NewCompact(),
			agentMaxTokens: 16384,
			wantMaxTokens:  4096, // task restricts
		},
		{
			name:           "planning mode restricts high config",
			task:           task.NewPlanning(),
			agentMaxTokens: 16384,
			wantMaxTokens:  4096, // task restricts
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llmCapture := newCapturingLLMProvider()
			validator := security.NewValidator()
			executor, err := NewExecutor(t.TempDir())
			if err != nil {
				t.Fatalf("failed to create executor: %v", err)
			}
			ctx := &Environment{WorkDir: t.TempDir()}
			emitter := events.NewEventEmitter(100)

			agent, err := newAgentForTest(llmCapture, executor, validator, ctx, emitter)
			if err != nil {
				t.Fatalf("failed to create agent: %v", err)
			}

			agent.config.MaxTokens = tt.agentMaxTokens

			req := &AgentRequest{
				Input: "test",
				Task:  tt.task,
			}

			_, _ = agent.Execute(context.Background(), req)

			capturedTokens := llmCapture.GetLastMaxTokens()
			if capturedTokens != tt.wantMaxTokens {
				t.Errorf("task=%s agent=%d want=%d got=%d",
					tt.task.Name(), tt.agentMaxTokens, tt.wantMaxTokens, capturedTokens)
			}
		})
	}
}

// TestAgent_NilTaskUsesAgentConfig verifies that when task is nil,
// the default task is used, which should have its own MaxTokens.
func TestAgent_NilTaskUsesAgentConfig(t *testing.T) {
	llmCapture := newCapturingLLMProvider()
	validator := security.NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := events.NewEventEmitter(100)

	agent, err := newAgentForTest(llmCapture, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	agent.config.MaxTokens = 8192

	// Request with no task (will use default which is "regular" with 16K)
	req := &AgentRequest{
		Input: "test",
		Task:  nil,
	}

	_, _ = agent.Execute(context.Background(), req)

	// Default task is "regular" with 16K tokens
	capturedTokens := llmCapture.GetLastMaxTokens()
	if capturedTokens != 16384 {
		t.Errorf("expected default task (regular) with 16384 tokens, got %d", capturedTokens)
	}
}

// Helper types for testing

// capturingLLMProvider is a mock LLM provider that captures the last request.
type capturingLLMProvider struct {
	*llm.MockProvider
	mu              sync.RWMutex
	lastMaxTokens   int
	lastRequestSeen bool
}

func newCapturingLLMProvider() *capturingLLMProvider {
	return &capturingLLMProvider{
		MockProvider: llm.NewMockProvider("capture"),
	}
}

func (p *capturingLLMProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	// Capture the MaxTokens from the request
	p.mu.Lock()
	p.lastMaxTokens = req.MaxTokens
	p.lastRequestSeen = true
	p.mu.Unlock()

	// Return a simple stream that completes immediately
	chunks := make(chan llm.StreamChunk, 2)
	go func() {
		defer close(chunks)
		chunks <- llm.StreamChunk{Type: llm.ChunkTypeContentDelta, Content: "ok"}
		chunks <- llm.StreamChunk{Type: llm.ChunkTypeDone, FinishReason: "stop"}
	}()

	return chunks, nil
}

func (p *capturingLLMProvider) GetLastMaxTokens() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.lastRequestSeen {
		return -1 // Indicate no request was seen
	}
	return p.lastMaxTokens
}

// mockTaskWithBudget is a simple task mock for testing token budgets.
type mockTaskWithBudget struct {
	name      string
	maxTokens int
}

func (m *mockTaskWithBudget) Name() string {
	return m.name
}

func (m *mockTaskWithBudget) SystemPrompt() string {
	return "test prompt"
}

func (m *mockTaskWithBudget) AllowedTools() []string {
	return []string{} // No tools to avoid test complexity
}

func (m *mockTaskWithBudget) MaxTokens() int {
	return m.maxTokens
}

func (m *mockTaskWithBudget) Validate() error {
	return nil
}
