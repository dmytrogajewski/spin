# ACE (Agentic Context Engineering) Integration Roadmap for Spin

## Executive Summary

This roadmap outlines the integration of ACE (Agentic Context Engineering) into **Spin**, our Go-based coding agent. ACE is a framework for evolving contexts that enable self-improving language models through comprehensive playbooks that accumulate, refine, and organize coding strategies, patterns, and domain knowledge.

**Project Goal:** Enhance Spin's coding capabilities by giving it persistent, self-improving memory that learns from every interaction.

**Core Value Propositions for Spin:**
- **Persistent Learning**: Remember successful patterns across conversations
- **Error Prevention**: Learn from mistakes and avoid repeating them
- **Domain Expertise**: Accumulate Go idioms, testing patterns, architecture principles
- **Context Efficiency**: Retrieve only relevant knowledge, not entire history
- **Self-Improvement**: Automatically refine its knowledge base over time

**Integration Philosophy:**
- Build ACE as native Spin components, not standalone system
- Leverage existing Spin architecture (LLM providers, event system, TUI)
- Focus on coding agent use cases, not general research benchmarks
- Prioritize practical features over academic completeness

---

## 📊 Quick Reference: ACE Paper Alignment Status

| ACE Paper Feature | Status | Implementation | Priority | ETA |
|-------------------|--------|----------------|----------|-----|
| **Core Architecture** | | | | |
| Itemized Bullets | ✅ Complete | internal/ace/bullet | - | Done |
| Playbook CRUD | ✅ Complete | internal/ace/playbook | - | Done |
| Generator | ✅ Complete | internal/ace/generator | - | Done |
| Reflector | ✅ Complete | internal/ace/reflector | - | Done |
| Curator | ✅ Complete | internal/ace/curator | - | Done |
| Adapter | ✅ Complete | internal/ace/adapter | - | Done |
| **Query & Retrieval** | | | | |
| Query = Current Input Only | ✅ Complete | playbook/search.go | - | Done |
| Semantic Search (Cosine) | ✅ Complete | retrieval/retriever.go | - | Done |
| Top-K + Score Filter | ✅ Complete | agent/ace_service.go | - | Done |
| **Updates & Refinement** | | | | |
| Incremental Delta Updates | ✅ Complete | delta/* | - | Done |
| Grow-and-Refine | ✅ Complete | refine/* | - | Done |
| Deduplication | ✅ Complete | curator/curator.go | - | Done |
| **Learning Modes** | | | | |
| Online Learning | ✅ Complete | adapter/* | - | Done |
| ItemizedLearning Feedback | ✅ Complete | feedback/* | - | Done |
| **Missing Critical Features** | | | | |
| Multi-Epoch Adaptation | ❌ Missing | - | P1 | Week 1-2 |
| Batch Delta Updates | ❌ Missing | - | P1 | Week 3-4 |
| Full Trajectory Capture | ❌ Missing | - | P1 | Week 5-6 |
| Offline Warmup Training | ❌ Missing | - | P1 | Week 7-8 |
| Convergence Detection | ❌ Missing | - | P2 | Week 9 |
| Performance Benchmarking | ❌ Missing | - | P2 | Week 10-11 |
| **Optional Features** | | | | |
| Go Domain Seeding | ⚠️ Partial | seedInitialBullets (stub) | P3 | Week 12 |
| Observability/TUI | ⚠️ Partial | Basic logging only | P3 | Week 13 |
| KV Cache Reuse | ⏸️ Deferred | LLM provider handles | P4 | Future |

**Legend:**
- ✅ Complete and tested (90%+ coverage)
- ⚠️ Partial implementation exists
- ❌ Missing, required for full alignment
- ⏸️ Deferred or handled by dependencies

**Overall Alignment: ~70% complete** (core infrastructure done, missing scalability & validation features)

---

## Current Status & ACE Paper Alignment

### ✅ What's Implemented (Phase 1-3 Complete)

**Core Infrastructure (100% aligned with paper):**
- ✅ Itemized bullet structure with metadata (ID, helpful/harmful counters, embeddings)
- ✅ Playbook CRUD operations with semantic search
- ✅ Incremental delta updates (prevents context collapse)
- ✅ Grow-and-refine mechanism (GrowthMonitor + Curator)
- ✅ Query formation: Current user input only (NOT full history)
- ✅ Semantic retrieval: Top-K + score filtering via cosine similarity

**Three-Component Architecture (100% aligned with paper):**
- ✅ **Generator**: Bullet generation from tasks/trajectories/feedback/errors (89.0% coverage)
- ✅ **Reflector**: Deep analysis, multi-iteration refinement, insight extraction (95.7% coverage)
- ✅ **Curator**: Deduplication, merging, quality control, refinement modes (88.6% coverage)

**Online Learning (90% aligned with paper):**
- ✅ **Adapter**: Online learning orchestration with decision tree (90.7% coverage)
- ✅ Sequential processing with session management
- ✅ 6 execution signal types (test, build, lint, error, tool_use, user)
- ✅ Labeled and unlabeled feedback support
- ✅ Memory management with auto-pruning

**Integration (Partial):**
- ✅ Agent integration with ACE service (callLLM integration)
- ✅ ItemizedLearning feedback loop (retrieve → inject → parse → update)
- ⚠️ Tool execution feedback (only LLM feedback, not full execution traces)

**Testing & Quality:**
- ✅ 180+ tests across all components (90%+ coverage)
- ✅ Race detector clean, zero lint errors
- ✅ Comprehensive FRD documentation

### ❌ Missing for Full ACE Paper Alignment

**Critical Gaps (Prioritized for Coding Agents):**

1. **🔴 URGENT: Context Window Management** (Not in paper, but critical)
   - Paper: Assumes context fits or truncates demonstrations
   - Current: No message history pruning, no token tracking
   - Impact: **Agent crashes on long conversations, loses all learning**
   - Priority: **P0 - Must fix immediately**

2. **Full Trajectory Capture** (Section 4.3 of paper)
   - Paper: Captures complete execution traces with tool calls, results, errors
   - Current: Only LLM response feedback, missing tool execution traces
   - Impact: Limited learning from tool usage patterns
   - Priority: **P1 - High** (enables richer learning from code execution)

3. **Offline Warmup Training** (Section 4.4 of paper)
   - Paper: Train playbooks on datasets before deployment (offline warmup)
   - Current: Only online learning during conversations
   - Impact: Cannot pre-train on Go codebases or existing projects
   - Priority: **P1 - High** (critical for coding agent bootstrapping)

4. **Multi-Epoch Adaptation** (Section 3, Page 4 of paper)
   - Paper: "ACE further supports multi-epoch adaptation, where the same queries are revisited to progressively strengthen the context."
   - Current: Single-pass only, no epoch tracking
   - Impact: Cannot iteratively improve on training datasets
   - Priority: **P2 - Medium** (useful for offline training)

5. **Batch Delta Updates** (Section 3.1 of paper)
   - Paper: "Multiple deltas can be merged in parallel, enabling batched adaptation at scale"
   - Current: Sequential updates only (one at a time)
   - Impact: Slower offline training on large codebases
   - Priority: **P2 - Medium** (optimization for offline training)

6. **Convergence Detection** (Section 3 of paper)
   - Paper: Stop adaptation when playbook converges (no more improvements)
   - Current: No convergence tracking or early stopping
   - Impact: Wastes compute on ineffective iterations
   - Priority: **P3 - Low** (nice optimization, not critical)

**Nice-to-Have (Lower Priority for Coding Agents):**

7. **Domain-Specific Seeding** (Section 4.1 of paper, Go equivalent)
   - Paper: Seed with domain knowledge (for AppWorld/Finance domains)
   - Current: Empty playbook initialization (seedInitialBullets is empty stub)
   - Priority: **P3 - Low** (useful but can be done via offline training)

**Removed Features (Not Applicable to Coding Agents):**

❌ **Performance Benchmarking Suite** - Removed (research-focused, not needed for production coding agent)
❌ **KV Cache Reuse** - Removed (LLM provider handles this, not our concern)
❌ **Cross-Session Playbook Sharing** - Removed (single-user coding agent, not multi-agent system)

### 📋 Roadmap to Full Alignment (Reprioritized for Coding Agents)

**Phase 4: Critical Fixes & Core Features (5-6 weeks)** ← CURRENT FOCUS
- 🔴 Feature 22: Context Window Management (1 week) - **URGENT**
- Feature 18: Full Trajectory Capture (2 weeks)
- Feature 19: Offline Warmup Training (2 weeks)

**Phase 5: Training Optimizations (3-4 weeks)**
- Feature 16: Multi-Epoch Adaptation (2 weeks)
- Feature 17: Batch Delta Updates (1 week)
- Feature 20: Convergence Detection (1 week) - optional

**Phase 6: Polish & Production (2-3 weeks)**
- Feature 14: Observability & TUI (1 week)
- Feature 10: Go Domain Seeding (1 week) - optional
- Feature 13: Configuration & Extensibility (complete)
- Feature 15: Documentation & Examples (complete)

**Total Time to Production-Ready:** ~10-13 weeks (~2.5-3 months)

**Removed from Roadmap:**
- ❌ Feature 21: Performance Benchmarking (research-focused)
- ❌ Feature 11: KV Cache Optimization (LLM provider handles)
- ❌ Feature 12: Evaluation Framework (academic benchmarks)

### 🎯 Next Immediate Steps (Start Phase 4)

**Week 1: 🔴 URGENT - Context Window Management (Feature 22)**
1. **Day 1-2**: Implement message history sliding window
   - Track token count estimates for messages
   - Prune old messages when approaching limit
   - Keep system prompt + recent N messages
2. **Day 3-4**: Add token budget tracking
   - Estimate tokens for messages + bullets
   - Calculate remaining context budget
   - Warn when nearing limit
3. **Day 5**: Trajectory checkpointing
   - Save partial trajectories every N turns
   - Enable learning from incomplete executions
   - Handle context overflow errors gracefully
4. **Testing & validation**
   - Test with 100+ turn conversations
   - Verify no crashes on long sessions
   - Ensure bullets still learned from partial trajectories

**Week 2-3: Full Trajectory Capture (Feature 18)**
1. Hook into tool execution lifecycle (Read, Write, Edit, Bash)
2. Capture tool arguments, results, errors, timing
3. Build complete TrajectoryStep for each tool call
4. Detect success/failure from exit codes and outputs
5. Privacy filters (redact API keys, secrets)
6. Integration with Reflector for tool pattern learning

**Week 4-5: Offline Warmup Training (Feature 19)**
1. Dataset loader (Go codebase, JSON/JSONL formats)
2. Offline training loop (iterate over examples)
3. CLI command: `spin ace train --dataset ./examples`
4. Warmup playbook validation and quality checks
5. Example training datasets (Go stdlib, popular packages)
6. Documentation and usage guide

**Week 6: Integration & Testing**
1. End-to-end testing with context window management
2. Validate learning from long conversations
3. Test offline training on real Go projects
4. TUI improvements for ACE visibility
5. Documentation updates

## Feature 22: Context Window Management (URGENT)

### Description
Implement context window management to prevent agent crashes on long conversations. Currently, message history grows unbounded, leading to context overflow errors and loss of all learning. This is critical for production coding agents that may have 50+ turn conversations.

### Components

1. **Message History Sliding Window**
   - Track message count and estimated token usage
   - Prune old messages when approaching context limit
   - Preserve: system prompt + initial user query + recent N messages
   - Configurable window size (default: 50 messages or 32K tokens)

2. **Token Budget Tracking**
   - Estimate tokens for messages (avg ~500 tokens/message)
   - Account for bullets injected into system prompt
   - Calculate remaining budget for LLM response
   - Warn when > 80% of context consumed

3. **Trajectory Checkpointing**
   - Save trajectory state every N turns (default: 10)
   - Enable learning from partial trajectories on overflow
   - Persist checkpoints to disk for crash recovery
   - Resume from last checkpoint on errors

4. **Graceful Degradation**
   - Detect context overflow errors from LLM API
   - Trigger emergency trajectory save
   - Learn from partial execution via Reflector
   - Return graceful error to user (not crash)

### Definition of Ready (DoR)
- [x] Problem identified (context overflow crashes agent)
- [x] Impact assessed (loses all learning on long conversations)
- [x] Token estimation strategy decided (avg 500 tokens/message)
- [x] Window size determined (50 messages or 80K tokens)
- [x] Checkpoint interval chosen (every 10 turns)

### Definition of Done (DoD)
- [ ] Sliding window implementation in `executeAgentLoop()`
- [ ] Token budget calculator with estimates
- [ ] Message pruning logic (keep system + recent N)
- [ ] Trajectory checkpoint system (save every 10 turns)
- [ ] Context overflow error detection and handling
- [ ] Learn from partial trajectories on overflow
- [ ] Unit tests for pruning logic (90%+ coverage)
- [ ] Integration tests with 100+ turn conversations
- [ ] Configuration options (window_size, checkpoint_interval)
- [ ] Documentation with token budget guide
- [ ] TUI warning when approaching context limit

### Technical Considerations

**Token Estimation Strategies:**
```go
// Option 1: Simple estimation (fast, approximate)
func estimateTokens(messages []Message) int {
    total := 0
    for _, msg := range messages {
        total += len(msg.Content) / 4 // ~4 chars per token
    }
    return total
}

// Option 2: Use tiktoken library (accurate, slower)
import "github.com/pkoukk/tiktoken-go"
func countTokens(messages []Message, model string) int {
    encoding := tiktoken.EncodingForModel(model)
    // ... precise counting
}
```

**Pruning Strategy:**
```go
func pruneMessages(messages []Message, maxTokens int) []Message {
    // Always keep: system prompt (index 0) + initial user query (index 1)
    preserved := messages[0:2]

    // Estimate tokens for preserved messages
    preservedTokens := estimateTokens(preserved)

    // Calculate how many recent messages we can keep
    remaining := maxTokens - preservedTokens
    recentMessages := keepRecentWithinBudget(messages[2:], remaining)

    return append(preserved, recentMessages...)
}
```

**Checkpoint Format:**
```go
type TrajectoryCheckpoint struct {
    Turn            int
    Messages        []Message
    ToolExecutions  []ToolExecution
    Timestamp       time.Time
    EstimatedTokens int
}
```

**Emergency Learning:**
```go
func (a *Agent) handleContextOverflow(err error, checkpoints []TrajectoryCheckpoint) {
    if !isContextOverflowError(err) {
        return err
    }

    // Build partial trajectory from checkpoints
    partialTrajectory := buildFromCheckpoints(checkpoints)

    // Learn from what we have (better than nothing!)
    if a.aceService != nil {
        a.aceService.GenerateBulletsWithReflection(ctx, partialTrajectory, "partial")
    }

    return gracefulError("Context window exceeded after %d turns", len(checkpoints))
}
```

**Configuration:**
```yaml
# ~/.spin/config.toml
[agent.context_window]
enabled = true
max_messages = 50          # Sliding window size
max_tokens = 80000         # ~80K tokens (leave 48K for response in 128K model)
checkpoint_interval = 10   # Save every 10 turns
warn_threshold = 0.8       # Warn at 80% capacity
```

### Success Metrics
- Agent survives 100+ turn conversations without crashes
- Learning still occurs from partial trajectories (>0 bullets)
- Token estimates within 10% of actual usage
- Performance impact < 5ms per turn (for estimation)
- Users see clear warnings before hitting limits

### Integration Points
- `internal/agent/loop.go:executeAgentLoop()` - Add pruning before each LLM call
- `internal/agent/agent.go:Execute()` - Handle overflow errors gracefully
- `internal/agent/ace_service.go:GenerateBulletsWithReflection()` - Support partial trajectories
- `internal/agent/config.go` - Add context window configuration

---

## Feature 16: Multi-Epoch Adaptation

### Description
Implement multi-epoch training where the same queries are revisited multiple times to progressively strengthen the playbook. This is a core feature from the ACE paper (Section 3, Page 4) that enables iterative improvement.

### Components
1. **Epoch Management**
   - Track current epoch number (1-N)
   - Support configurable max epochs (e.g., 5)
   - Maintain query-to-trajectory mapping per epoch

2. **Query Replay System**
   - Store original queries for replay
   - Re-run same query across multiple epochs
   - Track performance improvement per epoch

3. **Progressive Strengthening**
   - Use playbook from epoch N-1 in epoch N
   - Measure convergence across epochs
   - Adjust adaptation strategy per epoch (e.g., higher threshold in later epochs)

### Definition of Ready (DoR)
- [ ] Epoch configuration schema designed (max_epochs, queries_per_epoch)
- [ ] Query storage format decided (in-memory vs. persistent)
- [ ] Performance tracking per epoch defined
- [ ] Convergence criteria specified
- [ ] Integration with existing Adapter planned

### Definition of Done (DoD)
- [ ] Epoch tracker with current/max tracking
- [ ] Query replay mechanism (re-run same queries)
- [ ] Progressive playbook evolution (epoch N uses playbook from N-1)
- [ ] Performance metrics per epoch (accuracy, bullet count, changes)
- [ ] Early stopping on convergence (optional)
- [ ] Unit tests for epoch management (90%+ coverage)
- [ ] Integration tests with offline training
- [ ] Documentation with multi-epoch examples
- [ ] Configuration guide (epochs in config.toml)

### Technical Considerations
- Store queries in session or persist to disk
- Track which queries have been seen in which epochs
- Support online (conversation) epochs
- Measure improvement: compare epoch N performance vs. N-1
- Stop early if no improvement across 2+ consecutive epochs
- Visualize epoch progression in TUI

### Success Metrics
- Multi-epoch training improves performance by 5%+ over single-epoch
- Convergence detection reduces unnecessary iterations
- Users can configure epochs via config file

---
