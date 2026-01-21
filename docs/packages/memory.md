# Package: memory

**Location:** `internal/memory/`

## Overview

The memory package provides context offloading storage for the Spin agent. It enables the agent to store information outside the LLM's immediate context window, providing both session-scoped temporary storage (Scratchpad) and cross-session persistent storage (PersistentStore).

## Key Concepts

### Memory Scopes

| Scope | Description | Implementation |
|-------|-------------|----------------|
| `session` | Current session only, in-memory | Scratchpad |
| `thread` | Current conversation thread | (Future) |
| `persistent` | Cross-session, file-based | PersistentStore |

### Entry Types

Entries can be categorized by type for organization:

- `note` - Free-form notes
- `code` - Code snippets
- `reference` - File/URL references
- `decision` - Decisions made during the session
- `task` - Pending tasks

## Components

### MemoryStore Interface

The core interface for all memory implementations:

```go
type MemoryStore interface {
    // Put stores a value with optional configuration
    Put(ctx context.Context, key string, value string, opts PutOptions) error
    
    // Get retrieves an entry by key
    Get(ctx context.Context, key string) (*MemoryEntry, error)
    
    // Delete removes an entry by key
    Delete(ctx context.Context, key string) error
    
    // List returns keys matching a pattern
    List(ctx context.Context, pattern string) ([]string, error)
    
    // Search finds entries containing the query string
    Search(ctx context.Context, query string, topK int) ([]MemoryEntry, error)
}
```

### Scratchpad

Session-scoped in-memory storage with LRU eviction:

```go
// Create a scratchpad for the current session
pad := memory.NewScratchpad("session-123", 50)

// Store a value
pad.Put(ctx, "api_response", `{"status": "ok"}`, memory.PutOptions{})

// Retrieve a value
entry, err := pad.Get(ctx, "api_response")

// Pin important entries (prevent auto-eviction)
pad.Pin("api_response")

// Search for entries
results, _ := pad.Search(ctx, "status", 10)

// List all keys
keys, _ := pad.List(ctx, "*")

// Clear all entries
pad.Clear()
```

**Features:**
- LRU eviction when at capacity
- Pinned entries protection
- Access count tracking
- Case-insensitive search
- Pattern-based listing (`*` wildcard)

### PersistentStore

File-based cross-session storage:

```go
// Create a persistent store
store, err := memory.NewPersistentStore("~/.spin/memory")

// Store with namespace
store.Put(ctx, "preference", "tabs over spaces", memory.PutOptions{
    Namespace: "preferences",
    Tags:      []string{"code-style"},
})

// Retrieve
entry, err := store.Get(ctx, "preference")

// Search across all entries
results, _ := store.Search(ctx, "tabs", 10)

// Close when done
store.Close()
```

**Features:**
- Namespace organization (subdirectories)
- Atomic writes (crash-safe)
- Index rebuilding on startup
- TTL support
- Tag-based organization

## Memory Tools

The memory package integrates with the tools system to provide LLM-accessible memory operations.

### ScratchpadTool

Session-scoped memory tool for the LLM:

```go
// Created automatically when memory is enabled
tool := tools.NewScratchpadTool(scratchpad)
```

**Operations:**
- `put` - Store a value with optional namespace and tags
- `get` - Retrieve a value by key
- `delete` - Remove a value
- `list` - List keys matching a pattern
- `search` - Search entries by query
- `pin` - Pin entry to prevent eviction
- `unpin` - Unpin entry
- `clear` - Clear all entries

### MemoryTool

Persistent memory tool for cross-session storage:

```go
tool := tools.NewMemoryTool(persistentStore)
```

**Operations:**
- `put` - Store a value persistently
- `get` - Retrieve a value
- `delete` - Remove a value
- `list` - List keys matching a pattern
- `search` - Search entries by query

## Auto-Offloading

The package provides automatic context offloading to reduce context window usage.

### ContextAnalyzer

Analyzes messages to identify offloadable content:

```go
analyzer := memory.NewDefaultContextAnalyzer()
candidates := analyzer.Analyze(messages)
```

**Identified Content Types:**
- Large code blocks (configurable threshold)
- Long tool outputs
- Decision statements

### AutoOffloader

Automatically offloads content when approaching context limits:

```go
offloader := memory.NewAutoOffloader(memory.AutoOffloaderConfig{
    Scratchpad: scratchpad,
    Persistent: persistent,
    Threshold:  0.7, // Trigger at 70% usage
})

// Check if offloading is needed
if offloader.ShouldOffload(currentTokens, maxTokens) {
    modified, results, err := offloader.Offload(ctx, messages)
}

// Recall offloaded content
content, err := offloader.Recall(ctx, "code_0_0")
```

## Session Continuity

The package supports session handoff for continuing work across sessions.

### SessionHandoff

Manages context transfer between sessions:

```go
handoff := memory.NewSessionHandoff(persistentStore, summarizer)

// Save session state
data := memory.HandoffData{
    SessionID:    "session-123",
    Summary:      "Working on authentication feature",
    PendingTasks: []string{"Add unit tests"},
    Decisions:    []string{"Use JWT for auth"},
    WorkDir:      "/home/user/project",
}
err := handoff.SaveSession(ctx, data)

// Load previous session
loaded, err := handoff.LoadSession(ctx, "session-123")

// Build continuation prompt
prompt := handoff.BuildContinuationPrompt(loaded)
```

### SimpleSummarizer

Basic summarization by truncation (when no LLM available):

```go
summarizer := memory.NewSimpleSummarizer(500)
summary, err := summarizer.Summarize(ctx, content, maxTokens)
```

## Configuration

Memory is configured via the `memory` section in `ConfigV2`:

```yaml
memory:
  scratchpad:
    enabled: true
    max_entries: 50
    auto_evict: true
    
  persistent:
    enabled: false
    base_path: ~/.spin/memory
```

### Configuration Options

#### Scratchpad

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable scratchpad |
| `max_entries` | int | `50` | Maximum entries before eviction |
| `auto_evict` | bool | `true` | Enable LRU auto-eviction |

#### Persistent

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable persistent storage |
| `base_path` | string | `~/.spin/memory` | Storage directory |

## PutOptions

Configure individual Put operations:

```go
type PutOptions struct {
    TTL       time.Duration // 0 = no expiry
    Namespace string        // Logical grouping (default: "default")
    Tags      []string      // For filtering
    Overwrite bool          // Replace if exists
}
```

## Error Handling

The package defines sentinel errors:

```go
memory.ErrNotFound   // Key does not exist
memory.ErrKeyExists  // Key exists and Overwrite=false
memory.ErrEmptyKey   // Empty key provided
```

## Thread Safety

All implementations are safe for concurrent use:
- Scratchpad uses `sync.RWMutex` for thread-safe operations
- PersistentStore uses `sync.RWMutex` for index and file operations
- AutoOffloader uses `sync.Mutex` for offload operations

## Usage Examples

### Session Working Memory

```go
// Create scratchpad at session start
pad := memory.NewScratchpad(sessionID, 50)

// Store intermediate results during execution
pad.Put(ctx, "file_list", strings.Join(files, "\n"), memory.PutOptions{
    Namespace: "context",
})

// Pin important context
pad.Pin("file_list")

// Search later in the session
results, _ := pad.Search(ctx, "main.go", 5)
```

### Cross-Session Preferences

```go
// Create persistent store
store, _ := memory.NewPersistentStore("~/.spin/memory")
defer store.Close()

// Store user preference
store.Put(ctx, "indent_style", "tabs", memory.PutOptions{
    Namespace: "preferences",
    Tags:      []string{"formatting"},
})

// In a new session, recall preference
entry, err := store.Get(ctx, "indent_style")
if err == nil {
    fmt.Printf("Using indent style: %s\n", entry.Value)
}
```

### Storing Decisions

```go
// Store a decision with context
pad.Put(ctx, "decision_auth", "Using JWT for authentication because...", memory.PutOptions{
    Namespace: "decisions",
})

// Later, search for decisions
results, _ := pad.Search(ctx, "authentication", 10)
```

### Auto-Offloading Integration

```go
// Create offloader with both stores
offloader := memory.NewAutoOffloader(memory.AutoOffloaderConfig{
    Scratchpad: scratchpad,
    Persistent: persistent,
    Threshold:  0.7,
})

// In agent loop, check and offload if needed
if offloader.ShouldOffload(tokenCount, maxTokens) {
    messages, results, _ := offloader.Offload(ctx, messages)
    for _, r := range results {
        log.Printf("Offloaded %s: %s (saved %d tokens)", r.Key, r.Reason, r.TokensSaved)
    }
}
```

### Session Handoff

```go
// At session end
handoff := memory.NewSessionHandoff(persistentStore, summarizer)
data := memory.HandoffData{
    SessionID:    sessionID,
    Summary:      summary,
    WorkDir:      workDir,
    PendingTasks: pendingTasks,
    Decisions:    decisions,
}
handoff.SaveSession(ctx, data)

// At new session start
if loaded, err := handoff.LoadSession(ctx, previousSessionID); err == nil {
    continuationPrompt := handoff.BuildContinuationPrompt(loaded)
    // Prepend to system prompt or first message
}
```

## Performance

| Operation | Scratchpad | PersistentStore |
|-----------|------------|-----------------|
| Get | O(1) | O(1) index + file read |
| Put | O(1) | O(1) index + file write |
| Delete | O(1) | O(1) index + file delete |
| List | O(n) | O(n) |
| Search | O(n) | O(n) with file reads |

## Related Packages

- `internal/config` - Memory configuration
- `internal/tools` - ScratchpadTool and MemoryTool
- `internal/conversation` - Memory service integration
- `internal/session` - Session management
- `internal/history` - Conversation history

## Implementation Status

| Feature | Status |
|---------|--------|
| MemoryStore Interface | ✅ Complete |
| Scratchpad | ✅ Complete |
| PersistentStore | ✅ Complete |
| ScratchpadTool | ✅ Complete |
| MemoryTool | ✅ Complete |
| ContextAnalyzer | ✅ Complete |
| AutoOffloader | ✅ Complete |
| SessionHandoff | ✅ Complete |
| Semantic Search | 🔮 Future (embeddings) |
| ACE Integration | 🔮 Future |
