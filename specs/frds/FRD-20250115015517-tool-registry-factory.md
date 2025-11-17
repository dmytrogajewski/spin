# FRD-20250115015517: Extract Tool Registry Factory

## Metadata
- **Status**: COMPLETE
- **Priority**: P0 (CRITICAL)
- **Effort**: S (1 day)
- **Dependencies**: None
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-11-extract-tool-registry-factory)

## Problem Statement

Tool registry is manually constructed in multiple places with identical tool registrations. Currently:

1. **`appserver/processor.go`** (lines 119-127): Manually creates registry and registers 8 tools with hardcoded tool creation calls
2. **`conversation/tools.go`**: Has `buildToolRegistry()` method that uses `NewRegistryWithBuiltins()` then replaces tools with configured versions

**Issues:**
- Duplicate tool registration logic across codebase
- Manual tool registration requires updating multiple places when tools change
- `BuiltinTools` creates tools with nil/empty parameters, requiring replacement in each caller
- No centralized factory for creating fully-configured tool registry

**Impact:**
- Hard to maintain (must update multiple locations)
- Error-prone (easy to miss a registration site)
- Violates DRY principle

## Goals

1. **Create `tools.NewDefaultRegistry()` factory** that creates fully-configured tool registry
2. **Eliminate duplicate registration logic** from `appserver/processor.go`
3. **Standardize tool registry creation** across codebase
4. **Accept `Environment` parameter** for tools that need WorkDir
5. **Maintain backward compatibility** with existing `NewRegistryWithBuiltins()`

## Non-Goals

1. **NOT changing tool interfaces** - tools themselves remain unchanged
2. **NOT removing `NewRegistryWithBuiltins()`** - keep for cases where nil params are acceptable
3. **NOT refactoring conversation package** - that can be done in Feature 1.2 if needed

## Design

### Factory Function

```go
// NewDefaultRegistry creates a new registry with all builtin tools properly configured.
// This factory function accepts Environment parameter for tools that need WorkDir.
//
// Tools registered:
// - read_file, write_file, list_directory (no parameters needed)
// - shell_command (accepts nil parameters, can be configured separately)
// - get_context (requires Environment)
// - apply_patch (requires WorkDir)
// - file_search (requires WorkDir)
// - git_context (requires WorkDir)
//
// This is the recommended factory for most use cases where tools need proper configuration.
func NewDefaultRegistry(env *agent.Environment) *Registry
```

### Implementation Pattern

The factory follows the same pattern as `conversation/buildToolRegistry()`:
1. Create registry using `NewRegistryWithBuiltins()` as base
2. Replace tools that need configuration (WorkDir/Environment) with properly configured versions

**Tools requiring configuration:**
- `get_context`: Requires `Environment` parameter
- `apply_patch`: Requires `WorkDir` parameter
- `file_search`: Requires `WorkDir` parameter  
- `git_context`: Requires `WorkDir` parameter

**Tools that don't need configuration:**
- `read_file`: No parameters needed
- `write_file`: No parameters needed
- `list_directory`: No parameters needed
- `shell_command`: Accepts nil parameters (can be configured separately via RegisterOrReplace)

### Usage Example

```go
// Before (processor.go)
toolRegistry := tools.NewRegistry()
_ = toolRegistry.Register(tools.NewReadFileTool())
_ = toolRegistry.Register(tools.NewWriteFileTool())
_ = toolRegistry.Register(tools.NewListDirectoryTool())
_ = toolRegistry.Register(tools.NewShellCommandTool(nil, nil, nil))
_ = toolRegistry.Register(tools.NewGetContextTool(environment))
_ = toolRegistry.Register(tools.NewApplyPatchTool(environment.WorkDir))
_ = toolRegistry.Register(tools.NewFileSearchTool(environment.WorkDir))
_ = toolRegistry.Register(tools.NewGitContextTool(environment.WorkDir))

// After (using factory)
toolRegistry := tools.NewDefaultRegistry(environment)
```

## API Changes

### New Public API

```go
// NewDefaultRegistry creates a new registry with all builtin tools properly configured.
func NewDefaultRegistry(env *agent.Environment) *Registry
```

### Files to Create

1. `internal/tools/registry_factory.go` - Factory function implementation
2. `internal/tools/registry_factory_test.go` - Unit tests

### Files to Modify

1. `internal/appserver/processor.go` - Use factory instead of manual registration
2. `internal/conversation/tools.go` - Update `buildToolRegistry()` to use factory as base

## Testing Strategy

### Unit Tests (≥90% coverage)

**Factory function tests:**
```go
func TestNewDefaultRegistry(t *testing.T) {
    env := &agent.Environment{WorkDir: "/tmp"}
    registry := tools.NewDefaultRegistry(env)
    
    // Verify all tools are registered
    tools := []string{
        "read_file", "write_file", "list_directory",
        "shell_command", "get_context", "apply_patch",
        "file_search", "git_context",
    }
    for _, toolName := range tools {
        tool, err := registry.Get(toolName)
        require.NoError(t, err)
        assert.NotNil(t, tool)
    }
}

func TestNewDefaultRegistry_ToolsConfigured(t *testing.T) {
    env := &agent.Environment{WorkDir: "/tmp/workdir"}
    registry := tools.NewDefaultRegistry(env)
    
    // Verify tools that need WorkDir are configured correctly
    patchTool, _ := registry.Get("apply_patch")
    // Verify patchTool has correct WorkDir (implementation-specific check)
    
    searchTool, _ := registry.Get("file_search")
    // Verify searchTool has correct WorkDir
    
    gitTool, _ := registry.Get("git_context")
    // Verify gitTool has correct WorkDir
}

func TestNewDefaultRegistry_NilEnvironment(t *testing.T) {
    // Should handle nil gracefully or return error
    // Decision: Accept nil and create tools with empty WorkDir
    registry := tools.NewDefaultRegistry(nil)
    assert.NotNil(t, registry)
    // Verify tools are still registered (with empty WorkDir)
}
```

### Integration Tests

```go
func TestNewDefaultRegistry_IntegrationWithAgent(t *testing.T) {
    env := &agent.Environment{WorkDir: t.TempDir()}
    registry := tools.NewDefaultRegistry(env)
    
    // Create tool runtime with factory-created registry
    toolRuntime := agent.NewToolRuntime(agent.ToolRuntimeConfig{
        Registry: registry,
        // ... other config
    })
    
    // Verify tool runtime works correctly
    // Test tool execution
}
```

### Comparison Tests

```go
func TestNewDefaultRegistry_EquivalentToManual(t *testing.T) {
    env := &agent.Environment{WorkDir: "/tmp"}
    
    // Manual construction (old way)
    manual := tools.NewRegistry()
    _ = manual.Register(tools.NewReadFileTool())
    // ... register all tools manually
    
    // Factory construction (new way)
    factory := tools.NewDefaultRegistry(env)
    
    // Verify both registries have same tools
    assert.Equal(t, manual.List(), factory.List())
}
```

## Acceptance Criteria

### Code Quality
- ✅ `tools.NewDefaultRegistry()` factory function created
- ✅ All built-in tools registered in factory
- ✅ Factory accepts `Environment` parameter
- ✅ Tools requiring WorkDir/Environment are properly configured
- ✅ Factory has ≥90% test coverage
- ✅ `make lint` passes (zero errors)
- ✅ `make deadcode` shows zero dead functions
- ✅ Complexity ≤15 for factory function

### Functional Requirements
- ✅ Factory creates registry with all 8 built-in tools
- ✅ Tools with WorkDir requirements are configured correctly
- ✅ `appserver/processor.go` uses factory (no manual registration)
- ✅ All existing tests pass
- ✅ Integration tests verify factory works with Agent
- ✅ No functional regression

### Documentation
- ✅ Godoc complete for `NewDefaultRegistry()`
- ✅ Factory usage documented
- ✅ Migration guide (if needed) created
- ✅ `docs/packages/tools.md` updated (if it exists)

## Implementation Plan

### Step 1: Create Factory Function
1. Create `internal/tools/registry_factory.go`
2. Implement `NewDefaultRegistry(env *agent.Environment) *Registry`
3. Use `NewRegistryWithBuiltins()` as base
4. Replace tools that need configuration with configured versions
5. Add godoc comments

### Step 2: Write Unit Tests
1. Create `internal/tools/registry_factory_test.go`
2. Test factory creates all tools
3. Test tools are properly configured
4. Test nil environment handling
5. Test equivalence with manual registration
6. Verify ≥90% coverage

### Step 3: Update Processor
1. Update `internal/appserver/processor.go` to use factory
2. Remove manual tool registration code
3. Verify all Processor tests pass

### Step 4: Integration Testing
1. Write integration test with Agent
2. Verify tool execution works correctly
3. Run all existing tests

### Step 5: Code Quality
1. Run `make lint`
2. Run `make deadcode`
3. Verify complexity ≤15
4. Update documentation

## Definition of Done

- [x] FRD created and reviewed
- [x] `tools.NewDefaultRegistry()` factory implemented
- [x] All built-in tools registered in factory
- [x] Factory accepts `workDir string` and `env interface{}` parameters (avoids import cycles)
- [x] Tools requiring configuration are properly configured (get_context, apply_patch, file_search, git_context)
- [x] Factory has 100% test coverage
- [x] `appserver/processor.go` uses factory (eliminates 8 lines of manual registration)
- [x] `internal/conversation/tools.go` uses factory (replaces `NewRegistryWithBuiltins()`)
- [x] Manual tool registration removed from processor
- [x] Conversation's buildToolRegistry now uses shared factory
- [x] All unit tests pass
- [x] All integration tests pass
- [x] `go test -race ./...` passes
- [x] `go vet` passes (zero errors)
- [x] No linter errors found
- [x] Complexity ≤15 for factory function
- [x] Godoc complete for factory function
- [x] Documentation updated
- [x] Roadmap updated with completion status

**Completed**: 2025-01-15

**Result**: Successfully eliminated duplicate tool registry setup. Both `processor.go` and `conversation/tools.go` now use the shared `tools.NewDefaultRegistry()` factory, eliminating hardcoded tool registrations from multiple places as specified in the assessment.

## Related Work

**Part of:**
- Phase 1: Service Layer Consolidation
- Feature 1.1: Extract Tool Registry Factory

**Blocks:**
- Feature 1.2: Unify Processor to Use Builder Pattern (can use factory in Builder)

**Related FRDs:**
- None (first feature in duplication elimination roadmap)

## References

- [Codepath Duplication Assessment](../../docs/codepath-duplication-assessment.md)
- [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md)
- [AGENTS.md](../../../AGENTS.md) - Development guidelines
- [Effective Go](https://go.dev/doc/effective_go) - Go best practices

---

**Created**: 2025-01-15  
**Author**: Spin Agent  
**Version**: 1.0  
**Status**: DRAFT → Ready for implementation

