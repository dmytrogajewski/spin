# Feature Requirements Document: Cycle Auto-Discovery

**Feature ID:** FRD-20251015020000  
**Feature:** Cycle Auto-Discovery  
**Priority:** MEDIUM - Reliability improvement for autonomous operation  
**Status:** Implementation Ready  
**Date:** 2025-10-15  
**Author:** Spin Agent  

---

## Problem Statement

Autonomous agents can enter infinite reasoning loops where:
- LLM repeats similar responses 3+ times
- Same tool called repeatedly with no progress  
- Agent oscillates between two states (A → B → A → B)
- Same error occurs multiple times

These loops waste computational resources, frustrate users, and reduce agent reliability. Current safeguards (`MaxTurns`, `Timeout`) only limit damage but don't prevent cycles.

## Solution Overview

Implement automatic cycle detection with intelligent intervention strategies to break reasoning loops and maintain productivity. The system will:

1. **Detect cycles** using multiple methods (similarity, repeated patterns, oscillation)
2. **Apply interventions** based on cycle severity and conversation length
3. **Escalate to user** when automated interventions fail
4. **Emit events** for status bar integration and monitoring

## Requirements

### Functional Requirements

#### FR1: Cycle Detection Engine
- **Must detect** response similarity >80% for 3+ consecutive responses
- **Must detect** same tool called 3+ times consecutively  
- **Must detect** A→B→A→B oscillation patterns
- **Must detect** identical errors occurring 3+ times
- **Must maintain** rolling snapshot history (configurable window size)
- **Must be configurable** via YAML (thresholds, window size, enable/disable)

#### FR2: Intervention Strategies
- **Soft intervention**: Inject reflection prompt for early cycles (<10 turns)
- **Medium intervention**: Force context summarization for mid-stage cycles (10-30 turns)  
- **Hard intervention**: Escalate to user for persistent cycles (>30 turns)
- **Must preserve** conversation continuity during interventions
- **Must be extensible** for future intervention types

#### FR3: Agent Integration
- **Must integrate** seamlessly into existing `Agent.Execute()` loop
- **Must check for cycles** after each LLM response (before tool execution)
- **Must apply interventions** when cycles detected
- **Must continue execution** after successful intervention
- **Must pause agent** on user escalation intervention

#### FR4: Event Emission
- **Must emit** `EventCycleDetected` when cycles identified
- **Must emit** `EventInterventionApplied` when interventions triggered
- **Must emit** `EventTurnPaused` when escalating to user
- **Must include** cycle type, confidence, and intervention details in events

### Non-Functional Requirements

#### NFR1: Performance
- **Cycle detection latency**: <10ms per check (for minimal UX impact)
- **Memory overhead**: <1MB for snapshot history (default window)
- **No blocking operations** in detection logic

#### NFR2: Reliability
- **False positive rate**: <5% (doesn't flag legitimate similar responses)
- **Intervention success rate**: >70% (breaks cycles and agent continues productively)
- **Graceful degradation**: Continue agent operation if cycle detection fails

#### NFR3: Configuration
- **YAML integration**: Add cycle detection section to config
- **Runtime reconfiguration**: Support hot-reloading of detection parameters
- **Sensible defaults**: Work out-of-box without configuration

## Architecture

### Package Structure
```
internal/core/cycle/
├── detector.go       # Main CycleDetector with snapshot management
├── patterns.go       # Pattern detection algorithms (similarity, tools, oscillation)
├── intervention.go   # Intervention strategies and escalation ladder
├── similarity.go     # Text similarity calculation (Jaccard)
└── types.go         # Common types and interfaces
```

### Key Components

#### CycleDetector
```go
type Detector struct {
    history []Snapshot
    config  Config
    mu      sync.RWMutex
}

type Snapshot struct {
    Turn         int
    Response     string
    ToolCalls    []string
    Error        string
    Timestamp    time.Time
}

type Config struct {
    WindowSize        int     // Number of snapshots to compare (default: 3)
    SimilarityThresh  float64 // Similarity threshold (default: 0.8)
    ToolRepeatLimit   int     // Max identical tool calls (default: 3)
    ErrorRepeatLimit  int     // Max identical errors (default: 3)
    Enabled          bool    // Enable/disable cycle detection
}
```

#### Pattern Detection
- **Similarity Detection**: Jaccard similarity of response text
- **Tool Pattern**: Repeated tool calls with same parameters
- **Oscillation**: A→B→A→B state pattern detection
- **Error Pattern**: Repeated identical errors

#### Intervention Ladder
1. **Reflection** (<10 turns): "I notice you may be repeating yourself..."
2. **Summarization** (10-30 turns): Compress context to 50%
3. **User Escalation** (>30 turns): Pause agent, request guidance

### Integration Points

#### Agent.Execute() Integration
```go
// In agent loop, after LLM response
detector.Record(snapshot)

// Check for cycles
if cycleType := detector.Check(); cycleType != CycleNone {
    intervention := selectIntervention(cycleType, turn)
    if err := intervention.Apply(ctx, messages); err != nil {
        // Handle intervention error
    }
    
    // Emit events for status bar
    emitter.Emit(EventCycleDetected{...})
}
```

## Configuration

```yaml
agent:
  cycle_detection:
    enabled: true
    window_size: 3
    similarity_threshold: 0.8
    tool_repeat_limit: 3
    error_repeat_limit: 3
```

## Testing Strategy

### Unit Tests
- **CycleDetector**: Snapshot recording, pattern detection accuracy
- **Similarity**: Jaccard calculation correctness, edge cases
- **Interventions**: Each strategy applies correctly
- **Configuration**: YAML parsing, validation

### Integration Tests
- **Synthetic cycles**: Create conversations that trigger each cycle type
- **Intervention effectiveness**: Verify interventions break cycles
- **Agent continuity**: Ensure agent continues productively after intervention

### E2E Tests
- **Manual testing**: Prompts designed to cause cycles
- **Long conversations**: 50+ turn conversations with intermittent cycles
- **Performance**: Detection latency under load

## Success Metrics

### Functional
- **Detection accuracy**: >80% of actual cycles detected
- **False positive rate**: <5% 
- **Intervention success**: >70% of cycles broken

### Performance
- **Detection latency**: <10ms per check
- **Memory usage**: <1MB overhead
- **No blocking**: Zero impact on agent responsiveness

### Quality
- **Test coverage**: ≥90% for new packages
- **Linter compliance**: Zero errors
- **Documentation**: Complete Godoc coverage

## Implementation Plan

### Phase 1: Core Detection (Week 1)
1. Create package structure and interfaces
2. Implement CycleDetector with snapshot management
3. Implement similarity calculation
4. Add basic unit tests

### Phase 2: Pattern Detection (Week 1)
1. Implement tool pattern detection
2. Implement oscillation detection  
3. Implement error pattern detection
4. Add comprehensive unit tests

### Phase 3: Interventions (Week 2)
1. Implement reflection intervention
2. Implement summarization intervention
3. Implement user escalation intervention
4. Add intervention unit tests

### Phase 4: Agent Integration (Week 2)
1. Integrate detector into Agent.Execute()
2. Wire intervention application
3. Add event emission
4. Integration testing

### Phase 5: Configuration & Polish (Week 2)
1. Add YAML configuration support
2. Update agent options
3. Documentation and examples
4. Final testing and validation

## Risk Mitigation

### Risk 1: False Positives
**Impact**: Legitimate similar responses flagged as cycles  
**Mitigation**: Conservative thresholds, multiple detection methods, configuration options

### Risk 2: Performance Impact
**Impact**: Detection adds latency to agent loop  
**Mitigation**: Efficient algorithms (<10ms target), optional disabling, benchmarking

### Risk 3: Intervention Failure
**Impact**: Cycles not broken, agent remains stuck  
**Mitigation**: Escalation ladder, graceful degradation, user fallback

## Dependencies

- **Existing**: `internal/core/agent.go`, `internal/core/event.go`, `internal/core/history.go`
- **New**: `internal/core/cycle/` package (self-contained)
- **Optional**: Integration with context summarization for medium intervention

## Definition of Done

- [ ] FRD approved and all requirements implemented
- [ ] CycleDetector detects all pattern types with >80% accuracy
- [ ] Intervention strategies successfully break >70% of cycles
- [ ] Agent integration works seamlessly in existing loop
- [ ] Configuration options fully implemented
- [ ] Event emission integrated with status bar
- [ ] Unit tests: ≥90% coverage, all passing
- [ ] Integration tests: Synthetic cycle scenarios pass
- [ ] Performance benchmarks: <10ms detection latency
- [ ] Documentation: Complete Godoc and user guide updated
- [ ] Linter: Zero errors, complexity ≤15

---

**End of FRD**

*This FRD implements Feature 4 from the Advanced Features Roadmap (2025-10-12).*  
*Next: Begin implementation with core cycle detection package.*
