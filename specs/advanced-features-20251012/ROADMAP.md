# Implementation Roadmap: Advanced TUI and Agent Features

**Project:** Spin Coding Agent - Advanced Features
**Research Document:** [RESEARCH.md](./RESEARCH.md)
**Created:** 2025-10-12
**Timeline:** 6 weeks
**Methodology:** TRIZ-based systematic innovation

---

## Overview

This roadmap implements five advanced features for the Spin coding agent:

1. **Persistent Status Bar** - Real-time metrics display
2. **Context Summarization** - Automatic context compression
3. **VRAM Auto-Tuning** - Intelligent model parameter adjustment
4. **Cycle Auto-Discovery** - Detection of agent reasoning loops
5. **Enhanced Approval Mechanisms** - TUI approval dialogs

**Total Estimated Effort:** 6 weeks (1 developer)

**Quality Standards:**
- Test coverage: 90%+ for all new code
- Go 1.24 compliance
- SOLID, DRY, KISS principles
- Clean architecture

---

## Feature 1: Persistent Status Bar ✅ COMPLETE

**Objective:** Real-time metrics display at bottom of TUI showing agent state, context usage, provider info, and throughput.

**Estimated Effort:** 1 week
**Actual Effort:** Completed 2025-10-12 (6 phases)

### Functional Requirements

**FR-1.1: Status Metrics Display**
- Must display agent state (Thinking, Calling tools, Planning)
- Must show context usage (current tokens / max tokens, percentage)
- Must show task mode (regular, review, compact, planning)
- Must display provider and model information
- Must show throughput (tokens per second)
- Must display conversation/session ID

**FR-1.2: Adaptive Rendering**
- Must adapt to terminal width with 3 modes:
  - Compact mode: <60 columns (minimal display)
  - Medium mode: 60-100 columns (abbreviated display)
  - Full mode: 100+ columns (complete display)
- Must position status bar between output and prompt
- Must not disrupt terminal scrollback
- Must handle terminal resize events gracefully

**FR-1.3: Real-Time Updates**
- Must update metrics in real-time based on agent events
- Must calculate tokens/sec from content generation timing
- Must calculate context percentage from history
- Must extract agent state from event types
- Must be thread-safe (no race conditions)

**FR-1.4: Performance**
- Render time: <1ms (p99)
- Update latency: <10ms (p99)
- Must throttle updates to minimize flicker

**FR-1.5: Configuration**
- Must support enable/disable option
- Must allow configurable compact width threshold
- Must allow configurable update interval
- Must work in terminals as narrow as 40 columns
- Must gracefully degrade for very narrow terminals


---

## Feature 2: Context Summarization

**Objective:** Automatic context compression using importance-weighted message selection to prevent overflow.

**Estimated Effort:** 1 week

### Functional Requirements

**FR-2.1: Message Classification**
- Must classify messages by importance:
  - Critical: User messages, tool calls, errors
  - High: Code blocks, file changes
  - Medium: Regular assistant responses
  - Low: System messages, thinking content
- Classification must be deterministic and fast

**FR-2.2: Hybrid Compression Algorithm**
- Must compress messages using importance-weighted selection
- Must preserve 100% of critical messages
- Must maintain chronological order after compression
- Must work within target token budget
- Must use greedy selection algorithm for efficiency

**FR-2.3: Automatic Triggering**
- Must trigger automatically at 80% context capacity (configurable)
- Must integrate with existing History management
- Must recalculate tokens after compression
- Must emit compression events with before/after metrics

**FR-2.4: Metrics and Monitoring**
- Must track messages before/after compression
- Must track tokens before/after compression
- Must calculate compression ratio
- Must measure compression duration
- Must display TUI notifications for compression events

**FR-2.5: Performance**
- Compression time: <100ms for 1000 messages
- Compression overhead: <100ms per trigger
- Must prevent context overflow in 200+ turn conversations

**FR-2.6: Configuration**
- Must support enable/disable option
- Must allow configurable threshold (default: 0.8)
- Must allow strategy selection (hybrid, sliding_window)
- Must allow preserve_critical option

---

## Feature 3: VRAM Auto-Tuning

**Objective:** Intelligent model parameter selection based on available VRAM (quantization, context length, GPU layers).

**Estimated Effort:** 1 week

### Functional Requirements

**FR-3.1: Multi-Platform VRAM Detection**
- Must detect VRAM on NVIDIA GPUs (via nvidia-smi)
- Must detect VRAM on AMD GPUs (via rocm-smi)
- Must detect VRAM on macOS (via system_profiler for Metal)
- Must gracefully fallback for CPU-only systems
- Must report total and available VRAM
- Must report GPU name/model
- Detection time: <500ms

**FR-3.2: Model Requirements Calculation**
- Must calculate optimal quantization (f16, q8_0, q4_0) based on VRAM
- Must estimate KV cache size for context length
- Must reserve headroom (default: 1GB)
- Must try quantizations in quality order (f16 → q8_0 → q4_0)
- Must implement fallback strategies:
  - Reduce context length if needed
  - Offload to CPU (partial GPU layers) if needed

**FR-3.3: Ollama Integration**
- Must query model information from Ollama API
- Must apply calculated parameters to Ollama options (num_ctx, num_gpu, num_batch)
- Must provide AutoTune() method for automatic tuning
- Must log tuning decisions with details

**FR-3.4: Model Validation**
- Must validate if model fits in available VRAM
- Must provide user-friendly error messages for oversized models
- Must warn when near VRAM limit (>90%)
- Must suggest alternatives (smaller models, quantization)
- Must support startup validation option

**FR-3.5: Configuration**
- Must support enable/disable option
- Must allow configurable headroom (default: 1024MB)
- Must support validate_on_startup option
- Must handle missing detection tools gracefully

---

## Feature 4: Cycle Auto-Discovery

**Objective:** Automatic detection of agent reasoning loops with multi-level interventions.

**Estimated Effort:** 1 week

### Functional Requirements

**FR-4.1: Cycle Detection**
- Must detect 4 types of cycles:
  - Similar responses (using Jaccard similarity)
  - Repeated tool calls (same tool 3+ times)
  - Oscillation patterns (A → B → A → B)
  - Same error repeated (3+ times)
- Must use configurable sliding window (default: 3 turns)
- Must use configurable similarity threshold (default: 0.8)
- Must be thread-safe
- Detection time: <1ms per check

**FR-4.2: Multi-Level Interventions**
- Must implement 3 intervention levels:
  - **Soft**: Inject reflection prompt to break cycle
  - **Medium**: Force context summarization (50% reduction)
  - **Hard**: Escalate to user (pause execution)
- Must select intervention based on turn count (escalation ladder):
  - Turns <10: Reflection
  - Turns 10-30: Summarization
  - Turns >30: User escalation

**FR-4.3: Agent Integration**
- Must integrate cycle detection into agent turn loop
- Must record snapshot after each LLM response
- Must check for cycles after each turn
- Must emit warning events when cycles detected
- Must apply intervention automatically
- Must pause execution on user escalation

**FR-4.4: Event Notifications**
- Must emit cycle warning events with:
  - Cycle type detected
  - Intervention applied
  - Turn number
- Must display TUI notifications for cycle warnings

**FR-4.5: Configuration**
- Must support enable/disable option
- Must allow configurable window_size (default: 3)
- Must allow configurable similarity_threshold (default: 0.8)
- Must allow configurable tool_repeat_limit (default: 3)
- Must allow configurable intervention thresholds:
  - soft_turn_threshold (default: 10)
  - medium_turn_threshold (default: 30)
  - hard_turn_threshold (default: 50)

**FR-4.6: Metrics**
- Must track cycle detection rate
- Must track false positive rate (if possible)
- Must measure intervention effectiveness

---

## Feature 5: Enhanced Approval Mechanisms

**Objective:** TUI approval dialogs (leveraging existing validator infrastructure).

**Note:** 95% of approval infrastructure exists. Only TUI dialog missing (~150 lines).

**Estimated Effort:** 3 days

### Functional Requirements

**FR-5.1: TUI Approval Dialog**
- Must display modal approval dialog when dangerous command detected
- Must show:
  - Command to be executed
  - Reason for approval request
  - Working directory
  - Available actions (Approve, Deny, Modify, Help)
- Must center dialog on screen
- Must handle keyboard input (A/D for approve/deny)
- Must support timeout (default: 60s)
- Timeout must auto-deny and return to agent

**FR-5.2: Integration with Existing Approval System**
- Must integrate with existing Validator (command classification)
- Must integrate with existing Executor (command execution)
- Must pause output rendering during approval
- Must resume rendering after response
- Must auto-execute safe commands (no dialog)
- Must auto-block forbidden commands (no dialog)
- Must only show dialog for commands requiring approval

**FR-5.3: Approval Response Handling**
- Must handle approve response (execute command)
- Must handle deny response (skip command, notify agent)
- Must handle timeout (deny and notify)
- Must communicate response back to agent via channel

**FR-5.4: Optional Audit Trail**
- Must support optional audit logging to JSONL file
- Must log:
  - Timestamp
  - Request ID
  - Command
  - Reason
  - Working directory
  - Approval decision (approved/denied)
  - User reason (if denied)
  - Duration
- Audit file must have 0600 permissions (user read/write only)
- Must support configurable audit log path

**FR-5.5: Configuration**
- Must support enable/disable option
- Must allow configurable timeout (default: 60s)
- Must support optional audit_log path
- Must work without audit logging if not configured

---

## Integration & Production Readiness (Week 6)

**Objective:** End-to-end integration, performance optimization, security review, and release preparation.

**Estimated Effort:** 1 week

### Functional Requirements

**FR-6.1: Cross-Feature Integration**
- All 5 features must work together without conflicts
- Must pass 100-turn conversation test with all features enabled
- Must verify:
  - Status bar updates throughout conversation
  - Context compression triggers when needed
  - Cycle detection prevents or breaks loops
  - VRAM auto-tuning applies correctly
  - Approval dialogs work when triggered

**FR-6.2: Performance SLOs**
- Status Bar: <1ms render (p99), <10ms update latency (p99)
- Context Compression: <100ms for 1000 messages (p99)
- VRAM Auto-Tuning: <500ms detection (p99)
- Cycle Detection: <1ms per check (p99)
- Overall Agent: <20ms overhead per turn (p99)
- Memory: <500MB for 100-turn conversation
- No race conditions in any component

**FR-6.3: Security Requirements**
- Approval system must be bypass-proof
- Command validation must handle malicious inputs
- VRAM detection must be injection-safe
- Audit log must have 0600 permissions
- Context compression must not leak sensitive data
- All external commands must be sanitized

**FR-6.4: Error Handling**
- All errors must be properly wrapped with context
- No panics in production code
- All goroutines must have panic recovery
- Graceful degradation for all failures
- User-facing errors must be helpful

**FR-6.5: Documentation**
- Technical documentation for all packages
- User guide with examples and screenshots
- Configuration guide with all options
- Troubleshooting section
- Production configuration templates
- Release notes with breaking changes
- Migration guide (if applicable)

---

## Configuration Schema

### Complete Configuration Example

```yaml
# Agent configuration
agent:
  max_turns: 100
  timeout: 10m

  cycle_detection:
    enabled: true
    window_size: 3
    similarity_threshold: 0.8
    tool_repeat_limit: 3
    interventions:
      soft_turn_threshold: 10
      medium_turn_threshold: 30
      hard_turn_threshold: 50

# Context management
context:
  compression:
    enabled: true
    threshold: 0.8
    strategy: "hybrid"
    preserve_critical: true

# LLM configuration
llm:
  provider: ollama
  model: llama2:7b-q4_0

  auto_tune:
    enabled: true
    headroom_mb: 1024
    validate_on_startup: true

# Security configuration
security:
  approval:
    enabled: true
    timeout: 60s
    audit_log: ~/.spin/approval_audit.jsonl  # Optional

# UI configuration
ui:
  status_bar:
    enabled: true
    compact_width: 60
    update_interval: 100ms

# Logging configuration
logging:
  level: info
  file: /var/log/spin/spin.log
```

---

## Success Metrics

### Feature Adoption
- [ ] 80%+ users have status bar enabled
- [ ] Context compression triggers in >50% of long conversations
- [ ] VRAM auto-tuning used by >70% of local model users
- [ ] Cycle detection prevents at least 1 loop per 100 conversations

### Quality Metrics
- [ ] Test coverage: >90% for all new code
- [ ] Zero critical bugs in production
- [ ] <5 major bugs reported in first month
- [ ] User satisfaction: >4/5 in surveys

### Performance Metrics
- [ ] All SLOs met in production
- [ ] <5% CPU overhead from new features
- [ ] <100MB memory overhead
- [ ] No user-reported performance regressions

---

## Dependencies and Timeline

### Dependencies Matrix

| Feature | Depends On | Can Start After |
|---------|-----------|-----------------|
| **Status Bar** | None | Week 1 |
| **Context Compression** | None | Week 1 |
| **VRAM Auto-Tuning** | None | Week 1 |
| **Cycle Detection** | Context Compression (uses compressor) | Week 2 |
| **Approval** | None | Week 1 |
| **Integration** | All features complete | Week 5 |

### Timeline Summary

| Week | Focus | Deliverables |
|------|-------|--------------|
| 1 | Status Bar + Context Compression | StatusBar working, Compression working |
| 2 | VRAM Auto-Tuning + Approval | VRAM tuning working, TUI approval working |
| 3 | Cycle Detection | Cycle detection and interventions working |
| 4 | Feature Completion | All features complete and tested |
| 5 | Integration Testing | All features working together |
| 6 | Production Readiness | Security review, docs, release ready |

---

## Risk Assessment

### High-Risk Items

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| VRAM detection fails on some GPUs | Medium | High | Extensive testing on multiple GPU types; fallback to user config |
| Cycle detection has high false positive rate | Medium | Medium | Tunable thresholds; user can disable; extensive testing |
| Performance regression | Low | Medium | Benchmark tests in CI; performance SLOs |

### Medium-Risk Items

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Context compression loses critical information | Low | High | 100% critical message retention; comprehensive testing |
| Status bar disrupts scrollback | Low | Medium | Factory Droid principle; ANSI escape testing |
| Integration conflicts between features | Low | Medium | Integration testing; careful event handling |

---

**End of Roadmap**

*This roadmap follows SOLID, DRY, KISS principles and Go 1.24 standards. All implementations respect clean architecture and effective Go guidelines.*
