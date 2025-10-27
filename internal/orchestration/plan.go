package orchestration

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Plan-related errors
var (
	ErrEmptyTask       = errors.New("task description is empty")
	ErrStepNotFound    = errors.New("step not found")
	ErrCircularDeps    = errors.New("plan contains circular dependencies")
	ErrTooManySteps    = errors.New("plan exceeds maximum steps")
	ErrDuplicateStepID = errors.New("duplicate step ID")
)

// StepStatus represents the execution state of a step
type StepStatus int

const (
	StepStatusPending   StepStatus = iota
	StepStatusReady                // Dependencies satisfied, ready to execute
	StepStatusRunning              // Currently executing
	StepStatusCompleted            // Successfully completed
	StepStatusFailed               // Failed during execution
	StepStatusSkipped              // Skipped due to failed dependency
)

// String returns the string representation of StepStatus
func (s StepStatus) String() string {
	switch s {
	case StepStatusPending:
		return "pending"
	case StepStatusReady:
		return "ready"
	case StepStatusRunning:
		return "running"
	case StepStatusCompleted:
		return "completed"
	case StepStatusFailed:
		return "failed"
	case StepStatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// PlanStatus represents the overall state of a plan
type PlanStatus int

const (
	PlanStatusPending PlanStatus = iota
	PlanStatusInProgress
	PlanStatusCompleted
	PlanStatusFailed
	PlanStatusCancelled
)

// String returns the string representation of PlanStatus
func (p PlanStatus) String() string {
	switch p {
	case PlanStatusPending:
		return "pending"
	case PlanStatusInProgress:
		return "in_progress"
	case PlanStatusCompleted:
		return "completed"
	case PlanStatusFailed:
		return "failed"
	case PlanStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// StepResult contains the execution result of a step
type StepResult struct {
	Success bool   // Whether the step succeeded
	Output  string // Output from step execution
	Error   error  // Error if step failed
}

// Step represents a single step in a plan
type Step struct {
	ID                string        // Unique step identifier
	Description       string        // Human-readable description
	Action            string        // Specific action to perform
	DependsOn         []string      // IDs of prerequisite steps
	Status            StepStatus    // Current status
	EstimatedDuration time.Duration // Estimated time to complete
	StartedAt         *time.Time    // When step started (nil if not started)
	CompletedAt       *time.Time    // When step completed (nil if not completed)
	Result            *StepResult   // Execution result (nil if not completed)
}

// Plan represents a task execution plan with steps and dependencies
type Plan struct {
	ID                string              // Unique plan ID
	Task              string              // Original task description
	Steps             []Step              // All steps in the plan
	Dependencies      map[string][]string // Explicit dependency map (optional)
	CreatedAt         time.Time           // When plan was created
	EstimatedDuration time.Duration       // Total estimated time
	Status            PlanStatus          // Overall plan status
	Metadata          json.RawMessage     `json:"metadata,omitempty"` // Additional context
}

// GetStep retrieves a step by ID
func (p *Plan) GetStep(id string) (*Step, error) {
	for i := range p.Steps {
		if p.Steps[i].ID == id {
			return &p.Steps[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrStepNotFound, id)
}

// UpdateStepStatus updates the status of a step
func (p *Plan) UpdateStepStatus(id string, status StepStatus) error {
	for i := range p.Steps {
		if p.Steps[i].ID == id {
			p.Steps[i].Status = status

			now := time.Now()
			if status == StepStatusRunning && p.Steps[i].StartedAt == nil {
				p.Steps[i].StartedAt = &now
			}
			if (status == StepStatusCompleted || status == StepStatusFailed || status == StepStatusSkipped) && p.Steps[i].CompletedAt == nil {
				p.Steps[i].CompletedAt = &now
			}

			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrStepNotFound, id)
}

// GetReadySteps returns steps that are ready for execution
// (all dependencies completed and step is pending)
func (p *Plan) GetReadySteps() []Step {
	var ready []Step

	for _, step := range p.Steps {
		// Skip non-pending steps
		if step.Status != StepStatusPending {
			continue
		}

		// Check if all dependencies are completed
		allDepsComplete := true
		for _, depID := range step.DependsOn {
			dep, err := p.GetStep(depID)
			if err != nil || dep.Status != StepStatusCompleted {
				allDepsComplete = false
				break
			}
		}

		if allDepsComplete {
			ready = append(ready, step)
		}
	}

	return ready
}

// Progress returns the completion percentage (0-100)
func (p *Plan) Progress() float64 {
	if len(p.Steps) == 0 {
		return 0.0
	}

	completed := 0
	for _, step := range p.Steps {
		if step.Status == StepStatusCompleted || step.Status == StepStatusFailed {
			completed++
		}
	}

	return float64(completed) / float64(len(p.Steps)) * 100.0
}

// IsComplete returns true if all steps are completed
func (p *Plan) IsComplete() bool {
	if len(p.Steps) == 0 {
		return true
	}

	for _, step := range p.Steps {
		if step.Status != StepStatusCompleted {
			return false
		}
	}

	return true
}

// HasCycles detects circular dependencies using depth-first search
func (p *Plan) HasCycles() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, step := range p.Steps {
		if p.hasCycleUtil(step.ID, visited, recStack) {
			return true
		}
	}

	return false
}

// hasCycleUtil performs DFS for cycle detection
func (p *Plan) hasCycleUtil(stepID string, visited, recStack map[string]bool) bool {
	// Mark current node as visited and part of recursion stack
	visited[stepID] = true
	recStack[stepID] = true

	// Get dependencies for this step
	step, err := p.GetStep(stepID)
	if err != nil {
		return false
	}

	// Recurse for all dependencies
	for _, depID := range step.DependsOn {
		// If not visited, recurse
		if !visited[depID] {
			if p.hasCycleUtil(depID, visited, recStack) {
				return true
			}
		} else if recStack[depID] {
			// If in recursion stack, we found a cycle
			return true
		}
	}

	// Remove from recursion stack
	recStack[stepID] = false
	return false
}

// TopologicalSort returns steps in dependency order using Kahn's algorithm
func (p *Plan) TopologicalSort() ([]Step, error) {
	if p.HasCycles() {
		return nil, fmt.Errorf("%w: cannot sort plan with circular dependencies", ErrCircularDeps)
	}

	if len(p.Steps) == 0 {
		return []Step{}, nil
	}

	// Calculate in-degree for each step
	inDegree := make(map[string]int)
	for _, step := range p.Steps {
		inDegree[step.ID] = len(step.DependsOn)
	}

	// Queue of steps with no dependencies
	var queue []string
	for _, step := range p.Steps {
		if inDegree[step.ID] == 0 {
			queue = append(queue, step.ID)
		}
	}

	// Process queue
	var sorted []Step
	for len(queue) > 0 {
		// Dequeue
		stepID := queue[0]
		queue = queue[1:]

		// Add to sorted list
		step, _ := p.GetStep(stepID)
		sorted = append(sorted, *step)

		// Reduce in-degree for dependent steps
		for _, s := range p.Steps {
			for _, depID := range s.DependsOn {
				if depID == stepID {
					inDegree[s.ID]--
					if inDegree[s.ID] == 0 {
						queue = append(queue, s.ID)
					}
				}
			}
		}
	}

	return sorted, nil
}

// CalculateEstimatedDuration calculates total estimated time for the plan
// For linear dependencies, it sums all durations.
// For parallel branches, it uses the longest path (critical path).
func (p *Plan) CalculateEstimatedDuration() time.Duration {
	if len(p.Steps) == 0 {
		return 0
	}

	// Build a map of step durations
	durations := make(map[string]time.Duration)
	for _, step := range p.Steps {
		durations[step.ID] = step.EstimatedDuration
	}

	// Calculate longest path to each node
	longestPath := make(map[string]time.Duration)

	// Get topologically sorted steps
	sorted, err := p.TopologicalSort()
	if err != nil {
		// If we can't sort, just sum all durations as fallback
		var total time.Duration
		for _, step := range p.Steps {
			total += step.EstimatedDuration
		}
		return total
	}

	// Calculate longest path for each step
	for _, step := range sorted {
		// Start with this step's duration
		maxDuration := time.Duration(0)

		// Add the maximum duration from all dependencies
		for _, depID := range step.DependsOn {
			if longestPath[depID] > maxDuration {
				maxDuration = longestPath[depID]
			}
		}

		longestPath[step.ID] = maxDuration + durations[step.ID]
	}

	// Find the maximum duration among all final steps
	var maxTotal time.Duration
	for _, duration := range longestPath {
		if duration > maxTotal {
			maxTotal = duration
		}
	}

	return maxTotal
}

// ValidateStructure validates the plan structure
func (p *Plan) ValidateStructure() error {
	// Check task is not empty
	if p.Task == "" {
		return fmt.Errorf("%w", ErrEmptyTask)
	}

	// Check there is at least one step
	if len(p.Steps) == 0 {
		return errors.New("plan must have at least one step")
	}

	// Check for duplicate step IDs
	seen := make(map[string]bool)
	for _, step := range p.Steps {
		if seen[step.ID] {
			return fmt.Errorf("%w: %s", ErrDuplicateStepID, step.ID)
		}
		seen[step.ID] = true
	}

	// Validate each step
	for _, step := range p.Steps {
		// Check step ID is not empty
		if step.ID == "" {
			return errors.New("step ID cannot be empty")
		}

		// Check step description is not empty
		if step.Description == "" {
			return fmt.Errorf("step %s: description cannot be empty", step.ID)
		}

		// Check dependencies exist
		for _, depID := range step.DependsOn {
			if _, err := p.GetStep(depID); err != nil {
				return fmt.Errorf("step %s: dependency %s does not exist", step.ID, depID)
			}
		}
	}

	// Check for circular dependencies
	if p.HasCycles() {
		return fmt.Errorf("%w", ErrCircularDeps)
	}

	return nil
}
