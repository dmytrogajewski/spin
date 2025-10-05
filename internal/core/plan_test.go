package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlan_GetStep tests retrieving a step by ID
func TestPlan_GetStep(t *testing.T) {
	tests := []struct {
		name    string
		plan    *Plan
		stepID  string
		wantErr bool
	}{
		{
			name: "existing step",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Description: "First step"},
					{ID: "step-2", Description: "Second step"},
				},
			},
			stepID:  "step-1",
			wantErr: false,
		},
		{
			name: "non-existent step",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Description: "First step"},
				},
			},
			stepID:  "step-99",
			wantErr: true,
		},
		{
			name:    "empty plan",
			plan:    &Plan{Steps: []Step{}},
			stepID:  "step-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := tt.plan.GetStep(tt.stepID)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, step)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.stepID, step.ID)
			}
		})
	}
}

// TestPlan_UpdateStepStatus tests updating step status
func TestPlan_UpdateStepStatus(t *testing.T) {
	tests := []struct {
		name      string
		plan      *Plan
		stepID    string
		newStatus StepStatus
		wantErr   bool
	}{
		{
			name: "update existing step",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusPending},
				},
			},
			stepID:    "step-1",
			newStatus: StepStatusRunning,
			wantErr:   false,
		},
		{
			name: "update non-existent step",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusPending},
				},
			},
			stepID:    "step-99",
			newStatus: StepStatusCompleted,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.UpdateStepStatus(tt.stepID, tt.newStatus)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				step, _ := tt.plan.GetStep(tt.stepID)
				assert.Equal(t, tt.newStatus, step.Status)
			}
		})
	}
}

// TestPlan_GetReadySteps tests getting steps ready for execution
func TestPlan_GetReadySteps(t *testing.T) {
	tests := []struct {
		name      string
		plan      *Plan
		wantCount int
		wantIDs   []string
	}{
		{
			name: "no dependencies - all pending steps ready",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusPending, DependsOn: []string{}},
					{ID: "step-2", Status: StepStatusPending, DependsOn: []string{}},
				},
			},
			wantCount: 2,
			wantIDs:   []string{"step-1", "step-2"},
		},
		{
			name: "with dependencies - only steps with completed deps ready",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusCompleted, DependsOn: []string{}},
					{ID: "step-2", Status: StepStatusPending, DependsOn: []string{"step-1"}},
					{ID: "step-3", Status: StepStatusPending, DependsOn: []string{"step-2"}},
				},
			},
			wantCount: 1,
			wantIDs:   []string{"step-2"},
		},
		{
			name: "all completed - no ready steps",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusCompleted, DependsOn: []string{}},
					{ID: "step-2", Status: StepStatusCompleted, DependsOn: []string{"step-1"}},
				},
			},
			wantCount: 0,
			wantIDs:   []string{},
		},
		{
			name: "parallel execution - multiple ready steps",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusCompleted, DependsOn: []string{}},
					{ID: "step-2", Status: StepStatusPending, DependsOn: []string{"step-1"}},
					{ID: "step-3", Status: StepStatusPending, DependsOn: []string{"step-1"}},
					{ID: "step-4", Status: StepStatusPending, DependsOn: []string{"step-2"}},
				},
			},
			wantCount: 2,
			wantIDs:   []string{"step-2", "step-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready := tt.plan.GetReadySteps()
			assert.Equal(t, tt.wantCount, len(ready))

			if tt.wantCount > 0 {
				readyIDs := make([]string, len(ready))
				for i, step := range ready {
					readyIDs[i] = step.ID
				}
				assert.ElementsMatch(t, tt.wantIDs, readyIDs)
			}
		})
	}
}

// TestPlan_Progress tests progress calculation
func TestPlan_Progress(t *testing.T) {
	tests := []struct {
		name         string
		plan         *Plan
		wantProgress float64
	}{
		{
			name: "no steps completed",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusPending},
					{ID: "step-2", Status: StepStatusPending},
				},
			},
			wantProgress: 0.0,
		},
		{
			name: "all steps completed",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusCompleted},
					{ID: "step-2", Status: StepStatusCompleted},
				},
			},
			wantProgress: 100.0,
		},
		{
			name: "half completed",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusCompleted},
					{ID: "step-2", Status: StepStatusCompleted},
					{ID: "step-3", Status: StepStatusPending},
					{ID: "step-4", Status: StepStatusPending},
				},
			},
			wantProgress: 50.0,
		},
		{
			name: "mixed statuses",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusCompleted},
					{ID: "step-2", Status: StepStatusRunning},
					{ID: "step-3", Status: StepStatusFailed},
					{ID: "step-4", Status: StepStatusPending},
				},
			},
			wantProgress: 50.0, // Completed + Failed = 2 of 4
		},
		{
			name:         "empty plan",
			plan:         &Plan{Steps: []Step{}},
			wantProgress: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := tt.plan.Progress()
			assert.Equal(t, tt.wantProgress, progress)
		})
	}
}

// TestPlan_IsComplete tests plan completion check
func TestPlan_IsComplete(t *testing.T) {
	tests := []struct {
		name         string
		plan         *Plan
		wantComplete bool
	}{
		{
			name: "all steps completed",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusCompleted},
					{ID: "step-2", Status: StepStatusCompleted},
				},
			},
			wantComplete: true,
		},
		{
			name: "some steps pending",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusCompleted},
					{ID: "step-2", Status: StepStatusPending},
				},
			},
			wantComplete: false,
		},
		{
			name: "some steps failed",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", Status: StepStatusCompleted},
					{ID: "step-2", Status: StepStatusFailed},
				},
			},
			wantComplete: false,
		},
		{
			name:         "empty plan",
			plan:         &Plan{Steps: []Step{}},
			wantComplete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complete := tt.plan.IsComplete()
			assert.Equal(t, tt.wantComplete, complete)
		})
	}
}

// TestPlan_HasCycles tests cycle detection in dependency graph
func TestPlan_HasCycles(t *testing.T) {
	tests := []struct {
		name      string
		plan      *Plan
		hasCycles bool
	}{
		{
			name: "no cycles - linear dependencies",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", DependsOn: []string{}},
					{ID: "step-2", DependsOn: []string{"step-1"}},
					{ID: "step-3", DependsOn: []string{"step-2"}},
				},
			},
			hasCycles: false,
		},
		{
			name: "no cycles - parallel branches",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", DependsOn: []string{}},
					{ID: "step-2", DependsOn: []string{"step-1"}},
					{ID: "step-3", DependsOn: []string{"step-1"}},
					{ID: "step-4", DependsOn: []string{"step-2", "step-3"}},
				},
			},
			hasCycles: false,
		},
		{
			name: "simple cycle - two steps",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", DependsOn: []string{"step-2"}},
					{ID: "step-2", DependsOn: []string{"step-1"}},
				},
			},
			hasCycles: true,
		},
		{
			name: "complex cycle - three steps",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", DependsOn: []string{"step-3"}},
					{ID: "step-2", DependsOn: []string{"step-1"}},
					{ID: "step-3", DependsOn: []string{"step-2"}},
				},
			},
			hasCycles: true,
		},
		{
			name: "self-cycle",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", DependsOn: []string{"step-1"}},
				},
			},
			hasCycles: true,
		},
		{
			name:      "no steps",
			plan:      &Plan{Steps: []Step{}},
			hasCycles: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasCycles := tt.plan.HasCycles()
			assert.Equal(t, tt.hasCycles, hasCycles)
		})
	}
}

// TestPlan_TopologicalSort tests dependency-ordered sorting
func TestPlan_TopologicalSort(t *testing.T) {
	tests := []struct {
		name    string
		plan    *Plan
		wantErr bool
		check   func(*testing.T, []Step)
	}{
		{
			name: "linear dependencies",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-3", DependsOn: []string{"step-2"}},
					{ID: "step-1", DependsOn: []string{}},
					{ID: "step-2", DependsOn: []string{"step-1"}},
				},
			},
			wantErr: false,
			check: func(t *testing.T, sorted []Step) {
				assert.Equal(t, "step-1", sorted[0].ID)
				assert.Equal(t, "step-2", sorted[1].ID)
				assert.Equal(t, "step-3", sorted[2].ID)
			},
		},
		{
			name: "parallel branches converge",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", DependsOn: []string{}},
					{ID: "step-2", DependsOn: []string{"step-1"}},
					{ID: "step-3", DependsOn: []string{"step-1"}},
					{ID: "step-4", DependsOn: []string{"step-2", "step-3"}},
				},
			},
			wantErr: false,
			check: func(t *testing.T, sorted []Step) {
				// step-1 must be first
				assert.Equal(t, "step-1", sorted[0].ID)
				// step-4 must be last
				assert.Equal(t, "step-4", sorted[3].ID)
				// step-2 and step-3 in middle (any order)
				middleIDs := []string{sorted[1].ID, sorted[2].ID}
				assert.ElementsMatch(t, []string{"step-2", "step-3"}, middleIDs)
			},
		},
		{
			name: "no dependencies - any order valid",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", DependsOn: []string{}},
					{ID: "step-2", DependsOn: []string{}},
					{ID: "step-3", DependsOn: []string{}},
				},
			},
			wantErr: false,
			check: func(t *testing.T, sorted []Step) {
				assert.Equal(t, 3, len(sorted))
			},
		},
		{
			name: "circular dependency - should error",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", DependsOn: []string{"step-2"}},
					{ID: "step-2", DependsOn: []string{"step-1"}},
				},
			},
			wantErr: true,
			check:   nil,
		},
		{
			name:    "empty plan",
			plan:    &Plan{Steps: []Step{}},
			wantErr: false,
			check: func(t *testing.T, sorted []Step) {
				assert.Equal(t, 0, len(sorted))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted, err := tt.plan.TopologicalSort()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, sorted)
				}
			}
		})
	}
}

// TestPlan_CalculateEstimatedDuration tests duration calculation
func TestPlan_CalculateEstimatedDuration(t *testing.T) {
	tests := []struct {
		name         string
		plan         *Plan
		wantDuration time.Duration
	}{
		{
			name: "linear dependencies - sum all durations",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", EstimatedDuration: 5 * time.Minute, DependsOn: []string{}},
					{ID: "step-2", EstimatedDuration: 10 * time.Minute, DependsOn: []string{"step-1"}},
					{ID: "step-3", EstimatedDuration: 15 * time.Minute, DependsOn: []string{"step-2"}},
				},
			},
			wantDuration: 30 * time.Minute,
		},
		{
			name: "parallel branches - longest path",
			plan: &Plan{
				Steps: []Step{
					{ID: "step-1", EstimatedDuration: 5 * time.Minute, DependsOn: []string{}},
					{ID: "step-2", EstimatedDuration: 10 * time.Minute, DependsOn: []string{"step-1"}},
					{ID: "step-3", EstimatedDuration: 20 * time.Minute, DependsOn: []string{"step-1"}},
					{ID: "step-4", EstimatedDuration: 5 * time.Minute, DependsOn: []string{"step-2", "step-3"}},
				},
			},
			wantDuration: 30 * time.Minute, // 5 + 20 + 5 (longest path)
		},
		{
			name:         "empty plan",
			plan:         &Plan{Steps: []Step{}},
			wantDuration: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := tt.plan.CalculateEstimatedDuration()
			assert.Equal(t, tt.wantDuration, duration)
		})
	}
}

// TestStepStatus_String tests string representation of step status
func TestStepStatus_String(t *testing.T) {
	tests := []struct {
		status StepStatus
		want   string
	}{
		{StepStatusPending, "pending"},
		{StepStatusReady, "ready"},
		{StepStatusRunning, "running"},
		{StepStatusCompleted, "completed"},
		{StepStatusFailed, "failed"},
		{StepStatusSkipped, "skipped"},
		{StepStatus(999), "unknown"}, // Test unknown status
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

// TestPlanStatus_String tests string representation of plan status
func TestPlanStatus_String(t *testing.T) {
	tests := []struct {
		status PlanStatus
		want   string
	}{
		{PlanStatusPending, "pending"},
		{PlanStatusInProgress, "in_progress"},
		{PlanStatusCompleted, "completed"},
		{PlanStatusFailed, "failed"},
		{PlanStatusCancelled, "cancelled"},
		{PlanStatus(999), "unknown"}, // Test unknown status
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

// TestPlan_ValidateStructure tests plan structure validation
func TestPlan_ValidateStructure(t *testing.T) {
	tests := []struct {
		name    string
		plan    *Plan
		wantErr bool
		errType error
	}{
		{
			name: "valid plan",
			plan: &Plan{
				Task: "Test task",
				Steps: []Step{
					{ID: "step-1", Description: "Step 1", Action: "action1", DependsOn: []string{}},
					{ID: "step-2", Description: "Step 2", Action: "action2", DependsOn: []string{"step-1"}},
				},
			},
			wantErr: false,
		},
		{
			name: "empty task",
			plan: &Plan{
				Task: "",
				Steps: []Step{
					{ID: "step-1", Description: "Step 1", Action: "action1"},
				},
			},
			wantErr: true,
			errType: ErrEmptyTask,
		},
		{
			name: "no steps",
			plan: &Plan{
				Task:  "Test task",
				Steps: []Step{},
			},
			wantErr: true,
		},
		{
			name: "duplicate step IDs",
			plan: &Plan{
				Task: "Test task",
				Steps: []Step{
					{ID: "step-1", Description: "Step 1", Action: "action1"},
					{ID: "step-1", Description: "Step 2", Action: "action2"},
				},
			},
			wantErr: true,
			errType: ErrDuplicateStepID,
		},
		{
			name: "missing step ID",
			plan: &Plan{
				Task: "Test task",
				Steps: []Step{
					{ID: "", Description: "Step 1", Action: "action1"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing step description",
			plan: &Plan{
				Task: "Test task",
				Steps: []Step{
					{ID: "step-1", Description: "", Action: "action1"},
				},
			},
			wantErr: true,
		},
		{
			name: "dependency on non-existent step",
			plan: &Plan{
				Task: "Test task",
				Steps: []Step{
					{ID: "step-1", Description: "Step 1", Action: "action1", DependsOn: []string{"step-99"}},
				},
			},
			wantErr: true,
		},
		{
			name: "circular dependencies",
			plan: &Plan{
				Task: "Test task",
				Steps: []Step{
					{ID: "step-1", Description: "Step 1", Action: "action1", DependsOn: []string{"step-2"}},
					{ID: "step-2", Description: "Step 2", Action: "action2", DependsOn: []string{"step-1"}},
				},
			},
			wantErr: true,
			errType: ErrCircularDeps,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.ValidateStructure()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewPlan(t *testing.T) {
	taskDesc := "Implement new feature"
	plan := NewPlan(taskDesc)

	assert.NotNil(t, plan)
	assert.NotEmpty(t, plan.ID)
	assert.Equal(t, taskDesc, plan.Task)
	assert.Equal(t, PlanStatusPending, plan.Status)
	assert.NotNil(t, plan.Steps)
	assert.Len(t, plan.Steps, 0)
}
