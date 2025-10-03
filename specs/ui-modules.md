# Spin UI Modules - Technical Documentation

## Overview

Spin provides multiple user interface options for different use cases. This document covers:

1. **tui** - Interactive terminal user interface (primary)
2. **exec** - Non-interactive/headless mode (automation)
3. **cli** - Main CLI multitool (entry point)

---

## Module 1: spin-tui

**Path:** `cmd/spin-tui/`  
**Purpose:** Full-featured terminal user interface for interactive coding sessions

### Overview

The TUI is the primary interface for Spin, providing a rich, interactive experience in the terminal using the Bubble Tea framework.

**Framework:** Bubble Tea (Go terminal UI library based on The Elm Architecture)

### Architecture

```
cmd/spin-tui/
├── main.go              # Entry point
├── app.go               # Application model (Bubble Tea model)
├── ui/
│   ├── chat.go          # Chat message display
│   ├── input.go         # User input widget
│   ├── statusbar.go     # Status bar
│   ├── transcript.go    # Conversation history view
│   └── filepicker.go    # @ file search widget
├── handlers/
│   ├── keyboard.go      # Keyboard input processing
│   ├── events.go        # Core event processing
│   └── commands.go      # Bubble Tea commands
├── renderer/
│   ├── render.go        # Screen rendering
│   └── styles.go        # Lipgloss styles
└── state/
    └── state.go         # UI state management

internal/tui/
├── model.go             # Shared TUI models
├── messages.go          # Bubble Tea messages
└── styles.md            # Style guide
```

### Key Features

#### 1. **Chat Interface**

**Display:**
- Streaming AI responses
- Syntax-highlighted code blocks (via chroma)
- Formatted markdown (via glamour)
- ANSI color preservation
- Reasoning blocks (for compatible models)

**Interaction:**
- Type messages naturally
- Multi-line input support
- Paste large text blocks
- Edit previous messages (Esc-Esc)

#### 2. **File Search (@-trigger)**

**Usage:**
```
User: @
[File picker opens]
config ← user types
[Shows: config.toml, internal/config/config.go, ...]
[User presses Tab/Enter to select]
User: Read @config.toml and fix the bug
```

**Implementation:**
- Fuzzy search via `internal/filesearch` package
- Real-time filtering
- Keyboard navigation (↑↓ arrows)
- Instant path insertion

#### 3. **Tool Approval UI**

When AI proposes dangerous operations:

```
┌─────────────────────────────────────┐
│ Tool Call: shell                    │
├─────────────────────────────────────┤
│ Command: rm -rf node_modules        │
│                                     │
│ This command will delete files.     │
│                                     │
│ [A]pprove  [D]eny  [M]odify         │
└─────────────────────────────────────┘
```

**Options:**
- **Approve (A):** Execute as-is
- **Deny (D):** Reject and inform AI
- **Modify (M):** Edit command before execution

#### 4. **Status Bar**

**Information Displayed:**
- Current model (llama3.1, mixtral, gpt-4o, etc.)
- Sandbox mode (🔒 read-only, 📝 workspace-write)
- Working directory
- Connection status
- Tokens used (current turn / session)

**Example:**
```
llama3.1 | 🔒 read-only | ~/project | 1.2K / 5.4K tokens
```

#### 5. **Transcript View**

**Features:**
- Full conversation history
- Scroll through past messages
- Search within transcript (/)
- Export conversation (Ctrl+E)

#### 6. **Backtrack Mode (Esc-Esc)**

**Purpose:** Edit and resubmit previous messages

**Flow:**
1. Press Esc when input is empty
2. Press Esc again to enter backtrack mode
3. Transcript highlights last user message
4. Press Esc repeatedly to step to older messages
5. Press Enter to load message into input
6. Edit and resubmit (conversation forks)

**Use Case:** Fix typos, rephrase questions, try different approaches

#### 7. **Keyboard Shortcuts**

| Key | Action |
|-----|--------|
| Enter | Send message |
| Ctrl+C | Cancel current turn |
| Ctrl+D | Exit Spin |
| Esc-Esc | Enter backtrack mode |
| @ | Open file picker |
| ↑/↓ | Navigate file picker |
| Tab | Accept file picker suggestion |
| / | Search in transcript |
| Ctrl+E | Export conversation |
| Ctrl+L | Clear screen |
| PgUp/PgDn | Scroll transcript |

### State Management

#### App State (Bubble Tea Model)

```go
// AppState represents the different states of the TUI
type AppState int

const (
    StateIdle AppState = iota          // Waiting for user input
    StateWaitingResponse               // AI is generating response
    StateToolApproval                  // Waiting for tool approval
    StateFilePickerOpen                // @ file search active
    StateBacktrackMode                 // Esc-Esc mode
    StateExiting                       // Shutting down
)

// Model is the main Bubble Tea model
type Model struct {
    state       AppState
    input       textinput.Model
    transcript  []Message
    statusBar   StatusBar
    filePicker  FilePicker
    
    // Tool approval state
    pendingTool *ToolCall
    
    // Backtrack state
    backtrackIdx int
    
    // Core communication
    coreChan    chan CoreEvent
    
    // Viewport for scrolling
    viewport    viewport.Model
    
    // Dimensions
    width       int
    height      int
}
```

#### The Elm Architecture Pattern

```go
// Init initializes the model
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        textinput.Blink,
        waitForCoreEvent(m.coreChan),
    )
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyPress(msg)
    case tea.WindowSizeMsg:
        return m.handleResize(msg)
    case CoreEventMsg:
        return m.handleCoreEvent(msg)
    case TickMsg:
        return m, tick()
    }
    return m, nil
}

// View renders the UI
func (m Model) View() string {
    return lipgloss.JoinVertical(
        lipgloss.Top,
        m.renderTranscript(),
        m.renderInput(),
        m.renderStatusBar(),
    )
}
```

### Rendering Pipeline

**Process:**
1. Collect UI components (chat, input, status bar)
2. Calculate layout (using lipgloss)
3. Render each component to string buffer
4. Return final string

**Optimization:**
- Only redraw on model changes (Bubble Tea handles this)
- Efficient string building with strings.Builder
- Viewport for large transcripts (only render visible area)

### Styling

**Theme:** Defined using lipgloss

**Color Palette:**
- User messages: Blue
- AI messages: Green
- System messages: Yellow
- Errors: Red
- Code blocks: Syntax-highlighted (chroma)

**Customization:**
```go
var (
    UserStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("12")).
        Bold(true)
    
    AssistantStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("10"))
    
    ErrorStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("9")).
        Bold(true)
)
```

**Features:**
- Supports terminal color schemes
- Respects NO_COLOR environment variable
- Falls back to plain text if colors unavailable

### Integration with Core

**Communication:**
```go
// TUI → Core
type UserMessageEvent struct {
    Content string
}

// Send to core via channel
coreChan <- UserMessageEvent{Content: "Fix bug"}

// Core → TUI
// Listen for core events as Bubble Tea commands
func waitForCoreEvent(coreChan chan CoreEvent) tea.Cmd {
    return func() tea.Msg {
        event := <-coreChan
        return CoreEventMsg{Event: event}
    }
}

// Handle in Update()
case CoreEventMsg:
    switch msg.Event.Type {
    case EventAssistantDelta:
        // Append to transcript
        return m.appendDelta(msg.Event.Delta), waitForCoreEvent(m.coreChan)
    case EventToolCallProposed:
        // Show approval prompt
        return m.showApprovalPrompt(msg.Event.ToolCall), nil
    case EventTurnComplete:
        // Transition to idle
        return m.transitionToIdle(), waitForCoreEvent(m.coreChan)
    }
```

**Channel-Based:**
- Buffered channels for non-blocking communication
- Goroutine-safe message passing
- Bubble Tea command pattern for async operations

### Error Handling

**Display Strategy:**
- Inline errors in transcript
- Status bar for transient errors
- Modal overlays for critical errors

**Error Recovery:**
- Auto-reconnect on network failures
- Graceful degradation
- User-actionable error messages

**Implementation:**
```go
type ErrorDisplay struct {
    Message   string
    Severity  ErrorSeverity
    Timestamp time.Time
    Dismissible bool
}

func (m Model) showError(err error) Model {
    m.errors = append(m.errors, ErrorDisplay{
        Message:     err.Error(),
        Severity:    detectSeverity(err),
        Timestamp:   time.Now(),
        Dismissible: true,
    })
    return m
}
```

### Testing

**Test Infrastructure:**
- Table-driven tests for state transitions
- Mock core events
- Layout verification
- Snapshot testing for rendering

**Test Execution:**
```bash
go test ./cmd/spin-tui/...
go test ./internal/tui/...
```

**Example Test:**
```go
func TestBacktrackMode(t *testing.T) {
    m := NewModel()
    m.transcript = []Message{
        {Role: "user", Content: "First message"},
        {Role: "assistant", Content: "Response"},
        {Role: "user", Content: "Second message"},
    }
    
    // Enter backtrack mode
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
    
    assert.Equal(t, StateBacktrackMode, m.state)
    assert.Equal(t, 2, m.backtrackIdx) // Points to last user message
}
```

### Performance

**Metrics:**
- Render latency: <16ms (60 FPS)
- Input latency: <5ms
- Memory usage: <30MB
- Startup time: <100ms

**Optimization Techniques:**
- Lazy rendering (viewport only shows visible area)
- String interning for repeated text
- Efficient buffer reuse
- Minimal allocations in hot paths

---

## Module 2: spin-exec

**Path:** `cmd/spin-exec/`  
**Purpose:** Non-interactive, headless execution mode

### Overview

`spin exec` runs Spin without a TUI, suitable for:
- CI/CD pipelines
- Automation scripts
- Batch processing
- Cron jobs
- Containerized environments

### Usage

**Basic:**
```bash
spin exec "Run all tests and fix failures"
```

**From stdin:**
```bash
echo "Refactor authentication module" | spin exec
```

**With options:**
```bash
spin exec --model llama3.1 --sandbox workspace-write "Deploy to staging"
```

### Behavior

**Output:**
- Streams AI responses to stdout
- Logs to stderr (if SPIN_LOG_LEVEL set)
- Exits with code 0 on success, non-zero on failure

**Completion:**
- Runs until AI determines task is complete
- Or until error occurs
- Or until timeout (if specified)

**No Interaction:**
- Automatically approves "safe" operations
- Fails on dangerous operations (unless `--auto-approve`)
- No user prompts

### Options

```
spin exec [OPTIONS] <PROMPT>

Options:
  --model <MODEL>              Model to use (llama3.1, mixtral, etc.)
  --provider <PROVIDER>        Provider (ollama, lmstudio, openai, anthropic)
  --sandbox <MODE>             Sandbox mode
  --auto-approve               Approve all operations (DANGEROUS)
  --timeout <DURATION>         Max execution time (e.g., 5m, 1h)
  --config <KEY=VALUE>         Config overrides
  --cd <DIR>                   Working directory
  --format <FORMAT>            Output format (text, json)
```

### Use Cases

#### 1. CI/CD Integration

```yaml
# .github/workflows/spin.yml
name: Spin Auto-Fix
on: [pull_request]
jobs:
  fix:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install Spin
        run: go install github.com/yourusername/spin/cmd/spin@latest
      - name: Fix linter errors
        run: spin exec "Fix all linter errors"
      - name: Show changes
        run: git diff
```

#### 2. Scheduled Maintenance

```bash
# crontab
0 2 * * * cd /project && spin exec "Update dependencies and run tests"
```

#### 3. Batch Processing

```bash
for dir in projects/*; do
  (cd "$dir" && spin exec "Migrate to new API")
done
```

#### 4. Docker Integration

```dockerfile
FROM golang:1.24
RUN go install github.com/yourusername/spin/cmd/spin@latest
COPY . /workspace
WORKDIR /workspace
RUN spin exec "Build and test"
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication failed |
| 3 | Task failed |
| 4 | Timeout exceeded |
| 5 | User cancellation (SIGINT) |

### Output Format

**Default (Human-Readable):**
```
[Spin] Starting task: Fix tests
[Spin] Reading test files...
[Spin] Found 3 failing tests
[Spin] Applying fixes...
[Spin] ✓ Tests now passing
[Spin] Task complete
```

**Structured (JSON):**
```bash
spin exec --format json "Analyze code" | jq
```

```json
{
  "status": "complete",
  "messages": [...],
  "files_modified": ["internal/main.go"],
  "commands_executed": ["go test ./..."],
  "tokens_used": 1234
}
```

### Architecture

```go
// cmd/spin-exec/main.go
package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "time"
    
    "github.com/yourusername/spin/internal/core"
    "github.com/yourusername/spin/internal/config"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func run() error {
    args := parseArgs()
    cfg, err := config.Load(args.ConfigPath)
    if err != nil {
        return fmt.Errorf("load config: %w", err)
    }
    
    // Apply CLI overrides
    cfg = cfg.WithOverrides(args.Overrides)
    
    // Initialize core
    ctx := context.Background()
    if args.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, args.Timeout)
        defer cancel()
    }
    
    core, err := core.New(cfg)
    if err != nil {
        return fmt.Errorf("initialize core: %w", err)
    }
    defer core.Close()
    
    // Run task
    result, err := core.RunTask(ctx, args.Prompt)
    if err != nil {
        return fmt.Errorf("run task: %w", err)
    }
    
    // Output results
    return outputResults(result, args.Format)
}
```

**Simplified Architecture:**
- No UI components
- Direct output to stdout/stderr
- Minimal dependencies
- Fast startup

### Error Handling

**Strategy:**
- Print errors to stderr
- Exit immediately on critical errors
- Retry transient failures (network, rate limits)
- Log full error chain

**Example:**
```go
func formatError(err error) string {
    var b strings.Builder
    b.WriteString("Error: ")
    b.WriteString(err.Error())
    
    // Unwrap error chain
    if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
        b.WriteString("\nCaused by:")
        for i := 0; ; i++ {
            err = unwrapper.Unwrap()
            if err == nil {
                break
            }
            fmt.Fprintf(&b, "\n  %d: %v", i, err)
            unwrapper, ok = err.(interface{ Unwrap() error })
            if !ok {
                break
            }
        }
    }
    
    return b.String()
}
```

### Integration with Core

**Same Core Logic:**
- Uses `internal/core` package
- Same model interaction
- Same tool execution
- Same safety checks

**Different Interface:**
- No UI events
- No user prompts (in non-interactive mode)
- Direct logging with `log/slog`

---

## Module 3: spin-cli

**Path:** `cmd/spin/`  
**Purpose:** Main CLI multitool and entry point

### Overview

`spin` is the unified entry point that dispatches to different modes:
- Interactive TUI (default)
- Exec mode (`spin exec`)
- App server (`spin serve`)
- MCP server (`spin mcp-server`)
- Utility commands (`spin completion`, `spin mcp`, etc.)

### Command Structure

```
spin [GLOBAL_OPTIONS] [COMMAND] [COMMAND_OPTIONS]

Commands:
  (default)        Start interactive TUI
  exec             Non-interactive execution
  serve            Start JSON-RPC app server
  mcp-server       Start MCP server
  mcp              Manage MCP server configurations
  completion       Generate shell completions
  debug            Debug/test subcommands
  config           Manage configuration
  version          Show version information

Global Options:
  --model <MODEL>          Model to use
  --provider <PROVIDER>    Provider (ollama, lmstudio, openai, anthropic)
  --sandbox <MODE>         Sandbox mode
  --cd <DIR>               Change working directory
  -c, --config <KEY=VAL>   Config overrides
  --help                   Show help
  --version                Show version
```

### Implementation

```go
// cmd/spin/main.go
package main

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/yourusername/spin/internal/version"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "spin",
        Short: "AI-powered coding assistant",
        Long:  "Spin is an open-source AI coding assistant compatible with multiple LLM providers",
        RunE:  runTUI, // Default to TUI
    }
    
    // Global flags
    rootCmd.PersistentFlags().String("model", "", "Model to use")
    rootCmd.PersistentFlags().String("provider", "", "Provider")
    rootCmd.PersistentFlags().String("sandbox", "workspace-write", "Sandbox mode")
    rootCmd.PersistentFlags().String("cd", "", "Working directory")
    rootCmd.PersistentFlags().StringSliceP("config", "c", nil, "Config overrides")
    
    // Subcommands
    rootCmd.AddCommand(
        newExecCmd(),
        newServeCmd(),
        newMCPServerCmd(),
        newMCPCmd(),
        newCompletionCmd(),
        newDebugCmd(),
        newConfigCmd(),
        newVersionCmd(),
    )
    
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Special Modes

#### 1. Debug Sandbox

Test sandbox behavior:

```bash
# macOS
spin debug sandbox ls -la

# Linux
spin debug landlock ls -la
```

#### 2. Shell Completions

Generate completions for shells:

```bash
spin completion bash > /etc/bash_completion.d/spin
spin completion zsh > /usr/local/share/zsh/site-functions/_spin
spin completion fish > ~/.config/fish/completions/spin.fish
```

Implementation:
```go
func newCompletionCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "completion [bash|zsh|fish|powershell]",
        Short: "Generate shell completion script",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            switch args[0] {
            case "bash":
                return cmd.Root().GenBashCompletion(os.Stdout)
            case "zsh":
                return cmd.Root().GenZshCompletion(os.Stdout)
            case "fish":
                return cmd.Root().GenFishCompletion(os.Stdout, true)
            case "powershell":
                return cmd.Root().GenPowerShellCompletion(os.Stdout)
            default:
                return fmt.Errorf("unsupported shell: %s", args[0])
            }
        },
    }
    return cmd
}
```

#### 3. MCP Management

```bash
spin mcp add server-name command args...
spin mcp list
spin mcp get server-name
spin mcp remove server-name
```

### Binary Name Detection

**Special Behavior:**

```go
func main() {
    arg0 := os.Args[0]
    
    // Check for special binary names (symlinks)
    switch filepath.Base(arg0) {
    case "spin-apply-patch":
        os.Exit(runApplyPatchMode())
    case "spin-sandbox":
        os.Exit(runSandboxMode())
    }
    
    // Check for special flags
    for _, arg := range os.Args[1:] {
        if arg == "--spin-run-as-apply-patch" {
            os.Exit(runApplyPatchMode())
        }
    }
    
    // Normal CLI execution
    main()
}
```

### Configuration Loading

**Precedence:**
1. Command-line flags (highest)
2. `-c/--config` overrides
3. `~/.config/spin/config.toml`
4. Environment variables (`SPIN_*`)
5. Built-in defaults (lowest)

**Example:**
```bash
# All equivalent ways to set model:
spin --model llama3.1
spin -c model=llama3.1
SPIN_MODEL=llama3.1 spin
# Or in ~/.config/spin/config.toml: model = "llama3.1"
```

**Configuration File:**
```toml
# ~/.config/spin/config.toml

[llm]
provider = "ollama"
model = "llama3.1"
base_url = "http://localhost:11434"

[sandbox]
mode = "workspace-write"

[appearance]
theme = "auto"
no_color = false

[mcp]
servers = [
    { name = "filesystem", command = "npx", args = ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"] },
]
```

### Logging

**Using log/slog (Go 1.21+)**

```go
import "log/slog"

func setupLogging(level string) {
    var logLevel slog.Level
    switch level {
    case "debug":
        logLevel = slog.LevelDebug
    case "info":
        logLevel = slog.LevelInfo
    case "warn":
        logLevel = slog.LevelWarn
    case "error":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }
    
    handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: logLevel,
    })
    slog.SetDefault(slog.New(handler))
}
```

**Environment Variable:** `SPIN_LOG_LEVEL`

**Examples:**
```bash
# Info level for all
SPIN_LOG_LEVEL=info spin

# Debug for detailed logging
SPIN_LOG_LEVEL=debug spin exec "task"

# Quiet mode
SPIN_LOG_LEVEL=error spin exec "task"
```

---

## Cross-Module Comparison

| Feature | TUI | Exec | App Server | MCP Server |
|---------|-----|------|------------|------------|
| Interactive | ✓ | ✗ | ✓ (via client) | ✓ (via client) |
| Streaming Output | ✓ | ✓ | ✓ | ✓ |
| User Approval | ✓ | ✗ (auto) | ✓ | ✓ |
| File Search | ✓ (@) | ✗ | ✓ (API) | ✓ (API) |
| Backtrack | ✓ | ✗ | ✗ | ✗ |
| Suitable For | Daily coding | CI/CD | IDE integration | Agent composition |

---

## Best Practices

### For TUI Users

1. **Use @ for Files:** Faster than typing full paths
2. **Esc-Esc for Iteration:** Try different approaches
3. **Monitor Status Bar:** Check tokens, sandbox mode
4. **Approve Carefully:** Review dangerous commands

### For Exec Users

1. **Be Specific:** Clear, detailed prompts work best
2. **Set Timeouts:** Prevent infinite loops (`--timeout 5m`)
3. **Check Exit Codes:** Integrate into scripts properly
4. **Use Workspace-Write:** Allow file modifications

### For Developers

1. **Test Both Modes:** TUI and exec have different code paths
2. **Handle SIGINT:** Graceful shutdown on Ctrl+C
3. **Stream Output:** Don't buffer large responses
4. **Respect NO_COLOR:** Support color-disabled terminals
5. **Follow Go Conventions:** Use effective Go patterns

---

## Performance Characteristics

### TUI
- **Startup:** ~80ms
- **Input latency:** <5ms
- **Render rate:** 60 FPS
- **Memory:** ~30MB
- **Binary size:** ~15MB (statically linked)

### Exec
- **Startup:** ~40ms (no UI)
- **Overhead:** ~3% vs. direct API
- **Memory:** ~20MB
- **Binary size:** ~12MB

### CLI (dispatch)
- **Startup:** ~5ms (just parsing with cobra)
- **Dispatch:** <1ms
- **Binary size:** ~15MB (includes all subcommands)

---

## Project Structure (Go Standard Layout)

```
spin/
├── cmd/
│   ├── spin/           # Main CLI entry point
│   ├── spin-tui/       # TUI implementation
│   └── spin-exec/      # Exec mode implementation
├── internal/
│   ├── core/           # Core business logic
│   ├── tui/            # TUI shared components
│   ├── filesearch/     # File search functionality
│   ├── config/         # Configuration management
│   └── sandbox/        # Sandbox implementations
├── pkg/
│   └── spin/           # Public API (if needed)
├── go.mod
├── go.sum
├── README.md
└── Makefile
```

---

## Dependencies

### Key Go Libraries

**TUI:**
- `github.com/charmbracelet/bubbletea` - TUI framework (The Elm Architecture)
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - UI components (textinput, viewport, etc.)
- `github.com/charmbracelet/glamour` - Markdown rendering
- `github.com/alecthomas/chroma` - Syntax highlighting

**CLI:**
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management (optional)

**Core:**
- Standard library (net/http, context, io, etc.)
- `golang.org/x/sync/errgroup` - Concurrent error handling
- `golang.org/x/term` - Terminal utilities

**Testing:**
- `github.com/stretchr/testify` - Assertions
- Standard `testing` package

---

## Future Enhancements

### TUI
- [ ] Split panes (code + chat side-by-side)
- [ ] Inline diff viewer using `go-diff`
- [ ] Rich media support (images via sixel/kitty protocol, tables)
- [ ] Plugin system (custom commands via Go plugins or WASM)
- [ ] Themes and customization via config file

### Exec
- [ ] Progress reporting (`--progress` flag)
- [ ] Checkpoint/resume for long tasks
- [ ] Parallel execution (multiple tasks with goroutines)
- [ ] Interactive mode toggle (`--interactive`)

### CLI
- [ ] Config profiles (`--profile dev`)
- [ ] Global undo (`spin undo`)
- [ ] Session management (`spin sessions list`)
- [ ] Built-in tutorials (`spin tutorial`)
- [ ] Self-update command (`spin update`)

---

## Related Documentation

- **core-module.md** - Business logic used by all UIs
- **protocol-modules.md** - Event protocol
- **llm-auth-sdk.md** - LLM provider integration (vendor-agnostic)
- **mcp-modules.md** - MCP support

---

## Conclusion

Spin's UI modules provide flexible interfaces for different use cases:
- **TUI** for interactive development
- **Exec** for automation and CI/CD
- **CLI** as a unified, convenient entry point

By sharing the same core logic (following clean architecture principles), all interfaces provide consistent behavior while optimizing for their specific use cases. The implementation follows Go best practices and is fully open-source with support for multiple LLM providers (Ollama, LMStudio, OpenAI, Anthropic, etc.).


