# FRD-UI-4.1: MCP Management Commands

**Feature:** MCP (Model Context Protocol) server management CLI commands
**Module:** `cmd/spin/mcp.go`
**Roadmap:** Phase 4.1
**Priority:** P2 - Advanced Features
**Status:** Draft

---

## 1. Overview

### 1.1 Purpose

Provide command-line tools for managing MCP server configurations, allowing users to add, remove, list, and inspect MCP servers without manually editing configuration files.

### 1.2 Goals

- ✅ CRUD operations for MCP server configurations
- ✅ User-friendly CLI for non-technical users
- ✅ Safe configuration file manipulation (YAML/TOML/JSON)
- ✅ Validation of server configurations
- ✅ Integration with existing `internal/config` loader
- ✅ Support for all config formats (YAML, TOML, JSON)

### 1.3 Non-Goals

- ❌ Runtime MCP server management (start/stop processes)
- ❌ MCP server discovery or auto-configuration
- ❌ MCP protocol implementation (already in `internal/mcp`)
- ❌ MCP server health checking

---

## 2. Requirements

### 2.1 Functional Requirements

#### FR-1: Add MCP Server
**Priority:** P0 (Critical)

Users can add a new MCP server configuration:
```bash
spin mcp add <name> <command> [args...]

# Examples:
spin mcp add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace
spin mcp add github mcp-server-github --token-file ~/.github-token
spin mcp add postgres mcp-server-postgres --connection-string "postgresql://localhost/mydb"
```

**Behavior:**
- Validates server name is unique
- Validates command is executable (checks PATH)
- Stores configuration in active config file
- Creates default config file if none exists (~/.spin/spin.yaml)
- Supports environment variables via --env flag

#### FR-2: List MCP Servers
**Priority:** P0 (Critical)

Users can list all configured MCP servers:
```bash
spin mcp list

# Output:
# NAME          COMMAND                                       STATUS
# filesystem    npx -y @modelcontextprotocol/server-...      configured
# github        mcp-server-github --token-file ...           configured
# postgres      mcp-server-postgres --connection-string...   configured
```

**Behavior:**
- Shows all servers from active config
- Displays name, command (truncated if long), and configuration status
- Supports --format json for programmatic use
- Shows config file location with --verbose

#### FR-3: Get MCP Server Details
**Priority:** P1 (High)

Users can inspect a specific MCP server configuration:
```bash
spin mcp get <name>

# Example output:
# Name: filesystem
# Command: npx
# Args:
#   - -y
#   - @modelcontextprotocol/server-filesystem
#   - /workspace
# Environment: (none)
# Source: /home/user/.spin/spin.yaml
```

**Behavior:**
- Shows full server configuration
- Displays source config file
- Supports --format json/yaml for export
- Returns error if server not found

#### FR-4: Remove MCP Server
**Priority:** P0 (Critical)

Users can remove an MCP server configuration:
```bash
spin mcp remove <name>

# Example:
spin mcp remove filesystem
# Output: Removed MCP server 'filesystem' from /home/user/.spin/spin.yaml
```

**Behavior:**
- Removes server from config file
- Prompts for confirmation unless --yes flag used
- Preserves comments and formatting where possible
- Returns error if server not found

#### FR-5: Update MCP Server
**Priority:** P2 (Medium)

Users can update an existing MCP server:
```bash
spin mcp update <name> [--command <cmd>] [--args <args>] [--env <key=value>]

# Example:
spin mcp update filesystem --args "/new/workspace/path"
```

**Behavior:**
- Updates specified fields only (partial update)
- Validates new configuration
- Preserves other fields
- Returns error if server not found

### 2.2 Configuration Format

MCP servers are stored in the config file under `mcp.servers`:

**YAML:**
```yaml
mcp:
  servers:
    - name: filesystem
      command: npx
      args:
        - -y
        - @modelcontextprotocol/server-filesystem
        - /workspace
      env:
        NODE_ENV: production

    - name: github
      command: mcp-server-github
      args:
        - --token-file
        - ~/.github-token
```

**TOML:**
```toml
[[mcp.servers]]
name = "filesystem"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]

[mcp.servers.env]
NODE_ENV = "production"

[[mcp.servers]]
name = "github"
command = "mcp-server-github"
args = ["--token-file", "~/.github-token"]
```

**JSON:**
```json
{
  "mcp": {
    "servers": [
      {
        "name": "filesystem",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"],
        "env": {
          "NODE_ENV": "production"
        }
      }
    ]
  }
}
```

### 2.3 Non-Functional Requirements

#### NFR-1: Performance
- Command execution <100ms for list/get
- File operations atomic (no partial writes)
- Config parsing cached where possible

#### NFR-2: Reliability
- Safe config file updates (write to temp, then rename)
- Backup config before modification
- Validate config after modification
- Rollback on error

#### NFR-3: Usability
- Clear error messages
- Helpful examples in --help
- Support for common use cases (filesystem, github, postgres)
- Interactive prompts where appropriate

#### NFR-4: Compatibility
- Works with all config formats (YAML, TOML, JSON)
- Preserves formatting and comments where possible
- Compatible with existing config structure

---

## 3. Design

### 3.1 Architecture

```
cmd/spin/mcp.go (Cobra command)
       ↓
internal/config/mcp_manager.go (MCP config management)
       ↓
internal/config/loader.go (Viper-based config I/O)
       ↓
~/.spin/spin.yaml (User config file)
```

### 3.2 Components

#### Component 1: Cobra Commands (`cmd/spin/mcp.go`)

```go
// MCP command group
var mcpCmd = &cobra.Command{
    Use:   "mcp",
    Short: "Manage MCP (Model Context Protocol) servers",
    Long:  "Add, remove, list, and configure MCP servers.",
}

// Subcommands
var mcpAddCmd = &cobra.Command{...}
var mcpListCmd = &cobra.Command{...}
var mcpGetCmd = &cobra.Command{...}
var mcpRemoveCmd = &cobra.Command{...}
var mcpUpdateCmd = &cobra.Command{...}
```

#### Component 2: MCP Manager (`internal/config/mcp_manager.go`)

```go
// MCPServer represents an MCP server configuration
type MCPServer struct {
    Name    string            `yaml:"name" toml:"name" json:"name"`
    Command string            `yaml:"command" toml:"command" json:"command"`
    Args    []string          `yaml:"args,omitempty" toml:"args,omitempty" json:"args,omitempty"`
    Env     map[string]string `yaml:"env,omitempty" toml:"env,omitempty" json:"env,omitempty"`
}

// MCPManager manages MCP server configurations
type MCPManager struct {
    loader *Loader
}

func NewMCPManager(loader *Loader) *MCPManager
func (m *MCPManager) List() ([]MCPServer, error)
func (m *MCPManager) Get(name string) (*MCPServer, error)
func (m *MCPManager) Add(server MCPServer) error
func (m *MCPManager) Remove(name string) error
func (m *MCPManager) Update(name string, updates MCPServer) error
```

### 3.3 Data Flow

#### Add Server Flow
```
User: spin mcp add filesystem npx -y @mcp/server-filesystem /workspace
  ↓
Cobra: Parse command and args
  ↓
MCPManager.Add(MCPServer{name, command, args})
  ↓
Validate: name unique, command exists
  ↓
Loader: Load current config
  ↓
Append new server to mcp.servers array
  ↓
Loader: Write config (atomic)
  ↓
Success: "Added MCP server 'filesystem'"
```

#### Remove Server Flow
```
User: spin mcp remove filesystem
  ↓
MCPManager.Remove("filesystem")
  ↓
Prompt: "Remove MCP server 'filesystem'? (y/N)"
  ↓
Loader: Load current config
  ↓
Find and remove server from mcp.servers
  ↓
Loader: Write config (atomic)
  ↓
Success: "Removed MCP server 'filesystem'"
```

### 3.4 Error Handling

| Error Condition | User Message | Exit Code |
|----------------|--------------|-----------|
| Server already exists | `Error: MCP server '{name}' already exists. Use 'update' to modify.` | 1 |
| Server not found | `Error: MCP server '{name}' not found.` | 1 |
| Command not in PATH | `Warning: Command '{cmd}' not found in PATH. Server may not work.` | 0 (warning) |
| Config file read error | `Error: Failed to read config: {error}` | 1 |
| Config file write error | `Error: Failed to write config: {error}` | 1 |
| Invalid config format | `Error: Invalid configuration: {error}` | 1 |

---

## 4. Implementation Plan

### 4.1 Test-Driven Development

#### Phase 1: Write Tests
1. Create `internal/config/mcp_manager_test.go`
2. Test MCPServer struct marshaling (YAML, TOML, JSON)
3. Test MCPManager.List() - empty, single, multiple servers
4. Test MCPManager.Get() - found, not found
5. Test MCPManager.Add() - success, duplicate, validation
6. Test MCPManager.Remove() - success, not found
7. Test MCPManager.Update() - partial updates, not found

#### Phase 2: Implement MCPManager
1. Create `internal/config/mcp_manager.go`
2. Implement MCPServer struct with tags
3. Implement MCPManager methods
4. Add validation helpers
5. Add atomic file write helpers

#### Phase 3: Implement Cobra Commands
1. Create `cmd/spin/mcp.go`
2. Implement mcpCmd (parent command)
3. Implement mcpAddCmd
4. Implement mcpListCmd
5. Implement mcpGetCmd
6. Implement mcpRemoveCmd
7. Implement mcpUpdateCmd (optional)

#### Phase 4: Integration Tests
1. Create `cmd/spin/mcp_test.go`
2. Test end-to-end with temp config files
3. Test all three config formats (YAML, TOML, JSON)
4. Test error scenarios

### 4.2 Files to Create

```
specs/frds/FRD-UI-4.1.md                    (this file)
internal/config/mcp_manager.go              (new)
internal/config/mcp_manager_test.go         (new)
cmd/spin/mcp.go                             (new)
cmd/spin/mcp_test.go                        (new, optional)
```

### 4.3 Files to Modify

```
cmd/spin/root.go                            (register mcp command)
```

---

## 5. Testing Strategy

### 5.1 Unit Tests

**Package:** `internal/config`

```go
func TestMCPServer_Marshal(t *testing.T) {
    // Test YAML/TOML/JSON marshaling
}

func TestMCPManager_List(t *testing.T) {
    // Test listing servers
}

func TestMCPManager_Get(t *testing.T) {
    // Test getting specific server
}

func TestMCPManager_Add(t *testing.T) {
    // Test adding new server
    // Test duplicate name error
    // Test validation
}

func TestMCPManager_Remove(t *testing.T) {
    // Test removing server
    // Test not found error
}

func TestMCPManager_Update(t *testing.T) {
    // Test partial updates
}
```

### 5.2 Integration Tests

**Package:** `cmd/spin`

```go
func TestMCPCommand_Integration(t *testing.T) {
    // Create temp config
    // Run: spin mcp add filesystem npx -y @mcp/...
    // Verify config file updated
    // Run: spin mcp list
    // Verify output correct
    // Run: spin mcp get filesystem
    // Verify details correct
    // Run: spin mcp remove filesystem
    // Verify removed from config
}
```

### 5.3 Coverage Target

- **Unit Tests:** ≥90% coverage for MCPManager
- **Integration Tests:** ≥85% coverage for Cobra commands
- **Overall:** ≥85% coverage for Phase 4.1

---

## 6. User Experience

### 6.1 Help Output

```bash
$ spin mcp --help
Manage MCP (Model Context Protocol) servers.

MCP servers extend Spin with additional capabilities like filesystem
access, database queries, and API integrations.

Usage:
  spin mcp [command]

Available Commands:
  add         Add a new MCP server
  list        List all configured MCP servers
  get         Show details of an MCP server
  remove      Remove an MCP server
  update      Update an MCP server configuration

Flags:
  -h, --help   help for mcp

Use "spin mcp [command] --help" for more information about a command.
```

### 6.2 Example Session

```bash
# Add filesystem server
$ spin mcp add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace
Added MCP server 'filesystem' to /home/user/.spin/spin.yaml

# List servers
$ spin mcp list
NAME          COMMAND                                       STATUS
filesystem    npx -y @modelcontextprotocol/server-...      configured

# Get details
$ spin mcp get filesystem
Name: filesystem
Command: npx
Args:
  - -y
  - @modelcontextprotocol/server-filesystem
  - /workspace
Environment: (none)
Source: /home/user/.spin/spin.yaml

# Add another server with environment
$ spin mcp add github mcp-server-github --token-file ~/.github-token
Added MCP server 'github' to /home/user/.spin/spin.yaml

# Remove server
$ spin mcp remove filesystem
Remove MCP server 'filesystem'? (y/N): y
Removed MCP server 'filesystem' from /home/user/.spin/spin.yaml
```

---

## 7. Quality Gates

### 7.1 Definition of Done

- [x] FRD reviewed and approved
- [ ] All unit tests written (TDD)
- [ ] All unit tests passing (≥90% coverage)
- [ ] Integration tests passing (≥85% coverage)
- [ ] `make lint` passes (zero errors)
- [ ] `go test -race ./...` passes
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] User documentation updated
- [ ] Examples provided
- [ ] ROADMAP Phase 4.1 marked complete

### 7.2 Acceptance Criteria

1. ✅ Users can add MCP servers via CLI
2. ✅ Users can list all configured servers
3. ✅ Users can view server details
4. ✅ Users can remove servers
5. ✅ Config file updates are atomic and safe
6. ✅ Works with YAML, TOML, and JSON configs
7. ✅ Clear error messages for all error cases
8. ✅ Help documentation is comprehensive

---

## 8. References

- [MCP Specification](https://modelcontextprotocol.io/specification)
- [Internal MCP Client](../../internal/mcp/client/)
- [Config Loader](../../internal/config/loader.go)
- [Cobra Documentation](https://cobra.dev/)
- [Phase 4.1 ROADMAP](../ui-modules/ROADMAP.md#41-mcp-management-commands)

---

**Author:** Spin Development Team
**Created:** 2025-10-05
**Last Updated:** 2025-10-05
**Version:** 1.0
