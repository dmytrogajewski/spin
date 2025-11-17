# FRD: Add acp-go-sdk Dependency

**ID**: FRD-20251114020209  
**Feature**: 1.1 - Add acp-go-sdk Dependency  
**Status**: In Progress  
**Created**: 2025-11-14  
**Roadmap**: [specs/acp/ROADMAP.md](../../specs/acp/ROADMAP.md)

## Overview

Add the `acp-go-sdk` dependency to enable Agent Client Protocol (ACP) support in Spin. This is the foundational step for implementing ACP protocol integration.

## Goals

1. Add `github.com/coder/acp-go-sdk` as a dependency
2. Review SDK structure and interfaces
3. Document SDK integration approach
4. Ensure no breaking changes to existing code

## Requirements

### Functional Requirements

1. **Dependency Management**
   - Add `acp-go-sdk` to `go.mod`
   - Use stable version (v0.6.3 or latest stable)
   - Ensure compatibility with Go 1.24

2. **SDK Review**
   - Review SDK repository structure
   - Understand `acp.Agent` interface
   - Understand connection handling (`NewAgentSideConnection`)
   - Review helper functions (`acp.TextBlock`, `acp.ImageBlock`, etc.)

3. **Documentation**
   - Create `docs/packages/acp-sdk-integration.md`
   - Document SDK structure and key interfaces
   - Document integration approach

### Non-Functional Requirements

1. **Compatibility**
   - No breaking changes to existing code
   - All existing tests must pass
   - No new lint errors

2. **Quality**
   - Follow Go dependency management best practices
   - Document dependency rationale

## Design

### Dependency Addition

```go
// go.mod
require (
    // ... existing dependencies
    github.com/coder/acp-go-sdk v0.6.3
)
```

### SDK Structure (Expected)

Based on roadmap notes:
- `types_gen.go` - Generated ACP types (from schema)
- `agent.go` - `acp.Agent` interface
- `agent_gen.go` - Generated agent method types
- `connection.go` - Connection infrastructure
- `helpers.go` - Helper constructors

### Documentation Structure

```
docs/packages/acp-sdk-integration.md
├── Overview
├── SDK Structure
├── Key Interfaces
│   ├── acp.Agent
│   ├── Connection Handling
│   └── Helper Functions
├── Integration Approach
└── Examples
```

## Implementation Plan

### Phase 1: Dependency Addition
1. Add dependency: `go get github.com/coder/acp-go-sdk@v0.6.3`
2. Run `go mod tidy`
3. Verify dependency in `go.mod` and `go.sum`

### Phase 2: SDK Exploration
1. Review SDK source code structure
2. Identify key interfaces and types
3. Understand connection setup patterns
4. Review helper functions

### Phase 3: Documentation
1. Create `docs/packages/acp-sdk-integration.md`
2. Document SDK structure
3. Document integration approach
4. Add examples

### Phase 4: Verification
1. Run all tests: `go test ./...`
2. Run linter: `make lint`
3. Verify no breaking changes

## Testing Strategy

### Unit Tests
- No new tests required (dependency addition only)
- Verify dependency can be imported

### Integration Tests
- Verify SDK types can be used
- Verify no import conflicts

### E2E Tests
- Not applicable for this feature

## Acceptance Criteria

- [ ] `acp-go-sdk` added to `go.mod` at version v0.6.3 (or latest stable)
- [ ] `go mod tidy` runs successfully
- [ ] All existing tests pass
- [ ] No new lint errors
- [ ] SDK structure reviewed and documented
- [ ] `docs/packages/acp-sdk-integration.md` created
- [ ] Integration approach documented

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| SDK version incompatibility | Medium | Low | Use stable version, test thoroughly |
| Breaking changes in SDK | Medium | Low | Pin version, review changelog |
| Import conflicts | Low | Low | SDK uses standard Go module structure |

## Dependencies

### External Dependencies
- `github.com/coder/acp-go-sdk` v0.6.3 (or latest stable)

### Internal Dependencies
- None (this is foundational)

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md)
- [ACP Specification](../../specs/acp/specification.md)
- [acp-go-sdk Repository](https://github.com/coder/acp-go-sdk)

## Notes

- This is a foundational feature - no implementation of ACP agent yet
- Focus is on dependency management and SDK understanding
- Next feature (1.2) will review SDK types and interfaces in detail

