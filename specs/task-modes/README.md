# Task Mode System Integration

**Status**: 📋 Ready for Implementation
**Created**: 2025-10-12
**Priority**: Medium
**Estimated Effort**: 7-8 development days

## Overview

This specification folder contains the complete plan to integrate the existing task mode system into the Spin agent. The task mode system is **fully implemented** but **never wired up** to production code.

## What's Inside

### [specification.md](./specification.md)
**Complete technical specification** describing:
- Current state analysis (what exists vs what's missing)
- Integration architecture (4 phases of implementation)
- Detailed code changes for each component
- Testing strategy (unit, integration, e2e)
- Performance considerations and benchmarks
- Security implications
- Migration path for existing users

**Read this first** to understand the technical depth and scope.

### [ROADMAP.md](./ROADMAP.md)
**Detailed implementation roadmap** with:
- 19 granular tasks organized in 5 phases
- Definition of Ready (DoR) for each task
- Definition of Done (DoD) with acceptance criteria
- Time estimates and dependencies
- Risk assessment and mitigation
- Quality gates and success metrics

**Use this** as the day-to-day implementation guide.

## Quick Summary

### The Problem
The Spin agent has 4 task modes implemented in `internal/core/task/`:
- **Regular**: Full-featured interactive coding (16K tokens, all tools)
- **Review**: Read-only code analysis (12K tokens, read-only tools)
- **Compact**: Quick queries (4K tokens, minimal tools)
- **Planning**: Task decomposition (4K tokens, context tools)

**BUT**: These modes are never used. The agent always uses default behavior.

### The Solution
Wire task modes through 4 integration layers:

1. **Core Agent** - Task registry, resolution, tool filtering, token budgets
2. **Conversation** - Mode tracking, switching API, persistence
3. **CLI** - `--mode` flag, `/mode` REPL command, mode info commands
4. **AppServer** - WebSocket protocol support for mode switching

### The Value
- 🎯 **Better UX**: Right mode for the right task
- 💰 **Cost savings**: 75% token reduction in compact mode
- 🔒 **Safety**: Read-only mode for code review
- ⚡ **Performance**: Faster responses with fewer tools

## Implementation Phases

```
Phase 1: Core Agent Integration (2 days)
├─ P1.1: Add task registry to Agent ⭐ CRITICAL
├─ P1.2: Implement task resolution logic
├─ P1.3: Implement tool filtering
├─ P1.4: Apply token budget from task
└─ P1.5: Integration tests for core agent

Phase 2: Conversation Integration (1 day)
├─ P2.1: Add task mode to Conversation
├─ P2.2: Update Manager for task support
└─ P2.3: Integration tests for conversation

Phase 3: CLI Integration (1.5 days)
├─ P3.1: Add global task mode flag
├─ P3.2: Implement REPL mode switching
├─ P3.3: Add mode info command
└─ P3.4: CLI integration tests

Phase 4: AppServer Integration (1 day)
├─ P4.1: Update protocol with task mode field
├─ P4.2: Handle task mode in processor
└─ P4.3: AppServer integration tests

Phase 5: Documentation & Polish (1.5 days)
├─ P5.1: Update package documentation
├─ P5.2: Add mode usage examples
├─ P5.3: Update CLI help and documentation
├─ P5.4: Performance testing and optimization
└─ P5.5: Final integration testing
```

## Success Criteria

**Quality Gates:**
- ✅ Test coverage ≥85% (≥90% critical paths)
- ✅ `make lint` clean (zero errors)
- ✅ Race detector clean
- ✅ Complexity ≤15 for all functions
- ✅ All e2e tests passing

**Functional:**
- ✅ All 4 modes work end-to-end
- ✅ Tool filtering enforced
- ✅ Token budgets applied
- ✅ CLI commands work
- ✅ WebSocket protocol support

**Performance:**
- ✅ Tool filtering < 100μs
- ✅ Mode switching < 1μs
- ✅ Memory overhead < 10KB

## Getting Started

### For Implementers

1. **Read the specification** ([specification.md](./specification.md))
   - Understand current state and gaps
   - Review integration architecture
   - Study code examples

2. **Review the roadmap** ([ROADMAP.md](./ROADMAP.md))
   - Understand task breakdown
   - Review DoR/DoD for first task
   - Check dependencies and risks

3. **Start with P1.1** (Add Task Registry to Agent)
   - This is the critical foundation
   - All other work depends on it
   - Expected time: 2-3 hours

4. **Follow AGENTS.md workflow**
   - Read all docs/ before starting
   - Write tests first
   - Run uast/herr analysis
   - Keep lint clean

### For Reviewers

**Key areas to review:**
- Thread safety (TaskRegistry has RWMutex)
- Tool filtering security (enforces mode restrictions)
- Backward compatibility (all changes are additive)
- Error handling (clear messages for users)
- Performance (benchmarks included)

**Red flags:**
- Breaking API changes (should be zero)
- Tool filtering bypass (security issue)
- Race conditions (test with -race)
- Flaky tests (fix before merge)

## Timeline

**Week 1**: Core + Conversation (Days 1-3)
**Week 2**: CLI + AppServer (Days 4-5)
**Week 3**: Documentation + Testing (Days 6-7)

**Total**: ~7-8 days with buffer for issues

## Architecture Preview

```
┌─────────────────────────────────────────────┐
│              User Input                     │
└─────────────┬───────────────────────────────┘
              │
              v
┌─────────────────────────────────────────────┐
│  CLI / REPL / WebSocket                     │
│  - Parse --mode flag or /mode command      │
│  - Validate mode name                       │
└─────────────┬───────────────────────────────┘
              │
              v
┌─────────────────────────────────────────────┐
│  Conversation                               │
│  - Track current mode                       │
│  - SetTaskMode() / GetTaskMode()           │
└─────────────┬───────────────────────────────┘
              │
              v
┌─────────────────────────────────────────────┐
│  Agent                                      │
│  - TaskRegistry (4 modes registered)       │
│  - resolveTask() → Task                    │
│  - buildToolsForTask() → Filtered tools    │
└─────────────┬───────────────────────────────┘
              │
              v
┌─────────────────────────────────────────────┐
│  LLM Provider                               │
│  - Receives filtered tools                  │
│  - Respects token budget                    │
└─────────────────────────────────────────────┘
```

## Key Files to Modify

**Core Layer:**
- `internal/core/agent.go` (~200 lines added, ~20 modified)
- `internal/core/agent_test.go` (new tests)
- `internal/core/conversation.go` (~40 lines added)
- `internal/core/conversation_test.go` (new tests)

**CLI Layer:**
- `cmd/spin/root.go` (~10 lines added)
- `cmd/spin/mode.go` (NEW FILE ~80 lines)
- `cmd/spin/repl.go` (~50 lines added)

**AppServer Layer:**
- `internal/appserver/protocol.go` (~5 lines added)
- `internal/appserver/processor.go` (~10 lines added)

**Documentation:**
- `docs/packages/core.md` (updated)
- `docs/packages/tools.md` (updated)
- `docs/modes.md` (NEW FILE)
- `examples/task-modes/` (NEW DIRECTORY)

## Testing Strategy

**Unit Tests** (target: ≥90% coverage)
- Task resolution logic
- Tool filtering algorithm
- Token budget application
- Registry thread safety

**Integration Tests** (target: ≥85% coverage)
- Agent + task mode interaction
- Conversation mode switching
- Tool filtering in live conversations

**E2E Tests** (must all pass)
- CLI mode flag
- REPL mode commands
- WebSocket protocol
- Real tool restriction enforcement

## Common Questions

**Q: Is this a breaking change?**
A: No. All changes are additive. Default behavior unchanged.

**Q: What if a user doesn't specify a mode?**
A: Defaults to "regular" mode (current behavior).

**Q: Can modes be switched mid-conversation?**
A: Yes, via `/mode` command in REPL or task_mode field in WebSocket.

**Q: How are tools restricted?**
A: Tool filtering happens in `buildToolsForTask()` before sending to LLM.

**Q: What's the performance impact?**
A: Minimal. Tool filtering adds ~50-100μs per LLM call.

**Q: Can users define custom modes?**
A: Not in v1. Future enhancement (see spec section on custom modes).

## Related Work

**Depends On:**
- Existing task implementations (`internal/core/task/`)
- Tool registry (`internal/tools/`)
- Agent infrastructure (`internal/core/`)

**Enables:**
- Custom mode definitions (future)
- Auto mode selection (future)
- Mode composition (future)
- Per-tool permissions (future)

## Resources

- [Core Package Docs](../../docs/packages/core.md)
- [Tools Package Docs](../../docs/packages/tools.md)
- [AGENTS.md](../../AGENTS.md) - Development standards
- [Existing Task Tests](../../internal/core/task/task_test.go)

## Status Tracking

**Current Status**: ✅ COMPLETE - All Phases Done! ✨
**Next Action**: None - feature ready for production
**Blocked By**: None
**Blockers For**: None

**Progress**: 19 / 19 tasks complete (100%)

**Phase Status:**
- [x] Phase 1: Core Agent Integration (5/5) ✅ COMPLETE
- [x] Phase 2: Conversation Integration (3/3) ✅ COMPLETE
- [x] Phase 3: CLI Integration (4/4) ✅ COMPLETE
- [x] Phase 4: AppServer Integration (3/3) ✅ COMPLETE
- [x] Phase 5: Documentation & Polish (5/5) ✅ COMPLETE

**🎉 Feature Implementation Complete!**

**Key Achievements:**
- ✅ All 19 tasks completed successfully
- ✅ Test coverage: 85%+ (exceeds target)
- ✅ Performance: 2-10x better than targets
- ✅ Zero lint errors, race detector clean
- ✅ Comprehensive documentation (600+ lines)
- ✅ All 4 modes working end-to-end

---

**Last Updated**: 2025-10-12
**Maintainer**: Development Team
**Questions?**: See specification.md or AGENTS.md
