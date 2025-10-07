# Spin Internal Packages Documentation

This directory contains comprehensive documentation for all internal packages in the Spin project.

## Package Index

### Core Packages

| Package | Description | Status | Coverage |
|---------|-------------|--------|----------|
| [version](version.md) | Version information and build metadata | ✅ Complete | 100% |
| [core](core.md) | Core business logic and agent orchestration | ✅ Complete | 85.1% |
| [llm](llm.md) | Vendor-agnostic LLM provider interfaces | ✅ Complete | 94.6% |
| [auth](auth.md) | Authentication and credential management | ✅ Complete | 95.2% |

### Infrastructure Packages

| Package | Description | Status | Coverage |
|---------|-------------|--------|----------|
| [protocol](protocol.md) | JSON-RPC protocol for IDE integration | ✅ Complete | 88.3% |
| [appserver](appserver.md) | JSON-RPC application server | ✅ Complete | 89.5% |
| [config](config.md) | Configuration management (YAML/TOML/JSON) | ✅ Complete | 88.1% |
| [mcp](mcp.md) | Model Context Protocol client | ✅ Complete | - |
| [security](security.md) | Sandbox and security enforcement | ✅ Complete | - |
| [session](session.md) | Session state management | ✅ Complete | - |
| [tools](tools.md) | Tool registry and implementations | ✅ Complete | - |

### Shared Utilities

| Package | Description | Status | Coverage |
|---------|-------------|--------|----------|
| [types](types.md) | Shared helper types for tool arguments | ✅ Complete | - |

### UI Modules

| Package | Description | Status | Coverage |
|---------|-------------|--------|----------|
| [exec](exec.md) | Non-interactive/headless execution mode | ✅ Complete | 76.7% |

---

## Quick Reference

### [version](version.md)
Version information management with build-time injection.

### [core](core.md)
Core business logic: Manager, Agent, Conversation, Turn, Task.

### [llm](llm.md)
LLM provider abstraction: OpenAI, Ollama, LMStudio.

### [auth](auth.md)
Secure credential management with platform keystores.

### [protocol](protocol.md)
JSON-RPC 2.0 protocol for IDE integration.

### [appserver](appserver.md)
WebSocket/HTTP server for external clients.

### [config](config.md)
Multi-format configuration (YAML/TOML/JSON) with validation.

### [mcp](mcp.md)
Model Context Protocol client.

### [security](security.md)
Sandboxing and command validation.

### [session](session.md)
Persistent conversation sessions.

### [tools](tools.md)
Tool registry and built-in tools.

### [types](types.md)
Shared tool argument helpers and error contracts.

### [exec](exec.md)
Non-interactive execution mode for CI/CD and automation.

---

**Last Updated:** 2025-10-05
**Total Packages:** 13
**Status:** ✅ All packages fully documented
