# FRD: Event System - Type Safety Improvements

**Feature ID**: Phase 1.2 - Event System  
**Priority**: P0  
**Status**: In Progress (Revised Approach)  
**Created**: 2025-10-26  
**Revised**: 2025-10-26  
**Author**: Claude (Rob Pike persona)  
**Related**: [Empty Interface Elimination Roadmap](../ifacesroadmap.md#12-event-system-priority-p0)

---

## Executive Summary

**REVISED APPROACH**: After analyzing the codebase, this FRD proposes keeping `Event.Data` as `interface{}` (idiomatic for heterogeneous event streams) while improving type safety through:
1. Type-safe helper methods on Event struct
2. Strongly-typed payload structs (already exist)
3. Fixing `detection.event` to use specific `DetectionEventData` type
4. Adding Event.Data to roadmap's "Keep As-Is" section

**Scope**: 
- `internal/events/event.go` - Add type-safe helper methods
- `internal/detection/detection.go` - Replace interface{} with DetectionEventData
- `specs/ifacesroadmap.md` - Update to reflect idiomatic Go pattern

**Out of Scope**:
- Making Event generic (conflicts with heterogeneous event stream pattern)
- Event emission logic (no changes)
- Event subscription system (no changes)
- Event backpressure strategies (no changes)

---

## Business Context

### Problem Statement

The current event system has two issues:
1. **Event.Data interface{}** - Actually idiomatic for heterogeneous event streams, but lacks type-safe accessor methods
2. **detection.event.data interface{}** - Unnecessary; detection events always use map[string]interface{}, should be strongly typed

Consumers currently:
1. Perform runtime type assertions with potential panics
2. Lack helpful accessor methods for common patterns
3. Have to write repetitive type assertion code

### Current State Analysis

**File: internal/events/event.go (Lines 18, 32)**
```go
type Event struct {
    Type      EventType   `json:"type"`
    Timestamp time.Time   `json:"timestamp"`
    Data      interface{} `json:"data"`  // ← interface{}
}

func (e Event) GetData() interface{} {  // ← interface{}
    return e.Data
}
```

**File: internal/detection/detection.go (Line 91)**
```go
type event struct {
    eventType string
    timestamp time.Time
    data      interface{}  // ← interface{}
}

func (e *event) GetData() interface{} {  // ← interface{}
    return e.data
}
```

**Current Usage Patterns:**
```go
// Consumer code - requires type assertions
for event := range events {
    switch event.Type {
    case EventToolCallStart:
        data := event.Data.(ToolCallStartData)  // Runtime assertion
        // Use data...
    }
}
```

### Desired State

**Keep Event.Data as interface{} (Idiomatic Go):**
```go
type Event struct {
    Type      EventType   `json:"type"`
    Timestamp time.Time   `json:"timestamp"`
    Data      interface{} `json:"data"`  // ← Idiomatic for heterogeneous streams
}

// Add type-safe helper methods
func (e Event) ToolCallStartData() (ToolCallStartData, bool) {
    data, ok := e.Data.(ToolCallStartData)
    return data, ok
}

func (e Event) ContentDeltaData() (ContentDeltaData, bool) {
    data, ok := e.Data.(ContentDeltaData)
    return data, ok
}
// ... etc for each event type
```

**Fix detection.event to use specific type:**
```go
// Before
type event struct {
    eventType string
    timestamp time.Time
    data      interface{}  // ← Unnecessary interface{}
}

// After
type DetectionEventData map[string]interface{}

type event struct {
    eventType string
    timestamp time.Time
    data      DetectionEventData  // ← Specific type
}
```

**Type-Safe Usage:**
```go
// Consumer code - cleaner with helper methods
for event := range events {
    switch event.Type {
    case EventToolCallStart:
        if data, ok := event.ToolCallStartData(); ok {
            // Use data with autocomplete
        }
    }
}
```

### Benefits

1. **Compile-Time Safety**: Type errors caught at compile time, not runtime
2. **IDE Support**: Full autocomplete and type inference for event data
3. **No Runtime Panics**: Eliminates type assertion failures
4. **Self-Documenting**: Event signatures clearly show data types
5. **Refactoring Safety**: Breaking changes detected at compile time
6. **Performance**: Slightly faster (no reflection/type assertions)

---

## Requirements

### Functional Requirements

#### FR1: Generic Event Type
**Priority**: P0  
**Description**: Define `Event[T any]` generic struct

**Acceptance Criteria**:
- Event struct accepts type parameter `T any`
- GetData() returns type `T` instead of `interface{}`
- JSON marshaling/unmarshaling works correctly
- Backward compatible with cycle.Event interface

**Implementation**:
```go
type Event[T any] struct {
    Type      EventType `json:"type"`
    Timestamp time.Time `json:"timestamp"`
    Data      T         `json:"data"`
}

func (e Event[T]) GetData() T {
    return e.Data
}
```

#### FR2: Type-Safe Event Constructors
**Priority**: P0  
**Description**: Create helper constructors for each event type

**Acceptance Criteria**:
- Constructor for each EventType with appropriate data type
- Type inference works automatically
- Clear, self-documenting signatures

**Example**:
```go
func NewContentDeltaEvent(content, role string) Event[ContentDeltaData] {
    return Event[ContentDeltaData]{
        Type: EventContentDelta,
        Timestamp: time.Now(),
        Data: ContentDeltaData{Content: content, Role: role},
    }
}
```

#### FR3: Update Detection Event Types
**Priority**: P0  
**Description**: Convert detection.event to use generics

**Acceptance Criteria**:
- detection.event struct updated to generic
- GetData() returns typed data
- All usages updated

#### FR4: Maintain Interface Compatibility
**Priority**: P0  
**Description**: Ensure compatibility with existing cycle.Event interface

**Acceptance Criteria**:
- Event[T] implements cycle.Event interface methods
- GetType(), GetTimestamp(), GetData() all work
- No breaking changes to interface contracts

### Non-Functional Requirements

#### NFR1: Test Coverage
**Priority**: P0  
**Target**: 90%+ coverage

**Requirements**:
- Unit tests for generic Event[T] creation
- Tests for each typed event constructor
- Tests for JSON marshaling/unmarshaling with generics
- Tests for interface compatibility
- Tests for type safety (compile-time)

#### NFR2: Performance
**Priority**: P1  
**Target**: No regression

**Requirements**:
- Benchmark generic vs interface{} approach
- Ensure no allocation increase
- Verify negligible overhead (<5%)

#### NFR3: Documentation
**Priority**: P0  

**Requirements**:
- Godoc for generic Event[T] type
- Examples of typed event usage
- Migration guide from interface{} to generics
- Update architecture documentation

---

## Technical Design

### Architecture

**Component Diagram**:
```
┌─────────────────────────────────────────┐
│         Event Producer (Agent)          │
│  - Emits Event[ContentDeltaData]        │
│  - Emits Event[ToolCallStartData]       │
│  - Emits Event[TurnEventData]           │
└───────────────┬─────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────┐
│         EventEmitter (Generic)          │
│  - Subscribe() → chan Event[T]          │
│  - Emit(Event[T])                       │
│  - Type-safe distribution               │
└───────────────┬─────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────┐
│        Event Consumers (UI/TUI)         │
│  - Receive Event[T]                     │
│  - Access Data without type assertion   │
│  - Compile-time type safety             │
└─────────────────────────────────────────┘
```

### Data Model

**Before (Current)**:
```go
Event {
    Type: EventToolCallStart,
    Data: interface{} -> ToolCallStartData (runtime assertion)
}
```

**After (Generic)**:
```go
Event[ToolCallStartData] {
    Type: EventToolCallStart,
    Data: ToolCallStartData (compile-time typed)
}
```

### Implementation Plan

#### Phase 1: Core Generic Type (2 hours)
1. Define `Event[T any]` struct
2. Update `GetData() T` method
3. Implement JSON marshaling tests
4. Verify interface compatibility

#### Phase 2: Event Constructors (1 hour)
1. Create typed constructor for each event type:
   - `NewContentDeltaEvent()`
   - `NewToolCallStartEvent()`
   - `NewToolCallCompleteEvent()`
   - `NewTurnEvent()`
   - `NewApprovalEvent()`
   - `NewErrorEvent()`

#### Phase 3: Detection Events (1 hour)
1. Update `detection.event` to generic
2. Update `detection.Event` interface
3. Update EscalateIntervention usage

#### Phase 4: Testing (2 hours)
1. Unit tests for generic Event[T]
2. Tests for each constructor
3. JSON serialization tests
4. Interface compatibility tests
5. Achieve 90%+ coverage

#### Phase 5: Documentation (1 hour)
1. Update godoc
2. Add usage examples
3. Update architecture docs

**Total Estimated Time**: 7 hours

---

## Testing Strategy

### Unit Tests

**Test Cases**:

1. **Generic Event Creation**
   ```go
   func TestEvent_Generic(t *testing.T) {
       event := Event[ContentDeltaData]{
           Type: EventContentDelta,
           Data: ContentDeltaData{Content: "test", Role: "assistant"},
       }
       assert.Equal(t, "test", event.GetData().Content)
   }
   ```

2. **Type Safety**
   ```go
   func TestEvent_TypeSafety(t *testing.T) {
       event := NewContentDeltaEvent("test", "assistant")
       data := event.GetData() // No type assertion
       assert.Equal(t, "test", data.Content)
   }
   ```

3. **JSON Marshaling**
   ```go
   func TestEvent_JSONMarshaling(t *testing.T) {
       event := NewToolCallStartEvent("read_file", "call_1", params)
       bytes, err := json.Marshal(event)
       assert.NoError(t, err)
       
       var decoded Event[ToolCallStartData]
       err = json.Unmarshal(bytes, &decoded)
       assert.NoError(t, err)
       assert.Equal(t, "read_file", decoded.Data.ToolName)
   }
   ```

4. **Interface Compatibility**
   ```go
   func TestEvent_InterfaceCompatibility(t *testing.T) {
       var cycleEvent cycle.Event
       cycleEvent = NewContentDeltaEvent("test", "assistant")
       
       assert.Equal(t, "content_delta", cycleEvent.GetType())
       assert.NotZero(t, cycleEvent.GetTimestamp())
       assert.NotNil(t, cycleEvent.GetData())
   }
   ```

### Integration Tests

1. **Event Emission Flow**
   - Emit typed events through EventEmitter
   - Verify subscribers receive correctly typed events
   - Test with multiple event types

2. **Detection Service Integration**
   - Verify detection events work with generics
   - Test EscalateIntervention with typed events

### Performance Tests

```go
func BenchmarkEvent_Generic(b *testing.B) {
    for i := 0; i < b.N; i++ {
        event := NewContentDeltaEvent("test", "assistant")
        _ = event.GetData()
    }
}

func BenchmarkEvent_Interface(b *testing.B) {
    for i := 0; i < b.N; i++ {
        event := Event{Type: EventContentDelta, Data: ContentDeltaData{}}
        _ = event.Data.(ContentDeltaData)
    }
}
```

**Expected Result**: Generic approach ≤5% overhead vs interface{}

---

## Migration Strategy

### Backward Compatibility

**Approach**: Two-phase migration

**Phase 1**: Keep both implementations
- Add `Event[T]` alongside existing `Event`
- Gradually migrate consumers
- Maintain interface compatibility

**Phase 2**: Remove old implementation
- After all consumers migrated
- Remove interface{} version
- Update all references

### Breaking Changes

**None** - This is an internal refactoring:
- Event emission API unchanged
- Event subscription API unchanged
- Only internal type signatures change

### Rollout Plan

1. **Week 1**: Implement generic Event[T]
2. **Week 1**: Add tests (90%+ coverage)
3. **Week 1**: Update detection events
4. **Week 2**: Migrate consumers (if needed)
5. **Week 2**: Documentation update

---

## Risk Assessment

### Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| JSON unmarshaling issues with generics | Low | Medium | Extensive JSON tests |
| Performance regression | Low | Low | Benchmark tests |
| Interface compatibility issues | Low | Medium | Interface tests |
| Complex type inference | Low | Low | Explicit type constructors |

### Mitigation Strategies

1. **JSON Issues**: Test all event types with real-world payloads
2. **Performance**: Benchmark before/after, accept <5% overhead
3. **Compatibility**: Maintain cycle.Event interface contract
4. **Type Inference**: Provide helper constructors for clarity

---

## Success Metrics

### Code Quality Metrics

- ✅ Zero `interface{}` in event data fields
- ✅ 90%+ test coverage for event system
- ✅ All lint checks pass
- ✅ No cyclomatic complexity increase
- ✅ Zero deadcode in event system

### Performance Metrics

- ✅ Event emission latency unchanged (<5% difference)
- ✅ Memory allocations unchanged
- ✅ No goroutine leaks in event system

### Developer Experience Metrics

- ✅ IDE autocomplete works for event.GetData()
- ✅ Compile-time type errors for mismatched data
- ✅ Zero runtime type assertion panics
- ✅ Clear, self-documenting event constructors

---

## Dependencies

### Internal Dependencies

- `internal/tools` - ToolParameters type (already completed in Phase 1.1)
- `internal/detection` - Event interface compatibility

### External Dependencies

- Go 1.24 (generics support)
- `github.com/google/uuid` (no changes)
- `encoding/json` (standard library)

---

## Open Questions

1. **Q**: Should EventEmitter also be generic?  
   **A**: No - EventEmitter can remain non-generic and accept any `Event[T]`. The channel can use `interface{}` internally but consumers get typed events.

2. **Q**: How to handle event channel types?  
   **A**: Use `chan Event[any]` or keep as `chan Event` with interface{} data, then provide typed extractors.

3. **Q**: Impact on existing subscribers?  
   **A**: Minimal - subscribers already do type assertions, just move to typed constructors.

---

## References

### Related Documents

- [Empty Interface Elimination Roadmap](../ifacesroadmap.md)
- [Phase 1.1 Tool Parameters FRD](./FRD-20251026-tool-parameters.md)
- [AGENTS.md](../../AGENTS.md)

### Code References

- `internal/events/event.go:18` - Current interface{} usage
- `internal/events/event.go:32` - GetData() method
- `internal/detection/detection.go:91` - Detection event interface{}

### External References

- [Go Generics Tutorial](https://go.dev/doc/tutorial/generics)
- [Effective Go - Generics](https://go.dev/blog/intro-generics)

---

## Approval

**Status**: Draft → Ready for Implementation

**Stakeholders**:
- [ ] Architecture Review: Approved by Rob Pike persona
- [ ] Implementation: Claude (agent)
- [ ] Testing: 90%+ coverage required
- [ ] Documentation: Update docs/ and AGENTS.md

---

## Changelog

- **2025-10-26**: Initial FRD created for Phase 1.2 Event System
- **2025-10-26**: Defined generic Event[T] approach with type-safe constructors

---

**Next Steps**:
1. Implement `Event[T any]` generic type
2. Write comprehensive tests (target: 90%+ coverage)
3. Update detection events
4. Run `make lint` and fix all issues
5. Update roadmap (mark Phase 1.2 complete)
6. Update documentation in docs/
