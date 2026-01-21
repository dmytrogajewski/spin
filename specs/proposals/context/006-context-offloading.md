# Proposal: Context Offloading to External Storage

**ID**: PROP-CONTEXT-006  
**Title**: Context Offloading via Persistent Memory Tools  
**Status**: Draft  
**Created**: 2025-01-20  
**Author**: AI Assistant  
**References**: [LangChain Context Engineering](https://github.com/langchain-ai/how_to_fix_your_context)

## Summary

Implement context offloading mechanisms that store information outside the LLM's immediate context window, enabling persistent memory across sessions and reducing context overhead through dedicated storage tools.

## Problem Statement

### Current State

Spin's context handling is primarily in-memory and session-scoped:
- Conversation history exists only during session
- ACE playbook provides cross-session learning (bullets)
- No explicit scratchpad or working memory tool
- No structured way to store/retrieve session artifacts

### Identified Issues

1. **Session Isolation**: Information is lost between sessions
2. **Context Overhead**: All working information must stay in context
3. **No Structured Memory**: Can't explicitly save/load specific items
4. **Trajectory Accumulation**: Execution context grows unbounded

### Context Rot Risks

- **Context Distraction**: Too much accumulated context
- **Lost Continuity**: Previous session context unavailable
- **Redundant Retrieval**: Re-discovering same information repeatedly

### Two Patterns from LangChain Research

1. **Temporary Scratchpad**: Session-scoped working memory
2. **Persistent Store**: Cross-session key-value storage

## Proposed Solution

### 1. Memory Store Interface

Define unified interface for context offloading.

```go
// internal/memory/store.go

type MemoryStore interface {
    // Put stores a value with optional TTL
    Put(ctx context.Context, key string, value interface{}, opts PutOptions) error
    
    // Get retrieves a value by key
    Get(ctx context.Context, key string) (interface{}, error)
    
    // Delete removes a value
    Delete(ctx context.Context, key string) error
    
    // List returns keys matching a pattern
    List(ctx context.Context, pattern string) ([]string, error)
    
    // Search finds entries semantically similar to query
    Search(ctx context.Context, query string, topK int) ([]MemoryEntry, error)
}

type PutOptions struct {
    TTL         time.Duration   // 0 = no expiry
    Namespace   string          // Logical grouping
    Tags        []string        // For filtering
    Overwrite   bool            // Replace if exists
}

type MemoryEntry struct {
    Key       string
    Value     interface{}
    Namespace string
    Tags      []string
    CreatedAt time.Time
    UpdatedAt time.Time
    TTL       time.Duration
}

type MemoryScope string

const (
    ScopeSession    MemoryScope = "session"    // Current session only
    ScopeThread     MemoryScope = "thread"     // Current conversation thread
    ScopePersistent MemoryScope = "persistent" // Cross-session
)
```

### 2. Session Scratchpad

Temporary working memory for current session.

```go
// internal/memory/scratchpad.go

type Scratchpad struct {
    entries   map[string]*ScratchpadEntry
    maxSize   int
    mu        sync.RWMutex
    sessionID string
}

type ScratchpadEntry struct {
    Key       string
    Value     string
    Type      EntryType       // note, code, reference, decision
    CreatedAt time.Time
    AccessCount int
    Pinned    bool            // Don't auto-evict
}

type EntryType string

const (
    EntryTypeNote      EntryType = "note"       // Free-form notes
    EntryTypeCode      EntryType = "code"       // Code snippets
    EntryTypeReference EntryType = "reference"  // File/URL references
    EntryTypeDecision  EntryType = "decision"   // Decisions made
    EntryTypeTask      EntryType = "task"       // Pending tasks
)

func NewScratchpad(sessionID string, maxSize int) *Scratchpad {
    return &Scratchpad{
        entries:   make(map[string]*ScratchpadEntry),
        maxSize:   maxSize,
        sessionID: sessionID,
    }
}

func (s *Scratchpad) Put(ctx context.Context, key string, value interface{}, opts PutOptions) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Evict if at capacity
    if len(s.entries) >= s.maxSize && s.entries[key] == nil {
        s.evictLRU()
    }
    
    entry := &ScratchpadEntry{
        Key:       key,
        Value:     toString(value),
        Type:      inferEntryType(value),
        CreatedAt: time.Now(),
    }
    
    s.entries[key] = entry
    return nil
}

func (s *Scratchpad) Get(ctx context.Context, key string) (interface{}, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    entry, ok := s.entries[key]
    if !ok {
        return nil, ErrNotFound
    }
    
    entry.AccessCount++
    return entry.Value, nil
}

func (s *Scratchpad) Search(ctx context.Context, query string, topK int) ([]MemoryEntry, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Simple keyword matching for scratchpad
    results := []MemoryEntry{}
    queryLower := strings.ToLower(query)
    
    for _, entry := range s.entries {
        if strings.Contains(strings.ToLower(entry.Key), queryLower) ||
           strings.Contains(strings.ToLower(entry.Value), queryLower) {
            results = append(results, MemoryEntry{
                Key:       entry.Key,
                Value:     entry.Value,
                CreatedAt: entry.CreatedAt,
            })
        }
    }
    
    // Sort by relevance (access count) and return topK
    sort.Slice(results, func(i, j int) bool {
        return s.entries[results[i].Key].AccessCount > s.entries[results[j].Key].AccessCount
    })
    
    if len(results) > topK {
        results = results[:topK]
    }
    
    return results, nil
}

func (s *Scratchpad) evictLRU() {
    var lruKey string
    var lruAccess int = math.MaxInt
    
    for key, entry := range s.entries {
        if entry.Pinned {
            continue
        }
        if entry.AccessCount < lruAccess {
            lruAccess = entry.AccessCount
            lruKey = key
        }
    }
    
    if lruKey != "" {
        delete(s.entries, lruKey)
    }
}
```

### 3. Persistent Memory Store

Cross-session memory using file-based storage.

```go
// internal/memory/persistent.go

type PersistentStore struct {
    basePath   string
    index      *MemoryIndex
    embedder   embedder.Embedder
    mu         sync.RWMutex
}

type MemoryIndex struct {
    Entries   map[string]*IndexEntry
    Embeddings map[string][]float32
    hnswIndex *hnsw.Index
}

type IndexEntry struct {
    Key         string
    Namespace   string
    Tags        []string
    FilePath    string      // Path to value file
    CreatedAt   time.Time
    UpdatedAt   time.Time
    AccessCount int
    Size        int64
}

func NewPersistentStore(basePath string, embedder embedder.Embedder) (*PersistentStore, error) {
    store := &PersistentStore{
        basePath: basePath,
        embedder: embedder,
    }
    
    // Load or create index
    index, err := store.loadOrCreateIndex()
    if err != nil {
        return nil, err
    }
    store.index = index
    
    return store, nil
}

func (s *PersistentStore) Put(ctx context.Context, key string, value interface{}, opts PutOptions) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Serialize value
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    
    // Compute file path
    namespace := opts.Namespace
    if namespace == "" {
        namespace = "default"
    }
    filePath := filepath.Join(s.basePath, namespace, key+".json")
    
    // Write to file
    if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
        return err
    }
    if err := os.WriteFile(filePath, data, 0644); err != nil {
        return err
    }
    
    // Update index
    entry := &IndexEntry{
        Key:       key,
        Namespace: namespace,
        Tags:      opts.Tags,
        FilePath:  filePath,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Size:      int64(len(data)),
    }
    s.index.Entries[s.fullKey(namespace, key)] = entry
    
    // Compute embedding for semantic search
    if s.embedder != nil {
        embedding, err := s.embedder.Embed(ctx, toString(value))
        if err == nil {
            s.index.Embeddings[s.fullKey(namespace, key)] = embedding
            s.index.hnswIndex.Add(embedding, s.fullKey(namespace, key))
        }
    }
    
    // Persist index
    return s.saveIndex()
}

func (s *PersistentStore) Get(ctx context.Context, key string) (interface{}, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    entry, ok := s.index.Entries[key]
    if !ok {
        // Try with default namespace
        entry, ok = s.index.Entries[s.fullKey("default", key)]
        if !ok {
            return nil, ErrNotFound
        }
    }
    
    data, err := os.ReadFile(entry.FilePath)
    if err != nil {
        return nil, err
    }
    
    entry.AccessCount++
    
    var value interface{}
    if err := json.Unmarshal(data, &value); err != nil {
        return nil, err
    }
    
    return value, nil
}

func (s *PersistentStore) Search(ctx context.Context, query string, topK int) ([]MemoryEntry, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if s.embedder == nil || s.index.hnswIndex == nil {
        // Fall back to keyword search
        return s.keywordSearch(query, topK)
    }
    
    // Embed query
    embedding, err := s.embedder.Embed(ctx, query)
    if err != nil {
        return s.keywordSearch(query, topK)
    }
    
    // Semantic search
    neighbors, err := s.index.hnswIndex.SearchKNN(embedding, topK)
    if err != nil {
        return nil, err
    }
    
    results := make([]MemoryEntry, 0, len(neighbors))
    for _, key := range neighbors {
        entry := s.index.Entries[key]
        value, _ := s.Get(ctx, key)
        results = append(results, MemoryEntry{
            Key:       entry.Key,
            Value:     value,
            Namespace: entry.Namespace,
            Tags:      entry.Tags,
            CreatedAt: entry.CreatedAt,
        })
    }
    
    return results, nil
}
```

### 4. Memory Tools

Expose memory operations as LLM tools.

```go
// internal/tools/memory_tools.go

// Scratchpad Tool - Session-scoped
type ScratchpadTool struct {
    scratchpad *Scratchpad
}

func (t *ScratchpadTool) Definition() llm.Tool {
    return llm.Tool{
        Name:        "scratchpad",
        Description: "Store and retrieve notes, code snippets, and references during this session. Use for temporary working memory.",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "action": {
                    "type": "string",
                    "enum": []string{"put", "get", "list", "search", "delete"},
                    "description": "Action to perform",
                },
                "key": {
                    "type": "string",
                    "description": "Key for the entry (required for put/get/delete)",
                },
                "value": {
                    "type": "string",
                    "description": "Value to store (required for put)",
                },
                "query": {
                    "type": "string",
                    "description": "Search query (required for search)",
                },
                "pin": {
                    "type": "boolean",
                    "description": "Pin entry to prevent auto-eviction",
                },
            },
            "required": []string{"action"},
        },
    }
}

func (t *ScratchpadTool) Execute(ctx context.Context, params ScratchpadParams) (*ToolResult, error) {
    switch params.Action {
    case "put":
        err := t.scratchpad.Put(ctx, params.Key, params.Value, PutOptions{})
        if err != nil {
            return &ToolResult{Error: err.Error()}, nil
        }
        return &ToolResult{Output: fmt.Sprintf("Stored '%s' in scratchpad", params.Key)}, nil
        
    case "get":
        value, err := t.scratchpad.Get(ctx, params.Key)
        if err != nil {
            return &ToolResult{Error: fmt.Sprintf("Key '%s' not found", params.Key)}, nil
        }
        return &ToolResult{Output: toString(value)}, nil
        
    case "list":
        keys, _ := t.scratchpad.List(ctx, "*")
        return &ToolResult{Output: strings.Join(keys, "\n")}, nil
        
    case "search":
        results, _ := t.scratchpad.Search(ctx, params.Query, 5)
        output := formatSearchResults(results)
        return &ToolResult{Output: output}, nil
        
    case "delete":
        t.scratchpad.Delete(ctx, params.Key)
        return &ToolResult{Output: fmt.Sprintf("Deleted '%s' from scratchpad", params.Key)}, nil
    }
    
    return &ToolResult{Error: "Unknown action"}, nil
}

// Memory Tool - Persistent cross-session
type MemoryTool struct {
    store *PersistentStore
}

func (t *MemoryTool) Definition() llm.Tool {
    return llm.Tool{
        Name:        "memory",
        Description: "Store and retrieve information persistently across sessions. Use for facts, decisions, preferences that should be remembered.",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "action": {
                    "type": "string",
                    "enum": []string{"store", "recall", "search", "forget", "list"},
                    "description": "Action to perform",
                },
                "key": {
                    "type": "string",
                    "description": "Key for the memory",
                },
                "value": {
                    "type": "string",
                    "description": "Information to store",
                },
                "query": {
                    "type": "string",
                    "description": "Search query for semantic recall",
                },
                "namespace": {
                    "type": "string",
                    "description": "Namespace for organizing memories (e.g., 'project', 'preferences', 'decisions')",
                },
                "tags": {
                    "type": "array",
                    "items": {"type": "string"},
                    "description": "Tags for filtering",
                },
            },
            "required": []string{"action"},
        },
    }
}

func (t *MemoryTool) Execute(ctx context.Context, params MemoryParams) (*ToolResult, error) {
    switch params.Action {
    case "store":
        err := t.store.Put(ctx, params.Key, params.Value, PutOptions{
            Namespace: params.Namespace,
            Tags:      params.Tags,
        })
        if err != nil {
            return &ToolResult{Error: err.Error()}, nil
        }
        return &ToolResult{Output: fmt.Sprintf("Stored memory '%s' (namespace: %s)", params.Key, params.Namespace)}, nil
        
    case "recall":
        value, err := t.store.Get(ctx, params.Key)
        if err != nil {
            return &ToolResult{Output: fmt.Sprintf("No memory found for '%s'", params.Key)}, nil
        }
        return &ToolResult{Output: toString(value)}, nil
        
    case "search":
        results, _ := t.store.Search(ctx, params.Query, 5)
        if len(results) == 0 {
            return &ToolResult{Output: "No relevant memories found"}, nil
        }
        output := formatSearchResults(results)
        return &ToolResult{Output: output}, nil
        
    case "forget":
        t.store.Delete(ctx, params.Key)
        return &ToolResult{Output: fmt.Sprintf("Forgot memory '%s'", params.Key)}, nil
        
    case "list":
        keys, _ := t.store.List(ctx, params.Namespace+"/*")
        return &ToolResult{Output: strings.Join(keys, "\n")}, nil
    }
    
    return &ToolResult{Error: "Unknown action"}, nil
}
```

### 5. Automatic Context Offloading

Automatically offload context when approaching limits.

```go
// internal/context/auto_offload.go

type AutoOffloader struct {
    scratchpad    *Scratchpad
    store         *PersistentStore
    threshold     float64  // Token usage threshold (e.g., 0.7)
    analyzer      ContextAnalyzer
}

type ContextAnalyzer interface {
    // Analyze identifies offloadable content in messages
    Analyze(messages []Message) []OffloadCandidate
}

type OffloadCandidate struct {
    MessageIndex int
    Content      string
    Reason       string           // Why this should be offloaded
    Destination  MemoryScope      // Where to offload
    Key          string           // Suggested key
    Priority     int              // Higher = offload first
}

func (o *AutoOffloader) ShouldOffload(messages []Message, maxTokens int) bool {
    currentTokens := countTotalTokens(messages)
    return float64(currentTokens)/float64(maxTokens) > o.threshold
}

func (o *AutoOffloader) Offload(ctx context.Context, messages []Message) ([]Message, error) {
    candidates := o.analyzer.Analyze(messages)
    
    // Sort by priority
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].Priority > candidates[j].Priority
    })
    
    offloadedIndices := make(map[int]bool)
    
    for _, candidate := range candidates {
        // Store to appropriate destination
        switch candidate.Destination {
        case ScopeSession:
            o.scratchpad.Put(ctx, candidate.Key, candidate.Content, PutOptions{})
        case ScopePersistent:
            o.store.Put(ctx, candidate.Key, candidate.Content, PutOptions{})
        }
        
        offloadedIndices[candidate.MessageIndex] = true
    }
    
    // Replace offloaded content with references
    result := make([]Message, len(messages))
    for i, msg := range messages {
        if offloadedIndices[i] {
            candidate := findCandidate(candidates, i)
            result[i] = Message{
                Role:    msg.Role,
                Content: fmt.Sprintf("[Content offloaded to %s as '%s']", candidate.Destination, candidate.Key),
            }
        } else {
            result[i] = msg
        }
    }
    
    return result, nil
}

// Default analyzer implementation
type DefaultContextAnalyzer struct{}

func (a *DefaultContextAnalyzer) Analyze(messages []Message) []OffloadCandidate {
    candidates := []OffloadCandidate{}
    
    for i, msg := range messages {
        // Large code blocks -> scratchpad
        if codeBlocks := extractCodeBlocks(msg.Content); len(codeBlocks) > 0 {
            for j, block := range codeBlocks {
                if countTokens(block) > 500 {
                    candidates = append(candidates, OffloadCandidate{
                        MessageIndex: i,
                        Content:      block,
                        Reason:       "Large code block",
                        Destination:  ScopeSession,
                        Key:          fmt.Sprintf("code_%d_%d", i, j),
                        Priority:     100,
                    })
                }
            }
        }
        
        // Long tool outputs -> scratchpad
        if msg.Role == "tool" && countTokens(msg.Content) > 1000 {
            candidates = append(candidates, OffloadCandidate{
                MessageIndex: i,
                Content:      msg.Content,
                Reason:       "Large tool output",
                Destination:  ScopeSession,
                Key:          fmt.Sprintf("tool_output_%d", i),
                Priority:     80,
            })
        }
        
        // Decisions -> persistent
        if containsDecision(msg.Content) {
            candidates = append(candidates, OffloadCandidate{
                MessageIndex: i,
                Content:      extractDecision(msg.Content),
                Reason:       "Important decision",
                Destination:  ScopePersistent,
                Key:          fmt.Sprintf("decision_%d", time.Now().Unix()),
                Priority:     50,
            })
        }
    }
    
    return candidates
}
```

### 6. Session Handoff

Transfer context between sessions.

```go
// internal/memory/session_handoff.go

type SessionHandoff struct {
    store      *PersistentStore
    summarizer Summarizer
}

type HandoffData struct {
    SessionID       string
    Summary         string
    PendingTasks    []string
    Decisions       []string
    KeyReferences   map[string]string
    LastActivity    time.Time
}

func (h *SessionHandoff) SaveSession(ctx context.Context, sessionID string, tc *trajectory.Context) error {
    // Summarize the session
    summary, err := h.summarizer.Summarize(ctx, tc.String(), SummarizeOptions{
        MaxTokens:   500,
        Style:       StyleBrief,
    })
    if err != nil {
        return err
    }
    
    // Extract pending tasks and decisions
    pending := extractPendingTasks(tc)
    decisions := extractDecisions(tc)
    
    handoff := HandoffData{
        SessionID:     sessionID,
        Summary:       summary.Summary,
        PendingTasks:  pending,
        Decisions:     decisions,
        LastActivity:  time.Now(),
    }
    
    return h.store.Put(ctx, "session_"+sessionID, handoff, PutOptions{
        Namespace: "sessions",
        TTL:       7 * 24 * time.Hour, // Keep for 7 days
    })
}

func (h *SessionHandoff) LoadPreviousSession(ctx context.Context, sessionID string) (*HandoffData, error) {
    value, err := h.store.Get(ctx, "session_"+sessionID)
    if err != nil {
        return nil, err
    }
    
    handoff, ok := value.(*HandoffData)
    if !ok {
        return nil, errors.New("invalid handoff data")
    }
    
    return handoff, nil
}

func (h *SessionHandoff) BuildContinuationPrompt(handoff *HandoffData) string {
    var sb strings.Builder
    
    sb.WriteString("[Continuing from previous session]\n\n")
    sb.WriteString("Previous session summary:\n")
    sb.WriteString(handoff.Summary)
    sb.WriteString("\n\n")
    
    if len(handoff.PendingTasks) > 0 {
        sb.WriteString("Pending tasks:\n")
        for _, task := range handoff.PendingTasks {
            sb.WriteString("- " + task + "\n")
        }
        sb.WriteString("\n")
    }
    
    if len(handoff.Decisions) > 0 {
        sb.WriteString("Key decisions made:\n")
        for _, decision := range handoff.Decisions {
            sb.WriteString("- " + decision + "\n")
        }
    }
    
    return sb.String()
}
```

### 7. Integration with ACE System

Connect offloading with ACE's persistent learning.

```go
// internal/ace/offload_adapter.go

type OffloadAdapter struct {
    playbook   *playbook.Playbook
    memStore   *PersistentStore
}

// ConvertMemoryToBullets transforms frequently accessed memories into bullets
func (a *OffloadAdapter) ConvertMemoryToBullets(ctx context.Context) error {
    // Find high-value memories
    entries, err := a.memStore.List(ctx, "decisions/*")
    if err != nil {
        return err
    }
    
    for _, key := range entries {
        entry, _ := a.memStore.Get(ctx, key)
        memEntry := entry.(*MemoryEntry)
        
        // Check if accessed frequently
        if memEntry.AccessCount >= 3 {
            // Create bullet from memory
            bullet := &bullet.Bullet{
                ID:      uuid.New().String(),
                Content: toString(memEntry.Value),
                Tags:    map[string]string{"source": "memory", "key": key},
            }
            
            if err := a.playbook.Add(ctx, bullet); err != nil {
                continue
            }
            
            // Remove from memory (now in playbook)
            a.memStore.Delete(ctx, key)
        }
    }
    
    return nil
}
```

## Configuration

```yaml
memory:
  scratchpad:
    enabled: true
    max_entries: 50
    auto_evict: true
    eviction_strategy: "lru"
    
  persistent:
    enabled: true
    base_path: "${HOME}/.spin/memory"
    embeddings:
      enabled: true
      model: "ollama:nomic-embed-text"
    namespaces:
      - decisions
      - preferences
      - project_context
      - sessions
    retention:
      default: 30d
      decisions: 90d
      preferences: 365d
      
  auto_offload:
    enabled: true
    threshold: 0.7        # Trigger at 70% context usage
    analyze_interval: 5   # Check every 5 turns
    
  session_handoff:
    enabled: true
    retention: 7d
    auto_save: true       # Save on session end
    
  ace_integration:
    convert_threshold: 3  # Access count to convert to bullet
    check_interval: 1h
```

## Implementation Plan

### Phase 1: Core Stores (Week 1) - COMPLETED
1. [x] Implement `MemoryStore` interface
2. [x] Build `Scratchpad` (session-scoped)
3. [x] Build `PersistentStore` (file-based)
4. [x] Add basic CRUD operations
5. [x] Add Memory configuration to ConfigV2
6. [x] Documentation: `docs/packages/memory.md`
7. [x] FRD: `specs/frds/FRD-20260121-001-memory-core-stores.md`

### Phase 2: Memory Tools (Week 2) - COMPLETED
1. [x] Implement `ScratchpadTool`
2. [x] Implement `MemoryTool`
3. [x] Register tools with registry
4. [ ] Add semantic search (deferred - requires embeddings)

### Phase 3: Auto-Offloading (Week 3) - COMPLETED
1. [x] Implement `ContextAnalyzer`
2. [x] Build `AutoOffloader`
3. [ ] Integrate with agent loop (requires agent changes)
4. [ ] Add offload events (requires event system changes)

### Phase 4: Session Continuity (Week 4) - COMPLETED
1. [x] Implement `SessionHandoff`
2. [x] Add session save/load
3. [x] Build continuation prompt
4. [ ] Integrate with ACE (deferred - requires ACE changes)

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Effective Session Length | ~30 turns | Unlimited |
| Cross-Session Continuity | None | 90%+ recall |
| Context Overhead | 100% | 60% (40% offloaded) |
| Memory Retrieval Latency | N/A | <100ms |

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Memory corruption | High | Atomic writes, backups |
| Lost references | Medium | Keep minimal context pointers |
| Search quality | Medium | Embeddings + keyword fallback |
| Storage growth | Low | TTL, automatic cleanup |

## Examples

### Example 1: Scratchpad Usage

```
User: Let me save this API response for later
Agent: [Uses scratchpad tool]
> scratchpad(action="put", key="api_response", value="{...}")

[Later in conversation]

User: What was in that API response?
Agent: [Uses scratchpad tool]
> scratchpad(action="get", key="api_response")
```

### Example 2: Persistent Memory

```
User: Remember that I prefer tabs over spaces
Agent: [Uses memory tool]
> memory(action="store", key="code_style_preference", value="Prefers tabs over spaces for indentation", namespace="preferences")

[New session]

User: Format this code
Agent: [Uses memory tool]
> memory(action="recall", key="code_style_preference")
Agent: I'll format using tabs as per your preference.
```

### Example 3: Session Handoff

```
[End of session 1]
Saved: Summary of session, 2 pending tasks, 3 key decisions

[Start of session 2]
[Continuing from previous session]
Previous session summary: Was working on authentication bug in auth.go...
Pending tasks:
- Fix refreshToken() issue
- Add unit tests for token validation
Key decisions:
- Using JWT validation library
- Token expiry set to 1 hour
```

## References

- [LangChain Context Offloading](https://github.com/langchain-ai/how_to_fix_your_context/blob/main/notebooks/6_context_offloading.ipynb)
- [LangGraph Memory](https://langchain-ai.github.io/langgraph/concepts/memory/)
- [Persistent Memory in LLM Agents](https://arxiv.org/abs/2310.08560)
