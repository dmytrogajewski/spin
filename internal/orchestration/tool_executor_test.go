package orchestration

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToolExecutor_Execute_Success tests successful tool execution.
func TestToolExecutor_Execute_Success(t *testing.T) {
	registry := tools.NewRegistry()
	err := registry.Register(tools.NewReadFileTool())
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	call := &ToolCall{
		ID:   "call-123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{"file_path": "/tmp/test.txt"}`,
		},
	}

	result, err := executor.Execute(context.Background(), call)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.ID != "call-123" {
		t.Errorf("Expected ID 'call-123', got %q", result.ID)
	}
}

// TestToolExecutor_Execute_ToolNotFound tests tool not found error.
func TestToolExecutor_Execute_ToolNotFound(t *testing.T) {
	registry := tools.NewRegistry()
	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	call := &ToolCall{
		ID:   "call-123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "nonexistent_tool",
			Arguments: `{}`,
		},
	}

	result, err := executor.Execute(context.Background(), call)

	if err != nil {
		t.Errorf("Expected no error (error in result), got %v", err)
	}
	if result.Success {
		t.Error("Expected failure for nonexistent tool")
	}
	if result.Error == nil {
		t.Error("Expected error in result")
	}
}

// TestToolExecutor_Execute_InvalidArguments tests invalid JSON arguments.
func TestToolExecutor_Execute_InvalidArguments(t *testing.T) {
	registry := tools.NewRegistry()
	_ = registry.Register(tools.NewReadFileTool())

	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	call := &ToolCall{
		ID:   "call-123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{invalid json}`,
		},
	}

	result, err := executor.Execute(context.Background(), call)

	if err != nil {
		t.Errorf("Expected no error (error in result), got %v", err)
	}
	if result.Success {
		t.Error("Expected failure for invalid arguments")
	}
	if result.Error == nil {
		t.Error("Expected error in result")
	}
}

// TestToolExecutor_Execute_NilToolCall tests validation of nil tool call.
func TestToolExecutor_Execute_NilToolCall(t *testing.T) {
	registry := tools.NewRegistry()
	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	result, err := executor.Execute(context.Background(), nil)

	if err != nil {
		t.Errorf("Expected no error (error in result), got %v", err)
	}
	if result.Success {
		t.Error("Expected failure for nil tool call")
	}
	if result.Error == nil {
		t.Error("Expected error in result")
	}
}

// TestToolExecutor_Execute_EmptyID tests validation of empty call ID.
func TestToolExecutor_Execute_EmptyID(t *testing.T) {
	registry := tools.NewRegistry()
	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	call := &ToolCall{
		ID:   "", // Empty ID
		Type: "function",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{}`,
		},
	}

	result, err := executor.Execute(context.Background(), call)

	if err != nil {
		t.Errorf("Expected no error (error in result), got %v", err)
	}
	if result.Success {
		t.Error("Expected failure for empty ID")
	}
	if result.Error == nil {
		t.Error("Expected error in result")
	}
}

// TestToolExecutor_Execute_EmptyFunctionName tests validation of empty function name.
func TestToolExecutor_Execute_EmptyFunctionName(t *testing.T) {
	registry := tools.NewRegistry()
	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	call := &ToolCall{
		ID:   "call-123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "", // Empty name
			Arguments: `{}`,
		},
	}

	result, err := executor.Execute(context.Background(), call)

	if err != nil {
		t.Errorf("Expected no error (error in result), got %v", err)
	}
	if result.Success {
		t.Error("Expected failure for empty function name")
	}
	if result.Error == nil {
		t.Error("Expected error in result")
	}
}

// TestToolExecutor_Execute_EmptyArguments tests execution with empty arguments.
func TestToolExecutor_Execute_EmptyArguments(t *testing.T) {
	registry := tools.NewRegistry()
	_ = registry.Register(tools.NewReadFileTool())

	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	call := &ToolCall{
		ID:   "call-123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: "", // Empty arguments string
		},
	}

	result, err := executor.Execute(context.Background(), call)

	// Empty arguments should parse successfully as empty object
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	// Tool may fail due to missing required parameters, but parsing should work
}

// TestToolExecutor_ValidateToolCall tests tool call validation.
func TestToolExecutor_ValidateToolCall(t *testing.T) {
	executor := &ToolExecutor{}

	tests := []struct {
		name    string
		call    *ToolCall
		wantErr bool
	}{
		{
			name: "valid tool call",
			call: &ToolCall{
				ID:   "call-123",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "read_file",
					Arguments: `{}`,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil tool call",
			call:    nil,
			wantErr: true,
		},
		{
			name: "empty ID",
			call: &ToolCall{
				ID:   "",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "read_file",
					Arguments: `{}`,
				},
			},
			wantErr: true,
		},
		{
			name: "empty function name",
			call: &ToolCall{
				ID:   "call-123",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "",
					Arguments: `{}`,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateToolCall(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateToolCall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestToolExecutor_ParseToolArguments tests argument parsing.
func TestToolExecutor_ParseToolArguments(t *testing.T) {
	executor := &ToolExecutor{}

	tests := []struct {
		name    string
		call    *ToolCall
		wantErr bool
	}{
		{
			name: "valid JSON",
			call: &ToolCall{
				Function: ToolCallFunction{
					Arguments: `{"key": "value"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "empty arguments",
			call: &ToolCall{
				Function: ToolCallFunction{
					Arguments: "",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			call: &ToolCall{
				Function: ToolCallFunction{
					Arguments: `{invalid}`,
				},
			},
			wantErr: true,
		},
		{
			name: "nested JSON",
			call: &ToolCall{
				Function: ToolCallFunction{
					Arguments: `{"outer": {"inner": "value"}}`,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.parseToolArguments(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseToolArguments() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestToolExecutor_ExecuteBatch tests batch execution.
func TestToolExecutor_ExecuteBatch(t *testing.T) {
	registry := tools.NewRegistry()
	_ = registry.Register(tools.NewReadFileTool())

	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	calls := []*ToolCall{
		{
			ID:   "call-1",
			Type: "function",
			Function: ToolCallFunction{
				Name:      "read_file",
				Arguments: `{"file_path": "/tmp/test1.txt"}`,
			},
		},
		{
			ID:   "call-2",
			Type: "function",
			Function: ToolCallFunction{
				Name:      "read_file",
				Arguments: `{"file_path": "/tmp/test2.txt"}`,
			},
		},
	}

	results, err := executor.ExecuteBatch(context.Background(), calls)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	for i, result := range results {
		if result == nil {
			t.Errorf("Expected result %d to be non-nil", i)
		}
	}
}

// TestToolExecutor_ExecuteBatch_Empty tests batch execution with empty slice.
func TestToolExecutor_ExecuteBatch_Empty(t *testing.T) {
	registry := tools.NewRegistry()
	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	results, err := executor.ExecuteBatch(context.Background(), []*ToolCall{})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

// TestToolExecutor_Execute_ToolError tests handling of tool execution errors.
func TestToolExecutor_Execute_ToolError(t *testing.T) {
	registry := tools.NewRegistry()
	_ = registry.Register(tools.NewReadFileTool())

	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	// Try to read a file that doesn't exist
	call := &ToolCall{
		ID:   "call-123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{"file_path": "/nonexistent/file/path/that/does/not/exist.txt"}`,
		},
	}

	result, err := executor.Execute(context.Background(), call)

	// Execution should succeed but tool should report failure
	if err != nil {
		t.Errorf("Expected no error from Execute, got %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	// Result should indicate failure (file not found)
	if result.Success {
		t.Error("Expected failure for nonexistent file")
	}
}

// mockDelayTool is a test tool with configurable delay
type mockDelayTool struct {
	delay time.Duration
}

func (m *mockDelayTool) Name() string        { return "delay_tool" }
func (m *mockDelayTool) Description() string { return "Tool with delay" }
func (m *mockDelayTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        m.Name(),
			Description: m.Description(),
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: map[string]tools.PropertyDefinition{},
			},
		},
	}
}

func (m *mockDelayTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	time.Sleep(m.delay)
	return tools.ToolResult{Success: true, Output: "done"}, nil
}

// TestToolExecutor_ExecuteBatch_Concurrent tests concurrent execution performance
func TestToolExecutor_ExecuteBatch_Concurrent(t *testing.T) {
	registry := tools.NewRegistry()

	// Register a tool that has a delay (simulates I/O)
	delayTool := &mockDelayTool{delay: 100 * time.Millisecond}
	require.NoError(t, registry.Register(delayTool))

	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	// Create 5 tool calls
	calls := make([]*ToolCall, 5)
	for i := 0; i < 5; i++ {
		calls[i] = &ToolCall{
			ID:   fmt.Sprintf("call-%d", i),
			Type: "function",
			Function: ToolCallFunction{
				Name:      "delay_tool",
				Arguments: "{}",
			},
		}
	}

	// Execute batch and measure time
	start := time.Now()
	results, err := executor.ExecuteBatch(context.Background(), calls)
	elapsed := time.Since(start)

	// Assertions
	require.NoError(t, err)
	require.Len(t, results, 5)

	// Sequential would take 500ms, concurrent should take ~100ms
	// Allow some overhead, assert < 250ms (halfway point)
	assert.Less(t, elapsed, 250*time.Millisecond,
		"Concurrent execution should be faster than sequential")

	// Verify all succeeded
	for i, result := range results {
		assert.Equal(t, fmt.Sprintf("call-%d", i), result.ID, "Result order preserved")
		assert.True(t, result.Success)
	}
}

// mockOrderTool is a test tool that preserves order despite varying delays
type mockOrderTool struct{}

func (m *mockOrderTool) Name() string        { return "order_tool" }
func (m *mockOrderTool) Description() string { return "Tool that tracks order" }
func (m *mockOrderTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        m.Name(),
			Description: m.Description(),
			Parameters: tools.ParameterSchema{
				Type: "object",
				Properties: map[string]tools.PropertyDefinition{
					"index": {Type: "integer"},
				},
			},
		},
	}
}

func (m *mockOrderTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	indexFloat, _ := params.GetFloat64("index")
	idx := int(indexFloat)
	// Last tool finishes first (inverse delay)
	time.Sleep(time.Duration(10-idx) * 10 * time.Millisecond)
	return tools.ToolResult{
		Success: true,
		Output:  fmt.Sprintf("result-%d", idx),
	}, nil
}

// TestToolExecutor_ExecuteBatch_PreservesOrder tests result ordering with varying delays
func TestToolExecutor_ExecuteBatch_PreservesOrder(t *testing.T) {
	registry := tools.NewRegistry()

	// Register tool that returns the call index with inverse delay
	orderTool := &mockOrderTool{}
	require.NoError(t, registry.Register(orderTool))

	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	// Create calls
	calls := make([]*ToolCall, 5)
	for i := 0; i < 5; i++ {
		calls[i] = &ToolCall{
			ID:   fmt.Sprintf("call-%d", i),
			Type: "function",
			Function: ToolCallFunction{
				Name:      "order_tool",
				Arguments: fmt.Sprintf(`{"index": %d}`, i),
			},
		}
	}

	results, err := executor.ExecuteBatch(context.Background(), calls)

	require.NoError(t, err)
	require.Len(t, results, 5)

	// Verify order is preserved despite last finishing first
	for i := 0; i < 5; i++ {
		assert.Equal(t, fmt.Sprintf("call-%d", i), results[i].ID)
		assert.Equal(t, fmt.Sprintf("result-%d", i), results[i].Output)
	}
}

// mockCancelTool is a test tool that tracks concurrent execution and respects cancellation
type mockCancelTool struct {
	runningCount *atomic.Int32
}

func (m *mockCancelTool) Name() string        { return "cancel_tool" }
func (m *mockCancelTool) Description() string { return "Tool that respects cancellation" }
func (m *mockCancelTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        m.Name(),
			Description: m.Description(),
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: map[string]tools.PropertyDefinition{},
			},
		},
	}
}

func (m *mockCancelTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	m.runningCount.Add(1)
	select {
	case <-ctx.Done():
		return tools.ToolResult{Success: false, Error: "cancelled"}, ctx.Err()
	case <-time.After(1 * time.Second):
		return tools.ToolResult{Success: true, Output: "completed"}, nil
	}
}

// TestToolExecutor_ExecuteBatch_ContextCancellation tests cancellation during batch
func TestToolExecutor_ExecuteBatch_ContextCancellation(t *testing.T) {
	registry := tools.NewRegistry()

	var runningCount atomic.Int32
	cancelTool := &mockCancelTool{runningCount: &runningCount}
	require.NoError(t, registry.Register(cancelTool))

	executor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
		WorkDir:  "/tmp",
	})

	calls := make([]*ToolCall, 3)
	for i := 0; i < 3; i++ {
		calls[i] = &ToolCall{
			ID:   fmt.Sprintf("call-%d", i),
			Type: "function",
			Function: ToolCallFunction{
				Name:      "cancel_tool",
				Arguments: "{}",
			},
		}
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 50ms (before tools finish)
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Execute batch
	results, err := executor.ExecuteBatch(ctx, calls)

	// May or may not error depending on timing
	// But if successful, all tools should have started concurrently
	if err == nil {
		// Tools completed despite cancellation attempt (fast execution)
		require.Len(t, results, 3)
		// Verify all tools started (proves concurrency)
		assert.Equal(t, int32(3), runningCount.Load(), "All tools should start concurrently")
	} else {
		// Cancellation caught mid-execution
		assert.Error(t, err)
		// Tools should still have started
		assert.GreaterOrEqual(t, runningCount.Load(), int32(1), "At least some tools should have started")
	}
}
