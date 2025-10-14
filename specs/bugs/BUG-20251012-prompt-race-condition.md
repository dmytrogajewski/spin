# BUG: Prompt Rendering Race Condition

**ID:** BUG-20251012-prompt-race-condition
**Severity:** CRITICAL
**Status:** Root cause identified
**Created:** 2025-10-12

---

## Symptom

User types in prompt but sees no output or intermittent display.

## Root Cause

**Race condition:** Prompt loop and sticky coordinator both write to same io.Writer concurrently without synchronization.

### Race Detector Output
```
WARNING: DATA RACE
Write at bytes.Buffer by prompt.Loop.handleEvent()
  → renderer.Redraw() → buffer.Write()

Write at same bytes.Buffer by StickyBottomCoordinator.RenderStickyArea()
  → renderer.Redraw() → buffer.Write()
```

### Code Analysis

**In `internal/ui/prompt/loop.go`:**
```go
func (l *Loop) handleEvent(ctx context.Context, event term.KeyEvent) bool {
    switch event.Kind {
    case term.KeyRune:
        l.model.Insert(event.Rune)
        l.renderer.Redraw(l.model, "")  // ❌ Direct write to output
```

**In `internal/ui/sticky/coordinator.go`:**
```go
func (c *StickyBottomCoordinator) RenderStickyArea() error {
    // ...
    c.promptRenderer.Redraw(c.promptModel, "")  // ❌ Same output, different goroutine
```

**Result:** TWO goroutines writing to same buffer without mutex = RACE CONDITION

---

## Impact

- ✗ Typing doesn't show in prompt (intermittent)
- ✗ Race detector failures in tests
- ✗ Unpredictable UI behavior
- ✗ Data corruption possible

---

## Fix Strategy

**Option 1: Prompt Loop Notifies Coordinator** (CHOSEN)
- Prompt loop updates model only
- Sends notification to coordinator
- Coordinator renders sticky area (which includes prompt)
- ALL rendering goes through single lock

**Option 2: Lock Around All Writes**
- Add mutex to output writer
- All writes acquire lock
- More complex, less clean

---

## Implementation

### 1. Create Render Notification Channel

```go
// In StickyBottomCoordinator
type StickyBottomCoordinator struct {
    // ... existing fields
    renderRequests chan struct{}
}

func (c *StickyBottomCoordinator) RequestRender() {
    select {
    case c.renderRequests <- struct{}{}:
    default: // Drop if busy (throttle)
    }
}
```

### 2. Start Render Loop

```go
func (c *StickyBottomCoordinator) StartRenderLoop(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(100 * time.Millisecond)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-c.renderRequests:
                c.mu.Lock()
                c.RenderStickyArea()
                c.mu.Unlock()
            case <-ticker.C:
                // Periodic render for status updates
                c.mu.Lock()
                c.RenderStickyArea()
                c.mu.Unlock()
            }
        }
    }()
}
```

### 3. Fix Prompt Loop

```go
// In prompt/loop.go - add render callback
type Loop struct {
    model       *Model
    renderer    PromptRenderer
    keys        <-chan term.KeyEvent
    out         chan string
    onRender    func()  // NEW: Callback to request render
}

func (l *Loop) handleEvent(ctx context.Context, event term.KeyEvent) bool {
    switch event.Kind {
    case term.KeyRune:
        l.model.Insert(event.Rune)
        if l.onRender != nil {
            l.onRender()  // Request render, don't do it directly
        }
```

---

## Acceptance Criteria

- [ ] Race detector clean with interactive tests
- [ ] User typing shows up reliably
- [ ] All rendering goes through single lock
- [ ] No concurrent writes to output

---

## Status: Fix in progress

