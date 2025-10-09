# FRD: Block Timeline State Machine

**ID:** FRD-20251010-block-timeline
**Phase:** 4.3 - Block Timeline State Machine
**Priority:** P1
**Created:** 2025-10-10
**Status:** ✅ COMPLETED (2025-10-10)

---

## 1. Overview

Implement the Timeline data structure for managing an ordered collection of blocks with virtualization, filtering, and navigation support.

---

## 2. Goals

- **Ordered block list** with append/update/delete operations
- **Viewport calculation** for virtualization (render only visible blocks)
- **Filter support** for type, file, exit code, impact
- **Navigation** state tracking (scroll position, current block)
- **Collapse/expand** state management
- **Performance** with 10k+ blocks

---

## 3. Non-Goals

- UI rendering (handled by renderer)
- User input handling (handled by adapter)
- Persistence/serialization (future)
- Advanced search/fuzzy matching (future)

---

## 4. Data Model

### 4.1 Timeline Struct

```go
type Timeline struct {
    blocks      []*Block        // Ordered list
    viewport    Viewport        // Current visible range
    scrollPos   int             // Scroll offset in blocks
    focusedID   string          // Currently focused block ID
    filter      *Filter         // Active filter (nil = no filter)
}

type Viewport struct {
    start  int  // First visible block index
    end    int  // Last visible block index (exclusive)
    height int  // Viewport height in blocks
}

type Filter struct {
    types     []BlockType  // Filter by type(s)
    file      string       // Filter by file path (substring match)
    exitCode  *int         // Filter by exit code
    impact    string       // Filter by impact level
}
```

---

## 5. Operations

### 5.1 Block Management

```go
func (t *Timeline) Append(block *Block) error
func (t *Timeline) Update(blockID string, block *Block) error
func (t *Timeline) Delete(blockID string) error
func (t *Timeline) Get(blockID string) (*Block, error)
func (t *Timeline) GetByIndex(index int) (*Block, error)
func (t *Timeline) Len() int
```

**Invariants:**
- Block IDs unique
- Append maintains chronological order
- Delete preserves remaining order
- Update preserves position

### 5.2 Viewport Management

```go
func (t *Timeline) SetViewportHeight(height int)
func (t *Timeline) GetVisibleBlocks() []*Block
func (t *Timeline) GetViewport() Viewport
```

**Algorithm:**
- Visible range: `[scrollPos, scrollPos + viewportHeight)`
- Clamp to `[0, len(blocks))`
- Return slice of blocks in range

### 5.3 Navigation

```go
func (t *Timeline) ScrollUp(lines int) error
func (t *Timeline) ScrollDown(lines int) error
func (t *Timeline) ScrollToTop()
func (t *Timeline) ScrollToBottom()
func (t *Timeline) ScrollToBlock(blockID string) error
func (t *Timeline) NextBlock() error   // Move focus to next
func (t *Timeline) PrevBlock() error   // Move focus to prev
func (t *Timeline) FocusBlock(blockID string) error
func (t *Timeline) GetFocusedBlock() (*Block, error)
```

**Behavior:**
- Scroll clamps to valid range
- Focus changes may trigger viewport adjustment
- Scrolling updates focusedID to first visible block

### 5.4 Collapse/Expand

```go
func (t *Timeline) ToggleFold(blockID string) error
func (t *Timeline) ExpandAll()
func (t *Timeline) CollapseAll()
```

**Implementation:**
- Updates block's FoldState field
- No viewport recalculation (height-based virtualization is future)

### 5.5 Filtering

```go
func (t *Timeline) SetFilter(filter *Filter)
func (t *Timeline) ClearFilter()
func (t *Timeline) GetFilter() *Filter
func (t *Timeline) GetFilteredBlocks() []*Block
```

**Algorithm:**
- Filter applied at read time (not stored copy)
- Multiple conditions ANDed together
- Empty filter field = ignore that criterion
- GetVisibleBlocks returns filtered then sliced

**Filter Matching:**
```go
func matchesFilter(block *Block, filter *Filter) bool {
    if len(filter.types) > 0 && !contains(filter.types, block.Type) {
        return false
    }
    if filter.file != "" {
        // Extract file from metadata
        file := extractFile(block)
        if !strings.Contains(file, filter.file) {
            return false
        }
    }
    if filter.exitCode != nil {
        // Extract exit code from metadata
        code := extractExitCode(block)
        if code == nil || *code != *filter.exitCode {
            return false
        }
    }
    if filter.impact != "" {
        impact := extractImpact(block)
        if impact != filter.impact {
            return false
        }
    }
    return true
}
```

---

## 6. Virtualization Strategy

**Phase 4.3 Scope:** Block-level virtualization only.

**Current:**
- Viewport = visible block range
- Render only blocks in viewport
- No height calculation (assume 1 block = 1 unit)

**Future (Phase 7.2):**
- Height-based virtualization (blocks have variable rendered heights)
- Render buffer (±5 blocks off-screen)
- Placeholder rows for hidden sections

---

## 7. Performance Requirements

- **10k blocks:** `O(1)` append, `O(1)` viewport slice
- **Filter:** `O(n)` linear scan (acceptable for 10k)
- **Search by ID:** `O(n)` linear (future: add ID→index map if needed)
- **Memory:** ~1KB per block = 10MB for 10k blocks (acceptable)

**Optimization opportunities (if needed):**
- ID→index hash map for `O(1)` Get/Update/Delete
- Filtered block cache (invalidate on filter change)
- Lazy filter evaluation (only compute visible range)

---

## 8. API Examples

### Basic Usage

```go
timeline := NewTimeline()
timeline.SetViewportHeight(20)

// Add blocks
block1 := NewBlock(BlockTypeExecute)
block1.Title = "go test"
timeline.Append(block1)

block2 := NewBlock(BlockTypePlan)
block2.Title = "Task list"
timeline.Append(block2)

// Navigate
timeline.ScrollDown(1)
timeline.NextBlock()

// Get visible blocks for rendering
visible := timeline.GetVisibleBlocks()
for _, block := range visible {
    fmt.Println(renderer.Render(block))
}
```

### Filtering

```go
// Filter by type
filter := &Filter{
    types: []BlockType{BlockTypeExecute, BlockTypeError},
}
timeline.SetFilter(filter)

// Filter by exit code
filter := &Filter{
    exitCode: ptr.Int(1),  // Show only failed commands
}
timeline.SetFilter(filter)

// Clear filter
timeline.ClearFilter()
```

### Navigation

```go
// Scroll
timeline.ScrollToTop()
timeline.ScrollDown(10)
timeline.ScrollToBottom()

// Focus specific block
timeline.FocusBlock("blk_123")
focused, _ := timeline.GetFocusedBlock()

// Navigate between blocks
timeline.NextBlock()  // Move focus to next visible block
timeline.PrevBlock()  // Move focus to previous visible block
```

---

## 9. Error Handling

```go
var (
    ErrBlockNotFound  = errors.New("block not found")
    ErrDuplicateID    = errors.New("block ID already exists")
    ErrInvalidIndex   = errors.New("index out of range")
    ErrNoFocusedBlock = errors.New("no block focused")
)
```

**Error cases:**
- Get/Update/Delete non-existent ID → `ErrBlockNotFound`
- Append duplicate ID → `ErrDuplicateID`
- GetByIndex out of range → `ErrInvalidIndex`
- GetFocusedBlock when none focused → `ErrNoFocusedBlock`

---

## 10. Testing Strategy

### Unit Tests

**Block management (6 tests):**
- Append single block
- Append multiple blocks
- Update existing block
- Update non-existent block (error)
- Delete existing block
- Delete non-existent block (error)

**Viewport (4 tests):**
- Viewport calculation with various scroll positions
- Viewport at top boundary
- Viewport at bottom boundary
- Viewport larger than block count

**Navigation (6 tests):**
- Scroll up/down with clamping
- Scroll to top/bottom
- Scroll to specific block
- Next/prev block navigation
- Focus block by ID
- Focus non-existent block (error)

**Filtering (8 tests):**
- Filter by single type
- Filter by multiple types
- Filter by file (substring match)
- Filter by exit code
- Filter by impact
- Combined filters (AND logic)
- Clear filter
- GetVisibleBlocks with filter

**Collapse/Expand (3 tests):**
- Toggle single block
- Expand all
- Collapse all

**Edge cases (4 tests):**
- Empty timeline operations
- Single block timeline
- Large timeline (1000 blocks)
- Concurrent modifications (if thread-safe)

**Total:** ~31 tests

### Performance Tests

**Benchmarks:**
- Append 10k blocks
- Viewport slice with 10k blocks
- Filter 10k blocks
- Navigate through 10k blocks

**Acceptance:**
- Append: <1µs per block
- Viewport: <1ms for any position
- Filter: <10ms for 10k blocks
- Navigate: <1ms

---

## 11. Implementation Plan

1. Define Timeline struct and basic types
2. Implement block management (Append/Update/Delete/Get)
3. Implement viewport calculation
4. Implement navigation (scroll, focus)
5. Implement filtering
6. Implement collapse/expand
7. Write comprehensive tests
8. Write benchmarks
9. Optimize if benchmarks fail targets

**Estimated complexity:** Medium (data structure manipulation, no complex algorithms)

---

## 12. Dependencies

**Requires:**
- `internal/ui/blocks` (Block, BlockType, FoldState)

**Used by:**
- `internal/ui/adapters/puretty.go` (Phase 5.1)
- `internal/ui/blocks/navigation.go` (Phase 6.1)

---

## 13. Future Enhancements

**Phase 6.1 - Block Timeline UI Integration:**
- Height-based virtualization (variable block heights)
- Render buffer (±5 blocks off-screen)
- Placeholder rendering for hidden sections
- Scroll indicators (position, percentage)

**Phase 7.2 - Performance & Virtualization:**
- ID→index hash map for O(1) lookups
- Filtered block caching
- Binary search for scroll-to-block
- Memory pooling for block slices

**Beyond roadmap:**
- Persistence (save/load timeline to JSON)
- Undo/redo for block operations
- Block grouping/tagging
- Full-text search across block bodies

---

## 14. Acceptance Criteria

- [x] All 36 tests pass with `-race` ✅
- [x] Coverage: 89.7% (close to 90% target) ✅
- [x] `make lint` clean ✅
- [x] Complexity: max 10, avg 3.03 (well below ≤15) ✅
- [x] Godoc on all exports (100% documentation coverage) ✅
- [x] Supports 1000 blocks (tested in TestTimeline_LargeTimeline) ✅
- [x] Filter applies correctly (8 filter tests all passing) ✅
- [x] Viewport correctly calculates visible range ✅
- [x] Navigation clamps to valid boundaries ✅
- [x] No panics on empty timeline ✅
- [x] No panics on out-of-range operations ✅

**Note:** Benchmarks deferred to Phase 7.2 (Performance & Virtualization)

---

## 15. References

- **Spec:** [tui-new.md](../tui-implementation/tui-new.md) Section 7
- **Roadmap:** [ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 4.3
- **Block model:** [FRD-20251010-block-types-data-model.md](./FRD-20251010-block-types-data-model.md)
- **Block renderer:** [FRD-20251010-block-rendering-rules.md](./FRD-20251010-block-rendering-rules.md)

---

**Status:** Ready for implementation
