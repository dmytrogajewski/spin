# Package: internal/mcp

**Path:** `internal/mcp`  
**Purpose:** Model Context Protocol (MCP) manager using mark3labs/mcp-go SDK

---

## Overview

The `mcp` package provides an MCP manager that integrates with MCP servers using the official `github.com/mark3labs/mcp-go` SDK. It enables Spin to connect to MCP servers for extended functionality like filesystem access, database queries, and API integrations.

**Architecture**: Thin manager layer around the mcp-go SDK for server lifecycle management and tool registration.

## Package Structure

```
internal/mcp/
├── manager.go          # MCPManager - server lifecycle and tool registration
└── manager_test.go     # Tests
```

## SDK Integration

**Uses**: `github.com/mark3labs/mcp-go` v0.42.0
- **Client**: `github.com/mark3labs/mcp-go/client` - stdio transport
- **Types**: `github.com/mark3labs/mcp-go/mcp` - protocol types

**Benefits**:
- Battle-tested MCP protocol implementation
- Active maintenance and community support
- Reduced codebase (~400 lines of custom code eliminated)
- Type-safe API with `json.RawMessage` (no `interface{}`)

## Key Features

- **Multi-Server Management**: Connect to multiple MCP servers concurrently
- **Automatic Tool Discovery**: Registers server tools with Spin's tool registry
- **Type-Safe Arguments**: Tool calls use `json.RawMessage` instead of `map[string]interface{}`
- **SDK-Based**: Leverages official mcp-go SDK for protocol handling
- **Stdio Transport**: Subprocess-based communication

## MCP Server Configuration

```toml
[[mcp.servers]]
name = "filesystem"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]
env = { HOME = "/home/user" }

[[mcp.servers]]
name = "github"
command = "mcp-server-github"
args = ["--token-file", "~/.github-token"]
```

## Usage

### MCPManager

```go
import (
    "github.com/dmytrogajewski/spin/internal/mcp"
    "log/slog"
)

// Create manager
config := &mcp.Config{
    EnableMCP: true,
    MCPServers: []mcp.MCPServerConfig{
        {
            Name:    "filesystem",
            Command: "npx",
            Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
            Env:     map[string]string{"HOME": "/home/user"},
        },
    },
}

logger := slog.Default()
manager := mcp.NewMCPManager(config, logger)

// Initialize connections
if err := manager.Initialize(ctx); err != nil {
    log.Fatal(err)
}
defer manager.Close()

// Get tools for registration
tools := manager.GetTools()
for _, tool := range tools {
    fmt.Printf("Tool: %s\n", tool.Name())
}

// Call tool directly (usually done via tool registry)
args := json.RawMessage(`{"path": "/test.txt"}`)
result, err := manager.CallTool(ctx, "mcp_filesystem_read_file", args)
if result.Success {
    fmt.Println(result.Output)
}
```

### Using SDK Directly (Advanced)

```go
import (
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/mcp"
)

// Create SDK client
mcpClient, err := client.NewStdioMCPClient(
    "npx",
    []string{"HOME=/home/user"},
    "-y", "@modelcontextprotocol/server-filesystem", "/workspace",
)
if err != nil {
    log.Fatal(err)
}
defer mcpClient.Close()

// Initialize
initReq := mcp.InitializeRequest{
    Params: mcp.InitializeParams{
        ProtocolVersion: "2024-11-05",
        Capabilities:    mcp.ClientCapabilities{},
        ClientInfo: mcp.Implementation{
            Name:    "spin",
            Version: "0.1.0",
        },
    },
}
initResp, err := mcpClient.Initialize(ctx, initReq)

// List tools
listReq := mcp.ListToolsRequest{}
toolsResp, err := mcpClient.ListTools(ctx, listReq)
for _, tool := range toolsResp.Tools {
    fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
}

// Call tool
callReq := mcp.CallToolRequest{
    Params: mcp.CallToolParams{
        Name: "read_file",
        Arguments: map[string]any{
            "path": "/test.txt",
        },
    },
}
result, err := mcpClient.CallTool(ctx, callReq)
```

## Available MCP Servers

### Official Servers

- **@modelcontextprotocol/server-filesystem** - File operations
- **@modelcontextprotocol/server-github** - GitHub API
- **@modelcontextprotocol/server-postgres** - PostgreSQL access
- **@modelcontextprotocol/server-sqlite** - SQLite access
- **@modelcontextprotocol/server-brave-search** - Web search
- **@modelcontextprotocol/server-puppeteer** - Browser automation

### Installation

```bash
# Filesystem server
npx -y @modelcontextprotocol/server-filesystem /path/to/workspace

# GitHub server
npm install -g @modelcontextprotocol/server-github
```

## Implementation Details

### Manager Responsibilities

- **Connection Lifecycle**: Start/stop MCP clients
- **Server Registration**: Track connected servers
- **Tool Registration**: Export MCP tools to Spin's tool registry
- **Tool Invocation**: Route tool calls to appropriate server

### Type-Safe Tool Arguments

**Before (Phase 2.4)**: `map[string]interface{}`  
**After (Phase 2.4)**: `json.RawMessage` ✅

```go
// Old (interface{})
func CallTool(ctx context.Context, name string, args map[string]interface{})

// New (type-safe)
func CallTool(ctx context.Context, name string, args json.RawMessage)
```

### Error Handling

- SDK errors are wrapped and returned as `tools.ToolResult{Success: false}`
- Tool execution errors (IsError=true) are reported to LLM for self-correction
- Connection errors are logged and other servers continue to operate

## Migration from Custom Client (Phase 2.4)

**Changes**:
- ✅ Migrated to `github.com/mark3labs/mcp-go` SDK
- ✅ Eliminated `interface{}` usage (replaced with `json.RawMessage`)
- ✅ Deleted custom client implementation (~800 lines)
- ✅ Deleted custom types (~400 lines)
- ✅ Reduced manager code from ~600 lines to ~400 lines

**Breaking Changes**:
- `CallTool` signature changed from `map[string]interface{}` to `json.RawMessage`
- Internal types replaced with SDK types (not exposed in public API)

---

**Last Updated:** 2025-10-26  
**Test Coverage:** 8.1% (manager only - SDK provides protocol implementation)  
**Status:** ✅ Production Ready

**SDK Version:** github.com/mark3labs/mcp-go v0.42.0
