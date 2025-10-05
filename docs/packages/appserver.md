# Package: internal/appserver

**Path:** `internal/appserver`  
**Purpose:** JSON-RPC application server for IDE integration

---

## Overview

The `appserver` package implements a JSON-RPC 2.0 server that exposes Spin's functionality to IDE extensions, web clients, and other external integrations. It handles WebSocket connections, processes protocol messages, and manages conversation state.

## Key Features

- **JSON-RPC 2.0 Server**: Standards-compliant RPC server
- **WebSocket Support**: Real-time bidirectional communication
- **File Search**: Fuzzy file search with gitignore support
- **Event Streaming**: Real-time assistant deltas and tool calls
- **Session Management**: Persistent conversation sessions
- **Concurrent Connections**: Multiple simultaneous clients
- **Error Recovery**: Graceful error handling and recovery

## Package Structure

```
internal/appserver/
├── server.go       # Main server implementation
├── handler.go      # Request/response handlers
├── processor.go    # Message processing logic
└── file_search.go  # File search functionality
```

---

## Server Architecture

```
┌─────────────┐
│   Client    │ (IDE Extension, Web UI, etc.)
└──────┬──────┘
       │ WebSocket/HTTP
┌──────▼──────┐
│  AppServer  │
├─────────────┤
│  Handler    │ ← Process JSON-RPC requests
├─────────────┤
│  Processor  │ ← Business logic
├─────────────┤
│  Protocol   │ ← Message types
└──────┬──────┘
       │
┌──────▼──────┐
│    Core     │ (Agent, LLM, Tools)
└─────────────┘
```

---

## Server

### Creating a Server

```go
import (
    "github.com/dmytrogajewski/spin/internal/appserver"
    "github.com/dmytrogajewski/spin/internal/core"
)

// Create core manager
coreMgr, err := core.NewManager(cfg, opts...)
if err != nil {
    log.Fatal(err)
}

// Create app server
server := appserver.NewServer(appserver.Config{
    Host:    "localhost",
    Port:    8080,
    Manager: coreMgr,
})

// Start server
if err := server.Start(); err != nil {
    log.Fatal(err)
}
defer server.Stop()
```

### Configuration

```go
type Config struct {
    // Server address
    Host string
    Port int
    
    // Core manager
    Manager *core.Manager
    
    // Server options
    MaxConnections int
    ReadTimeout    time.Duration
    WriteTimeout   time.Duration
    
    // TLS configuration (optional)
    TLSCert string
    TLSKey  string
}
```

**Defaults:**
```go
DefaultConfig = Config{
    Host:           "localhost",
    Port:           8080,
    MaxConnections: 100,
    ReadTimeout:    30 * time.Second,
    WriteTimeout:   30 * time.Second,
}
```

---

## Supported Methods

### Conversation Methods

#### conversation.send_message

Send a user message and start a new turn.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "conversation.send_message",
  "params": {
    "content": "Fix the authentication bug",
    "context": {
      "working_directory": "/home/user/project"
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "turn_id": "turn_abc123",
    "status": "started"
  }
}
```

#### conversation.cancel_turn

Cancel an in-progress turn.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "conversation.cancel_turn",
  "params": {
    "turn_id": "turn_abc123"
  }
}
```

#### conversation.get_history

Retrieve conversation history.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "conversation.get_history",
  "params": {
    "limit": 50,
    "offset": 0
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "messages": [
      {
        "role": "user",
        "content": "Hello",
        "timestamp": "2025-10-05T10:00:00Z"
      },
      {
        "role": "assistant",
        "content": "Hi! How can I help?",
        "timestamp": "2025-10-05T10:00:01Z"
      }
    ],
    "total": 2
  }
}
```

### Tool Methods

#### tool.approve

Approve a proposed tool call.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tool.approve",
  "params": {
    "tool_call_id": "call_xyz789",
    "modified_args": null
  }
}
```

#### tool.reject

Reject a proposed tool call.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tool.reject",
  "params": {
    "tool_call_id": "call_xyz789",
    "reason": "Permission denied"
  }
}
```

### File Search

#### files.search

Search for files with fuzzy matching.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "files.search",
  "params": {
    "query": "config",
    "limit": 10,
    "respect_gitignore": true
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "result": {
    "files": [
      {
        "path": "config/app.toml",
        "score": 0.95
      },
      {
        "path": "internal/config/config.go",
        "score": 0.87
      }
    ]
  }
}
```

---

## Event Streaming

The server sends notifications for real-time updates:

### turn.start

```json
{
  "jsonrpc": "2.0",
  "method": "turn.start",
  "params": {
    "turn_id": "turn_abc123",
    "user_message": "Fix the bug"
  }
}
```

### turn.delta

Streaming content from the assistant.

```json
{
  "jsonrpc": "2.0",
  "method": "turn.delta",
  "params": {
    "delta": "I'll help you ",
    "reasoning": null
  }
}
```

### tool.proposed

Tool call proposed by the assistant.

```json
{
  "jsonrpc": "2.0",
  "method": "tool.proposed",
  "params": {
    "tool_call_id": "call_xyz789",
    "tool_name": "bash",
    "arguments": {"command": "git status"},
    "requires_approval": false
  }
}
```

### tool.executed

Tool call completed.

```json
{
  "jsonrpc": "2.0",
  "method": "tool.executed",
  "params": {
    "tool_call_id": "call_xyz789",
    "success": true,
    "output": "On branch main...",
    "error": null
  }
}
```

### turn.complete

Turn finished.

```json
{
  "jsonrpc": "2.0",
  "method": "turn.complete",
  "params": {
    "turn_id": "turn_abc123",
    "finish_reason": "stop",
    "tokens_used": 1234
  }
}
```

---

## File Search

### Features

- **Fuzzy Matching**: Smart file name matching
- **Gitignore Support**: Respects .gitignore patterns
- **Fast Scanning**: Optimized for large codebases
- **Scoring**: Results ranked by relevance

### Implementation

```go
type FileSearcher struct {
    rootDir         string
    respectGitignore bool
    maxResults      int
}

func (fs *FileSearcher) Search(query string) ([]FileResult, error) {
    // Fuzzy search implementation
    results := fs.fuzzyMatch(query)
    return results, nil
}
```

### Usage

```go
searcher := appserver.NewFileSearcher(appserver.FileSearchConfig{
    RootDir:          "/home/user/project",
    RespectGitignore: true,
    MaxResults:       20,
})

results, err := searcher.Search("config")
for _, result := range results {
    fmt.Printf("%s (score: %.2f)\n", result.Path, result.Score)
}
```

---

## WebSocket Connection

### Client Connection

```javascript
// JavaScript example
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  // Send JSON-RPC request
  ws.send(JSON.stringify({
    jsonrpc: '2.0',
    id: 1,
    method: 'conversation.send_message',
    params: {
      content: 'Hello, Spin!'
    }
  }));
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  
  if (msg.method === 'turn.delta') {
    // Handle streaming delta
    console.log(msg.params.delta);
  } else if (msg.result) {
    // Handle response
    console.log('Result:', msg.result);
  }
};
```

### HTTP Endpoint

For non-WebSocket clients:

```bash
curl -X POST http://localhost:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "conversation.send_message",
    "params": {
      "content": "Hello!"
    }
  }'
```

---

## Error Handling

### Standard Errors

```go
var (
    ErrMethodNotFound   = errors.New("method not found")
    ErrInvalidParams    = errors.New("invalid params")
    ErrInternalError    = errors.New("internal error")
    ErrTurnNotFound     = errors.New("turn not found")
    ErrConversationBusy = errors.New("conversation busy")
)
```

### Error Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32601,
    "message": "Method not found",
    "data": {
      "method": "unknown.method"
    }
  }
}
```

---

## Concurrent Connections

The server supports multiple simultaneous connections:

```go
// Server handles concurrent clients automatically
server := appserver.NewServer(cfg)

// Each connection gets its own goroutine
// Thread-safe message processing
// Automatic cleanup on disconnect
```

**Best Practices:**
- Limit max connections based on system resources
- Set appropriate timeouts
- Monitor connection count
- Implement rate limiting if needed

---

## Security

### Authentication

```go
type Config struct {
    // ... other fields
    
    // API key for authentication
    APIKey string
    
    // TLS configuration
    TLSCert string
    TLSKey  string
}
```

### Authorization

Each method can implement authorization checks:

```go
func (h *Handler) HandleSendMessage(params json.RawMessage) (any, error) {
    // Check permissions
    if !h.isAuthorized() {
        return nil, ErrUnauthorized
    }
    
    // Process request
    return h.processor.SendMessage(params)
}
```

---

## Performance

**Benchmarks:**
- Connection handling: ~100 concurrent connections
- Message processing: <10ms average latency
- File search: <50ms for 10k files
- Memory: ~50MB per connection

**Optimization Tips:**
- Use connection pooling
- Enable WebSocket compression
- Batch requests when possible
- Cache file search results

---

## Testing

```go
func TestServer(t *testing.T) {
    // Create test server
    server := appserver.NewServer(testConfig)
    go server.Start()
    defer server.Stop()
    
    // Connect test client
    client := appserver.NewTestClient("ws://localhost:8080/ws")
    
    // Test method call
    result, err := client.Call("conversation.send_message", params)
    assert.NoError(t, err)
}
```

---

## Related Packages

- [internal/protocol](protocol.md) - Protocol message types
- [internal/core](core.md) - Core business logic
- [cmd/spin](../../cmd/spin/) - CLI integration

---

**Last Updated:** 2025-10-05  
**Test Coverage:** 89.5%  
**Status:** ✅ Production Ready
