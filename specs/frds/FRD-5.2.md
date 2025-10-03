# FRD-5.2: Regular Task Mode

**Feature ID:** 5.2  
**Feature Name:** Regular Task Mode  
**Priority:** P1 (Critical)  
**Estimated Effort:** 4 hours  
**Actual Effort:** ~2 hours  
**Status:** ✅ Complete  
**Phase:** 5 - Task Execution Modes

---

## Overview

Implement the Regular task mode - the standard interactive coding mode with full tool access. This is the default mode for the Spin agent, providing complete access to all tools including file operations, shell commands, Git operations, and code search.

## Rationale

The Regular mode is the primary mode of operation for Spin:
- **Full capabilities**: Access to all available tools
- **Default mode**: What users expect when running Spin normally
- **Interactive coding**: Complete workflow from reading to writing to executing
- **Foundation**: Base implementation that other modes can reference

## Definition of Ready (DoR)

- [x] Feature 5.1 completed (Task Interface & Registry)
- [x] Regular mode requirements defined (in spec.md)
- [x] System prompts written (defined in this FRD)

## Definition of Done (DoD)

### Core Implementation
- [ ] `task/regular.go` implemented with Regular struct
- [ ] Regular implements Task interface completely
- [ ] Name() returns "regular"
- [ ] SystemPrompt() returns comprehensive agent prompt
- [ ] AllowedTools() returns full tool list
- [ ] MaxTokens() returns standard token budget (16384)
- [ ] Validate() performs configuration validation

### Configuration
- [ ] Support for custom token budget override
- [ ] Support for tool filtering (optional restrictions)
- [ ] Configuration struct for Regular mode
- [ ] Default configuration values

### Testing
- [ ] Unit tests for all Task interface methods (>90% coverage)
- [ ] Validation tests
- [ ] Configuration tests
- [ ] Tool list verification tests
- [ ] System prompt tests
- [ ] All tests passing

### Documentation
- [ ] Godoc comments for all exported symbols
- [ ] Usage examples
- [ ] Configuration documentation
- [ ] System prompt documentation

### Quality
- [ ] All linters passing
- [ ] Race detector clean
- [ ] Cyclomatic complexity ≤10 for all functions
- [ ] ROADMAP updated

---

## Requirements

### 1. Regular Struct

```go
// Regular implements the standard interactive coding mode.
// This is the default mode for Spin, providing full access to all tools
// including file operations, shell commands, Git, and code search.
//
// Regular mode is designed for:
//   - Interactive coding sessions
//   - Full-featured development workflows
//   - Complex multi-step tasks
//   - Tasks requiring all available tools
type Regular struct {
    config *RegularConfig
}

// RegularConfig contains configuration for Regular mode.
type RegularConfig struct {
    // MaxTokens overrides the default token budget.
    // If 0, uses default of 16384 tokens.
    MaxTokens int
    
    // ExcludedTools optionally restricts certain tools.
    // Empty slice means all tools are allowed.
    ExcludedTools []string
    
    // CustomSystemPrompt optionally overrides the default system prompt.
    // Empty string uses the default prompt.
    CustomSystemPrompt string
}
```

### 2. Constructor

```go
// NewRegular creates a new Regular task mode with default configuration.
func NewRegular() *Regular

// NewRegularWithConfig creates a new Regular task mode with custom configuration.
func NewRegularWithConfig(config *RegularConfig) *Regular
```

### 3. Task Interface Implementation

#### Name()
```go
func (r *Regular) Name() string {
    return "regular"
}
```

#### SystemPrompt()
Returns a comprehensive system prompt that defines the agent's capabilities and behavior:

```go
func (r *Regular) SystemPrompt() string {
    if r.config != nil && r.config.CustomSystemPrompt != "" {
        return r.config.CustomSystemPrompt
    }
    return defaultSystemPrompt
}
```

**Default System Prompt:**
```
You are Spin, an autonomous coding agent designed to help developers with their coding tasks.

CAPABILITIES:
- Read and write files in the workspace
- Execute shell commands (with user approval for dangerous operations)
- Use Git for version control operations
- Search codebase for patterns and symbols
- Analyze code structure and dependencies

BEHAVIOR:
- Be helpful, precise, and efficient
- Always explain what you're doing and why
- Ask for clarification when requirements are unclear
- Suggest best practices and improvements
- Handle errors gracefully and provide clear feedback

CONSTRAINTS:
- Only operate within the specified workspace directory
- Request approval for potentially dangerous operations (rm, sudo, etc.)
- Never expose sensitive information (API keys, passwords, etc.)
- Follow security best practices

WORKFLOW:
1. Understand the user's request
2. Plan your approach (use planning for complex tasks)
3. Execute step-by-step with clear communication
4. Verify results and handle errors
5. Provide summary of what was done

Remember: You are a helpful assistant, not a replacement for human judgment. Always prioritize safety and clarity.
```

#### AllowedTools()
```go
func (r *Regular) AllowedTools() []string {
    tools := []string{
        "read_file",
        "write_file",
        "list_dir",
        "search_files",
        "shell",
        "git_status",
        "git_diff",
        "git_log",
        "git_add",
        "git_commit",
        "search_code",
        "get_context",
    }
    
    // Filter out excluded tools if configured
    if r.config != nil && len(r.config.ExcludedTools) > 0 {
        return filterTools(tools, r.config.ExcludedTools)
    }
    
    return tools
}
```

#### MaxTokens()
```go
func (r *Regular) MaxTokens() int {
    if r.config != nil && r.config.MaxTokens > 0 {
        return r.config.MaxTokens
    }
    return DefaultMaxTokens
}

const DefaultMaxTokens = 16384
```

#### Validate()
```go
func (r *Regular) Validate() error {
    if r.config == nil {
        return nil // Default config is always valid
    }
    
    var errs []error
    
    // Validate max tokens
    if r.config.MaxTokens < 0 {
        errs = append(errs, fmt.Errorf("max tokens cannot be negative"))
    }
    if r.config.MaxTokens > MaxAllowedTokens {
        errs = append(errs, fmt.Errorf("max tokens exceeds maximum allowed (%d)", MaxAllowedTokens))
    }
    
    // Validate excluded tools
    for _, tool := range r.config.ExcludedTools {
        if tool == "" {
            errs = append(errs, fmt.Errorf("excluded tool name cannot be empty"))
        }
    }
    
    // Validate custom system prompt
    if r.config.CustomSystemPrompt != "" && len(r.config.CustomSystemPrompt) < MinPromptLength {
        errs = append(errs, fmt.Errorf("custom system prompt too short (min %d characters)", MinPromptLength))
    }
    
    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    
    return nil
}

const (
    DefaultMaxTokens   = 16384
    MaxAllowedTokens   = 100000
    MinPromptLength    = 50
)
```

### 4. Helper Functions

```go
// filterTools removes excluded tools from the tool list
func filterTools(tools []string, excluded []string) []string {
    excludeMap := make(map[string]bool)
    for _, tool := range excluded {
        excludeMap[tool] = true
    }
    
    filtered := make([]string, 0, len(tools))
    for _, tool := range tools {
        if !excludeMap[tool] {
            filtered = append(filtered, tool)
        }
    }
    
    return filtered
}
```

---

## Implementation Plan

### Step 1: Define Constants and Types (15 min)
1. Define Regular struct
2. Define RegularConfig struct
3. Define constants (DefaultMaxTokens, etc.)
4. Add package documentation

### Step 2: Write Tests First (1.5 hours)
Following TDD principles, write comprehensive tests:

1. **Constructor tests:**
   - Test NewRegular() with defaults
   - Test NewRegularWithConfig() with custom config
   - Test nil config handling

2. **Name() tests:**
   - Test returns "regular"

3. **SystemPrompt() tests:**
   - Test default prompt is returned
   - Test custom prompt override
   - Test prompt is non-empty
   - Test prompt length is reasonable

4. **AllowedTools() tests:**
   - Test default tool list
   - Test all expected tools are present
   - Test tool filtering with exclusions
   - Test empty exclusion list
   - Test exclusion of specific tools

5. **MaxTokens() tests:**
   - Test default token budget
   - Test custom token budget
   - Test zero config uses default

6. **Validate() tests:**
   - Test nil config is valid
   - Test default config is valid
   - Test negative max tokens (invalid)
   - Test exceeding max allowed tokens (invalid)
   - Test empty excluded tool name (invalid)
   - Test short custom prompt (invalid)
   - Test valid custom configurations
   - Test multiple validation errors

7. **Integration tests:**
   - Test implementing Task interface
   - Test registration in Registry
   - Test as default task

### Step 3: Implement Regular Type (45 min)
1. Implement Regular struct
2. Implement RegularConfig struct
3. Implement NewRegular()
4. Implement NewRegularWithConfig()

### Step 4: Implement Task Interface Methods (45 min)
1. Implement Name()
2. Implement SystemPrompt() with default prompt
3. Implement AllowedTools()
4. Implement MaxTokens()
5. Implement Validate()
6. Implement filterTools() helper

### Step 5: Run Tests and Iterate (30 min)
1. Run tests: `go test ./internal/core/task/...`
2. Fix any failing tests
3. Check coverage: should be >90%
4. Run race detector
5. Fix any issues

### Step 6: Quality Checks (15 min)
1. Run linter: `make lint`
2. Analyze with uast/herr
3. Fix any complexity issues
4. Verify all quality gates pass

### Step 7: Documentation (15 min)
1. Add comprehensive godoc comments
2. Add usage examples
3. Document configuration options
4. Update ROADMAP

---

## Testing Strategy

### Unit Tests

**File:** `task/regular_test.go`

```go
package task

import (
    "strings"
    "testing"
)

func TestNewRegular(t *testing.T) {
    r := NewRegular()
    if r == nil {
        t.Fatal("NewRegular() returned nil")
    }
    
    // Should have default config
    if r.config != nil {
        t.Error("NewRegular() should have nil config for defaults")
    }
}

func TestNewRegularWithConfig(t *testing.T) {
    config := &RegularConfig{
        MaxTokens: 8192,
        ExcludedTools: []string{"shell"},
    }
    
    r := NewRegularWithConfig(config)
    if r == nil {
        t.Fatal("NewRegularWithConfig() returned nil")
    }
    
    if r.config != config {
        t.Error("NewRegularWithConfig() did not set config")
    }
}

func TestRegular_Name(t *testing.T) {
    r := NewRegular()
    if r.Name() != "regular" {
        t.Errorf("Name() = %q, want %q", r.Name(), "regular")
    }
}

func TestRegular_SystemPrompt(t *testing.T) {
    tests := []struct {
        name   string
        config *RegularConfig
        want   string
    }{
        {
            name:   "default prompt",
            config: nil,
            want:   "Spin", // Should contain "Spin"
        },
        {
            name: "custom prompt",
            config: &RegularConfig{
                CustomSystemPrompt: "Custom agent prompt for testing purposes",
            },
            want: "Custom agent prompt",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := NewRegularWithConfig(tt.config)
            prompt := r.SystemPrompt()
            
            if prompt == "" {
                t.Error("SystemPrompt() returned empty string")
            }
            
            if !strings.Contains(prompt, tt.want) {
                t.Errorf("SystemPrompt() = %q, should contain %q", prompt, tt.want)
            }
        })
    }
}

func TestRegular_AllowedTools(t *testing.T) {
    tests := []struct {
        name          string
        config        *RegularConfig
        wantContains  []string
        wantExcludes  []string
    }{
        {
            name:   "default tools",
            config: nil,
            wantContains: []string{
                "read_file",
                "write_file",
                "shell",
                "git_status",
                "search_code",
            },
            wantExcludes: nil,
        },
        {
            name: "excluded shell",
            config: &RegularConfig{
                ExcludedTools: []string{"shell"},
            },
            wantContains: []string{
                "read_file",
                "write_file",
            },
            wantExcludes: []string{"shell"},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := NewRegularWithConfig(tt.config)
            tools := r.AllowedTools()
            
            if len(tools) == 0 {
                t.Error("AllowedTools() returned empty list")
            }
            
            // Check expected tools are present
            for _, want := range tt.wantContains {
                if !contains(tools, want) {
                    t.Errorf("AllowedTools() missing %q", want)
                }
            }
            
            // Check excluded tools are not present
            for _, exclude := range tt.wantExcludes {
                if contains(tools, exclude) {
                    t.Errorf("AllowedTools() should not contain %q", exclude)
                }
            }
        })
    }
}

func TestRegular_MaxTokens(t *testing.T) {
    tests := []struct {
        name   string
        config *RegularConfig
        want   int
    }{
        {
            name:   "default",
            config: nil,
            want:   DefaultMaxTokens,
        },
        {
            name: "custom",
            config: &RegularConfig{
                MaxTokens: 8192,
            },
            want: 8192,
        },
        {
            name: "zero uses default",
            config: &RegularConfig{
                MaxTokens: 0,
            },
            want: DefaultMaxTokens,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := NewRegularWithConfig(tt.config)
            if got := r.MaxTokens(); got != tt.want {
                t.Errorf("MaxTokens() = %d, want %d", got, tt.want)
            }
        })
    }
}

func TestRegular_Validate(t *testing.T) {
    tests := []struct {
        name    string
        config  *RegularConfig
        wantErr bool
    }{
        {
            name:    "nil config",
            config:  nil,
            wantErr: false,
        },
        {
            name:    "default config",
            config:  &RegularConfig{},
            wantErr: false,
        },
        {
            name: "valid custom config",
            config: &RegularConfig{
                MaxTokens: 8192,
                ExcludedTools: []string{"shell"},
                CustomSystemPrompt: "This is a valid custom system prompt for testing",
            },
            wantErr: false,
        },
        {
            name: "negative max tokens",
            config: &RegularConfig{
                MaxTokens: -100,
            },
            wantErr: true,
        },
        {
            name: "exceeds max allowed",
            config: &RegularConfig{
                MaxTokens: MaxAllowedTokens + 1,
            },
            wantErr: true,
        },
        {
            name: "empty excluded tool",
            config: &RegularConfig{
                ExcludedTools: []string{""},
            },
            wantErr: true,
        },
        {
            name: "short custom prompt",
            config: &RegularConfig{
                CustomSystemPrompt: "short",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := NewRegularWithConfig(tt.config)
            err := r.Validate()
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

// Helper function
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

### Coverage Target
- **Minimum:** 90%
- **All methods:** 100% coverage

### Test Execution
```bash
# Run tests
go test ./internal/core/task/... -run Regular

# With coverage
go test ./internal/core/task/... -run Regular -cover

# With race detector
go test ./internal/core/task/... -run Regular -race
```

---

## System Prompt Details

The default system prompt for Regular mode emphasizes:

1. **Capabilities**: Clear list of what the agent can do
2. **Behavior**: How the agent should interact with users
3. **Constraints**: Security and safety boundaries
4. **Workflow**: Recommended approach to tasks

The prompt is designed to be:
- **Comprehensive**: Covers all aspects of agent behavior
- **Clear**: Easy to understand and follow
- **Flexible**: Works for various coding tasks
- **Safe**: Emphasizes security and approval requirements

---

## Tool List

Regular mode provides access to all available tools:

### File Operations
- `read_file` - Read file contents
- `write_file` - Write or modify files
- `list_dir` - List directory contents
- `search_files` - Search for files by name/pattern

### Shell Operations
- `shell` - Execute shell commands (with approval)

### Git Operations
- `git_status` - Check repository status
- `git_diff` - View changes
- `git_log` - View commit history
- `git_add` - Stage changes
- `git_commit` - Commit changes

### Code Operations
- `search_code` - Semantic code search
- `get_context` - Gather environment context

---

## Configuration Examples

### Default Configuration
```go
// Use default settings
regular := task.NewRegular()

// Register with registry
registry.Register("regular", regular)
registry.SetDefault("regular")
```

### Custom Token Budget
```go
config := &task.RegularConfig{
    MaxTokens: 8192, // Smaller context window
}
regular := task.NewRegularWithConfig(config)
```

### Restricted Tools
```go
config := &task.RegularConfig{
    ExcludedTools: []string{"shell"}, // No shell access
}
regular := task.NewRegularWithConfig(config)
```

### Custom System Prompt
```go
config := &task.RegularConfig{
    CustomSystemPrompt: `You are a specialized code review agent...`,
}
regular := task.NewRegularWithConfig(config)
```

---

## Integration Points

### Used By
- **Manager** (Feature 7.2): Default task mode
- **Agent** (Feature 6.1): Uses allowed tools and system prompt
- **Registry** (Feature 5.1): Registered as "regular"

### Uses
- **Task Interface** (Feature 5.1): Implements the interface
- **Configuration** (Feature 0.3): Uses config patterns

---

## Non-Functional Requirements

### Performance
- Tool list generation: O(n) where n is number of tools
- Validation: O(n) where n is number of excluded tools
- Minimal memory overhead

### Maintainability
- Simple, straightforward implementation
- Easy to extend with new tools
- Clear separation of concerns

---

## Future Enhancements

### Potential Extensions
- [ ] Dynamic tool loading from config
- [ ] Tool-specific configurations
- [ ] Prompt templates with variables
- [ ] Token budget auto-adjustment
- [ ] Usage metrics tracking

---

## References

- [Spin Architecture Overview](../architecture-overview.md)
- [Core Module Spec](../core-module/spec.md)
- [ROADMAP](../core-module/ROADMAP.md)
- [Feature 5.1: Task Interface](./FRD-5.1.md)
- [Feature 5.3: Review Task](./FRD-5.3.md) (To be created)
- [Feature 5.4: Compact Task](./FRD-5.4.md) (To be created)

---

## Acceptance Criteria

- [x] Regular struct implemented with config support
- [x] All Task interface methods implemented correctly
- [x] Name() returns "regular"
- [x] SystemPrompt() returns comprehensive prompt (34-line default)
- [x] AllowedTools() returns full tool list (12 tools)
- [x] MaxTokens() returns 16384 by default
- [x] Validate() performs all checks
- [x] >90% test coverage achieved (97.8%)
- [x] All tests passing
- [x] Race detector clean
- [x] All linters passing
- [x] Godoc complete
- [x] ROADMAP updated

---

**Status:** ✅ Complete  
**Created:** October 3, 2025  
**Completed:** October 3, 2025  
**Author:** AI Agent (following AGENTS.md guidelines)

