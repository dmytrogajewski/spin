# FRD-20251029-003: Spin Manager Integration with ACE

**Feature:** Spin Manager Integration (Feature 9)  
**Roadmap:** ACE (Agentic Context Engineering) - Phase 2  
**Status:** In Development  
**Created:** 2025-10-29  
**Author:** Spin Agent  

---

## 1. Executive Summary

This FRD defines the integration of ACE (Agentic Context Engineering) into Spin's Manager component, enabling the coding agent to leverage context playbooks during task execution, tool usage, and code generation.

**Key Objectives:**
- Inject ACE context into Agent conversation flow
- Hook into tool execution lifecycle for feedback collection
- Track coding patterns and outcomes automatically
- Enable ItemizedLearning workflow during agent execution
- Add TUI visualization for ACE status

**Success Metrics:**
- Context injection adds <50ms latency (P99)
- 90%+ test coverage on integration code
- All existing Manager tests continue passing
- Zero race conditions with concurrent playbook access

---

## 2. Background

### 2.1 Problem Statement

Currently, Spin's Agent operates with:
- **Short-term memory**: Conversation history (compressed at 80% capacity)
- **No learning**: Each conversation starts fresh with only system prompt
- **No pattern accumulation**: Successful strategies are lost after conversation ends
- **Repeated mistakes**: Same errors occur across different sessions

### 2.2 Solution Approach

ACE integration will provide:
- **Persistent memory**: Playbooks survive across conversations
- **Automatic learning**: Extract lessons from tool execution results
- **Pattern reuse**: Inject relevant strategies from past successes
- **Error prevention**: Learn from failures and avoid repeating them

### 2.3 Prior Work

**Completed (Phase 1):**
- ✅ Bullet data structure (94.3% coverage)
- ✅ Playbook Manager (91.1% coverage)
- ✅ Generator with ItemizedLearning (89.0% coverage)
- ✅ Semantic retrieval (88.9% coverage)
- ✅ Prompt builder (100% coverage)
- ✅ Feedback parser (100% coverage)

**Existing Spin Components:**
- `internal/core/manager.go` - Conversation orchestration (513 lines)
- `internal/core/agent.go` - Agent with service architecture
- `internal/core/services.go` - SecurityService, DetectionService, OrchestrationService
- `internal/core/event.go` - Event system for observability
- `internal/tools/` - Tool registry and execution

---

## 3. Requirements

### 3.1 Functional Requirements

#### FR-1: ACE Service Integration
**Priority:** P0 (Critical)

Manager must integrate ACE as a service:
- Create ACEService during Manager initialization
- Inject ACEService into Agent via builder pattern
- Load playbook from persistent storage on startup
- Save playbook to storage on shutdown or periodically
- Provide configuration for ACE features (enable/disable, playbook path)

**Acceptance Criteria:**
- ✅ ACEService is constructed in Manager.NewManager()
- ✅ Agent has access to ACE via injected service
- ✅ Playbook loads from ~/.spin/ace/playbooks/default.json
- ✅ Playbook saves atomically (crash-safe)
- ✅ ACE can be disabled via config (fallback to no-op service)
- ✅ All existing tests pass after integration

#### FR-2: Context Injection at Agent Level
**Priority:** P0 (Critical)

Agent must inject ACE context into LLM calls:
- Retrieve relevant bullets based on current task/query
- Build enhanced system prompt with ItemizedLearning instructions
- Inject top-K bullets into system prompt (K=5 default, configurable)
- Track which bullets were included in each turn
- Measure injection latency and log warnings if >50ms

**Acceptance Criteria:**
- ✅ Agent.ProcessTurn() retrieves bullets via ACEService
- ✅ System prompt includes ItemizedLearning instructions
- ✅ Top-5 relevant bullets appear in prompt
- ✅ Bullet IDs are logged in turn metadata
- ✅ Injection latency monitored via metrics
- ✅ Fallback to standard prompt if ACE fails

#### FR-3: Feedback Collection from Tool Execution
**Priority:** P0 (Critical)

Agent must collect feedback signals from tool execution:
- Parse LLM responses for HELPFUL/HARMFUL markers
- Track tool execution success/failure
- Record test execution outcomes (pass/fail counts)
- Capture build/lint errors and associate with bullets
- Update bullet counters based on feedback

**Execution signals to track:**
1. **Test results**: `go test` exit code, pass/fail counts from output
2. **Build results**: `go build` success/failure, error messages
3. **Lint results**: `make lint` success/failure, error counts
4. **Tool errors**: File not found, permission denied, syntax errors

**Acceptance Criteria:**
- ✅ Agent parses HELPFUL/HARMFUL markers via FeedbackParser
- ✅ Tool execution events include success/failure status
- ✅ Test results extracted from shell command output
- ✅ Bullet counters updated via Playbook.UpdateBullet()
- ✅ Feedback parsing errors logged but don't block execution

#### FR-4: Trajectory Recording
**Priority:** P1 (High)

Agent must record execution trajectories for future learning:
- Create Trajectory structure for each conversation turn
- Record all tool calls with parameters
- Capture tool execution results (output, exit code, duration)
- Store user feedback (approval/denial of dangerous commands)
- Save trajectories to disk for offline analysis

**Trajectory structure:**
```go
type Trajectory struct {
    ID           string                 // UUID
    ConversationID string               // Links to conversation
    TurnID       string                 // Links to turn
    Query        string                 // User's request
    BulletsUsed  []string               // Bullet IDs injected
    Steps        []TrajectoryStep       // Execution trace
    Outcome      TrajectoryOutcome      // Success/Failure/Partial
    CreatedAt    time.Time
}

type TrajectoryStep struct {
    ToolName     string                 // e.g., "shell_command"
    Parameters   map[string]interface{} // Tool inputs
    Result       ToolResult             // Output, error, exit code
    Duration     time.Duration
    Timestamp    time.Time
}

type TrajectoryOutcome string
const (
    OutcomeSuccess TrajectoryOutcome = "success"
    OutcomeFailure TrajectoryOutcome = "failure"
    OutcomePartial TrajectoryOutcome = "partial"
)
```

**Acceptance Criteria:**
- ✅ Trajectory created at turn start
- ✅ Each tool execution appends a step
- ✅ Outcome determined at turn end (based on errors, test results)
- ✅ Trajectories saved to ~/.spin/ace/trajectories/{date}/
- ✅ Trajectories can be loaded for offline analysis
- ✅ 90%+ coverage on trajectory recording logic

#### FR-5: ItemizedLearning Workflow Integration
**Priority:** P1 (High)

Agent must execute ItemizedLearning workflow per conversation turn:
1. **Retrieve**: Get relevant bullets from playbook (semantic search)
2. **Inject**: Build system prompt with bullets
3. **Execute**: Call LLM and execute tools
4. **Parse**: Extract HELPFUL/HARMFUL markers from response
5. **Update**: Increment bullet counters based on feedback

**Workflow timing:**
- Retrieve: Before LLM call
- Inject: During system prompt construction
- Execute: Normal agent execution
- Parse: After LLM response received
- Update: After turn completion (async to not block UI)

**Acceptance Criteria:**
- ✅ Agent uses Generator.ItemizedLearning() per turn
- ✅ Retrieval happens before LLM call (<10ms target)
- ✅ Injection adds bullets to system prompt
- ✅ Feedback parsed from LLM response
- ✅ Bullet updates happen asynchronously
- ✅ Errors in workflow logged but don't crash agent
- ✅ Workflow can be disabled via config flag

#### FR-6: Configuration Support
**Priority:** P1 (High)

Add ACE configuration to Spin's config system:
```yaml
ace:
  enabled: true
  playbook_path: "~/.spin/ace/playbooks/default.json"
  trajectory_path: "~/.spin/ace/trajectories/"
  retrieval:
    top_k: 5              # Number of bullets to inject
    min_score: 0.3        # Minimum similarity threshold
  itemized_learning:
    enabled: true
    parse_feedback: true  # Parse HELPFUL/HARMFUL markers
    update_async: true    # Update bullets asynchronously
  generation:
    enabled: false        # Bullet generation (Phase 3)
    auto_reflect: false   # Automatic reflection (Phase 3)
```

**Acceptance Criteria:**
- ✅ Config struct defined in `internal/config/ace.go`
- ✅ Config loaded from YAML/TOML/JSON
- ✅ Defaults provided if config missing
- ✅ Config validation (e.g., top_k > 0, min_score in [0,1])
- ✅ Config can be overridden via environment variables

#### FR-7: TUI Integration and Visualization
**Priority:** P2 (Medium)

Add ACE status indicators to Spin's TUI:
- Show ACE enabled/disabled status in status bar
- Display bullet count in playbook
- Show which bullets were used in current turn (hover/detail view)
- Indicate when bullets are being retrieved (loading indicator)
- Show feedback being collected (HELPFUL/HARMFUL markers)

**Status bar additions:**
```
┌─ Status ────────────────────────────────────────────────────────┐
│ Spin v0.1.0 | ACE: ✓ (247 bullets) | Last updated: 2m ago      │
└─────────────────────────────────────────────────────────────────┘
```

**Bullet usage display (in turn details):**
```
┌─ Turn #5 ─────────────────────────────────────────────────────┐
│ User: How do I implement a mutex-protected map?               │
│                                                                │
│ ACE: 3 bullets retrieved (0.7, 0.6, 0.5 similarity)           │
│   • bullet-123: Use sync.RWMutex for concurrent maps          │
│   • bullet-456: Avoid nested lock acquisitions (deadlock)     │
│   • bullet-789: Table-driven tests for concurrent code        │
│                                                                │
│ Assistant: [response with ItemizedLearning instructions]       │
└────────────────────────────────────────────────────────────────┘
```

**Acceptance Criteria:**
- ✅ Status bar shows ACE status
- ✅ Turn details show retrieved bullets
- ✅ Loading indicators during retrieval
- ✅ Feedback markers visible in assistant response
- ✅ Bullet click/hover shows full content (future: interactive UI)

---

### 3.2 Non-Functional Requirements

#### NFR-1: Performance
**Priority:** P0 (Critical)

- Bullet retrieval: <10ms P99 (semantic search over 1000 bullets)
- Context injection: <50ms P99 total latency added to LLM call
- Playbook save: <100ms (atomic write)
- Playbook load: <500ms (on startup)
- Trajectory save: Async, <50ms per trajectory
- Memory overhead: <50MB for 500-bullet playbook

**Measurement:**
- Add metrics to track latency per operation
- Log warnings if thresholds exceeded
- Include performance tests in test suite

#### NFR-2: Reliability
**Priority:** P0 (Critical)

- ACE failures must not crash Agent
- Playbook corruption recovery (validate on load, restore from backup)
- Graceful degradation if ACE unavailable (fallback to standard prompt)
- Race-free concurrent access (all playbook ops use RWMutex)
- Atomic saves (write to temp file, then atomic rename)

#### NFR-3: Testability
**Priority:** P0 (Critical)

- 90%+ code coverage on ACE integration code
- Unit tests for each integration point
- Integration tests with mock playbook
- End-to-end tests with real Agent execution
- Concurrent access tests (race detector)

#### NFR-4: Observability
**Priority:** P1 (High)

- Log all ACE operations at DEBUG level
- Emit events for key actions (retrieval, injection, feedback update)
- Track metrics: bullet count, retrieval latency, injection latency
- Include ACE status in health checks
- Provide debug commands to inspect playbook state

#### NFR-5: Backward Compatibility
**Priority:** P1 (High)

**Note:** As per instructions, we DO NOT maintain backward compatibility. However:
- Existing Spin functionality must continue working
- ACE is additive (doesn't remove existing features)
- ACE can be disabled via config (feature flag)

---

## 4. Architecture

### 4.1 Component Diagram

```
┌──────────────────────────────────────────────────────────────┐
│ Manager                                                       │
│                                                               │
│  ┌──────────────┐      ┌──────────────┐                     │
│  │  NewManager  │─────▶│  ACEService  │                     │
│  └──────────────┘      └──────┬───────┘                     │
│         │                     │                              │
│         │                ┌────▼────────────────┐            │
│         │                │  Playbook Manager   │            │
│         │                │  Generator          │            │
│         │                │  Retriever          │            │
│         │                │  PromptBuilder      │            │
│         │                │  FeedbackParser     │            │
│         │                └─────────────────────┘            │
│         │                                                    │
│  ┌──────▼────────────────────────────────────────────────┐  │
│  │  Agent                                                 │  │
│  │                                                        │  │
│  │  ┌────────────────┐    ┌─────────────────────────┐   │  │
│  │  │  ProcessTurn   │───▶│  ACEService.Retrieve    │   │  │
│  │  └────────────────┘    └─────────────────────────┘   │  │
│  │         │                                             │  │
│  │  ┌──────▼───────┐      ┌─────────────────────────┐   │  │
│  │  │  BuildPrompt │─────▶│  PromptBuilder.Build    │   │  │
│  │  └──────────────┘      └─────────────────────────┘   │  │
│  │         │                                             │  │
│  │  ┌──────▼───────┐                                     │  │
│  │  │  CallLLM     │                                     │  │
│  │  └──────┬───────┘                                     │  │
│  │         │                                             │  │
│  │  ┌──────▼───────────┐  ┌─────────────────────────┐   │  │
│  │  │  ParseFeedback   │─▶│  FeedbackParser.Parse   │   │  │
│  │  └──────────────────┘  └─────────────────────────┘   │  │
│  │         │                                             │  │
│  │  ┌──────▼──────────┐   ┌─────────────────────────┐   │  │
│  │  │  UpdateBullets  │──▶│  Playbook.UpdateBullet  │   │  │
│  │  └─────────────────┘   └─────────────────────────┘   │  │
│  │                                                        │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

### 4.2 Data Flow

**Turn Execution with ACE:**

```
1. User sends message
   ↓
2. Agent.ProcessTurn() starts
   ↓
3. ACEService.Retrieve(query) → top-K bullets
   ↓
4. PromptBuilder.Build(bullets) → enhanced system prompt
   ↓
5. LLM.Complete(prompt) → response with ItemizedLearning
   ↓
6. Agent executes tools (read, write, shell)
   ↓
7. FeedbackParser.Parse(response) → HELPFUL/HARMFUL markers
   ↓
8. Playbook.UpdateBullet(ID, increment) → async counter update
   ↓
9. Trajectory.Save() → persist execution trace
   ↓
10. Agent.ProcessTurn() completes
```

### 4.3 Integration Points

#### 4.3.1 Manager Integration

**File:** `internal/core/manager.go`

```go
type Manager struct {
    // Existing fields
    llmProvider  llm.Provider
    toolRegistry *tools.Registry
    taskRegistry *task.Registry
    emitter      *EventEmitter
    config       *Config
    
    // NEW: ACE service
    aceService   *ACEService
}

func NewManager(cfg *Config, opts ...ManagerOption) (*Manager, error) {
    // Existing initialization
    // ...
    
    // NEW: Initialize ACE service
    aceService, err := NewACEService(cfg.ACE, cfg.WorkDir)
    if err != nil {
        return nil, fmt.Errorf("failed to create ACE service: %w", err)
    }
    
    mgr := &Manager{
        // ...existing fields
        aceService: aceService,
    }
    
    return mgr, nil
}

func (m *Manager) NewConversation(ctx context.Context, workDir string) (*Conversation, error) {
    // Build agent with ACE service
    agent := m.buildAgent(ctx, workDir)
    // ...
}

func (m *Manager) buildAgent(ctx context.Context, workDir string) *Agent {
    return &Agent{
        // ...existing fields
        aceService: m.aceService,  // NEW: Inject ACE
    }
}
```

#### 4.3.2 Agent Integration

**File:** `internal/core/agent.go`

```go
type Agent struct {
    // Existing fields
    llm              llm.Provider
    orchestration    *OrchestrationService
    security         *SecurityService
    detection        *DetectionService
    emitter          *EventEmitter
    
    // NEW: ACE service
    aceService       *ACEService
}

func (a *Agent) ProcessTurn(ctx context.Context, userMsg string) (<-chan Event, error) {
    // NEW: Retrieve bullets from ACE
    bullets, err := a.aceService.Retrieve(ctx, userMsg)
    if err != nil {
        log.Warn("ACE retrieval failed, continuing without bullets", "error", err)
        bullets = nil  // Graceful degradation
    }
    
    // NEW: Build enhanced prompt with bullets
    systemPrompt, err := a.aceService.BuildPrompt(ctx, a.taskMode, bullets)
    if err != nil {
        log.Warn("ACE prompt building failed, using default", "error", err)
        systemPrompt = a.buildDefaultSystemPrompt()
    }
    
    // Existing: Build LLM request
    req := &llm.CompletionRequest{
        Messages: []llm.Message{
            {Role: "system", Content: systemPrompt},
            // ...history messages
            {Role: "user", Content: userMsg},
        },
        Tools: a.taskMode.AllowedTools(),
    }
    
    // Existing: Call LLM
    response, err := a.llm.Complete(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // NEW: Parse feedback from response
    feedback, err := a.aceService.ParseFeedback(response.Content)
    if err != nil {
        log.Debug("No feedback markers found in response")
    }
    
    // NEW: Update bullet counters asynchronously
    if feedback != nil {
        go a.aceService.UpdateBullets(context.Background(), feedback)
    }
    
    // Existing: Execute tools and emit events
    // ...
}
```

#### 4.3.3 ACEService Structure

**File:** `internal/core/ace_service.go` (NEW)

```go
package core

import (
    "context"
    "fmt"
    "time"
    
    "github.com/yourorg/spin/internal/ace/bullet"
    "github.com/yourorg/spin/internal/ace/embedding"
    "github.com/yourorg/spin/internal/ace/feedback"
    "github.com/yourorg/spin/internal/ace/generator"
    "github.com/yourorg/spin/internal/ace/playbook"
    "github.com/yourorg/spin/internal/ace/prompt"
    "github.com/yourorg/spin/internal/ace/retrieval"
    "github.com/yourorg/spin/internal/config"
)

// ACEService provides ACE functionality to Agent
type ACEService struct {
    config       *config.ACEConfig
    playbook     *playbook.Playbook
    generator    *generator.Generator
    retriever    *retrieval.SemanticRetriever
    promptBuilder *prompt.PromptBuilder
    feedbackParser *feedback.RegexParser
    
    // Trajectory recording
    trajectories map[string]*generator.Trajectory
    trajectoryDir string
}

func NewACEService(cfg *config.ACEConfig, workDir string) (*ACEService, error) {
    if !cfg.Enabled {
        return NewNoOpACEService(), nil
    }
    
    // Load playbook from disk
    pb, err := playbook.LoadFromFile(cfg.PlaybookPath)
    if err != nil {
        // Create new playbook if file doesn't exist
        pb = playbook.NewPlaybook()
    }
    
    // Create embedder (mock for now, real in Phase 3)
    embedder := embedding.NewMockEmbedder()
    
    // Create components
    retriever := retrieval.NewSemanticRetriever(pb, embedder)
    promptBuilder := prompt.NewPromptBuilder()
    feedbackParser := feedback.NewRegexParser()
    gen := generator.NewGenerator(pb, retriever, promptBuilder, feedbackParser)
    
    return &ACEService{
        config:         cfg,
        playbook:       pb,
        generator:      gen,
        retriever:      retriever,
        promptBuilder:  promptBuilder,
        feedbackParser: feedbackParser,
        trajectories:   make(map[string]*generator.Trajectory),
        trajectoryDir:  cfg.TrajectoryPath,
    }, nil
}

// Retrieve fetches top-K relevant bullets for given query
func (s *ACEService) Retrieve(ctx context.Context, query string) ([]*bullet.Bullet, error) {
    results, err := s.retriever.Retrieve(ctx, query, s.config.Retrieval.TopK)
    if err != nil {
        return nil, fmt.Errorf("retrieval failed: %w", err)
    }
    
    // Filter by minimum score threshold
    filtered := make([]*bullet.Bullet, 0, len(results))
    for _, result := range results {
        if result.Score >= s.config.Retrieval.MinScore {
            filtered = append(filtered, result.Bullet)
        }
    }
    
    return filtered, nil
}

// BuildPrompt constructs system prompt with bullets
func (s *ACEService) BuildPrompt(ctx context.Context, taskMode Task, bullets []*bullet.Bullet) (string, error) {
    systemPrompt := s.promptBuilder.BuildSystemPrompt(taskMode.Name(), taskMode.Instructions())
    
    if len(bullets) > 0 {
        ilPrompt := s.promptBuilder.BuildItemizedLearning(bullets)
        systemPrompt = systemPrompt + "\n\n" + ilPrompt
    }
    
    return systemPrompt, nil
}

// ParseFeedback extracts HELPFUL/HARMFUL markers from LLM response
func (s *ACEService) ParseFeedback(response string) ([]*generator.BulletFeedback, error) {
    return s.feedbackParser.Parse(response)
}

// UpdateBullets increments bullet counters based on feedback
func (s *ACEService) UpdateBullets(ctx context.Context, feedback []*generator.BulletFeedback) error {
    for _, fb := range feedback {
        bullet, err := s.playbook.GetBullet(fb.BulletID)
        if err != nil {
            continue  // Skip if bullet not found
        }
        
        if fb.Helpful {
            bullet.IncrementHelpful()
        } else {
            bullet.IncrementHarmful()
        }
        
        if err := s.playbook.UpdateBullet(bullet); err != nil {
            return fmt.Errorf("failed to update bullet %s: %w", fb.BulletID, err)
        }
    }
    
    // Save playbook asynchronously
    if s.config.ItemizedLearning.UpdateAsync {
        go s.savePlaybook()
    } else {
        if err := s.savePlaybook(); err != nil {
            return err
        }
    }
    
    return nil
}

func (s *ACEService) savePlaybook() error {
    return s.playbook.SaveToFile(s.config.PlaybookPath)
}

// NoOpACEService provides a no-op implementation when ACE is disabled
type NoOpACEService struct{}

func NewNoOpACEService() *ACEService {
    return &ACEService{
        config: &config.ACEConfig{Enabled: false},
    }
}

func (s *NoOpACEService) Retrieve(ctx context.Context, query string) ([]*bullet.Bullet, error) {
    return nil, nil
}

func (s *NoOpACEService) BuildPrompt(ctx context.Context, taskMode Task, bullets []*bullet.Bullet) (string, error) {
    return "", nil
}

func (s *NoOpACEService) ParseFeedback(response string) ([]*generator.BulletFeedback, error) {
    return nil, nil
}

func (s *NoOpACEService) UpdateBullets(ctx context.Context, feedback []*generator.BulletFeedback) error {
    return nil
}
```

---

## 5. Implementation Plan

### 5.1 Phase 1: ACEService Foundation (Week 1, Days 1-2)

**Tasks:**
1. Create `internal/core/ace_service.go` with ACEService struct
2. Implement NewACEService() with playbook loading
3. Implement Retrieve() method (delegates to SemanticRetriever)
4. Implement BuildPrompt() method (delegates to PromptBuilder)
5. Implement ParseFeedback() method (delegates to FeedbackParser)
6. Implement UpdateBullets() method (increments counters, saves playbook)
7. Implement NoOpACEService for when ACE is disabled

**Tests:**
- `ace_service_test.go` - Unit tests for each method
- Mock playbook, retriever, parser
- Test graceful degradation (errors don't crash)
- Test no-op service returns nil without errors

**Deliverable:** ACEService with 90%+ coverage, all unit tests passing

---

### 5.2 Phase 2: Configuration Support (Week 1, Days 3-4)

**Tasks:**
1. Create `internal/config/ace.go` with ACEConfig struct
2. Add ACE config to main Config struct
3. Implement config loading from YAML/TOML/JSON
4. Add environment variable overrides (ACE_ENABLED, ACE_PLAYBOOK_PATH, etc.)
5. Implement config validation
6. Add default config values

**Tests:**
- `ace_config_test.go` - Config loading and validation
- Test defaults, overrides, invalid values

**Deliverable:** ACE configuration working, 90%+ coverage

---

### 5.3 Phase 3: Manager Integration (Week 1, Days 5-6)

**Tasks:**
1. Add aceService field to Manager struct
2. Initialize ACEService in NewManager()
3. Inject ACEService into Agent via buildAgent()
4. Add ACE shutdown logic (save playbook on exit)
5. Update Manager tests to include ACE

**Tests:**
- Update existing Manager tests
- Add ACE-specific Manager tests
- Test with ACE enabled/disabled
- Test playbook persistence across restarts

**Deliverable:** Manager integrates ACE, all tests passing

---

### 5.4 Phase 4: Agent Integration (Week 1-2, Days 7-10)

**Tasks:**
1. Add aceService field to Agent struct
2. Modify ProcessTurn() to call ACEService.Retrieve()
3. Modify buildSystemPrompt() to call ACEService.BuildPrompt()
4. Add feedback parsing after LLM response
5. Add bullet update logic (async)
6. Add error handling and logging
7. Add metrics for retrieval/injection latency

**Tests:**
- `agent_ace_test.go` - Integration tests
- Test full ItemizedLearning workflow
- Test with mock playbook containing known bullets
- Test feedback parsing and bullet updates
- Test graceful degradation on ACE errors
- Test latency is within bounds (<50ms)

**Deliverable:** Agent uses ACE, 90%+ coverage on new code

---

### 5.5 Phase 5: Trajectory Recording (Week 2, Days 11-12)

**Tasks:**
1. Create `internal/core/trajectory.go` with Trajectory structs
2. Add trajectory creation in Agent.ProcessTurn()
3. Add trajectory step recording for each tool execution
4. Implement trajectory save to disk (JSON)
5. Add trajectory loading utilities

**Tests:**
- `trajectory_test.go` - Unit tests
- Test trajectory creation and step recording
- Test save/load from disk
- Test concurrent trajectory recording

**Deliverable:** Trajectories recorded and saved, 90%+ coverage

---

### 5.6 Phase 6: TUI Integration (Week 2, Days 13-14)

**Tasks:**
1. Add ACE status to status bar (enabled, bullet count)
2. Add bullet retrieval indicators (loading spinner)
3. Add bullet usage display in turn details
4. Add feedback marker highlighting in response
5. Add ACE debug command (/ace status, /ace bullets)

**Tests:**
- TUI snapshot tests (if applicable)
- Manual testing with visual inspection

**Deliverable:** TUI shows ACE status, user-visible

---

### 5.7 Phase 7: End-to-End Testing (Week 2, Day 15)

**Tasks:**
1. Create E2E test: Agent with ACE solves coding task
2. Create E2E test: Bullet counters update correctly
3. Create E2E test: Playbook persists across restarts
4. Create E2E test: Graceful degradation on ACE failure
5. Run full test suite with race detector
6. Run performance benchmarks

**Tests:**
- `e2e_ace_test.go` - Full integration tests
- Benchmark latency overhead

**Deliverable:** All E2E tests passing, performance validated

---

## 6. Testing Strategy

### 6.1 Unit Tests

**Coverage targets: 90%+**

Files to test:
- `ace_service.go` - ACEService methods
- `ace_config.go` - Config loading and validation
- `trajectory.go` - Trajectory recording

Test scenarios:
- ✅ ACEService methods with valid inputs
- ✅ ACEService methods with nil/empty inputs
- ✅ ACEService methods with errors (graceful degradation)
- ✅ Config loading with valid/invalid YAML
- ✅ Config validation (boundary conditions)
- ✅ Trajectory recording and serialization

### 6.2 Integration Tests

**Coverage targets: 90%+**

Test scenarios:
- ✅ Manager creates ACEService and injects into Agent
- ✅ Agent retrieves bullets and builds prompt
- ✅ Agent parses feedback and updates bullets
- ✅ Playbook saves and loads correctly
- ✅ ACE disabled mode works (no-op service)
- ✅ Concurrent access to playbook (race detector)

### 6.3 End-to-End Tests

**Coverage: Key user scenarios**

Test scenarios:
- ✅ Agent completes coding task with ACE guidance
- ✅ Bullet counters increase after helpful usage
- ✅ Bullet counters decrease after harmful usage
- ✅ Playbook persists across Agent restarts
- ✅ ACE works with different task modes
- ✅ Performance is within bounds (<50ms overhead)

### 6.4 Performance Tests

**Benchmarks:**
- Bullet retrieval latency (target <10ms)
- Context injection latency (target <50ms)
- Playbook save latency (target <100ms)
- Memory usage with 500-bullet playbook (target <50MB)

---

## 7. Metrics and Observability

### 7.1 Metrics to Track

**ACE Operations:**
- `ace_retrieval_latency_ms` - Histogram of retrieval times
- `ace_injection_latency_ms` - Histogram of injection times
- `ace_playbook_save_latency_ms` - Histogram of save times
- `ace_bullet_count` - Gauge of total bullets in playbook
- `ace_feedback_parsed_count` - Counter of feedback markers parsed
- `ace_bullet_updates_count` - Counter of bullet updates
- `ace_errors_count` - Counter of ACE errors (by type)

**Performance:**
- `ace_retrieval_p50`, `ace_retrieval_p99` - Percentiles
- `ace_injection_p50`, `ace_injection_p99` - Percentiles

### 7.2 Logging

**Log levels:**
- DEBUG: All ACE operations (retrieval, injection, feedback)
- INFO: Playbook load/save, bullet updates
- WARN: ACE errors (graceful degradation)
- ERROR: Critical failures (playbook corruption)

**Log fields:**
- `component`: "ace"
- `operation`: "retrieve", "inject", "parse", "update"
- `latency_ms`: Duration
- `bullet_count`: Number of bullets involved
- `error`: Error message if applicable

### 7.3 Events

**New event types:**
- `EventACERetrievalStart` - Bullet retrieval started
- `EventACERetrievalComplete` - Bullets retrieved (includes count, latency)
- `EventACEInjectionComplete` - Bullets injected into prompt
- `EventACEFeedbackParsed` - Feedback markers parsed
- `EventACEBulletsUpdated` - Bullet counters updated
- `EventACEPlaybookSaved` - Playbook saved to disk

---

## 8. Risks and Mitigations

### 8.1 Technical Risks

**Risk 1: Performance Degradation**
- **Impact:** Agent feels slower to users
- **Probability:** Medium
- **Mitigation:** Strict latency targets (<50ms), async bullet updates, caching
- **Contingency:** Make ACE optional, add "fast mode" that skips retrieval

**Risk 2: Playbook Corruption**
- **Impact:** Loss of learned knowledge
- **Probability:** Low
- **Mitigation:** Atomic writes, validation on load, daily backups
- **Contingency:** Restore from backup, start with empty playbook

**Risk 3: Memory Growth**
- **Impact:** High memory usage with large playbooks
- **Probability:** Medium
- **Mitigation:** Monitor playbook size, add pruning in Phase 3
- **Contingency:** Limit playbook to 1000 bullets, auto-prune oldest

**Risk 4: Race Conditions**
- **Impact:** Data corruption, crashes
- **Probability:** Low
- **Mitigation:** RWMutex on all playbook ops, race detector in CI
- **Contingency:** Add more granular locking, use channels for updates

### 8.2 Integration Risks

**Risk 5: Breaking Existing Tests**
- **Impact:** CI fails, development blocked
- **Probability:** High
- **Mitigation:** Run full test suite after each change, keep ACE additive
- **Contingency:** Feature flag to disable ACE, revert if tests fail

**Risk 6: TUI Performance Impact**
- **Impact:** UI lag, poor user experience
- **Probability:** Medium
- **Mitigation:** Async bullet updates, debounce UI updates, virtualization
- **Contingency:** Disable TUI indicators, show ACE status in CLI only

---

## 9. Success Criteria

### 9.1 Functional Success

- ✅ ACE service integrated into Manager and Agent
- ✅ Bullets retrieved and injected per turn
- ✅ Feedback parsed and bullet counters updated
- ✅ Playbook persists across restarts
- ✅ Trajectories recorded and saved
- ✅ TUI shows ACE status
- ✅ Configuration works (enable/disable, paths, thresholds)

### 9.2 Quality Success

- ✅ 90%+ code coverage on new ACE integration code
- ✅ All existing tests pass
- ✅ Race detector clean (`go test -race`)
- ✅ Linter passes (`make lint`)
- ✅ No deadcode (UAST/herr analysis)

### 9.3 Performance Success

- ✅ Retrieval latency <10ms P99
- ✅ Injection latency <50ms P99
- ✅ Playbook save <100ms
- ✅ Memory overhead <50MB (500 bullets)
- ✅ Zero latency impact when ACE disabled

### 9.4 User Experience Success

- ✅ TUI shows ACE status clearly
- ✅ Users can see which bullets were used
- ✅ Feedback markers visible in responses
- ✅ ACE debug commands work
- ✅ Documentation explains ACE features

---

## 10. Future Work (Phase 3+)

**Deferred to future phases:**
- Reflector component (analyze trajectories, extract insights)
- Curator component (synthesize deltas, de-duplicate bullets)
- Bullet generation (automatic from errors/successes)
- Offline playbook training (learn from codebases)
- Real embedding model integration (replace mock)
- Advanced TUI features (interactive bullet editing)
- Cross-project playbook sharing

---

## 11. References

- **ACE Paper:** `specs/ace-agentic-context-engineering/2510.04618v1.pdf`
- **ACE Roadmap:** `specs/ace-agentic-context-engineering/ROADMAP.md`
- **Core Data Structures FRD:** `specs/frds/FRD-20251029-001-core-data-structures.md`
- **Generator Component FRD:** `specs/frds/FRD-20251029-002-generator-component.md`
- **ACE Package Docs:** `docs/packages/ace.md`
- **Spin Architecture:** `docs/architectural-anti-patterns.md`
- **Testing Patterns:** `docs/testing-patterns.md`

---

## 12. Appendix

### 12.1 Example Configuration

**File: `~/.spin/config.yaml`**

```yaml
ace:
  enabled: true
  playbook_path: "~/.spin/ace/playbooks/default.json"
  trajectory_path: "~/.spin/ace/trajectories/"
  
  retrieval:
    top_k: 5
    min_score: 0.3
  
  itemized_learning:
    enabled: true
    parse_feedback: true
    update_async: true
  
  generation:
    enabled: false  # Phase 3
    auto_reflect: false  # Phase 3
```

### 12.2 Example Bullet Usage

**System Prompt with ItemizedLearning:**

```
You are Spin, a pragmatic Golang coding agent.

## Itemized Learning

You have access to a knowledge base of coding strategies. Below are the most relevant strategies for this task:

### BULLET-abc123: Use sync.RWMutex for concurrent maps
When implementing concurrent access to maps, use sync.RWMutex instead of sync.Mutex. RWMutex allows multiple concurrent readers, improving performance when reads outnumber writes.

### BULLET-def456: Avoid nested lock acquisitions
Never acquire a lock while holding another lock. This can lead to deadlocks. Instead, release all locks before acquiring new ones, or use a lock hierarchy.

### BULLET-ghi789: Table-driven tests for concurrent code
When testing concurrent code, use table-driven tests with multiple goroutines and race detector. This helps catch race conditions early.

## Instructions

After using these strategies, mark them as:
- HELPFUL(bullet-id): Strategy was useful
- HARMFUL(bullet-id): Strategy was misleading

Example:
"I implemented the concurrent map using sync.RWMutex as suggested. HELPFUL(abc123)"

Now, please help with the user's request:
```

**LLM Response with Feedback:**

```
I'll help you implement a thread-safe map using sync.RWMutex. HELPFUL(abc123)

```go
type SafeMap struct {
    mu    sync.RWMutex
    data  map[string]interface{}
}

func (m *SafeMap) Get(key string) (interface{}, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    val, ok := m.data[key]
    return val, ok
}
```

I avoided nested lock acquisitions as per the guideline. HELPFUL(def456)

For testing, I recommend table-driven tests with the race detector enabled. HELPFUL(ghi789)
```

**Feedback Parsing Result:**
- Bullet abc123: helpful_count++
- Bullet def456: helpful_count++
- Bullet ghi789: helpful_count++

---

**Document Version:** 1.0  
**Last Updated:** 2025-10-29  
**Status:** Ready for Implementation  
**Estimated Effort:** 2 weeks (15 person-days)  
