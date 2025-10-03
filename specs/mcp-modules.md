# Spin MCP Modules - Technical Documentation

## Overview

Model Context Protocol (MCP) integration enables Spin to:
- **As Client:** Connect to external MCP servers to access additional tools and resources
- **As Server:** Expose Spin capabilities to other MCP clients

**MCP Homepage:** https://modelcontextprotocol.io/

**Key Packages:**
1. **internal/mcp/types** - MCP protocol type definitions
2. **internal/mcp/client** - MCP client implementation
3. **internal/mcp/server** - MCP server implementation
4. **internal/mcp/transport** - Transport layer abstractions (stdio, HTTP/SSE)

---

## Model Context Protocol (MCP)

### What is MCP?

**Model Context Protocol** is an open protocol that standardizes how AI applications provide context to Large Language Models (LLMs).

**Key Concepts:**

1. **Servers** - Provide tools, resources, and prompts
2. **Clients** - Consume server capabilities
3. **Protocol** - JSON-RPC 2.0 over stdio or HTTP

**Comparison to LSP:**
- MCP is to AI what Language Server Protocol (LSP) is to editors
- Standardized interface for AI tool integration
- Single protocol, many implementations

### MCP Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    AI Application                        │
│                     (Spin Core)                          │
└─────────────┬────────────────────────┬───────────────────┘
              │                        │
    ┌─────────▼────────┐    ┌─────────▼─────────┐
    │   MCP Client     │    │   MCP Client      │
    │   (stdio)        │    │   (HTTP/SSE)      │
    └─────────┬────────┘    └─────────┬─────────┘
              │                       │
    ┌─────────▼────────┐    ┌─────────▼─────────┐
    │   MCP Server     │    │   Remote Server   │
    │   (Filesystem)   │    │   (Database)      │
    └──────────────────┘    └───────────────────┘
```

---

## Package 1: internal/mcp/types

**Path:** `internal/mcp/types/`  
**Purpose:** Go type definitions for Model Context Protocol

### Overview

Provides strongly-typed Go structs for the MCP specification, using standard `encoding/json` for serialization.

**Specification:** https://modelcontextprotocol.io/specification/2025-06-18/basic

### Project Structure

```
internal/mcp/types/
├── types.go              # Core type definitions
├── request.go            # Request types
├── response.go           # Response types
├── content.go            # Content types
├── capabilities.go       # Capability types
├── tool.go              # Tool definitions
├── resource.go          # Resource definitions
├── prompt.go            # Prompt definitions
└── types_test.go        # Type tests
```

### Core Types

#### Protocol Messages

**Requests:**
```go
package types

// InitializeRequest initializes the MCP connection
type InitializeRequest struct {
    ProtocolVersion string             `json:"protocolVersion"`
    Capabilities    ClientCapabilities `json:"capabilities"`
    ClientInfo      Implementation     `json:"clientInfo"`
}

// ListToolsRequest lists available tools
type ListToolsRequest struct {
    Cursor *string `json:"cursor,omitempty"`
}

// CallToolRequest invokes a tool
type CallToolRequest struct {
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ListResourcesRequest lists available resources
type ListResourcesRequest struct {
    Cursor *string `json:"cursor,omitempty"`
}

// ReadResourceRequest reads a resource
type ReadResourceRequest struct {
    URI string `json:"uri"`
}

// ListPromptsRequest lists available prompts
type ListPromptsRequest struct {
    Cursor *string `json:"cursor,omitempty"`
}

// GetPromptRequest gets a specific prompt
type GetPromptRequest struct {
    Name      string            `json:"name"`
    Arguments map[string]string `json:"arguments,omitempty"`
}
```

**Responses:**
```go
// InitializeResponse contains server initialization info
type InitializeResponse struct {
    ProtocolVersion string             `json:"protocolVersion"`
    Capabilities    ServerCapabilities `json:"capabilities"`
    ServerInfo      Implementation     `json:"serverInfo"`
}

// ListToolsResponse contains list of tools
type ListToolsResponse struct {
    Tools      []Tool  `json:"tools"`
    NextCursor *string `json:"nextCursor,omitempty"`
}

// CallToolResponse contains tool execution result
type CallToolResponse struct {
    Content []Content `json:"content"`
    IsError bool      `json:"isError,omitempty"`
}

// ListResourcesResponse contains list of resources
type ListResourcesResponse struct {
    Resources  []Resource `json:"resources"`
    NextCursor *string    `json:"nextCursor,omitempty"`
}

// ReadResourceResponse contains resource contents
type ReadResourceResponse struct {
    Contents []ResourceContents `json:"contents"`
}

// ListPromptsResponse contains list of prompts
type ListPromptsResponse struct {
    Prompts    []Prompt `json:"prompts"`
    NextCursor *string  `json:"nextCursor,omitempty"`
}

// GetPromptResponse contains prompt details
type GetPromptResponse struct {
    Description *string         `json:"description,omitempty"`
    Messages    []PromptMessage `json:"messages"`
}
```

#### Core Types

**Tool Definition:**
```go
// Tool represents an MCP tool
type Tool struct {
    Name        string          `json:"name"`
    Description *string         `json:"description,omitempty"`
    InputSchema json.RawMessage `json:"inputSchema"` // JSON Schema
}
```

**Resource Definition:**
```go
// Resource represents an MCP resource
type Resource struct {
    URI         string  `json:"uri"`
    Name        string  `json:"name"`
    Description *string `json:"description,omitempty"`
    MimeType    *string `json:"mimeType,omitempty"`
}

// ResourceContents contains resource data
type ResourceContents struct {
    URI      string  `json:"uri"`
    MimeType *string `json:"mimeType,omitempty"`
    Text     *string `json:"text,omitempty"`
    Blob     *string `json:"blob,omitempty"` // Base64-encoded binary
}
```

**Prompt Definition:**
```go
// Prompt represents an MCP prompt template
type Prompt struct {
    Name        string           `json:"name"`
    Description *string          `json:"description,omitempty"`
    Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument defines a prompt parameter
type PromptArgument struct {
    Name        string `json:"name"`
    Description *string `json:"description,omitempty"`
    Required    bool   `json:"required,omitempty"`
}

// PromptMessage is a message in a prompt
type PromptMessage struct {
    Role    string    `json:"role"` // "user" or "assistant"
    Content []Content `json:"content"`
}
```

**Capabilities:**
```go
// ClientCapabilities describes client features
type ClientCapabilities struct {
    Tools     *ToolsCapability     `json:"tools,omitempty"`
    Resources *ResourcesCapability `json:"resources,omitempty"`
    Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

// ServerCapabilities describes server features
type ServerCapabilities struct {
    Tools     *ToolsCapability     `json:"tools,omitempty"`
    Resources *ResourcesCapability `json:"resources,omitempty"`
    Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

// ToolsCapability indicates tool support
type ToolsCapability struct {
    ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates resource support
type ResourcesCapability struct {
    Subscribe   bool `json:"subscribe,omitempty"`
    ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability indicates prompt support
type PromptsCapability struct {
    ListChanged bool `json:"listChanged,omitempty"`
}
```

#### Content Types

```go
// Content represents different content types
type Content struct {
    Type     string  `json:"type"` // "text", "image", "resource"
    Text     *string `json:"text,omitempty"`
    Data     *string `json:"data,omitempty"`     // Base64 for images
    MimeType *string `json:"mimeType,omitempty"`
    URI      *string `json:"uri,omitempty"`
}

// Helper constructors
func TextContent(text string) Content {
    return Content{Type: "text", Text: &text}
}

func ImageContent(base64Data, mimeType string) Content {
    return Content{Type: "image", Data: &base64Data, MimeType: &mimeType}
}

func ResourceContent(uri string, mimeType *string) Content {
    return Content{Type: "resource", URI: &uri, MimeType: mimeType}
}
```

**Implementation Info:**
```go
// Implementation describes client or server info
type Implementation struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}
```

### Testing

**Test Suite:** `types_test.go`

Validates:
- JSON marshaling/unmarshaling
- Schema compliance
- Compatibility with MCP spec

```bash
go test ./internal/mcp/types/...
```

---

## Package 2: internal/mcp/client

**Path:** `internal/mcp/client/`  
**Purpose:** MCP client implementation for connecting to MCP servers

### Architecture

```
internal/mcp/client/
├── client.go           # Main client implementation
├── stdio.go            # Stdio transport
├── connection.go       # Connection management
├── jsonrpc.go          # JSON-RPC handling
└── client_test.go      # Client tests
```

### Implementation

#### `Client` Interface

**Responsibilities:**
- Spawn and manage MCP server processes
- Communicate via JSON-RPC over stdio
- Handle request/response correlation
- Manage server lifecycle

**Interface:**
```go
package client

import (
    "context"
    "spin/internal/mcp/types"
)

// Client represents an MCP client connection
type Client interface {
    // Initialize the connection
    Initialize(ctx context.Context, req types.InitializeRequest) (*types.InitializeResponse, error)
    
    // Tool operations
    ListTools(ctx context.Context) (*types.ListToolsResponse, error)
    CallTool(ctx context.Context, name string, arguments json.RawMessage) (*types.CallToolResponse, error)
    
    // Resource operations
    ListResources(ctx context.Context) (*types.ListResourcesResponse, error)
    ReadResource(ctx context.Context, uri string) (*types.ReadResourceResponse, error)
    
    // Prompt operations
    ListPrompts(ctx context.Context) (*types.ListPromptsResponse, error)
    GetPrompt(ctx context.Context, name string, arguments map[string]string) (*types.GetPromptResponse, error)
    
    // Connection management
    Close() error
}
```

#### `StdioClient` Implementation

**Usage:**
```go
package main

import (
    "context"
    "log"
    "spin/internal/mcp/client"
    "spin/internal/mcp/types"
)

func main() {
    ctx := context.Background()
    
    // Create client with command to spawn server
    cfg := client.Config{
        Command: "npx",
        Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
        Env:     map[string]string{},
    }
    
    c, err := client.NewStdioClient(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()
    
    // Initialize
    initReq := types.InitializeRequest{
        ProtocolVersion: "2024-11-05",
        Capabilities:    types.ClientCapabilities{},
        ClientInfo: types.Implementation{
            Name:    "spin",
            Version: "0.1.0",
        },
    }
    
    initResp, err := c.Initialize(ctx, initReq)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Connected to: %s %s", initResp.ServerInfo.Name, initResp.ServerInfo.Version)
    
    // List tools
    toolsResp, err := c.ListTools(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, tool := range toolsResp.Tools {
        log.Printf("Tool: %s", tool.Name)
    }
    
    // Call a tool
    args := json.RawMessage(`{"path": "README.md"}`)
    result, err := c.CallTool(ctx, "read_file", args)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, content := range result.Content {
        if content.Text != nil {
            log.Printf("Result: %s", *content.Text)
        }
    }
}
```

#### Server Lifecycle

**Startup:**
1. Spawn process with `exec.Command`
2. Connect to stdin/stdout pipes
3. Start JSON-RPC message pump
4. Send `initialize` request
5. Verify protocol version compatibility

**Communication:**
- JSON-RPC 2.0 over stdio
- One JSON object per line (JSONL)
- Bidirectional (requests and notifications)

**Shutdown:**
1. Send optional shutdown notification
2. Close stdin pipe
3. Wait for process exit with timeout
4. Clean up goroutines and resources

#### Error Handling

**Error Types:**
```go
package client

import "errors"

var (
    ErrSpawnFailed       = errors.New("failed to spawn MCP server process")
    ErrProtocolError     = errors.New("invalid JSON-RPC message")
    ErrVersionMismatch   = errors.New("incompatible protocol version")
    ErrToolFailed        = errors.New("tool execution failed")
    ErrTimeout           = errors.New("request timeout exceeded")
    ErrConnectionClosed  = errors.New("connection closed")
    ErrInvalidResponse   = errors.New("invalid response format")
)

// Error wraps MCP errors with context
type Error struct {
    Op  string // Operation
    Err error  // Underlying error
}

func (e *Error) Error() string {
    return fmt.Sprintf("mcp client: %s: %v", e.Op, e.Err)
}

func (e *Error) Unwrap() error {
    return e.Err
}
```

### Configuration in Spin

**Config File:** `~/.spin/config.yaml`

```yaml
mcp_servers:
  filesystem:
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-filesystem"
      - "/path/to/workspace"
    env: {}
    
  database:
    command: node
    args:
      - "./mcp-server-db.js"
    env:
      DATABASE_URL: "postgres://localhost/mydb"
      
  github:
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-github"
    env:
      GITHUB_TOKEN: "ghp_xxxxxxxxxxxx"
```

**CLI Management:**
```bash
# Add MCP server
spin mcp add filesystem npx -y @modelcontextprotocol/server-filesystem /path

# List configured servers
spin mcp list

# Get server details
spin mcp get filesystem

# Remove server
spin mcp remove filesystem

# Test server connection
spin mcp test filesystem
```

### Integration with Core

**In `internal/core` package:**

#### 1. Connection Manager

**File:** `internal/core/mcp_manager.go`

```go
package core

import (
    "context"
    "sync"
    "spin/internal/mcp/client"
    "spin/internal/mcp/types"
)

// MCPManager manages connections to MCP servers
type MCPManager struct {
    mu      sync.RWMutex
    clients map[string]client.Client
    tools   map[string]MCPTool // tool name -> MCP tool mapping
}

// MCPTool represents a tool from an MCP server
type MCPTool struct {
    ServerID    string
    Tool        types.Tool
    Client      client.Client
}

// NewMCPManager creates a new MCP manager
func NewMCPManager() *MCPManager {
    return &MCPManager{
        clients: make(map[string]client.Client),
        tools:   make(map[string]MCPTool),
    }
}

// ConnectServers connects to all configured MCP servers
func (m *MCPManager) ConnectServers(ctx context.Context, configs map[string]client.Config) error {
    for serverID, cfg := range configs {
        if err := m.connectServer(ctx, serverID, cfg); err != nil {
            return fmt.Errorf("connect to %s: %w", serverID, err)
        }
    }
    return nil
}

// connectServer connects to a single MCP server
func (m *MCPManager) connectServer(ctx context.Context, serverID string, cfg client.Config) error {
    c, err := client.NewStdioClient(cfg)
    if err != nil {
        return err
    }
    
    // Initialize
    initReq := types.InitializeRequest{
        ProtocolVersion: "2024-11-05",
        Capabilities:    types.ClientCapabilities{},
        ClientInfo: types.Implementation{
            Name:    "spin",
            Version: "0.1.0",
        },
    }
    
    if _, err := c.Initialize(ctx, initReq); err != nil {
        c.Close()
        return err
    }
    
    // Discover tools
    toolsResp, err := c.ListTools(ctx)
    if err != nil {
        c.Close()
        return err
    }
    
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.clients[serverID] = c
    
    // Register tools
    for _, tool := range toolsResp.Tools {
        toolKey := fmt.Sprintf("%s/%s", serverID, tool.Name)
        m.tools[toolKey] = MCPTool{
            ServerID: serverID,
            Tool:     tool,
            Client:   c,
        }
    }
    
    return nil
}

// CallTool invokes an MCP tool
func (m *MCPManager) CallTool(ctx context.Context, serverID, toolName string, arguments json.RawMessage) (*types.CallToolResponse, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    toolKey := fmt.Sprintf("%s/%s", serverID, toolName)
    mcpTool, ok := m.tools[toolKey]
    if !ok {
        return nil, fmt.Errorf("tool not found: %s", toolKey)
    }
    
    return mcpTool.Client.CallTool(ctx, toolName, arguments)
}

// ListAllTools returns all available MCP tools
func (m *MCPManager) ListAllTools() []MCPTool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    tools := make([]MCPTool, 0, len(m.tools))
    for _, tool := range m.tools {
        tools = append(tools, tool)
    }
    return tools
}

// Close closes all MCP connections
func (m *MCPManager) Close() error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    var errs []error
    for id, c := range m.clients {
        if err := c.Close(); err != nil {
            errs = append(errs, fmt.Errorf("close %s: %w", id, err))
        }
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("close errors: %v", errs)
    }
    return nil
}
```

#### 2. Tool Registration

**Integration Flow:**
```
1. Spin starts
2. Core reads mcp_servers from config
3. MCPManager connects to each server
4. Each client initializes and lists tools
5. Tools registered in core's tool registry
6. AI can now use MCP tools via standard tool interface
```

---

## Package 3: internal/mcp/server

**Path:** `internal/mcp/server/`  
**Purpose:** Expose Spin capabilities as an MCP server

### Architecture

```
internal/mcp/server/
├── server.go               # Main server implementation
├── handler.go              # Request handler
├── stdio.go                # Stdio transport
├── tools.go                # Tool implementations
├── resources.go            # Resource handlers
├── approval.go             # Approval system
└── server_test.go          # Server tests
```

### Exposed Tools

#### 1. **read_file**
Read contents of a file.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "path": { "type": "string" }
  },
  "required": ["path"]
}
```

**Implementation:**
```go
func (s *Server) handleReadFile(ctx context.Context, args json.RawMessage) (*types.CallToolResponse, error) {
    var req struct {
        Path string `json:"path"`
    }
    if err := json.Unmarshal(args, &req); err != nil {
        return nil, err
    }
    
    // Validate path is within workspace
    absPath, err := s.validatePath(req.Path)
    if err != nil {
        return nil, err
    }
    
    content, err := os.ReadFile(absPath)
    if err != nil {
        return nil, err
    }
    
    text := string(content)
    return &types.CallToolResponse{
        Content: []types.Content{
            types.TextContent(text),
        },
    }, nil
}
```

#### 2. **write_file**
Write or modify a file.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "path": { "type": "string" },
    "content": { "type": "string" }
  },
  "required": ["path", "content"]
}
```

**Approval:** Requires user approval for file changes

#### 3. **run_command**
Execute shell commands.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "command": { "type": "string" },
    "args": { 
      "type": "array",
      "items": { "type": "string" }
    }
  },
  "required": ["command"]
}
```

**Approval:** Depends on command safety classification

#### 4. **search_files**
Search for files by name or content.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "pattern": { "type": "string" },
    "type": { 
      "type": "string",
      "enum": ["name", "content"]
    }
  },
  "required": ["pattern"]
}
```

### Resources

#### workspace://
Access to workspace files.

**URIs:**
- `workspace://path/to/file.txt` - Access file in workspace
- `workspace://README.md` - Access README

### Approval System

#### Command Approval

**File:** `internal/mcp/server/approval.go`

```go
package server

import (
    "context"
    "fmt"
)

// ApprovalRequest represents a request for user approval
type ApprovalRequest struct {
    ID       string
    Type     string // "exec", "write", "delete"
    Details  map[string]interface{}
    Response chan ApprovalResponse
}

// ApprovalResponse is the user's approval decision
type ApprovalResponse struct {
    Approved bool
    Reason   string
}

// Approver handles approval requests
type Approver interface {
    RequestApproval(ctx context.Context, req ApprovalRequest) (ApprovalResponse, error)
}

// DefaultApprover implements interactive approval
type DefaultApprover struct {
    autoApprove bool
}

func NewDefaultApprover(autoApprove bool) *DefaultApprover {
    return &DefaultApprover{autoApprove: autoApprove}
}

func (a *DefaultApprover) RequestApproval(ctx context.Context, req ApprovalRequest) (ApprovalResponse, error) {
    if a.autoApprove {
        return ApprovalResponse{Approved: true}, nil
    }
    
    // Send approval request notification to client
    // Wait for response with timeout
    
    select {
    case resp := <-req.Response:
        return resp, nil
    case <-ctx.Done():
        return ApprovalResponse{Approved: false, Reason: "timeout"}, ctx.Err()
    }
}
```

**Command Safety Classification:**
```go
// CommandClassifier determines if a command is safe
type CommandClassifier struct {
    safeCommands map[string]bool
}

func NewCommandClassifier() *CommandClassifier {
    return &CommandClassifier{
        safeCommands: map[string]bool{
            "ls":   true,
            "cat":  true,
            "echo": true,
            "pwd":  true,
            // ... safe read-only commands
        },
    }
}

func (c *CommandClassifier) IsSafe(command string) bool {
    return c.safeCommands[command]
}
```

### Starting MCP Server

**Command:**
```bash
spin serve-mcp
```

**Options:**
```bash
spin serve-mcp \
    --workspace /path/to/workspace \
    --auto-approve \
    --log-level debug
```

**Flags:**
- `--workspace PATH` - Set working directory (default: current directory)
- `--auto-approve` - Automatically approve all actions (DANGEROUS, for testing)
- `--log-level LEVEL` - Set logging level (debug, info, warn, error)

### Server Implementation

**File:** `cmd/spin/serve_mcp.go`

```go
package main

import (
    "context"
    "log"
    "os"
    "spin/internal/mcp/server"
    "spin/internal/mcp/types"
)

func serveMCP(workspacePath string, autoApprove bool) error {
    ctx := context.Background()
    
    // Create server
    srv := server.New(server.Config{
        WorkspacePath: workspacePath,
        AutoApprove:   autoApprove,
        ServerInfo: types.Implementation{
            Name:    "spin",
            Version: "0.1.0",
        },
    })
    
    // Register tools
    srv.RegisterTool("read_file", readFileTool)
    srv.RegisterTool("write_file", writeFileTool)
    srv.RegisterTool("run_command", runCommandTool)
    srv.RegisterTool("search_files", searchFilesTool)
    
    // Serve over stdio
    return srv.ServeStdio(ctx, os.Stdin, os.Stdout)
}
```

### Testing with MCP Inspector

**MCP Inspector** is a development tool for testing MCP servers:

```bash
npx @modelcontextprotocol/inspector spin serve-mcp
```

**Features:**
- Interactive UI for testing tools
- View server capabilities
- Inspect requests/responses
- Manual approval handling

---

## Package 4: internal/mcp/transport

**Path:** `internal/mcp/transport/`  
**Purpose:** Transport layer abstractions (stdio, HTTP/SSE)

### Overview

Provides pluggable transport implementations for MCP protocol.

**Transports:**
1. **Stdio** - Process-based, local servers
2. **HTTP/SSE** - Network-based, remote servers

### Transport Interface

```go
package transport

import (
    "context"
    "encoding/json"
)

// Transport abstracts the underlying communication mechanism
type Transport interface {
    // Send a JSON-RPC message
    Send(ctx context.Context, msg json.RawMessage) error
    
    // Receive JSON-RPC messages
    Receive(ctx context.Context) (<-chan json.RawMessage, <-chan error)
    
    // Close the transport
    Close() error
}
```

### Stdio Transport

**Implementation:**
```go
package transport

import (
    "bufio"
    "context"
    "encoding/json"
    "io"
    "os/exec"
)

// StdioTransport implements Transport over stdin/stdout
type StdioTransport struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.ReadCloser
    cancel context.CancelFunc
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(command string, args []string) (*StdioTransport, error) {
    cmd := exec.Command(command, args...)
    
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, err
    }
    
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }
    
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    
    return &StdioTransport{
        cmd:    cmd,
        stdin:  stdin,
        stdout: stdout,
    }, nil
}

func (t *StdioTransport) Send(ctx context.Context, msg json.RawMessage) error {
    _, err := t.stdin.Write(append(msg, '\n'))
    return err
}

func (t *StdioTransport) Receive(ctx context.Context) (<-chan json.RawMessage, <-chan error) {
    msgCh := make(chan json.RawMessage)
    errCh := make(chan error, 1)
    
    go func() {
        defer close(msgCh)
        defer close(errCh)
        
        scanner := bufio.NewScanner(t.stdout)
        for scanner.Scan() {
            select {
            case <-ctx.Done():
                errCh <- ctx.Err()
                return
            case msgCh <- scanner.Bytes():
            }
        }
        
        if err := scanner.Err(); err != nil {
            errCh <- err
        }
    }()
    
    return msgCh, errCh
}

func (t *StdioTransport) Close() error {
    t.stdin.Close()
    return t.cmd.Wait()
}
```

### HTTP/SSE Transport

**Implementation:**
```go
package transport

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

// HTTPTransport implements Transport over HTTP + Server-Sent Events
type HTTPTransport struct {
    baseURL    string
    httpClient *http.Client
    eventsURL  string
    cancel     context.CancelFunc
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(baseURL string) *HTTPTransport {
    return &HTTPTransport{
        baseURL:    baseURL,
        httpClient: &http.Client{},
        eventsURL:  baseURL + "/events",
    }
}

func (t *HTTPTransport) Send(ctx context.Context, msg json.RawMessage) error {
    req, err := http.NewRequestWithContext(ctx, "POST", t.baseURL+"/message", bytes.NewReader(msg))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := t.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, body)
    }
    
    return nil
}

func (t *HTTPTransport) Receive(ctx context.Context) (<-chan json.RawMessage, <-chan error) {
    msgCh := make(chan json.RawMessage)
    errCh := make(chan error, 1)
    
    go func() {
        defer close(msgCh)
        defer close(errCh)
        
        req, err := http.NewRequestWithContext(ctx, "GET", t.eventsURL, nil)
        if err != nil {
            errCh <- err
            return
        }
        
        req.Header.Set("Accept", "text/event-stream")
        
        resp, err := t.httpClient.Do(req)
        if err != nil {
            errCh <- err
            return
        }
        defer resp.Body.Close()
        
        // Parse SSE stream
        // Implementation details...
    }()
    
    return msgCh, errCh
}

func (t *HTTPTransport) Close() error {
    if t.cancel != nil {
        t.cancel()
    }
    return nil
}
```

---

## MCP in Spin: Complete Flow

### 1. Configuration

User configures MCP servers in `~/.spin/config.yaml`:

```yaml
mcp_servers:
  github:
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-github"
    env:
      GITHUB_TOKEN: "ghp_xxxxxxxxxxxx"
```

### 2. Startup

**On Spin Start:**
1. Core reads `mcp_servers` from config
2. Creates `MCPManager`
3. For each server:
   - Spawns MCP client
   - Client starts server process
   - Client sends `initialize`
   - Client sends `list_tools`

### 3. Tool Registration

**Tool Discovery:**
```go
// For each MCP server
for serverID, cfg := range config.MCPServers {
    client, err := client.NewStdioClient(cfg)
    if err != nil {
        return err
    }
    
    tools, err := client.ListTools(ctx)
    if err != nil {
        return err
    }
    
    // Register as Spin tools
    for _, tool := range tools.Tools {
        toolRegistry.Register(Tool{
            Name:        fmt.Sprintf("%s/%s", serverID, tool.Name),
            Description: tool.Description,
            Schema:      tool.InputSchema,
            Execute: func(ctx context.Context, args json.RawMessage) (string, error) {
                resp, err := client.CallTool(ctx, tool.Name, args)
                if err != nil {
                    return "", err
                }
                return formatToolResponse(resp), nil
            },
        })
    }
}
```

### 4. Tool Invocation

**When AI calls MCP tool:**
```
1. LLM proposes tool call: "github/create_issue"
2. Core looks up tool in registry
3. Registry identifies it's an MCP tool from "github" server
4. Core routes to MCP manager
5. Manager finds "github" server's client
6. Client sends CallToolRequest via JSON-RPC
7. Server executes tool (creates GitHub issue)
8. Client receives CallToolResponse
9. Manager translates to standard format
10. Core returns result to LLM
```

### 5. Resource Access

**When AI needs resource:**
```go
// AI requests: "Show me workspace://README.md"
resource, err := mcpManager.ReadResource(ctx, "workspace", "workspace://README.md")
if err != nil {
    return err
}

// Returns file contents as ResourceContents
for _, content := range resource.Contents {
    if content.Text != nil {
        fmt.Println(*content.Text)
    }
}
```

### 6. Shutdown

**On Spin Exit:**
1. Manager sends shutdown to all servers
2. Clients close server processes
3. Clean up goroutines and resources

---

## MCP Configuration Examples

### Filesystem Server

```yaml
mcp_servers:
  filesystem:
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-filesystem"
      - "/home/user/documents"
```

**Provides:**
- File read/write tools
- Directory listing
- File search

### Database Server

```yaml
mcp_servers:
  database:
    command: node
    args:
      - "./mcp-db-server.js"
    env:
      DATABASE_URL: "postgresql://localhost/mydb"
```

**Provides:**
- SQL query execution
- Schema inspection
- Data migration tools

### GitHub Server

```yaml
mcp_servers:
  github:
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-github"
    env:
      GITHUB_TOKEN: "ghp_xxxxxxxxxxxx"
```

**Provides:**
- Repository operations
- Issue management
- Pull request operations

### Custom Server

```yaml
mcp_servers:
  custom:
    command: /usr/local/bin/my-mcp-server
    args:
      - "--config"
      - "/etc/mcp/config.json"
```

---

## Security Considerations

### MCP Client Security

**Risks:**
- Untrusted MCP servers can provide malicious tools
- Tool calls can have side effects
- Resources may contain sensitive data

**Mitigations:**
1. **Tool Approval:** Dangerous tools require user approval
2. **Sandboxing:** Execute in restricted environment (consider seccomp, namespaces)
3. **Validation:** Validate tool responses against schema
4. **Auditing:** Log all tool calls to audit trail
5. **Resource Limits:** Limit tool execution time, memory, output size

### MCP Server Security

**Risks:**
- Clients can request dangerous operations
- Resource URIs could access sensitive files
- Tool abuse (e.g., DOS via repeated calls)

**Mitigations:**
1. **Approval System:** Require approval for file/exec operations
2. **Path Restrictions:** Limit resource URIs to workspace using `filepath.Clean` and prefix checking
3. **Rate Limiting:** Prevent abuse with token bucket or similar
4. **Authentication:** Verify client identity (if needed for remote servers)

**Path Validation Example:**
```go
func (s *Server) validatePath(requestedPath string) (string, error) {
    // Clean and make absolute
    absPath := filepath.Join(s.workspaceRoot, filepath.Clean(requestedPath))
    
    // Ensure it's within workspace (prevent directory traversal)
    if !strings.HasPrefix(absPath, s.workspaceRoot) {
        return "", fmt.Errorf("path outside workspace: %s", requestedPath)
    }
    
    return absPath, nil
}
```

---

## Debugging

### Enable MCP Logging

```bash
SPIN_LOG_LEVEL=debug spin
```

**Structured Logging:**
```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

logger.Debug("mcp client: sending request",
    "server", serverID,
    "method", "tools/call",
    "tool", toolName)
```

### Test MCP Server

```bash
npx @modelcontextprotocol/inspector spin serve-mcp
```

### Test MCP Client

```bash
spin mcp test filesystem --tool read_file --args '{"path":"README.md"}'
```

---

## Performance

**Latency:**
- Stdio transport: ~5-10ms per tool call (local process)
- HTTP/SSE transport: ~50-100ms per tool call (network overhead)

**Throughput:**
- Stdio: Thousands of calls/second (limited by process spawning)
- HTTP/SSE: Hundreds of calls/second (limited by network)

**Optimization Strategies:**
1. **Connection Pooling:** Reuse MCP client connections
2. **Response Caching:** Cache immutable resources (with TTL)
3. **Parallel Requests:** Use goroutines for concurrent tool calls
4. **Batching:** Group requests when server supports it

**Concurrency:**
```go
// Execute multiple MCP tools in parallel
var wg sync.WaitGroup
results := make([]types.CallToolResponse, len(toolCalls))

for i, tc := range toolCalls {
    wg.Add(1)
    go func(idx int, call ToolCall) {
        defer wg.Done()
        resp, err := mcpManager.CallTool(ctx, call.Server, call.Name, call.Args)
        if err != nil {
            // Handle error
            return
        }
        results[idx] = *resp
    }(i, tc)
}

wg.Wait()
```

---

## Testing

### Unit Tests

**Test MCP types:**
```bash
go test ./internal/mcp/types/...
```

**Test MCP client:**
```bash
go test ./internal/mcp/client/...
```

**Test MCP server:**
```bash
go test ./internal/mcp/server/...
```

### Integration Tests

**Test Suite:** `tests/integration/mcp_test.go`

```go
package integration

import (
    "testing"
    "spin/internal/mcp/client"
    "spin/internal/mcp/server"
)

func TestMCPIntegration(t *testing.T) {
    // Start MCP server
    srv := server.New(server.Config{
        WorkspacePath: t.TempDir(),
        AutoApprove:   true,
    })
    
    go srv.ServeStdio(ctx, serverReader, serverWriter)
    
    // Create client
    cfg := client.Config{
        Command: "spin",
        Args:    []string{"serve-mcp", "--auto-approve"},
    }
    
    c, err := client.NewStdioClient(cfg)
    if err != nil {
        t.Fatal(err)
    }
    defer c.Close()
    
    // Test tool call
    resp, err := c.CallTool(ctx, "read_file", json.RawMessage(`{"path":"test.txt"}`))
    if err != nil {
        t.Fatal(err)
    }
    
    // Verify response
    // ...
}
```

---

## Related Packages

- **internal/core** - MCP integration and tool routing
- **internal/tools** - Tool registry and execution
- **pkg/config** - Configuration management
- **cmd/spin** - CLI commands

---

## Dependencies

**Standard Library:**
- `encoding/json` - JSON marshaling
- `os/exec` - Process execution
- `net/http` - HTTP transport
- `bufio` - Stream processing
- `context` - Context management

**External (Optional):**
- `gopkg.in/yaml.v3` - YAML config parsing
- `github.com/google/uuid` - Request ID generation

**No Vendor Lock-in:**
- All implementations use standard interfaces
- Compatible with any MCP server (Node.js, Python, Rust, etc.)
- Works with Ollama, LM Studio, and other open-source LLM providers

---

## Future Enhancements

- [ ] MCP server discovery (registry/marketplace)
- [ ] Automatic server installation and management
- [ ] Enhanced resource streaming (large files)
- [ ] Bidirectional resource updates (watch mode)
- [ ] Multi-hop MCP chains (server-to-server)
- [ ] WebSocket transport for real-time communication
- [ ] gRPC transport for high-performance scenarios

---

## Conclusion

MCP integration makes Spin extensible and interoperable:
- **As Client:** Access external tools and data sources from any MCP server
- **As Server:** Expose Spin capabilities to other AI systems and tools
- **Standard Protocol:** Works with entire MCP ecosystem
- **Open Source:** No vendor lock-in, compatible with Ollama, LM Studio, etc.

The combination of Go's simplicity, performance, and the flexible MCP protocol enables safe, efficient integration with diverse tools and services while maintaining clean architecture principles.

