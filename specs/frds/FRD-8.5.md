# FRD-8.5: Comprehensive Testing Suite

**Feature ID:** 8.5
**Feature Name:** Comprehensive Testing Suite
**Package:** `internal/core` (all subpackages)
**Priority:** P0 (Blocker)
**Status:** In Progress
**Estimated Effort:** 24 hours

---

## Overview

Implement comprehensive test coverage for the core module to achieve >90% coverage for critical paths and >85% overall. This includes unit tests, integration tests, end-to-end tests, race condition tests, and benchmark tests.

---

## Business Context

### Problem Statement

The core module currently has 82.7% coverage in the main package, which is below the 85% minimum requirement. While individual subpackages (stream, task, turn) meet or exceed targets, gaps remain in:
- Integration scenarios across packages
- End-to-end conversation flows
- Concurrent execution edge cases
- Performance-critical paths

### Success Criteria

1. **Unit Test Coverage:** >90% for critical paths (conversation, agent, manager)
2. **Overall Coverage:** >85% for all packages in `internal/core`
3. **Integration Tests:** All major flows covered
4. **Race Detection:** `go test -race` passes cleanly
5. **Benchmarks:** Performance baselines established

---

## Definition of Ready (DoR)

- [x] All Phase 0-7 features completed
- [x] Features 8.1, 8.2, 8.3 completed
- [x] Current test infrastructure exists
- [x] Coverage baseline established (82.7%)
- [x] Test utilities available (mock LLM, mock tools)

---

## Requirements

### Functional Requirements

#### FR-8.5.1: Unit Test Coverage
- Achieve >90% coverage for critical packages:
  - `conversation.go` - Conversation lifecycle
  - `agent.go` - Agent orchestration
  - `manager.go` - Manager API
  - `executor.go` - Command execution
  - `validator.go` - Safety validation
  - `context.go` - Environment context
  - `history.go` - History management
  - `planner.go` - Task planning
  - `event.go` - Event system

#### FR-8.5.2: Integration Tests
- Test complete flows across package boundaries:
  - Manager → Conversation → Agent → Executor
  - Session persistence and restoration
  - Event streaming end-to-end
  - Tool execution with real filesystem
  - History truncation with token budgets
  - Multi-turn conversations

#### FR-8.5.3: End-to-End Tests
- Complete conversation scenarios:
  - Create session, run turn, save, resume
  - Multi-turn conversation with context
  - Tool execution with approval workflow
  - Error handling and recovery
  - Graceful cancellation

#### FR-8.5.4: Concurrent Execution Tests
- Race condition detection:
  - Concurrent RunTurn calls (should be prevented)
  - Concurrent event subscriptions
  - Concurrent history access
  - Concurrent session saves
  - Event emitter thread safety

#### FR-8.5.5: Benchmark Tests
- Performance baselines for:
  - Agent.Execute() full loop
  - History.Truncate() with large histories
  - Context.Gather() with large projects
  - Event emission throughput
  - Session serialization/deserialization

#### FR-8.5.6: Error Path Coverage
- Test all error scenarios:
  - Invalid inputs
  - LLM failures
  - Tool execution errors
  - Timeout scenarios
  - Policy violations
  - Storage errors
  - Cancellation handling

### Non-Functional Requirements

#### NFR-8.5.1: Test Quality
- Table-driven tests for complex scenarios
- Descriptive test names following `Test<Type>_<Method>_<Scenario>` convention
- Comprehensive error messages
- Clear test documentation

#### NFR-8.5.2: Test Utilities
- Reusable test helpers in `internal/core/testing/`
- Mock implementations updated and complete
- Test fixtures for common scenarios
- Helper functions to reduce test boilerplate

#### NFR-8.5.3: Test Performance
- Tests complete in reasonable time (<5 seconds per package)
- Benchmarks run in <10 seconds
- No flaky tests

#### NFR-8.5.4: Test Organization
- Tests colocated with source files (`*_test.go`)
- Integration tests in separate files (`*_integration_test.go`)
- Test data in `testdata/` directories
- Shared test utilities in `testing/` package

---

## Technical Design

### Test Structure

```
internal/core/
├── manager_test.go              # Unit tests
├── manager_integration_test.go  # Integration tests
├── conversation_test.go
├── conversation_integration_test.go
├── agent_test.go
├── agent_integration_test.go
├── executor_test.go
├── validator_test.go
├── context_test.go
├── history_test.go
├── planner_test.go
├── event_test.go
├── benchmark_test.go            # Benchmarks for all
├── testdata/                    # Test fixtures
│   ├── sessions/
│   ├── conversations/
│   └── fixtures/
└── testing/                     # Test utilities
    ├── helpers.go              # Test helper functions
    ├── mocks.go                # Additional mocks
    └── fixtures.go             # Fixture builders
```

### Coverage Gaps Analysis

Current coverage by file (from go test -cover):

```bash
# Need to identify specific functions/lines not covered
go test -coverprofile=coverage.out ./internal/core/...
go tool cover -func=coverage.out | grep -v "100.0%"
```

### Test Categories

#### 1. Unit Tests
Focus on isolated component behavior:
- Input validation
- State transitions
- Error handling
- Edge cases

#### 2. Integration Tests
Focus on component interactions:
- Manager + Conversation + Agent
- Agent + Executor + Validator
- History + Context + Event
- Session + Storage + Manager

#### 3. End-to-End Tests
Focus on complete user flows:
- Simple conversation (create, run, complete)
- Multi-turn conversation (context maintenance)
- Conversation resume (persistence)
- Tool execution with approval
- Error recovery

#### 4. Race Tests
Focus on concurrency safety:
```go
// Run with: go test -race
func TestConversation_ConcurrentTurns(t *testing.T)
func TestEventEmitter_ConcurrentSubscribers(t *testing.T)
func TestHistory_ConcurrentAccess(t *testing.T)
```

#### 5. Benchmark Tests
Focus on performance measurement:
```go
func BenchmarkAgent_Execute(b *testing.B)
func BenchmarkHistory_Truncate(b *testing.B)
func BenchmarkContext_Gather(b *testing.B)
func BenchmarkSession_Save(b *testing.B)
```

### Test Helpers

```go
// internal/core/testing/helpers.go

// NewTestManager creates a fully configured test manager
func NewTestManager(t *testing.T, opts ...ManagerOption) *Manager

// NewTestConversation creates a test conversation
func NewTestConversation(t *testing.T) *Conversation

// CollectEvents collects all events from a channel with timeout
func CollectEvents(ch <-chan Event, timeout time.Duration) []Event

// AssertEventType checks if an event type is present
func AssertEventType(t *testing.T, events []Event, typ EventType)

// CreateTestSession creates a populated test session
func CreateTestSession(t *testing.T, turns int) *session.Session

// MockLLMWithResponses creates a mock LLM with predefined responses
func MockLLMWithResponses(responses ...string) llm.Provider
```

### Mock Enhancements

Enhance existing mocks to support more test scenarios:

```go
// internal/llm/mock.go

type MockProvider struct {
    // Responses to return
    Responses []CompletionResponse

    // Errors to return
    Errors []error

    // Call history for verification
    Calls []CompletionRequest

    // Delay simulation
    Delay time.Duration

    // Tool calls to inject
    ToolCalls []ToolCall
}
```

---

## Implementation Plan

### Phase 1: Coverage Analysis (2 hours)
1. Generate coverage report for all core packages
2. Identify uncovered functions and branches
3. Prioritize based on criticality
4. Create coverage improvement plan

### Phase 2: Unit Test Gaps (8 hours)
1. Add missing tests for manager.go
2. Add missing tests for conversation.go
3. Add missing tests for agent.go
4. Expand executor.go tests (error paths)
5. Expand validator.go tests (edge cases)
6. Expand context.go tests (large projects)
7. Expand history.go tests (truncation edge cases)
8. Expand planner.go tests (complex plans)
9. Expand event.go tests (subscriber cleanup)

### Phase 3: Integration Tests (6 hours)
1. Manager + Session flow
2. Conversation + Agent + Executor flow
3. Event streaming integration
4. Tool execution integration
5. History + Context integration

### Phase 4: End-to-End Tests (4 hours)
1. Simple conversation scenario
2. Multi-turn with context
3. Resume conversation
4. Error recovery scenarios
5. Cancellation scenarios

### Phase 5: Concurrency Tests (2 hours)
1. Concurrent turn execution (should fail safely)
2. Concurrent event subscriptions
3. Concurrent history modifications
4. Race detector verification

### Phase 6: Benchmark Tests (2 hours)
1. Agent execution benchmarks
2. History truncation benchmarks
3. Context gathering benchmarks
4. Session persistence benchmarks
5. Event throughput benchmarks

---

## Test Examples

### Unit Test Example

```go
// internal/core/manager_test.go

func TestManager_NewConversation(t *testing.T) {
    tests := []struct {
        name    string
        workDir string
        wantErr bool
    }{
        {
            name:    "valid directory",
            workDir: t.TempDir(),
            wantErr: false,
        },
        {
            name:    "nonexistent directory",
            workDir: "/nonexistent/path",
            wantErr: true,
        },
        {
            name:    "empty workdir",
            workDir: "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mgr := NewTestManager(t)
            conv, err := mgr.NewConversation(context.Background(), tt.workDir)

            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, conv)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, conv)
                assert.Equal(t, tt.workDir, conv.session.WorkDir)
            }
        })
    }
}
```

### Integration Test Example

```go
// internal/core/manager_integration_test.go

func TestManager_CompleteConversationFlow(t *testing.T) {
    // Setup
    ctx := context.Background()
    tempDir := t.TempDir()

    mgr := NewTestManager(t,
        WithLLM(MockLLMWithResponse("I'll list the files.")),
        WithStorage(session.NewFileStorage(tempDir)),
    )

    // Create conversation
    conv, err := mgr.NewConversation(ctx, tempDir)
    require.NoError(t, err)

    // Run turn
    err = conv.RunTurn(ctx, "List files")
    require.NoError(t, err)

    // Verify events
    events := CollectEvents(conv.Stream(), 100*time.Millisecond)
    AssertEventType(t, events, EventComplete)

    // Verify session saved
    sessions, err := mgr.ListConversations(ctx, Filter{})
    require.NoError(t, err)
    assert.Len(t, sessions, 1)

    // Resume conversation
    sessionID := conv.session.ID
    conv2, err := mgr.ResumeConversation(ctx, sessionID)
    require.NoError(t, err)
    assert.Equal(t, sessionID, conv2.session.ID)
}
```

### Benchmark Example

```go
// internal/core/benchmark_test.go

func BenchmarkAgent_Execute(b *testing.B) {
    agent := NewTestAgent(b,
        WithLLM(MockLLMWithResponse("Done.")),
    )

    req := Request{
        UserInput: "Simple task",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := agent.Execute(context.Background(), req)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkHistory_Truncate(b *testing.B) {
    // Setup large history
    h := NewHistory(16384, NewSimpleTokenizer())
    for i := 0; i < 100; i++ {
        h.AddMessage(Message{
            Role:    "user",
            Content: strings.Repeat("test ", 100),
        })
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        h.Truncate(8192)
    }
}
```

---

## Acceptance Criteria

### Coverage Metrics
- [ ] Overall core package coverage: ≥85%
- [ ] manager.go coverage: ≥90%
- [ ] conversation.go coverage: ≥90%
- [ ] agent.go coverage: ≥90%
- [ ] All subpackages maintain current coverage (≥85%)

### Test Quality
- [ ] All new tests follow naming conventions
- [ ] All tests have descriptive names
- [ ] Table-driven tests used for complex scenarios
- [ ] Error messages are clear and actionable
- [ ] No test warnings or skipped tests

### Test Execution
- [ ] All tests pass: `go test ./internal/core/...`
- [ ] Race detector clean: `go test -race ./internal/core/...`
- [ ] Benchmarks run successfully: `go test -bench=. ./internal/core/...`
- [ ] Tests complete in <5 seconds per package

### Test Organization
- [ ] Test utilities in `testing/` package
- [ ] Test fixtures in `testdata/` directories
- [ ] Integration tests clearly marked
- [ ] Helper functions reduce test boilerplate

---

## Definition of Done (DoD)

- [ ] Coverage analysis completed and gaps identified
- [ ] Unit tests added to achieve >90% for critical paths
- [ ] Integration tests cover all major flows
- [ ] End-to-end tests validate complete scenarios
- [ ] Race detector tests pass cleanly
- [ ] Benchmark tests establish performance baselines
- [ ] Overall coverage ≥85% verified
- [ ] Critical path coverage ≥90% verified
- [ ] All tests documented with clear descriptions
- [ ] Test utilities enhanced and documented
- [ ] Mock implementations complete and tested
- [ ] CI/CD integration verified (if applicable)
- [ ] Code reviewed and approved
- [ ] ROADMAP updated with completion status

---

## Dependencies

### Required Packages
- `internal/core` (all existing components)
- `internal/llm` (mock provider)
- `internal/tools` (mock tools)
- `internal/security` (for integration tests)

### External Dependencies
- `github.com/stretchr/testify/assert`
- `github.com/stretchr/testify/require`
- Standard library `testing`

---

## Testing Strategy

### Coverage Improvement Plan

1. **Identify gaps:**
   ```bash
   go test -coverprofile=coverage.out ./internal/core/...
   go tool cover -func=coverage.out | grep -v "100.0%"
   ```

2. **Prioritize critical paths:**
   - Manager API (NewConversation, ResumeConversation)
   - Conversation lifecycle (RunTurn, Stop)
   - Agent execution loop
   - Error handling paths

3. **Write tests incrementally:**
   - Start with highest-impact gaps
   - Verify coverage increase after each test
   - Focus on edge cases and error paths

4. **Verify with coverage report:**
   ```bash
   go test -coverprofile=coverage.out ./internal/core/...
   go tool cover -html=coverage.out
   ```

### Test Execution Commands

```bash
# All tests
make test

# With coverage
make test-coverage

# With race detector
make test-race

# Benchmarks
make test-bench

# Specific package
go test ./internal/core/

# Specific test
go test -run TestManager_NewConversation ./internal/core/

# Verbose output
go test -v ./internal/core/...

# Coverage HTML report
go test -coverprofile=coverage.out ./internal/core/...
go tool cover -html=coverage.out -o coverage.html
```

---

## Risk Assessment

### High Risk
- **Flaky Tests:** Concurrent tests may be timing-dependent
  - *Mitigation:* Use proper synchronization, avoid sleep-based timing

- **Coverage Target Difficulty:** Some error paths may be hard to trigger
  - *Mitigation:* Use dependency injection to inject errors

### Medium Risk
- **Test Maintenance:** Large test suite requires ongoing maintenance
  - *Mitigation:* Good test organization, clear documentation

- **Performance Impact:** Many tests may slow down CI
  - *Mitigation:* Parallel test execution, benchmark separate workflow

### Low Risk
- **Test Infrastructure:** Test utilities already exist
  - *Mitigation:* Enhance existing infrastructure incrementally

---

## Success Metrics

### Quantitative
- Overall coverage: ≥85% (currently 82.7%)
- Critical path coverage: ≥90%
- Number of integration tests: ≥5
- Number of end-to-end tests: ≥3
- Benchmark tests: ≥5

### Qualitative
- All major user flows tested
- Concurrent scenarios validated
- Error paths comprehensively tested
- Performance baselines established
- Test code is readable and maintainable

---

## Timeline

- **Phase 1 (Coverage Analysis):** 2 hours
- **Phase 2 (Unit Tests):** 8 hours
- **Phase 3 (Integration Tests):** 6 hours
- **Phase 4 (End-to-End Tests):** 4 hours
- **Phase 5 (Concurrency Tests):** 2 hours
- **Phase 6 (Benchmarks):** 2 hours

**Total Estimated Effort:** 24 hours

---

## References

- [AGENTS.md](../../AGENTS.md) - Testing guidelines
- [ROADMAP.md](../core-module/ROADMAP.md) - Feature roadmap
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)

---

**Feature Owner:** Development Team
**Created:** 2025-10-04
**Last Updated:** 2025-10-04
