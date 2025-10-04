# FRD-8.13: MCP ListResources Implementation

**Status**: ✅ Implementation
**Priority**: Medium
**Created**: 2025-10-04
**Component**: `internal/mcp/client`

---

## Overview

Implement MCP (Model Context Protocol) resources/list functionality in the StdioClient to enable listing available resources from MCP servers. This completes a core part of the MCP specification that allows agents to discover and access server-provided resources (files, data sources, etc.).

## Background

The MCP specification defines resources as server-provided data that clients can discover and read. Resources are identified by URIs and can contain text or binary data. The `resources/list` method allows clients to discover what resources are available from a server.

Currently, the `StdioClient.ListResources()` method is a stub that returns "not implemented" error.

## Requirements

### Functional Requirements

**FR-1**: Implement `ListResources` method that:
- Accepts a `context.Context` parameter for cancellation/timeout
- Calls the MCP `resources/list` JSON-RPC method
- Returns `*types.ListResourcesResponse` containing available resources
- Returns error on failure

**FR-2**: Validate client state:
- Return error if client is not initialized (similar to `ListTools`)
- Ensure proper error wrapping with operation context

**FR-3**: Handle pagination (future-proof):
- Accept optional cursor parameter for paginated results
- Return nextCursor in response if provided by server
- Initial implementation can use empty cursor (list all)

**FR-4**: Follow existing patterns:
- Use same JSON-RPC request/response flow as `ListTools`
- Use same error handling patterns as other methods
- Maintain consistent operation naming for errors

### Non-Functional Requirements

**NFR-1**: **Performance**:
- No blocking operations on client initialization
- Respect context timeouts and cancellation
- Reuse existing JSON-RPC infrastructure

**NFR-2**: **Reliability**:
- Handle network errors gracefully
- Proper error wrapping for debugging
- Validate response structure

**NFR-3**: **Maintainability**:
- Follow existing code patterns in stdio.go
- Use consistent naming conventions
- Add appropriate code comments

## Design

### API Design

```go
// ListResources lists available resources from the server.
func (c *StdioClient) ListResources(ctx context.Context) (*types.ListResourcesResponse, error)
```

### Implementation Approach

1. **Pre-condition check**: Verify client is initialized
2. **Request preparation**: Marshal `ListResourcesRequest` (empty cursor initially)
3. **RPC call**: Invoke `c.call(ctx, "resources/list", params)`
4. **Response parsing**: Unmarshal result into `ListResourcesResponse`
5. **Error handling**: Wrap errors with operation context

### Error Handling

- `ErrNotInitialized`: Client not initialized before calling ListResources
- `Marshal errors`: Invalid request structure (unlikely with empty cursor)
- `Call errors`: Network, timeout, or server errors
- `Unmarshal errors`: Invalid response from server

### Type Dependencies

Uses existing types from `internal/mcp/types`:
- `ListResourcesRequest` (request.go:32-36)
- `ListResourcesResponse` (response.go:33-40)
- `Resource` (types.go:32-45)

## Implementation Plan

### Phase 1: Core Implementation
1. Implement `ListResources` method following `ListTools` pattern
2. Add initialization check
3. Implement JSON-RPC call to `resources/list`
4. Parse and return response

### Phase 2: Testing
1. Unit test: successful resource listing
2. Unit test: uninitialized client error
3. Unit test: server error handling
4. Unit test: response unmarshaling
5. Unit test: context cancellation
6. Integration test: real MCP server (if available)

### Phase 3: Documentation
1. Add method documentation
2. Update missing.md to mark as complete
3. Note any limitations or future enhancements

## Testing Strategy

### Unit Tests

```go
// Test cases:
TestListResources_Success              // Happy path with resources
TestListResources_EmptyList            // Server returns no resources
TestListResources_NotInitialized       // Error before initialize
TestListResources_ServerError          // Server returns error
TestListResources_InvalidResponse      // Malformed JSON response
TestListResources_ContextCancellation  // Context timeout/cancel
TestListResources_WithPagination       // Future: cursor support
```

### Test Data

```go
// Sample response from MCP server:
{
  "resources": [
    {
      "uri": "file:///path/to/document.txt",
      "name": "Document",
      "description": "A text document",
      "mimeType": "text/plain"
    }
  ],
  "nextCursor": null
}
```

## Dependencies

- Existing `types.ListResourcesRequest` and `types.ListResourcesResponse` types
- Existing JSON-RPC infrastructure in `StdioClient`
- No new external dependencies required

## Limitations

1. **Initial implementation**: Does not support pagination cursor parameter
2. **Server requirement**: Requires MCP server to implement resources capability
3. **Capability check**: Does not verify server capabilities before calling (consistent with ListTools)

## Future Enhancements

1. Add cursor parameter support for pagination
2. Add capability checking before method calls
3. Add caching of resource list if appropriate
4. Implement `ReadResource` method (separate FRD)

## Acceptance Criteria

- [ ] `ListResources` method implemented and follows existing patterns
- [ ] Initialization check prevents calls before initialize
- [ ] Proper error wrapping with operation context
- [ ] All unit tests pass (minimum 5 test cases)
- [ ] Code analysis with `uast parse | herr analyze` shows no issues
- [ ] Integration with existing codebase verified
- [ ] Documentation updated (method comments)
- [ ] missing.md updated to mark task complete

## References

- MCP Specification: https://modelcontextprotocol.io/specification
- Related: `ListTools` implementation in stdio.go:138-160
- Related: `CallTool` implementation in stdio.go:162-193
- Related Types: `internal/mcp/types/request.go`, `response.go`

---

**Implementation Date**: TBD
**Implemented By**: Claude (AI Agent)
**Reviewed By**: TBD
