# Refactoring Specifications

This directory contains specifications for architectural refactoring work.

## Documents

### Primary Documents

1. **[ROADMAP.md](ROADMAP.md)** - Detailed refactoring roadmap with 24 tasks across 3 phases
   - Phase 1: Foundation (Critical Issues) - 3 tasks, ~10 days
   - Phase 2: Consolidation (Medium Priority) - 3 tasks, ~7 days
   - Phase 3: Polish (Low Priority) - 4 tasks, ~7 days

### Supporting Documents

2. **[Architectural Anti-Patterns Analysis](../../docs/architectural-anti-patterns.md)** - Comprehensive analysis of 10 anti-patterns with code examples and refactoring recommendations

## Quick Links

### Critical Issues (Phase 1)

- [ ] **1.1 Extract Agent Services** - Break god object into smaller services
  - FRD: `specs/frds/FRD-202510-001-agent-service-extraction.md` (TO CREATE)
  - Priority: P0 - CRITICAL
  - Effort: 5 days

- [ ] **1.2 Isolate Event Emitter** - Per-conversation event isolation
  - FRD: `specs/frds/FRD-202510-002-event-emitter-isolation.md` (TO CREATE)
  - Priority: P0 - CRITICAL
  - Effort: 3 days

- [ ] **1.3 Move TUI Mapper** - Remove UI coupling from core
  - FRD: `specs/frds/FRD-202510-003-tui-mapper-relocation.md` (TO CREATE)
  - Priority: P0 - CRITICAL
  - Effort: 2 days

### Medium Priority (Phase 2)

- [ ] **2.1 Consolidate Tool Registry** - Single source for default tools
  - FRD: `specs/frds/FRD-202510-004-tool-registry-consolidation.md` (TO CREATE)
  - Priority: P1 - HIGH
  - Effort: 2 days

- [ ] **2.2 Consolidate Task Registry** - Single initialization function
  - FRD: `specs/frds/FRD-202510-005-task-registry-consolidation.md` (TO CREATE)
  - Priority: P1 - HIGH
  - Effort: 1 day

- [ ] **2.3 Extract Builder Pattern** - Move construction from Manager
  - FRD: `specs/frds/FRD-202510-006-builder-pattern-extraction.md` (TO CREATE)
  - Priority: P2 - MEDIUM
  - Effort: 4 days

### Low Priority (Phase 3)

- [ ] **3.1 Add Interface Segregation** - Agent depends on interfaces
  - FRD: `specs/frds/FRD-202510-007-interface-segregation.md` (TO CREATE)
  - Priority: P3 - LOW
  - Effort: 3 days

- [ ] **3.2 Enrich Config Model** - Add behavior to Config
  - FRD: `specs/frds/FRD-202510-008-config-enrichment.md` (TO CREATE)
  - Priority: P3 - LOW
  - Effort: 2 days

- [ ] **3.3 Type-safe Event Types** - String-based enums
  - FRD: `specs/frds/FRD-202510-009-typesafe-events.md` (TO CREATE)
  - Priority: P3 - LOW
  - Effort: 1 day

- [ ] **3.4 Remove Deprecated Fields** - Clean up Agent struct
  - FRD: `specs/frds/FRD-202510-010-deprecated-cleanup.md` (TO CREATE)
  - Priority: P3 - LOW
  - Effort: 1 day

## Workflow

### For Each Task

1. **Read the documentation**
   - Review [Anti-Patterns Analysis](../../docs/architectural-anti-patterns.md)
   - Read the specific section in [ROADMAP.md](ROADMAP.md)
   - Review ALL `docs/` for context

2. **Create FRD**
   - Use template from ROADMAP.md Appendix A
   - Create `specs/frds/FRD-YYYYMMDD-NNN-task-name.md`
   - Get review/approval

3. **Implement**
   - Follow 14-step workflow from AGENTS.md
   - Write tests FIRST (TDD)
   - Run quality checks continuously

4. **Verify**
   - All tests pass with `-race`
   - Coverage ≥90% for new code
   - `make lint` passes
   - `make deadcode` passes
   - Complexity ≤15

5. **Document**
   - Update `docs/packages/`
   - Update AGENTS.md if needed
   - Update examples
   - Check off in ROADMAP.md

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Agent dependencies | 17 | ≤7 |
| Manager LOC | 513 | <200 |
| Tool registry sources | 3 | 1 |
| Event isolation | Shared | Per-conv |
| Core→UI coupling | Yes | No |
| Test coverage | Current | ≥85% |

## Commands

```bash
# Run tests for a package
go test ./internal/core/... -v -race

# Check coverage
go test ./internal/core/... -cover

# Run linter
make lint

# Check for dead code
make deadcode

# Check complexity
gocyclo -over 15 ./internal/core/...
```

## Related Documentation

- [AGENTS.md](../../AGENTS.md) - Development workflow and principles
- [docs/packages/core.md](../../docs/packages/core.md) - Core package documentation
- [docs/architectural-anti-patterns.md](../../docs/architectural-anti-patterns.md) - Detailed analysis
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Contribution guidelines

## Status

**Current Phase:** Planning  
**Next Action:** Create first FRD for Agent service extraction  
**Last Updated:** 2025-10-19  

---

**Remember:**
- Quality over speed
- Tests first (TDD always)
- No feature merges without e2e coverage
- Refactor, but never simplify implementation
- Zero dead code
- Follow ALL 14 steps from AGENTS.md

