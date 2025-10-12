# Spin Internal Packages Documentation

This directory contains documentation for Spin's internal packages.

## Package Index

### Core Packages

| Package | Description | Coverage |
|---------|-------------|----------|
| [version](version.md) | Version information and build metadata | 100% |
| [core](core.md) | Core business logic and agent orchestration | 85.1% |
| [llm](llm.md) | Vendor-agnostic LLM provider interfaces | 94.6% |
| [auth](auth.md) | Authentication and credential management | 95.2% |

### Infrastructure Packages

| Package | Description | Coverage |
|---------|-------------|----------|
| [protocol](protocol.md) | JSON-RPC protocol for IDE integration | 88.3% |
| [appserver](appserver.md) | JSON-RPC application server | 89.5% |
| [config](config.md) | Configuration management (YAML/TOML/JSON) | 88.1% |
| [mcp](mcp.md) | Model Context Protocol client | - |
| [security](security.md) | Sandbox and security enforcement | - |
| [session](session.md) | Session state management | - |
| [tools](tools.md) | Tool registry and implementations | - |
| [filesearch](filesearch.md) | File scanning and fuzzy matching with gitignore support | 93.6% |
| [git](git.md) | Git repository operations and context gathering | 27.3% |

### UI Packages

| Package | Description | Coverage |
|---------|-------------|----------|
| [ui-blocks](ui-blocks.md) | Block data model and rendering system | - |
| [ui-output](ui-output.md) | Append-only output with streaming support | - |
| [exec](exec.md) | Non-interactive/headless execution mode | 76.7% |

### Shared Utilities

| Package | Description | Coverage |
|---------|-------------|----------|
| [pathutil](pathutil.md) | Secure path validation and manipulation | 84.6% |
| [strutil](strutil.md) | String manipulation utilities for code processing | 95.5% |
| [ansi](ansi.md) | ANSI escape sequence handling for terminal formatting | 97.9% |
| [patchapply](patchapply.md) | AI-friendly patch format parser for safe file modifications | 91.1% |
| [types](types.md) | Shared helper types for tool arguments | - |

---

## Quick Reference

### Core Architecture

**[core](core.md)** - Core business logic with Manager, Agent, Conversation, Turn, and Task abstractions.

**[llm](llm.md)** - LLM provider abstraction supporting OpenAI, Anthropic, Ollama, and LM Studio.

**[auth](auth.md)** - Secure credential management with platform-specific keystores (macOS Keychain, Linux Secret Service, Windows Credential Manager).

### User Interfaces

**[exec](exec.md)** - Non-interactive execution mode for CI/CD and automation.

**[ui-blocks](ui-blocks.md)** - Block-based timeline rendering for TUI (EXECUTE, PLAN, READ, GREP, APPLY_PATCH, SUMMARY, TESTING, NOTICE, ERROR).

**[ui-output](ui-output.md)** - Streaming output with coalescing for smooth LLM response rendering.

### Integration

**[protocol](protocol.md)** - JSON-RPC 2.0 protocol for bidirectional communication between UI and core.

**[appserver](appserver.md)** - WebSocket/HTTP server exposing protocol for IDE extensions and web clients.

**[mcp](mcp.md)** - Model Context Protocol client for external tool integration.

### Configuration & Tools

**[config](config.md)** - Multi-format configuration (YAML/TOML/JSON) with environment variable support.

**[tools](tools.md)** - Extensible tool registry with built-in tools and custom tool support.

**[session](session.md)** - Persistent conversation sessions with history tracking.

**[security](security.md)** - Sandboxing (Landlock on Linux, Seatbelt on macOS) and command validation.

### Utilities

**[pathutil](pathutil.md)** - Secure path validation and manipulation utilities preventing path traversal attacks.

**[strutil](strutil.md)** - String manipulation utilities including line operations, indentation detection, similarity algorithms, and case conversion.

**[ansi](ansi.md)** - ANSI escape sequence handling for terminal formatting with parsing, stripping, and fluent styling API.

**[patchapply](patchapply.md)** - AI-friendly patch format parser for safe file modifications with comprehensive path validation and error reporting.

**[types](types.md)** - Shared `ToolCallArguments` container with typed accessors and error contracts.

**[version](version.md)** - Build-time version injection via ldflags.

---

## Documentation Status

✅ **Complete** - All major packages have documentation
📊 **Test Coverage** - Most packages have >85% test coverage
🔄 **Maintained** - Documentation updated alongside code changes

---

**Last Updated:** 2025-10-12
**Latest Additions:**
- patchapply - Patch Parser for AI-Friendly File Modifications (91.1% coverage)
- filesearch - File Scanning and Fuzzy Matching with Gitignore Support (93.6% coverage)
