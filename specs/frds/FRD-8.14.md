# FRD-8.14: MCP ReadResource Implementation

**Feature ID**: FRD-8.14
**Component**: MCP Client
**Status**: In Progress
**Priority**: Medium
**Created**: 2025-10-04

---

## Overview

This FRD specifies the implementation of the `ReadResource` method in the MCP (Model Context Protocol) client. This method allows reading the contents of a specific resource identified by its URI.

## Background

The MCP protocol supports resource management through two operations:
- `resources/list` - Lists available resources (✅ implemented in FRD-8.13)
- `resources/read` - Reads the contents of a specific resource (⏳ this FRD)

Currently, the `StdioClient.ReadResource()` method returns a "not implemented" error. This FRD addresses the implementation gap.

## Requirements

### Functional Requirements

1. **Resource Reading**
   - Implement `ReadResource(ctx context.Context, uri string)` method
   - Follow the same pattern as `ListResources` and other MCP methods
   - Support JSON-RPC call to `resources/read` method
   - Handle resource contents (text and/or binary blob)

2. **Error Handling**
   - Check if client is initialized before making the call
   - Handle marshaling/unmarshaling errors
   - Handle network/communication errors
   - Provide clear error messages with operation context

3. **Request/Response Processing**
   - Marshal `ReadResourceRequest` with the provided URI
   - Make JSON-RPC call using the `call()` method
   - Unmarshal `ReadResourceResponse` from the result
   - Support both text and binary (base64-encoded) content

### Non-Functional Requirements

1. **Code Quality**
   - Follow existing code patterns in `stdio.go`
   - Maintain consistency with `ListResources` implementation
   - Use proper error wrapping with `&Error{Op: ..., Err: ...}`
   - Keep cognitive complexity low (target: < 5)

2. **Testing**
   - Comprehensive unit tests covering all scenarios
   - Test initialization check
   - Test successful resource reading
   - Test request marshaling
   - Test response unmarshaling
   - Test error conditions
   - Test both text and binary content types

3. **Documentation**
   - Clear code comments following Go conventions
   - Update any relevant documentation

## Technical Design

### Implementation Pattern

Follow the established pattern from `ListResources`:

```go
func (c *StdioClient) ReadResource(ctx context.Context, uri string) (*types.ReadResourceResponse, error) {
	// 1. Check initialization
	if !c.initialized {
		return nil, &Error{Op: "ReadResource", Err: fmt.Errorf("client not initialized")}
	}

	// 2. Marshal request
	params, err := json.Marshal(types.ReadResourceRequest{URI: uri})
	if err != nil {
		return nil, &Error{Op: "ReadResource.Marshal", Err: err}
	}

	// 3. Make JSON-RPC call
	result, err := c.call(ctx, "resources/read", params)
	if err != nil {
		return nil, &Error{Op: "ReadResource.Call", Err: err}
	}

	// 4. Unmarshal response
	var resp types.ReadResourceResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, &Error{Op: "ReadResource.Unmarshal", Err: err}
	}

	return &resp, nil
}
```

### Data Structures

Existing types (already defined in `types` package):

```go
// ReadResourceRequest reads a specific resource.
type ReadResourceRequest struct {
	URI string `json:"uri"`
}

// ReadResourceResponse contains resource contents.
type ReadResourceResponse struct {
	Contents []ResourceContents `json:"contents"`
}

// ResourceContents represents the content of a resource.
type ResourceContents struct {
	URI      string  `json:"uri"`
	MimeType *string `json:"mimeType,omitempty"`
	Text     *string `json:"text,omitempty"`
	Blob     *string `json:"blob,omitempty"` // base64-encoded
}
```

## Test Cases

### Unit Tests

1. **TestReadResource_NotInitialized**
   - Verify error when client not initialized
   - Expected: Error with "client not initialized" message

2. **TestReadResourceRequest_Marshal**
   - Verify request marshaling with URI
   - Expected: Valid JSON with "uri" field

3. **TestReadResourceResponse_Unmarshal**
   - Test unmarshaling response with text content
   - Test unmarshaling response with binary blob
   - Test unmarshaling with mime type
   - Test unmarshaling multiple contents
   - Test unmarshaling empty contents

4. **TestResourceContents_AllFields**
   - Verify all ResourceContents fields are properly defined
   - Test text content scenario
   - Test binary blob scenario
   - Test with mime type

5. **Integration Test**
   - Add ReadResource test to existing `TestStdioClient_InitializeBeforeOtherCalls`
   - Verify it follows the initialization requirement

## Success Criteria

1. ✅ `ReadResource` method fully implemented
2. ✅ All unit tests pass (minimum 5 test cases)
3. ✅ Code analysis shows good quality metrics:
   - Cohesion: > 95%
   - Cognitive Complexity: < 5
   - Cyclomatic Complexity: < 3
4. ✅ No breaking changes to existing code
5. ✅ Consistent with existing MCP client patterns
6. ✅ Integration test passes

## Dependencies

- Existing MCP types in `internal/mcp/types/`
- `StdioClient.call()` method
- JSON-RPC infrastructure
- `ResourceContents` type already defined

## Implementation Notes

1. **URI Validation**: Consider if URI validation is needed (currently not done in ListResources)
2. **Content Type**: The response can contain multiple contents and both text/binary formats
3. **Error Wrapping**: Use consistent error wrapping pattern with operation name
4. **Context Support**: Method accepts context for cancellation/timeout support

## Related FRDs

- **FRD-8.13**: MCP ListResources Implementation (pattern reference)
- Related to MCP protocol resource management capabilities

## References

- MCP Protocol Specification (Model Context Protocol)
- Existing implementation in `internal/mcp/client/stdio.go`
- Type definitions in `internal/mcp/types/`

---

**Implementation Status**: ⏳ In Progress
**Target Completion**: 2025-10-04
