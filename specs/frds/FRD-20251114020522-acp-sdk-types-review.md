# FRD: Review SDK Types and Interfaces

**ID**: FRD-20251114020522  
**Feature**: 1.2 - Review SDK Types and Interfaces  
**Status**: In Progress  
**Created**: 2025-11-14  
**Roadmap**: [specs/acp/ROADMAP.md](../../specs/acp/ROADMAP.md)

## Overview

Review and understand all SDK-provided types and interfaces to create a comprehensive type mapping document between ACP SDK types and Spin's internal types. This prevents duplication and ensures proper integration.

## Goals

1. Review all SDK type definitions
2. Understand `acp.Agent` interface in detail
3. Understand connection infrastructure
4. Review helper functions
5. Create type mapping document: SDK types → Spin internal types
6. Write exploratory tests to verify understanding

## Requirements

### Functional Requirements

1. **SDK Type Review**
   - Review `types_gen.go` - all generated ACP types
   - Review `agent.go` - Agent interface
   - Review `agent_gen.go` - Generated agent method types
   - Review `connection.go` - Connection infrastructure
   - Review `helpers.go` - Helper constructors

2. **Interface Understanding**
   - Understand `acp.Agent` interface methods and signatures
   - Understand request/response types for each method
   - Understand notification types
   - Understand capability types

3. **Type Mapping**
   - Map ACP ContentBlock types → Spin ContentItem types
   - Map ACP Message types → Spin Message types
   - Map ACP Tool types → Spin ToolCall types
   - Map ACP Session types → Spin Session types
   - Map ACP MCP types → Spin MCP types

4. **Exploratory Tests**
   - Test SDK type creation and usage
   - Test helper functions
   - Test type conversions (where applicable)
   - Verify no duplicate type definitions needed

### Non-Functional Requirements

1. **Documentation**
   - Type mapping document created
   - All SDK types documented
   - Conversion patterns documented

2. **Testing**
   - Exploratory tests written
   - Tests demonstrate SDK type usage
   - Tests verify understanding

## Design

### SDK Type Categories

1. **Request/Response Types**
   - InitializeRequest/Response
   - NewSessionRequest/Response
   - PromptRequest/Response
   - CancelNotification
   - SetSessionModeRequest/Response
   - AuthenticateRequest/Response

2. **Content Types**
   - ContentBlock (union type)
   - ContentBlockText
   - ContentBlockImage
   - ContentBlockAudio
   - ContentBlockResource
   - ContentBlockResourceLink

3. **Session Types**
   - SessionUpdate
   - AgentMessageChunk
   - UserMessageChunk
   - ToolCall
   - ToolCallUpdate

4. **Capability Types**
   - AgentCapabilities
   - ClientCapabilities
   - PromptCapabilities
   - McpCapabilities

5. **MCP Types**
   - McpServer
   - McpServerStdio
   - McpServerHttp
   - McpServerSse

### Type Mapping Strategy

Create mapping functions (not implementations yet, just understanding):
- `SpinContentItemToACP()` - Convert Spin ContentItem → ACP ContentBlock
- `ACPContentBlockToSpin()` - Convert ACP ContentBlock → Spin ContentItem
- `SpinMessageToACP()` - Convert Spin Message → ACP message types
- `ACPToolCallToSpin()` - Convert ACP ToolCall → Spin ToolCall

## Implementation Plan

### Phase 1: SDK Type Exploration
1. Review SDK source files
2. Document all type definitions
3. Understand type relationships

### Phase 2: Type Mapping
1. Map ACP types to Spin types
2. Identify conversion points
3. Document conversion patterns

### Phase 3: Exploratory Tests
1. Write tests for SDK type creation
2. Write tests for helper functions
3. Write tests for type exploration

### Phase 4: Documentation
1. Create type mapping document
2. Update SDK integration docs
3. Document conversion patterns

## Testing Strategy

### Exploratory Tests

Write tests that:
- Create SDK types using helper functions
- Explore SDK type structures
- Verify type relationships
- Test type conversions (conceptual, not full implementation)

### Test Coverage

- SDK type creation
- Helper function usage
- Type structure exploration
- No duplicate type definitions

## Acceptance Criteria

- [ ] All SDK type definitions reviewed
- [ ] `acp.Agent` interface fully understood
- [ ] Connection infrastructure understood
- [ ] Helper functions reviewed and documented
- [ ] Type mapping document created (SDK types → Spin types)
- [ ] Exploratory tests written and passing
- [ ] No duplicate type definitions planned

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Missing type mappings | Medium | Low | Comprehensive review of all SDK types |
| Incorrect understanding | High | Medium | Write exploratory tests to verify |
| Type conversion complexity | Medium | Medium | Document conversion patterns clearly |

## Dependencies

### External Dependencies
- `github.com/coder/acp-go-sdk` v0.6.3 (already added in Feature 1.1)

### Internal Dependencies
- `internal/message` - Message types
- `internal/protocol` - Protocol types
- `internal/orchestration` - Tool types
- `internal/session` - Session types
- `internal/mcp` - MCP types

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md)
- [ACP SDK Integration Docs](../../docs/packages/acp-sdk-integration.md)
- [acp-go-sdk Repository](https://github.com/coder/acp-go-sdk)

## Notes

- This is an exploration phase - no production code yet
- Focus on understanding, not implementation
- Type mapping document will guide future implementation
- Tests are exploratory, not production tests

