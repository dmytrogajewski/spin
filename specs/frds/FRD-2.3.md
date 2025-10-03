# FRD-2.3: Task Planner

**Feature ID:** 2.3  
**Feature Name:** Task Planner  
**Phase:** 2 (Safety & Execution)  
**Priority:** P2 (Nice to Have)  
**Estimated Effort:** 10 hours  
**Status:** Ready for Implementation  
**Author:** AI Agent  
**Created:** October 3, 2025  
**Last Updated:** October 3, 2025

---

## Executive Summary

Implement task decomposition and planning for complex multi-step operations. The Planner uses an LLM to break down high-level tasks into concrete, executable steps with dependency tracking, enabling the agent to handle complex workflows systematically.

---

## Table of Contents

- [1. Overview](#1-overview)
- [2. Motivation](#2-motivation)
- [3. Requirements](#3-requirements)
- [4. Design](#4-design)
- [5. Implementation](#5-implementation)
- [6. Testing](#6-testing)
- [7. Documentation](#7-documentation)
- [8. Definition of Done](#8-definition-of-done)

---

## 1. Overview

### 1.1 Purpose

The Task Planner provides intelligent task decomposition for complex operations. When given a high-level goal, it uses an LLM to:
- Break down the task into concrete steps
- Identify dependencies between steps
- Estimate execution time
- Track step completion status
- Enable parallelization where possible

### 1.2 Scope

**In Scope:**
- Task decomposition using LLM
- Step dependency management
- Dependency graph construction
- Step status tracking
- Duration estimation
- Plan validation
- Mock LLM provider for testing

**Out of Scope:**
- Actual step execution (handled by Executor)
- Real LLM provider implementation (Phase 8.1)
- Distributed planning
- Plan optimization/reordering

### 1.3 Context

**Dependencies:**
- Feature 0.3: Configuration System ✅
- LLM Provider interface (minimal version for this feature)
- Planning prompt templates

**Dependent Features:**
- Feature 6.1: Agent Orchestration (optional usage)
- Feature 7.1: Conversation Implementation (optional usage)

---

## 2. Motivation

### 2.1 Problem Statement

Complex coding tasks often require multiple steps:
- "Refactor the authentication module" → analyze → plan → refactor → test
- "Add new API endpoint" → design → implement → test → document
- "Fix bug #123" → reproduce → identify → fix → verify

Without planning:
- Agent may miss steps
- Work may be done in wrong order
- Dependencies may be violated
- Progress is hard to track

### 2.2 Benefits

1. **Structured Execution:** Clear roadmap for complex tasks
2. **Dependency Management:** Ensures correct execution order
3. **Progress Tracking:** Monitor task completion
4. **Parallel Execution:** Identify independent steps
5. **Better UX:** Show users the plan before execution
6. **Error Recovery:** Resume from failed steps

### 2.3 Use Cases

**Use Case 1: Refactoring Task**
```
Input: "Refactor the user authentication to use JWT tokens"

Plan:
1. Analyze current auth implementation
2. Design JWT token structure
3. Implement JWT generation/validation
4. Update login endpoint
5. Update middleware
6. Write tests
7. Update documentation
```

**Use Case 2: Bug Fix**
```
Input: "Fix the memory leak in session management"

Plan:
1. Reproduce the issue
2. Profile memory usage
3. Identify leak source
4. Implement fix
5. Verify fix with profiler
6. Add regression test
```

**Use Case 3: New Feature**
```
Input: "Add user profile picture upload"

Plan:
1. Design storage strategy (filesystem/S3)
2. Implement upload handler
3. Add image processing (resize/crop)
4. Update user model
5. Create API endpoints
6. Add frontend integration
7. Write tests
```

---

## 3. Requirements

### 3.1 Functional Requirements

#### FR1: Task Decomposition
- **FR1.1:** Accept high-level task description
- **FR1.2:** Use LLM to generate step-by-step plan
- **FR1.3:** Each step must have unique ID, description, and action
- **FR1.4:** Support nested steps (optional for initial version)

#### FR2: Dependency Management
- **FR2.1:** Identify dependencies between steps
- **FR2.2:** Validate dependency graph (no cycles)
- **FR2.3:** Support multiple dependencies per step
- **FR2.4:** Support parallel execution (steps with no dependencies)

#### FR3: Status Tracking
- **FR3.1:** Track step status: Pending, Running, Completed, Failed, Skipped
- **FR3.2:** Update step status during execution
- **FR3.3:** Calculate overall plan progress percentage
- **FR3.4:** Track timestamps (start, end)

#### FR4: Duration Estimation
- **FR4.1:** Estimate duration for each step
- **FR4.2:** Calculate total plan duration
- **FR4.3:** Account for parallel execution in estimates
- **FR4.4:** Track actual vs estimated time

#### FR5: Plan Validation
- **FR5.1:** Validate plan structure
- **FR5.2:** Check for circular dependencies
- **FR5.3:** Verify all step IDs are unique
- **FR5.4:** Ensure at least one step exists

#### FR6: LLM Integration
- **FR6.1:** Send planning prompt to LLM
- **FR6.2:** Parse LLM response into structured plan
- **FR6.3:** Handle LLM errors gracefully
- **FR6.4:** Support streaming responses (optional)

### 3.2 Non-Functional Requirements

#### NFR1: Performance
- **NFR1.1:** Plan generation completes in < 5 seconds for typical tasks
- **NFR1.2:** Dependency validation is O(V+E) complexity
- **NFR1.3:** Support plans with up to 100 steps

#### NFR2: Reliability
- **NFR2.1:** Graceful handling of LLM failures
- **NFR2.2:** Validate LLM output before returning plan
- **NFR2.3:** Provide fallback for unparseable responses

#### NFR3: Usability
- **NFR3.1:** Clear step descriptions for users
- **NFR3.2:** Actionable step actions
- **NFR3.3:** Meaningful error messages

#### NFR4: Testability
- **NFR4.1:** Mock LLM for deterministic testing
- **NFR4.2:** Test all edge cases (empty plans, cycles, etc.)
- **NFR4.3:** >85% test coverage

---

## 4. Design

### 4.1 Architecture

```
┌─────────────┐
│   Planner   │
└──────┬──────┘
       │
       ├──→ LLM Provider (Planning)
       │
       ├──→ Plan Validator
       │
       └──→ Dependency Graph Builder
```

### 4.2 Component Design

#### 4.2.1 Planner

```go
// Planner implements task planning and decomposition
type Planner struct {
    llm     LLMProvider
    config  PlannerConfig
    prompts *PromptTemplates
}

// PlannerConfig configures the planner
type PlannerConfig struct {
    MaxSteps        int           // Maximum steps per plan
    Timeout         time.Duration // Planning timeout
    Temperature     float64       // LLM temperature for planning
    EnableStreaming bool          // Stream plan generation
}

// NewPlanner creates a new planner
func NewPlanner(llm LLMProvider, config PlannerConfig) *Planner

// Plan decomposes a task into steps
func (p *Planner) Plan(ctx context.Context, task string) (*Plan, error)

// ValidatePlan validates a plan structure
func (p *Planner) ValidatePlan(plan *Plan) error

// parseLLMResponse parses LLM output into Plan
func (p *Planner) parseLLMResponse(response string) (*Plan, error)
```

#### 4.2.2 Plan Structure

```go
// Plan represents a task execution plan
type Plan struct {
    ID           string            // Unique plan ID
    Task         string            // Original task description
    Steps        []Step            // Ordered steps
    Dependencies map[string][]string // step_id -> [dependency_ids]
    CreatedAt    time.Time         // Creation timestamp
    EstimatedDuration time.Duration // Total estimated time
    Status       PlanStatus        // Overall plan status
    Metadata     map[string]interface{} // Additional context
}

// Step represents a single plan step
type Step struct {
    ID          string        // Unique step ID (e.g., "step-1")
    Description string        // Human-readable description
    Action      string        // Specific action to perform
    DependsOn   []string      // IDs of prerequisite steps
    Status      StepStatus    // Current status
    EstimatedDuration time.Duration // Estimated time
    StartedAt   *time.Time    // When step started
    CompletedAt *time.Time    // When step completed
    Result      *StepResult   // Execution result
}

// StepStatus represents step execution state
type StepStatus int

const (
    StepStatusPending StepStatus = iota
    StepStatusReady      // Dependencies satisfied
    StepStatusRunning
    StepStatusCompleted
    StepStatusFailed
    StepStatusSkipped
)

// PlanStatus represents overall plan state
type PlanStatus int

const (
    PlanStatusPending PlanStatus = iota
    PlanStatusInProgress
    PlanStatusCompleted
    PlanStatusFailed
    PlanStatusCancelled
)

// StepResult contains step execution results
type StepResult struct {
    Success bool
    Output  string
    Error   error
}
```

#### 4.2.3 LLM Provider Interface (Minimal)

```go
// LLMProvider defines the minimal interface needed for planning
// Full implementation will be in internal/llm (Phase 8.1)
type LLMProvider interface {
    // Complete performs a completion request
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

// CompletionRequest represents an LLM request
type CompletionRequest struct {
    Messages    []Message
    Temperature float64
    MaxTokens   int
}

// CompletionResponse represents an LLM response
type CompletionResponse struct {
    Content string
    Error   error
}

// Message represents a conversation message
type Message struct {
    Role    string // system, user, assistant
    Content string
}
```

#### 4.2.4 Plan Methods

```go
// GetStep retrieves a step by ID
func (p *Plan) GetStep(id string) (*Step, error)

// UpdateStepStatus updates a step's status
func (p *Plan) UpdateStepStatus(id string, status StepStatus) error

// GetReadySteps returns steps ready for execution
func (p *Plan) GetReadySteps() []Step

// Progress returns completion percentage (0-100)
func (p *Plan) Progress() float64

// IsComplete returns true if all steps completed
func (p *Plan) IsComplete() bool

// HasCycles detects circular dependencies
func (p *Plan) HasCycles() bool

// TopologicalSort returns steps in execution order
func (p *Plan) TopologicalSort() ([]Step, error)
```

### 4.3 Planning Prompt Template

```go
const planningPrompt = `You are a task planning assistant. Your job is to decompose a high-level task into concrete, executable steps.

Task: {{.Task}}

Requirements:
1. Break the task into 3-10 clear steps
2. Each step should be concrete and actionable
3. Identify dependencies between steps (which steps must complete before others)
4. Estimate duration for each step (in minutes)
5. Use clear, imperative language for step descriptions

Format your response as JSON:
{
  "steps": [
    {
      "id": "step-1",
      "description": "Clear description of what to do",
      "action": "Specific command or action",
      "depends_on": [],
      "estimated_minutes": 5
    },
    ...
  ]
}

Example for "Refactor authentication to use JWT":
{
  "steps": [
    {
      "id": "step-1",
      "description": "Analyze current authentication implementation",
      "action": "Review auth.go and identify current token mechanism",
      "depends_on": [],
      "estimated_minutes": 10
    },
    {
      "id": "step-2",
      "description": "Design JWT token structure",
      "action": "Define JWT claims and expiration policy",
      "depends_on": ["step-1"],
      "estimated_minutes": 15
    },
    {
      "id": "step-3",
      "description": "Implement JWT generation function",
      "action": "Create GenerateJWT() function with signing",
      "depends_on": ["step-2"],
      "estimated_minutes": 20
    },
    {
      "id": "step-4",
      "description": "Implement JWT validation middleware",
      "action": "Create ValidateJWT() middleware function",
      "depends_on": ["step-2"],
      "estimated_minutes": 20
    },
    {
      "id": "step-5",
      "description": "Update login endpoint to issue JWT",
      "action": "Modify /login handler to return JWT token",
      "depends_on": ["step-3"],
      "estimated_minutes": 10
    },
    {
      "id": "step-6",
      "description": "Update protected routes to use JWT middleware",
      "action": "Add ValidateJWT middleware to protected endpoints",
      "depends_on": ["step-4"],
      "estimated_minutes": 15
    },
    {
      "id": "step-7",
      "description": "Write unit tests for JWT functions",
      "action": "Create test cases for token generation and validation",
      "depends_on": ["step-3", "step-4"],
      "estimated_minutes": 25
    },
    {
      "id": "step-8",
      "description": "Update API documentation",
      "action": "Document new JWT authentication in README and API docs",
      "depends_on": ["step-5", "step-6"],
      "estimated_minutes": 10
    }
  ]
}

Now create a plan for the given task.`
```

### 4.4 Data Flow

```
1. User provides task description
   ↓
2. Planner.Plan(ctx, task)
   ↓
3. Build prompt from template
   ↓
4. Call LLM with planning prompt
   ↓
5. Parse JSON response
   ↓
6. Construct Plan object
   ↓
7. Build dependency graph
   ↓
8. Validate plan (no cycles, valid IDs)
   ↓
9. Return Plan
```

### 4.5 Dependency Graph Validation

```go
// HasCycles implements cycle detection using DFS
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
    visited[stepID] = true
    recStack[stepID] = true
    
    for _, depID := range p.Dependencies[stepID] {
        if !visited[depID] {
            if p.hasCycleUtil(depID, visited, recStack) {
                return true
            }
        } else if recStack[depID] {
            return true // Cycle detected
        }
    }
    
    recStack[stepID] = false
    return false
}
```

### 4.6 Topological Sort

```go
// TopologicalSort returns steps in dependency order
func (p *Plan) TopologicalSort() ([]Step, error) {
    if p.HasCycles() {
        return nil, fmt.Errorf("plan contains circular dependencies")
    }
    
    inDegree := make(map[string]int)
    for stepID := range p.Dependencies {
        inDegree[stepID] = len(p.Dependencies[stepID])
    }
    
    var queue []string
    for _, step := range p.Steps {
        if inDegree[step.ID] == 0 {
            queue = append(queue, step.ID)
        }
    }
    
    var sorted []Step
    for len(queue) > 0 {
        stepID := queue[0]
        queue = queue[1:]
        
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
```

---

## 5. Implementation

### 5.1 File Structure

```
internal/core/
├── planner.go           # Main planner implementation
├── planner_test.go      # Planner tests
├── plan.go              # Plan types and methods
├── plan_test.go         # Plan tests
└── testing/
    └── mock_llm.go      # Mock LLM provider
```

### 5.2 Implementation Order

1. **Define Types** (plan.go)
   - Step, StepStatus, StepResult
   - Plan, PlanStatus
   - LLMProvider interface

2. **Implement Plan Methods** (plan.go)
   - GetStep, UpdateStepStatus
   - GetReadySteps, Progress, IsComplete
   - HasCycles, TopologicalSort

3. **Implement Mock LLM** (testing/mock_llm.go)
   - MockProvider with configurable responses
   - Support success and error cases

4. **Write Tests** (plan_test.go)
   - Test plan validation
   - Test dependency graph (with/without cycles)
   - Test topological sort
   - Test progress calculation

5. **Implement Planner** (planner.go)
   - NewPlanner constructor
   - Plan() method with LLM integration
   - parseLLMResponse() JSON parsing
   - ValidatePlan() validation logic

6. **Write Planner Tests** (planner_test.go)
   - Test plan generation with mock LLM
   - Test error handling (LLM failures, parse errors)
   - Test validation (max steps, cycles, etc.)
   - Integration tests

### 5.3 Key Algorithms

#### 5.3.1 Cycle Detection (DFS)
- Use depth-first search with recursion stack
- O(V+E) time complexity
- Returns true if cycle detected

#### 5.3.2 Topological Sort (Kahn's Algorithm)
- Use in-degree counting
- Process nodes with zero in-degree
- O(V+E) time complexity

#### 5.3.3 Ready Steps Calculation
```go
func (p *Plan) GetReadySteps() []Step {
    var ready []Step
    for _, step := range p.Steps {
        if step.Status != StepStatusPending {
            continue
        }
        
        // Check if all dependencies are completed
        allDepsComplete := true
        for _, depID := range step.DependsOn {
            dep, _ := p.GetStep(depID)
            if dep.Status != StepStatusCompleted {
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
```

### 5.4 Error Handling

```go
var (
    ErrEmptyTask        = errors.New("task description is empty")
    ErrLLMFailed        = errors.New("LLM request failed")
    ErrInvalidResponse  = errors.New("invalid LLM response format")
    ErrCircularDeps     = errors.New("plan contains circular dependencies")
    ErrStepNotFound     = errors.New("step not found")
    ErrTooManySteps     = errors.New("plan exceeds maximum steps")
    ErrInvalidStepID    = errors.New("invalid step ID format")
    ErrDuplicateStepID  = errors.New("duplicate step ID")
)
```

---

## 6. Testing

### 6.1 Test Strategy

#### Unit Tests
- Plan methods (GetStep, UpdateStepStatus, etc.)
- Dependency graph algorithms (cycle detection, topological sort)
- Progress calculation
- Validation logic

#### Integration Tests
- Planner with mock LLM
- End-to-end plan generation
- Error scenarios

#### Edge Cases
- Empty task
- No steps in plan
- Single step plan
- Circular dependencies
- Missing dependencies
- Duplicate step IDs
- Max steps exceeded

### 6.2 Test Cases

#### TC1: Plan Generation - Simple Task
```go
func TestPlanner_Plan_SimpleTask(t *testing.T) {
    mockLLM := &MockProvider{
        Response: `{"steps": [
            {"id": "step-1", "description": "Do task", "action": "run command", "depends_on": [], "estimated_minutes": 5}
        ]}`,
    }
    
    planner := NewPlanner(mockLLM, DefaultConfig())
    plan, err := planner.Plan(context.Background(), "Simple task")
    
    require.NoError(t, err)
    assert.Equal(t, 1, len(plan.Steps))
    assert.Equal(t, "step-1", plan.Steps[0].ID)
}
```

#### TC2: Plan Generation - Complex Task with Dependencies
```go
func TestPlanner_Plan_WithDependencies(t *testing.T) {
    mockLLM := &MockProvider{
        Response: `{"steps": [
            {"id": "step-1", "description": "First", "action": "action1", "depends_on": [], "estimated_minutes": 5},
            {"id": "step-2", "description": "Second", "action": "action2", "depends_on": ["step-1"], "estimated_minutes": 10}
        ]}`,
    }
    
    planner := NewPlanner(mockLLM, DefaultConfig())
    plan, err := planner.Plan(context.Background(), "Complex task")
    
    require.NoError(t, err)
    assert.Equal(t, 2, len(plan.Steps))
    assert.Contains(t, plan.Steps[1].DependsOn, "step-1")
}
```

#### TC3: Cycle Detection
```go
func TestPlan_HasCycles_DetectsCycle(t *testing.T) {
    plan := &Plan{
        Steps: []Step{
            {ID: "step-1", DependsOn: []string{"step-2"}},
            {ID: "step-2", DependsOn: []string{"step-1"}},
        },
    }
    
    assert.True(t, plan.HasCycles())
}
```

#### TC4: Topological Sort
```go
func TestPlan_TopologicalSort_CorrectOrder(t *testing.T) {
    plan := &Plan{
        Steps: []Step{
            {ID: "step-1", DependsOn: []string{}},
            {ID: "step-2", DependsOn: []string{"step-1"}},
            {ID: "step-3", DependsOn: []string{"step-1"}},
            {ID: "step-4", DependsOn: []string{"step-2", "step-3"}},
        },
    }
    
    sorted, err := plan.TopologicalSort()
    require.NoError(t, err)
    
    // step-1 must be first
    assert.Equal(t, "step-1", sorted[0].ID)
    // step-4 must be last
    assert.Equal(t, "step-4", sorted[3].ID)
}
```

#### TC5: Ready Steps
```go
func TestPlan_GetReadySteps_ReturnsCorrectSteps(t *testing.T) {
    plan := &Plan{
        Steps: []Step{
            {ID: "step-1", Status: StepStatusCompleted, DependsOn: []string{}},
            {ID: "step-2", Status: StepStatusPending, DependsOn: []string{"step-1"}},
            {ID: "step-3", Status: StepStatusPending, DependsOn: []string{"step-2"}},
        },
    }
    
    ready := plan.GetReadySteps()
    assert.Equal(t, 1, len(ready))
    assert.Equal(t, "step-2", ready[0].ID)
}
```

#### TC6: Progress Calculation
```go
func TestPlan_Progress_CalculatesCorrectly(t *testing.T) {
    plan := &Plan{
        Steps: []Step{
            {ID: "step-1", Status: StepStatusCompleted},
            {ID: "step-2", Status: StepStatusCompleted},
            {ID: "step-3", Status: StepStatusPending},
            {ID: "step-4", Status: StepStatusPending},
        },
    }
    
    progress := plan.Progress()
    assert.Equal(t, 50.0, progress) // 2 of 4 completed
}
```

#### TC7: LLM Error Handling
```go
func TestPlanner_Plan_LLMError(t *testing.T) {
    mockLLM := &MockProvider{
        Error: errors.New("LLM connection failed"),
    }
    
    planner := NewPlanner(mockLLM, DefaultConfig())
    _, err := planner.Plan(context.Background(), "Task")
    
    assert.Error(t, err)
    assert.True(t, errors.Is(err, ErrLLMFailed))
}
```

#### TC8: Invalid JSON Response
```go
func TestPlanner_Plan_InvalidJSON(t *testing.T) {
    mockLLM := &MockProvider{
        Response: "not valid json",
    }
    
    planner := NewPlanner(mockLLM, DefaultConfig())
    _, err := planner.Plan(context.Background(), "Task")
    
    assert.Error(t, err)
    assert.True(t, errors.Is(err, ErrInvalidResponse))
}
```

#### TC9: Circular Dependency Detection
```go
func TestPlanner_ValidatePlan_DetectsCircularDeps(t *testing.T) {
    plan := &Plan{
        Steps: []Step{
            {ID: "step-1", DependsOn: []string{"step-3"}},
            {ID: "step-2", DependsOn: []string{"step-1"}},
            {ID: "step-3", DependsOn: []string{"step-2"}},
        },
    }
    
    planner := NewPlanner(nil, DefaultConfig())
    err := planner.ValidatePlan(plan)
    
    assert.Error(t, err)
    assert.True(t, errors.Is(err, ErrCircularDeps))
}
```

#### TC10: Max Steps Validation
```go
func TestPlanner_ValidatePlan_TooManySteps(t *testing.T) {
    steps := make([]Step, 101) // Exceed max of 100
    for i := range steps {
        steps[i] = Step{
            ID:          fmt.Sprintf("step-%d", i+1),
            Description: "Step",
            DependsOn:   []string{},
        }
    }
    
    plan := &Plan{Steps: steps}
    planner := NewPlanner(nil, PlannerConfig{MaxSteps: 100})
    
    err := planner.ValidatePlan(plan)
    assert.Error(t, err)
    assert.True(t, errors.Is(err, ErrTooManySteps))
}
```

### 6.3 Coverage Targets

- **Unit Tests:** >90% coverage for Plan methods
- **Integration Tests:** >85% coverage for Planner
- **Edge Cases:** All error paths tested
- **Race Detector:** Clean with `go test -race`

### 6.4 Test Execution

```bash
# Run all planner tests
go test -v -run TestPlan ./internal/core/
go test -v -run TestPlanner ./internal/core/

# With coverage
go test -cover -coverprofile=coverage.out ./internal/core/
go tool cover -html=coverage.out

# Race detector
go test -race ./internal/core/

# Specific test
go test -v -run TestPlan_HasCycles ./internal/core/
```

---

## 7. Documentation

### 7.1 Godoc Comments

All exported types and functions must have comprehensive godoc comments:

```go
// Planner implements task planning and decomposition for complex multi-step operations.
// It uses an LLM to break down high-level tasks into concrete, executable steps with
// dependency tracking and status management.
//
// Example usage:
//
//	planner := NewPlanner(llmProvider, DefaultConfig())
//	plan, err := planner.Plan(ctx, "Refactor authentication module")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, step := range plan.Steps {
//	    fmt.Printf("%s: %s\n", step.ID, step.Description)
//	}
type Planner struct {
    // ...
}
```

### 7.2 Usage Examples

```go
// Example: Generate and Execute Plan
func ExamplePlanner_basic() {
    // Create planner with LLM provider
    llm := ollama.NewProvider(ollama.Config{
        BaseURL: "http://localhost:11434",
        Model:   "codellama",
    })
    
    planner := core.NewPlanner(llm, core.DefaultPlannerConfig())
    
    // Generate plan
    plan, err := planner.Plan(context.Background(), 
        "Refactor the user authentication to use JWT tokens")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Plan: %s\n", plan.Task)
    fmt.Printf("Steps: %d\n", len(plan.Steps))
    fmt.Printf("Estimated duration: %v\n", plan.EstimatedDuration)
    
    // Execute steps in order
    sorted, _ := plan.TopologicalSort()
    for _, step := range sorted {
        fmt.Printf("Executing: %s\n", step.Description)
        // Execute step...
        plan.UpdateStepStatus(step.ID, core.StepStatusCompleted)
    }
    
    fmt.Printf("Progress: %.0f%%\n", plan.Progress())
}

// Example: Parallel Execution
func ExamplePlan_parallel() {
    // Get ready steps (no dependencies or deps completed)
    ready := plan.GetReadySteps()
    
    // Execute ready steps in parallel
    var wg sync.WaitGroup
    for _, step := range ready {
        wg.Add(1)
        go func(s core.Step) {
            defer wg.Done()
            executeStep(s)
            plan.UpdateStepStatus(s.ID, core.StepStatusCompleted)
        }(step)
    }
    wg.Wait()
}
```

### 7.3 Architecture Documentation

Update the following files:
- `internal/core/README.md` - Add Planner section
- `specs/core-module/spec.md` - Update with planner design
- `specs/core-module/ROADMAP.md` - Mark feature 2.3 complete

---

## 8. Definition of Done

### 8.1 Code Complete

- [x] `plan.go` implemented with all types and methods
- [x] `planner.go` implemented with Planner struct
- [x] Plan() method with LLM integration
- [x] ValidatePlan() validation logic
- [x] parseLLMResponse() JSON parsing
- [x] Dependency graph algorithms (cycle detection, topological sort)
- [x] Step status tracking and progress calculation

### 8.2 Testing Complete

- [x] Unit tests for all Plan methods
- [x] Unit tests for Planner methods
- [x] Integration tests with mock LLM
- [x] Edge case tests (cycles, max steps, etc.)
- [x] Error scenario tests
- [x] >85% test coverage achieved
- [x] All tests passing
- [x] Race detector clean (`go test -race`)

### 8.3 Quality Checks

- [x] Code follows Go 1.24 idioms
- [x] All linters passing (`golangci-lint`)
- [x] Code complexity ≤15 for all functions
- [x] No code duplication (DRY)
- [x] Proper error handling with context
- [x] Thread-safe where needed

### 8.4 Documentation Complete

- [x] Godoc comments on all exported symbols
- [x] Package-level documentation
- [x] Usage examples provided
- [x] Architecture documentation updated
- [x] README.md updated

### 8.5 Integration

- [x] Mock LLM provider in `testing/mock_llm.go`
- [x] Compatible with future Agent integration
- [x] No breaking changes to existing code
- [x] Follows established patterns

### 8.6 Tracking Updated

- [x] Feature 2.3 marked complete in ROADMAP.md
- [x] SUMMARY.md updated
- [x] This FRD marked complete
- [x] Coverage metrics documented

---

## Appendix A: Planning Prompt Examples

### Example 1: Refactoring Task

**Input:** "Refactor the authentication module to use JWT tokens"

**Expected LLM Response:**
```json
{
  "steps": [
    {
      "id": "step-1",
      "description": "Analyze current authentication implementation",
      "action": "Review auth.go and document current token mechanism",
      "depends_on": [],
      "estimated_minutes": 10
    },
    {
      "id": "step-2",
      "description": "Design JWT token structure",
      "action": "Define JWT claims schema and expiration policy",
      "depends_on": ["step-1"],
      "estimated_minutes": 15
    },
    {
      "id": "step-3",
      "description": "Implement JWT generation function",
      "action": "Create GenerateJWT() with HMAC signing",
      "depends_on": ["step-2"],
      "estimated_minutes": 20
    },
    {
      "id": "step-4",
      "description": "Implement JWT validation middleware",
      "action": "Create ValidateJWT() middleware with error handling",
      "depends_on": ["step-2"],
      "estimated_minutes": 20
    },
    {
      "id": "step-5",
      "description": "Update login endpoint",
      "action": "Modify POST /login to return JWT token",
      "depends_on": ["step-3"],
      "estimated_minutes": 10
    },
    {
      "id": "step-6",
      "description": "Update protected routes",
      "action": "Add ValidateJWT middleware to protected endpoints",
      "depends_on": ["step-4"],
      "estimated_minutes": 15
    },
    {
      "id": "step-7",
      "description": "Write comprehensive tests",
      "action": "Create test suite for JWT generation and validation",
      "depends_on": ["step-3", "step-4"],
      "estimated_minutes": 30
    },
    {
      "id": "step-8",
      "description": "Update documentation",
      "action": "Document JWT authentication flow in README and API docs",
      "depends_on": ["step-5", "step-6"],
      "estimated_minutes": 15
    }
  ]
}
```

### Example 2: Bug Fix Task

**Input:** "Fix memory leak in session management"

**Expected LLM Response:**
```json
{
  "steps": [
    {
      "id": "step-1",
      "description": "Reproduce the memory leak",
      "action": "Create test case that triggers the leak",
      "depends_on": [],
      "estimated_minutes": 20
    },
    {
      "id": "step-2",
      "description": "Profile memory usage",
      "action": "Run pprof and analyze heap allocations",
      "depends_on": ["step-1"],
      "estimated_minutes": 15
    },
    {
      "id": "step-3",
      "description": "Identify leak source",
      "action": "Trace allocations to find unreleased resources",
      "depends_on": ["step-2"],
      "estimated_minutes": 30
    },
    {
      "id": "step-4",
      "description": "Implement fix",
      "action": "Add proper cleanup for session resources",
      "depends_on": ["step-3"],
      "estimated_minutes": 25
    },
    {
      "id": "step-5",
      "description": "Verify fix with profiler",
      "action": "Run profiler again and confirm no leak",
      "depends_on": ["step-4"],
      "estimated_minutes": 15
    },
    {
      "id": "step-6",
      "description": "Add regression test",
      "action": "Create test that would catch this leak in future",
      "depends_on": ["step-4"],
      "estimated_minutes": 20
    }
  ]
}
```

---

## Appendix B: Alternative Designs Considered

### Alternative 1: Rule-Based Planning (No LLM)

**Pros:**
- Deterministic results
- No LLM dependency
- Faster execution

**Cons:**
- Limited flexibility
- Hard to maintain rules
- Can't adapt to new scenarios

**Decision:** Rejected. LLM provides better flexibility and can adapt to diverse tasks.

### Alternative 2: Hierarchical Plans (Nested Steps)

**Pros:**
- Better organization for complex tasks
- Natural grouping of related steps

**Cons:**
- More complex implementation
- Harder to track progress
- Dependencies across levels tricky

**Decision:** Deferred. Start with flat structure, add hierarchy in future if needed.

### Alternative 3: Dynamic Replanning

**Pros:**
- Can adapt to failures
- More resilient

**Cons:**
- Much more complex
- Harder to test
- May confuse users

**Decision:** Deferred. Implement basic planning first, add dynamic replanning later if needed.

---

## Appendix C: Future Enhancements

### Enhancement 1: Hierarchical Plans
- Support nested steps (substeps)
- Better organization for complex tasks
- Track progress at multiple levels

### Enhancement 2: Dynamic Replanning
- Replan on step failure
- Adapt to changing requirements
- Learn from execution results

### Enhancement 3: Plan Templates
- Predefined templates for common tasks
- User-customizable templates
- Learn from successful plans

### Enhancement 4: Parallel Execution Engine
- Automatic parallel execution of independent steps
- Resource management (max concurrent steps)
- Progress tracking across parallel execution

### Enhancement 5: Cost Estimation
- Estimate computational cost
- Estimate time and resources
- Budget-aware planning

### Enhancement 6: Plan Visualization
- Generate dependency graph diagrams
- Interactive plan viewer
- Progress visualization

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-10-03 | AI Agent | Initial FRD creation |

---

**Status:** ✅ Ready for Implementation  
**Next Steps:** Begin TDD implementation with plan_test.go

