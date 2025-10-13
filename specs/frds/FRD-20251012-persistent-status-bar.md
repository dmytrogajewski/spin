# FRD: Persistent Status Bar for TUI

**Feature ID:** FRD-20251012-persistent-status-bar
**Created:** 2025-10-12
**Status:** In Development
**Parent Roadmap:** [Advanced Features 2025-10-12](../advanced-features-20251012/ROADMAP.md)

---

## 1. Overview

### 1.1 Summary

Implements a real-time persistent status bar at the bottom of the TUI (between output and prompt) displaying agent state, context usage, provider information, and throughput metrics. The status bar adapts to terminal width and updates based on agent events without disrupting the Factory Droid principle (append-only, native scrollback).

### 1.2 Motivation

Users need visibility into agent operations without disrupting the native terminal experience. Currently, there's no persistent visual feedback showing:
- Agent activity state ("Thinking", "Calling tools", etc.)
- Context fill percentage (risk of overflow)
- Current task mode and provider
- Performance metrics (tokens/sec)
- Session identification

### 1.3 Goals

**Primary:**
- Display real-time metrics between output and prompt
- Adapt to terminal width (3 modes: compact, medium, full)
- Update based on agent events with <10ms latency
- Zero disruption to terminal scrollback

**Secondary:**
- Configurable enable/disable
- Throttled updates to minimize flicker
- Terminal-agnostic (works in SSH, tmux, screen)

### 1.4 Non-Goals

- Full-screen status dashboard
- Interactive status bar (no clickable elements)
- Historical metrics (only current state)
- Multiple status bars or regions

---

## 2. Functional Requirements

### FR-1.1: Status Metrics Display

**Metrics to display:**

1. **Agent State** (priority: critical)
   - Values: "Thinking", "Calling tools", "Planning", "Summarizing", "Loading", "Waiting approval", "Idle"
   - Derived from: Event type (EventTurnStart, EventToolCallStart, EventContentDelta, etc.)
   - Visual: Spinner icon + text

2. **Context Usage** (priority: critical)
   - Format: `42%` or `8.5K/20K (42%)`
   - Derived from: History token count / task mode max tokens
   - Color: Green (<50%), Yellow (50-80%), Red (>80%)

3. **Task Mode** (priority: high)
   - Values: "regular", "review", "compact", "planning"
   - Derived from: Current task mode name
   - Visual: Mode name, color-coded

4. **Provider + Model** (priority: medium)
   - Format: `ollama/qwen3:1.7b` or `openai/gpt-4`
   - Derived from: Config

5. **Tokens/sec** (priority: low)
   - Format: `125 tok/s`
   - Derived from: Time delta between EventContentDelta events
   - Calculation: Rolling average over last 5 deltas

6. **Conversation ID** (priority: low)
   - Format: `conv:abc123` (first 6 chars of UUID)
   - Derived from: Conversation ID

**MUST:** Display (1) and (2) in all modes
**SHOULD:** Display (3), (4) in medium/full modes
**MAY:** Display (5), (6) in full mode only

---

### FR-1.2: Adaptive Rendering

**Width Modes:**

1. **Compact mode** (<60 columns)
   - Display: `[●] 42% Thinking`
   - Metrics: State icon, context %, state text

2. **Medium mode** (60-100 columns)
   - Display: `[●] 42% (8.5K/20K) · regular · ollama/qwen · 125 tok/s`
   - Metrics: Add task mode, provider, tokens/sec

3. **Full mode** (≥100 columns)
   - Display: `[●] 42% (8.5K/20K) · Mode: regular · Provider: ollama/qwen3:1.7b · 125 tok/s · ID: abc123`
   - Metrics: All fields with labels

**MUST:** Detect terminal width via ANSI queries or stored viewport width
**MUST:** Switch modes dynamically on terminal resize
**MUST:** Gracefully degrade for very narrow terminals (<40 cols): hide status bar

---

### FR-1.3: Real-Time Updates

**Update Triggers:**

1. **EventTurnStart**: Update agent state to "Thinking", increment turn counter
2. **EventToolCallStart**: Update agent state to "Calling tools" (or specific tool name)
3. **EventContentDelta**: Update tokens/sec, agent state to "Generating"
4. **EventTurnComplete**: Update agent state to "Idle", finalize turn counter
5. **EventCommandApproval**: Update agent state to "Waiting approval"
6. **EventInfo** (compression): Update agent state to "Summarizing"

**Metrics Calculation:**

- **Context %**: `(history.TokensUsed / taskMode.MaxTokens) * 100`
- **Tokens/sec**: 
  ```go
  // Track last 5 delta timestamps + token counts
  // Calculate: sum(tokens) / sum(time_deltas)
  ```
- **Agent State**: Map event type to human-readable string

**MUST:** Calculate metrics from event data in <1ms
**MUST:** Updates queue if render in progress (prevent concurrent writes)
**SHOULD:** Throttle updates to 100ms minimum interval (configurable)

---

### FR-1.4: Performance

**SLOs:**

| Metric | Target (p99) | Measurement |
|--------|--------------|-------------|
| Render time | <1ms | Time to build ANSI string |
| Update latency | <10ms | Event received → visible on screen |
| Update overhead | <0.1ms | Per-event metric extraction |

**MUST:** Zero allocations in hot path (render loop)
**SHOULD:** Pre-allocate string builders with capacity hints
**MUST:** Thread-safe updates (concurrent event processing)

**Throttling Strategy:**
- Coalesce updates within 100ms window
- Immediate flush on state change (Thinking → Calling tools)
- Defer metrics-only updates (tokens/sec)

---

### FR-1.5: Configuration

**Config options:**

```yaml
ui:
  status_bar:
    enabled: true             # Enable/disable status bar
    compact_width: 60         # Columns threshold for compact mode
    update_interval: 100ms    # Minimum time between updates
    show_spinner: true        # Show animated spinner
```

**MUST:** Respect `enabled` flag (completely hide if false)
**SHOULD:** Validate thresholds (compact_width ∈ [40, 120])
**MUST:** Work without config (sensible defaults)

**Minimal Graceful Terminal:**
- **<40 columns**: Hide status bar entirely
- **40-60 columns**: Compact mode
- **60-100 columns**: Medium mode
- **≥100 columns**: Full mode

---

## 3. Technical Design

### 3.1 Critical Architecture: Sticky Bottom Area

**⚠️ CORE CHALLENGE (18 previous attempts failed here):**

The fundamental problem is creating a **sticky bottom area** where status bar + prompt stay anchored at the bottom while output scrolls above. This requires architectural changes to the output coordination model.

**Current Architecture (Won't Work):**
```
Output appends → Prompt redraws at new position
❌ Problem: Everything scrolls up, no fixed bottom area
```

**Required Architecture (Sticky Bottom):**
```
┌─────────────────────────────────────┐
│ Scrollable Output Area             │← Stops N lines from bottom
│ (grows upward, scrolls)            │
├─────────────────────────────────────┤← Fixed boundary
│ Status Bar (1 line, fixed)         │← Always at terminal_height - 2
├─────────────────────────────────────┤
│ Prompt (1 line, fixed)             │← Always at terminal_height - 1
└─────────────────────────────────────┘
```

**Key Requirements:**
1. **Reserve bottom 2 lines** for sticky area (status + prompt)
2. **Output must stop** before entering sticky area
3. **ANSI absolute positioning** for status bar (save cursor, move to line N-2, render, restore)
4. **No scrolling of sticky area** - it updates in-place
5. **Maintain Factory Droid principle** for output area (append-only above sticky area)

### 3.2 Architecture Overview

**Component Hierarchy:**

```
PureTTY Adapter
    │
    ├── StickyBottomCoordinator ← NEW (CRITICAL)
    │       │
    │       ├── Output Printer (with bottom margin)
    │       ├── StatusBar (absolute positioned)
    │       └── Prompt Renderer (absolute positioned)
    │
    ├── Metrics Aggregator ← NEW
    │
    └── Event Mapper
```

**New Package:** `internal/ui/sticky/`

**Files:**
- `coordinator.go`: StickyBottomCoordinator - manages reserved area
- `statusbar.go`: StatusBar rendering
- `metrics.go`: Metrics aggregation
- `aggregator.go`: Event-to-metrics mapper
- `coordinator_test.go`: Unit tests
- `sticky_integration_test.go`: Integration tests

### 3.3 StickyBottomCoordinator (Core Component)

**This is the architectural breakthrough:**

```go
package sticky

import (
    "io"
    "sync"
    "github.com/dmytrogajewski/spin/internal/ui/output"
    "github.com/dmytrogajewski/spin/internal/ui/prompt"
)

// StickyBottomCoordinator manages a fixed bottom area for status + prompt
// while allowing output to scroll above.
type StickyBottomCoordinator struct {
    out           io.Writer
    printer       *output.Printer
    promptRenderer *prompt.Renderer
    promptModel    prompt.PromptModel
    statusBar     *StatusBar
    
    termHeight    int           // Terminal height in lines
    termWidth     int           // Terminal width in columns
    stickyLines   int           // Number of reserved bottom lines (2)
    outputMargin  int           // Lines from bottom where output stops
    
    mu            sync.Mutex
}

// NewStickyBottomCoordinator creates a coordinator with reserved bottom area.
func NewStickyBottomCoordinator(
    out io.Writer,
    printer *output.Printer,
    renderer *prompt.Renderer,
    model prompt.PromptModel,
    statusBar *StatusBar,
    termHeight, termWidth int,
) *StickyBottomCoordinator {
    return &StickyBottomCoordinator{
        out:            out,
        printer:        printer,
        promptRenderer: renderer,
        promptModel:    model,
        statusBar:      statusBar,
        termHeight:     termHeight,
        termWidth:      termWidth,
        stickyLines:    2,  // Status (1) + Prompt (1)
        outputMargin:   2,  // Keep 2 lines clear at bottom
    }
}

// PrintLine prints a line and updates sticky area.
func (c *StickyBottomCoordinator) PrintLine(s string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // 1. Print output (stops before sticky area)
    if err := c.printer.PrintLine(s); err != nil {
        return err
    }
    
    // 2. Check if we need to scroll to make room for sticky area
    // (This is where the magic happens)
    c.ensureStickyAreaVisible()
    
    // 3. Render sticky area at absolute positions
    return c.renderStickyArea()
}

// ensureStickyAreaVisible ensures output doesn't overflow into sticky area.
func (c *StickyBottomCoordinator) ensureStickyAreaVisible() {
    // Get cursor position
    // If cursor is in bottom N lines, emit newlines to push it up
    // This maintains Factory Droid (append-only) while reserving space
    
    // Emit enough newlines to clear sticky area
    for i := 0; i < c.stickyLines; i++ {
        c.out.Write([]byte("\n"))
    }
}

// renderStickyArea renders status bar + prompt at fixed positions.
func (c *StickyBottomCoordinator) renderStickyArea() error {
    // Save cursor position
    c.out.Write([]byte("\x1b[s"))  // ANSI: Save cursor
    
    // Move to status bar line (terminal_height - 2)
    statusLine := c.termHeight - 1  // 0-indexed
    c.out.Write([]byte(fmt.Sprintf("\x1b[%d;1H", statusLine)))
    
    // Clear line and render status bar
    c.out.Write([]byte("\x1b[2K"))  // ANSI: Clear line
    if c.statusBar != nil {
        c.statusBar.Render(c.out, c.termWidth)
    }
    
    // Move to prompt line (terminal_height - 1)
    promptLine := c.termHeight
    c.out.Write([]byte(fmt.Sprintf("\x1b[%d;1H", promptLine)))
    
    // Clear line and render prompt
    c.out.Write([]byte("\x1b[2K"))  // ANSI: Clear line
    c.promptRenderer.Redraw(c.promptModel, "")
    
    // Restore cursor position
    c.out.Write([]byte("\x1b[u"))  // ANSI: Restore cursor
    
    return nil
}

// OnResize handles terminal resize events.
func (c *StickyBottomCoordinator) OnResize(height, width int) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.termHeight = height
    c.termWidth = width
    
    // Re-render sticky area at new positions
    c.renderStickyArea()
}

// PrintChunks streams chunks and updates sticky area.
func (c *StickyBottomCoordinator) PrintChunks(ctx context.Context, chunks <-chan string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Stream output
    if err := c.printer.PrintChunks(ctx, chunks); err != nil {
        return err
    }
    
    // Ensure sticky area visible
    c.ensureStickyAreaVisible()
    
    // Re-render sticky area
    return c.renderStickyArea()
}
```

**Why This Works:**

1. **Reserve Space**: `ensureStickyAreaVisible()` emits newlines to push content up, keeping bottom N lines clear
2. **Absolute Positioning**: Use ANSI escape sequences to render at fixed line numbers
3. **In-Place Updates**: Status bar updates don't scroll, they overwrite at fixed position
4. **Factory Droid Preserved**: Output area above sticky boundary is still append-only
5. **Terminal Resize**: Recalculate positions and re-render

**This is the architectural breakthrough that previous attempts missed.**

---

### 3.2 Data Model

```go
package statusbar

// StatusBar renders a persistent status bar at the bottom of the TUI.
type StatusBar struct {
    metrics   Metrics
    renderer  *Renderer
    mu        sync.RWMutex
    lastRender time.Time
    throttle  time.Duration
}

// Metrics contains current agent state metrics.
type Metrics struct {
    // Core metrics
    AgentState      string    // "Thinking", "Calling tools", etc.
    ContextUsed     int       // Tokens used
    ContextMax      int       // Max tokens for current task mode
    
    // Metadata
    TaskMode        string    // "regular", "review", "compact", "planning"
    Provider        string    // "ollama", "openai"
    Model           string    // "qwen3:1.7b", "gpt-4"
    
    // Performance
    TokensPerSec    float64   // Rolling average
    
    // Identification
    ConversationID  string    // UUID
    TurnNumber      int       // Current turn
    MaxTurns        int       // Max turns
}

// Renderer handles adaptive rendering based on terminal width.
type Renderer struct {
    width         int
    compactWidth  int  // Threshold for compact mode
    showSpinner   bool
}
```

---

### 3.3 API

```go
// NewStatusBar creates a new status bar.
func NewStatusBar(opts ...Option) *StatusBar

// Option is a functional option for StatusBar.
type Option func(*StatusBar)

// WithThrottle sets the minimum update interval.
func WithThrottle(d time.Duration) Option

// WithCompactWidth sets the compact mode threshold.
func WithCompactWidth(cols int) Option

// Update updates metrics (thread-safe).
func (s *StatusBar) Update(m Metrics) error

// Render renders the status bar to a writer.
// Returns ANSI-escaped string ready for terminal.
func (s *StatusBar) Render(w io.Writer, width int) error

// SetWidth updates the terminal width (handles resize).
func (s *StatusBar) SetWidth(width int)

// Clear clears the status bar from terminal.
func (s *StatusBar) Clear(w io.Writer) error
```

---

### 3.4 Rendering Logic

**Layout Modes:**

```go
func (r *Renderer) Render(m Metrics, width int) (string, error) {
    if width < 40 {
        return "", nil  // Hide completely
    } else if width < r.compactWidth {
        return r.renderCompact(m)
    } else if width < 100 {
        return r.renderMedium(m)
    } else {
        return r.renderFull(m)
    }
}

func (r *Renderer) renderCompact(m Metrics) (string, error) {
    // [●] 42% Thinking
    spinner := r.getSpinner(m.AgentState)
    pct := (m.ContextUsed * 100) / m.ContextMax
    color := r.getContextColor(pct)
    
    return fmt.Sprintf("[%s] %s%d%%\x1b[0m %s",
        spinner, color, pct, m.AgentState), nil
}

func (r *Renderer) renderMedium(m Metrics) (string, error) {
    // [●] 42% (8.5K/20K) · regular · ollama/qwen · 125 tok/s
    compact, _ := r.renderCompact(m)
    
    return fmt.Sprintf("%s (%s/%s) · %s · %s/%s · %.0f tok/s",
        compact,
        humanize.Tokens(m.ContextUsed),
        humanize.Tokens(m.ContextMax),
        m.TaskMode,
        m.Provider,
        m.Model,
        m.TokensPerSec), nil
}

func (r *Renderer) renderFull(m Metrics) (string, error) {
    // [●] 42% (8.5K/20K) · Mode: regular · Provider: ollama/qwen3:1.7b · 125 tok/s · ID: abc123 · Turn: 5/50
    medium, _ := r.renderMedium(m)
    
    convID := m.ConversationID
    if len(convID) > 6 {
        convID = convID[:6]
    }
    
    return fmt.Sprintf("%s · ID: %s · Turn: %d/%d",
        medium, convID, m.TurnNumber, m.MaxTurns), nil
}
```

**Color Mapping:**

```go
func (r *Renderer) getContextColor(pct int) string {
    if pct < 50 {
        return ansi.Green
    } else if pct < 80 {
        return ansi.Yellow
    } else {
        return ansi.Red
    }
}
```

---

### 3.5 Metrics Aggregation

**Aggregator Interface:**

```go
// Aggregator processes events and updates metrics.
type Aggregator struct {
    metrics       Metrics
    mu            sync.RWMutex
    tokenDeltas   []tokenDelta
    maxDeltas     int
}

type tokenDelta struct {
    tokens    int
    timestamp time.Time
}

// ProcessEvent updates metrics based on event type.
func (a *Aggregator) ProcessEvent(event core.Event) {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    switch event.Type {
    case core.EventTurnStart:
        data := event.Data.(core.TurnEventData)
        a.metrics.AgentState = "Thinking"
        a.metrics.TurnNumber = data.TurnsUsed + 1
        a.metrics.MaxTurns = data.MaxTurns
        
    case core.EventToolCallStart:
        data := event.Data.(core.ToolCallStartData)
        a.metrics.AgentState = fmt.Sprintf("Calling %s", data.ToolName)
        
    case core.EventContentDelta:
        data := event.Data.(core.ContentDeltaData)
        a.metrics.AgentState = "Generating"
        a.trackTokenDelta(len(data.Content), event.Timestamp)
        a.recalculateTokensPerSec()
        
    case core.EventTurnComplete:
        a.metrics.AgentState = "Idle"
        
    case core.EventCommandApproval:
        a.metrics.AgentState = "Waiting approval"
        
    case core.EventInfo:
        data := event.Data.(core.SystemEventData)
        if strings.Contains(data.Message, "compressed") {
            a.metrics.AgentState = "Summarizing"
        }
    }
}

// GetMetrics returns current metrics (thread-safe).
func (a *Aggregator) GetMetrics() Metrics {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.metrics
}

// SetContextInfo updates context metrics.
func (a *Aggregator) SetContextInfo(used, max int) {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.metrics.ContextUsed = used
    a.metrics.ContextMax = max
}

// SetProviderInfo updates provider metadata.
func (a *Aggregator) SetProviderInfo(provider, model string) {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.metrics.Provider = provider
    a.metrics.Model = model
}

func (a *Aggregator) trackTokenDelta(tokens int, ts time.Time) {
    a.tokenDeltas = append(a.tokenDeltas, tokenDelta{tokens, ts})
    if len(a.tokenDeltas) > a.maxDeltas {
        a.tokenDeltas = a.tokenDeltas[1:]
    }
}

func (a *Aggregator) recalculateTokensPerSec() {
    if len(a.tokenDeltas) < 2 {
        a.metrics.TokensPerSec = 0
        return
    }
    
    totalTokens := 0
    for _, d := range a.tokenDeltas {
        totalTokens += d.tokens
    }
    
    duration := a.tokenDeltas[len(a.tokenDeltas)-1].timestamp.Sub(
        a.tokenDeltas[0].timestamp)
    
    if duration.Seconds() == 0 {
        return
    }
    
    a.metrics.TokensPerSec = float64(totalTokens) / duration.Seconds()
}
```

---

### 3.6 Integration with Coordinator

**Modified `CoordinatedWriter`:**

```go
type CoordinatedWriter struct {
    printer    *output.Printer
    renderer   *prompt.Renderer
    model      prompt.PromptModel
    statusBar  *statusbar.StatusBar  // NEW
    mu         sync.Mutex
}

// Sequence: Output → StatusBar → Prompt
func (c *CoordinatedWriter) PrintLine(s string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // 1. Print output
    if err := c.printer.PrintLine(s); err != nil {
        return err
    }
    
    // 2. Render status bar (if enabled)
    if c.statusBar != nil {
        if err := c.statusBar.Render(c.printer.Writer(), c.getWidth()); err != nil {
            return err
        }
    }
    
    // 3. Redraw prompt
    return c.renderer.Redraw(c.model, c.status)
}
```

---

### 3.7 Configuration Integration

**Config struct update:**

```go
// internal/config/loader.go
type UIConfig struct {
    StatusBar StatusBarConfig `yaml:"status_bar"`
}

type StatusBarConfig struct {
    Enabled        bool          `yaml:"enabled"`
    CompactWidth   int           `yaml:"compact_width"`
    UpdateInterval time.Duration `yaml:"update_interval"`
    ShowSpinner    bool          `yaml:"show_spinner"`
}

// Defaults
var DefaultStatusBarConfig = StatusBarConfig{
    Enabled:        true,
    CompactWidth:   60,
    UpdateInterval: 100 * time.Millisecond,
    ShowSpinner:    true,
}
```

---

## 4. Testing Strategy

### 4.1 Unit Tests

**File:** `statusbar_test.go`

**Test cases:**

1. **TestStatusBar_UpdateMetrics**
   - Update metrics and verify thread-safety
   - Concurrent updates from multiple goroutines

2. **TestStatusBar_RenderCompact**
   - Render with width=50, verify compact format
   - Check spinner, percentage, state text

3. **TestStatusBar_RenderMedium**
   - Render with width=80, verify medium format
   - Check all medium mode fields

4. **TestStatusBar_RenderFull**
   - Render with width=120, verify full format
   - Check all fields including conversation ID

5. **TestStatusBar_Throttling**
   - Rapid updates within throttle window
   - Verify only one render per interval

6. **TestStatusBar_Clear**
   - Render status bar, then clear
   - Verify ANSI clear sequence

7. **TestStatusBar_WidthResize**
   - Change width from 120 → 50
   - Verify mode switch

---

### 4.2 Integration Tests

**File:** `statusbar_integration_test.go`

**Test cases:**

1. **TestStatusBar_WithEventEmitter**
   - Subscribe to event stream
   - Send events, verify metrics update

2. **TestStatusBar_WithCoordinator**
   - PrintLine with status bar enabled
   - Verify sequence: output → status → prompt

3. **TestStatusBar_ContextOverflow**
   - Set context to 95% full
   - Verify red color in render

4. **TestStatusBar_TokensPerSec**
   - Send 5 ContentDelta events with delays
   - Verify tokens/sec calculation

---

### 4.3 E2E Tests

**File:** `e2e/statusbar_test.go`

**Test cases:**

1. **TestE2E_StatusBar_FullConversation**
   - Start TUI with status bar enabled
   - Send multi-turn conversation
   - Verify status bar updates throughout

2. **TestE2E_StatusBar_TerminalResize**
   - Start with 120-col terminal
   - Resize to 50 cols mid-conversation
   - Verify status bar adapts

3. **TestE2E_StatusBar_SSH**
   - Run in SSH session (PTY)
   - Verify status bar renders correctly

---

### 4.4 Performance Tests

**File:** `statusbar_bench_test.go`

**Benchmarks:**

```go
func BenchmarkStatusBar_Render(b *testing.B)
func BenchmarkStatusBar_Update(b *testing.B)
func BenchmarkAggregator_ProcessEvent(b *testing.B)
```

**Acceptance Criteria:**
- Render: <1ms (p99)
- Update: <0.1ms (p99)
- ProcessEvent: <0.05ms (p99)

---

## 5. Acceptance Criteria

### 5.1 Functional

- [ ] Status bar displays between output and prompt
- [ ] Shows agent state and context % in all modes
- [ ] Adapts to terminal width (compact/medium/full)
- [ ] Updates in real-time on events
- [ ] Hides for terminals <40 columns
- [ ] Respects `enabled` config flag
- [ ] Thread-safe concurrent updates

### 5.2 Performance

- [ ] Render time: <1ms (p99)
- [ ] Update latency: <10ms (p99)
- [ ] Zero race conditions (`-race` detector clean)
- [ ] Throttling prevents excessive updates

### 5.3 Visual Quality

- [ ] No scrollback disruption (Factory Droid principle)
- [ ] No tearing (atomic render)
- [ ] Correct ANSI coloring
- [ ] Spinner animates smoothly (if enabled)
- [ ] Truncation handled gracefully

---

## 6. Implementation Plan

**⚠️ CRITICAL PATH: Sticky Bottom Area First**

Previous attempts failed because they tried to add status bar without fixing the foundation. We must build the sticky area infrastructure first.

### Phase 1: Sticky Bottom Coordinator (CRITICAL - Day 1-2)

**Priority: P0 - Foundation for everything else**

- [ ] Create `internal/ui/sticky/` package
- [ ] Implement `StickyBottomCoordinator` core:
  - [ ] `NewStickyBottomCoordinator()` constructor
  - [ ] `ensureStickyAreaVisible()` - reserves bottom lines
  - [ ] `renderStickyArea()` - absolute ANSI positioning
  - [ ] `OnResize()` - terminal resize handling
  - [ ] `PrintLine()` - output + sticky area update
  - [ ] `PrintChunks()` - streaming + sticky area update
- [ ] Write unit tests:
  - [ ] Test sticky area rendering at correct positions
  - [ ] Test output stops before sticky boundary
  - [ ] Test terminal resize updates positions
  - [ ] Test concurrent access (thread safety)
- [ ] Visual validation in real terminal:
  - [ ] Run in kitty, alacritty, iTerm2
  - [ ] Test scrollback preservation
  - [ ] Test with SSH, tmux, screen
- [ ] Coverage: 90%+

**Acceptance:** Prompt stays at bottom, output scrolls above without disrupting it

### Phase 2: StatusBar Rendering (Day 2-3)

**Depends on: Phase 1 complete**

- [ ] Create `statusbar.go` in `internal/ui/sticky/`
- [ ] Implement `StatusBar`, `Metrics`, `Renderer` structs
- [ ] Implement rendering modes:
  - [ ] `renderCompact()` - <60 cols
  - [ ] `renderMedium()` - 60-100 cols
  - [ ] `renderFull()` - ≥100 cols
- [ ] Add color mapping, spinner logic
- [ ] Write unit tests for rendering
- [ ] Coverage: 90%+

### Phase 3: Metrics Aggregation (Day 3)

- [ ] Create `metrics.go` and `aggregator.go`
- [ ] Implement `Aggregator`:
  - [ ] `ProcessEvent()` - event → metrics
  - [ ] `SetContextInfo()` - update context metrics
  - [ ] `SetProviderInfo()` - update provider metadata
  - [ ] Token/sec calculation with rolling average
- [ ] Write unit tests for aggregator
- [ ] Coverage: 90%+

### Phase 4: Integration with PureTTY (Day 4)

- [ ] Update `internal/ui/adapters/puretty.go`:
  - [ ] Replace `CoordinatedWriter` with `StickyBottomCoordinator`
  - [ ] Wire event stream to `Aggregator`
  - [ ] Pass terminal dimensions on init and resize
- [ ] Add configuration support (`config.yaml`)
- [ ] Write integration tests
- [ ] Coverage: 85%+

### Phase 5: E2E Testing & Polish (Day 5)

- [ ] Write E2E tests:
  - [ ] Full conversation with status bar updates
  - [ ] Terminal resize mid-conversation
  - [ ] SSH/tmux compatibility
- [ ] Performance benchmarking:
  - [ ] Render time <1ms (p99)
  - [ ] Update latency <10ms (p99)
- [ ] Visual testing across terminals
- [ ] Documentation updates

### Phase 6: Analysis & Merge (Day 5-6)

- [ ] Run `uast parse | herr analyze` on all files
- [ ] Fix complexity/dead code issues
- [ ] Run `make lint` - zero errors
- [ ] Run tests with `-race` - zero races
- [ ] Code review
- [ ] Merge to main

---

## 7. Rollout Plan

### 7.1 Feature Flag

```yaml
ui:
  status_bar:
    enabled: false  # Disabled by default initially
```

### 7.2 Gradual Rollout

1. **Week 1**: Internal testing, status bar disabled by default
2. **Week 2**: Enable for beta users (`enabled: true` in their configs)
3. **Week 3**: Enable by default for all users

### 7.3 Rollback Plan

If issues arise:
1. Set `enabled: false` in default config
2. Fix issues
3. Re-enable

---

## 8. Documentation Updates

### 8.1 User Documentation

**File:** `docs/tui.md`

Add section:

```markdown
### Status Bar

The status bar displays real-time agent metrics at the bottom of the TUI:
- Agent state (Thinking, Calling tools, etc.)
- Context usage percentage
- Task mode, provider, model
- Tokens per second
- Conversation ID and turn number

**Configuration:**
```yaml
ui:
  status_bar:
    enabled: true
    compact_width: 60
    update_interval: 100ms
```

**Width Modes:**
- Compact (<60 cols): `[●] 42% Thinking`
- Medium (60-100 cols): `[●] 42% (8.5K/20K) · regular · ollama/qwen · 125 tok/s`
- Full (≥100 cols): All fields with labels
```

### 8.2 Developer Documentation

**File:** `docs/packages/ui-statusbar.md` (new)

Document:
- Package overview
- API reference
- Integration guide
- Performance characteristics

---

## 9. Dependencies

**Internal:**
- `internal/ui/output`: Printer integration
- `internal/ui/prompt`: Prompt redraw coordination
- `internal/core`: Event types and payloads
- `internal/config`: Configuration loading
- `pkg/ansi`: ANSI escape sequences

**External:**
- None (uses standard library only)

---

## 10. Risks & Mitigations

### Risk 1: Performance Regression

**Probability:** Low
**Impact:** Medium

**Mitigation:**
- Throttle updates to 100ms
- Pre-allocate string builders
- Benchmark before merge

### Risk 2: Terminal Compatibility

**Probability:** Medium
**Impact:** Medium

**Mitigation:**
- Test on multiple terminals (kitty, alacritty, iTerm2, xterm)
- Test in SSH, tmux, screen
- Graceful degradation for narrow terminals

### Risk 3: Visual Disruption

**Probability:** Low
**Impact:** High

**Mitigation:**
- Atomic render (lock during output → status → prompt)
- Use ANSI cursor positioning (not clear screen)
- Extensive visual testing

---

## 11. Future Enhancements

### Phase 2 (Future):
- Configurable metrics display (user chooses which fields)
- Click-to-expand for detailed metrics
- Historical metrics graph (last 10 turns)
- Custom colors/themes

---

## 12. References

- **Roadmap:** [Advanced Features 2025-10-12](../advanced-features-20251012/ROADMAP.md) - Feature 1
- **Research:** [RESEARCH.md](../advanced-features-20251012/RESEARCH.md) - Feature 1
- **TUI Spec:** [docs/tui.md](../../docs/tui.md)
- **Performance:** [docs/performance.md](../../docs/performance.md)
- **AGENTS.md:** [AGENTS.md](../../AGENTS.md) - Development workflow

---

**End of FRD**

*This document follows SOLID, DRY, KISS principles and Go 1.24 standards.*

