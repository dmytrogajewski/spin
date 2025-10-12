# FRD-20251012141500: Manager Task Support

**Status**: Ready for Implementation
**Created**: 2025-10-12 14:15:00
**Priority**: MEDIUM
**Complexity**: Low (1 hour)
**Phase**: P2.2 - Conversation Integration
**Depends On**: P2.1 (Task Mode to Conversation) ✅ COMPLETE
**Roadmap**: [specs/task-modes/ROADMAP.md](../task-modes/ROADMAP.md)

## Overview

Add manager-level support for creating conversations with specific task modes. This allows the Manager to pass task registry to Agent during conversation creation and provides a convenience method to create conversations that start in a specific mode.

## Background

**Phase 1 (COMPLETE)**: Agent has task registry with 4 built-in modes
**Phase 2.1 (COMPLETE)**: Conversation tracks current task mode and provides SetTaskMode()/GetTaskMode()
**Phase 2.2 (THIS FRD)**: Manager needs to wire task registry and provide NewConversationWithTask()

## Problem Statement

Currently:
1. Manager creates Agent without passing custom task registry
2. Agent always uses its own default task registry (created in NewAgent)
3. No way to create a conversation that starts in a specific mode (e.g., "review")
4. Manager cannot customize task modes for all its conversations

**Goal**: Manager should support custom task registries and provide API to create conversations in specific modes.

## Requirements

### Functional Requirements

**FR1**: Manager can accept custom task registry via WithTaskRegistry() option
**FR2**: Manager passes task registry to Agent during conversation creation
**FR3**: Manager provides NewConversationWithTask(ctx, workDir, taskName) method
**FR4**: NewConversationWithTask validates task name before creating conversation
**FR5**: Existing NewConversation(ctx, workDir) continues to work unchanged (regular mode default)

### Non-Functional Requirements

**NFR1**: Thread-safe for concurrent conversation creation
**NFR2**: Clear error messages for invalid task names
**NFR3**: Zero breaking changes to existing Manager API
**NFR4**: Test coverage ≥90%

## Design

### 1. Add Task Registry Field to Manager

```go
// internal/core/manager.go

type Manager struct {
	cfg             *Config
	llm             llm.Provider
	emitter         *EventEmitter
	storage         session.Storage
	toolRegistry    *tools.Registry
	taskRegistry    *task.Registry    // NEW: Task registry for all conversations
	mcpManager      *MCPManager
	approvalHandler ApprovalHandler
}
```

### 2. Add WithTaskRegistry Option

```go
// WithTaskRegistry sets a custom task registry for all agents created by this manager
func WithTaskRegistry(registry *task.Registry) ManagerOption {
	return func(m *Manager) error {
		if registry == nil {
			return errors.New("task registry cannot be nil")
		}
		m.taskRegistry = registry
		return nil
	}
}
```

### 3. Update NewConversation to Pass Task Registry

**Modify existing method** (lines 109-151 in manager.go):

```go
func (m *Manager) NewConversation(ctx context.Context, workDir string) (*Conversation, error) {
	logger := withContext(ctx)
	logger.Info("creating new conversation", "work_dir", workDir)

	if workDir == "" {
		workDir = m.cfg.WorkDir
	}

	// Build core dependencies for an Agent
	validator := NewValidator()
	executor, err := NewExecutor(workDir)
	if err != nil {
		logger.Error("failed to create executor", "error", err, "work_dir", workDir)
		return nil, err
	}
	ctxEnv := &Environment{WorkDir: workDir}

	// Build agent with optional tool registry, task registry, and approval
	var agentOpts []AgentOption
	// Enable approval for dangerous commands
	agentOpts = append(agentOpts, WithRequireApproval(true))
	// Set approval handler if configured
	if m.approvalHandler != nil {
		agentOpts = append(agentOpts, WithApprovalHandler(m.approvalHandler))
	}
	if m.toolRegistry != nil {
		agentOpts = append(agentOpts, WithToolRegistry(m.toolRegistry))
		logger.Debug("using custom tool registry", "tool_count", len(m.toolRegistry.ListSchemas()))
	}
	// NEW: Pass task registry if configured
	if m.taskRegistry != nil {
		agentOpts = append(agentOpts, WithTaskRegistry(m.taskRegistry))
		logger.Debug("using custom task registry", "task_count", len(m.taskRegistry.List()))
	}

	agent, err := NewAgent(m.llm, executor, validator, ctxEnv, m.emitter, agentOpts...)
	if err != nil {
		logger.Error("failed to create agent", "error", err)
		return nil, err
	}
	history := NewHistoryWithDefaults()
	_ = history.AddSystemMessage("You are a helpful AI coding assistant.")

	conv := NewConversation(agent, history, m.emitter)
	logger.Info("conversation created successfully")
	return conv, nil
}
```

### 4. Add NewConversationWithTask Method

**NEW method** (add after NewConversation):

```go
// NewConversationWithTask creates a new conversation starting in a specific task mode.
// This is a convenience method that calls NewConversation and then SetTaskMode.
//
// Parameters:
//   - ctx: Context for the operation
//   - workDir: Working directory for the conversation (empty string uses config default)
//   - taskName: Name of the task mode ("regular", "review", "compact", "planning")
//
// Returns:
//   - *Conversation: The new conversation in the specified mode
//   - error: If conversation creation fails or task mode is invalid
//
// Example:
//   conv, err := mgr.NewConversationWithTask(ctx, "/path/to/project", "review")
//   if err != nil {
//       log.Fatal(err)
//   }
//   // Conversation is now in review mode (read-only)
func (m *Manager) NewConversationWithTask(ctx context.Context, workDir string, taskName string) (*Conversation, error) {
	logger := withContext(ctx)
	logger.Info("creating new conversation with task mode", "work_dir", workDir, "task_mode", taskName)

	// Validate task name before creating conversation
	if taskName == "" {
		return nil, errors.New("task name cannot be empty")
	}

	// Create conversation
	conv, err := m.NewConversation(ctx, workDir)
	if err != nil {
		return nil, err
	}

	// Set task mode
	if err := conv.SetTaskMode(taskName); err != nil {
		logger.Error("failed to set task mode", "error", err, "task_mode", taskName)
		return nil, fmt.Errorf("set task mode: %w", err)
	}

	logger.Info("conversation created successfully with task mode", "task_mode", taskName)
	return conv, nil
}
```

### 5. Update ResumeConversation

**Optionally** pass task registry to restored conversations (lines 153-218):

```go
func (m *Manager) ResumeConversation(ctx context.Context, sessionID string) (*Conversation, error) {
	// ... existing code ...

	// Build agent with optional tool registry, task registry, and approval
	var agentOpts []AgentOption
	// Enable approval for dangerous commands
	agentOpts = append(agentOpts, WithRequireApproval(true))
	// Set approval handler if configured
	if m.approvalHandler != nil {
		agentOpts = append(agentOpts, WithApprovalHandler(m.approvalHandler))
	}
	if m.toolRegistry != nil {
		agentOpts = append(agentOpts, WithToolRegistry(m.toolRegistry))
	}
	// NEW: Pass task registry if configured
	if m.taskRegistry != nil {
		agentOpts = append(agentOpts, WithTaskRegistry(m.taskRegistry))
		logger.Debug("using custom task registry for resumed conversation", "task_count", len(m.taskRegistry.List()))
	}

	agent, err := NewAgent(m.llm, executor, validator, ctxEnv, m.emitter, agentOpts...)
	// ... rest of method unchanged ...
}
```

## Implementation Plan

### Tasks

1. ✅ DoR: Read ROADMAP.md P2.2 section
2. ✅ DoR: Read manager.go and understand structure
3. ✅ DoR: Read conversation.go SetTaskMode/GetTaskMode
4. ✅ DoR: Read agent.go WithTaskRegistry option
5. Write FRD (this document)
6. Add `taskRegistry *task.Registry` field to Manager struct
7. Implement `WithTaskRegistry()` option
8. Update `NewConversation()` to pass task registry to agent
9. Update `ResumeConversation()` to pass task registry to agent
10. Implement `NewConversationWithTask()` method
11. Write unit tests (5 tests minimum)
12. Run `go test -race ./internal/core/`
13. Run `make lint` and fix any errors
14. Update docs/packages/core.md with new API
15. Update ROADMAP.md to mark P2.2 complete

## Testing Strategy

### Unit Tests (5 tests)

**Test 1: WithTaskRegistry Option**
```go
func TestManager_WithTaskRegistry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	// Create custom task registry
	customRegistry := task.NewRegistry()
	_ = customRegistry.Register("custom", task.NewCompact())
	_ = customRegistry.SetDefault("custom")

	// Create manager with custom registry
	mgr, err := NewManager(cfg, WithTaskRegistry(customRegistry))
	require.NoError(t, err)
	require.NotNil(t, mgr)
	require.NotNil(t, mgr.taskRegistry)
	require.Equal(t, 1, len(mgr.taskRegistry.List()))
}
```

**Test 2: NewConversationWithTask Success**
```go
func TestManager_NewConversationWithTask_Success(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	mgr, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	require.NoError(t, err)

	// Create conversation in review mode
	conv, err := mgr.NewConversationWithTask(context.Background(), cfg.WorkDir, "review")
	require.NoError(t, err)
	require.NotNil(t, conv)

	// Verify mode is set
	assert.Equal(t, "review", conv.GetTaskMode())
}
```

**Test 3: NewConversationWithTask Invalid Mode**
```go
func TestManager_NewConversationWithTask_InvalidMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	mgr, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	require.NoError(t, err)

	// Try to create conversation with invalid mode
	conv, err := mgr.NewConversationWithTask(context.Background(), cfg.WorkDir, "invalid")
	assert.Error(t, err)
	assert.Nil(t, conv)
	assert.Contains(t, err.Error(), "not found")
}
```

**Test 4: NewConversationWithTask Empty Task Name**
```go
func TestManager_NewConversationWithTask_EmptyTaskName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	mgr, err := NewManager(cfg)
	require.NoError(t, err)

	// Try to create conversation with empty task name
	conv, err := mgr.NewConversationWithTask(context.Background(), cfg.WorkDir, "")
	assert.Error(t, err)
	assert.Nil(t, conv)
	assert.Contains(t, err.Error(), "cannot be empty")
}
```

**Test 5: Task Registry Passed to Agent**
```go
func TestManager_TaskRegistryPassedToAgent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	// Create custom task registry with custom mode
	customRegistry := task.NewRegistry()
	customTask := task.NewCompact() // Use compact as base
	_ = customRegistry.Register("custom-mode", customTask)
	_ = customRegistry.SetDefault("custom-mode")

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	// Create manager with custom registry
	mgr, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter), WithTaskRegistry(customRegistry))
	require.NoError(t, err)

	// Create conversation
	conv, err := mgr.NewConversation(context.Background(), cfg.WorkDir)
	require.NoError(t, err)

	// Verify agent has the custom registry
	modes := conv.agent.ListTaskModes()
	require.Contains(t, modes, "custom-mode")
	assert.Equal(t, 1, len(modes)) // Only our custom mode

	// Verify can switch to custom mode
	err = conv.SetTaskMode("custom-mode")
	require.NoError(t, err)
	assert.Equal(t, "custom-mode", conv.GetTaskMode())
}
```

### Integration Test

**Test 6: End-to-End Mode Creation and Usage**
```go
func TestManager_Integration_TaskModeEndToEnd(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(100)

	mgr, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	require.NoError(t, err)

	// Test all 4 built-in modes
	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			conv, err := mgr.NewConversationWithTask(context.Background(), cfg.WorkDir, mode)
			require.NoError(t, err)
			assert.Equal(t, mode, conv.GetTaskMode())

			// Run a turn to ensure everything is wired correctly
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err = conv.RunTurn(ctx, "test message")
			require.NoError(t, err)
		})
	}
}
```

## Acceptance Criteria

✅ **AC1**: Manager can be created with custom task registry via WithTaskRegistry()
✅ **AC2**: NewConversation passes task registry to agent if configured
✅ **AC3**: NewConversationWithTask creates conversation in specified mode
✅ **AC4**: Invalid task names return clear error messages
✅ **AC5**: Empty task names return error
✅ **AC6**: ResumeConversation also passes task registry to restored conversations
✅ **AC7**: All 6 tests pass
✅ **AC8**: Test coverage ≥90% for new code
✅ **AC9**: `make lint` passes (zero errors)
✅ **AC10**: Race detector clean (`go test -race`)
✅ **AC11**: Godoc complete on all new exports
✅ **AC12**: Existing tests continue to pass (no regressions)

## Definition of Done

- [ ] All tasks completed
- [ ] `taskRegistry` field added to Manager struct
- [ ] `WithTaskRegistry()` option implemented
- [ ] `NewConversation()` updated to pass task registry
- [ ] `ResumeConversation()` updated to pass task registry
- [ ] `NewConversationWithTask()` method implemented
- [ ] 6 unit/integration tests written and passing
- [ ] Test coverage = 100% for new methods
- [ ] `go test -race ./internal/core/` passes
- [ ] `make lint` passes (zero errors)
- [ ] Godoc complete on all new exports
- [ ] docs/packages/core.md updated with new API
- [ ] ROADMAP.md updated to mark P2.2 complete

## Risks and Mitigations

**Risk**: Breaking changes to existing Manager API
**Mitigation**: All changes are additive; existing NewConversation() unchanged

**Risk**: Task registry not properly passed to agent
**Mitigation**: Test coverage verifies agent receives registry; integration test confirms

**Risk**: Error handling for invalid modes unclear
**Mitigation**: Clear error messages with context; documented in godoc

**Risk**: Nil task registry handling
**Mitigation**: WithTaskRegistry validates non-nil; agent has default registry fallback

## Related Work

**Depends On**:
- P1.1-P1.5: Agent task registry integration ✅ COMPLETE
- P2.1: Conversation task mode tracking ✅ COMPLETE

**Enables**:
- P2.3: Integration tests for conversation-level task modes
- P3.x: CLI integration with task modes
- P4.x: AppServer protocol support for task modes

## References

- [ROADMAP.md P2.2](../task-modes/ROADMAP.md#p22-update-manager-for-task-support)
- [Specification](../task-modes/specification.md)
- [AGENTS.md](../../AGENTS.md)
- [docs/packages/core.md](../../docs/packages/core.md)
- [internal/core/manager.go](../../internal/core/manager.go)
- [internal/core/conversation.go](../../internal/core/conversation.go)

---

**Status**: Ready for Implementation
**Next Action**: Implement changes in manager.go
**Estimated Time**: 1 hour
**Last Updated**: 2025-10-12
