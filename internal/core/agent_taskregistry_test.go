package core

import (
	"sync"
	"testing"

	"github.com/dmytrogajewski/spin/internal/core/task"
	"github.com/dmytrogajewski/spin/internal/llm"
)

// TestNewAgent_InitializesTaskRegistry verifies that the agent
// initializes a task registry with all 4 built-in modes.
func TestNewAgent_InitializesTaskRegistry(t *testing.T) {
	// Create dependencies
	llmProvider := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	// Create agent
	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("NewAgent() returned nil agent")
	}

	// Verify task registry is not nil
	if agent.taskRegistry == nil {
		t.Error("agent.taskRegistry should not be nil")
	}

	// Verify all 4 modes are registered
	modes := agent.taskRegistry.List()
	expectedModes := []string{"compact", "planning", "regular", "review"}
	if len(modes) != 4 {
		t.Errorf("expected 4 modes, got %d: %v", len(modes), modes)
	}

	// Verify each expected mode exists
	for _, expectedMode := range expectedModes {
		found := false
		for _, mode := range modes {
			if mode == expectedMode {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected mode %q not found in registry", expectedMode)
		}
	}

	// Verify each mode can be retrieved
	t.Run("regular mode exists", func(t *testing.T) {
		regularTask, err := agent.taskRegistry.Get("regular")
		if err != nil {
			t.Errorf("failed to get regular task: %v", err)
		}
		if regularTask == nil {
			t.Error("regular task is nil")
		}
		if regularTask.Name() != "regular" {
			t.Errorf("regular task name = %q, want 'regular'", regularTask.Name())
		}
	})

	t.Run("review mode exists", func(t *testing.T) {
		reviewTask, err := agent.taskRegistry.Get("review")
		if err != nil {
			t.Errorf("failed to get review task: %v", err)
		}
		if reviewTask == nil {
			t.Error("review task is nil")
		}
		if reviewTask.Name() != "review" {
			t.Errorf("review task name = %q, want 'review'", reviewTask.Name())
		}
	})

	t.Run("compact mode exists", func(t *testing.T) {
		compactTask, err := agent.taskRegistry.Get("compact")
		if err != nil {
			t.Errorf("failed to get compact task: %v", err)
		}
		if compactTask == nil {
			t.Error("compact task is nil")
		}
		if compactTask.Name() != "compact" {
			t.Errorf("compact task name = %q, want 'compact'", compactTask.Name())
		}
	})

	t.Run("planning mode exists", func(t *testing.T) {
		planningTask, err := agent.taskRegistry.Get("planning")
		if err != nil {
			t.Errorf("failed to get planning task: %v", err)
		}
		if planningTask == nil {
			t.Error("planning task is nil")
		}
		if planningTask.Name() != "planning" {
			t.Errorf("planning task name = %q, want 'planning'", planningTask.Name())
		}
	})

	// Verify default is "regular"
	defaultTask, err := agent.taskRegistry.GetDefault()
	if err != nil {
		t.Errorf("failed to get default task: %v", err)
	}
	if defaultTask == nil {
		t.Error("default task is nil")
	} else if defaultTask.Name() != "regular" {
		t.Errorf("default task name = %q, want 'regular'", defaultTask.Name())
	}
}

// TestWithTaskRegistry_CustomRegistry verifies that a custom
// task registry can be provided via functional option.
func TestWithTaskRegistry_CustomRegistry(t *testing.T) {
	// Create custom registry with a task
	customRegistry := task.NewRegistry()
	compactTask := task.NewCompact()
	// Register with key "my-compact" but task itself is still "compact"
	err := customRegistry.Register("my-compact", compactTask)
	if err != nil {
		t.Fatalf("failed to register custom task: %v", err)
	}
	err = customRegistry.SetDefault("my-compact")
	if err != nil {
		t.Fatalf("failed to set default: %v", err)
	}

	// Create dependencies
	llmProvider := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	// Create agent with custom registry
	agent, err := NewAgent(
		llmProvider,
		executor,
		validator,
		ctx,
		emitter,
		WithTaskRegistry(customRegistry),
	)
	if err != nil {
		t.Fatalf("NewAgent() unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("NewAgent() returned nil agent")
	}

	// Verify custom registry is used
	if agent.taskRegistry != customRegistry {
		t.Error("agent.taskRegistry should be the custom registry")
	}

	// Verify task exists by registry key
	retrievedTask, err := agent.taskRegistry.Get("my-compact")
	if err != nil {
		t.Errorf("failed to get task by key 'my-compact': %v", err)
	}
	if retrievedTask == nil {
		t.Error("retrieved task is nil")
	}
	// The task's intrinsic Name() is still "compact"
	if retrievedTask != nil && retrievedTask.Name() != "compact" {
		t.Errorf("task.Name() = %q, want 'compact'", retrievedTask.Name())
	}

	// Verify default is the registered task
	defaultTask, err := agent.taskRegistry.GetDefault()
	if err != nil {
		t.Errorf("failed to get default task: %v", err)
	}
	if defaultTask == nil {
		t.Error("default task is nil")
	}
	// The default task should be the same instance
	if defaultTask != compactTask {
		t.Error("default task should be the same instance as registered task")
	}
}

// TestWithTaskRegistry_RejectsNil verifies that passing nil
// to WithTaskRegistry returns an error.
func TestWithTaskRegistry_RejectsNil(t *testing.T) {
	// Create dependencies
	llmProvider := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	// Attempt to create agent with nil registry
	agent, err := NewAgent(
		llmProvider,
		executor,
		validator,
		ctx,
		emitter,
		WithTaskRegistry(nil),
	)

	// Verify error is returned
	if err == nil {
		t.Error("NewAgent() expected error for nil task registry, got nil")
	}
	if agent != nil {
		t.Error("NewAgent() expected nil agent for error case, got non-nil")
	}

	// Verify error message is descriptive
	if err != nil {
		errMsg := err.Error()
		if errMsg == "" {
			t.Error("error message is empty")
		}
		// Error message should mention "task registry" and "nil"
		expectedSubstrings := []string{"task registry", "nil"}
		for _, substr := range expectedSubstrings {
			if !contains(errMsg, substr) {
				t.Errorf("error message %q should contain %q", errMsg, substr)
			}
		}
	}
}

// TestAgent_TaskRegistry_ThreadSafe verifies that concurrent
// access to the task registry is thread-safe.
func TestAgent_TaskRegistry_ThreadSafe(t *testing.T) {
	// Create agent
	agent, err := newTestAgent(t)
	if err != nil {
		t.Fatalf("failed to create test agent: %v", err)
	}

	// Verify registry exists
	if agent.taskRegistry == nil {
		t.Fatal("agent.taskRegistry is nil")
	}

	// Perform concurrent reads
	const numGoroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Read all modes
			modes := agent.taskRegistry.List()
			if len(modes) == 0 {
				errors <- &testError{msg: "List() returned empty slice"}
				return
			}

			// Get default task
			defaultTask, err := agent.taskRegistry.GetDefault()
			if err != nil {
				errors <- err
				return
			}
			if defaultTask == nil {
				errors <- &testError{msg: "GetDefault() returned nil task"}
				return
			}

			// Get each registered task
			for _, modeName := range modes {
				task, err := agent.taskRegistry.Get(modeName)
				if err != nil {
					errors <- err
					return
				}
				if task == nil {
					errors <- &testError{msg: "Get() returned nil task for " + modeName}
					return
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent access error: %v", err)
	}
}

// TestAgent_GetTaskRegistry verifies the GetTaskRegistry helper method.
func TestAgent_GetTaskRegistry(t *testing.T) {
	agent, err := newTestAgent(t)
	if err != nil {
		t.Fatalf("failed to create test agent: %v", err)
	}

	registry := agent.GetTaskRegistry()
	if registry == nil {
		t.Error("GetTaskRegistry() returned nil")
	}
	if registry != agent.taskRegistry {
		t.Error("GetTaskRegistry() should return the internal task registry")
	}
}

// TestAgent_ListTaskModes verifies the ListTaskModes helper method.
func TestAgent_ListTaskModes(t *testing.T) {
	agent, err := newTestAgent(t)
	if err != nil {
		t.Fatalf("failed to create test agent: %v", err)
	}

	modes := agent.ListTaskModes()
	if len(modes) != 4 {
		t.Errorf("ListTaskModes() returned %d modes, want 4", len(modes))
	}

	// Verify modes are sorted
	expectedModes := []string{"compact", "planning", "regular", "review"}
	for i, expected := range expectedModes {
		if i >= len(modes) {
			t.Errorf("missing mode at index %d: want %q", i, expected)
			continue
		}
		if modes[i] != expected {
			t.Errorf("mode at index %d = %q, want %q", i, modes[i], expected)
		}
	}
}

// TestAgent_ListTaskModes_NilRegistry verifies that ListTaskModes
// handles nil registry gracefully (defensive programming).
func TestAgent_ListTaskModes_NilRegistry(t *testing.T) {
	agent, err := newTestAgent(t)
	if err != nil {
		t.Fatalf("failed to create test agent: %v", err)
	}

	// Simulate nil registry (should not happen in practice)
	agent.taskRegistry = nil

	modes := agent.ListTaskModes()
	if modes != nil {
		t.Errorf("ListTaskModes() with nil registry should return nil, got %v", modes)
	}
}

// TestWithTaskRegistry_OptionsOrdering verifies that WithTaskRegistry
// option correctly overrides the default registry when applied.
func TestWithTaskRegistry_OptionsOrdering(t *testing.T) {
	// Create custom registry
	customRegistry := task.NewRegistry()
	customTask := task.NewRegular()
	err := customRegistry.Register("only-regular", customTask)
	if err != nil {
		t.Fatalf("failed to register task: %v", err)
	}

	// Create dependencies
	llmProvider := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	// Create agent with custom registry
	agent, err := NewAgent(
		llmProvider,
		executor,
		validator,
		ctx,
		emitter,
		WithTaskRegistry(customRegistry),
	)
	if err != nil {
		t.Fatalf("NewAgent() unexpected error: %v", err)
	}

	// Verify custom registry was used
	modes := agent.taskRegistry.List()
	if len(modes) != 1 {
		t.Errorf("expected 1 mode in custom registry, got %d", len(modes))
	}
	if len(modes) > 0 && modes[0] != "only-regular" {
		t.Errorf("expected mode 'only-regular', got %q", modes[0])
	}

	// Verify default registry was replaced (not just appended)
	_, err = agent.taskRegistry.Get("review")
	if err == nil {
		t.Error("'review' mode should not exist in custom registry")
	}
}

// Helper functions

// newTestAgent creates a test agent with default configuration.
func newTestAgent(t *testing.T) (*Agent, error) {
	t.Helper()

	llmProvider := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		return nil, err
	}
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	return NewAgent(llmProvider, executor, validator, ctx, emitter)
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
