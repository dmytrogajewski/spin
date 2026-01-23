# MCP Package

The MCP (Model Context Protocol) package provides support for connecting to MCP servers that extend Spin with additional capabilities.

## Overview

MCP servers expose tools that can be used by the AI agent. Spin supports three transport types for connecting to MCP servers:

| Transport | Description | Use Case |
|-----------|-------------|----------|
| `stdio` | Local process communication via standard I/O | Local MCP servers |
| `sse` | Server-Sent Events over HTTP | Remote MCP servers (e.g., Smithery) |
| `streamable-http` | HTTP streaming | Remote MCP servers |

## Configuration

### Stdio Transport (Local Servers)

```yaml
protocol:
  enable_mcp: true
  mcp_servers:
    - name: filesystem
      command: npx
      args:
        - -y
        - "@modelcontextprotocol/server-filesystem"
        - /workspace
      env:
        DEBUG: "true"
```

### SSE Transport (Remote Servers)

```yaml
protocol:
  enable_mcp: true
  mcp_servers:
    - name: smithery-memory
      transport: sse
      url: https://server.smithery.ai/@anthropics/claude-code-mcp-memory/sse
      headers:
        Authorization: "Bearer ${SMITHERY_API_KEY}"
```

### Streamable HTTP Transport

```yaml
protocol:
  enable_mcp: true
  mcp_servers:
    - name: remote-tools
      transport: streamable-http
      url: https://mcp.example.com/v1/tools
      headers:
        X-API-Key: "${API_KEY}"
```

### OAuth Authentication

For protected MCP servers that require OAuth:

```yaml
protocol:
  enable_mcp: true
  mcp_servers:
    - name: protected-server
      transport: sse
      url: https://protected.example.com/mcp/sse
      oauth:
        client_id: "my-client-id"
        client_secret: "${OAUTH_CLIENT_SECRET}"
        redirect_url: "http://localhost:8080/callback"
        scopes:
          - read
          - write
```

## CLI Commands

### Add MCP Server

```bash
# Add a local stdio server
spin mcp add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace

# Add a remote SSE server
spin mcp add smithery --transport sse --url https://server.smithery.ai/sse

# Add a remote server with headers
spin mcp add remote-api --transport sse --url https://api.example.com/mcp \
    --header "Authorization=Bearer token" --header "X-Custom=value"

# Add a streamable HTTP server
spin mcp add http-server --transport streamable-http --url https://mcp.example.com/v1

# Add a server with OAuth
spin mcp add protected --transport sse --url https://protected.example.com/mcp \
    --oauth-client-id "my-client" --oauth-client-secret "secret"
```

### List MCP Servers

```bash
spin mcp list

# Output:
# NAME              TRANSPORT        URL/COMMAND                              STATUS
# filesystem        stdio            npx -y @modelcontextprotocol/server...  configured
# smithery-memory   sse              https://server.smithery.ai/sse          configured
```

### Get Server Details

```bash
spin mcp get smithery-memory

# Output:
# Name: smithery-memory
# Transport: sse
# URL: https://server.smithery.ai/sse
# Headers:
#   Authorization: ***
# Source: /home/user/.spin/spin.yaml
```

### Remove MCP Server

```bash
spin mcp remove smithery-memory

# Or without confirmation
spin mcp remove smithery-memory --yes
```

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────────┐
│                     MCPServerManager                             │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ StdioClient  │  │  SSEClient   │  │ StreamableHTTPClient │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Tool Registry                         │   │
│  │  (MCP tools are registered as mcp_{server}_{tool})       │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Key Types

#### MCPServerConfig

```go
type MCPServerConfig struct {
    Name      string            // Server name (required)
    Transport TransportType     // stdio, sse, or streamable-http
    
    // Stdio transport
    Command   string            // Executable command
    Args      []string          // Command arguments
    Env       map[string]string // Environment variables
    
    // Remote transport
    URL       string            // Server URL
    Headers   map[string]string // HTTP headers
    
    // OAuth (optional)
    OAuth     *OAuthConfig      // OAuth configuration
}
```

#### TransportType

```go
type TransportType string

const (
    TransportStdio          TransportType = "stdio"
    TransportSSE            TransportType = "sse"
    TransportStreamableHTTP TransportType = "streamable-http"
)
```

### Validation Rules

1. **Name**: Always required
2. **Stdio Transport**:
   - `command` is required
   - `url` is not allowed
   - `oauth` is not allowed
3. **Remote Transport (sse, streamable-http)**:
   - `url` is required (must be valid HTTP/HTTPS URL)
   - `command` is not allowed
4. **OAuth**:
   - Only allowed for remote transports
   - `client_id` is required when OAuth is configured

### Environment Variable Expansion

Headers and URLs support environment variable expansion using `${VAR}` syntax:

```yaml
headers:
  Authorization: "Bearer ${SMITHERY_API_KEY}"
```

## Integration with Agent

MCP tools are automatically registered with the agent's tool registry when a session is created. Tools from MCP servers are prefixed with `mcp_{server_name}_` to avoid naming conflicts.

Example: A tool named `read_file` from a server named `filesystem` becomes `mcp_filesystem_read_file`.

## Error Handling

The MCP manager handles connection failures gracefully:
- If one server fails to connect, other servers still initialize
- Connection errors are logged but don't prevent session creation
- Tools from failed servers are simply not available

## Security Considerations

1. **Sensitive Headers**: The `spin mcp get` command masks sensitive headers (Authorization, secrets)
2. **Environment Variables**: Use `${VAR}` syntax for secrets instead of hardcoding
3. **OAuth**: Store OAuth secrets in environment variables
4. **HTTPS**: Remote servers should use HTTPS in production
