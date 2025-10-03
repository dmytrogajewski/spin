# FRD-5.1: Task Interface & Registry

**Feature ID:** 5.1  
**Feature Name:** Task Interface & Registry  
**Priority:** P1 (Critical)  
**Estimated Effort:** 6 hours  
**Actual Effort:** ~3 hours  
**Status:** ✅ Complete  
**Phase:** 5 - Task Execution Modes

---

## Overview

Implement the Task interface that defines different execution modes for the Spin agent, along with a Registry for managing task mode registration, lookup, and validation. This provides the foundation for features 5.2-5.4 (Regular, Review, and Compact modes).

## Rationale

Different use cases require different agent behaviors:
- **Regular mode**: Full interactive coding with all tools
- **Review mode**: Read-only code review and analysis
- **Compact mode**: Minimal context for quick tasks

The Task interface provides:
- **Consistent API**: All modes implement the same interface
- **Flexibility**: Easy to add new modes without modifying core
- **Validation**: Ensure mode constraints are respected
- **Configuration**: Mode-specific settings and restrictions

## Definition of Ready (DoR)

- [x] Feature 0.3 completed (Configuration System)
- [x] Task modes documented (in spec.md and ROADMAP)
- [x] Tool restrictions per mode defined (in spec.md)
- [x] Phase 0-4 features completed

## Definition of Done (DoD)

### Core Implementation
- [x] `task/task.go` with Task interface fully defined
- [x] Task interface methods:
  - [x] `Name() string` - Returns task mode name
  - [x] `SystemPrompt() string` - Returns mode-specific system prompt
  - [x] `AllowedTools() []string` - Returns allowed tool names
  - [x] `MaxTokens() int` - Returns token budget
  - [x] `Validate() error` - Validates task configuration

### Registry Implementation
- [x] Registry struct for managing task modes
- [x] `NewRegistry() *Registry` constructor
- [x] `Register(name string, task Task) error` method
- [x] `Get(name string) (Task, error)` method
- [x] `List() []string` method - Lists all registered task names
- [x] `Has(name string) bool` method - Checks if task exists
- [x] Thread-safe registry operations (mutex)
- [x] Duplicate registration prevention
- [x] Default task support

### Validation
- [x] Task name validation (non-empty, alphanumeric)
- [x] System prompt validation (non-empty)
- [x] Allowed tools validation (non-empty slice)
- [x] Max tokens validation (positive value)
- [x] Error definitions for registry operations

### Testing
- [x] Unit tests for Task interface contract (96.4% coverage - exceeds 85% target)
- [x] Registry tests:
  - [x] Registration tests (success, duplicates, nil)
  - [x] Get tests (success, not found)
  - [x] List tests
  - [x] Has tests
  - [x] Concurrent registration tests
  - [x] Default task tests
- [x] Validation tests for all Task methods
- [x] Thread-safety tests with race detector
- [x] All tests passing

### Documentation
- [x] Godoc comments for all exported symbols
- [x] Package-level documentation
- [x] Usage examples in godoc
- [x] Error handling documentation
- [x] Thread-safety guarantees documented

### Quality
- [x] All linters passing (`make lint`)
- [x] Code analyzed (no complexity issues)
- [x] Cyclomatic complexity ≤15 for all functions
- [x] Race detector clean (`go test -race`)
- [x] ROADMAP updated to mark 5.1 complete

---

## Requirements

### 1. Task Interface

The Task interface defines the contract for all execution modes:

```go
// Task defines different execution modes for the agent.
// Each mode has specific behavior, tool access, and constraints.
type Task interface {
    // Name returns the unique identifier for this task mode.
    // Names should be lowercase alphanumeric (e.g., "regular", "review", "compact").
    Name() string
    
    // SystemPrompt returns the mode-specific system prompt that defines
    // the agent's behavior and constraints for this execution mode.
    SystemPrompt() string
    
    // AllowedTools returns the list of tool names that are permitted
    // in this execution mode. An empty slice means no tools allowed.
    AllowedTools() []string
    
    // MaxTokens returns the maximum token budget for this mode.
    // This affects context window size and truncation strategy.
    MaxTokens() int
    
    // Validate validates the task configuration and returns an error
    // if the task is misconfigured or invalid.
    Validate() error
}
```

**Design Decisions:**
- **Simple interface**: Keep it minimal to allow easy implementation
- **String-based tool names**: Simple lookup, extensible
- **Validation included**: Each task can enforce its own constraints
- **No configuration in interface**: Implementations hold their config

### 2. Registry Implementation

The Registry manages task mode registration and lookup:

```go
// Registry manages task mode registration and lookup.
// It provides thread-safe operations for registering and
// retrieving task implementations.
type Registry struct {
    tasks   map[string]Task
    mu      sync.RWMutex
    default string
}

// NewRegistry creates a new task registry.
func NewRegistry() *Registry

// Register registers a task mode with the given name.
// Returns an error if the task is nil or the name is already registered.
func (r *Registry) Register(name string, task Task) error

// Get retrieves a task by name.
// Returns an error if the task is not found.
func (r *Registry) Get(name string) (Task, error)

// List returns all registered task names in sorted order.
func (r *Registry) List() []string

// Has returns true if a task with the given name is registered.
func (r *Registry) Has(name string) bool

// SetDefault sets the default task mode name.
// Returns an error if the task is not registered.
func (r *Registry) SetDefault(name string) error

// GetDefault returns the default task.
// Returns an error if no default is set.
func (r *Registry) GetDefault() (Task, error)
```

**Design Decisions:**
- **Thread-safe**: Use RWMutex for concurrent access
- **String keys**: Task names are strings for simplicity
- **Error returns**: Explicit error handling for all operations
- **Sorted list**: Deterministic ordering for testing
- **Default support**: Common pattern for task selection

### 3. Error Definitions

```go
// Common errors for task operations
var (
    // ErrTaskNotFound is returned when a task is not in the registry
    ErrTaskNotFound = errors.New("task not found")
    
    // ErrTaskAlreadyRegistered is returned when registering a duplicate task
    ErrTaskAlreadyRegistered = errors.New("task already registered")
    
    // ErrInvalidTaskName is returned for invalid task names
    ErrInvalidTaskName = errors.New("invalid task name")
    
    // ErrNilTask is returned when attempting to register a nil task
    ErrNilTask = errors.New("task cannot be nil")
    
    // ErrNoDefaultTask is returned when no default task is set
    ErrNoDefaultTask = errors.New("no default task set")
    
    // ErrInvalidTask is returned when task validation fails
    ErrInvalidTask = errors.New("invalid task configuration")
)
```

### 4. Validation Rules

**Task Name Validation:**
- Must not be empty
- Must be lowercase
- Must be alphanumeric (including hyphens and underscores)
- Length: 1-50 characters
- Pattern: `^[a-z0-9_-]+$`

**System Prompt Validation:**
- Must not be empty
- Must be at least 10 characters
- Should be reasonable length (< 50KB)

**Allowed Tools Validation:**
- Can be empty (for no tools)
- Tool names must not be empty strings
- Tool names must be unique in the list

**Max Tokens Validation:**
- Must be positive (> 0)
- Reasonable upper bound (< 1,000,000)
- Typical range: 1,000 - 100,000

---

## Implementation Plan

### Step 1: Define Task Interface (30 min)
1. Define Task interface in `task/task.go`
2. Add godoc comments
3. Add package-level documentation

### Step 2: Define Error Types (15 min)
1. Define all error variables
2. Add error documentation
3. Consider using wrapped errors for context

### Step 3: Implement Registry (1 hour)
1. Define Registry struct
2. Implement NewRegistry()
3. Implement Register() with validation
4. Implement Get()
5. Implement List() with sorting
6. Implement Has()
7. Implement SetDefault() and GetDefault()
8. Add mutex protection

### Step 4: Add Validation (30 min)
1. Implement task name validation
2. Add helper validation functions
3. Implement validation in Register()

### Step 5: Write Tests (2.5 hours)
1. **Interface tests:**
   - Test that nil implements are caught
   - Test interface contract expectations

2. **Registry tests:**
   - Test NewRegistry creates empty registry
   - Test Register success case
   - Test Register duplicate (should error)
   - Test Register nil task (should error)
   - Test Register invalid name (should error)
   - Test Get success case
   - Test Get not found (should error)
   - Test List on empty registry
   - Test List on populated registry
   - Test List ordering
   - Test Has on existing task
   - Test Has on non-existing task
   - Test SetDefault success
   - Test SetDefault non-existent task (should error)
   - Test GetDefault success
   - Test GetDefault when not set (should error)
   - Test concurrent registrations (race detector)
   - Test concurrent reads during writes

3. **Validation tests:**
   - Test valid task names
   - Test invalid task names (empty, uppercase, special chars)
   - Test edge cases (length limits)

### Step 6: Documentation (45 min)
1. Add comprehensive godoc comments
2. Add package-level examples
3. Document thread-safety guarantees
4. Add usage examples

### Step 7: Quality Checks (30 min)
1. Run `make lint`
2. Run `go test -race`
3. Analyze with `uast parse task.go | herr analyze`
4. Fix any issues
5. Verify coverage with `go test -cover`

---

## Testing Strategy

### Unit Tests

**File:** `task/task_test.go`

```go
package task

import (
    "testing"
    "strings"
)

// Mock task for testing
type mockTask struct {
    name         string
    systemPrompt string
    allowedTools []string
    maxTokens    int
    validateErr  error
}

func (m *mockTask) Name() string         { return m.name }
func (m *mockTask) SystemPrompt() string { return m.systemPrompt }
func (m *mockTask) AllowedTools() []string { return m.allowedTools }
func (m *mockTask) MaxTokens() int       { return m.maxTokens }
func (m *mockTask) Validate() error      { return m.validateErr }

func TestRegistry_Register(t *testing.T) {
    tests := []struct {
        name    string
        taskName string
        task    Task
        wantErr bool
        errType error
    }{
        {
            name:     "valid task",
            taskName: "test",
            task: &mockTask{
                name:         "test",
                systemPrompt: "test prompt",
                allowedTools: []string{"tool1"},
                maxTokens:    1000,
            },
            wantErr: false,
        },
        {
            name:     "nil task",
            taskName: "test",
            task:     nil,
            wantErr:  true,
            errType:  ErrNilTask,
        },
        {
            name:     "empty name",
            taskName: "",
            task:     &mockTask{name: "test"},
            wantErr:  true,
            errType:  ErrInvalidTaskName,
        },
        {
            name:     "invalid name uppercase",
            taskName: "TestTask",
            task:     &mockTask{name: "TestTask"},
            wantErr:  true,
            errType:  ErrInvalidTaskName,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := NewRegistry()
            err := r.Register(tt.taskName, tt.task)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if tt.wantErr && tt.errType != nil {
                if !errors.Is(err, tt.errType) {
                    t.Errorf("Register() error = %v, want %v", err, tt.errType)
                }
            }
        })
    }
}

func TestRegistry_Get(t *testing.T) {
    r := NewRegistry()
    task := &mockTask{name: "test"}
    r.Register("test", task)
    
    tests := []struct {
        name     string
        taskName string
        wantTask Task
        wantErr  bool
    }{
        {
            name:     "existing task",
            taskName: "test",
            wantTask: task,
            wantErr:  false,
        },
        {
            name:     "non-existent task",
            taskName: "nonexistent",
            wantTask: nil,
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := r.Get(tt.taskName)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if got != tt.wantTask {
                t.Errorf("Get() = %v, want %v", got, tt.wantTask)
            }
        })
    }
}

func TestRegistry_List(t *testing.T) {
    r := NewRegistry()
    
    // Empty registry
    if list := r.List(); len(list) != 0 {
        t.Errorf("List() on empty registry = %v, want []", list)
    }
    
    // Add tasks in unsorted order
    r.Register("zebra", &mockTask{name: "zebra"})
    r.Register("alpha", &mockTask{name: "alpha"})
    r.Register("beta", &mockTask{name: "beta"})
    
    list := r.List()
    expected := []string{"alpha", "beta", "zebra"}
    
    if !reflect.DeepEqual(list, expected) {
        t.Errorf("List() = %v, want %v (sorted)", list, expected)
    }
}

func TestRegistry_Concurrent(t *testing.T) {
    r := NewRegistry()
    
    // Test concurrent registrations
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            name := fmt.Sprintf("task%d", n)
            r.Register(name, &mockTask{name: name})
        }(i)
    }
    wg.Wait()
    
    // Test concurrent reads
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            name := fmt.Sprintf("task%d", n)
            _, _ = r.Get(name)
            _ = r.Has(name)
        }(i)
    }
    wg.Wait()
    
    list := r.List()
    if len(list) != 10 {
        t.Errorf("List() after concurrent operations = %d tasks, want 10", len(list))
    }
}
```

### Coverage Target
- **Minimum:** 85% overall
- **Critical paths:** 90%+

### Test Execution
```bash
# Run tests
go test ./internal/core/task/...

# With coverage
go test -cover ./internal/core/task/...

# With race detector
go test -race ./internal/core/task/...

# Verbose
go test -v ./internal/core/task/...
```

---

## Usage Examples

### Basic Usage

```go
// Create registry
registry := task.NewRegistry()

// Register tasks (done by core during initialization)
registry.Register("regular", &Regular{config: cfg})
registry.Register("review", &Review{config: cfg})
registry.Register("compact", &Compact{config: cfg})

// Set default
registry.SetDefault("regular")

// Get task by name
task, err := registry.Get("review")
if err != nil {
    log.Fatal(err)
}

// Use task
fmt.Println("Task:", task.Name())
fmt.Println("Max Tokens:", task.MaxTokens())
fmt.Println("Allowed Tools:", task.AllowedTools())

// Get default task
defaultTask, err := registry.GetDefault()

// List all tasks
allTasks := registry.List()  // ["compact", "regular", "review"]

// Check if task exists
if registry.Has("review") {
    fmt.Println("Review mode is available")
}
```

### Implementing a Custom Task

```go
type CustomTask struct {
    cfg *Config
}

func (c *CustomTask) Name() string {
    return "custom"
}

func (c *CustomTask) SystemPrompt() string {
    return "You are a custom AI agent..."
}

func (c *CustomTask) AllowedTools() []string {
    return []string{"read_file", "search"}
}

func (c *CustomTask) MaxTokens() int {
    return 8192
}

func (c *CustomTask) Validate() error {
    if c.cfg == nil {
        return fmt.Errorf("config is required")
    }
    return nil
}

// Register custom task
registry.Register("custom", &CustomTask{cfg: myConfig})
```

---

## Dependencies

### Internal Packages
- `internal/core` - Config type
- Standard library only for implementation

### External Packages
- None (pure Go implementation)

---

## Integration Points

### Used By
- **Agent** (Feature 6.1): Uses task to determine allowed tools and prompts
- **Manager** (Feature 7.2): Uses registry to get task modes
- **Configuration**: Task selection via config

### Uses
- **Config** (Feature 0.3): Task-specific configuration

---

## Non-Functional Requirements

### Performance
- Registry operations should be O(1) for Get/Has
- List operation O(n log n) due to sorting
- Minimal memory overhead

### Thread Safety
- All registry operations are thread-safe
- Use RWMutex for concurrent reads
- Documented in godoc

### Error Handling
- All errors are sentinel errors for easy checking
- Clear error messages
- Proper error wrapping where appropriate

---

## Future Enhancements

### Potential Extensions
- [ ] Task composition (combine tasks)
- [ ] Task inheritance (extend existing tasks)
- [ ] Dynamic task loading (plugins)
- [ ] Task metrics (usage tracking)
- [ ] Task permissions (RBAC)

---

## References

- [Spin Architecture Overview](../architecture-overview.md)
- [Core Module Spec](../core-module/spec.md)
- [ROADMAP](../core-module/ROADMAP.md)
- [Feature 5.2: Regular Task](./FRD-5.2.md) (To be created)
- [Feature 5.3: Review Task](./FRD-5.3.md) (To be created)
- [Feature 5.4: Compact Task](./FRD-5.4.md) (To be created)

---

## Acceptance Criteria

- [x] Task interface fully defined with 5 methods
- [x] Registry implemented with all specified methods
- [x] All validation rules implemented
- [x] >85% test coverage achieved (96.4%)
- [x] Race detector clean
- [x] All linters passing
- [x] Godoc complete and accurate
- [x] Examples in godoc work correctly
- [x] Code analyzed (complexity ≤15)
- [x] ROADMAP updated

---

**Status:** ✅ Complete  
**Created:** October 3, 2025  
**Completed:** October 3, 2025  
**Author:** AI Agent (following AGENTS.md guidelines)

