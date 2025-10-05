# Package: internal/session

**Path:** `internal/session`  
**Purpose:** Session state management and persistence

---

## Overview

The `session` package manages persistent conversation sessions, enabling users to resume conversations across restarts and maintaining conversation history.

## Key Features

- **Persistence**: Save/load conversation state
- **Session Management**: Create, list, delete sessions
- **History Tracking**: Full conversation history
- **Metadata**: Session metadata (created, updated, etc.)
- **Storage Backends**: File-based, database (future)

## Package Structure

```
internal/session/
├── session.go      # Session management
├── store.go        # Storage interface
├── filestore.go    # File-based storage
└── memory.go       # In-memory storage (testing)
```

## Usage

### Create Session

```go
import "github.com/dmytrogajewski/spin/internal/session"

// Create session manager
mgr := session.NewManager(session.Config{
    StorePath: "~/.config/spin/sessions",
})

// Create new session
sess, err := mgr.NewSession("my-project")
if err != nil {
    log.Fatal(err)
}
```

### Save State

```go
// Add messages to session
sess.AddMessage(session.Message{
    Role:    "user",
    Content: "Hello!",
})

sess.AddMessage(session.Message{
    Role:    "assistant",
    Content: "Hi! How can I help?",
})

// Save session
if err := mgr.SaveSession(sess); err != nil {
    log.Fatal(err)
}
```

### Load Session

```go
// List sessions
sessions, err := mgr.ListSessions()
for _, s := range sessions {
    fmt.Printf("Session: %s (%s)\n", s.Name, s.ID)
}

// Load specific session
sess, err := mgr.LoadSession("session-id")
if err != nil {
    log.Fatal(err)
}

// Resume conversation
history := sess.GetHistory()
```

## Session Structure

```go
type Session struct {
    ID        string
    Name      string
    CreatedAt time.Time
    UpdatedAt time.Time
    Messages  []Message
    Metadata  map[string]string
}

type Message struct {
    Role      string
    Content   string
    Timestamp time.Time
    ToolCalls []ToolCall
}
```

## Storage Format

Sessions are stored as JSON files:

```
~/.config/spin/sessions/
├── session-abc123.json
├── session-def456.json
└── session-ghi789.json
```

**Example session file:**
```json
{
  "id": "session-abc123",
  "name": "my-project",
  "created_at": "2025-10-05T10:00:00Z",
  "updated_at": "2025-10-05T10:30:00Z",
  "messages": [
    {
      "role": "user",
      "content": "Hello!",
      "timestamp": "2025-10-05T10:00:00Z"
    }
  ],
  "metadata": {
    "project": "/home/user/my-project",
    "model": "gpt-4"
  }
}
```

---

**Last Updated:** 2025-10-05  
**Status:** ✅ Production Ready
