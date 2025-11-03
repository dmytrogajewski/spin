# FRD-20251103-004: Progressive Retrieval Decision Logic

**Feature:** Phase 2 - Progressive Retrieval Decision Logic  
**Status:** Implementation  
**Created:** 2025-11-03  
**Phase:** 2.1  
**Roadmap:** [ace-progressive-context/ROADMAP.md](../ace-progressive-context/ROADMAP.md)

---

## Overview

Implement decision logic to determine WHEN retrieval should be triggered during agent execution based on trajectory state. This replaces the current "retrieve every turn" approach with intelligent, event-driven retrieval that responds to execution context.

**Key Innovation:** Instead of retrieving bullets every turn with the same query, we analyze the trajectory context and trigger retrieval only when meaningful changes occur (errors, tool changes, cache expiration).

---

## Requirements

### Functional Requirements

**FR1: Trigger Type System**
- Support 4 trigger types: initial, error, tool_change, interval
- Each trigger has specific detection logic
- Triggers are mutually exclusive (return first match)

**FR2: Initial Trigger (Turn 0)**
- Always trigger on turn 0
- Provides baseline bullets for execution start
- No trajectory analysis needed

**FR3: Error Trigger**
- Detect errors in recent steps (configurable lookback window)
- Use `HasRecentError()` from trajectory helpers
- Higher priority than tool_change and interval

**FR4: Tool Change Trigger**
- Detect when agent switches to a different tool
- Use `GetRecentTools()` and `hasToolChange()` from trajectory helpers
- Indicates strategy shift (e.g., bash → read → grep)

**FR5: Interval Trigger (Cache Expiration)**
- Trigger when last retrieval was N turns ago (TTL)
- Prevents stale bullet context in long conversations
- Lowest priority trigger

**FR6: Configuration**
- Progressive context can be enabled/disabled
- TTL configurable (default: 10 turns)
- Triggers can be selectively enabled
- All settings in ACE config section

### Non-Functional Requirements

**NFR1: Performance**
- Decision logic must complete in < 1ms
- No expensive operations (simple checks only)
- Leverage existing trajectory helpers

**NFR2: Maintainability**
- Clear separation of concerns (each trigger isolated)
- Easy to add new trigger types
- Well-tested with high coverage (90%+)

**NFR3: Backward Compatibility**
- When progressive context disabled, behave as before
- No breaking changes to existing ACE flow
- Opt-in feature

---

## API Specification

### shouldRetrieveProgressive

```go
// shouldRetrieveProgressive determines if retrieval should be triggered based on trajectory state.
// Returns (shouldRetrieve, triggerType).
//
// Trigger priority (first match wins):
// 1. TriggerInitial - Always on turn 0
// 2. TriggerError - Recent error detected
// 3. TriggerToolChange - Tool usage changed
// 4. TriggerInterval - Cache TTL expired
//
// Requires: ctx != nil
// Ensures: if shouldRetrieve, then triggerType is valid (not empty)
func (a *Agent) shouldRetrieveProgressive(ctx *trajectory.TrajectoryContext) (bool, trajectory.TriggerType)
```

**Behavior:**

1. **Check if enabled** - Return (false, "") if progressive context disabled
2. **Turn 0 check** - If CurrentTurn == 0, return (true, TriggerInitial)
3. **Error check** - If HasRecentError(lookback), return (true, TriggerError)
4. **Tool change check** - If tools changed recently, return (true, TriggerToolChange)
5. **Interval check** - If (CurrentTurn - LastRetrievalTurn) >= TTL, return (true, TriggerInterval)
6. **Default** - Return (false, "")

**Parameters:**
- `ctx`: TrajectoryContext containing execution state
  - Must not be nil (precondition)
  - Contains steps, retrieval history, current turn

**Returns:**
- `shouldRetrieve`: boolean indicating if retrieval needed
- `triggerType`: trajectory.TriggerType indicating why (empty string if false)

### Configuration Structure

```go
// ProgressiveContextConfig configures progressive retrieval behavior.
type ProgressiveContextConfig struct {
    // Enabled controls whether progressive context is active (default: false, opt-in)
    Enabled bool `yaml:"enabled" mapstructure:"enabled"`
    
    // CacheTTL is the number of turns before cache expires (default: 10)
    CacheTTL int `yaml:"cache_ttl" mapstructure:"cache_ttl"`
    
    // ErrorLookback is the number of recent steps to check for errors (default: 5)
    ErrorLookback int `yaml:"error_lookback" mapstructure:"error_lookback"`
    
    // ToolChangeLookback is the number of recent steps to check for tool changes (default: 3)
    ToolChangeLookback int `yaml:"tool_change_lookback" mapstructure:"tool_change_lookback"`
    
    // EnabledTriggers lists which triggers are active (default: all)
    // Valid values: "initial", "error", "tool_change", "interval"
    EnabledTriggers []string `yaml:"enabled_triggers" mapstructure:"enabled_triggers"`
}
```

**Add to ACERetrievalConfig:**

```go
type ACERetrievalConfig struct {
    TopK                int                        `yaml:"top_k" mapstructure:"top_k"`
    MinScore            float64                    `yaml:"min_score" mapstructure:"min_score"`
    ProgressiveContext  ProgressiveContextConfig   `yaml:"progressive_context" mapstructure:"progressive_context"` // NEW
}
```

---

## Design Decisions

### Why This Trigger Priority Order?

1. **Initial (highest)** - Must retrieve on first turn (baseline context)
2. **Error** - Errors are critical events requiring immediate strategy adjustment
3. **Tool Change** - Strategy shifts indicate need for tool-specific guidance
4. **Interval (lowest)** - Fallback to prevent stale context

### Why First-Match-Wins?

- Simplifies logic (no complex trigger combinations)
- Prevents redundant retrievals
- Each trigger logs why retrieval occurred

### Why Lookback Windows?

- Error lookback: Recent errors (last 5 steps) more relevant than old ones
- Tool lookback: Recent tools (last 3 steps) indicate current strategy
- Configurable for different use cases

### Why Opt-In (Enabled: false default)?

- Phase 2 is experimental
- Allows gradual rollout
- Existing users unaffected
- Can gather metrics before default-on

---

## Test Strategy

### Unit Tests

**Test Coverage Target: 90%+**

1. **TestShouldRetrieveProgressive_Disabled**
   - Progressive context disabled → always returns (false, "")
   - Verifies backward compatibility

2. **TestShouldRetrieveProgressive_TurnZero**
   - CurrentTurn == 0 → (true, TriggerInitial)
   - Works even with empty steps

3. **TestShouldRetrieveProgressive_RecentError**
   - Error in last N steps → (true, TriggerError)
   - No error → checks next trigger
   - Respects error lookback config

4. **TestShouldRetrieveProgressive_ToolChange**
   - Different tools in recent steps → (true, TriggerToolChange)
   - Same tool repeated → checks next trigger
   - Respects tool lookback config

5. **TestShouldRetrieveProgressive_Interval**
   - (CurrentTurn - LastRetrievalTurn) >= TTL → (true, TriggerInterval)
   - Within TTL → (false, "")
   - Respects cache TTL config

6. **TestShouldRetrieveProgressive_TriggerPriority**
   - Multiple triggers active → returns highest priority
   - Example: Turn 0 with error → returns TriggerInitial

7. **TestShouldRetrieveProgressive_EdgeCases**
   - Nil context → handled gracefully
   - Empty steps → works correctly
   - No previous retrievals → LastRetrievalTurn == 0

8. **TestProgressiveContextConfig_Defaults**
   - Verifies default values
   - Validates configuration

### Integration Tests (Future - Feature 2.3)

Will test in actual agent loop context.

---

## Implementation Plan

### Step 1: Add Configuration

1. Define `ProgressiveContextConfig` struct in `config.go`
2. Add to `ACERetrievalConfig.ProgressiveContext`
3. Add `DefaultProgressiveContextConfig()` function
4. Add validation method

### Step 2: Implement Core Logic

1. Add `shouldRetrieveProgressive()` method to Agent in `loop.go`
2. Implement trigger checks in priority order
3. Return first match

### Step 3: Helper Methods (if needed)

1. `isTriggerEnabled(trigger)` - Check if trigger in enabled list
2. May extract trigger logic to separate methods for clarity

### Step 4: Tests

1. Write comprehensive unit tests in `loop_test.go`
2. Achieve 90%+ coverage
3. Test all trigger types and edge cases

---

## Usage Examples

### Example 1: Initial Trigger

```go
// Turn 0 - always retrieves
ctx := trajectory.NewTrajectoryContext("install nodejs")
ctx.CurrentTurn = 0

shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)
// shouldRetrieve = true
// trigger = trajectory.TriggerInitial
```

### Example 2: Error Trigger

```go
// Turn 5 - error detected
ctx := trajectory.NewTrajectoryContext("install package")
ctx.CurrentTurn = 5
ctx.LastRetrievalTurn = 0

// Add steps with error
ctx.AppendSteps([]generator.TrajectoryStep{
    {StepNumber: 3, Content: "running npm install"},
    {StepNumber: 4, Content: "error: package not found"},
})

shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)
// shouldRetrieve = true
// trigger = trajectory.TriggerError
```

### Example 3: Tool Change Trigger

```go
// Turn 10 - tool changed from bash to grep
ctx := trajectory.NewTrajectoryContext("analyze logs")
ctx.CurrentTurn = 10
ctx.LastRetrievalTurn = 5

ctx.AppendSteps([]generator.TrajectoryStep{
    {StepNumber: 8, Content: "Tool: bash"},
    {StepNumber: 9, Content: "Tool: grep"}, // Different tool
})

shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)
// shouldRetrieve = true
// trigger = trajectory.TriggerToolChange
```

### Example 4: Interval Trigger

```go
// Turn 15 - no errors/tool changes, but cache expired
ctx := trajectory.NewTrajectoryContext("long task")
ctx.CurrentTurn = 15
ctx.LastRetrievalTurn = 0  // 15 turns ago

// TTL = 10, so 15 - 0 >= 10
shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)
// shouldRetrieve = true
// trigger = trajectory.TriggerInterval
```

### Example 5: No Trigger

```go
// Turn 8 - no events, within TTL
ctx := trajectory.NewTrajectoryContext("task")
ctx.CurrentTurn = 8
ctx.LastRetrievalTurn = 5  // 3 turns ago

// No errors, no tool changes, within TTL (10)
shouldRetrieve, trigger := agent.shouldRetrieveProgressive(ctx)
// shouldRetrieve = false
// trigger = ""
```

---

## Configuration Example

```yaml
# config.yml
ace:
  enabled: true
  retrieval:
    top_k: 5
    min_score: 0.7
    progressive_context:
      enabled: true
      cache_ttl: 10
      error_lookback: 5
      tool_change_lookback: 3
      enabled_triggers:
        - initial
        - error
        - tool_change
        - interval
```

---

## Acceptance Criteria

- [x] `ProgressiveContextConfig` defined in config.go ✅
- [x] `shouldRetrieveProgressive()` implemented ✅
- [x] All 4 trigger types working correctly ✅
- [x] Unit tests written (100% coverage) ✅
- [x] All tests pass ✅
- [x] Edge cases handled (nil ctx, empty steps, etc.) ✅
- [x] Performance < 1ms (validated) ✅
- [x] Backward compatible (disabled by default) ✅
- [x] Documentation updated ✅

---

## Definition of Done

- [x] Configuration struct complete ✅
- [x] Core method implemented ✅
- [x] Unit tests written and passing (8 tests) ✅
- [x] Coverage = 100% ✅
- [x] Race detector clean ✅
- [x] Linter clean (go vet, go fmt) ✅
- [x] Documentation updated ✅
- [x] Roadmap item closed ✅

**Completion Date:** 2025-11-03

**Implementation Summary:**
- Added `ProgressiveContextConfig` to `ACERetrievalConfig`
- Implemented `shouldRetrieveProgressive()` with all 4 triggers
- 8 comprehensive tests covering all triggers, priority, and edge cases
- 100% test coverage on progressive.go
- All linters pass, race detector clean
- Backward compatible (disabled by default)

**Files Created:**
- `internal/agent/progressive.go` (47 lines)
- `internal/agent/progressive_test.go` (203 lines, 8 tests)

**Files Modified:**
- `internal/agent/config.go` (added ProgressiveContextConfig)

---

## Follow-Up Features

- Feature 2.2: Dynamic Query Building (will use trigger type)
- Feature 2.3: Agent Loop Integration (will call this method)
- Feature 4.1: Enhanced Configuration (more trigger options)

---

## References

- [Feature 1.2 FRD](./FRD-20251102-002-trajectory-context-helpers.md) - Helper methods used
- [Roadmap](../ace-progressive-context/ROADMAP.md)
- [Proposal](../ace-progressive-context/PROPOSAL-ACE-PROGRESSIVE-CONTEXT-RETRIEVAL.md)
- [Trajectory Context Docs](../../docs/trajectory-context.md)
