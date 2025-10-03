# Spin - Technical Architecture Overview

## Executive Summary

Spin is an open-source autonomous coding agent that runs locally on user computers. It provides a command-line interface (CLI), IDE integrations, and programmatic SDK access. The project is written in Go (1.24+) following idiomatic Go practices and the standard project layout guidelines.

**Project Location:** `/home/dmytrogajewski/sources/spin`

**License:** Apache 2.0

**Philosophy:** Vendor-agnostic, zero lock-in. Works with any OpenAI-compatible API endpoint including Ollama, LMStudio, LocalAI, vLLM, and cloud providers.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      User Interfaces                        │
├─────────────────┬─────────────────┬─────────────────────────┤
│   TUI (Bubble)  │   IDE Extension │   SDK (Go package)      │
│   (cmd/tui)     │   (cmd/server)  │   (pkg/sdk)             │
└────────┬────────┴────────┬────────┴──────────┬──────────────┘
         │                 │                   │
         └─────────────────┼───────────────────┘
                           │
         ┌─────────────────▼─────────────────┐
         │     Core Business Logic           │
         │        (internal/core)            │
         │  • Task orchestration             │
         │  • LLM provider abstraction       │
         │  • Tool execution                 │
         │  • State management               │
         └─────────────────┬─────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────▼─────┐    ┌─────▼──────┐    ┌─────▼──────┐
    │ Security │    │  Protocol  │    │   Tools    │
    │ Sandbox  │    │    Layer   │    │ & Utilities│
    └──────────┘    └────────────┘    └────────────┘
```

## Project Structure

Following the [Go Standard Project Layout](https://github.com/golang-standards/project-layout):

```
spin/
├── cmd/                    # Application entry points
│   ├── spin/              # Main CLI application
│   ├── tui/               # Interactive terminal UI
│   ├── server/            # JSON-RPC server for IDE extensions
│   └── exec/              # Headless/non-interactive mode
├── internal/              # Private application code
│   ├── core/             # Core agent orchestration
│   ├── llm/              # LLM provider implementations
│   │   ├── openai/       # OpenAI-compatible API
│   │   ├── ollama/       # Ollama-specific optimizations
│   │   ├── lmstudio/     # LMStudio-specific optimizations
│   │   └── provider.go   # Provider interface
│   ├── security/         # Sandbox and execution policy
│   ├── mcp/              # Model Context Protocol
│   ├── tools/            # Agent tools (file ops, git, etc.)
│   ├── protocol/         # Internal protocol definitions
│   └── config/           # Configuration management
├── pkg/                  # Public library code
│   ├── sdk/              # SDK for programmatic access
│   ├── patch/            # Diff/patch operations
│   ├── search/           # File search functionality
│   └── git/              # Git integration utilities
├── api/                  # API definitions (OpenAPI, Protocol Buffers)
├── configs/              # Configuration file templates and examples
├── scripts/              # Build, install, and release scripts
├── test/                 # Additional test data and fixtures
├── docs/                 # User-facing documentation
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Core Components

### 1. **Command Line Interface** (`cmd/`)

**Purpose:** Multiple entry points for different interaction modes.

**Structure:**
- `cmd/spin/` - Main CLI multitool (cobra-based)
- `cmd/tui/` - Interactive Bubble Tea TUI
- `cmd/server/` - JSON-RPC server for IDE extensions
- `cmd/exec/` - Headless execution for automation

**Key Features:**
- Unified command structure
- Graceful degradation (TUI → plain CLI)
- Context-aware subcommands

### 2. **Internal Application Code** (`internal/`)

#### A. Core Orchestration (`internal/core/`)

**Purpose:** Central agent logic and task coordination.

**Key Packages:**
- `agent/` - Main agent loop and decision making
- `session/` - Conversation state management
- `executor/` - Tool execution coordination
- `planner/` - Task decomposition and planning

**Responsibilities:**
- Coordinate LLM interactions
- Manage conversation context
- Execute tools safely
- Handle streaming responses

#### B. LLM Provider Abstraction (`internal/llm/`)

**Purpose:** Vendor-agnostic LLM integration.

**Interface:**
```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan CompletionChunk, error)
    Models(ctx context.Context) ([]Model, error)
    Capabilities() Capabilities
}
```

**Implementations:**
- `openai/` - OpenAI-compatible APIs (default)
- `ollama/` - Ollama-specific features (local models)
- `lmstudio/` - LMStudio optimizations
- `anthropic/` - Anthropic Claude (if API compatible)

**Configuration:**
```yaml
providers:
  - name: ollama
    endpoint: http://localhost:11434/v1
    model: codellama:13b
    type: ollama
  - name: lmstudio
    endpoint: http://localhost:1234/v1
    model: codellama-13b-instruct
    type: openai-compatible
  - name: vllm
    endpoint: http://gpu-server:8000/v1
    model: deepseek-coder-33b
    type: openai-compatible
```

#### C. Security & Sandboxing (`internal/security/`)

**Packages:**
- `policy/` - Command validation engine
- `sandbox/` - OS-specific isolation
  - `sandbox_linux.go` - Landlock LSM
  - `sandbox_darwin.go` - sandbox-exec wrapper
- `hardening/` - Process security measures

**Policy Engine:**
- Uses Starlark or Go templates for policy definitions
- Command classification: safe, interactive, dangerous, forbidden
- Configurable per-workspace

#### D. Model Context Protocol (`internal/mcp/`)

**Purpose:** MCP client and server implementations.

**Packages:**
- `client/` - Connect to external MCP servers
- `server/` - Expose spin as MCP server
- `transport/` - stdio and HTTP/SSE transports
- `types/` - Protocol type definitions

#### E. Tools (`internal/tools/`)

**Purpose:** Agent capabilities and actions.

**Tool Categories:**
- `filesystem/` - Read, write, search files
- `git/` - Repository operations
- `shell/` - Command execution
- `patch/` - Apply code changes
- `search/` - Semantic code search
- `lsp/` - Language server integration (future)

**Tool Interface:**
```go
type Tool interface {
    Name() string
    Description() string
    Parameters() ParameterSchema
    Execute(ctx context.Context, params map[string]interface{}) (Result, error)
}
```

#### F. Protocol Layer (`internal/protocol/`)

**Purpose:** Communication formats for IDE and SDK integration.

**Formats:**
- JSON-RPC 2.0 for IDE extensions
- Streaming events for real-time updates
- Thread-based conversation model

### 3. **Public SDK** (`pkg/sdk/`)

**Purpose:** Programmatic API for embedding spin in applications.

**Example Usage:**
```go
import "github.com/yourusername/spin/pkg/sdk"

client := sdk.NewClient(sdk.Config{
    Provider: "ollama",
    Model:    "codellama:13b",
    WorkDir:  "/path/to/project",
})

thread := client.NewThread()
run := thread.CreateRun("Refactor the auth module")

for event := range run.Stream() {
    fmt.Println(event.Type, event.Content)
}
```

**Key Features:**
- Thread-based conversations
- Streaming and blocking modes
- Custom working directories
- Tool approval callbacks

### 4. **Public Utilities** (`pkg/`)

**Reusable Packages:**
- `pkg/patch/` - Unified diff operations
- `pkg/search/` - Fuzzy file search
- `pkg/git/` - Git operations wrapper
- `pkg/ansi/` - ANSI escape handling

## Data Flow

### 1. Interactive TUI Flow

```
User Input → cmd/tui → internal/core/agent → LLM Provider
                ↓            ↓
         Display Output  Execute Tools
                ↓            ↓
         Update UI (Bubble Tea)  Apply Changes
                           ↓
                    Validate (policy)
                           ↓
                    Execute (sandbox)
```

### 2. IDE Extension Flow

```
IDE → cmd/server (JSON-RPC) → internal/core → LLM
         ↓                        ↓
    Stream Events          Execute Actions
         ↓                        ↓
    Update IDE UI       Return Results
```

### 3. SDK Flow

```
Application → pkg/sdk → internal/core → LLM Provider
                ↓              ↓
         Handle Callbacks  Execute Task
                ↓              ↓
         Return Result   Stream Updates
```

## Security Architecture

### Multi-Layer Defense

1. **Execution Policy** (`internal/security/policy`)
   - Starlark-based or Go template policy engine
   - Command classification with regex patterns
   - Argument validation and sanitization
   - Per-workspace policy overrides

2. **Platform Sandboxing** (`internal/security/sandbox`)
   - **macOS:** `sandbox-exec` with custom profiles
   - **Linux:** Landlock LSM for filesystem isolation
   - **Windows:** (future) AppContainer or WSL sandbox
   
3. **Sandbox Modes**
   - `read-only` - Default, no write access
   - `workspace-only` - Write within workspace directory
   - `full-access` - Disabled sandbox (container environments)

4. **Process Hardening**
   - Capability dropping (Linux)
   - Memory locking for credentials
   - Secure environment variable handling
   - Signal handling for cleanup

5. **API Key Protection**
   - Environment variable isolation
   - Memory-locked storage
   - Optional separate proxy process

## Configuration System

**File:** `~/.spin/config.yaml` (or TOML)

**Structure:**
```yaml
# LLM Provider Configuration
providers:
  default: ollama-local
  
  ollama-local:
    type: ollama
    endpoint: http://localhost:11434
    model: codellama:13b
    
  lmstudio:
    type: openai-compatible
    endpoint: http://localhost:1234/v1
    model: codellama-13b-instruct
    
  cloud-fallback:
    type: openai
    endpoint: https://api.openai.com/v1
    api_key_env: OPENAI_API_KEY
    model: gpt-4

# Security
security:
  sandbox:
    mode: workspace-only
    policy_file: ~/.spin/policy.star
  
  allowed_commands:
    - git
    - make
    - go

# MCP Servers
mcp:
  servers:
    filesystem:
      command: mcp-server-filesystem
      args: ["/workspace"]
    
    github:
      command: mcp-server-github
      env:
        GITHUB_TOKEN: ${GITHUB_TOKEN}

# UI Preferences
ui:
  theme: auto
  editor: vim
  pager: less

# Advanced
agent:
  max_iterations: 50
  timeout: 300s
  tools:
    - filesystem
    - git
    - shell
    - search
```

**Precedence:** CLI flags > Environment > Config file > Defaults

## Model Context Protocol (MCP)

Spin functions as both MCP client and server:

### As MCP Client
- Connect to external MCP servers for extended capabilities
- Discover tools and resources dynamically
- Support stdio and SSE transports

### As MCP Server
```bash
spin mcp serve --transport stdio
spin mcp serve --transport sse --port 8080
```

**Exposed Tools:**
- File operations (read, write, search)
- Git operations
- Code search
- Shell execution (policy-controlled)

## Build System

**Technology:** Go 1.24+ with standard toolchain

**Build Commands:**
```bash
# Development build
make build

# Release build (optimized)
make release

# Cross-compile
make build-all

# Run tests
make test

# Install locally
make install
```

**Makefile Targets:**
- `build` - Development binary
- `release` - Optimized binary (stripped, compressed)
- `test` - Run all tests
- `lint` - golangci-lint
- `clean` - Remove artifacts
- `install` - Install to $GOPATH/bin

**Build Flags:**
```bash
go build -ldflags="-s -w -X main.version=$(VERSION)" ./cmd/spin
```

## Technology Stack

### Core Technologies
- **Go 1.24+** - Primary language
- **Bubble Tea** - Terminal UI framework
- **Cobra** - CLI framework
- **Viper** - Configuration management

### Key Dependencies
- `github.com/charmbracelet/bubbletea` - TUI
- `github.com/spf13/cobra` - CLI
- `github.com/spf13/viper` - Config
- `golang.org/x/sync/errgroup` - Concurrency
- `github.com/sourcegraph/jsonrpc2` - JSON-RPC
- Standard library for HTTP, JSON, etc.

### Optional Dependencies
- `go.starlark.net` - Policy engine (if Starlark chosen)
- `github.com/sashabaranov/go-openai` - OpenAI client reference

## Distribution Channels

1. **Go Install:** `go install github.com/yourusername/spin/cmd/spin@latest`
2. **GitHub Releases:** Pre-built binaries for major platforms
3. **Homebrew:** `brew install spin` (future)
4. **Package Managers:** apt, yum, pacman (future)
5. **Docker:** `docker run spin` (containerized mode)

## Deployment Targets

- **Linux:** x86_64, arm64, arm (musl and glibc)
- **macOS:** x86_64, arm64 (Apple Silicon)
- **Windows:** x86_64, arm64 (native and WSL)
- **BSD:** FreeBSD, OpenBSD (community supported)

## Module Dependency Graph (Simplified)

```
cmd/spin → {cmd/tui, cmd/server, cmd/exec}
              ↓
         internal/core → {internal/llm, internal/protocol, internal/mcp}
              ↓
              ├─→ internal/security/policy
              ├─→ internal/security/sandbox
              ├─→ internal/tools/*
              └─→ pkg/{patch,search,git}
```

## Development Workflow

### Building from Source

```bash
git clone https://github.com/yourusername/spin.git
cd spin
make build
./bin/spin --version
```

### Running Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# Integration tests
make test-integration

# Specific package
go test ./internal/core/...
```

### Development Mode

```bash
# Run with local config
make run ARGS="--config ./configs/dev.yaml"

# Run TUI in dev mode
go run ./cmd/tui -v --provider ollama

# Watch and rebuild
make watch
```

## Key Design Decisions

1. **Go over Rust:** Simpler deployment, faster compile times, excellent stdlib, widespread adoption
2. **Vendor-Agnostic:** Works with any OpenAI-compatible API - no lock-in
3. **Provider Abstraction:** Easy to add new LLM backends (Ollama, LMStudio, etc.)
4. **Standard Project Layout:** Following Go community conventions
5. **Minimal Dependencies:** Leverage Go stdlib, avoid heavy frameworks
6. **MCP Protocol:** Standard protocol for tool/resource integration
7. **Sandbox-First:** Security by default with opt-in relaxations
8. **Local-First:** Optimized for local LLMs (Ollama, LMStudio)

## Extension Points

1. **Custom LLM Providers:** Implement `llm.Provider` interface
2. **Custom Tools:** Implement `tools.Tool` interface
3. **MCP Servers:** Add external capabilities via MCP protocol
4. **Execution Policies:** Custom Starlark/template files
5. **SDK Integration:** Embed spin in custom applications
6. **IDE Extensions:** Connect via JSON-RPC server

## Provider Configuration Examples

### Ollama (Local)
```yaml
providers:
  ollama:
    type: ollama
    endpoint: http://localhost:11434
    model: codellama:13b
    options:
      temperature: 0.7
      num_ctx: 8192
```

### LMStudio (Local)
```yaml
providers:
  lmstudio:
    type: openai-compatible
    endpoint: http://localhost:1234/v1
    model: codellama-13b-instruct
    options:
      temperature: 0.7
      max_tokens: 4096
```

### vLLM (Self-Hosted)
```yaml
providers:
  vllm:
    type: openai-compatible
    endpoint: http://gpu-server:8000/v1
    model: deepseek-coder-33b-instruct
```

### OpenAI (Cloud Fallback)
```yaml
providers:
  openai:
    type: openai
    endpoint: https://api.openai.com/v1
    api_key_env: OPENAI_API_KEY
    model: gpt-4
```

### Multiple Providers with Routing
```yaml
providers:
  default: local
  
  local:
    type: ollama
    endpoint: http://localhost:11434
    model: codellama:13b
    
  powerful:
    type: openai-compatible
    endpoint: http://gpu-cluster:8000/v1
    model: deepseek-coder-70b
    
routing:
  rules:
    - if: "task.complexity > 0.8"
      provider: powerful
    - if: "task.type == 'chat'"
      provider: local
```

## Next Steps

For detailed module-specific documentation, see:
- `core-module.md` - Core business logic
- `security-modules.md` - Security and sandboxing
- `llm-providers.md` - LLM provider implementations
- `protocol-modules.md` - Communication protocols
- `tools-modules.md` - Agent tools and capabilities
- `mcp-modules.md` - Model Context Protocol integration
- `ui-modules.md` - User interface implementations
- `sdk-module.md` - Go SDK for programmatic access

## Comparison with Codex

| Feature | Codex (Rust) | Spin (Go) |
|---------|-------------|-----------|
| **Language** | Rust | Go |
| **LLM Providers** | OpenAI-centric | Vendor-agnostic |
| **Local LLMs** | Limited | First-class (Ollama, LMStudio) |
| **Distribution** | npm wrapper | Native Go install |
| **Compile Time** | Slow (Rust) | Fast (Go) |
| **Memory Safety** | Compile-time | Runtime + best practices |
| **Concurrency** | Tokio async | Goroutines |
| **TUI Framework** | Ratatui | Bubble Tea |
| **Dependencies** | 35+ crates | Minimal (Go stdlib focus) |
| **License** | Apache 2.0 | Apache 2.0 |

## Philosophy

Spin is built on these principles:

1. **Freedom:** No vendor lock-in, works with any compatible LLM
2. **Local-First:** Optimized for local models (privacy, cost, speed)
3. **Simplicity:** Clean Go code, minimal dependencies
4. **Security:** Sandboxed by default, explicit permissions
5. **Extensibility:** MCP protocol, plugin architecture
6. **Standards:** Follow Go conventions and community practices
7. **Transparency:** Open source, readable code, clear documentation


