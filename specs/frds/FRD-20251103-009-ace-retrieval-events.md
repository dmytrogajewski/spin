# FRD-20251103-009: ACE Retrieval Events for TUI Integration

**Status:** ✅ COMPLETED  
**Created:** 2025-11-03  
**Completed:** 2025-11-03  
**Feature:** Phase 4, Feature 4.2 - Metrics & Logging (ACE Events)  
**Related:**
- FRD-20251103-004: Progressive Retrieval Decision Logic
- FRD-20251103-005: Dynamic Query Building
- ROADMAP.md Phase 4

---

## 1. Overview

### Purpose
Add ACE-specific event types to the event system to enable real-time TUI updates when progressive retrieval happens. This provides visibility into retrieval decisions, cache performance, and query evolution.

### Current State
- Event system exists (`internal/events/`) with tool/turn/content events
- Progressive retrieval implemented with logging (slog)
- TrajectoryContext tracks metrics (CacheHits, CacheMisses, TotalRetrievals)
- **Missing**: ACE-specific events for TUI integration

### Target State
- New `EventACERetrieval` event type
- New `ACERetrievalData` struct with retrieval details
- Agent loop emits ACE events when retrieval triggers
- TUI can display real-time retrieval information

---

## 2. Requirements

### Functional Requirements

**FR-1: ACE Event Type**
- MUST add `EventACERetrieval` to EventType enum
- MUST implement String() representation

**FR-2: ACE Event Data**
- MUST define `ACERetrievalData` struct with fields:
  - Turn (int): Which turn triggered retrieval
  - Trigger (string): Trigger type (initial, error, tool_change, interval)
  - Query (string): The query used for retrieval
  - BulletsRetrieved (int): Number of bullets retrieved
  - BulletsNew (int): Number of new bullets (not cached)
  - CacheSize (int): Total bullets in cache after retrieval
  - CacheHitRate (float64): Current cache hit rate percentage

**FR-3: Helper Method**
- MUST add `ACERetrievalData() (ACERetrievalData, bool)` method to Event type

**FR-4: Event Emission**
- MUST emit ACE retrieval event in agent loop after successful retrieval
- MUST only emit when EmitACEEvents config flag is true
- MUST include all metrics in event data

### Non-Functional Requirements

**NFR-1: Performance**
- Event emission MUST NOT add measurable latency to retrieval
- Use BackpressureDrop for ACE events (fire-and-forget)

**NFR-2: Backward Compatibility**
- MUST NOT break existing event consumers
- TUI must gracefully handle absence of ACE events

**NFR-3: Test Coverage**
- MUST achieve 100% coverage for new event type
- MUST test event emission in agent loop

---

## 3. Design

### 3.1 Event Type Addition

```go
// internal/events/event.go

const (
	// ... existing event types
	
	// ACE events - agentic context engineering
	EventACERetrieval
)

// Update String() method
func (e EventType) String() string {
	names := []string{
		// ... existing names
		"ace_retrieval",
	}
	// ...
}
```

### 3.2 ACE Event Data Structure

```go
// internal/events/event.go

// ACERetrievalData contains ACE progressive retrieval information.
type ACERetrievalData struct {
	Turn             int     `json:"turn"`
	Trigger          string  `json:"trigger"`           // initial, error, tool_change, interval
	Query            string  `json:"query"`
	BulletsRetrieved int     `json:"bullets_retrieved"` // Total bullets from this retrieval
	BulletsNew       int     `json:"bullets_new"`       // New bullets (not cached)
	CacheSize        int     `json:"cache_size"`        // Total cached bullets after retrieval
	CacheHitRate     float64 `json:"cache_hit_rate"`    // Hit rate: hits / (hits + misses)
}

// ACERetrievalData returns the event data as ACERetrievalData if possible.
func (e Event) ACERetrievalData() (ACERetrievalData, bool) {
	data, ok := e.Data.(ACERetrievalData)
	return data, ok
}
```

### 3.3 Agent Loop Integration

```go
// internal/agent/loop.go

// After successful progressive retrieval
if shouldRetrieve {
	query := a.buildQueryFromContext(trajCtx, trigger)
	
	retrievedBullets, err := a.aceService.Retrieve(ctx, query)
	if err != nil {
		slog.Warn("ACE retrieval failed", "error", err)
	} else {
		// Record retrieval event
		event := trajectory.RetrievalEvent{...}
		trajCtx.RecordRetrieval(event, retrievedBullets)
		
		// Emit ACE event for TUI (NEW)
		if a.config.ACE.Retrieval.ProgressiveContext.EmitACEEvents {
			a.emitACERetrievalEvent(trajCtx, trigger, query, len(retrievedBullets), turn)
		}
	}
}

// Helper method (NEW)
func (a *Agent) emitACERetrievalEvent(
	ctx *trajectory.TrajectoryContext,
	trigger trajectory.TriggerType,
	query string,
	bulletsRetrieved int,
	turn int,
) {
	// Calculate cache hit rate
	total := ctx.CacheHits + ctx.CacheMisses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(ctx.CacheHits) / float64(total)
	}
	
	// Determine new bullets vs cached
	bulletsNew := ctx.CacheMisses  // Approximation
	
	a.emitter.Emit(events.Event{
		Type: events.EventACERetrieval,
		Data: events.ACERetrievalData{
			Turn:             turn,
			Trigger:          string(trigger),
			Query:            query,
			BulletsRetrieved: bulletsRetrieved,
			BulletsNew:       bulletsNew,
			CacheSize:        len(ctx.BulletCache),
			CacheHitRate:     hitRate,
		},
	})
}
```

---

## 4. Test Strategy

### 4.1 Unit Tests

**Test: EventType String Representation**
- Verify EventACERetrieval.String() returns "ace_retrieval"

**Test: ACERetrievalData Type Assertion**
- Create Event with ACERetrievalData
- Verify ACERetrievalData() returns data and true
- Verify other type assertions return false

**Test: Agent Emits ACE Event**
- Mock event emitter
- Trigger progressive retrieval
- Verify EventACERetrieval emitted with correct data
- Verify EmitACEEvents flag is respected

**Test: Cache Hit Rate Calculation**
- Set up context with known hits/misses
- Emit event
- Verify hit rate is correctly calculated

---

## 5. Implementation Plan (Micro-TDD)

### Step 1: Add EventACERetrieval constant
- Test: EventType string representation
- Implement: Add constant and update String()

### Step 2: Add ACERetrievalData struct
- Test: Type assertion helper method
- Implement: Define struct and helper

### Step 3: Emit event in agent loop
- Test: Event emission with mocked emitter
- Implement: Add emitACERetrievalEvent() method

### Step 4: Respect EmitACEEvents flag
- Test: No emission when flag is false
- Implement: Add config check

---

## 6. Configuration

No new configuration needed. Uses existing:

```yaml
ace:
  retrieval:
    progressive_context:
      emit_ace_events: true  # Default: true
```

---

## 7. Acceptance Criteria

- [x] EventACERetrieval type added to event.go ✅
- [x] ACERetrievalData struct defined ✅
- [x] ACERetrievalData() helper method implemented ✅
- [x] Agent emits ACE events on retrieval ✅
- [x] EmitACEEvents flag respected ✅
- [x] All tests pass ✅
- [x] 100% test coverage for new code ✅
- [x] go vet clean ✅
- [x] go fmt clean ✅
- [x] go test -race clean ✅

---

## 8. Definition of Done

- [x] Implementation complete ✅
- [x] All tests pass ✅
- [x] Test coverage 100% ✅
- [x] Quality checks pass ✅
- [x] Documentation updated ✅
- [x] FRD marked as COMPLETED ✅
- [x] Roadmap updated ✅

---

## 9. Implementation Summary

**Files Modified:**
- `internal/events/event.go`: Added EventACERetrieval constant, ACERetrievalData struct, helper method (45 lines)
- `internal/events/event_test.go`: Added TestEvent_ACERetrievalData (54 lines)
- `internal/agent/agent.go`: Added emitACERetrievalEvent helper method (32 lines)
- `internal/agent/agent_test.go`: Added TestAgent_emitACERetrievalEvent (70 lines)
- `internal/agent/loop.go`: Integrated event emission after RecordRetrieval (4 lines)

**Test Coverage:**
- 2 unit tests (events and agent)
- 100% coverage of new code
- All existing tests continue to pass

**Implementation Notes:**
- EventACERetrieval added after EventCommandApproval in enum
- ACERetrievalData includes 7 fields: Turn, Trigger, Query, BulletsRetrieved, BulletsNew, CacheSize, CacheHitRate
- Cache hit rate calculated as: hits / (hits + misses)
- Event emission respects EmitACEEvents config flag (default: true)
- No breaking changes, fully backward compatible

---

**END OF FRD**
