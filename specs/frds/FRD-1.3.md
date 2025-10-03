# FRD-1.3: History Management

**Feature ID:** 1.3  
**Feature Name:** History Management  
**Phase:** Phase 1 - State Management  
**Priority:** P1 (Critical)  
**Estimated Effort:** 10 hours  
**Status:** Ready for Implementation

---

## Overview

Implement comprehensive conversation history management with token-aware truncation, message storage, and efficient retrieval. This feature provides the foundation for maintaining conversation context within token budget constraints while preserving critical information for the agent's decision-making process.

## Context

**History** manages the conversation message history between the user and AI, including system messages, user inputs, assistant responses, and tool interactions. This is critical for:

- Maintaining conversation context for LLM
- Implementing token-aware truncation when context exceeds limits
- Preserving conversation history for replay and analysis
- Supporting context window management across different LLM providers
- Enabling conversation export and archival

The History component must efficiently manage potentially large conversations while respecting token budget constraints and ensuring thread-safe concurrent access.

## Definition of Ready (DoR)

- [x] Feature 1.2 (Turn State Machine) completed
- [ ] Tokenizer interface defined or approach determined
- [ ] Truncation strategy documented
- [ ] Message format aligned with LLM provider expectations

## Definition of Done (DoD)

- [ ] `history.go` implemented with History struct
- [ ] AddMessage() method with thread-safety
- [ ] Messages() retrieval method
- [ ] Truncate() with token budget awareness
- [ ] Export() for saving history to file
- [ ] Message struct with all fields (Role, Content, ToolCalls, etc.)
- [ ] Token counting integration (or stub for future integration)
- [ ] Smart truncation preserving system message
- [ ] Unit tests for history operations (>90% coverage)
- [ ] Truncation algorithm tests
- [ ] Concurrent access tests
- [ ] Godoc comments for all exported symbols
- [ ] Code analyzed with uast/herr (complexity <15)
- [ ] All linters passing

---

## Requirements

### Functional Requirements

#### FR-1.3.1: Message Structure

**Description:** Define comprehensive message structure compatible with LLM APIs.

**Type:**

```go
type Message struct {
    // Identity
    ID        string    // Unique message ID
    
    // Content
    Role      Role      // Message role (system, user, assistant, tool)
    Content   string    // Message content
    
    // Tool Interaction
    ToolCalls []ToolCall   // Tool calls made by assistant
    ToolCallID string      // Tool call ID (for tool role messages)
    
    // Metadata
    Timestamp time.Time  // When message was created
    Tokens    int        // Token count for this message
    
    // Optional metadata
    Name      string     // Optional name field
    Metadata  map[string]interface{} // Extensible metadata
}

// Role represents message role
type Role string

const (
    RoleSystem    Role = "system"
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleTool      Role = "tool"
)

// ToolCall represents a tool invocation in assistant message
type ToolCall struct {
    ID       string                 `json:"id"`
    Type     string                 `json:"type"`      // Usually "function"
    Function ToolCallFunction       `json:"function"`
}

type ToolCallFunction struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments"` // JSON string
}
```

**Acceptance Criteria:**
- Message struct supports all LLM API message formats
- Role enum covers all standard roles
- ToolCall structure compatible with OpenAI format
- JSON serialization works correctly
- All fields documented

---

#### FR-1.3.2: History Structure

**Description:** Implement History struct for managing message collection.

**Type:**

```go
type History struct {
    messages  []Message        // Message history
    maxTokens int              // Maximum token budget
    tokenizer Tokenizer        // Token counting interface
    mu        sync.RWMutex     // Thread-safety
}

// Tokenizer interface for counting tokens
type Tokenizer interface {
    Count(text string) int
    CountMessages(messages []Message) int
}
```

**Methods:**

```go
// NewHistory creates a new history manager
func NewHistory(maxTokens int, tokenizer Tokenizer) *History

// NewHistoryWithDefaults creates history with sensible defaults
func NewHistoryWithDefaults() *History
```

**Acceptance Criteria:**
- History struct properly encapsulates message management
- Thread-safe using RWMutex
- Tokenizer interface allows pluggable token counting
- Constructor validates inputs
- Default constructor provides reasonable defaults

---

#### FR-1.3.3: Add Message

**Description:** Thread-safe message addition with automatic token tracking.

**Method:**

```go
// AddMessage appends a message to history
func (h *History) AddMessage(msg Message) error

// AddSystemMessage adds a system message (convenience)
func (h *History) AddSystemMessage(content string) error

// AddUserMessage adds a user message (convenience)
func (h *History) AddUserMessage(content string) error

// AddAssistantMessage adds an assistant message (convenience)
func (h *History) AddAssistantMessage(content string) error

// AddToolMessage adds a tool result message (convenience)
func (h *History) AddToolMessage(toolCallID, content string) error
```

**Behavior:**
- Automatically count tokens for message
- Validate message before adding
- Thread-safe addition
- Return error for invalid messages
- Update total token count

**Acceptance Criteria:**
- Thread-safe message addition
- Token counting on add
- Convenience methods for common roles
- Input validation
- Error handling for invalid messages

---

#### FR-1.3.4: Retrieve Messages

**Description:** Efficient message retrieval with optional filtering.

**Methods:**

```go
// Messages returns all messages (copy for safety)
func (h *History) Messages() []Message

// MessagesForLLM returns messages formatted for LLM API
func (h *History) MessagesForLLM() []Message

// GetMessage retrieves a specific message by ID
func (h *History) GetMessage(id string) (*Message, error)

// LastMessage returns the most recent message
func (h *History) LastMessage() (*Message, error)

// MessagesSince returns messages after a specific timestamp
func (h *History) MessagesSince(timestamp time.Time) []Message

// Count returns the number of messages
func (h *History) Count() int

// TokenCount returns total tokens in history
func (h *History) TokenCount() int

// IsEmpty returns true if history has no messages
func (h *History) IsEmpty() bool
```

**Acceptance Criteria:**
- Messages() returns defensive copy
- MessagesForLLM() formats correctly for API
- Efficient retrieval operations
- No data races on concurrent reads
- Proper error handling

---

#### FR-1.3.5: Token-Aware Truncation

**Description:** Intelligent history truncation to fit within token budget.

**Method:**

```go
// Truncate reduces history to fit within token budget
func (h *History) Truncate(budget int) error

// TruncateToFit truncates to fit within configured maxTokens
func (h *History) TruncateToFit() error

// WouldExceedBudget checks if adding message would exceed budget
func (h *History) WouldExceedBudget(msg Message) bool
```

**Truncation Strategy:**
1. **Always preserve system message** (first message if role is system)
2. **Remove oldest non-system messages** until within budget
3. **Keep message pairs intact** (user + assistant)
4. **Preserve recent tool interactions** (last N tool messages)
5. **Stop when budget is satisfied**

**Algorithm:**

```go
func (h *History) Truncate(budget int) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    if h.TokenCount() <= budget {
        return nil // Already within budget
    }
    
    // Separate system message
    var systemMsg *Message
    messages := []Message{}
    
    if len(h.messages) > 0 && h.messages[0].Role == RoleSystem {
        systemMsg = &h.messages[0]
        messages = h.messages[1:]
    } else {
        messages = h.messages
    }
    
    // Calculate system message tokens
    systemTokens := 0
    if systemMsg != nil {
        systemTokens = systemMsg.Tokens
    }
    
    // Keep recent messages that fit in budget
    truncated := []Message{}
    if systemMsg != nil {
        truncated = append(truncated, *systemMsg)
    }
    
    tokens := systemTokens
    
    // Iterate from most recent backwards
    for i := len(messages) - 1; i >= 0; i-- {
        msgTokens := messages[i].Tokens
        if tokens + msgTokens > budget {
            break
        }
        // Prepend to maintain order
        truncated = append([]Message{messages[i]}, truncated...)
        tokens += msgTokens
    }
    
    h.messages = truncated
    return nil
}
```

**Acceptance Criteria:**
- System message always preserved
- Truncation respects token budget
- Most recent messages retained
- Token counting accurate
- Thread-safe operation
- Handles edge cases (empty history, no system message)

---

#### FR-1.3.6: Clear and Reset

**Description:** Methods for clearing history.

**Methods:**

```go
// Clear removes all messages except system message
func (h *History) Clear() error

// ClearAll removes all messages including system message
func (h *History) ClearAll() error

// Reset resets history to initial state with optional system message
func (h *History) Reset(systemMessage string) error
```

**Acceptance Criteria:**
- Clear preserves system message
- ClearAll removes everything
- Reset allows fresh start
- Thread-safe operations

---

#### FR-1.3.7: Export and Import

**Description:** Save and load history from files.

**Methods:**

```go
// Export saves history to file in JSON format
func (h *History) Export(path string) error

// ExportJSON returns history as JSON bytes
func (h *History) ExportJSON() ([]byte, error)

// Import loads history from file
func Import(path string, tokenizer Tokenizer) (*History, error)

// ImportJSON creates history from JSON bytes
func ImportJSON(data []byte, tokenizer Tokenizer) (*History, error)

// Clone creates a deep copy of history
func (h *History) Clone() *History
```

**File Format (JSON):**

```json
{
  "version": "1.0",
  "maxTokens": 8192,
  "totalTokens": 1250,
  "messageCount": 10,
  "messages": [
    {
      "id": "msg-1",
      "role": "system",
      "content": "You are a helpful coding assistant.",
      "timestamp": "2025-10-03T10:00:00Z",
      "tokens": 8
    },
    {
      "id": "msg-2",
      "role": "user",
      "content": "List files",
      "timestamp": "2025-10-03T10:01:00Z",
      "tokens": 3
    }
  ]
}
```

**Acceptance Criteria:**
- Export creates valid JSON
- Import reconstructs history correctly
- Token counts recalculated on import
- Version field for future compatibility
- Clone creates independent copy

---

### Non-Functional Requirements

#### NFR-1.3.1: Performance

- Message addition: <1ms (p99)
- Token counting: <5ms per message (p99)
- Truncation: <10ms for 100 messages (p99)
- Export: <100ms for 1000 messages (p99)
- Message retrieval: <100μs (p99)

#### NFR-1.3.2: Thread Safety

- All public methods thread-safe
- RWMutex for concurrent reads
- No data races (verified with race detector)
- Defensive copies on return

#### NFR-1.3.3: Memory

- Typical history (50 messages): <100KB
- Large history (1000 messages): <2MB
- Efficient truncation (no memory spikes)

#### NFR-1.3.4: Testability

- >90% test coverage
- All truncation strategies tested
- Concurrent access tested
- Edge cases covered

---

## Design

### Architecture

```
┌─────────────────────────────────────┐
│          History Manager            │
├─────────────────────────────────────┤
│  - messages []Message               │
│  - maxTokens int                    │
│  - tokenizer Tokenizer              │
│  - mu sync.RWMutex                  │
├─────────────────────────────────────┤
│  + AddMessage(Message)              │
│  + Messages() []Message             │
│  + Truncate(int)                    │
│  + Export(string)                   │
│  + TokenCount() int                 │
└─────────────────────────────────────┘
         │
         │ uses
         ▼
┌─────────────────────────────────────┐
│         Tokenizer Interface         │
├─────────────────────────────────────┤
│  + Count(string) int                │
│  + CountMessages([]Message) int     │
└─────────────────────────────────────┘
         ▲
         │ implements
         │
┌─────────────────────────────────────┐
│      SimpleTokenizer (stub)         │
├─────────────────────────────────────┤
│  Simple word-based estimation       │
│  (for testing/fallback)             │
└─────────────────────────────────────┘
```

### File Structure

```
internal/core/
├── history.go           # History struct and methods
├── history_test.go      # History tests
├── message.go           # Message struct and helpers
└── tokenizer.go         # Tokenizer interface and simple impl
```

### Token Counting Strategy

For now, implement a simple tokenizer based on word count:

```go
type SimpleTokenizer struct{}

func (t *SimpleTokenizer) Count(text string) int {
    // Rough estimation: ~1.3 tokens per word
    words := len(strings.Fields(text))
    return int(float64(words) * 1.3)
}

func (t *SimpleTokenizer) CountMessages(messages []Message) int {
    total := 0
    for _, msg := range messages {
        total += t.Count(msg.Content)
        // Add overhead per message (~4 tokens)
        total += 4
    }
    return total
}
```

**Note:** This will be replaced with proper tokenizer (e.g., tiktoken) in future when LLM provider integration is complete.

---

## Implementation Plan

### Task Breakdown

#### Task 1: Define types (1 hour)
- [ ] Create message.go with Message struct
- [ ] Define Role type and constants
- [ ] Define ToolCall and related types
- [ ] Add JSON tags for serialization
- [ ] Write message construction helpers

#### Task 2: Implement tokenizer interface (1 hour)
- [ ] Create tokenizer.go
- [ ] Define Tokenizer interface
- [ ] Implement SimpleTokenizer
- [ ] Write tokenizer tests
- [ ] Document token counting approach

#### Task 3: Implement History struct (2 hours)
- [ ] Create history.go
- [ ] Define History struct
- [ ] Implement NewHistory()
- [ ] Implement AddMessage() with locking
- [ ] Implement convenience add methods
- [ ] Add token tracking logic
- [ ] Write basic tests

#### Task 4: Implement retrieval methods (1.5 hours)
- [ ] Implement Messages()
- [ ] Implement MessagesForLLM()
- [ ] Implement GetMessage()
- [ ] Implement LastMessage()
- [ ] Implement Count(), TokenCount(), IsEmpty()
- [ ] Write retrieval tests

#### Task 5: Implement truncation (2 hours)
- [ ] Implement Truncate() algorithm
- [ ] Implement TruncateToFit()
- [ ] Implement WouldExceedBudget()
- [ ] Handle system message preservation
- [ ] Write comprehensive truncation tests
- [ ] Test edge cases

#### Task 6: Implement export/import (1.5 hours)
- [ ] Implement Export()
- [ ] Implement ExportJSON()
- [ ] Implement Import()
- [ ] Implement ImportJSON()
- [ ] Implement Clone()
- [ ] Write export/import tests

#### Task 7: Testing and polish (2 hours)
- [ ] Write concurrent access tests
- [ ] Write edge case tests
- [ ] Achieve >90% coverage
- [ ] Run race detector tests
- [ ] Add godoc comments
- [ ] Run linters
- [ ] Analyze with uast/herr

---

## Testing Strategy

### Unit Tests

#### Message Tests

```go
func TestMessage_Creation(t *testing.T) {
    msg := Message{
        Role:    RoleUser,
        Content: "Test message",
    }
    
    assert.Equal(t, RoleUser, msg.Role)
    assert.Equal(t, "Test message", msg.Content)
}

func TestMessage_JSONSerialization(t *testing.T) {
    msg := Message{
        ID:        "msg-1",
        Role:      RoleAssistant,
        Content:   "Response",
        Timestamp: time.Now(),
    }
    
    data, err := json.Marshal(msg)
    assert.NoError(t, err)
    
    var decoded Message
    err = json.Unmarshal(data, &decoded)
    assert.NoError(t, err)
    assert.Equal(t, msg.Role, decoded.Role)
}
```

#### History Tests

```go
func TestHistory_AddMessage(t *testing.T) {
    h := NewHistory(1000, &SimpleTokenizer{})
    
    err := h.AddUserMessage("Hello")
    assert.NoError(t, err)
    assert.Equal(t, 1, h.Count())
    
    messages := h.Messages()
    assert.Equal(t, RoleUser, messages[0].Role)
    assert.Equal(t, "Hello", messages[0].Content)
}

func TestHistory_TokenCount(t *testing.T) {
    h := NewHistory(1000, &SimpleTokenizer{})
    
    h.AddUserMessage("Hello world")  // ~2 words * 1.3 + 4 overhead
    
    count := h.TokenCount()
    assert.Greater(t, count, 0)
}
```

#### Truncation Tests

```go
func TestHistory_Truncate_PreservesSystemMessage(t *testing.T) {
    h := NewHistory(1000, &SimpleTokenizer{})
    
    h.AddSystemMessage("You are a helpful assistant")
    h.AddUserMessage("Message 1")
    h.AddAssistantMessage("Response 1")
    h.AddUserMessage("Message 2")
    h.AddAssistantMessage("Response 2")
    
    // Truncate to small budget
    err := h.Truncate(50)
    assert.NoError(t, err)
    
    messages := h.Messages()
    
    // System message should be first
    assert.Greater(t, len(messages), 0)
    assert.Equal(t, RoleSystem, messages[0].Role)
}

func TestHistory_Truncate_KeepsRecentMessages(t *testing.T) {
    h := NewHistory(1000, &SimpleTokenizer{})
    
    for i := 0; i < 20; i++ {
        h.AddUserMessage(fmt.Sprintf("Message %d", i))
        h.AddAssistantMessage(fmt.Sprintf("Response %d", i))
    }
    
    initialCount := h.Count()
    
    err := h.Truncate(200)  // Small budget
    assert.NoError(t, err)
    
    finalCount := h.Count()
    assert.Less(t, finalCount, initialCount)
    
    // Most recent messages should be present
    messages := h.Messages()
    lastMsg := messages[len(messages)-1]
    assert.Contains(t, lastMsg.Content, "19")  // Last message
}
```

#### Concurrent Access Tests

```go
func TestHistory_ConcurrentAdd(t *testing.T) {
    h := NewHistory(10000, &SimpleTokenizer{})
    
    var wg sync.WaitGroup
    numGoroutines := 100
    
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            h.AddUserMessage(fmt.Sprintf("Message %d", id))
        }(i)
    }
    
    wg.Wait()
    assert.Equal(t, numGoroutines, h.Count())
}

func TestHistory_ConcurrentReadWrite(t *testing.T) {
    h := NewHistory(10000, &SimpleTokenizer{})
    
    var wg sync.WaitGroup
    
    // Writers
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            h.AddUserMessage(fmt.Sprintf("Message %d", id))
        }(i)
    }
    
    // Readers
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _ = h.Messages()
            _ = h.TokenCount()
        }()
    }
    
    wg.Wait()
}
```

#### Export/Import Tests

```go
func TestHistory_Export(t *testing.T) {
    h := NewHistory(1000, &SimpleTokenizer{})
    h.AddSystemMessage("System message")
    h.AddUserMessage("User message")
    
    tmpFile := filepath.Join(t.TempDir(), "history.json")
    err := h.Export(tmpFile)
    assert.NoError(t, err)
    
    // File should exist
    _, err = os.Stat(tmpFile)
    assert.NoError(t, err)
}

func TestHistory_ImportExport_RoundTrip(t *testing.T) {
    h := NewHistory(1000, &SimpleTokenizer{})
    h.AddSystemMessage("System")
    h.AddUserMessage("User")
    h.AddAssistantMessage("Assistant")
    
    tmpFile := filepath.Join(t.TempDir(), "history.json")
    err := h.Export(tmpFile)
    assert.NoError(t, err)
    
    // Import
    imported, err := Import(tmpFile, &SimpleTokenizer{})
    assert.NoError(t, err)
    
    // Should match
    assert.Equal(t, h.Count(), imported.Count())
    assert.Equal(t, h.Messages()[0].Content, imported.Messages()[0].Content)
}
```

### Integration Tests

```go
func TestHistory_WithSession(t *testing.T) {
    // Test history usage within session context
    s := session.NewSession("/tmp/test")
    h := NewHistory(1000, &SimpleTokenizer{})
    
    // Add messages
    h.AddUserMessage("Create a file")
    h.AddAssistantMessage("File created")
    
    // Export for session
    data, err := h.ExportJSON()
    assert.NoError(t, err)
    
    // Could be stored in session metadata
    s.Metadata["history"] = string(data)
}
```

---

## Error Handling

### Error Types

```go
var (
    ErrEmptyHistory      = errors.New("history is empty")
    ErrMessageNotFound   = errors.New("message not found")
    ErrInvalidMessage    = errors.New("invalid message")
    ErrInvalidRole       = errors.New("invalid message role")
    ErrExportFailed      = errors.New("export failed")
    ErrImportFailed      = errors.New("import failed")
    ErrTokenizerNil      = errors.New("tokenizer cannot be nil")
)
```

### Error Handling Patterns

```go
func (h *History) AddMessage(msg Message) error {
    if msg.Role == "" {
        return fmt.Errorf("%w: role is required", ErrInvalidMessage)
    }
    
    h.mu.Lock()
    defer h.mu.Unlock()
    
    // Count tokens
    if msg.Tokens == 0 {
        msg.Tokens = h.tokenizer.Count(msg.Content)
    }
    
    h.messages = append(h.messages, msg)
    return nil
}

func (h *History) GetMessage(id string) (*Message, error) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    for i := range h.messages {
        if h.messages[i].ID == id {
            return &h.messages[i], nil
        }
    }
    
    return nil, fmt.Errorf("%w: id=%s", ErrMessageNotFound, id)
}
```

---

## Dependencies

### Internal Dependencies
- `internal/core/error.go` - Error types

### External Dependencies
- `github.com/google/uuid` - UUID generation (already in go.mod)
- Standard library: `time`, `sync`, `encoding/json`, `os`, `strings`

### Future Dependencies
- `github.com/pkoukk/tiktoken-go` - Proper tokenization (when LLM integration is complete)

---

## Examples

### Basic Usage

```go
// Create history
tokenizer := &SimpleTokenizer{}
history := NewHistory(4096, tokenizer)

// Add system message
history.AddSystemMessage("You are a helpful coding assistant.")

// Add user message
history.AddUserMessage("List files in current directory")

// Add assistant response with tool call
history.AddAssistantMessage("I'll list the files for you.")

// Add tool result
history.AddToolMessage("call-1", "file1.txt\nfile2.txt\nfile3.txt")

// Get messages for LLM
messages := history.MessagesForLLM()

// Check token count
if history.TokenCount() > 3000 {
    history.Truncate(3000)
}
```

### Truncation Example

```go
history := NewHistory(8192, tokenizer)

// Build up conversation
for i := 0; i < 50; i++ {
    history.AddUserMessage(fmt.Sprintf("Question %d", i))
    history.AddAssistantMessage(fmt.Sprintf("Answer %d", i))
}

// Truncate when approaching limit
if history.TokenCount() > 7000 {
    // Truncate to 80% of budget
    history.Truncate(6500)
}
```

### Export/Import Example

```go
// Export history
history := NewHistory(4096, tokenizer)
history.AddSystemMessage("System prompt")
history.AddUserMessage("User message")

err := history.Export("/tmp/conversation-history.json")
if err != nil {
    log.Fatal(err)
}

// Import later
imported, err := Import("/tmp/conversation-history.json", tokenizer)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Loaded %d messages\n", imported.Count())
```

---

## Acceptance Tests

### Test Case 1: Basic Message Management

**Given:** A new history is created  
**When:** Messages are added with different roles  
**Then:** All messages are stored correctly with proper roles and timestamps

### Test Case 2: Token Counting

**Given:** A history with tokenizer  
**When:** Messages are added  
**Then:** Token counts are calculated and tracked correctly

### Test Case 3: System Message Preservation

**Given:** A history with system message and many user/assistant messages  
**When:** Truncate() is called with small budget  
**Then:** System message is preserved as first message

### Test Case 4: Recent Message Retention

**Given:** A history with 100 messages  
**When:** Truncate() is called with budget for 20 messages  
**Then:** Most recent ~20 messages are retained

### Test Case 5: Concurrent Access

**Given:** A history instance  
**When:** Multiple goroutines add and read messages concurrently  
**Then:** No race conditions, all operations succeed, correct message count

### Test Case 6: Export/Import

**Given:** A history with several messages  
**When:** Export() then Import() is called  
**Then:** Imported history matches original

---

## Performance Requirements

### Benchmarks

```go
func BenchmarkHistory_AddMessage(b *testing.B) {
    h := NewHistory(10000, &SimpleTokenizer{})
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        h.AddUserMessage("Test message")
    }
}

func BenchmarkHistory_Truncate(b *testing.B) {
    h := NewHistory(10000, &SimpleTokenizer{})
    
    // Populate with 100 messages
    for i := 0; i < 100; i++ {
        h.AddUserMessage(fmt.Sprintf("Message %d", i))
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        h.Truncate(5000)
    }
}

func BenchmarkHistory_Messages(b *testing.B) {
    h := NewHistory(10000, &SimpleTokenizer{})
    
    // Populate
    for i := 0; i < 50; i++ {
        h.AddUserMessage(fmt.Sprintf("Message %d", i))
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = h.Messages()
    }
}
```

---

## Security Considerations

### Sensitive Data

- Filter sensitive environment variables from export
- Sanitize tool arguments before storing
- Optional encryption for exported history
- Provide Clear() method for sensitive conversations

### Resource Limits

- Validate maxTokens is reasonable (1-1M tokens)
- Limit maximum number of messages (prevent DoS)
- Implement maximum message size
- Truncate automatically when approaching limits

---

## Documentation Requirements

### Godoc Comments

```go
// History manages conversation message history with token-aware truncation.
//
// History provides thread-safe management of conversation messages between
// the user and AI assistant. It supports automatic token counting, smart
// truncation to fit within token budgets, and export/import capabilities.
//
// Token Management:
//   - Automatically counts tokens for each message
//   - Tracks total token usage
//   - Provides truncation to fit within budget
//   - Preserves system messages during truncation
//
// Thread Safety:
//   - All methods are thread-safe
//   - Concurrent reads and writes are supported
//   - Uses RWMutex for efficient concurrent reads
//
// Example:
//   history := NewHistory(4096, tokenizer)
//   history.AddSystemMessage("You are helpful")
//   history.AddUserMessage("Hello")
//   
//   if history.TokenCount() > 3000 {
//       history.Truncate(3000)
//   }
type History struct { ... }
```

---

## Success Criteria

- [ ] All DoD items checked off
- [ ] Test coverage >90%
- [ ] All truncation scenarios tested
- [ ] Race detector clean
- [ ] Linters passing
- [ ] Code complexity <15 (verified with uast/herr)
- [ ] Documentation complete
- [ ] Can be integrated with Session (Feature 1.1)
- [ ] Can be used by Agent (Feature 6.1)
- [ ] Can be used by Conversation (Feature 7.1)

---

## References

- [Core Module Spec](../core-module/spec.md)
- [ROADMAP](../core-module/ROADMAP.md)
- [Feature 1.1 - Session Management](./FRD-1.1.md)
- [Feature 1.2 - Turn State Machine](./FRD-1.2.md)
- [Effective Go](https://go.dev/doc/effective_go)
- [OpenAI Chat API Format](https://platform.openai.com/docs/api-reference/chat)

---

**Created:** 2025-10-03  
**Completed:** 2025-10-03  
**Author:** Development Team  
**Status:** ✅ Completed

