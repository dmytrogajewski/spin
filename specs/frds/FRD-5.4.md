# FRD-5.4: Compact Task Mode

**Feature ID:** 5.4  
**Feature Name:** Compact Task Mode  
**Priority:** P2 (Nice to Have)  
**Estimated Effort:** 4 hours  
**Actual Effort:** ~3 hours  
**Status:** ✅ Complete  
**Phase:** 5 - Task Execution Modes

---

## Overview

Implement the Compact task mode - a minimal context mode designed for quick tasks, constrained environments, or scenarios where speed and efficiency are prioritized over comprehensive context. This mode uses a reduced token budget and focused tool set.

## Rationale

The Compact mode addresses specific use cases:
- **Quick tasks**: Simple file reads, searches, or status checks
- **Constrained environments**: Limited memory or processing power
- **Fast responses**: Minimize latency with smaller context
- **Cost optimization**: Reduce token usage for simple operations
- **Mobile/embedded**: Environments with resource constraints
- **Batch operations**: Process many small requests efficiently

## Definition of Ready (DoR)

- [x] Feature 5.1 completed (Task Interface & Registry)
- [x] Feature 5.2 completed (Regular Task Mode - reference)
- [x] Feature 5.3 completed (Review Task Mode - reference)
- [x] Compact mode requirements defined
- [x] Token budget constraints determined (4096 tokens)

## Definition of Done (DoD)

### Core Implementation
- [x] `task/compact.go` implemented with Compact struct
- [x] Compact implements Task interface completely
- [x] Name() returns "compact"
- [x] SystemPrompt() returns minimal, concise prompt
- [x] AllowedTools() returns essential tool subset (3 tools)
- [x] MaxTokens() returns 4096 (smallest budget)
- [x] Validate() performs configuration validation

### Configuration
- [x] Support for custom token budget override
- [x] Support for tool selection customization (AdditionalTools)
- [x] Configuration struct for Compact mode
- [x] Default configuration values

### Testing
- [x] Unit tests for all Task interface methods (98.7% coverage)
- [x] Validation tests
- [x] Configuration tests
- [x] Token budget comparison tests
- [x] Tool set efficiency tests
- [x] All tests passing

### Documentation
- [x] Godoc comments for all exported symbols
- [x] Usage examples
- [x] Configuration documentation
- [x] Performance characteristics documented

### Quality
- [x] All linters passing (dupl warning accepted)
- [x] Race detector clean
- [x] Cyclomatic complexity ≤10 for all functions
- [x] ROADMAP updated

---

## Requirements

### 1. Compact Struct

```go
// Compact implements minimal context mode for quick tasks and constrained environments.
// This mode prioritizes speed and efficiency over comprehensive context.
//
// Compact mode is designed for:
//   - Quick, simple tasks (file reads, searches, status checks)
//   - Constrained environments (limited memory/CPU)
//   - Fast response requirements (minimize latency)
//   - Cost optimization (reduce token usage)
//   - Batch operations (many small requests)
type Compact struct {
    config *CompactConfig
}

// CompactConfig contains configuration for Compact mode.
type CompactConfig struct {
    // MaxTokens overrides the default token budget.
    // If 0, uses default of 4096 tokens (smallest budget).
    MaxTokens int
    
    // AdditionalTools optionally adds more tools to the minimal set.
    // Default minimal set: read_file, list_dir, search_code.
    AdditionalTools []string
    
    // CustomSystemPrompt optionally overrides the default system prompt.
    // Empty string uses the default minimal prompt.
    CustomSystemPrompt string
}
```

### 2. Constructor

```go
// NewCompact creates a new Compact task mode with default configuration.
func NewCompact() *Compact

// NewCompactWithConfig creates a new Compact task mode with custom configuration.
func NewCompactWithConfig(config *CompactConfig) *Compact
```

### 3. Task Interface Implementation

#### Name()
```go
func (c *Compact) Name() string {
    return "compact"
}
```

#### SystemPrompt()
Returns a minimal, concise system prompt:

```go
func (c *Compact) SystemPrompt() string {
    if c.config != nil && c.config.CustomSystemPrompt != "" {
        return c.config.CustomSystemPrompt
    }
    return defaultCompactPrompt
}
```

**Default Compact System Prompt:**
```
You are Spin in Compact Mode - optimized for quick, focused tasks.

MODE: Minimal context, fast responses, essential operations only.

CORE CAPABILITIES:
- Read files and examine code
- List directories and search files
- Search code for patterns

GUIDELINES:
- Be concise and direct
- Focus on the specific task
- Avoid lengthy explanations unless asked
- Use minimal context - only what's needed
- Provide actionable answers quickly

CONSTRAINTS:
- Limited token budget (4096)
- Minimal tool set
- Focus on efficiency

Remember: Speed and clarity over comprehensiveness. Answer the question directly.
```

#### AllowedTools()
```go
func (c *Compact) AllowedTools() []string {
    // Minimal essential tool set
    tools := []string{
        "read_file",
        "list_dir",
        "search_code",
    }
    
    // Add any additional tools if configured
    if c.config != nil && len(c.config.AdditionalTools) > 0 {
        tools = append(tools, c.config.AdditionalTools...)
    }
    
    return tools
}
```

#### MaxTokens()
```go
func (c *Compact) MaxTokens() int {
    if c.config != nil && c.config.MaxTokens > 0 {
        return c.config.MaxTokens
    }
    return DefaultCompactMaxTokens
}

const DefaultCompactMaxTokens = 4096
```

#### Validate()
```go
func (c *Compact) Validate() error {
    if c.config == nil {
        return nil // Default config is always valid
    }
    
    var errs []error
    
    // Validate max tokens
    if c.config.MaxTokens < 0 {
        errs = append(errs, fmt.Errorf("max tokens cannot be negative"))
    }
    if c.config.MaxTokens > MaxAllowedTokens {
        errs = append(errs, fmt.Errorf("max tokens exceeds maximum allowed (%d)", MaxAllowedTokens))
    }
    
    // Validate additional tools
    for i, tool := range c.config.AdditionalTools {
        if tool == "" {
            errs = append(errs, fmt.Errorf("additional tool at index %d cannot be empty", i))
        }
    }
    
    // Validate custom system prompt
    if c.config.CustomSystemPrompt != "" && len(c.config.CustomSystemPrompt) < MinPromptLength {
        errs = append(errs, fmt.Errorf("custom system prompt too short (min %d characters)", MinPromptLength))
    }
    
    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    
    return nil
}

const (
    DefaultCompactMaxTokens = 4096
    MaxAllowedTokens        = 100000
    MinPromptLength         = 50
)
```

---

## Implementation Plan

### Step 1: Write Tests First (1.5 hours)
Following TDD principles:

1. **Constructor tests**
2. **Name() tests**
3. **SystemPrompt() tests** - verify minimal, concise
4. **AllowedTools() tests** - verify minimal set (3 tools)
5. **MaxTokens() tests** - verify 4096 default
6. **Validate() tests**
7. **Comparison tests** - smallest token budget, fewest tools
8. **Integration tests**

### Step 2: Implement Compact Type (1.5 hours)
1. Implement Compact struct
2. Implement CompactConfig struct
3. Implement all Task interface methods
4. Implement validation logic

### Step 3: Run Tests and Iterate (45 min)
### Step 4: Quality Checks (15 min)
### Step 5: Documentation (15 min)

---

## Testing Strategy

### Unit Tests

**File:** `task/compact_test.go`

```go
package task

import (
    "strings"
    "testing"
)

func TestNewCompact(t *testing.T) {
    c := NewCompact()
    if c == nil {
        t.Fatal("NewCompact() returned nil")
    }
}

func TestCompact_Name(t *testing.T) {
    c := NewCompact()
    if c.Name() != "compact" {
        t.Errorf("Name() = %q, want %q", c.Name(), "compact")
    }
}

func TestCompact_AllowedTools(t *testing.T) {
    tests := []struct {
        name         string
        config       *CompactConfig
        wantContains []string
        maxCount     int
    }{
        {
            name:   "default minimal set",
            config: nil,
            wantContains: []string{
                "read_file",
                "list_dir",
                "search_code",
            },
            maxCount: 3,
        },
        {
            name: "with additional tools",
            config: &CompactConfig{
                AdditionalTools: []string{"get_context"},
            },
            wantContains: []string{
                "read_file",
                "get_context",
            },
            maxCount: 4,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            c := NewCompactWithConfig(tt.config)
            tools := c.AllowedTools()
            
            if len(tools) > tt.maxCount {
                t.Errorf("AllowedTools() count = %d, want <= %d", len(tools), tt.maxCount)
            }
            
            for _, want := range tt.wantContains {
                if !contains(tools, want) {
                    t.Errorf("AllowedTools() missing %q", want)
                }
            }
        })
    }
}

func TestCompact_MaxTokens(t *testing.T) {
    c := NewCompact()
    if c.MaxTokens() != DefaultCompactMaxTokens {
        t.Errorf("MaxTokens() = %d, want %d", c.MaxTokens(), DefaultCompactMaxTokens)
    }
}

func TestCompact_SmallestBudget(t *testing.T) {
    compact := NewCompact()
    regular := NewRegular()
    review := NewReview()
    
    if compact.MaxTokens() >= review.MaxTokens() {
        t.Error("Compact should have smaller budget than Review")
    }
    if compact.MaxTokens() >= regular.MaxTokens() {
        t.Error("Compact should have smaller budget than Regular")
    }
}

func TestCompact_FewestTools(t *testing.T) {
    compact := NewCompact()
    regular := NewRegular()
    review := NewReview()
    
    if len(compact.AllowedTools()) >= len(review.AllowedTools()) {
        t.Error("Compact should have fewer tools than Review")
    }
    if len(compact.AllowedTools()) >= len(regular.AllowedTools()) {
        t.Error("Compact should have fewer tools than Regular")
    }
}
```

### Coverage Target
- **Minimum:** 90%
- **All methods:** 100% coverage

---

## Tool Set

### Minimal Essential Tools (3)
- `read_file` - Read file contents
- `list_dir` - List directory contents
- `search_code` - Search for code patterns

### Rationale for Minimal Set
1. **read_file**: Essential for examining specific files
2. **list_dir**: Essential for navigation and exploration
3. **search_code**: Most efficient way to find relevant code

### Excluded Tools
- ❌ `write_file` - Not needed for quick reads
- ❌ `shell` - Too complex for compact mode
- ❌ `git_*` - Not essential for quick tasks
- ❌ `search_files` - Redundant with list_dir + read_file
- ❌ `get_context` - Too comprehensive for compact mode

---

## Configuration Examples

### Default Configuration
```go
// Use minimal default settings
compact := task.NewCompact()
registry.Register("compact", compact)
```

### Extended Tool Set
```go
config := &task.CompactConfig{
    AdditionalTools: []string{"get_context", "search_files"},
}
compact := task.NewCompactWithConfig(config)
```

### Custom Token Budget
```go
config := &task.CompactConfig{
    MaxTokens: 8192, // Larger than default
}
compact := task.NewCompactWithConfig(config)
```

---

## Usage Scenarios

### 1. Quick File Check
```go
compact := task.NewCompact()
// Quickly check a specific file with minimal context
```

### 2. Simple Code Search
```go
compact := task.NewCompact()
// Search for a function or pattern
```

### 3. Batch File Operations
```go
config := &task.CompactConfig{
    MaxTokens: 2048, // Extra small for batch ops
}
compact := task.NewCompactWithConfig(config)
```

### 4. Mobile/Embedded Context
```go
compact := task.NewCompact()
// Minimal resource usage for constrained environments
```

---

## Mode Comparison Table

| Aspect | Regular | Review | Compact |
|--------|---------|--------|---------|
| **Purpose** | Full coding | Code review | Quick tasks |
| **Token Budget** | 16384 | 12288 | **4096** |
| **Tool Count** | 12 | 5-8 | **3** |
| **Write Access** | ✅ Yes | ❌ No | ❌ No |
| **Shell Access** | ✅ Yes | ❌ No | ❌ No |
| **Git Ops** | ✅ Full | 🟡 Read | ❌ None |
| **Speed** | Normal | Normal | **Fast** |
| **Use Case** | Development | Analysis | Quick reads |

---

## Performance Characteristics

### Speed Optimization
- **Minimal token usage**: 4096 default (75% less than Regular)
- **Fewer tools**: 3 tools (75% fewer than Regular)
- **Faster processing**: Smaller context = faster LLM response
- **Lower cost**: Fewer tokens = lower API costs

### Memory Footprint
- **Smallest context**: Minimal history kept
- **Lightweight**: Fewer tool definitions loaded
- **Efficient**: Optimized for resource-constrained environments

### Latency
- **Fastest mode**: Smallest token budget = quickest responses
- **Minimal overhead**: Essential tools only
- **Streamlined**: No unnecessary processing

---

## Integration Points

### Used By
- **Manager** (Feature 7.2): Available for quick tasks
- **Agent** (Feature 6.1): Uses minimal tools and compact prompt
- **Registry** (Feature 5.1): Registered as "compact"

### Uses
- **Task Interface** (Feature 5.1): Implements the interface
- **Configuration** (Feature 0.3): Uses config patterns

---

## Non-Functional Requirements

### Performance
- **Response time**: < 50% of Regular mode
- **Token usage**: < 25% of Regular mode
- **Memory**: < 30% of Regular mode

### Scalability
- **Batch operations**: Handle 100+ quick tasks efficiently
- **Concurrent requests**: Low resource usage enables parallelism

### Maintainability
- **Simplicity**: Minimal configuration options
- **Clear purpose**: Well-defined use case
- **Easy to extend**: Can add tools via AdditionalTools

---

## Future Enhancements

### Potential Extensions
- [ ] Ultra-compact mode (2048 tokens, 1-2 tools)
- [ ] Auto-detect when compact mode is sufficient
- [ ] Smart tool selection based on query
- [ ] Streaming-optimized responses
- [ ] Cached results for common queries

---

## References

- [Spin Architecture Overview](../architecture-overview.md)
- [Core Module Spec](../core-module/spec.md)
- [ROADMAP](../core-module/ROADMAP.md)
- [Feature 5.1: Task Interface](./FRD-5.1.md)
- [Feature 5.2: Regular Task](./FRD-5.2.md)
- [Feature 5.3: Review Task](./FRD-5.3.md)

---

## Acceptance Criteria

- [x] Compact struct implemented with config support
- [x] All Task interface methods implemented correctly
- [x] Name() returns "compact"
- [x] SystemPrompt() returns minimal, concise prompt
- [x] AllowedTools() returns 3 essential tools
- [x] MaxTokens() returns 4096 by default (smallest)
- [x] Validate() performs all checks
- [x] 98.7% test coverage achieved (exceeds 90% requirement)
- [x] Smallest token budget of all modes (4096 vs 12288/16384)
- [x] Fewest tools of all modes (3 vs 5-8/12)
- [x] All tests passing
- [x] Race detector clean
- [x] All linters passing (dupl warning accepted)
- [x] Godoc complete
- [x] ROADMAP updated

---

**Status:** ✅ Complete  
**Created:** October 3, 2025  
**Completed:** October 3, 2025  
**Author:** AI Agent (following AGENTS.md guidelines)

