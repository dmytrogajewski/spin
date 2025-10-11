# Spin AI Agent

Spin is a powerful AI coding agent designed to help developers with autonomous task execution, code generation, and intelligent assistance. Built with a focus on security, type safety, and extensibility.

## Features

- **🤖 Autonomous AI Agent**: Powered by state-of-the-art language models with tool-calling capabilities
- **🛡️ Security-First Design**: Multi-layered security with command validation, sandboxing, and process hardening
- **🔧 Extensible Tool System**: Built-in tools with support for custom tool registration
- **📝 Type-Safe Architecture**: Fully typed with Go generics, eliminating runtime type errors
- **🎨 Interactive TUI**: Beautiful terminal interface with real-time event streaming
- **🔌 MCP Support**: Model Context Protocol integration for enhanced capabilities
- **📊 Observability**: Structured logging and OpenTelemetry tracing

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/dmytrogajewski/spin.git
cd spin

# Build the project
make build

# Run tests
make test
```

### Basic Usage

```bash
# Start interactive session
./bin/spin

# Execute a single command
./bin/spin exec "list all Go files in the current directory"

# Use with specific provider
./bin/spin exec --provider openai --model gpt-4 "explain this code"
```

## Architecture

Spin follows clean architecture principles with clear separation of concerns:

```
┌─────────────────────────────────────────┐
│           Terminal UI (TUI)             │
├─────────────────────────────────────────┤
│         Core Orchestration              │
│  • Manager • Agent • Conversation       │
├─────────────────────────────────────────┤
│      Security & Validation Layer        │
│  • Policy • Sandbox • Hardening         │
├─────────────────────────────────────────┤
│         Provider Interfaces             │
│  • LLM • Tools • MCP • Session          │
└─────────────────────────────────────────┘
```

## Key Components

### Core Package (`internal/core`)
The heart of Spin, providing:
- Conversation management
- Agent orchestration
- Event streaming
- State management
- Type-safe event system with generics

### Security Modules (`internal/security`)
Multi-layered defense system:
- **Policy Engine**: Command classification and validation
- **Sandbox**: Platform-specific filesystem isolation (Landlock on Linux, Seatbelt on macOS)
- **Hardening**: Process-level security measures

### LLM Providers (`internal/llm`)
Support for multiple providers:
- OpenAI / Azure OpenAI
- Anthropic Claude
- Google Gemini
- Ollama (local models)
- LM Studio

### Tool System (`internal/tools`)
Extensible tool framework:
- Built-in tools for common operations
- Type-safe tool arguments
- Custom tool registration
- MCP tool integration

### Terminal UI (TUI) (`internal/ui`)
Native-scrollback terminal interface with block-based timeline rendering:
- **Factory Droid principle**: Append-only transcript, preserves native scrollback
- **Block timeline**: Visual blocks for all agent actions (EXECUTE, PLAN, diffs, summaries)
- **Streaming output**: Real-time LLM response with coalescing (8.7M chunks/sec)
- **Keyboard navigation**: PgUp/PgDn, filtering, collapse/expand, copy/save
- **Performance**: 100k+ blocks without lag (0.52ms viewport render)
- **Command palette**: Fuzzy search with Ctrl-P
- **Works in**: SSH, tmux, screen (no alt-screen buffer)

**Documentation:**
- [Full TUI docs](docs/tui.md) - Complete guide with keymap, block types, troubleshooting
- [Performance](docs/performance.md) - Benchmarks and scalability

**Examples:**
- [Minimal TUI](examples/tui-demo/) - Simplest possible usage (~50 lines)
- [Streaming demo](examples/tui-streaming/) - LLM token streaming simulation
- [Block types demo](examples/tui-blocks/) - All 9 block types with navigation

## Type Safety

Recent refactoring introduced comprehensive type safety using Go generics:

### Type-Safe Events
```go
// Before: runtime type assertions
event.Data.(map[string]interface{})["content"]

// After: compile-time type safety
typedEvent := core.FromGenericEvent[core.ContentEventData](event)
content := typedEvent.Data.Content
```

### Type-Safe Tool Arguments
```go
// Type-safe argument handling
args := types.ToolCallArguments{}
command, err := args.GetString("command")
```

## Security Features

### Command Validation
Commands are classified into safety levels:
- **Safe**: Auto-approved read-only operations
- **Interactive**: Requires approval for write operations
- **Dangerous**: Requires explicit approval
- **Forbidden**: Never executed

### Sandboxing
Platform-specific isolation:
- **Linux**: Landlock LSM (kernel 5.13+)
- **macOS**: Seatbelt/sandbox-exec
- **Windows**: Coming soon

### Process Hardening
- Disabled core dumps
- Disabled ptrace attachment
- Sanitized environment variables
- Memory protection

## Configuration

Create a configuration file at `~/.spin/config.yaml`:

```yaml
# LLM Settings
model: claude-3-5-sonnet-20241022
temperature: 0.7
max_tokens: 4096

# Security Settings
require_approval: true
sandbox_mode: workspace_write

# Provider Settings
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
```

## Development

### Project Structure
```
spin/
├── cmd/               # Command-line applications
├── internal/          # Private application code
│   ├── core/         # Core business logic
│   ├── llm/          # LLM provider implementations
│   ├── security/     # Security modules
│   ├── tools/        # Tool implementations
│   ├── tui/          # Terminal UI
│   └── types/        # Shared type definitions
├── configs/          # Configuration files
├── docs/             # Documentation
├── examples/         # Example code
└── specs/           # Technical specifications
```

### Testing
```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run benchmarks
make bench

# Run linters
make lint
```

### Building
```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build with debug symbols
make debug
```

## Documentation

- [Core Package](docs/packages/core.md) - Core orchestration and management
- [Security Modules](specs/security-modules.md) - Detailed security architecture
- [Protocol Modules](specs/protocol-modules.md) - Communication protocols
- [Tools Modules](specs/tools-modules.md) - Tool system architecture
- [UI Modules](specs/ui-modules/spec.md) - Terminal UI design
- [Refactoring Summary](REFACTORING_SUMMARY.md) - Recent type safety improvements

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Guidelines
1. Maintain test coverage above 85%
2. Follow Go idioms and Google Go Style Guide
3. Document all exported APIs
4. Run linters before submitting PRs
5. Add benchmarks for performance-critical code

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with Go 1.23+
- Uses advanced Go features including generics
- Implements Google Go Style Guide best practices
- Inspired by clean architecture principles

## Support

For issues, questions, or suggestions:
- Open an issue on GitHub
- Check the [documentation](docs/)
- Review the [examples](examples/)

---

**Note**: Spin is under active development. APIs may change as we continue to improve the system.