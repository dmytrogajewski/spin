# FRD-20251015010000: Cycle Auto-Discovery

**Feature:** Cycle Auto-Discovery  
**Status:** Draft  
**Date:** 2025-10-15  
**Author:** Spin Agent  
**Related:** specs/advanced-features-20251012/ROADMAP.md (Feature 4), RESEARCH.md

---

## 1. Executive Summary

Implement automatic detection and intervention for agent reasoning cycles to prevent infinite loops and improve reliability. Detect repeated patterns in responses, tool calls, errors, and state oscillation, with escalating interventions from reflection prompts to user escalation.

**Impact:** >80% detection of cycles, <5% false positives, >70% successful interventions.

---

## 2. Problem Statement

Agents can enter unproductive loops:
- Repeated similar responses
- Cyclic tool calls without progress
- Oscillating states
- Persistent errors

Current safeguards (MaxTurns, Timeout) are blunt; need intelligent detection/intervention.

---

## 3. Requirements

### Functional
- Detect 4 cycle types: similarity (>80%, last 3), repeated tools (3+), oscillation (A-B-A-B), same error (3+).
- Escalation: soft (inject prompt), medium (force compression), hard (pause + user escalate).
- Integrate into agent loop: check after each response, before tools.
- Emit warning events on detection.
- Configurable thresholds via YAML.

### Non-Functional
- Detection <10ms per check.
- Thread-safe.
- Tests ≥90% coverage, synthetic scenarios.

---

## 4. Design

### Package
internal/core/cycle/
- detector.go: CycleDetector with history snapshots
- patterns.go: Detection funcs (Jaccard similarity, etc.)
- intervention.go: Intervention strategies
- doc.go

### Integration
- Agent records snapshot after LLM response.
- CheckForCycle() before tool exec.
- If detected, apply intervention level based on turns.

---

## 5. Implementation Plan

Phase 1: Detector + Patterns
Phase 2: Interventions
Phase 3: Agent Integration
Phase 4: Testing + Config

---

## 6. Testing

- Unit: Each detection type
- Integration: Synthetic loops
- E2E: Prompts causing cycles

---

**End of FRD**
