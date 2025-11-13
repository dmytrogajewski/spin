package orchestration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrchestrationService(t *testing.T) {
	toolRegistry := tools.NewRegistry()
	toolExecutor := NewToolExecutor(ToolExecutorConfig{
		Registry: toolRegistry,
	})

	tests := []struct {
		name         string
		toolExecutor *ToolExecutor
		toolRegistry *tools.Registry
		wantNil      bool
	}{
		{
			name:         "with all dependencies",
			toolExecutor: toolExecutor,
			toolRegistry: toolRegistry,
			wantNil:      false,
		},
		{
			name:         "with nil tool executor",
			toolExecutor: nil,
			toolRegistry: toolRegistry,
			wantNil:      false, // Service allows nil
		},
		{
			name:         "with nil tool registry",
			toolExecutor: toolExecutor,
			toolRegistry: nil,
			wantNil:      false, // Service allows nil
		},
		{
			name:         "with all nil",
			toolExecutor: nil,
			toolRegistry: nil,
			wantNil:      false, // Service allows nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewOrchestrationService(tt.toolExecutor, tt.toolRegistry)

			if tt.wantNil {
				assert.Nil(t, svc)
			} else {
				assert.NotNil(t, svc)
			}
		})
	}
}

func TestOrchestrationService_ExecuteTool(t *testing.T) {
	registry := tools.NewRegistryWithBuiltins()

	toolExecutor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
	})

	svc := NewOrchestrationService(toolExecutor, registry)

	tests := []struct {
		name    string
		call    *ToolCall
		wantErr bool
	}{
		{
			name: "valid tool call",
			call: &ToolCall{
				ID: "call-1",
				Function: ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path": "test.txt"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tool name",
			call: &ToolCall{
				ID: "call-2",
				Function: ToolCallFunction{
					Name:      "nonexistent_tool",
					Arguments: `{}`,
				},
			},
			wantErr: false, // Returns result with error, not Go error
		},
		{
			name:    "nil call",
			call:    nil,
			wantErr: false, // Returns result with error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := svc.ExecuteTool(ctx, tt.call)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestOrchestrationService_ExecuteTool_NilExecutor(t *testing.T) {
	svc := NewOrchestrationService(nil, nil)

	call := &ToolCall{
		ID: "call-1",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{}`,
		},
	}

	ctx := context.Background()
	result, err := svc.ExecuteTool(ctx, call)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "tool executor not configured")
}

func TestOrchestrationService_ExecuteBatch(t *testing.T) {
	registry := tools.NewRegistryWithBuiltins()

	toolExecutor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
	})

	svc := NewOrchestrationService(toolExecutor, registry)

	tests := []struct {
		name      string
		calls     []*ToolCall
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty batch",
			calls:     []*ToolCall{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "single call",
			calls: []*ToolCall{
				{
					ID: "call-1",
					Function: ToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path": "test.txt"}`,
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple calls",
			calls: []*ToolCall{
				{
					ID: "call-1",
					Function: ToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path": "test1.txt"}`,
					},
				},
				{
					ID: "call-2",
					Function: ToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path": "test2.txt"}`,
					},
				},
			},
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := svc.ExecuteBatch(ctx, tt.calls)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
		})
	}
}

func TestOrchestrationService_ExecuteBatch_NilExecutor(t *testing.T) {
	svc := NewOrchestrationService(nil, nil)

	calls := []*ToolCall{
		{
			ID: "call-1",
			Function: ToolCallFunction{
				Name:      "read_file",
				Arguments: `{}`,
			},
		},
	}

	ctx := context.Background()
	results, err := svc.ExecuteBatch(ctx, calls)

	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestOrchestrationService_GetPlanner(t *testing.T) {
	svc := NewOrchestrationService(nil, nil)

	planner := svc.GetPlanner()
	assert.Nil(t, planner)
}

func TestOrchestrationService_SetPlanner(t *testing.T) {
	svc := NewOrchestrationService(nil, nil)

	plan := &Plan{
		ID:           "test-plan-1",
		Task:         "test task",
		Steps:        []Step{},
		Dependencies: make(map[string][]string),
		CreatedAt:    time.Now(),
		Status:       PlanStatusPending,
		Metadata:     nil, // json.RawMessage - nil is valid
	}
	svc.SetPlanner(plan)

	retrieved := svc.GetPlanner()
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test task", retrieved.Task)
}

// Benchmark tests
func BenchmarkOrchestrationService_ExecuteTool(b *testing.B) {
	registry := tools.NewRegistryWithBuiltins()

	toolExecutor := NewToolExecutor(ToolExecutorConfig{
		Registry: registry,
	})

	svc := NewOrchestrationService(toolExecutor, registry)

	call := &ToolCall{
		ID: "call-1",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{"path": "test.txt"}`,
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.ExecuteTool(ctx, call)
	}
}

// Metadata JSON marshaling tests

func TestTurn_Metadata_JSON(t *testing.T) {
	tests := []struct {
		name     string
		turn     *Turn
		wantJSON string
	}{
		{
			name: "nil metadata",
			turn: &Turn{
				ID:       "turn-1",
				State:    StatePending,
				Metadata: nil,
			},
			wantJSON: `{"id":"turn-1","session_id":"","user_input":"","ai_response":"","tool_calls":null,"tool_results":null,"state":"pending","started_at":"0001-01-01T00:00:00Z","completed_at":"0001-01-01T00:00:00Z","tokens":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`,
		},
		{
			name: "empty metadata",
			turn: &Turn{
				ID:       "turn-2",
				State:    StateRunning,
				Metadata: json.RawMessage(`{}`),
			},
			wantJSON: `{"id":"turn-2","session_id":"","user_input":"","ai_response":"","tool_calls":null,"tool_results":null,"state":"running","started_at":"0001-01-01T00:00:00Z","completed_at":"0001-01-01T00:00:00Z","tokens":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0},"metadata":{}}`,
		},
		{
			name: "metadata with content",
			turn: &Turn{
				ID:       "turn-3",
				State:    StateCompleted,
				Metadata: json.RawMessage(`{"task_mode":"review","tags":["important"]}`),
			},
			wantJSON: `{"id":"turn-3","session_id":"","user_input":"","ai_response":"","tool_calls":null,"tool_results":null,"state":"completed","started_at":"0001-01-01T00:00:00Z","completed_at":"0001-01-01T00:00:00Z","tokens":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0},"metadata":{"task_mode":"review","tags":["important"]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.turn)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(data))

			// Unmarshal round-trip
			var decoded Turn
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tt.turn.ID, decoded.ID)
			assert.Equal(t, tt.turn.State, decoded.State)

			// Compare metadata
			if tt.turn.Metadata == nil {
				assert.Nil(t, decoded.Metadata)
			} else {
				assert.JSONEq(t, string(tt.turn.Metadata), string(decoded.Metadata))
			}
		})
	}
}

func TestPlan_Metadata_JSON(t *testing.T) {
	tests := []struct {
		name     string
		plan     *Plan
		wantJSON string
	}{
		{
			name: "nil metadata",
			plan: &Plan{
				ID:       "plan-1",
				Task:     "test task",
				Status:   PlanStatusPending,
				Metadata: nil,
			},
			wantJSON: `{"ID":"plan-1","Task":"test task","Steps":null,"Dependencies":null,"CreatedAt":"0001-01-01T00:00:00Z","EstimatedDuration":0,"Status":0}`,
		},
		{
			name: "empty metadata",
			plan: &Plan{
				ID:       "plan-2",
				Task:     "test task",
				Status:   PlanStatusInProgress,
				Metadata: json.RawMessage(`{}`),
			},
			wantJSON: `{"ID":"plan-2","Task":"test task","Steps":null,"Dependencies":null,"CreatedAt":"0001-01-01T00:00:00Z","EstimatedDuration":0,"Status":1,"metadata":{}}`,
		},
		{
			name: "metadata with content",
			plan: &Plan{
				ID:       "plan-3",
				Task:     "complex task",
				Status:   PlanStatusCompleted,
				Metadata: json.RawMessage(`{"priority":"high","estimated_cost":42}`),
			},
			wantJSON: `{"ID":"plan-3","Task":"complex task","Steps":null,"Dependencies":null,"CreatedAt":"0001-01-01T00:00:00Z","EstimatedDuration":0,"Status":2,"metadata":{"priority":"high","estimated_cost":42}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.plan)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(data))

			// Unmarshal round-trip
			var decoded Plan
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tt.plan.ID, decoded.ID)
			assert.Equal(t, tt.plan.Task, decoded.Task)
			assert.Equal(t, tt.plan.Status, decoded.Status)

			// Compare metadata
			if tt.plan.Metadata == nil {
				assert.Nil(t, decoded.Metadata)
			} else {
				assert.JSONEq(t, string(tt.plan.Metadata), string(decoded.Metadata))
			}
		})
	}
}
