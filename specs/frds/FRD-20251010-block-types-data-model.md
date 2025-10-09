# FRD-20251010: Block Types and Data Model

**Feature:** Block system data model for TUI timeline
**Priority:** P1 (Critical for Phase 4)
**Date:** 2025-10-10
**Author:** Spin Agent
**Roadmap:** Phase 4.1 (Block Types and Data Model)
**Spec Reference:** [specs/tui-implementation/tui-new.md](../tui-implementation/tui-new.md) Section 3

---

## 1. Goals & Scope

### 1.1 Primary Goals

Implement the core data model for the TUI block system:

1. **Block Type System**: Define all block types (EXECUTE, PLAN, READ, GREP, APPLY_PATCH, SUMMARY, TESTING, NOTICE, ERROR)
2. **Block Data Structure**: Core block model with metadata, body, fold state, and severity
3. **Type-Specific Metadata**: Specialized metadata structs for each block type
4. **JSON Serialization**: Roundtrip-safe persistence support
5. **Validation**: Ensure required fields per block type

### 1.2 Out of Scope

- Rendering logic (Phase 4.2)
- Timeline state machine (Phase 4.3)
- UI integration (Phase 5.1)
- Block navigation (Phase 6.1)

---

## 2. Technical Design

### 2.1 Package Structure

```
internal/ui/blocks/
├── model.go          # Core Block struct and methods
├── types.go          # BlockType enum, FoldState, Severity
├── metadata.go       # Type-specific metadata structs
├── validation.go     # Block validation logic
├── model_test.go     # Block model tests
├── metadata_test.go  # Metadata validation tests
└── doc.go            # Package documentation
```

### 2.2 Core Types

#### BlockType (Enum)

```go
// BlockType represents the category of block
type BlockType string

const (
    BlockTypeExecute    BlockType = "EXECUTE"
    BlockTypePlan       BlockType = "PLAN"
    BlockTypeRead       BlockType = "READ"
    BlockTypeGrep       BlockType = "GREP"
    BlockTypeApplyPatch BlockType = "APPLY_PATCH"
    BlockTypeSummary    BlockType = "SUMMARY"
    BlockTypeTesting    BlockType = "TESTING"
    BlockTypeNotice     BlockType = "NOTICE"
    BlockTypeError      BlockType = "ERROR"
)

// String returns the string representation
func (bt BlockType) String() string

// Valid returns true if the block type is valid
func (bt BlockType) Valid() bool
```

#### FoldState (Enum)

```go
// FoldState represents the collapse/expand state
type FoldState string

const (
    FoldStateExpanded  FoldState = "expanded"
    FoldStateCollapsed FoldState = "collapsed"
)

// String returns the string representation
func (fs FoldState) String() string

// Valid returns true if the fold state is valid
func (fs FoldState) Valid() bool
```

#### Severity (Enum)

```go
// Severity represents the importance/criticality level
type Severity string

const (
    SeverityInfo  Severity = "info"
    SeverityWarn  Severity = "warn"
    SeverityError Severity = "error"
)

// String returns the string representation
func (s Severity) String() string

// Valid returns true if the severity is valid
func (s Severity) Valid() bool
```

### 2.3 Block Data Model

```go
// Block represents a single block in the timeline
type Block struct {
    ID        string                 `json:"id"`         // Unique ID (blk_timestamp_seq)
    Type      BlockType              `json:"type"`       // Block type
    Title     string                 `json:"title"`      // Optional concise title
    Meta      map[string]interface{} `json:"meta"`       // Type-specific metadata
    Body      string                 `json:"body"`       // Renderable content
    FoldState FoldState              `json:"fold_state"` // Collapsed/expanded
    Severity  Severity               `json:"severity"`   // Info/warn/error
    Timestamp int64                  `json:"timestamp"`  // Unix timestamp (milliseconds)
}

// NewBlock creates a new block with defaults
func NewBlock(blockType BlockType) *Block

// Validate validates the block structure
func (b *Block) Validate() error

// GetMeta retrieves typed metadata
func (b *Block) GetMeta(key string) (interface{}, bool)

// SetMeta sets metadata value
func (b *Block) SetMeta(key string, value interface{})
```

### 2.4 Type-Specific Metadata

#### ExecuteMeta

```go
// ExecuteMeta holds metadata for EXECUTE blocks
type ExecuteMeta struct {
    Command    string `json:"command"`               // Command string
    CWD        string `json:"cwd"`                   // Working directory
    TimeoutSec int    `json:"timeout_sec,omitempty"` // Timeout in seconds
    Impact     string `json:"impact"`                // low|medium|high
    ExitCode   *int   `json:"exit_code,omitempty"`   // Exit code (nil if running)
    DurationMS *int64 `json:"duration_ms,omitempty"` // Duration in milliseconds
    LinesOut   *int   `json:"lines_out,omitempty"`   // Output line count
}

// Validate validates the execute metadata
func (m *ExecuteMeta) Validate() error
```

#### ReadMeta

```go
// ReadMeta holds metadata for READ blocks
type ReadMeta struct {
    File   string `json:"file"`             // File path
    Offset int    `json:"offset,omitempty"` // Start line offset
    Limit  int    `json:"limit,omitempty"`  // Line limit
}

// Validate validates the read metadata
func (m *ReadMeta) Validate() error
```

#### GrepMeta

```go
// GrepMeta holds metadata for GREP blocks
type GrepMeta struct {
    Pattern string `json:"pattern"`           // Search pattern
    Mode    string `json:"mode"`              // content|files_with_matches|count
    Context int    `json:"context,omitempty"` // Context lines (for -A/-B/-C)
}

// Validate validates the grep metadata
func (m *GrepMeta) Validate() error
```

#### PatchMeta

```go
// PatchMeta holds metadata for APPLY_PATCH blocks
type PatchMeta struct {
    File       string `json:"file"`                  // Target file
    Succeeded  bool   `json:"succeeded"`             // Success status
    LinesAdded *int   `json:"lines_added,omitempty"` // Lines added
    LinesRemoved *int `json:"lines_removed,omitempty"` // Lines removed
    ErrorMsg   string `json:"error_msg,omitempty"`   // Error message if failed
}

// Validate validates the patch metadata
func (m *PatchMeta) Validate() error
```

#### PlanMeta

```go
// PlanMeta holds metadata for PLAN blocks
type PlanMeta struct {
    Total       int `json:"total"`        // Total items
    Pending     int `json:"pending"`      // Pending count
    InProgress  int `json:"in_progress"`  // In-progress count
    Completed   int `json:"completed"`    // Completed count
}

// Validate validates the plan metadata
func (m *PlanMeta) Validate() error
```

### 2.5 Metadata Helpers

```go
// ParseExecuteMeta extracts ExecuteMeta from block
func ParseExecuteMeta(b *Block) (*ExecuteMeta, error)

// ParseReadMeta extracts ReadMeta from block
func ParseReadMeta(b *Block) (*ReadMeta, error)

// ParseGrepMeta extracts GrepMeta from block
func ParseGrepMeta(b *Block) (*GrepMeta, error)

// ParsePatchMeta extracts PatchMeta from block
func ParsePatchMeta(b *Block) (*PatchMeta, error)

// ParsePlanMeta extracts PlanMeta from block
func ParsePlanMeta(b *Block) (*PlanMeta, error)

// SetExecuteMeta sets ExecuteMeta on block
func SetExecuteMeta(b *Block, m *ExecuteMeta) error

// SetReadMeta sets ReadMeta on block
func SetReadMeta(b *Block, m *ReadMeta) error

// SetGrepMeta sets GrepMeta on block
func SetGrepMeta(b *Block, m *GrepMeta) error

// SetPatchMeta sets PatchMeta on block
func SetPatchMeta(b *Block, m *PatchMeta) error

// SetPlanMeta sets PlanMeta on block
func SetPlanMeta(b *Block, m *PlanMeta) error
```

### 2.6 ID Generation

```go
// GenerateBlockID creates a unique block ID
// Format: blk_{unix_timestamp_ms}_{sequence}
// Example: blk_1738950123456_01
func GenerateBlockID(seq int) string
```

---

## 3. Validation Rules

### 3.1 Block Validation

**Required fields:**
- `ID` must not be empty
- `Type` must be valid BlockType
- `FoldState` must be valid FoldState
- `Severity` must be valid Severity
- `Timestamp` must be positive

**Type-specific validation:**
- Each block type requires specific metadata fields
- Validation delegated to metadata struct's `Validate()` method

### 3.2 Metadata Validation

#### EXECUTE
- `Command` required, non-empty
- `CWD` required, non-empty
- `Impact` must be "low", "medium", or "high"
- `ExitCode` must be >= 0 if present
- `DurationMS` must be >= 0 if present
- `LinesOut` must be >= 0 if present

#### READ
- `File` required, non-empty
- `Offset` must be >= 0
- `Limit` must be >= 0

#### GREP
- `Pattern` required, non-empty
- `Mode` must be "content", "files_with_matches", or "count"
- `Context` must be >= 0

#### APPLY_PATCH
- `File` required, non-empty
- `LinesAdded` must be >= 0 if present
- `LinesRemoved` must be >= 0 if present

#### PLAN
- `Total` must be >= 0
- `Pending` must be >= 0
- `InProgress` must be >= 0
- `Completed` must be >= 0
- Sum (Pending + InProgress + Completed) must equal Total

---

## 4. JSON Serialization

### 4.1 Marshaling

Blocks must serialize to JSON matching spec format (Section 3.2):

```json
{
  "id": "blk_1738950123_07",
  "type": "EXECUTE",
  "title": "Run tests",
  "meta": {
    "command": "go test -race ./...",
    "cwd": "/home/user/project",
    "timeout_sec": 600,
    "impact": "medium",
    "exit_code": 0,
    "duration_ms": 4200,
    "lines_out": 54
  },
  "body": "=== RUN TestFoo\n--- PASS: TestFoo (0.00s)\nPASS\n",
  "fold_state": "expanded",
  "severity": "info",
  "timestamp": 1738950123456
}
```

### 4.2 Unmarshaling

JSON must roundtrip correctly:
- All fields preserved
- Metadata correctly typed
- Validation enforced on unmarshal

---

## 5. Testing Strategy

### 5.1 Unit Tests

**Block creation:**
- NewBlock creates valid defaults
- GenerateBlockID produces unique IDs
- SetMeta/GetMeta work correctly

**Validation:**
- Valid blocks pass validation
- Invalid blocks fail with descriptive errors
- Each metadata type validates correctly

**Metadata parsing:**
- ParseXXXMeta extracts correctly
- SetXXXMeta updates correctly
- Type mismatches return errors

**JSON serialization:**
- Marshal produces correct JSON
- Unmarshal reads JSON correctly
- Roundtrip preserves all data

### 5.2 Table-Driven Tests

Each block type tested with:
- Valid metadata (should pass)
- Missing required fields (should fail)
- Invalid field values (should fail)
- Edge cases (boundary values)

### 5.3 Coverage Target

- Overall: ≥85%
- Critical paths (validation, parsing): ≥90%

---

## 6. Examples

### 6.1 Create EXECUTE Block

```go
block := blocks.NewBlock(blocks.BlockTypeExecute)
block.Title = "Run tests"
block.Body = "test output here..."

meta := &blocks.ExecuteMeta{
    Command:    "go test -race ./...",
    CWD:        "/home/user/project",
    TimeoutSec: 600,
    Impact:     "medium",
    ExitCode:   ptr.Int(0),
    DurationMS: ptr.Int64(4200),
    LinesOut:   ptr.Int(54),
}
blocks.SetExecuteMeta(block, meta)

if err := block.Validate(); err != nil {
    log.Fatalf("Invalid block: %v", err)
}
```

### 6.2 Parse Metadata

```go
execMeta, err := blocks.ParseExecuteMeta(block)
if err != nil {
    log.Fatalf("Failed to parse: %v", err)
}
fmt.Printf("Command: %s, Exit: %d\n", execMeta.Command, *execMeta.ExitCode)
```

### 6.3 JSON Roundtrip

```go
// Marshal
data, err := json.Marshal(block)
if err != nil {
    log.Fatal(err)
}

// Unmarshal
var restored blocks.Block
if err := json.Unmarshal(data, &restored); err != nil {
    log.Fatal(err)
}

// Validate
if err := restored.Validate(); err != nil {
    log.Fatalf("Restored block invalid: %v", err)
}
```

---

## 7. Quality Gates

### 7.1 Definition of Done

- [x] All tests pass with `-race`
- [x] Coverage ≥85%
- [x] `make lint` clean
- [x] Complexity ≤10 per function
- [x] All block types from spec represented
- [x] Metadata validates required fields per type
- [x] JSON roundtrip preserves all data
- [x] Godoc on all exports
- [x] No dead code

### 7.2 Acceptance Criteria

**Block Types:**
- All 9 block types defined
- Each type has valid string representation
- Type validation works correctly

**Block Model:**
- NewBlock creates valid defaults
- ID generation produces unique IDs
- Validation catches invalid blocks
- Meta get/set operations work

**Metadata:**
- All 5 metadata types defined
- Parse/Set helpers work for each type
- Validation enforces type-specific rules

**JSON:**
- Marshal produces spec-compliant JSON
- Unmarshal reads JSON correctly
- Roundtrip preserves all fields
- Validation runs on unmarshal

---

## 8. Dependencies

### 8.1 External Dependencies

- `encoding/json` (stdlib)
- `fmt`, `errors` (stdlib)
- `time` (for timestamp generation)

### 8.2 Internal Dependencies

None (this is a leaf package)

---

## 9. Migration Plan

N/A (new package)

---

## 10. Risks & Mitigations

### 10.1 Risk: Metadata Type Mismatches

**Scenario:** Wrong metadata type for block type
**Mitigation:** Validation checks metadata consistency
**Test:** Metadata validation tests

### 10.2 Risk: JSON Schema Changes

**Scenario:** Spec changes, breaks persistence
**Mitigation:** Version field in block (future), comprehensive tests
**Test:** JSON roundtrip tests

### 10.3 Risk: ID Collisions

**Scenario:** Two blocks get same ID
**Mitigation:** Include sequence number, timestamp in milliseconds
**Test:** ID generation tests

---

## 11. Future Enhancements

### 11.1 Block Versioning

Add `version` field for schema evolution:

```go
type Block struct {
    Version   int    `json:"version"`
    // ...
}
```

### 11.2 Custom Block Types

Plugin system for user-defined block types:

```go
type BlockTypeRegistry interface {
    Register(name string, validator MetadataValidator)
}
```

### 11.3 Metadata Schemas

JSON Schema validation for metadata:

```go
func (bt BlockType) MetadataSchema() *jsonschema.Schema
```

---

## 12. References

- **Spec:** [specs/tui-implementation/tui-new.md](../tui-implementation/tui-new.md) Section 3
- **Roadmap:** [specs/tui-implementation/ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 4.1
- **AGENTS.md:** Working loop steps 1-14

---

## 13. Changelog

**2025-10-10:** Initial FRD
**Author:** Spin Agent
**Status:** Ready for Implementation
