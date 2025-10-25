# Model Context Protocol (MCP) API Documentation

This directory contains API documentation for the Model Context Protocol (MCP) implementation in Spin.

## Overview

The Model Context Protocol (MCP) is an open protocol that enables seamless integration between AI applications and external data sources and tools. MCP uses JSON-RPC 2.0 over stdio (standard input/output) for communication.

**Official Specification:** https://modelcontextprotocol.io/specification

**Protocol Version:** 2024-11-05

## Files

- **`openapi.yaml`** - OpenAPI 3.1 specification documenting all MCP methods, request/response types, and error codes
- **`README.md`** - This file

## Quick Start

### Viewing the OpenAPI Specification

You can view the OpenAPI specification using various tools:

#### Option 1: Swagger UI (Web-based)

```bash
# Install swagger-ui-watcher
npm install -g swagger-ui-watcher

# Serve the spec
swagger-ui-watcher api/mcp/openapi.yaml
```

Then open http://localhost:8080 in your browser.

#### Option 2: Redoc (Web-based)

```bash
# Install redoc-cli
npm install -g redoc-cli

# Serve the spec
redoc-cli serve api/mcp/openapi.yaml
```

#### Option 3: VS Code Extension

Install the **OpenAPI (Swagger) Editor** extension in VS Code and open `openapi.yaml`.

### Using the MCP Client

Spin includes an MCP client for connecting to MCP servers. Example usage:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/dmytrogajewski/spin/internal/mcp/client"
    "github.com/dmytrogajewski/spin/internal/mcp/types"
)

func main() {
    // Create MCP client configuration
    config := client.Config{
        Command: "uvx",
        Args:    []string{"mcp-server-git"},
        Env: map[string]string{
            "GIT_DIR": "/path/to/repo/.git",
        },
    }

    // Create client
    mcpClient, err := client.NewStdioClient(config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    // Initialize connection
    ctx := context.Background()
    initResp, err := mcpClient.Initialize(ctx, types.InitializeRequest{
        ProtocolVersion: "2024-11-05",
        Capabilities:    types.ClientCapabilities{},
        ClientInfo: types.Implementation{
            Name:    "spin",
            Version: "0.1.0",
        },
    })
    if err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }

    fmt.Printf("Connected to server: %s v%s\n",
        initResp.ServerInfo.Name,
        initResp.ServerInfo.Version)

    // List available tools
    toolsResp, err := mcpClient.ListTools(ctx, types.ListToolsRequest{})
    if err != nil {
        log.Fatalf("Failed to list tools: %v", err)
    }

    fmt.Printf("Available tools: %d\n", len(toolsResp.Tools))
    for _, tool := range toolsResp.Tools {
        fmt.Printf("  - %s: %s\n", tool.Name, *tool.Description)
    }

    // Call a tool
    callResp, err := mcpClient.CallTool(ctx, types.CallToolRequest{
        Name: "read_file",
        Arguments: map[string]interface{}{
            "path": "README.md",
        },
    })
    if err != nil {
        log.Fatalf("Failed to call tool: %v", err)
    }

    if callResp.IsError {
        log.Fatalf("Tool execution failed")
    }

    fmt.Printf("Tool result: %+v\n", callResp.Content)

    // Close connection
    if err := mcpClient.Close(); err != nil {
        log.Printf("Failed to close client: %v", err)
    }
}
```

## MCP Methods

### Connection Lifecycle

#### `initialize`

Establishes an MCP connection and negotiates capabilities. This must be the first request sent.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "spin",
      "version": "0.1.0"
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
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {}
    },
    "serverInfo": {
      "name": "example-server",
      "version": "1.0.0"
    }
  }
}
```

### Tools

#### `tools/list`

Lists all tools available from the MCP server.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "read_file",
        "description": "Read contents of a file",
        "inputSchema": {
          "type": "object",
          "properties": {
            "path": {
              "type": "string",
              "description": "Path to file"
            }
          },
          "required": ["path"]
        }
      }
    ]
  }
}
```

#### `tools/call`

Executes a specific tool with the provided arguments.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "read_file",
    "arguments": {
      "path": "/etc/hosts"
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "127.0.0.1 localhost\n::1 localhost"
      }
    ],
    "isError": false
  }
}
```

### Resources

#### `resources/list`

Lists all resources available from the MCP server.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "resources/list",
  "params": {}
}
```

#### `resources/read`

Reads the contents of a specific resource.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "resources/read",
  "params": {
    "uri": "file:///home/user/README.md"
  }
}
```

### Prompts

#### `prompts/list`

Lists all prompt templates available from the MCP server.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "prompts/list",
  "params": {}
}
```

#### `prompts/get`

Retrieves a specific prompt template with arguments filled in.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "prompts/get",
  "params": {
    "name": "analyze_code",
    "arguments": {
      "filename": "main.go"
    }
  }
}
```

## Error Handling

MCP uses JSON-RPC 2.0 error codes:

| Code | Message | Description |
|------|---------|-------------|
| -32700 | Parse error | Invalid JSON received |
| -32600 | Invalid Request | JSON-RPC request is invalid |
| -32601 | Method not found | Method does not exist |
| -32602 | Invalid params | Invalid method parameters |
| -32603 | Internal error | Server internal error |
| -32000 to -32099 | Server error | Server-specific errors |

**Error Response Example:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32601,
    "message": "Method not found",
    "data": {
      "method": "invalid/method"
    }
  }
}
```

## Transport: stdio

MCP servers communicate via standard input/output (stdio). Each JSON-RPC message is sent as a single line terminated by `\n`.

**Message Flow:**

```
Client                          Server
  |                               |
  |-- initialize ---------------→ |
  |← InitializeResponse ----------|
  |                               |
  |-- tools/list ----------------→|
  |← ListToolsResponse -----------|
  |                               |
  |-- tools/call ----------------→|
  |← CallToolResponse ------------|
  |                               |
  |-- (close connection) -------→ |
```

## Capabilities

### Client Capabilities

Clients can advertise their capabilities in the `initialize` request:

```json
{
  "capabilities": {
    "experimental": {},
    "sampling": {}
  }
}
```

### Server Capabilities

Servers advertise their capabilities in the `initialize` response:

```json
{
  "capabilities": {
    "tools": {
      "listChanged": true
    },
    "resources": {
      "subscribe": true,
      "listChanged": true
    },
    "prompts": {
      "listChanged": true
    },
    "logging": {}
  }
}
```

## Implementation Details

### Spin MCP Client

The Spin MCP client is implemented in `internal/mcp/client/stdio.go` and provides:

- **Automatic reconnection** on server crashes
- **Request timeout handling** (configurable)
- **Concurrent request support** via goroutines
- **Clean shutdown** with proper process cleanup
- **Thread-safe operations** for concurrent access

### Type Definitions

All MCP types are defined in `internal/mcp/types/`:

- `types.go` - Core types (Tool, Resource, Prompt, Content)
- `request.go` - All request types
- `response.go` - All response types
- `capabilities.go` - Capability types

## Testing

### Unit Tests

Run MCP client tests:

```bash
go test ./internal/mcp/client/... -v
go test ./internal/mcp/types/... -v
```

### Integration Tests

Test against a real MCP server:

```bash
# Install an example MCP server
npm install -g @modelcontextprotocol/server-everything

# Run integration tests
go test ./tests/integration/mcp_test.go -v
```

### Manual Testing

Test with `npx` MCP servers:

```bash
# Test with Git MCP server
go run ./cmd/spin debug --mcp-server "uvx:mcp-server-git"

# Test with filesystem MCP server
go run ./cmd/spin debug --mcp-server "npx:-y:@modelcontextprotocol/server-filesystem:/path/to/dir"
```

## Configuration

MCP servers are configured in `~/.spin/config.yaml`:

```yaml
mcp:
  servers:
    git:
      command: uvx
      args: [mcp-server-git]
      env:
        GIT_DIR: /home/user/project/.git
    
    filesystem:
      command: npx
      args: [-y, "@modelcontextprotocol/server-filesystem", "/home/user/documents"]
    
    brave-search:
      command: npx
      args: [-y, "@modelcontextprotocol/server-brave-search"]
      env:
        BRAVE_API_KEY: ${BRAVE_API_KEY}
```

## Further Reading

- **Official MCP Specification:** https://modelcontextprotocol.io/specification
- **MCP SDKs:** https://github.com/modelcontextprotocol
- **Example MCP Servers:** https://github.com/modelcontextprotocol/servers
- **JSON-RPC 2.0 Spec:** https://www.jsonrpc.org/specification

## Contributing

When adding new MCP features:

1. Update type definitions in `internal/mcp/types/`
2. Update client implementation in `internal/mcp/client/`
3. Update `openapi.yaml` with new endpoints/schemas
4. Add examples to this README
5. Add tests in `internal/mcp/client/*_test.go`

## License

See the main project LICENSE file.
