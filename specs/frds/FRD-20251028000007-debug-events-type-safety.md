# FRD-20251028000007: Debug Events Type Safety

**Status**: Complete  
**Created**: 2025-10-28  
**Component**: internal/debug  
**Related**: Phase 6.4 - Empty Interface Elimination

## Problem Statement

The debug events logger uses `map[string]interface{}` for JSON output structure, which loses type safety. This occurs in:

1. `events.go:123` - JSON output structure in `logEventJSON`
2. `events_test.go` - Test data structures using `map[string]interface{}`

## Current Implementation

```go
// events.go:123
output := map[string]interface{}{
    "timestamp": timestamp,
    "type":      event.Type,
    "data":      json.RawMessage(data),
}
```

## Proposed Solution

### Define EventLogOutput struct

Replace the map with a properly typed struct:

```go
// EventLogOutput represents a structured event log entry
type EventLogOutput struct {
    Timestamp string          `json:"timestamp"`
    Type      events.EventType `json:"type"`
    Data      json.RawMessage `json:"data"`
}
```

### Update logEventJSON

```go
func (el *EventLogger) logEventJSON(timestamp string, event events.Event) {
    data, err := json.Marshal(event.Data)
    if err != nil {
        data = []byte("{}")
    }

    output := EventLogOutput{
        Timestamp: timestamp,
        Type:      event.Type,
        Data:      json.RawMessage(data),
    }

    encoded, err := json.Marshal(output)
    if err != nil {
        fmt.Fprintf(el.writer, `{"error": "failed to encode event"}`+"\n")
        return
    }

    fmt.Fprintf(el.writer, "%s\n", encoded)
}
```

### Update Tests

Update test files to use proper typed structures or json.RawMessage instead of `map[string]interface{}` where appropriate. For test event data that truly has varying structure, keep using the event system's typed structures.

## Benefits

1. **Type Safety**: Compiler enforces correct field types
2. **IDE Support**: Auto-completion for output structure fields
3. **Documentation**: Struct clearly documents output format
4. **Consistency**: Matches pattern from other phases (shell, git integration)

## Interface{} Eliminated

- 1 occurrence in production code (events.go:123)
- ~5 occurrences in test code (test data structures)

## Testing Strategy

- Verify all existing tests pass
- Ensure JSON output format unchanged
- Test error handling paths (invalid data marshaling)
- Run with real events to verify output

## Migration Impact

- **Breaking Changes**: None - internal implementation only
- **API Compatibility**: No changes to public API
- **Performance**: No impact (same marshaling operations)

## Implementation Checklist

- [x] Define EventLogOutput struct
- [x] Update logEventJSON to use struct
- [x] Update tests to use typed event data structures (ContentDeltaData, ToolCallStartData)
- [x] Run all tests - all 15 tests pass
- [x] Run go vet - clean
- [x] Update documentation
- [x] Update roadmap

## Implementation Results

**Files Modified**:
- `internal/debug/events.go` - Added EventLogOutput struct, updated logEventJSON (+13 lines)
- `internal/debug/events_test.go` - Updated 4 tests to use typed event data (-12 lines of interface{})

**Interface{} Eliminated**: 6 occurrences total
- 1 occurrence in production code (events.go:123)
- 5 occurrences in test code (replaced with typed structures)

**Test Results**: All 15 tests pass, 55.3% coverage

**Pattern Applied**: Defined typed struct for JSON output, matching patterns from Phase 4.2 (shell) and Phase 6.2 (git)
