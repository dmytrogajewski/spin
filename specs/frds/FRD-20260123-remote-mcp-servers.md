# FRD-20260123: Remote MCP Server Support

## Summary

Add support for remote MCP (Model Context Protocol) servers via SSE and Streamable HTTP transports, enabling connection to hosted MCP services like Smithery.

## Problem Statement

Currently, Spin only supports stdio-based MCP servers, which require local process execution. This limits users to locally installed MCP servers and prevents connection to:
- Hosted MCP services (e.g., Smithery)
- Remote MCP servers running on different machines
- Cloud-based MCP server deployments

## Goals

1. Support SSE (Server-Sent Events) transport for remote MCP servers
2. Support Streamable HTTP transport for remote MCP servers
3. Support OAuth authentication for protected remote servers
4. Maintain backward compatibility with existing stdio configuration
5. Update CLI commands to manage remote MCP servers
6. Provide comprehensive E2E tests for all transport types

## Non-Goals

- WebSocket transport (not supported by mcp-go SDK)
- Custom authentication schemes beyond OAuth
- MCP server hosting (server-side implementation)

## Design

### Transport Types

The mcp-go SDK v0.42.0 supports the following client transports:

| Transport | Function | Use Case |
|-----------|----------|----------|
| Stdio | `NewStdioMCPClient` | Local process execution |
| SSE | `NewSSEMCPClient` | Remote server via Server-Sent Events |
| Streamable HTTP | `NewStreamableHttpClient` | Remote server via HTTP streaming |
| OAuth SSE | `NewOAuthSSEClient` | SSE with OAuth authentication |
| OAuth Streamable HTTP | `NewOAuthStreamableHttpClient` | HTTP streaming with OAuth |

### Configuration Schema Changes

#### Current Schema (MCPServerConfigV2)

```go
type MCPServerConfigV2 struct {
    Name    string            `yaml:"name"`
    Command string            `yaml:"command"`
    Args    []string          `yaml:"args"`
    Env     map[string]string `yaml:"env"`
}
```

#### New Schema (MCPServerConfigV2)

```go
// TransportType defines the MCP server connection transport.
type TransportType string

const (
    TransportStdio          TransportType = "stdio"
    TransportSSE            TransportType = "sse"
    TransportStreamableHTTP TransportType = "streamable-http"
)

type MCPServerConfigV2 struct {
    // Common fields
    Name      string        `yaml:"name"`
    Transport TransportType `yaml:"transport"` // Default: "stdio"
    
    // Stdio transport fields (mutually exclusive with URL)
    Command string            `yaml:"command,omitempty"`
    Args    []string          `yaml:"args,omitempty"`
    Env     map[string]string `yaml:"env,omitempty"`
    
    // Remote transport fields (mutually exclusive with Command)
    URL     string            `yaml:"url,omitempty"`
    Headers map[string]string `yaml:"headers,omitempty"`
    
    // OAuth configuration (optional, for protected servers)
    OAuth *OAuthConfigV2 `yaml:"oauth,omitempty"`
}

type OAuthConfigV2 struct {
    ClientID     string `yaml:"client_id"`
    ClientSecret string `yaml:"client_secret,omitempty"`
    RedirectURL  string `yaml:"redirect_url,omitempty"`
    Scopes       []string `yaml:"scopes,omitempty"`
}
```

### Configuration Examples

#### Stdio (Current - Backward Compatible)

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
```

#### SSE Transport

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

#### Streamable HTTP Transport

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

#### OAuth Protected Server

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

### Manager Implementation Changes

#### MCPServerManager Updates

```go
// createSDKClient creates an MCP client based on transport type.
func (m *MCPServerManager) createSDKClient(config MCPServerConfig) (*client.Client, error) {
    switch config.Transport {
    case TransportStdio, "":
        return m.createStdioClient(config)
    case TransportSSE:
        return m.createSSEClient(config)
    case TransportStreamableHTTP:
        return m.createStreamableHTTPClient(config)
    default:
        return nil, fmt.Errorf("unsupported transport: %s", config.Transport)
    }
}

func (m *MCPServerManager) createStdioClient(config MCPServerConfig) (*client.Client, error) {
    env := m.buildEnvSlice(config.Env)
    return client.NewStdioMCPClient(config.Command, env, config.Args...)
}

func (m *MCPServerManager) createSSEClient(config MCPServerConfig) (*client.Client, error) {
    opts := m.buildClientOptions(config)
    if config.OAuth != nil {
        return client.NewOAuthSSEClient(config.URL, m.toOAuthConfig(config.OAuth), opts...)
    }
    return client.NewSSEMCPClient(config.URL, opts...)
}

func (m *MCPServerManager) createStreamableHTTPClient(config MCPServerConfig) (*client.Client, error) {
    opts := m.buildStreamableHTTPOptions(config)
    if config.OAuth != nil {
        return client.NewOAuthStreamableHttpClient(config.URL, m.toOAuthConfig(config.OAuth), opts...)
    }
    return client.NewStreamableHttpClient(config.URL, opts...)
}
```

### CLI Command Updates

#### Add Command Enhancement

```bash
# Stdio (current behavior)
spin mcp add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace

# SSE transport
spin mcp add --transport sse --url https://server.smithery.ai/sse smithery-server

# Streamable HTTP
spin mcp add --transport streamable-http --url https://mcp.example.com/v1 remote-tools

# With headers
spin mcp add --transport sse --url https://api.example.com/mcp \
    --header "Authorization=Bearer token" \
    --header "X-Custom=value" \
    my-server

# With OAuth
spin mcp add --transport sse --url https://protected.example.com/mcp \
    --oauth-client-id "my-client" \
    --oauth-client-secret "secret" \
    protected-server
```

#### List Command Enhancement

```
$ spin mcp list

NAME              TRANSPORT        URL/COMMAND                              STATUS
filesystem        stdio            npx -y @modelcontextprotocol/server...  configured
smithery-memory   sse              https://server.smithery.ai/sse          configured
remote-tools      streamable-http  https://mcp.example.com/v1              configured
```

### Validation Rules

1. **Mutually Exclusive Fields**:
   - `Command` and `URL` are mutually exclusive
   - If `Transport` is `stdio` or empty, `Command` is required
   - If `Transport` is `sse` or `streamable-http`, `URL` is required

2. **URL Validation**:
   - Must be a valid URL with scheme (http/https)
   - HTTPS recommended for production

3. **OAuth Validation**:
   - If `OAuth` is provided, `ClientID` is required
   - `OAuth` is only valid for remote transports

## Test Plan

### Unit Tests

1. **Config Validation**:
   - Valid stdio config
   - Valid SSE config
   - Valid streamable-http config
   - Invalid: missing command for stdio
   - Invalid: missing URL for SSE
   - Invalid: both command and URL specified
   - Invalid: OAuth with stdio transport

2. **Client Creation**:
   - Create stdio client
   - Create SSE client
   - Create SSE client with OAuth
   - Create streamable HTTP client
   - Create streamable HTTP client with OAuth
   - Headers are properly passed

3. **Environment Variable Expansion**:
   - `${VAR}` syntax in URLs
   - `${VAR}` syntax in headers
   - Missing env var handling

### Integration Tests

1. **Mock SSE Server**:
   - Connect to mock SSE endpoint
   - List tools from mock server
   - Call tool on mock server
   - Handle server disconnect

2. **Mock HTTP Server**:
   - Connect to mock HTTP endpoint
   - List tools from mock server
   - Call tool on mock server

### E2E Tests

1. **Smithery Integration** (requires API key):
   - Connect to Smithery server
   - List available tools
   - Execute tool call

2. **Multi-Transport Session**:
   - Session with both stdio and SSE servers
   - Tools from both servers available
   - Call tools from different servers

3. **Error Handling**:
   - Invalid URL handling
   - Network timeout handling
   - Authentication failure handling

## File Changes

### Modified Files

1. `internal/config/config_v2.go`
   - Add `TransportType` type
   - Update `MCPServerConfigV2` struct
   - Add `OAuthConfigV2` struct
   - Update validation logic

2. `internal/mcp/manager.go`
   - Add transport-aware client creation
   - Add SSE client creation
   - Add streamable HTTP client creation
   - Add OAuth configuration handling
   - Add header support

3. `internal/config/mcp_manager.go`
   - Update `MCPServer` struct
   - Update validation logic

4. `cmd/spin/mcp.go`
   - Add `--transport` flag
   - Add `--url` flag
   - Add `--header` flag (repeatable)
   - Add OAuth flags
   - Update list output format

### New Files

1. `internal/mcp/transport.go`
   - Transport type constants
   - Transport-specific configuration helpers

2. `internal/mcp/manager_test.go`
   - Unit tests for transport creation

3. `tests/e2e/mcp/remote_test.go`
   - E2E tests for remote MCP servers

## Migration

### Backward Compatibility

Existing configurations remain valid:
- Empty `transport` defaults to `stdio`
- `command` and `args` continue to work for stdio

### Environment Variable Support

Headers and URLs support environment variable expansion:
```yaml
headers:
  Authorization: "Bearer ${SMITHERY_API_KEY}"
```

## Acceptance Criteria

1. [ ] SSE transport connects to remote MCP server
2. [ ] Streamable HTTP transport connects to remote MCP server
3. [ ] OAuth authentication works with both transports
4. [ ] CLI supports adding remote servers
5. [ ] CLI list shows transport type
6. [ ] Existing stdio configurations continue to work
7. [ ] E2E tests pass for all transport types
8. [ ] Documentation updated with examples
9. [ ] `make lint` passes
10. [ ] Code coverage >= 85%

## References

- [MCP Protocol Specification](https://spec.modelcontextprotocol.io/)
- [mark3labs/mcp-go SDK](https://github.com/mark3labs/mcp-go)
- [Smithery MCP Directory](https://smithery.ai/)
