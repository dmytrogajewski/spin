# Package: internal/protocol

**Path:** `internal/protocol`  
**Purpose:** JSON-RPC 2.0 protocol for IDE integration and UI communication

---

## Overview

The `protocol` package defines the communication protocol between Spin's core engine and user interfaces (TUI, IDE extensions, web clients). It implements a bidirectional event-based protocol over JSON-RPC 2.0, enabling real-time streaming, tool approval flows, and state synchronization.

## Key Features

- **Bidirectional Communication**: UI ↔ Core message passing
- **Event Streaming**: Real-time assistant deltas and tool calls
- **JSON-RPC 2.0**: Standard RPC protocol with batch support
- **Type Safety**: Strongly typed message structures
- **Tool Approval Flow**: Interactive approval for dangerous operations
- **Conversation Management**: Turn-based conversation lifecycle

## Package Structure

```
internal/protocol/
├── protocol.go         # Core protocol message types
├── models.go           # Data models (Message, Tool, etc.)
├── config.go           # Protocol configuration
├── conversation.go     # Conversation state management
├── adapters.go         # Adapter utilities
└── jsonrpc/            # JSON-RPC implementation
    ├── jsonrpc.go      # JSON-RPC 2.0 types
    ├── server.go       # JSON-RPC server
    └── client.go       # JSON-RPC client
```

---

## Message Types

### Inbound Messages (UI → Core)

#### UserMessage
User's text input to the agent.

```go
type UserMessage struct {
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}
```

**Example:**
```json
{
  "content": "Fix the authentication bug",
  "timestamp": "2025-10-05T10:30:00Z"
}
```

#### ToolApproval
User's approval/rejection of a proposed tool call.

```go
type ToolApproval struct {
    ToolCallID   string           `json:"tool_call_id"`
    Approved     bool             `json:"approved"`
    ModifiedArgs *json.RawMessage `json:"modified_args,omitempty"`
}
```

**Example:**
```json
{
  "tool_call_id": "call_abc123",
  "approved": true,
  "modified_args": null
}
```

#### CancelRequest
Request to cancel an in-progress turn.

```go
type CancelRequest struct {
    TurnID string `json:"turn_id"`
}
```

---

### Outbound Messages (Core → UI)

#### TurnStart
Signals the beginning of a conversation turn.

```go
type TurnStart struct {
    TurnID      string `json:"turn_id"`
    UserMessage string `json:"user_message"`
}
```

#### AssistantDelta
Incremental text from the AI model (streaming).

```go
type AssistantDelta struct {
    Delta     string  `json:"delta"`
    Reasoning *string `json:"reasoning,omitempty"`
}
```

**Example:**
```json
{
  "delta": "I'll help you fix ",
  "reasoning": null
}
```

#### ToolCallProposed
AI proposes a tool invocation.

```go
type ToolCallProposed struct {
    ToolCallID       string          `json:"tool_call_id"`
    ToolName         string          `json:"tool_name"`
    Arguments        json.RawMessage `json:"arguments"`
    RequiresApproval bool            `json:"requires_approval"`
}
```

**Example:**
```json
{
  "tool_call_id": "call_xyz789",
  "tool_name": "bash",
  "arguments": {"command": "git status"},
  "requires_approval": false
}
```

#### ToolCallExecuted
Result of a tool call execution.

```go
type ToolCallExecuted struct {
    ToolCallID string `json:"tool_call_id"`
    Success    bool   `json:"success"`
    Output     string `json:"output"`
    Error      string `json:"error,omitempty"`
}
```

#### TurnComplete
Signals the end of a conversation turn.

```go
type TurnComplete struct {
    TurnID       string `json:"turn_id"`
    FinishReason string `json:"finish_reason"` // "stop", "length", "error"
    TokensUsed   int    `json:"tokens_used"`
}
```

#### ErrorMessage
Error notification.

```go
type ErrorMessage struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}
```

---

## JSON-RPC 2.0 Implementation

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "conversation.send_message",
  "params": {
    "content": "Hello, Spin!"
  }
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "turn_id": "turn_123"
  }
}
```

### Notification Format (no response expected)

```json
{
  "jsonrpc": "2.0",
  "method": "turn.delta",
  "params": {
    "delta": "Hello! "
  }
}
```

### Error Response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32600,
    "message": "Invalid Request"
  }
}
```

---

## Conversation Flow

### Typical Turn Lifecycle

```
UI                          Core
│                            │
├─ send_message ────────────>│
│                            ├─ TurnStart
│<───────────────────────────┤
│                            ├─ AssistantDelta
│<───────────────────────────┤
│                            ├─ AssistantDelta
│<───────────────────────────┤
│                            ├─ ToolCallProposed
│<───────────────────────────┤
├─ approve_tool ────────────>│
│                            ├─ ToolCallExecuted
│<───────────────────────────┤
│                            ├─ AssistantDelta
│<───────────────────────────┤
│                            ├─ TurnComplete
│<───────────────────────────┤
```

### With Cancellation

```
UI                          Core
│                            │
├─ send_message ────────────>│
│                            ├─ TurnStart
│<───────────────────────────┤
│                            ├─ AssistantDelta
│<───────────────────────────┤
├─ cancel_turn ─────────────>│
│                            ├─ TurnComplete (cancelled)
│<───────────────────────────┤
```

---

## Usage Examples

### Server-Side (Core)

```go
import (
    "github.com/dmytrogajewski/spin/internal/protocol"
    "github.com/dmytrogajewski/spin/internal/protocol/jsonrpc"
)

// Create JSON-RPC server
server := jsonrpc.NewServer()

// Register method handlers
server.RegisterMethod("conversation.send_message", func(params json.RawMessage) (any, error) {
    var msg protocol.UserMessage
    if err := json.Unmarshal(params, &msg); err != nil {
        return nil, err
    }
    
    // Process message
    turnID := processMessage(msg)
    return map[string]string{"turn_id": turnID}, nil
})

// Send notifications to client
server.Notify("turn.delta", protocol.AssistantDelta{
    Delta: "Hello! ",
})
```

### Client-Side (UI)

```go
import (
    "github.com/dmytrogajewski/spin/internal/protocol/jsonrpc"
)

// Create JSON-RPC client
client := jsonrpc.NewClient(conn)

// Send request
var result map[string]string
err := client.Call("conversation.send_message", protocol.UserMessage{
    Content:   "Hello!",
    Timestamp: time.Now(),
}, &result)

// Handle notifications
client.OnNotification("turn.delta", func(params json.RawMessage) {
    var delta protocol.AssistantDelta
    json.Unmarshal(params, &delta)
    fmt.Print(delta.Delta)
})
```

---

## Protocol Methods

### Conversation Methods

| Method | Direction | Description |
|--------|-----------|-------------|
| `conversation.send_message` | UI → Core | Send user message |
| `conversation.cancel_turn` | UI → Core | Cancel active turn |
| `conversation.get_history` | UI → Core | Get conversation history |
| `conversation.clear` | UI → Core | Clear conversation |

### Tool Methods

| Method | Direction | Description |
|--------|-----------|-------------|
| `tool.approve` | UI → Core | Approve tool call |
| `tool.reject` | UI → Core | Reject tool call |
| `tool.modify` | UI → Core | Modify tool arguments |

### Notification Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `turn.start` | Core → UI | Turn started |
| `turn.delta` | Core → UI | Streaming content |
| `turn.complete` | Core → UI | Turn completed |
| `tool.proposed` | Core → UI | Tool call proposed |
| `tool.executed` | Core → UI | Tool call executed |
| `error` | Core → UI | Error occurred |

---

## Configuration

```go
type Config struct {
    // Protocol version
    Version string
    
    // Maximum message size (bytes)
    MaxMessageSize int
    
    // Request timeout
    Timeout time.Duration
    
    // Enable batch requests
    EnableBatch bool
    
    // Enable compression
    EnableCompression bool
}
```

**Defaults:**
```go
DefaultConfig = Config{
    Version:           "2.0",
    MaxMessageSize:    10 * 1024 * 1024, // 10MB
    Timeout:           30 * time.Second,
    EnableBatch:       true,
    EnableCompression: false,
}
```

---

## Transport Layers

The protocol supports multiple transport mechanisms:

### WebSocket
Real-time bidirectional communication for web clients.

### HTTP/HTTPS
Request-response pattern for stateless interactions.

### stdio
Standard input/output for CLI integration.

### Unix Domain Sockets
Local inter-process communication.

---

## Error Codes

Following JSON-RPC 2.0 specification:

| Code | Message | Meaning |
|------|---------|---------|
| -32700 | Parse error | Invalid JSON |
| -32600 | Invalid Request | Invalid request object |
| -32601 | Method not found | Method doesn't exist |
| -32602 | Invalid params | Invalid method parameters |
| -32603 | Internal error | Server internal error |

**Custom Codes:**
| Code | Message | Meaning |
|------|---------|---------|
| -32000 | Turn not found | Invalid turn ID |
| -32001 | Tool approval required | Awaiting approval |
| -32002 | Conversation busy | Turn in progress |

---

## Thread Safety

All protocol types are safe for concurrent use when used through the JSON-RPC server/client abstractions.

---

## Testing

```go
// Mock server for testing
func TestProtocol(t *testing.T) {
    server := jsonrpc.NewServer()
    client := jsonrpc.NewClient(mockConn)
    
    // Test method call
    server.RegisterMethod("test.echo", func(params json.RawMessage) (any, error) {
        return params, nil
    })
    
    var result string
    err := client.Call("test.echo", "hello", &result)
    assert.NoError(t, err)
    assert.Equal(t, "hello", result)
}
```

---

## Performance Considerations

- **Streaming**: Use notifications for high-frequency updates (deltas)
- **Batching**: Batch multiple requests when possible
- **Message Size**: Keep individual messages under 1MB
- **Connection Pooling**: Reuse connections for multiple requests
- **Compression**: Enable for large payloads

---

## Related Packages

- [internal/appserver](appserver.md) - JSON-RPC application server
- [internal/core](core.md) - Core business logic
- [cmd/spin-tui](../../cmd/spin-tui/) - TUI client implementation

---

**Last Updated:** 2025-10-05  
**Test Coverage:** 88.3%  
**Status:** ✅ Production Ready
