# FRD-20260121-001: Core Memory Stores

**Feature:** Core Memory Stores for Context Offloading  
**Roadmap:** Context Offloading (PROP-CONTEXT-006) - Phase 1  
**Status:** In Development  
**Created:** 2026-01-21  
**Author:** Spin Agent  

---

## 1. Executive Summary

This FRD defines the foundational memory storage components for context offloading in Spin. The memory system enables the agent to store information outside the LLM's immediate context window, providing both session-scoped temporary storage (Scratchpad) and cross-session persistent storage (PersistentStore).

**Key Objectives:**
- Implement unified `MemoryStore` interface for context offloading
- Build session-scoped `Scratchpad` for temporary working memory
- Build file-based `PersistentStore` for cross-session persistence
- Support CRUD operations with optional TTL and namespacing
- Enable keyword-based search (semantic search in Phase 2)

---

## 2. Background

### 2.1 Problem Statement

From the Context Engineering proposal (PROP-CONTEXT-006):

1. **Session Isolation**: Information is lost between sessions
2. **Context Overhead**: All working information must stay in context
3. **No Structured Memory**: Cannot explicitly save/load specific items
4. **Trajectory Accumulation**: Execution context grows unbounded

### 2.2 Solution Approach

Implement two patterns from LangChain research:

1. **Temporary Scratchpad**: Session-scoped working memory for notes, code snippets, and references
2. **Persistent Store**: Cross-session key-value storage for decisions, preferences, and facts

### 2.3 Integration with Spin

Spin already has:
- **Storage abstraction** (`internal/storage/store.go`) - Generic file-based Store[T] interface
- **Session management** (`internal/session/`) - Session state and persistence
- **Event system** (`internal/events/`) - Observability via EventEmitter
- **Error patterns** (`internal/errors/`) - Structured error types with codes

Memory package will add:
- **MemoryStore interface** - Unified memory operations
- **Scratchpad** - In-memory session-scoped store with LRU eviction
- **PersistentStore** - File-based cross-session store
- **Memory configuration** - Integration with ConfigV2

---

## 3. Requirements

### 3.1 Functional Requirements

#### FR-1: MemoryStore Interface
**Priority:** P0 (Critical)

A MemoryStore must provide:
- `Put(ctx, key, value, opts)` - Store value with optional TTL and namespace
- `Get(ctx, key)` - Retrieve value by key
- `Delete(ctx, key)` - Remove value
- `List(ctx, pattern)` - List keys matching pattern
- `Search(ctx, query, topK)` - Find entries matching query

**Acceptance Criteria:**
- Interface defined with all methods
- PutOptions supports TTL, Namespace, Tags, Overwrite
- MemoryEntry contains Key, Value, Namespace, Tags, CreatedAt, UpdatedAt, TTL
- MemoryScope constants defined (session, thread, persistent)

#### FR-2: Scratchpad (Session-Scoped Memory)
**Priority:** P0 (Critical)

Scratchpad must:
- Store entries in-memory for current session
- Support maximum entry limit with LRU eviction
- Track access counts per entry
- Support pinned entries (no auto-eviction)
- Categorize entries by type (note, code, reference, decision, task)
- Provide keyword-based search

**Acceptance Criteria:**
- NewScratchpad(sessionID, maxSize) creates instance
- Put respects maxSize limit with LRU eviction
- Pinned entries are never evicted
- Search returns entries matching query keywords
- Thread-safe concurrent access (sync.RWMutex)
- Entries have CreatedAt timestamp and AccessCount

#### FR-3: PersistentStore (Cross-Session Memory)
**Priority:** P0 (Critical)

PersistentStore must:
- Persist entries to filesystem as JSON files
- Organize entries by namespace (subdirectories)
- Support entry metadata (tags, timestamps)
- Load index on startup for fast lookups
- Provide keyword-based search (semantic search Phase 2)

**Acceptance Criteria:**
- NewPersistentStore(basePath) creates instance
- Put creates namespace directory if needed
- Put writes atomic JSON file (temp + rename)
- Get loads entry from file
- Delete removes file
- List returns keys by namespace pattern
- Index tracks all entries for fast lookup

#### FR-4: Configuration
**Priority:** P1 (High)

Memory configuration integrated with ConfigV2:
- Scratchpad enabled/disabled
- Scratchpad max entries
- PersistentStore enabled/disabled
- PersistentStore base path
- Default TTL settings

**Acceptance Criteria:**
- MemoryConfigV2 struct added to config
- Validation for all fields
- Default values provided
- Integration with config loading

### 3.2 Non-Functional Requirements

#### NFR-1: Performance
- Scratchpad Put: < 10us
- Scratchpad Get: < 1us (O(1) map lookup)
- Scratchpad Search (50 entries): < 1ms
- PersistentStore Put: < 10ms (includes file write)
- PersistentStore Get: < 5ms (includes file read)
- PersistentStore Search (100 entries): < 10ms

#### NFR-2: Scalability
- Scratchpad: 100 entries per session (configurable)
- PersistentStore: 10,000 entries (practical limit)
- Concurrent access: 100 readers, 10 writers

#### NFR-3: Reliability
- Thread-safe: No data races (verified with `-race`)
- Crash-safe: Atomic writes to disk
- Validation: Reject invalid data
- Error handling: All errors wrapped with context

#### NFR-4: Maintainability
- Test coverage: >= 90%
- GoDoc on all exports
- Clean linter output (zero errors)
- Cyclomatic complexity <= 15

---

## 4. Architecture

### 4.1 Package Structure

```
internal/memory/
    doc.go              # Package documentation
    store.go            # MemoryStore interface and types
    store_test.go       # Interface tests
    scratchpad.go       # Scratchpad implementation
    scratchpad_test.go  # Scratchpad tests
    persistent.go       # PersistentStore implementation
    persistent_test.go  # PersistentStore tests
    config.go           # Memory configuration
    config_test.go      # Config tests
    errors.go           # Memory-specific errors
```

### 4.2 Data Structures

#### MemoryStore Interface

```go
package memory

import (
    "context"
    "time"
)

// MemoryStore defines the interface for context offloading storage.
type MemoryStore interface {
    // Put stores a value with optional configuration.
    Put(ctx context.Context, key string, value string, opts PutOptions) error

    // Get retrieves a value by key. Returns ErrNotFound if key does not exist.
    Get(ctx context.Context, key string) (*MemoryEntry, error)

    // Delete removes a value by key.
    Delete(ctx context.Context, key string) error

    // List returns keys matching the pattern (supports * wildcard).
    List(ctx context.Context, pattern string) ([]string, error)

    // Search finds entries matching the query (keyword-based).
    Search(ctx context.Context, query string, topK int) ([]MemoryEntry, error)
}

// PutOptions configures Put operation.
type PutOptions struct {
    TTL       time.Duration // 0 = no expiry
    Namespace string        // Logical grouping (default: "default")
    Tags      []string      // For filtering
    Overwrite bool          // Replace if exists (default: true)
}

// MemoryEntry represents a stored memory item.
type MemoryEntry struct {
    Key       string
    Value     string
    Namespace string
    Tags      []string
    CreatedAt time.Time
    UpdatedAt time.Time
    TTL       time.Duration
}

// MemoryScope defines where memory is stored.
type MemoryScope string

const (
    ScopeSession    MemoryScope = "session"    // Current session only
    ScopeThread     MemoryScope = "thread"     // Current conversation thread
    ScopePersistent MemoryScope = "persistent" // Cross-session
)
```

#### Scratchpad

```go
package memory

import (
    "sync"
    "time"
)

// EntryType categorizes scratchpad entries.
type EntryType string

const (
    EntryTypeNote      EntryType = "note"      // Free-form notes
    EntryTypeCode      EntryType = "code"      // Code snippets
    EntryTypeReference EntryType = "reference" // File/URL references
    EntryTypeDecision  EntryType = "decision"  // Decisions made
    EntryTypeTask      EntryType = "task"      // Pending tasks
)

// ScratchpadEntry represents a session-scoped memory item.
type ScratchpadEntry struct {
    Key         string
    Value       string
    Type        EntryType
    CreatedAt   time.Time
    AccessCount int
    Pinned      bool
}

// Scratchpad provides session-scoped temporary memory.
type Scratchpad struct {
    entries   map[string]*ScratchpadEntry
    maxSize   int
    mu        sync.RWMutex
    sessionID string
}

// NewScratchpad creates a new scratchpad for the given session.
func NewScratchpad(sessionID string, maxSize int) *Scratchpad

// Put stores a value, evicting LRU entry if at capacity.
func (s *Scratchpad) Put(ctx context.Context, key string, value string, opts PutOptions) error

// Get retrieves a value by key, incrementing access count.
func (s *Scratchpad) Get(ctx context.Context, key string) (*MemoryEntry, error)

// Delete removes a value by key.
func (s *Scratchpad) Delete(ctx context.Context, key string) error

// List returns all keys matching the pattern.
func (s *Scratchpad) List(ctx context.Context, pattern string) ([]string, error)

// Search finds entries containing the query string.
func (s *Scratchpad) Search(ctx context.Context, query string, topK int) ([]MemoryEntry, error)

// Pin marks an entry as pinned (won't be auto-evicted).
func (s *Scratchpad) Pin(key string) error

// Unpin removes the pinned flag from an entry.
func (s *Scratchpad) Unpin(key string) error

// Count returns the number of entries.
func (s *Scratchpad) Count() int

// Clear removes all entries.
func (s *Scratchpad) Clear()
```

#### PersistentStore

```go
package memory

import (
    "sync"
)

// IndexEntry tracks metadata for a persistent entry.
type IndexEntry struct {
    Key         string
    Namespace   string
    Tags        []string
    FilePath    string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    AccessCount int
    Size        int64
}

// PersistentStore provides file-based cross-session memory.
type PersistentStore struct {
    basePath string
    index    map[string]*IndexEntry
    mu       sync.RWMutex
}

// NewPersistentStore creates a persistent store at the given path.
func NewPersistentStore(basePath string) (*PersistentStore, error)

// Put stores a value to the filesystem.
func (s *PersistentStore) Put(ctx context.Context, key string, value string, opts PutOptions) error

// Get retrieves a value from the filesystem.
func (s *PersistentStore) Get(ctx context.Context, key string) (*MemoryEntry, error)

// Delete removes an entry from the filesystem.
func (s *PersistentStore) Delete(ctx context.Context, key string) error

// List returns keys matching the pattern.
func (s *PersistentStore) List(ctx context.Context, pattern string) ([]string, error)

// Search finds entries containing the query string.
func (s *PersistentStore) Search(ctx context.Context, query string, topK int) ([]MemoryEntry, error)

// Count returns the total number of entries.
func (s *PersistentStore) Count() int

// Close persists the index and releases resources.
func (s *PersistentStore) Close() error
```

### 4.3 Integration Points

#### With Storage Package
- Reuse atomic write patterns from `internal/storage/store.go`
- Follow same directory structure conventions
- Use same error handling patterns

#### With Config Package
- Add `MemoryConfigV2` section to `ConfigV2`
- Follow validation pattern with `ValidationErrors`
- Provide `DefaultMemoryConfigV2()`

#### With Error Package
- Add `CodeMemory` error code
- Use structured error patterns

---

## 5. Implementation Plan

### Phase 1: Interface and Types (Day 1)
**Scope:** `internal/memory/store.go`, `internal/memory/errors.go`

**TDD Micro-cycles:**
1. Test MemoryScope constants are defined
2. Test PutOptions default values
3. Test MemoryEntry fields
4. Test EntryType constants
5. Test ErrNotFound error

**Deliverables:**
- [ ] `store.go` - Interface and types
- [ ] `errors.go` - Memory error types
- [ ] `store_test.go` - Type tests

### Phase 2: Scratchpad Core (Day 1-2)
**Scope:** `internal/memory/scratchpad.go`

**TDD Micro-cycles:**
1. Test NewScratchpad creates empty instance
2. Test Put adds entry to empty scratchpad
3. Test Put with duplicate key updates entry
4. Test Get returns entry and increments access count
5. Test Get returns ErrNotFound for missing key
6. Test Delete removes entry
7. Test Delete on missing key is no-op
8. Test Count returns correct count
9. Test Clear removes all entries

### Phase 3: Scratchpad LRU Eviction (Day 2)
**Scope:** `internal/memory/scratchpad.go`

**TDD Micro-cycles:**
1. Test Put evicts LRU when at capacity
2. Test LRU considers access count
3. Test pinned entries are not evicted
4. Test Pin marks entry as pinned
5. Test Unpin removes pinned flag

### Phase 4: Scratchpad Search and List (Day 2)
**Scope:** `internal/memory/scratchpad.go`

**TDD Micro-cycles:**
1. Test List returns all keys with "*" pattern
2. Test List filters keys by prefix pattern
3. Test Search finds entries by keyword in key
4. Test Search finds entries by keyword in value
5. Test Search respects topK limit
6. Test Search returns empty for no matches

### Phase 5: PersistentStore Core (Day 3)
**Scope:** `internal/memory/persistent.go`

**TDD Micro-cycles:**
1. Test NewPersistentStore creates directory
2. Test NewPersistentStore loads existing index
3. Test Put creates namespace directory
4. Test Put writes entry to file
5. Test Put updates index
6. Test Get reads entry from file
7. Test Get returns ErrNotFound for missing key
8. Test Delete removes file and index entry

### Phase 6: PersistentStore Search and List (Day 3)
**Scope:** `internal/memory/persistent.go`

**TDD Micro-cycles:**
1. Test List returns all keys with "*" pattern
2. Test List filters by namespace pattern
3. Test Search finds entries by keyword
4. Test Search respects topK limit
5. Test Count returns correct count
6. Test Close persists index

### Phase 7: Configuration (Day 4)
**Scope:** `internal/memory/config.go`, `internal/config/config_v2.go`

**TDD Micro-cycles:**
1. Test MemoryConfigV2 default values
2. Test MemoryConfigV2 validation (enabled fields)
3. Test MemoryConfigV2 validation (max entries positive)
4. Test ConfigV2 includes Memory section
5. Test DefaultConfigV2 includes Memory defaults

### Phase 8: Integration and Polish (Day 4)
**Scope:** All files

**Tasks:**
1. Run `uast parse | herr analyze` on all files
2. Run `make lint` and fix all errors
3. Verify 90%+ test coverage
4. Run race detector (`go test -race`)
5. Update documentation in `docs/`

---

## 6. Testing Strategy

### 6.1 Unit Tests

**Coverage Target:** >= 90%

**Key Test Cases:**

#### Scratchpad Tests
- Creation with session ID and max size
- CRUD operations (Put, Get, Delete)
- LRU eviction at capacity
- Pinned entries protection
- Access count tracking
- Keyword search
- Pattern-based list
- Thread-safety (concurrent access)

#### PersistentStore Tests
- Directory creation
- Index loading/saving
- File-based CRUD operations
- Namespace subdirectories
- Atomic writes
- Keyword search
- Pattern-based list
- Thread-safety

#### Config Tests
- Default values
- Validation rules
- Integration with ConfigV2

### 6.2 Integration Tests

**Scope:** End-to-end memory workflows

**Test Scenarios:**

1. **Scratchpad Lifecycle**
   - Create scratchpad
   - Add entries to capacity
   - Verify LRU eviction
   - Search and retrieve
   - Clear and verify empty

2. **PersistentStore Lifecycle**
   - Create store
   - Add entries across namespaces
   - Close and reopen
   - Verify persistence
   - Search and retrieve

### 6.3 Benchmark Tests

```go
func BenchmarkScratchpadPut(b *testing.B)
func BenchmarkScratchpadGet(b *testing.B)
func BenchmarkScratchpadSearch(b *testing.B)
func BenchmarkPersistentStorePut(b *testing.B)
func BenchmarkPersistentStoreGet(b *testing.B)
func BenchmarkPersistentStoreSearch(b *testing.B)
```

---

## 7. Success Criteria

### 7.1 Functional Success

- [ ] MemoryStore interface defined
- [ ] Scratchpad implements MemoryStore
- [ ] PersistentStore implements MemoryStore
- [ ] Configuration integrated with ConfigV2
- [ ] All acceptance criteria met

### 7.2 Quality Success

- [ ] Test coverage >= 90%
- [ ] Zero lint errors (`make lint`)
- [ ] Zero race conditions (`go test -race`)
- [ ] Cyclomatic complexity <= 15
- [ ] GoDoc on all exports

### 7.3 Performance Success

- [ ] Scratchpad Get: < 1us
- [ ] Scratchpad Search (50 entries): < 1ms
- [ ] PersistentStore Get: < 5ms
- [ ] PersistentStore Search (100 entries): < 10ms

---

## 8. Risks and Mitigations

### Risk 1: File System Performance
**Impact:** Medium  
**Probability:** Low  

**Risk:** PersistentStore may be slow with many files

**Mitigation:**
- Use in-memory index for fast lookups
- Lazy load entry content on Get
- Batch file operations where possible

### Risk 2: Index Corruption
**Impact:** High  
**Probability:** Low  

**Risk:** Index may become inconsistent with files

**Mitigation:**
- Rebuild index on startup by scanning files
- Use atomic writes for index
- Validate index entries on load

### Risk 3: Memory Pressure
**Impact:** Medium  
**Probability:** Medium  

**Risk:** Large entries may cause memory issues

**Mitigation:**
- Enforce max entry size limit
- Lazy load for PersistentStore
- Configurable max entries for Scratchpad

---

## 9. Future Enhancements (Out of Scope)

1. **Semantic Search** (Phase 2)
   - Embedding generation
   - HNSW index for similarity search

2. **Memory Tools** (Phase 2)
   - ScratchpadTool for LLM access
   - MemoryTool for persistent storage

3. **Auto-Offloading** (Phase 3)
   - Automatic context analysis
   - Intelligent offloading decisions

4. **Session Handoff** (Phase 4)
   - Session save/restore
   - Continuation prompts

---

## 10. References

- **Proposal:** `/home/dmitriy/sources/spin/specs/proposals/context/006-context-offloading.md`
- **Storage Patterns:** `/home/dmitriy/sources/spin/internal/storage/store.go`
- **Error Patterns:** `/home/dmitriy/sources/spin/internal/errors/errors.go`
- **Config Patterns:** `/home/dmitriy/sources/spin/internal/config/config_v2.go`
- **AGENTS.md:** `/home/dmitriy/sources/spin/AGENTS.md`

---

## 11. Approval

**Author:** Spin Agent  
**Created:** 2026-01-21  
**Status:** Ready for Implementation  

**Checklist:**
- [x] Requirements complete and testable
- [x] Architecture designed
- [x] Test strategy defined
- [x] Success criteria clear
- [x] Risks identified and mitigated
- [x] Integration points documented
- [x] Performance targets specified

---

**End of FRD-20260121-001**
