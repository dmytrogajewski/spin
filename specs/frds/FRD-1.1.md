# FRD-1.1: Session Management

**Feature ID:** 1.1  
**Feature Name:** Session Management  
**Phase:** 1 - State Management  
**Priority:** P0 (Blocker)  
**Estimated Effort:** 12 hours  
**Status:** 🚧 In Progress  

---

## Overview

Implement session state management including session creation, persistence, and retrieval. Sessions represent persistent conversation threads that can be saved, resumed, and managed across multiple interactions. This is foundational for stateful agent conversations.

## Business Value

- Enables conversation persistence across agent restarts
- Supports conversation history and replay
- Allows users to resume interrupted conversations
- Provides audit trail for agent actions
- Enables multi-session management
- Foundation for advanced features (session branching, sharing, etc.)

## Functional Requirements

### FR-1.1.1: Session Data Structure

Define comprehensive Session struct with all state:

```go
// Session represents a persistent conversation session
type Session struct {
    ID          string              // Unique session identifier (UUID)
    WorkDir     string              // Working directory for this session
    CreatedAt   time.Time           // Session creation timestamp
    UpdatedAt   time.Time           // Last update timestamp
    Turns       []*turn.Turn        // Conversation turns
    Metadata    Metadata            // Session metadata
    State       State               // Current session state
    Config      *Config             // Session-specific configuration
    Version     int                 // Schema version for migrations
}

// Metadata contains session metadata
type Metadata struct {
    Title       string              // User-friendly session title
    Description string              // Session description
    Tags        []string            // User-defined tags
    TotalTurns  int                 // Total turn count
    TokensUsed  int                 // Total tokens consumed
    LastError   string              // Last error message (if any)
}

// State represents session execution state
type State string

const (
    StateActive     State = "active"      // Session is active
    StateCompleted  State = "completed"   // Session completed successfully
    StateFailed     State = "failed"      // Session failed
    StateArchived   State = "archived"    // Session archived
    StateCancelled  State = "cancelled"   // Session cancelled by user
)
```

### FR-1.1.2: Session Creation

Implement session creation:

```go
// NewSession creates a new session
func NewSession(workDir string, cfg *Config) *Session

// Returns:
// - Session with unique ID (UUID v4)
// - CreatedAt and UpdatedAt set to current time
// - Empty Turns slice
// - Default metadata
// - State set to Active
// - Config copied (not referenced)
// - Version set to current schema version
```

**Requirements:**
- Generate unique session ID using UUID v4
- Initialize all timestamps
- Validate workDir exists and is accessible
- Deep copy configuration to prevent mutations
- Set schema version for future migrations

### FR-1.1.3: Turn Management

Implement turn operations:

```go
// AddTurn appends a turn to the session
func (s *Session) AddTurn(t *turn.Turn) error

// GetTurn retrieves a turn by ID
func (s *Session) GetTurn(turnID string) (*turn.Turn, error)

// LastTurn returns the most recent turn
func (s *Session) LastTurn() *turn.Turn

// TurnCount returns the number of turns
func (s *Session) TurnCount() int
```

**Requirements:**
- AddTurn validates turn is non-nil
- AddTurn updates UpdatedAt timestamp
- AddTurn updates Metadata.TotalTurns
- Thread-safe turn access (mutex protection)
- Maintain turn order (append-only)

### FR-1.1.4: Session Persistence

Implement file-based persistence:

```go
// Save persists session to storage
func (s *Session) Save(storage Storage) error

// Load retrieves session from storage
func Load(storage Storage, id string) (*Session, error)

// Delete removes session from storage
func Delete(storage Storage, id string) error

// Exists checks if session exists
func Exists(storage Storage, id string) (bool, error)
```

**Storage Format:**
- File: `~/.spin/sessions/{session-id}.json`
- Format: JSON (human-readable, version-controlled)
- Atomic writes (write to temp file, then rename)
- Validate on load (schema version, required fields)

**Requirements:**
- Atomic writes to prevent corruption
- Handle concurrent access (file locking)
- Validate session data on load
- Return clear errors for corruption
- Support schema migration (Version field)

### FR-1.1.5: Storage Interface

Define storage abstraction:

```go
// Storage provides session persistence
type Storage interface {
    // Save writes session to storage
    Save(s *Session) error
    
    // Load reads session from storage
    Load(id string) (*Session, error)
    
    // Delete removes session from storage
    Delete(id string) error
    
    // Exists checks if session exists
    Exists(id string) (bool, error)
    
    // List returns all session IDs with optional filter
    List(filter Filter) ([]string, error)
    
    // ListMetadata returns session metadata without loading full sessions
    ListMetadata(filter Filter) ([]*Metadata, error)
}

// Filter for session queries
type Filter struct {
    State       *State              // Filter by state
    WorkDir     string              // Filter by working directory
    CreatedAfter  *time.Time        // Filter by creation date
    CreatedBefore *time.Time        // Filter by creation date
    Tags        []string            // Filter by tags (OR logic)
    Limit       int                 // Limit results
    Offset      int                 // Pagination offset
}
```

### FR-1.1.6: File-Based Storage Implementation

Implement default file-based storage:

```go
// FileStorage implements Storage using filesystem
type FileStorage struct {
    baseDir string              // Base directory (e.g., ~/.spin/sessions)
    mu      sync.RWMutex        // Concurrent access protection
}

// NewFileStorage creates file-based storage
func NewFileStorage(baseDir string) (*FileStorage, error)
```

**Requirements:**
- Create baseDir if it doesn't exist (with proper permissions: 0700)
- Validate baseDir is writable
- Use proper file permissions (0600 for session files)
- Handle missing directory errors gracefully
- Support concurrent reads (RWMutex)
- Exclusive writes (lock during save)

### FR-1.1.7: Session Metadata Management

Implement metadata operations:

```go
// UpdateMetadata updates session metadata
func (s *Session) UpdateMetadata(fn func(*Metadata)) error

// SetState updates session state
func (s *Session) SetState(state State) error

// AddTag adds a tag to session
func (s *Session) AddTag(tag string) error

// RemoveTag removes a tag from session
func (s *Session) RemoveTag(tag string) error

// SetTitle updates session title
func (s *Session) SetTitle(title string) error
```

**Requirements:**
- Thread-safe metadata updates
- Validate state transitions (Active → Completed/Failed/Cancelled)
- Update UpdatedAt on any change
- Prevent duplicate tags
- Sanitize title (max length, no control characters)

### FR-1.1.8: Session State Machine

Implement valid state transitions:

```
Active → Completed (successful completion)
Active → Failed (error occurred)
Active → Cancelled (user cancelled)
Active → Archived (user archived)
Completed → Archived (cleanup)
Failed → Archived (cleanup)
Cancelled → Archived (cleanup)
```

**Invalid Transitions:**
- Archived → any state (archived is terminal)
- Completed/Failed/Cancelled → Active (can't reactivate)

### FR-1.1.9: Concurrent Access Safety

Implement thread-safe operations:
- RWMutex for session state access
- Read lock for queries (GetTurn, LastTurn, etc.)
- Write lock for mutations (AddTurn, UpdateMetadata, etc.)
- Atomic file operations for persistence
- File locking for concurrent process access

### FR-1.1.10: Session Validation

Implement validation:

```go
// Validate checks session integrity
func (s *Session) Validate() error
```

**Checks:**
- ID is valid UUID
- WorkDir is non-empty
- CreatedAt <= UpdatedAt
- State is valid value
- All turns have unique IDs
- Turn timestamps are monotonic
- Metadata is consistent with actual data
- Schema version is supported

## Non-Functional Requirements

### NFR-1.1.1: Performance
- Session creation: <1ms
- Session save: <50ms for typical session (~100 turns)
- Session load: <100ms for typical session
- Session list: <200ms for 1000 sessions
- Concurrent read operations: no lock contention
- File storage: handle sessions up to 10MB

### NFR-1.1.2: Reliability
- Atomic writes prevent corruption
- Sessions survive process crashes
- Data integrity validated on load
- Clear error messages for all failures
- Graceful degradation for corrupted sessions

### NFR-1.1.3: Scalability
- Support thousands of sessions per user
- Efficient filtering without loading all sessions
- Pagination support for large session lists
- Metadata indexing for fast queries

### NFR-1.1.4: Maintainability
- Schema versioning for future migrations
- Clear separation of concerns (Session vs Storage)
- Easy to add alternative storage backends
- Comprehensive test coverage

### NFR-1.1.5: Security
- Session files have restrictive permissions (0600)
- Session directory has restrictive permissions (0700)
- Validate session data to prevent injection
- Sanitize file paths to prevent traversal

## Technical Design

### Session Lifecycle

```
1. Creation:
   NewSession(workDir, config) → Session
   ├─ Generate UUID
   ├─ Initialize timestamps
   ├─ Set default metadata
   └─ Set State to Active

2. Usage:
   session.AddTurn(turn) → error
   ├─ Validate turn
   ├─ Append to Turns
   ├─ Update metadata
   └─ Update timestamp

3. Persistence:
   session.Save(storage) → error
   ├─ Validate session
   ├─ Serialize to JSON
   ├─ Write atomically
   └─ Update file metadata

4. Resumption:
   Load(storage, id) → (*Session, error)
   ├─ Read file
   ├─ Deserialize JSON
   ├─ Validate schema
   ├─ Migrate if needed
   └─ Return session

5. Cleanup:
   Delete(storage, id) → error
   └─ Remove file
```

### File Storage Layout

```
~/.spin/sessions/
├── {session-id-1}.json
├── {session-id-2}.json
├── {session-id-3}.json
└── .index (optional: future optimization)
```

### JSON Schema

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "work_dir": "/home/user/project",
  "created_at": "2025-10-03T10:00:00Z",
  "updated_at": "2025-10-03T10:15:00Z",
  "state": "active",
  "version": 1,
  "config": {
    "provider": "ollama",
    "model": "codellama:13b",
    ...
  },
  "metadata": {
    "title": "Implement authentication",
    "description": "Add JWT-based authentication",
    "tags": ["auth", "security"],
    "total_turns": 5,
    "tokens_used": 12450,
    "last_error": ""
  },
  "turns": [
    {
      "id": "turn-1",
      "user_input": "Add JWT authentication",
      "ai_response": "I'll implement JWT...",
      ...
    }
  ]
}
```

### Atomic Write Pattern

```go
func (fs *FileStorage) Save(s *Session) error {
    fs.mu.Lock()
    defer fs.mu.Unlock()
    
    // 1. Validate session
    if err := s.Validate(); err != nil {
        return fmt.Errorf("invalid session: %w", err)
    }
    
    // 2. Serialize to JSON
    data, err := json.MarshalIndent(s, "", "  ")
    if err != nil {
        return fmt.Errorf("serialize: %w", err)
    }
    
    // 3. Write to temporary file
    tmpPath := filepath.Join(fs.baseDir, s.ID+".tmp")
    if err := os.WriteFile(tmpPath, data, 0600); err != nil {
        return fmt.Errorf("write temp: %w", err)
    }
    
    // 4. Atomic rename
    finalPath := filepath.Join(fs.baseDir, s.ID+".json")
    if err := os.Rename(tmpPath, finalPath); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("atomic rename: %w", err)
    }
    
    return nil
}
```

### Concurrent Access Safety

```go
type Session struct {
    // ... fields ...
    mu sync.RWMutex  // Protects all fields
}

func (s *Session) AddTurn(t *turn.Turn) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // ... add turn ...
}

func (s *Session) LastTurn() *turn.Turn {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if len(s.Turns) == 0 {
        return nil
    }
    return s.Turns[len(s.Turns)-1]
}
```

## Definition of Ready (DoR)

- [x] Feature 0.3 (Configuration System) completed
- [x] Storage format decided (JSON)
- [x] Session directory structure defined (`~/.spin/sessions/`)
- [x] Turn package structure exists (`internal/core/turn/`)
- [x] UUID library available (use `google/uuid` or stdlib)

## Definition of Done (DoD)

- [ ] `session/session.go` implemented with Session struct
- [ ] `session/storage.go` with Storage interface
- [ ] `session/file_storage.go` with FileStorage implementation
- [ ] `session/metadata.go` with Metadata struct and operations
- [ ] NewSession() constructor implemented
- [ ] AddTurn(), GetTurn(), LastTurn() implemented
- [ ] Save(), Load(), Delete(), Exists() implemented
- [ ] Storage interface with all methods
- [ ] FileStorage with atomic writes
- [ ] State machine with validation
- [ ] Concurrent access safety (RWMutex)
- [ ] Session validation implemented
- [ ] Filter implementation for queries
- [ ] Unit tests for Session operations (>90% coverage)
- [ ] Unit tests for FileStorage (>90% coverage)
- [ ] Unit tests for metadata operations
- [ ] Unit tests for state transitions
- [ ] Unit tests for concurrent access
- [ ] Integration tests for persistence
- [ ] Integration tests with real filesystem
- [ ] Test atomic write behavior
- [ ] Test corruption recovery
- [ ] All tests passing
- [ ] Code passes linter without errors
- [ ] Code analyzed with uast/herr (complexity ≤15)
- [ ] Godoc comments for all exported symbols
- [ ] Session management documented

## Testing Strategy

### Unit Tests

**Test File:** `internal/core/session/session_test.go`

```go
// Session Creation
func TestNewSession(t *testing.T)
func TestNewSession_ValidatesWorkDir(t *testing.T)
func TestNewSession_GeneratesUniqueID(t *testing.T)
func TestNewSession_InitializesTimestamps(t *testing.T)
func TestNewSession_CopiesConfig(t *testing.T)

// Turn Management
func TestSession_AddTurn(t *testing.T)
func TestSession_AddTurn_NilTurn(t *testing.T)
func TestSession_AddTurn_UpdatesMetadata(t *testing.T)
func TestSession_GetTurn(t *testing.T)
func TestSession_GetTurn_NotFound(t *testing.T)
func TestSession_LastTurn(t *testing.T)
func TestSession_LastTurn_EmptySession(t *testing.T)
func TestSession_TurnCount(t *testing.T)

// Metadata Operations
func TestSession_UpdateMetadata(t *testing.T)
func TestSession_SetState(t *testing.T)
func TestSession_SetState_InvalidTransition(t *testing.T)
func TestSession_AddTag(t *testing.T)
func TestSession_AddTag_Duplicate(t *testing.T)
func TestSession_RemoveTag(t *testing.T)
func TestSession_SetTitle(t *testing.T)

// Validation
func TestSession_Validate_Valid(t *testing.T)
func TestSession_Validate_InvalidID(t *testing.T)
func TestSession_Validate_EmptyWorkDir(t *testing.T)
func TestSession_Validate_InvalidTimestamps(t *testing.T)
func TestSession_Validate_InvalidState(t *testing.T)
func TestSession_Validate_DuplicateTurnIDs(t *testing.T)

// Concurrency
func TestSession_ConcurrentReads(t *testing.T)
func TestSession_ConcurrentWrites(t *testing.T)
func TestSession_ConcurrentReadWrite(t *testing.T)
```

**Test File:** `internal/core/session/storage_test.go`

```go
// FileStorage Creation
func TestNewFileStorage(t *testing.T)
func TestNewFileStorage_CreatesDirectory(t *testing.T)
func TestNewFileStorage_ValidatesPermissions(t *testing.T)

// Persistence
func TestFileStorage_Save(t *testing.T)
func TestFileStorage_Save_AtomicWrite(t *testing.T)
func TestFileStorage_Save_InvalidSession(t *testing.T)
func TestFileStorage_Load(t *testing.T)
func TestFileStorage_Load_NotFound(t *testing.T)
func TestFileStorage_Load_CorruptedData(t *testing.T)
func TestFileStorage_Delete(t *testing.T)
func TestFileStorage_Exists(t *testing.T)

// Listing and Filtering
func TestFileStorage_List(t *testing.T)
func TestFileStorage_List_EmptyStorage(t *testing.T)
func TestFileStorage_ListMetadata(t *testing.T)
func TestFileStorage_List_WithFilter_State(t *testing.T)
func TestFileStorage_List_WithFilter_WorkDir(t *testing.T)
func TestFileStorage_List_WithFilter_Date(t *testing.T)
func TestFileStorage_List_WithFilter_Tags(t *testing.T)
func TestFileStorage_List_WithPagination(t *testing.T)

// Concurrent Access
func TestFileStorage_ConcurrentSaves(t *testing.T)
func TestFileStorage_ConcurrentReads(t *testing.T)
```

**Test File:** `internal/core/session/metadata_test.go`

```go
func TestMetadata_Update(t *testing.T)
func TestMetadata_TokenTracking(t *testing.T)
func TestMetadata_TurnCountConsistency(t *testing.T)
```

### Integration Tests

**Test File:** `internal/core/session/integration_test.go`

```go
func TestSession_SaveAndLoad_RoundTrip(t *testing.T)
func TestSession_SaveAndLoad_LargeSession(t *testing.T)
func TestSession_SaveAndLoad_WithTurns(t *testing.T)
func TestSession_Persistence_AcrossRestarts(t *testing.T)
func TestSession_CorruptionRecovery(t *testing.T)
func TestSession_Migration_OldVersion(t *testing.T)
```

### Test Data

Create test fixtures in `internal/core/session/testdata/`:
- `valid_session.json` - Valid session file
- `corrupted_session.json` - Corrupted JSON
- `old_version_session.json` - Old schema version
- `minimal_session.json` - Minimal valid session
- `large_session.json` - Session with many turns

### Coverage Target
- Minimum 90% coverage for all session files
- 100% coverage for state transitions
- 100% coverage for validation paths
- All error cases tested
- All concurrent access paths tested

## Implementation Tasks

1. ✅ Create FRD-1.1.md (this document)
2. Create `internal/core/session/` directory
3. Create `internal/core/session/session_test.go` with all test cases (TDD)
4. Create `internal/core/session/storage_test.go` with all test cases (TDD)
5. Create `internal/core/session/metadata_test.go` with all test cases (TDD)
6. Create test fixtures in `testdata/`
7. Implement `session.go` with Session struct
8. Implement `metadata.go` with Metadata struct
9. Implement NewSession() constructor
10. Implement turn management methods
11. Implement metadata operations
12. Implement state machine
13. Implement validation
14. Implement `storage.go` with Storage interface
15. Implement `file_storage.go` with FileStorage
16. Implement Save() with atomic writes
17. Implement Load() with validation
18. Implement Delete() and Exists()
19. Implement List() and ListMetadata() with filtering
20. Run tests and fix failures
21. Add concurrent access tests
22. Add integration tests
23. Run linter and fix issues
24. Analyze with `uast parse session.go | herr analyze`
25. Analyze with `uast parse storage.go | herr analyze`
26. Analyze with `uast parse file_storage.go | herr analyze`
27. Optimize complexity if needed (target ≤15)
28. Add godoc comments
29. Update ROADMAP.md (mark 1.1 as complete)
30. Update SUMMARY.md

## Dependencies

### Prerequisites
- Feature 0.3 (Configuration System) completed
- Turn package stub (minimal Turn struct)
- UUID generation library

### Blocks
- Feature 1.2 (Turn State Machine) - needs Session to store turns
- Feature 7.1 (Conversation Implementation) - needs Session for state
- Feature 7.2 (Conversation Manager) - needs Session for persistence

### Blocked By
- Feature 0.3 (Configuration) - for Config field

## Risks and Mitigations

### Risk 1: File System Limitations
**Impact:** Different filesystems handle atomic operations differently  
**Mitigation:** Use os.Rename() which is atomic on all major platforms, test on multiple OSes

### Risk 2: Concurrent Process Access
**Impact:** Multiple spin instances accessing same session file  
**Mitigation:** File-level locking (flock), clear error messages, consider lock timeout

### Risk 3: Large Session Files
**Impact:** Sessions with thousands of turns may be slow to load  
**Mitigation:** Implement pagination, consider compression, document limits

### Risk 4: Schema Evolution
**Impact:** Future changes may break old sessions  
**Mitigation:** Version field, migration logic, backward compatibility tests

### Risk 5: Data Corruption
**Impact:** Power loss or crash during save could corrupt session  
**Mitigation:** Atomic writes, temp file pattern, validation on load, backup on write

## Success Criteria

1. ✅ All unit tests passing with >90% coverage
2. ✅ All integration tests passing
3. ✅ Can create and save sessions
4. ✅ Can load and resume sessions
5. ✅ Sessions survive process crashes
6. ✅ Concurrent access is safe
7. ✅ Linter passes without errors
8. ✅ Cyclomatic complexity ≤15 for all functions
9. ✅ Documentation is comprehensive
10. ✅ State transitions work correctly

## Examples

### Creating a Session

```go
// Create new session
cfg := DefaultConfig()
cfg.Provider = "ollama"
cfg.Model = "codellama:13b"

session := NewSession("/home/user/project", cfg)
session.SetTitle("Implement authentication")
session.AddTag("auth")
session.AddTag("security")

// Save session
storage, err := NewFileStorage("~/.spin/sessions")
if err != nil {
    log.Fatal(err)
}

if err := session.Save(storage); err != nil {
    log.Fatal(err)
}

fmt.Printf("Session created: %s\n", session.ID)
```

### Adding Turns

```go
// Add turn to session
turn := turn.NewTurn(session.ID, "Add JWT authentication")
turn.SetState(turn.StateCompleted)

if err := session.AddTurn(turn); err != nil {
    log.Fatal(err)
}

// Save updated session
if err := session.Save(storage); err != nil {
    log.Fatal(err)
}
```

### Loading a Session

```go
// Load existing session
storage, err := NewFileStorage("~/.spin/sessions")
if err != nil {
    log.Fatal(err)
}

session, err := Load(storage, sessionID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Loaded session: %s\n", session.Metadata.Title)
fmt.Printf("Total turns: %d\n", session.TurnCount())
```

### Listing Sessions

```go
// List all active sessions
filter := Filter{
    State: &StateActive,
    Limit: 10,
}

metadata, err := storage.ListMetadata(filter)
if err != nil {
    log.Fatal(err)
}

for _, m := range metadata {
    fmt.Printf("%s: %s (%d turns)\n", m.Title, m.Description, m.TotalTurns)
}
```

### Session State Transitions

```go
// Complete session
if err := session.SetState(StateCompleted); err != nil {
    log.Fatal(err)
}

// Archive completed session
if err := session.SetState(StateArchived); err != nil {
    log.Fatal(err)
}

// Save final state
if err := session.Save(storage); err != nil {
    log.Fatal(err)
}
```

## Notes

- Use `github.com/google/uuid` for UUID generation
- JSON format allows easy debugging and version control
- Atomic writes using temp file + rename pattern
- File permissions: 0700 for directory, 0600 for files
- Schema version starts at 1, increment on breaking changes
- Metadata.TotalTurns must match len(Turns)
- Metadata.TokensUsed is sum of all turn token usage
- Session files are human-readable for debugging
- Consider GZIP compression for large sessions (future enhancement)
- Consider SQLite storage backend (future enhancement)

## References

- [Go UUID Package](https://github.com/google/uuid)
- [Atomic File Writes in Go](https://stackoverflow.com/questions/11024224/atomic-file-write-operations-in-golang)
- [File Locking in Go](https://pkg.go.dev/github.com/gofrs/flock)
- [Core Module Specification](../core-module/spec.md)
- [Architecture Overview](../architecture-overview.md)
- [ROADMAP](../core-module/ROADMAP.md)

---

**Created:** 2025-10-03  
**Author:** Development Team  
**Reviewers:** TBD  
**Approved:** TBD

