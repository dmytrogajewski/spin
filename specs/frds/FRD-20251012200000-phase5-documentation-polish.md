# FRD: Phase 5 - Documentation & Polish

**ID**: FRD-20251012200000
**Status**: Implementation
**Created**: 2025-10-12
**Roadmap**: Phase 5 - Documentation & Polish
**Phase**: 5 - Final Documentation and Testing
**Priority**: HIGH
**Estimated Effort**: 1.5 days (8-10 hours)

## Overview

Complete the task mode system integration with comprehensive documentation, examples, performance testing, and final E2E validation. This is the final phase that makes the feature production-ready.

## Problem Statement

Phases 1-4 are complete with working code:
- ✅ P1.1-P1.5: Core Agent Integration (100%)
- ✅ P2.1-P2.3: Conversation Integration (100%)
- ✅ P3.1-P3.4: CLI Integration (100%)
- ✅ P4.1-P4.3: AppServer Integration (100%)

**Missing for production readiness:**
1. **Documentation**: Users don't know how to use task modes
2. **Examples**: No working code samples for each mode
3. **Performance**: No benchmarks to verify overhead is acceptable
4. **E2E Testing**: No comprehensive system tests

Without Phase 5, the feature is technically complete but not production-ready.

## Goals

### P5.1: Update Package Documentation
1. Document task modes in `docs/packages/core.md`
2. Document tool filtering in `docs/packages/tools.md`
3. Create comprehensive `docs/modes.md` guide
4. Add migration guide for existing users

### P5.2: Add Mode Usage Examples
1. Create `examples/task-modes/` directory with 4 working examples
2. Demonstrate each mode's capabilities
3. Provide clear README with usage instructions

### P5.3: Update CLI Help and Documentation
1. Update `spin --help` to mention modes
2. Ensure `spin mode --help` is comprehensive
3. Add task modes section to main README.md
4. Update getting started guide

### P5.4: Performance Testing and Optimization
1. Create benchmarks for task resolution, tool filtering, mode switching
2. Verify overhead targets: filtering <100μs, switching <1μs, memory <10KB
3. Optimize if needed
4. Document performance characteristics

### P5.5: Final Integration Testing
1. Create comprehensive E2E test suite
2. Test all 4 modes end-to-end
3. Test mode switching mid-conversation
4. Test tool restriction enforcement
5. Verify all quality gates

## Non-Goals

- Custom mode definitions (future enhancement)
- Auto mode selection (future enhancement)
- Web UI for mode management (future enhancement)
- Mode composition/inheritance (future enhancement)

## Design

### P5.1: Package Documentation Updates

#### docs/packages/core.md

Add new section:

```markdown
## Task Modes

The agent supports multiple task modes, each optimized for specific workflows:

### Available Modes

#### Regular Mode (default)
- **Token Budget**: 16K
- **Tools**: All tools available
- **Use Case**: Full-featured interactive coding
- **Best For**: General development, feature implementation, debugging

#### Review Mode
- **Token Budget**: 12K
- **Tools**: Read-only (read_file, list_directory, get_context, file_search, git_context)
- **Use Case**: Code review and analysis
- **Best For**: Code audits, understanding existing code, reviewing PRs

#### Compact Mode
- **Token Budget**: 4K
- **Tools**: Minimal (read_file, get_context, file_search)
- **Use Case**: Quick queries with minimal context
- **Best For**: Simple questions, quick lookups, fast responses

#### Planning Mode
- **Token Budget**: 4K
- **Tools**: Context tools (get_context, file_search, git_context)
- **Use Case**: Task decomposition and planning
- **Best For**: Breaking down features, understanding architecture, planning work

### API Usage

```go
// Create agent with default task registry
agent, err := core.NewAgent(llm, executor, validator, context, emitter)

// Use specific task mode
req := &core.AgentRequest{
    Input:    "Review this code",
    TaskName: "review",
    History:  history,
}

resp, err := agent.Execute(ctx, req)
```

### Conversation API

```go
// Create conversation
conv, err := manager.NewConversation(ctx, workDir)

// Switch task mode
err = conv.SetTaskMode("review")

// Get current mode
mode := conv.GetTaskMode() // returns "review"
```

### Custom Task Registry

```go
// Create custom registry
registry := core.NewTaskRegistry()
registry.Register("regular", task.NewRegular())
registry.Register("custom", myCustomTask)
registry.SetDefault("custom")

// Use in agent
agent, err := core.NewAgent(
    llm, executor, validator, context, emitter,
    core.WithTaskRegistry(registry),
)
```
```

#### docs/packages/tools.md

Add new section:

```markdown
## Tool Filtering by Task Mode

Tool availability is controlled by the current task mode. This ensures:
- **Safety**: Read-only modes prevent destructive operations
- **Cost Optimization**: Fewer tools = smaller context = lower token costs
- **Focus**: Only relevant tools for the current task

### How Filtering Works

1. **Task Definition**: Each task specifies allowed tools
   ```go
   func (t *ReviewTask) AllowedTools() []string {
       return []string{"read_file", "list_directory", "get_context", "file_search", "git_context"}
   }
   ```

2. **Build Time**: Agent filters tool schemas before sending to LLM
   ```go
   tools, err := agent.buildToolsForTask(task)
   // Only includes tools from AllowedTools()
   ```

3. **Runtime**: LLM can only call allowed tools (others are invisible)

### Tool Access by Mode

| Tool | Regular | Review | Compact | Planning |
|------|---------|--------|---------|----------|
| read_file | ✅ | ✅ | ✅ | ❌ |
| write_file | ✅ | ❌ | ❌ | ❌ |
| execute_command | ✅ | ❌ | ❌ | ❌ |
| list_directory | ✅ | ✅ | ❌ | ❌ |
| get_context | ✅ | ✅ | ✅ | ✅ |
| file_search | ✅ | ✅ | ✅ | ✅ |
| git_context | ✅ | ✅ | ❌ | ✅ |

### Performance

Tool filtering adds ~50-100μs overhead per LLM call. This is negligible compared to network latency and LLM processing time.
```

#### docs/modes.md (NEW FILE)

Create comprehensive user guide:

```markdown
# Task Modes Guide

## Overview

Spin supports four task modes, each optimized for specific workflows. Task modes control:
- **Token budget**: How much context the LLM can process
- **Tool access**: Which tools are available to the agent
- **System prompt**: Task-specific instructions

## When to Use Each Mode

### Regular Mode (Default)
**Token Budget**: 16K | **Tools**: All

Use when:
- Implementing new features
- Debugging complex issues
- General interactive coding
- You need full tool access

Example:
```bash
spin --mode regular
> Create a new API endpoint for user authentication
```

### Review Mode
**Token Budget**: 12K | **Tools**: Read-only

Use when:
- Reviewing code changes
- Understanding existing code
- Conducting security audits
- Analyzing architecture

Example:
```bash
spin --mode review
> Review the authentication logic in auth.go for security issues
```

### Compact Mode
**Token Budget**: 4K | **Tools**: Minimal (read, search)

Use when:
- Quick questions
- Fast lookups
- Simple queries
- Cost-sensitive operations

Example:
```bash
spin --mode compact
> What does the validateUser function do?
```

### Planning Mode
**Token Budget**: 4K | **Tools**: Context tools

Use when:
- Planning feature implementation
- Breaking down tasks
- Understanding project structure
- Architectural decisions

Example:
```bash
spin --mode planning
> How should I structure the user authentication system?
```

## Usage

### CLI Flag
```bash
# Start with specific mode
spin --mode review

# Short form
spin -m compact
```

### Interactive Mode Switching
```bash
$ spin
> /mode review
Switched to review mode

> /mode
Current mode: review

> /help
Commands:
  /mode [name]  - Show or switch task mode
  /help         - Show this help
  /exit         - Exit the session
```

### Standalone Mode Command
```bash
# List all modes
$ spin mode list
Available modes:
  regular   - Full-featured interactive coding (16K tokens, all tools)
  review    - Read-only code review (12K tokens, read-only tools)
  compact   - Quick queries (4K tokens, 3 essential tools)
  planning  - Task decomposition (4K tokens, context tools)

# Describe specific mode
$ spin mode describe review
Mode: review
Description: Read-only code analysis mode
Max Tokens: 12288
Allowed Tools:
  - read_file
  - list_directory
  - get_context
  - file_search
  - git_context
```

### WebSocket/JSON-RPC API
```json
{
  "method": "send_message",
  "params": {
    "message": "Review this code",
    "task_mode": "review"
  }
}
```

Response includes current mode:
```json
{
  "result": {
    "conversation_id": "conv-123",
    "turn_id": "turn-456",
    "task_mode": "review"
  }
}
```

## Best Practices

### Cost Optimization
- Use `compact` mode for simple queries (75% token savings)
- Use `planning` mode for architecture discussions
- Reserve `regular` mode for implementation work

### Safety
- Use `review` mode when you only need to read code
- Prevents accidental modifications during code review
- Ideal for security audits and PR reviews

### Performance
- Smaller modes (compact, planning) respond faster
- Fewer tools = less context for LLM to process
- Use the smallest mode that fits your task

## Migration from Pre-Mode Version

If you were using Spin before task modes:

**Old behavior (still works):**
```bash
spin
> Create a new file
```
→ Uses "regular" mode (same as before)

**New capabilities:**
```bash
# Explicitly choose mode
spin --mode review
> Analyze this code

# Switch modes mid-session
> /mode compact
> Quick question?
```

**No breaking changes**: All existing code and CLI usage works unchanged. Task modes are opt-in.

## Troubleshooting

### "Invalid task mode" error
```
Error: invalid task mode: invalid (valid: regular, review, compact, planning)
```
**Solution**: Check mode name spelling. Valid modes: `regular`, `review`, `compact`, `planning`

### Tools not available in mode
If a tool is unavailable, you may be in a restricted mode.
```bash
> /mode
Current mode: review

> /mode regular
Switched to regular mode
```

### Which mode should I use?
- **Not sure?** Use `regular` (default)
- **Read-only?** Use `review`
- **Quick question?** Use `compact`
- **Planning work?** Use `planning`

## Examples

See `examples/task-modes/` for working code examples demonstrating each mode.
```

### P5.2: Mode Usage Examples

Create `examples/task-modes/` with structure:

```
examples/task-modes/
├── README.md
├── regular/
│   └── main.go
├── review/
│   └── main.go
├── compact/
│   └── main.go
└── planning/
    └── main.go
```

Each example demonstrates:
- How to create agent/conversation with that mode
- Typical use cases
- Expected behavior

### P5.3: CLI Help Updates

Update help text to mention modes prominently:

```go
// cmd/spin/root.go
Use:   "spin",
Short: "Spin - AI-powered coding assistant",
Long: `Spin is an AI-powered coding assistant with task mode support.

Task Modes:
  regular  - Full-featured coding (default)
  review   - Read-only code review
  compact  - Quick queries
  planning - Task planning

Use --mode flag or /mode command to switch modes.`,
```

### P5.4: Performance Benchmarks

Create `internal/core/agent_bench_test.go`:

```go
func BenchmarkAgent_TaskResolution(b *testing.B) {
    agent := newBenchAgent(b)
    req := &AgentRequest{TaskName: "review"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = agent.resolveTask(req)
    }
}

func BenchmarkAgent_ToolFiltering(b *testing.B) {
    agent := newBenchAgent(b)
    task := task.NewReview()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = agent.buildToolsForTask(task)
    }
}

func BenchmarkConversation_ModeSwitch(b *testing.B) {
    conv := newBenchConversation(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = conv.SetTaskMode("review")
        _ = conv.SetTaskMode("regular")
    }
}
```

**Targets:**
- Task resolution: < 1μs
- Tool filtering: < 100μs
- Mode switching: < 1μs

### P5.5: Final E2E Testing

Create `e2e/task_modes_test.go` (Go-based E2E tests):

```go
func TestE2E_AllModes(t *testing.T) {
    modes := []string{"regular", "review", "compact", "planning"}

    for _, mode := range modes {
        t.Run(mode, func(t *testing.T) {
            // Test mode end-to-end with real agent
            testModeE2E(t, mode)
        })
    }
}

func TestE2E_ModeSwitching(t *testing.T) {
    // Test switching modes mid-conversation
}

func TestE2E_ToolRestrictionEnforcement(t *testing.T) {
    // Verify review mode can't write files
    // Verify compact mode has minimal tools
}
```

## Testing Strategy

### P5.1: Documentation Tests
- [ ] All links work
- [ ] Code examples are syntactically correct
- [ ] Markdown linting passes
- [ ] Examples can be copy-pasted

### P5.2: Example Tests
- [ ] Each example compiles
- [ ] Examples run without errors
- [ ] Examples demonstrate correct usage
- [ ] README is clear and helpful

### P5.3: CLI Help Tests
- [ ] `spin --help` shows mode info
- [ ] `spin mode --help` is comprehensive
- [ ] Help text is accurate
- [ ] Examples work

### P5.4: Performance Tests
- [ ] All benchmarks run successfully
- [ ] Performance targets met
- [ ] No regressions vs baseline
- [ ] Results documented

### P5.5: E2E Tests
- [ ] All modes work end-to-end
- [ ] Mode switching works
- [ ] Tool restrictions enforced
- [ ] No flaky tests
- [ ] Execution time < 5 minutes

## Definition of Done

### Overall Phase 5
- [ ] All 5 sub-tasks (P5.1-P5.5) complete
- [ ] Documentation comprehensive and accurate
- [ ] Examples work correctly
- [ ] Performance targets met
- [ ] E2E tests passing
- [ ] `make lint` clean
- [ ] ROADMAP.md updated to 100% complete

### P5.1: Documentation
- [ ] `docs/packages/core.md` updated
- [ ] `docs/packages/tools.md` updated
- [ ] `docs/modes.md` created
- [ ] All examples work
- [ ] Links verified
- [ ] Markdown linting passes

### P5.2: Examples
- [ ] 4 examples created (regular, review, compact, planning)
- [ ] README explains each example
- [ ] Examples compile and run
- [ ] Examples demonstrate key features

### P5.3: CLI Help
- [ ] Root help mentions modes
- [ ] Mode command help complete
- [ ] README.md updated
- [ ] Getting started guide updated

### P5.4: Performance
- [ ] Benchmarks created
- [ ] Task resolution < 1μs ✅
- [ ] Tool filtering < 100μs ✅
- [ ] Mode switching < 1μs ✅
- [ ] Memory overhead < 10KB ✅
- [ ] Results documented

### P5.5: E2E Testing
- [ ] E2E test suite created
- [ ] All modes tested
- [ ] Mode switching tested
- [ ] Tool restrictions tested
- [ ] All tests pass
- [ ] No flakes

## Success Criteria

**User Experience:**
1. Users can discover task modes via --help
2. Users understand when to use each mode
3. Examples provide clear starting points
4. Documentation answers common questions

**Quality:**
1. Test coverage ≥85% overall (already achieved)
2. Performance targets met
3. No lint errors (already achieved)
4. No race conditions (already verified)

**Completeness:**
1. All documentation updated
2. All examples working
3. All tests passing
4. ROADMAP 100% complete

## References

- [ROADMAP Phase 5](../../task-modes/ROADMAP.md#phase-5-documentation--polish-15-days)
- [Task Modes Specification](../../task-modes/specification.md)
- [AGENTS.md](../../../AGENTS.md)
- [Core Package Docs](../../../docs/packages/core.md)
- [Tools Package Docs](../../../docs/packages/tools.md)

## Changelog

- **2025-10-12**: Initial version
