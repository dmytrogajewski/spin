# FRD-20251028000002: UI Blocks Metadata Type Safety

## Metadata
- **Status**: Draft
- **Author**: Rob Pike (Claude)
- **Created**: 2025-10-28
- **Updated**: 2025-10-28
- **Related**: 
  - `specs/ifacesroadmap.md` - Empty interface elimination roadmap (Phase 5.1)
  - `docs/packages/ui.md` - UI blocks documentation

## Problem Statement

The UI blocks system currently uses `map[string]interface{}` for metadata storage in `Block.Meta`, which has several issues:

### Current Issues

1. **Type Unsafety**:
   ```go
   type Block struct {
       Meta map[string]interface{} `json:"meta"`  // Can store any type
   }
   ```
   - No compile-time type checking
   - Runtime type assertions required
   - Easy to store wrong types

2. **Inefficient Double-Marshaling**:
   ```go
   // Current pattern in SetExecuteMeta:
   data, _ := json.Marshal(m)        // Marshal struct → JSON
   var meta map[string]interface{}
   json.Unmarshal(data, &meta)       // Unmarshal JSON → map  
   b.Meta = meta                     // Store map
   ```
   - Marshal struct to JSON bytes
   - Immediately unmarshal JSON bytes to map
   - Wastes CPU cycles and allocations

3. **Inconsistent Access Patterns**:
   ```go
   // GetMeta returns interface{} - requires type assertion
   func (b *Block) GetMeta(key string) (interface{}, bool)
   
   // SetMeta accepts interface{} - no validation
   func (b *Block) SetMeta(key string, value interface{})
   ```

### Well-Defined Metadata Structures Exist

The codebase already has properly-typed metadata structs:
- `ExecuteMeta` - Command execution metadata
- `ReadMeta` - File read metadata  
- `GrepMeta` - Search metadata
- `ToolMeta` - Tool execution metadata
- `PatchMeta` - Patch application metadata
- `PlanMeta` - Planning metadata

These structs have validation, proper types, and are the source of truth.

## Goals

### Primary Goals
1. **Eliminate `interface{}`**: Use `json.RawMessage` for metadata storage
2. **Reduce Allocations**: Store JSON directly, no double-marshaling
3. **Type Safety**: Use typed structs for all access
4. **Backward Compatible JSON**: JSON serialization unchanged

### Non-Goals
- Changing metadata struct definitions (already well-designed)
- Modifying JSON wire format
- Adding new metadata types

## Proposed Solution

### Change Meta Field Type

```go
type Block struct {
    ID        string          `json:"id"`
    Type      BlockType       `json:"type"`
    Title     string          `json:"title,omitempty"`
    Meta      json.RawMessage `json:"meta"`           // Changed from map[string]interface{}
    Body      string          `json:"body"`
    FoldState FoldState       `json:"fold_state"`
    Severity  Severity        `json:"severity"`
    Timestamp int64           `json:"timestamp"`
    CompletionPrinted bool    `json:"-"`
}
```

### Update Helper Methods

**Remove Old Generic Methods**:
```go
// DELETE these methods:
func (b *Block) GetMeta(key string) (interface{}, bool)  // ❌ Delete
func (b *Block) SetMeta(key string, value interface{})   // ❌ Delete
```

**Add Type-Safe Accessors**:
```go
// Add these methods:
func (b *Block) GetExecuteMeta() (*ExecuteMeta, error)
func (b *Block) GetReadMeta() (*ReadMeta, error)
func (b *Block) GetGrepMeta() (*GrepMeta, error)
func (b *Block) GetToolMeta() (*ToolMeta, error)
func (b *Block) GetPatchMeta() (*PatchMeta, error)
func (b *Block) GetPlanMeta() (*PlanMeta, error)

func (b *Block) SetExecuteMeta(m *ExecuteMeta) error
func (b *Block) SetReadMeta(m *ReadMeta) error
func (b *Block) SetGrepMeta(m *GrepMeta) error
func (b *Block) SetToolMeta(m *ToolMeta) error
func (b *Block) SetPatchMeta(m *PatchMeta) error
func (b *Block) SetPlanMeta(m *PlanMeta) error
```

### Update metadata.go

**Simplify Set Functions** (eliminate double-marshaling):
```go
// BEFORE (inefficient):
func SetExecuteMeta(b *Block, m *ExecuteMeta) error {
    if err := m.Validate(); err != nil {
        return err
    }
    data, _ := json.Marshal(m)               // Step 1: Marshal to JSON
    var meta map[string]interface{}
    json.Unmarshal(data, &meta)              // Step 2: Unmarshal to map
    b.Meta = meta                            // Step 3: Store map
    return nil
}

// AFTER (efficient):
func SetExecuteMeta(b *Block, m *ExecuteMeta) error {
    if err := m.Validate(); err != nil {
        return err
    }
    data, err := json.Marshal(m)             // Marshal once
    if err != nil {
        return fmt.Errorf("marshal failed: %w", err)
    }
    b.Meta = data                            // Store JSON directly
    return nil
}
```

**Simplify Parse Functions**:
```go
// BEFORE:
func ParseExecuteMeta(b *Block) (*ExecuteMeta, error) {
    data, _ := json.Marshal(b.Meta)          // Re-marshal map
    var meta ExecuteMeta
    json.Unmarshal(data, &meta)              // Unmarshal
    return &meta, nil
}

// AFTER:
func ParseExecuteMeta(b *Block) (*ExecuteMeta, error) {
    var meta ExecuteMeta
    if err := json.Unmarshal(b.Meta, &meta); err != nil {
        return nil, fmt.Errorf("unmarshal failed: %w", err)
    }
    return &meta, nil
}
```

### Update NewBlock Constructor

```go
// BEFORE:
func NewBlock(blockType BlockType) *Block {
    return &Block{
        ID:        GenerateBlockID(0),
        Type:      blockType,
        Meta:      make(map[string]interface{}),  // Empty map
        // ...
    }
}

// AFTER:
func NewBlock(blockType BlockType) *Block {
    return &Block{
        ID:        GenerateBlockID(0),
        Type:      blockType,
        Meta:      nil,                           // nil (or []byte("{}"))
        // ...
    }
}
```

## Implementation Plan

### Phase 1: Update Core Types
1. Change `Block.Meta` from `map[string]interface{}` to `json.RawMessage`
2. Update `NewBlock()` to initialize `Meta` as `nil`
3. Remove `GetMeta()` and `SetMeta()` methods
4. Add type-safe accessor methods to `model.go`

### Phase 2: Update metadata.go
1. Simplify all `SetXxxMeta()` functions (remove double-marshaling)
2. Simplify all `ParseXxxMeta()` functions (direct unmarshal)
3. Keep validation logic intact

### Phase 3: Update Tests
1. Update `model_test.go` to use new accessors
2. Update `renderer_bench_test.go` metadata initialization
3. Update `renderer_tool_test.go` metadata usage
4. Ensure 90%+ test coverage

### Phase 4: Update Adapters (if needed)
1. Check `internal/ui/adapters/puretty.go` for metadata usage
2. Update any direct `Meta` field access to use type-safe methods

## Benefits

### Performance
- **Eliminate double-marshaling**: ~50% reduction in JSON operations
- **Reduce allocations**: No intermediate map allocations
- **Smaller memory footprint**: `[]byte` vs `map[string]interface{}`

### Type Safety
- **Compile-time checking**: Can't call wrong getter for block type
- **No type assertions**: All access through typed structs
- **Validation enforced**: Can't set invalid metadata

### Code Quality
- **Clearer intent**: `GetExecuteMeta()` vs `GetMeta("command")`
- **IDE support**: Auto-complete for metadata fields
- **Easier refactoring**: Change struct, compiler finds all usage

## Testing Strategy

### Unit Tests
1. **Type Safety Tests**:
   - Set ExecuteMeta, get ExecuteMeta → success
   - Set ExecuteMeta, get ReadMeta → error
   - Set invalid metadata → validation error

2. **JSON Compatibility Tests**:
   - Marshal block → JSON unchanged
   - Unmarshal old JSON → works with new code

3. **Accessor Tests**:
   - All 6 metadata types: Get/Set round-trip
   - Nil metadata → appropriate error
   - Invalid JSON → appropriate error

### Benchmark Tests
1. Compare old vs new SetMeta performance
2. Compare old vs new ParseMeta performance
3. Verify no performance regression

## Migration Notes

### Breaking Changes
- `GetMeta(key string)` removed → use `GetXxxMeta()` methods
- `SetMeta(key, value)` removed → use `SetXxxMeta()` methods

### Non-Breaking
- JSON wire format unchanged
- Metadata struct types unchanged
- Block serialization/deserialization unchanged

## Interface{} Elimination

**Before**: ~11 occurrences in ui/blocks package
- `Block.Meta` field (1)
- `GetMeta()` return type (1)
- `SetMeta()` parameter (1)
- `SetXxxMeta()` intermediate map (6)
- `ParseXxxMeta()` re-marshaling (6)
- Test usage (~5)

**After**: 0 occurrences
- All metadata accessed through typed structs
- No intermediate conversions
- `json.RawMessage` used for storage

## Success Criteria

- [ ] `Block.Meta` is `json.RawMessage`
- [ ] No `interface{}` in blocks package
- [ ] All tests pass (90%+ coverage)
- [ ] No performance regression
- [ ] JSON compatibility maintained
- [ ] Zero lint errors
- [ ] Documentation updated

## References

- Go JSON best practices: https://go.dev/blog/json
- json.RawMessage documentation: https://pkg.go.dev/encoding/json#RawMessage
- FRD-20251027000004 - Orchestration metadata (similar pattern)
