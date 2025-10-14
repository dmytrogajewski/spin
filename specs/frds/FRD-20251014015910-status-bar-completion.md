# FRD: Status Bar Completion

**Feature ID**: FRD-20251014015910  
**Title**: Complete Persistent Status Bar Implementation  
**Status**: Draft  
**Created**: 2025-10-14  
**Author**: AI Implementation Agent  
**Related**: specs/advanced-features-20251012/RESEARCH.md, specs/advanced-features-20251012/ROADMAP.md

---

## 1. Executive Summary

Complete the persistent status bar feature by implementing full metrics display, adaptive layout, and proper integration with core agent state. Infrastructure (sticky bottom area, scrolling regions) is complete. This FRD covers the remaining display and data integration work.

**Current State**: 🟡 Infrastructure complete, basic status text working  
**Target State**: ✅ Full status bar with all required metrics  
**Estimated Effort**: 4-6 hours  
**Priority**: HIGH (completes Feature 1 from roadmap)

---

## 2. Problem Statement

### 2.1 What's Working
- ✅ ANSI scrolling regions reserve bottom 2 lines
- ✅ Cursor management keeps content in scrolling area
- ✅ Basic status text displays ("Ready", "Executing tool...")
- ✅ Event-driven updates via `StatusAggregator`
- ✅ Data structures exist for all metrics

### 2.2 What's Missing
The current implementation only shows simple status messages. Required comprehensive metrics bar is not implemented:

**Missing Features**:
1. **Context fill percentage**: E.g., "42% (8.5K/20K tokens)"
2. **Agent activity state**: User-friendly state names ("Calling tools", "Summarizing", "Planning")
3. **Task mode**: Current mode display (regular/review/compact/planning)
4. **Provider & model**: E.g., "ollama/qwen3:1.7b"
5. **Tokens per second**: Real-time throughput, e.g., "125 tok/s"
6. **Conversation ID**: Session identifier for reference
7. **Hotkey information**: Quick reference, e.g., "?:help ^C:quit"
8. **Adaptive layout**: Different displays for narrow/medium/wide terminals
9. **Visual formatting**: Borders, proper spacing, colors

### 2.3 Required Layout (from Research)
```
┌────────────────────────────────────────────────────────────────────┐
│ [●] 42%  Planning  ollama/qwen3:1.7b  125 tok/s  conv:abc123  ?:help │
└────────────────────────────────────────────────────────────────────┘
> _
```

**Components**:
- `[●]` - Activity indicator (animated during processing)
- `42%` - Context usage percentage
- `Planning` - Current agent state
- `ollama/qwen3:1.7b` - Provider and model
- `125 tok/s` - Tokens per second throughput
- `conv:abc123` - Conversation ID (shortened)
- `?:help` - Hotkey hint

---

## 3. Requirements

### 3.1 Functional Requirements

#### FR-1: Context Usage Display
**Priority**: MUST HAVE

Display context usage as percentage and absolute values:
- Format: `{percentage}% ({used}/{max} tokens)`
- Example: `42% (8.5K/20K)`
- Update: Real-time as tokens are consumed
- Source: `StatusManager.Metrics.TokenCount` and `MaxTokens`
- Edge cases:
  - When MaxTokens is 0: Show "N/A"
  - When usage > 80%: Yellow color
  - When usage > 95%: Red color
  - Round to 1 decimal place for percentage
  - Use K/M suffix for large numbers (8500 → 8.5K)

#### FR-2: Agent Activity State
**Priority**: MUST HAVE

Display human-readable agent state:
- States to map:
  - `EventTurnStart` → "Starting..."
  - `EventToolCallStart` → "Calling tools"
  - `EventToolCallProgress` → "Executing: {tool_name}"
  - `EventContentDelta` → "Thinking"
  - `EventContentComplete` → "Complete"
  - `EventCommandApproval` → "Waiting approval"
  - Default/idle → "Ready"
- Max length: 20 characters (truncate with "...")
- Update: On every core event
- Color: Dim when idle, bright when active

#### FR-3: Task Mode Display
**Priority**: MUST HAVE

Display current task mode:
- Values: "regular", "review", "compact", "planning"
- Format: Capitalize first letter
- Source: Agent config or runtime state
- Static (doesn't change during conversation)
- Omit if default (regular mode)

#### FR-4: Provider & Model Display
**Priority**: MUST HAVE

Display LLM provider and model:
- Format: `{provider}/{model}`
- Example: `ollama/qwen3:1.7b`
- Truncate model name if > 25 chars
- Source: `StatusManager.Metrics.Provider` and `Model`
- Update: On provider initialization
- Static during conversation

#### FR-5: Tokens Per Second
**Priority**: SHOULD HAVE

Display real-time token generation throughput:
- Format: `{tps} tok/s`
- Example: `125 tok/s`
- Round to integer
- Update: During `EventContentDelta` streaming
- Calculate: Track token count and time delta between events
- Show only when actively generating (hide when idle)

#### FR-6: Conversation ID
**Priority**: SHOULD HAVE

Display session/conversation identifier:
- Format: `conv:{short_id}`
- Example: `conv:abc123` (first 6 chars of UUID)
- Source: Session ID from `internal/session`
- Static during conversation
- Clickable (future): Copy full ID on interaction

#### FR-7: Hotkey Information
**Priority**: SHOULD HAVE

Display quick reference to key shortcuts:
- Format: `?:help ^C:quit ^P:palette`
- Show most important keys only
- Omit on narrow terminals (<80 cols)
- Non-interactive (just display text)

#### FR-8: Adaptive Layout
**Priority**: MUST HAVE

Adjust displayed information based on terminal width:

**Compact (< 60 columns)**:
```
[●] 42% Thinking
```
- Activity indicator
- Context percentage only
- Agent state

**Medium (60-100 columns)**:
```
[●] 42% Thinking  ollama/qwen  125tok/s
```
- Add provider/model (truncated)
- Add tokens/sec

**Full (≥100 columns)**:
```
[●] 42% (8.5K/20K)  Planning  ollama/qwen3:1.7b  125tok/s  conv:abc123  ?:help
```
- All fields
- Full formatting with spacing
- Hotkeys

**Implementation**: Use terminal width from `internal/ui/term` and format accordingly.

#### FR-9: Visual Formatting
**Priority**: SHOULD HAVE

- **Border characters**: Use Unicode box-drawing characters (┌─┐│└┘)
- **Colors**: ANSI 256-color support
  - Activity indicator: Green (active), dim gray (idle)
  - Context %: Green (<80%), yellow (80-95%), red (>95%)
  - Agent state: Bright white
  - Static fields: Dim gray
- **Spacing**: Proper padding between fields (2-3 spaces)
- **Alignment**: Left-aligned within status bar

---

### 3.2 Non-Functional Requirements

#### NFR-1: Performance
- Status bar render: <5ms
- No blocking operations
- Event processing: <1ms per event
- Update frequency: Max 10Hz (100ms minimum interval)

#### NFR-2: Thread Safety
- All `StatusManager` methods thread-safe (already implemented)
- No race conditions in formatting or rendering
- Test with `go test -race`

#### NFR-3: Testability
- Unit tests for all formatting functions
- Integration tests for event → display flow
- Test all terminal width scenarios
- Mock terminal for deterministic tests

#### NFR-4: Maintainability
- Clean separation: data (Manager) → formatting (FormatFull) → rendering (Renderer)
- Well-documented formatting functions
- Easy to add new fields
- Configuration-driven field visibility

---

## 4. Design

### 4.1 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Core Agent                             │
│                   (emits events)                            │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│                   StatusAggregator                          │
│        (processes events, updates Manager)                  │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│                   StatusManager                             │
│              (holds metrics data)                           │
│  • Context: used, max, percentage                           │
│  • State: current agent activity                            │
│  • Provider: name, model                                    │
│  • Performance: TPS, response time                          │
│  • Session: conversation ID                                 │
│  • Config: task mode                                        │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼ (format)
┌─────────────────────────────────────────────────────────────┐
│              Formatter (new methods)                        │
│  • FormatCompact() → minimal string                         │
│  • FormatMedium() → medium string                           │
│  • FormatFull() → full string                               │
│  • FormatAdaptive(width) → auto-select                      │
└────────────┬────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│                  StatusRenderer                             │
│           (renders to ANSI output)                          │
│  • Draws border (optional)                                  │
│  • Applies colors                                           │
│  • Positions at line (height-1)                             │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Component Modifications

#### StatusManager (internal/ui/status/manager.go)
**Additions**:
```go
// Extended Metrics struct
type Metrics struct {
    // ... existing fields ...
    
    // New fields
    AgentState     string    // Current agent activity
    TaskMode       string    // regular, review, compact, planning
    ConversationID string    // Session ID
}

// New formatting methods
func (m *Manager) FormatMedium(width int) string
func (m *Manager) FormatFull(width int) string
func (m *Manager) FormatAdaptive(width int) string

// Helper methods
func (m *Manager) SetAgentState(state string)
func (m *Manager) SetTaskMode(mode string)
func (m *Manager) SetConversationID(id string)
func (m *Manager) CalculateTPS(tokens int64, duration time.Duration)
```

#### StatusAggregator (internal/ui/status/aggregator.go)
**Enhancements**:
```go
// Add state mapping
func (a *Aggregator) ProcessEvent(event *core.Event) {
    // ... existing code ...
    
    // Map events to user-friendly states
    switch event.Type {
    case core.EventTurnStart:
        a.manager.SetAgentState("Starting...")
    case core.EventToolCallStart:
        if data, ok := event.Data.(core.ToolCallStartData); ok {
            a.manager.SetAgentState("Calling: " + data.ToolName)
        }
    case core.EventContentDelta:
        a.manager.SetAgentState("Thinking")
        // Calculate TPS
        // ...
    // ... etc ...
    }
}

// Add TPS calculation
func (a *Aggregator) calculateTPS(event *core.Event)
```

#### StatusRenderer (internal/ui/status/renderer.go)
**Enhancements**:
```go
// Modified Render to accept formatted string and apply styling
func (r *Renderer) Render(formattedStatus string, style RenderStyle) error

type RenderStyle struct {
    ShowBorder    bool
    ColorScheme   ColorScheme
    ActivityColor string
}

type ColorScheme struct {
    Activity    string // ANSI color code
    ContextLow  string // <80%
    ContextMed  string // 80-95%
    ContextHigh string // >95%
    Static      string // Provider, ID, etc.
}
```

### 4.3 Data Flow

```
1. Core Event (e.g., EventToolCallStart)
     ↓
2. StatusAggregator.ProcessEvent()
     ↓
3. Update StatusManager metrics
   - SetAgentState("Calling: read_file")
   - UpdateMetrics (tokens, TPS)
     ↓
4. PureTTY.updateStatusBar()
   - Get terminal width
   - Call manager.FormatAdaptive(width)
     ↓
5. StatusRenderer.Render(formattedString)
   - Add colors
   - Position at line (height-1)
   - Write to stdout
```

### 4.4 Configuration

Add to `configs/example.yaml`:
```yaml
ui:
  status_bar:
    enabled: true
    show_border: false  # Optional border around status
    show_hotkeys: true  # Show hotkey hints
    show_conversation_id: true  # Show session ID
    update_interval_ms: 100  # Min time between updates
    
    # Adaptive breakpoints
    compact_width: 60
    medium_width: 100
    
    # Color scheme (ANSI 256-color codes)
    colors:
      activity_active: "2"    # Green
      activity_idle: "8"      # Dim gray
      context_ok: "2"         # Green
      context_warn: "3"       # Yellow
      context_critical: "1"   # Red
      static_fields: "8"      # Dim gray
```

---

## 5. Implementation Plan

### 5.1 Phase 1: Extend Data Model (30 min)
**Tasks**:
1. Add new fields to `Metrics` struct
2. Add setter methods (`SetAgentState`, `SetTaskMode`, `SetConversationID`)
3. Write unit tests for new methods
4. Run tests: `go test ./internal/ui/status`

**Files**:
- `internal/ui/status/manager.go` (~50 lines added)
- `internal/ui/status/manager_test.go` (~80 lines added)

**Acceptance**:
- ✅ All tests pass
- ✅ New fields accessible via getters
- ✅ Thread-safe (test with `-race`)

### 5.2 Phase 2: Implement Formatting (1 hour)
**Tasks**:
1. Implement `FormatMedium(width int) string`
2. Implement `FormatFull(width int) string`
3. Implement `FormatAdaptive(width int) string`
4. Add helper functions (humanizeBytes, formatPercentage, etc.)
5. Write comprehensive tests for all layouts
6. Test edge cases (0 tokens, no provider, etc.)

**Files**:
- `internal/ui/status/formatter.go` (new file, ~200 lines)
- `internal/ui/status/formatter_test.go` (new file, ~250 lines)

**Example Implementation**:
```go
func (m *Manager) FormatFull(width int) string {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if !m.enabled {
        return ""
    }
    
    // Activity indicator
    activity := "[●]"
    if m.status.Metrics.Connected {
        activity = "[●]"  // Green in render phase
    } else {
        activity = "[○]"  // Gray in render phase
    }
    
    // Context percentage
    contextPct := formatPercentage(m.status.Metrics.TokenUsage)
    contextAbs := fmt.Sprintf("(%s/%s)",
        humanizeNumber(m.status.Metrics.TokenCount),
        humanizeNumber(m.status.Metrics.MaxTokens))
    
    // Agent state
    state := m.status.Metrics.AgentState
    if state == "" {
        state = "Ready"
    }
    
    // Provider/model
    provider := fmt.Sprintf("%s/%s",
        m.status.Metrics.Provider,
        truncate(m.status.Metrics.Model, 20))
    
    // TPS
    tps := ""
    if m.status.Metrics.TokensPerSec > 0 {
        tps = fmt.Sprintf("%.0f tok/s", m.status.Metrics.TokensPerSec)
    }
    
    // Conversation ID
    convID := ""
    if m.status.Metrics.ConversationID != "" {
        convID = "conv:" + m.status.Metrics.ConversationID[:6]
    }
    
    // Hotkeys
    hotkeys := "?:help ^C:quit"
    
    // Assemble with proper spacing
    parts := []string{activity, contextPct, contextAbs, state, provider, tps, convID, hotkeys}
    return strings.Join(filterEmpty(parts), "  ")
}
```

**Acceptance**:
- ✅ All format functions return correct strings
- ✅ Adaptive layout selects correct format
- ✅ Edge cases handled (nil, empty, zero values)
- ✅ Tests cover all width ranges

### 5.3 Phase 3: Enhance Aggregator (30 min)
**Tasks**:
1. Add agent state mapping for all event types
2. Add TPS calculation during streaming
3. Integrate session ID (read from context or config)
4. Integrate task mode (read from agent config)
5. Write integration tests

**Files**:
- `internal/ui/status/aggregator.go` (~50 lines modified)
- `internal/ui/status/aggregator_test.go` (~60 lines added)

**Acceptance**:
- ✅ Events correctly map to agent states
- ✅ TPS calculated accurately during streaming
- ✅ Session ID and task mode populated
- ✅ Integration tests pass

### 5.4 Phase 4: Update Renderer (30 min)
**Tasks**:
1. Add color support to `Render()` method
2. Add optional border rendering
3. Add configuration support
4. Write tests for colored output

**Files**:
- `internal/ui/status/renderer.go` (~40 lines modified)
- `internal/ui/status/renderer_test.go` (~50 lines added)

**Acceptance**:
- ✅ Colors applied correctly
- ✅ Border renders when enabled
- ✅ Configuration respected
- ✅ Tests verify ANSI codes

### 5.5 Phase 5: Integrate with PureTTY (30 min)
**Tasks**:
1. Update `updateStatusBar()` to use `FormatAdaptive()`
2. Pass terminal width to formatter
3. Update on resize events
4. Test end-to-end flow

**Files**:
- `internal/ui/adapters/puretty.go` (~20 lines modified)

**Acceptance**:
- ✅ Status bar shows full metrics
- ✅ Adapts to terminal width
- ✅ Updates on resize
- ✅ Manual testing confirms visual appearance

### 5.6 Phase 6: Testing & Polish (1 hour)
**Tasks**:
1. Run full test suite with `-race`
2. Run `make lint` and fix issues
3. Manual testing on different terminal sizes
4. Performance profiling
5. Update documentation

**Files**:
- All test files
- `docs/tui.md` (update status bar section)
- `specs/advanced-features-20251012/ROADMAP.md` (mark complete)

**Acceptance**:
- ✅ All tests pass (`go test ./...`)
- ✅ Race detector clean (`go test -race ./...`)
- ✅ Linter clean (`make lint`)
- ✅ Performance: <5ms render time
- ✅ Documentation updated

---

## 6. Testing Strategy

### 6.1 Unit Tests

**StatusManager** (`internal/ui/status/manager_test.go`):
- TestSetAgentState
- TestSetTaskMode
- TestSetConversationID
- TestCalculateTPS

**Formatter** (`internal/ui/status/formatter_test.go`):
- TestFormatCompact
- TestFormatMedium
- TestFormatFull
- TestFormatAdaptive_NarrowTerminal
- TestFormatAdaptive_MediumTerminal
- TestFormatAdaptive_WideTerminal
- TestHumanizeNumber
- TestFormatPercentage
- TestTruncate
- TestEdgeCases (zero values, nil, empty strings)

**Aggregator** (`internal/ui/status/aggregator_test.go`):
- TestProcessEvent_StateMapping
- TestCalculateTPS
- TestSessionIDIntegration

**Renderer** (`internal/ui/status/renderer_test.go`):
- TestRender_WithColors
- TestRender_WithBorder
- TestRender_Configuration

### 6.2 Integration Tests

**StatusIntegration** (`internal/ui/status/integration_test.go`):
- TestFullFlow_EventToDisplay
- TestAdaptiveLayout_Resize
- TestThroughput_100UpdatesPerSecond

### 6.3 Manual Testing

**Terminal Sizes**:
- [ ] 40 columns (very narrow)
- [ ] 60 columns (compact)
- [ ] 80 columns (medium)
- [ ] 120 columns (wide)
- [ ] 200 columns (very wide)

**Terminal Types**:
- [ ] iTerm2 (macOS)
- [ ] GNOME Terminal (Linux)
- [ ] Windows Terminal
- [ ] tmux session
- [ ] SSH session

**Scenarios**:
- [ ] Long conversation (100+ turns)
- [ ] High token usage (>80%, >95%)
- [ ] Rapid streaming (test TPS display)
- [ ] Tool execution (state changes)
- [ ] Terminal resize during conversation

---

## 7. Acceptance Criteria

### 7.1 Functional
- [ ] **Context percentage** displays correctly with color coding
- [ ] **Agent state** updates in real-time for all event types
- [ ] **Task mode** displays current mode (or omits if default)
- [ ] **Provider/model** shows correct information
- [ ] **Tokens per second** displays during streaming, hides when idle
- [ ] **Conversation ID** shows shortened session ID
- [ ] **Hotkeys** display on wide terminals
- [ ] **Adaptive layout** works for all terminal widths (40-200 cols)
- [ ] **Visual formatting** includes borders and colors

### 7.2 Non-Functional
- [ ] **Performance**: Render time <5ms (measure with benchmarks)
- [ ] **Thread safety**: No race conditions (`go test -race` passes)
- [ ] **Test coverage**: ≥90% for new code
- [ ] **Linter**: Zero errors (`make lint` passes)
- [ ] **Documentation**: Updated `docs/tui.md` with status bar details

### 7.3 Integration
- [ ] **Core events** trigger correct status updates
- [ ] **Terminal resize** updates layout immediately
- [ ] **Configuration** changes take effect
- [ ] **No regressions**: Existing TUI functionality still works

---

## 8. Risks & Mitigation

### Risk 1: TPS Calculation Inaccurate
**Impact**: Medium  
**Likelihood**: Medium  
**Mitigation**: 
- Use sliding window average (last 5 seconds)
- Handle edge cases (first event, pauses)
- Test with real LLM streaming

### Risk 2: Layout Doesn't Fit
**Impact**: Low  
**Likelihood**: Low  
**Mitigation**:
- Conservative width breakpoints
- Test on narrow terminals (40 cols)
- Graceful truncation at all widths

### Risk 3: Performance Degradation
**Impact**: Medium  
**Likelihood**: Low  
**Mitigation**:
- Benchmark all formatting functions
- Throttle updates (max 10Hz)
- Profile with `pprof` if needed

### Risk 4: Color Compatibility
**Impact**: Low  
**Likelihood**: Low  
**Mitigation**:
- Use standard ANSI 256 colors
- Fallback to basic colors if unsupported
- Test on various terminal types

---

## 9. Open Questions

1. **Q**: Should activity indicator animate (spinner)?  
   **A**: No, for v1. Static indicator with color change is sufficient.

2. **Q**: How to get session ID?  
   **A**: Read from `internal/session` package. Pass during PureTTY initialization.

3. **Q**: Should status bar be configurable (enable/disable fields)?  
   **A**: No, for v1. Use adaptive layout based on width only. Configuration in future version.

4. **Q**: What if token count exceeds max?  
   **A**: Display >100% and show in red. This indicates context compression failed.

---

## 10. Success Metrics

- [ ] **User feedback**: Positive response to status visibility
- [ ] **Performance**: <5ms render time (100% of samples)
- [ ] **Reliability**: Zero crashes or visual glitches
- [ ] **Coverage**: ≥90% test coverage on new code
- [ ] **Adoption**: Status bar enabled by default, <5% users disable it

---

## 11. Future Enhancements

**Phase 2** (Future):
- Animated activity indicator (spinner)
- Clickable conversation ID (copy to clipboard)
- Configurable field visibility
- Custom color schemes
- Status history graph (mini sparkline)
- Network latency indicator
- Cost tracking ($ per request)

---

## 12. References

- [RESEARCH.md](../advanced-features-20251012/RESEARCH.md) - Original research document
- [ROADMAP.md](../advanced-features-20251012/ROADMAP.md) - Implementation roadmap
- [docs/tui.md](../../docs/tui.md) - TUI documentation
- [docs/modes.md](../../docs/modes.md) - Task modes reference
- [AGENTS.md](../../AGENTS.md) - Development workflow

---

**End of FRD**

*Next Step: Review and approval, then begin Phase 1 implementation*

