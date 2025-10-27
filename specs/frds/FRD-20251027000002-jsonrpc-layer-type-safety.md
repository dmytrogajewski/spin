# FRD-20251027000002: JSON-RPC Layer Type Safety

## Metadata
- **FRD ID**: FRD-20251027000002
- **Title**: JSON-RPC Layer Type Safety Improvements
- **Status**: Draft
- **Created**: 2025-10-27
- **Author**: Claude (Rob Pike persona)
- **Related Documents**: 
  - `specs/ifacesroadmap.md` - Phase 3.2
  - `docs/packages/protocol.md`
  - FRD-20251027000001 - Protocol Layer Typed Messages

## 1. Overview

### 1.1 Purpose
Eliminate `interface{}` usage in the JSON-RPC layer by:
1. Converting `InitializeParams.Config` from `map[string]interface{}` to `json.RawMessage`
2. Updating `Handler.HandleRequest` return type from `interface{}` to `json.RawMessage`

### 1.2 Scope
**In Scope:**
- `internal/protocol/jsonrpc/jsonrpc.go` - Type definitions
- `internal/protocol/jsonrpc/server.go` - Handler interface and server implementation
- `internal/appserver/handler.go` - Handler implementation
- `internal/appserver/processor.go` - Config handling
- Test files for all affected packages

**Out of Scope:**
- Changes to other protocol layers
- Backward compatibility (as per project requirements)
- Changes to application-level configuration

### 1.3 Background
The JSON-RPC layer currently uses `interface{}` in two locations:
1. `InitializeParams.Config map[string]interface{}` - Configuration parameters
2. `Handler.HandleRequest(...) (interface{}, error)` - Method return values

Both cases can be replaced with `json.RawMessage` for type safety while maintaining JSON flexibility.

## 2. Current State Analysis

### 2.1 Current Interface{} Usage

**Location 1: InitializeParams.Config**
```go
// internal/protocol/jsonrpc/jsonrpc.go:115
type InitializeParams struct {
    WorkspacePath string                 `json:"workspace_path"`
    Config        map[string]interface{} `json:"config,omitempty"`
}
```

**Usage in appserver/processor.go:103:**
```go
type Processor struct {
    // ...
    config map[string]interface{} // Runtime config overrides
}
```

**Location 2: Handler.HandleRequest return type**
```go
// internal/protocol/jsonrpc/server.go:10
type Handler interface {
    HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error)
}
```

**Implementation in appserver/handler.go:**
```go
func (h *Handler) HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
    switch method {
    case "initialize":
        // returns jsonrpc.InitializeResult (struct)
    case "send_message":
        // returns jsonrpc.SendMessageResult (struct)
    // ... all cases return specific struct types
    }
}
```

### 2.2 Problems with Current Approach

**InitializeParams.Config:**
- No type safety when accessing config values
- Easy to introduce runtime errors with wrong type assertions
- Difficult to validate at compile time
- No IDE support for config structure

**Handler.HandleRequest return type:**
- Forces all result types to be cast to `interface{}`
- Server code must marshal result back to JSON anyway
- Loss of type information between handler and server
- No compile-time guarantees about result structure

## 3. Requirements

### 3.1 Functional Requirements

**FR1: Config Type Safety**
- `InitializeParams.Config` must use `json.RawMessage` instead of `map[string]interface{}`
- Config must be parseable into structured types by consumers
- Empty/nil config must be valid

**FR2: Handler Return Type Safety**
- `Handler.HandleRequest` must return `json.RawMessage` instead of `interface{}`
- All handler implementations must return JSON-encoded results
- Error handling must remain unchanged

**FR3: Backward Compatibility**
- No backward compatibility required (per project guidelines)
- All code must be updated atomically

### 3.2 Non-Functional Requirements

**NFR1: Performance**
- No performance regression in JSON-RPC message handling
- Marshaling overhead must be minimal

**NFR2: Code Quality**
- 90%+ test coverage for modified code
- Zero lint errors
- Zero deadcode warnings
- All tests must pass

**NFR3: Maintainability**
- Clear documentation of type changes
- Type-safe helper methods where appropriate

## 4. Design Decision

### 4.1 Option Analysis

**Option A: Use `json.RawMessage` for both**
- ✅ Maintains JSON flexibility
- ✅ Delays parsing until needed
- ✅ Single marshaling step at handler level
- ✅ Consistent with other protocol layer usage
- ❌ Still requires unmarshaling by consumers

**Option B: Define strict struct types**
- ✅ Full compile-time type safety
- ❌ Loses JSON flexibility
- ❌ Breaks if config schema changes
- ❌ Multiple marshaling/unmarshaling steps

**Option C: Use generics**
- ✅ Type-safe at call site
- ❌ Complicates interface design
- ❌ Requires Go 1.18+
- ❌ Over-engineering for this use case

### 4.2 Selected Approach: Option A

**Rationale:**
1. **Consistency**: `json.RawMessage` is already used throughout the protocol layer (Request.Params, Response.Result, etc.)
2. **Simplicity**: Handler implementations already marshal their results - just return the marshaled bytes
3. **Flexibility**: Config can evolve without breaking the interface
4. **Performance**: One less marshal/unmarshal cycle in server.go

## 5. Implementation Plan

### 5.1 Phase 1: Type Definitions (jsonrpc.go)

**Change 1: InitializeParams.Config**
```go
// BEFORE
type InitializeParams struct {
    WorkspacePath string                 `json:"workspace_path"`
    Config        map[string]interface{} `json:"config,omitempty"`
}

// AFTER
type InitializeParams struct {
    WorkspacePath string          `json:"workspace_path"`
    Config        json.RawMessage `json:"config,omitempty"`
}
```

**Change 2: Add config helper methods (optional)**
```go
// ParseConfig unmarshals the config into a target struct
func (p *InitializeParams) ParseConfig(target interface{}) error {
    if len(p.Config) == 0 {
        return nil // Empty config is valid
    }
    return json.Unmarshal(p.Config, target)
}
```

### 5.2 Phase 2: Handler Interface (server.go)

**Change 3: Handler interface**
```go
// BEFORE
type Handler interface {
    HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error)
}

// AFTER
type Handler interface {
    HandleRequest(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error)
}
```

**Change 4: Server.Serve method**
```go
// BEFORE (server.go:54)
result, err := s.handler.HandleRequest(ctx, req.Method, req.Params)
// ...
if err != nil {
    // error handling
} else {
    // Success response
    resultJSON, _ := json.Marshal(result)  // <-- Extra marshal
    resp.Result = resultJSON
}

// AFTER
result, err := s.handler.HandleRequest(ctx, req.Method, req.Params)
// ...
if err != nil {
    // error handling (unchanged)
} else {
    // Success response - result is already JSON
    resp.Result = result
}
```

### 5.3 Phase 3: Handler Implementation (appserver/handler.go)

**Change 5: Update HandleRequest return values**
```go
// BEFORE
func (h *Handler) HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
    switch method {
    case "initialize":
        var p jsonrpc.InitializeParams
        if err := json.Unmarshal(params, &p); err != nil {
            return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
        }
        return h.processor.HandleInitialize(ctx, p)  // returns struct
    // ...
    }
}

// AFTER
func (h *Handler) HandleRequest(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
    switch method {
    case "initialize":
        var p jsonrpc.InitializeParams
        if err := json.Unmarshal(params, &p); err != nil {
            return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
        }
        result, err := h.processor.HandleInitialize(ctx, p)
        if err != nil {
            return nil, err
        }
        return json.Marshal(result)  // Marshal here instead of in server
    // ... repeat for all cases
    }
}
```

### 5.4 Phase 4: Processor Config Handling (appserver/processor.go)

**Change 6: Update Processor.config field**
```go
// BEFORE
type Processor struct {
    // ...
    config map[string]interface{} // Runtime config overrides
}

// AFTER
type Processor struct {
    // ...
    config json.RawMessage // Runtime config overrides
}
```

**Change 7: Update config usage in HandleInitialize**
```go
// Store config as-is (it's already json.RawMessage)
p.mu.Lock()
p.config = params.Config
p.mu.Unlock()
```

### 5.5 Phase 5: Tests

**Add tests for:**
1. `InitializeParams` with various config payloads (empty, null, complex)
2. `ParseConfig` helper method (if added)
3. Handler return type marshaling
4. Server handling of marshaled results
5. Error cases (invalid JSON in config, marshal failures)

**Update existing tests:**
1. `jsonrpc_test.go` - Config field tests
2. `server_test.go` - Handler result handling
3. `appserver/handler_test.go` - Return value marshaling (if exists)
4. `appserver/processor_test.go` - Config handling

## 6. Testing Strategy

### 6.1 Unit Tests

**Test Coverage Targets:**
- `jsonrpc.go`: 90%+ (up from current coverage)
- `server.go`: 90%+ (maintain or improve)
- `handler.go`: 90%+ (new tests for marshaling)

**Key Test Cases:**

```go
// Test: InitializeParams with json.RawMessage config
func TestInitializeParams_WithConfig(t *testing.T) {
    params := InitializeParams{
        WorkspacePath: "/workspace",
        Config:        json.RawMessage(`{"key":"value"}`),
    }
    // Test marshaling/unmarshaling
    // Test ParseConfig helper
}

// Test: Handler returns json.RawMessage
func TestHandler_ReturnsJSONRawMessage(t *testing.T) {
    // Mock processor returns struct
    // Handler should marshal to json.RawMessage
    // Verify result is valid JSON
}

// Test: Server handles json.RawMessage result
func TestServer_HandlesJSONRawMessageResult(t *testing.T) {
    // Handler returns json.RawMessage
    // Server should use it directly without re-marshaling
}
```

### 6.2 Integration Tests

**Scenarios:**
1. Full initialize flow with config
2. Send message flow with result marshaling
3. Error handling (marshal failures)

## 7. Migration Path

### 7.1 Steps (in order)

1. ✅ Read and understand current implementation
2. ✅ Write FRD document
3. Update `jsonrpc.go` type definitions
4. Update `server.go` interface and implementation
5. Update `handler.go` implementation
6. Update `processor.go` config handling
7. Write new tests
8. Update existing tests
9. Run `make lint` and fix issues
10. Run full test suite
11. Update documentation
12. Update roadmap

### 7.2 Rollback Plan

Since backward compatibility is not required:
- If issues found, revert entire commit atomically
- All changes are in internal packages
- No external API surface affected

## 8. Success Criteria

### 8.1 Acceptance Criteria

- [ ] No `interface{}` in `InitializeParams.Config`
- [ ] No `interface{}` in `Handler.HandleRequest` return type
- [ ] All tests pass (42 packages)
- [ ] Test coverage ≥90% for modified files
- [ ] Zero lint errors
- [ ] Zero deadcode warnings
- [ ] Documentation updated

### 8.2 Metrics

**Before:**
- `interface{}` occurrences in jsonrpc package: 2
- Lines of code: ~200

**After (Target):**
- `interface{}` occurrences in jsonrpc package: 0
- Lines of code: ~210 (slight increase for marshaling)
- Performance: No regression (less marshaling overhead)

## 9. Risks and Mitigations

### 9.1 Risks

**R1: Marshal failures in handlers**
- **Impact**: Runtime errors if result structs can't be marshaled
- **Mitigation**: All result types are simple structs with json tags (already tested)
- **Likelihood**: Low

**R2: Config parsing breaks**
- **Impact**: Processor can't read config
- **Mitigation**: Config is optional, empty config is valid
- **Likelihood**: Low

**R3: Test coverage gaps**
- **Impact**: Bugs slip through
- **Mitigation**: Write comprehensive tests before implementation
- **Likelihood**: Low

### 9.2 Assumptions

1. All handler methods return JSON-marshalable structs
2. Config is optional and can be empty
3. No external code depends on these internal types
4. Performance impact of marshaling in handlers is negligible

## 10. Documentation Updates

### 10.1 Files to Update

- `docs/packages/protocol.md` - Add JSON-RPC type safety section
- `specs/ifacesroadmap.md` - Mark Phase 3.2 complete
- `internal/protocol/jsonrpc/doc.go` - Update package docs (if exists)

### 10.2 Code Comments

Add comments explaining:
- Why `json.RawMessage` is used for config
- Why handlers return `json.RawMessage`
- How to parse config in processor

## 11. References

### 11.1 Related FRDs
- FRD-20251027000001 - Protocol Layer Typed Messages (Phase 3.1)
- FRD-20251026000001 - MCP Go SDK Migration (Phase 2.4)
- FRD-20251026-event-system-generics - Event System (Phase 1.2)

### 11.2 External References
- Go `encoding/json` documentation
- JSON-RPC 2.0 Specification
- Go Best Practices: Effective Go

## 12. Appendix

### 12.1 Interface{} Elimination Count

**This FRD eliminates:**
- `InitializeParams.Config`: 1 occurrence
- `Handler.HandleRequest` return: 1 occurrence
- `Processor.config`: 1 occurrence
- **Total: 3 occurrences**

**Progress:**
- Phase 3.1: Eliminated 9 occurrences (protocol.go parsers)
- Phase 3.2: Eliminating 3 occurrences (this FRD)
- **Phase 3 total: 12 occurrences eliminated**

### 12.2 Implementation Checklist

- [ ] Update `InitializeParams.Config` type
- [ ] Update `Handler.HandleRequest` return type
- [ ] Update `Server.Serve` result handling
- [ ] Update `appserver.Handler.HandleRequest` implementation
- [ ] Update `Processor.config` field
- [ ] Write tests for config parsing
- [ ] Write tests for result marshaling
- [ ] Update existing tests
- [ ] Run `make lint`
- [ ] Run full test suite
- [ ] Update documentation
- [ ] Update roadmap

---

**Document Version**: 1.0
**Last Updated**: 2025-10-27
**Status**: Ready for Implementation
