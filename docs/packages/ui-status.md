# Package: internal/ui/status

**Purpose**: Persistent status bar for real-time agent metrics display  
**Status**: ✅ Complete  
**Coverage**: 95%+  
**Created**: 2025-10-14

---

## Overview

The `internal/ui/status` package provides a persistent status bar that displays real-time agent metrics at the bottom of the terminal without disrupting scrollback. It implements adaptive layout, event-driven updates, and comprehensive metrics tracking.

---

## Components

### Manager (`manager.go`)
**Purpose**: Thread-safe status data management

**Key Types:**
```go
type Metrics struct {
    // Conversation metrics
    TurnCount      int
    TokenCount     int64
    MaxTokens      int64
    TokenUsage     float64 // Percentage (0-100)
    
    // Performance metrics
    ResponseTime   time.Duration
    TokensPerSec   float64
    
    // Connection metrics
    Provider       string
    Model          string
    Connected      bool
    
    // Agent state
    AgentState     string // "Thinking", "Calling tools", etc.
    TaskMode       string // regular, review, compact, planning
    ConversationID string // Session identifier
    
    // Timestamps
    LastUpdate     time.Time
    SessionStart   time.Time
}

type Manager struct {
    // Thread-safe metrics storage and formatting
}
```

**Key Methods:**
- `SetAgentState(state string)` - Update current agent activity
- `SetTaskMode(mode string)` - Set task mode
- `SetConversationID(id string)` - Set session ID
- `AddTokens(prompt, completion int64)` - Track token usage
- `CalculateTPS(tokens int64, duration time.Duration)` - Calculate throughput
- `FormatCompact(width int) string` - Format for narrow terminals
- `FormatMedium(width int) string` - Format for medium terminals
- `FormatFull(width int) string` - Format for wide terminals
- `FormatAdaptive(width int) string` - Auto-select best format

**Thread Safety**: All methods are thread-safe using RWMutex

---

### Aggregator (`aggregator.go`)
**Purpose**: Process core events and update status metrics

**Key Functionality:**
- Maps `core.Event` types to agent states
- Updates metrics in real-time
- Calculates tokens per second during streaming
- Increments turn count on turn start

**Event Mapping:**
| Core Event | Agent State |
|------------|-------------|
| `EventTurnStart` | "Starting" |
| `EventContentDelta` | "Thinking" |
| `EventToolCallStart` | "Calling: {tool_name}" |
| `EventToolCallProgress` | "Executing" |
| `EventToolCallComplete` | "Complete" |
| `EventTurnComplete` | "Ready" |
| `EventCommandApproval` | "Waiting approval" |
| `EventError` | "Error" |
| `EventWarning` | "Warning" |

**Usage:**
```go
manager := status.NewManager()
aggregator := status.NewAggregator(manager)

// Process events
aggregator.ProcessEvent(&event)
```

---

### Formatter (`formatter.go`)
**Purpose**: Adaptive status bar formatting based on terminal width

**Formatting Functions:**
- `FormatCompact(width)` - Minimal display for narrow terminals
- `FormatMedium(width)` - Balanced display for medium terminals
- `FormatFull(width)` - Complete display for wide terminals
- `FormatAdaptive(width)` - Auto-select based on terminal width

**Helper Functions:**
- `humanizeNumber(n int64) string` - Format numbers with K/M suffixes
- `formatPercentage(pct float64) string` - Format percentages
- `truncate(s string, maxLen int) string` - Truncate with "..."
- `capitalize(s string) string` - Capitalize first letter
- `activityIndicator(connected bool) string` - Activity indicator

**Layout Examples:**

**Compact (<60 cols):**
```
[●] 42% Thinking
```

**Medium (60-100 cols):**
```
[●] 42% Thinking  ollama/qwen  125tok/s
```

**Full (≥100 cols):**
```
[●] 42% (8.5K/20K)  Thinking  ollama/qwen3:1.7b  125 tok/s  conv:abc123
```

**Extra Wide (≥120 cols):**
```
[●] 42% (8.5K/20K)  Thinking  ollama/qwen3:1.7b  125 tok/s  conv:abc123  ?:help ^C:quit
```

---

### Renderer (`renderer.go`)
**Purpose**: ANSI-based status bar rendering with scrolling region management

**Key Features:**
- Sets up ANSI scrolling region to reserve bottom 2 lines
- Positions status bar at line (height-1)
- Positions prompt at line (height)
- Saves/restores cursor position to avoid disrupting content
- Provides `MoveToScrollRegion()` for cursor management

**ANSI Sequences Used:**
- `\x1b[1;Nr` - Set scrolling region (lines 1 to N)
- `\x1b[line;1H` - Absolute cursor positioning
- `\x1b[2K` - Clear line
- `\x1b7` - Save cursor position
- `\x1b8` - Restore cursor position
- `\x1b[r` - Reset scrolling region (on exit)

**Usage:**
```go
renderer := status.NewRenderer(os.Stdout, width, height)
renderer.Render("Status text here")
renderer.MoveToScrollRegion() // Move cursor back to content area
```

---

## Integration

### With PureTTY Adapter

The status bar integrates seamlessly with the PureTTY adapter:

```go
// In internal/ui/adapters/puretty.go

// Initialization
statusManager := status.NewManager()
statusAggregator := status.NewAggregator(statusManager)
statusRenderer := status.NewRenderer(out, width, height)

// Set initial metadata
statusManager.SetTaskMode(taskMode)
statusManager.SetConversationID(sessionID)
statusManager.SetProvider(provider, model)

// Process events
func (u *PureTTY) ProcessEvent(event *core.Event) {
    statusAggregator.ProcessEvent(event)
    updateStatusBar()
}

// Update display
func (u *PureTTY) updateStatusBar() {
    w, _ := u.tty.Size()
    formatted := statusManager.FormatAdaptive(w)
    statusRenderer.Render(formatted)
}
```

### With Core Events

Events automatically trigger status updates:

```go
// Content generation
EventContentDelta → AgentState: "Thinking"

// Tool execution
EventToolCallStart → AgentState: "Calling: read_file"

// Turn completion
EventTurnComplete → AgentState: "Ready"
```

---

## Configuration

**Current**: Status bar is always enabled  
**Future**: Configuration options

```yaml
# Future configuration
ui:
  status_bar:
    enabled: true
    show_border: false
    show_hotkeys: true
    show_conversation_id: true
    update_interval_ms: 100
    compact_width: 60
    medium_width: 100
```

---

## Performance

**Benchmarks:**
- Status bar render: **<5ms** (typically <1ms)
- Format adaptive: **<1ms**
- Event processing: **<0.1ms**
- Thread-safe updates: **No lock contention** (RWMutex)

**Update Frequency:**
- Event-driven (no polling)
- Updates only when status changes
- Max 10Hz during rapid streaming (100ms throttle)

---

## Testing

**Test Coverage**: 95%+

**Test Files:**
- `manager_test.go` - Status manager unit tests (16 tests)
- `aggregator_test.go` - Event processing tests (4 tests)
- `formatter_test.go` - Formatting function tests (14 tests)
- `renderer_test.go` - ANSI rendering tests (4 tests)
- `integration_test.go` - End-to-end flow tests (5 tests)

**Test with:**
```bash
go test ./internal/ui/status -v
go test -race ./internal/ui/status
go test -bench=. ./internal/ui/status
```

---

## Examples

### Basic Usage

```go
// Create manager
manager := status.NewManager()

// Set initial data
manager.SetProvider("ollama", "qwen3:1.7b")
manager.SetTaskMode("review")
manager.SetMaxTokens(16384)

// Update during operation
manager.SetAgentState("Thinking")
manager.AddTokens(100, 50)
manager.CalculateTPS(125, 1*time.Second)

// Get formatted output
formatted := manager.FormatAdaptive(120)
// Output: "[●] 1% (150/16.4K)  Thinking  Review  ollama/qwen3:1.7b  125 tok/s  ?:help ^C:quit"
```

### With Event Processing

```go
// Create aggregator
aggregator := status.NewAggregator(manager)

// Process events
for event := range events {
    aggregator.ProcessEvent(&event)
    
    // Get updated status
    formatted := manager.FormatAdaptive(terminalWidth)
    renderer.Render(formatted)
}
```

---

## Files

| File | Lines | Purpose |
|------|-------|---------|
| `manager.go` | ~260 | Status data management |
| `aggregator.go` | ~90 | Event processing |
| `formatter.go` | ~180 | Adaptive formatting |
| `renderer.go` | ~120 | ANSI rendering |
| `integration_test.go` | ~110 | Integration tests |
| **Total** | **~760** | **Complete package** |

---

## Dependencies

**Internal:**
- `internal/core` - Core events and types
- `internal/ui/output` - Coordinator integration
- `internal/ui/prompt` - Prompt renderer

**External:**
- None (pure Go stdlib)

---

## Future Enhancements

- [ ] Animated activity spinner
- [ ] Color schemes configuration
- [ ] Clickable conversation ID (copy to clipboard)
- [ ] Status history graph (mini sparkline)
- [ ] Network latency indicator
- [ ] Cost tracking ($ per request)
- [ ] Custom field visibility configuration

---

## Troubleshooting

**Q: Status bar not showing**
- Check terminal height ≥3 lines
- Verify `StatusManager.IsEnabled()` returns true
- Check `FormatAdaptive()` returns non-empty string

**Q: Status updates too frequent**
- Updates are throttled to changed values only
- Check `lastStatusText` tracking in PureTTY

**Q: Layout doesn't fit terminal**
- Adaptive layout automatically adjusts
- Test with different widths: 40, 60, 80, 120, 200 columns

---

**Last Updated**: 2025-10-14  
**Status**: ✅ Production Ready  
**Next**: Feature 2 (Context Summarization)

