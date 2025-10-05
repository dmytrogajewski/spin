# Package: internal/mcp

**Path:** `internal/mcp`  
**Purpose:** Model Context Protocol (MCP) client implementation

---

## Overview

The `mcp` package implements a client for the Model Context Protocol, enabling Spin to integrate with MCP servers for extended functionality like filesystem access, database queries, and API integrations.

## Package Structure

```
internal/mcp/
├── client/           # MCP client implementation
│   ├── client.go     # Core client logic
│   ├── stdio.go      # Stdio transport
│   ├── error.go      # Error types
│   └── client_test.go # Tests
└── types/            # MCP protocol types
    ├── types.go      # Core types
    ├── request.go    # Request types
    ├── response.go   # Response types
    ├── capabilities.go # Server capabilities
    └── types_test.go # Type tests
```

**Note:** The package is organized into:
- **`client/`** - MCP client implementation with stdio transport support
- **`types/`** - Protocol message types and definitions

## Key Features

- **MCP Client**: Full MCP protocol support
- **Server Management**: Start/stop MCP servers
- **Tool Discovery**: Automatic tool discovery from servers
- **Resource Access**: Access to server-provided resources
- **Prompt Templates**: Server-provided prompts
- **Multi-Server**: Multiple concurrent MCP servers

## MCP Server Configuration

```toml
[[mcp.servers]]
name = "filesystem"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]

[[mcp.servers]]
name = "github"
command = "mcp-server-github"
args = ["--token-file", "~/.github-token"]
```

## Usage

### Basic Client Usage

```go
import (
    "github.com/dmytrogajewski/spin/internal/mcp/client"
    "github.com/dmytrogajewski/spin/internal/mcp/types"
)

// Create MCP client
mcpClient := client.New(client.Config{
    ServerCommand: "npx",
    ServerArgs:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/workspace"},
})

// Start server
if err := mcpClient.Start(); err != nil {
    log.Fatal(err)
}
defer mcpClient.Stop()

// Discover tools
tools, err := mcpClient.ListTools()
for _, tool := range tools {
    fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
}

// Call tool
result, err := mcpClient.CallTool("read_file", map[string]any{
    "path": "config.toml",
})
```

### Using Protocol Types

```go
import "github.com/dmytrogajewski/spin/internal/mcp/types"

// Create request
req := types.Request{
    Method: "tools/list",
    Params: map[string]any{},
}

// Handle response
var resp types.ListToolsResponse
// ... unmarshal response
```

## Available MCP Servers

- **@modelcontextprotocol/server-filesystem** - File operations
- **@modelcontextprotocol/server-github** - GitHub API
- **@modelcontextprotocol/server-postgres** - PostgreSQL access
- **@modelcontextprotocol/server-sqlite** - SQLite access

---

**Last Updated:** 2025-10-05  
**Status:** ✅ Production Ready
