# Spin Examples

This directory contains examples demonstrating how to use Spin's TUI (Terminal User Interface) and configuration files.

## TUI Examples

### 1. Minimal TUI Demo

Location: [`tui-demo/`](tui-demo/)

The simplest possible TUI usage (~50 lines). Demonstrates:
- Initializing the PureTTY adapter
- Printing lines to output
- Accepting user input
- Clean shutdown with Ctrl-C or Ctrl-D

Run:
```bash
cd examples/tui-demo
go run main.go
```

**Learn more:** [tui-demo/README.md](tui-demo/README.md)

### 2. Streaming Demo

Location: [`tui-streaming/`](tui-streaming/)

Shows how to stream LLM responses token-by-token. Demonstrates:
- Word-by-word and character-by-character streaming
- Realistic LLM response timing simulation
- Coalescing behavior with fast streams (8.7M chunks/sec)
- Transient status indicators

Run:
```bash
cd examples/tui-streaming
go run main.go
```

**Learn more:** [tui-streaming/README.md](tui-streaming/README.md)

### 3. Block Types Demo

Location: [`tui-blocks/`](tui-blocks/)

Comprehensive demonstration of all 9 block types with navigation. Demonstrates:
- All block types (EXECUTE, PLAN, READ, GREP, APPLY_PATCH, SUMMARY, TESTING, NOTICE, ERROR)
- Block metadata and rendering
- Timeline navigation (PgUp/PgDn, jump, filter)
- Block actions (fold/expand, copy, save, rerun)

Run:
```bash
cd examples/tui-blocks
go run main.go
```

**Learn more:** [tui-blocks/README.md](tui-blocks/README.md)

## Configuration Examples

Configuration files for different LLM providers:

- [`config-openai.yaml`](config-openai.yaml) - OpenAI/Azure OpenAI setup
- [`config-anthropic.yaml`](config-anthropic.yaml) - Anthropic Claude setup
- [`config-ollama.yaml`](config-ollama.yaml) - Ollama (local models) setup
- [`config-lmstudio.yaml`](config-lmstudio.yaml) - LM Studio setup
- [`config-custom.yaml`](config-custom.yaml) - Custom provider template

**Provider configuration guide:** [PROVIDER-CONFIG.md](PROVIDER-CONFIG.md)

## Prerequisites

- Go 1.23 or later
- A terminal that supports ANSI escape codes

```bash
go version
```

## More Information

- [TUI Documentation](../docs/tui.md) - Complete guide with keymap, block types, troubleshooting
- [Performance Benchmarks](../docs/performance.md) - TUI scalability and throughput
- [Project README](../README.md) - Main project documentation
