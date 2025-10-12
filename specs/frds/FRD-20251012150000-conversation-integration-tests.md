# FRD-20251012150000: Conversation Integration Tests for Task Modes

**Status**: Draft
**Created**: 2025-10-12
**Author**: Spin Agent
**Related Roadmap**: specs/task-modes/ROADMAP.md - Phase 2, Task P2.3
**Related Spec**: specs/task-modes/specification.md

## Overview

End-to-end integration tests for conversation-level task mode functionality. These tests verify that task modes work correctly throughout the entire conversation lifecycle, including mode switching, tool filtering enforcement, token budget application, and multi-turn persistence.

## Background

Phase 2.1-2.2 successfully implemented conversation-level task mode support:
- ✅ P2.1: Conversation tracking with SetTaskMode() / GetTaskMode()
- ✅ P2.2: Manager support for NewConversationWithTask()

However, we lack comprehensive integration tests that prove the system works end-to-end across real conversation flows. Unit tests validate individual methods, but integration tests must verify:
1. Mode switching affects actual tool availability in live turns
2. Token budgets are applied correctly to LLM calls
3. Mode state persists across multiple turns
4. Concurrent operations are safe
5. Error handling works in realistic scenarios

## Goals

1. **E2E Verification**: Prove task modes work in realistic conversation scenarios
2. **Tool Filtering Proof**: Verify that mode restrictions are enforced in actual LLM calls
3. **Token Budget Proof**: Verify that mode budgets are passed correctly to LLM provider
4. **Persistence Proof**: Verify mode state survives across multiple turns
5. **Concurrency Proof**: Verify thread safety under realistic concurrent access
6. **Coverage**: Achieve ≥85% coverage for conversation.go

## Non-Goals

- UI-level testing (that's Phase 3: CLI integration tests)
- Protocol-level testing (that's Phase 4: AppServer integration tests)
- Performance benchmarking (that's Phase 5: P5.4)
- Custom mode testing (future enhancement)

## Requirements

### Functional Requirements

**FR1**: Integration test shall verify mode switching affects tool availability
**FR2**: Integration test shall verify mode switching affects token budgets
**FR3**: Integration test shall verify mode persists across multiple turns
**FR4**: Integration test shall verify concurrent mode switches don't race
**FR5**: Integration test shall verify invalid mode handling
**FR6**: Integration test shall verify Manager.NewConversationWithTask()

### Non-Functional Requirements

**NFR1**: Tests shall be deterministic (no flakes)
**NFR2**: Tests shall use realistic LLM mocking (not empty stubs)
**NFR3**: Tests shall complete in < 10 seconds total
**NFR4**: Test coverage ≥ 85% for conversation.go
**NFR5**: All tests shall pass with race detector enabled

## Design

### Test Architecture

#### Mock LLM Strategy

We need a sophisticated mock LLM that captures:
1. The tools passed in the request (for tool filtering verification)
2. The max tokens passed in the request (for budget verification)
3. Realistic streaming responses (not just empty data)

```go
// File: internal/core/conversation_test.go

// mockLLMProvider captures LLM requests for verification
type mockLLMProvider struct {
    mu            sync.Mutex
    lastRequest   *llm.CompletionRequest
    responses     []mockResponse
    responseIndex int
}

type mockResponse struct {
    content  string
    toolCall *llm.ToolCall
    err      error
}

func newMockLLM() *mockLLMProvider {
    return &mockLLMProvider{
        responses: []mockResponse{
            {content: "I'll help with that.", toolCall: nil, err: nil},
        },
    }
}

func (m *mockLLMProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
    m.mu.Lock()
    m.lastRequest = &req
    m.mu.Unlock()

    if m.responseIndex >= len(m.responses) {
        return nil, errors.New("no more mock responses")
    }

    resp := m.responses[m.responseIndex]
    m.responseIndex++

    if resp.err != nil {
        return nil, resp.err
    }

    return &llm.CompletionResponse{
        Content: resp.content,
        ToolCalls: func() []llm.ToolCall {
            if resp.toolCall != nil {
                return []llm.ToolCall{*resp.toolCall}
            }
            return nil
        }(),
    }, nil
}

func (m *mockLLMProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
    m.mu.Lock()
    m.lastRequest = &req
    m.mu.Unlock()

    ch := make(chan llm.StreamChunk, 10)
    go func() {
        defer close(ch)

        if m.responseIndex >= len(m.responses) {
            ch <- llm.StreamChunk{Error: errors.New("no more mock responses")}
            return
        }

        resp := m.responses[m.responseIndex]
        m.responseIndex++

        if resp.err != nil {
            ch <- llm.StreamChunk{Error: resp.err}
            return
        }

        // Send content in chunks (realistic streaming)
        words := strings.Fields(resp.content)
        for _, word := range words {
            ch <- llm.StreamChunk{
                Delta: llm.CompletionDelta{
                    Content: word + " ",
                },
            }
        }

        // Send tool call if any
        if resp.toolCall != nil {
            ch <- llm.StreamChunk{
                Delta: llm.CompletionDelta{
                    ToolCalls: []llm.ToolCall{*resp.toolCall},
                },
            }
        }

        // Send done chunk
        ch <- llm.StreamChunk{
            Delta: llm.CompletionDelta{
                FinishReason: "stop",
            },
        }
    }()

    return ch, nil
}

// Helper to inspect last request
func (m *mockLLMProvider) getLastRequest() llm.CompletionRequest {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.lastRequest == nil {
        return llm.CompletionRequest{}
    }
    return *m.lastRequest
}

func (m *mockLLMProvider) setResponses(responses []mockResponse) {
    m.mu.Lock()
    m.responses = responses
    m.responseIndex = 0
    m.mu.Unlock()
}
```

### Integration Tests

#### Test 1: Mode Switch Affects Tool Availability

**Purpose**: Prove that switching from regular to review mode restricts tools in actual LLM calls.

```go
func TestConversation_Integration_ModeSwitchAffectsTools(t *testing.T) {
    // Setup
    mockLLM := newMockLLM()
    agent := setupTestAgentWithLLM(t, mockLLM)
    conv := setupTestConversation(t, agent)

    // Turn 1: Regular mode (all tools)
    mockLLM.setResponses([]mockResponse{
        {content: "Reading file", toolCall: &llm.ToolCall{
            Function: llm.FunctionCall{
                Name:      "read_file",
                Arguments: `{"path": "test.go"}`,
            },
        }},
        {content: "Done", toolCall: nil},
    })

    events, err := conv.SendMessage(context.Background(), "Read test.go", nil)
    require.NoError(t, err)
    drainEvents(events)

    // Verify regular mode had all tools
    req1 := mockLLM.getLastRequest()
    toolNames1 := extractToolNames(req1.Tools)
    assert.Contains(t, toolNames1, "read_file")
    assert.Contains(t, toolNames1, "write_file")
    assert.Contains(t, toolNames1, "execute_command")

    // Switch to review mode
    err = conv.SetTaskMode("review")
    require.NoError(t, err)

    // Turn 2: Review mode (read-only tools)
    mockLLM.setResponses([]mockResponse{
        {content: "Reading file", toolCall: &llm.ToolCall{
            Function: llm.FunctionCall{
                Name:      "read_file",
                Arguments: `{"path": "test2.go"}`,
            },
        }},
        {content: "Done", toolCall: nil},
    })

    events, err = conv.SendMessage(context.Background(), "Read test2.go", nil)
    require.NoError(t, err)
    drainEvents(events)

    // Verify review mode has only read tools
    req2 := mockLLM.getLastRequest()
    toolNames2 := extractToolNames(req2.Tools)
    assert.Contains(t, toolNames2, "read_file")
    assert.NotContains(t, toolNames2, "write_file", "review mode should not have write_file")
    assert.NotContains(t, toolNames2, "execute_command", "review mode should not have execute_command")
}

func extractToolNames(tools []llm.Tool) []string {
    names := make([]string, len(tools))
    for i, tool := range tools {
        names[i] = tool.Function.Name
    }
    return names
}

func drainEvents(events <-chan Event) {
    for range events {
        // Consume all events
    }
}
```

#### Test 2: Mode Switch Affects Token Budget

**Purpose**: Prove that compact mode applies its 4K token budget to LLM calls.

```go
func TestConversation_Integration_ModeSwitchAffectsTokenBudget(t *testing.T) {
    // Setup
    mockLLM := newMockLLM()
    agent := setupTestAgentWithLLM(t, mockLLM)
    agent.config.MaxTokens = 16384 // Agent default: 16K
    conv := setupTestConversation(t, agent)

    // Turn 1: Regular mode (16K tokens)
    mockLLM.setResponses([]mockResponse{{content: "Done", toolCall: nil}})
    events, err := conv.SendMessage(context.Background(), "Hello", nil)
    require.NoError(t, err)
    drainEvents(events)

    req1 := mockLLM.getLastRequest()
    assert.Equal(t, 16384, req1.MaxTokens, "regular mode should use 16K tokens")

    // Switch to compact mode (4K tokens)
    err = conv.SetTaskMode("compact")
    require.NoError(t, err)

    // Turn 2: Compact mode (4K tokens)
    mockLLM.setResponses([]mockResponse{{content: "Done", toolCall: nil}})
    events, err = conv.SendMessage(context.Background(), "What is 2+2?", nil)
    require.NoError(t, err)
    drainEvents(events)

    req2 := mockLLM.getLastRequest()
    assert.Equal(t, 4096, req2.MaxTokens, "compact mode should use 4K tokens")

    // Switch to planning mode (4K tokens)
    err = conv.SetTaskMode("planning")
    require.NoError(t, err)

    // Turn 3: Planning mode (4K tokens)
    mockLLM.setResponses([]mockResponse{{content: "Done", toolCall: nil}})
    events, err = conv.SendMessage(context.Background(), "Plan the feature", nil)
    require.NoError(t, err)
    drainEvents(events)

    req3 := mockLLM.getLastRequest()
    assert.Equal(t, 4096, req3.MaxTokens, "planning mode should use 4K tokens")
}
```

#### Test 3: Mode Persists Across Multiple Turns

**Purpose**: Prove that setting mode once affects all subsequent turns until changed.

```go
func TestConversation_Integration_ModePersistsAcrossTurns(t *testing.T) {
    // Setup
    mockLLM := newMockLLM()
    agent := setupTestAgentWithLLM(t, mockLLM)
    conv := setupTestConversation(t, agent)

    // Set to review mode
    err := conv.SetTaskMode("review")
    require.NoError(t, err)
    assert.Equal(t, "review", conv.GetTaskMode())

    // Turn 1
    mockLLM.setResponses([]mockResponse{{content: "Turn 1", toolCall: nil}})
    events, err := conv.SendMessage(context.Background(), "Turn 1", nil)
    require.NoError(t, err)
    drainEvents(events)

    // Mode should still be review
    assert.Equal(t, "review", conv.GetTaskMode())
    req1 := mockLLM.getLastRequest()
    toolNames1 := extractToolNames(req1.Tools)
    assert.NotContains(t, toolNames1, "write_file")

    // Turn 2 (no mode change)
    mockLLM.setResponses([]mockResponse{{content: "Turn 2", toolCall: nil}})
    events, err = conv.SendMessage(context.Background(), "Turn 2", nil)
    require.NoError(t, err)
    drainEvents(events)

    // Mode should STILL be review
    assert.Equal(t, "review", conv.GetTaskMode())
    req2 := mockLLM.getLastRequest()
    toolNames2 := extractToolNames(req2.Tools)
    assert.NotContains(t, toolNames2, "write_file")

    // Turn 3 (no mode change)
    mockLLM.setResponses([]mockResponse{{content: "Turn 3", toolCall: nil}})
    events, err = conv.SendMessage(context.Background(), "Turn 3", nil)
    require.NoError(t, err)
    drainEvents(events)

    // Mode should STILL be review
    assert.Equal(t, "review", conv.GetTaskMode())
    req3 := mockLLM.getLastRequest()
    toolNames3 := extractToolNames(req3.Tools)
    assert.NotContains(t, toolNames3, "write_file")
}
```

#### Test 4: Concurrent Mode Switches Are Safe

**Purpose**: Prove that concurrent mode switches + turn executions don't race.

```go
func TestConversation_Integration_ConcurrentModeSwitches(t *testing.T) {
    // Setup
    mockLLM := newMockLLM()
    agent := setupTestAgentWithLLM(t, mockLLM)
    conv := setupTestConversation(t, agent)

    // Prepare many responses
    responses := make([]mockResponse, 100)
    for i := range responses {
        responses[i] = mockResponse{
            content: fmt.Sprintf("Response %d", i),
            toolCall: nil,
        }
    }
    mockLLM.setResponses(responses)

    var wg sync.WaitGroup
    ctx := context.Background()
    modes := []string{"regular", "review", "compact", "planning"}

    // Start 50 concurrent turn executions
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            events, err := conv.SendMessage(ctx, fmt.Sprintf("Message %d", i), nil)
            if err != nil {
                t.Logf("Message %d error: %v", i, err)
                return
            }
            drainEvents(events)
        }(i)
    }

    // Start 50 concurrent mode switches
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            mode := modes[i%len(modes)]
            _ = conv.SetTaskMode(mode)
        }(i)
    }

    // Wait for all to complete
    wg.Wait()

    // If we get here without race detector errors, we're good
    // Verify conversation is still in valid state
    currentMode := conv.GetTaskMode()
    assert.Contains(t, modes, currentMode, "final mode should be valid")
}
```

#### Test 5: Manager Creates Conversation With Task

**Purpose**: Prove that Manager.NewConversationWithTask() works correctly.

```go
func TestManager_Integration_NewConversationWithTask(t *testing.T) {
    // Setup manager
    mockLLM := newMockLLM()
    manager := setupTestManagerWithLLM(t, mockLLM)

    // Create conversation in review mode
    conv, err := manager.NewConversationWithTask(context.Background(), "/tmp/test", "review")
    require.NoError(t, err)
    assert.NotNil(t, conv)

    // Verify mode is set
    assert.Equal(t, "review", conv.GetTaskMode())

    // Execute turn and verify review mode is active
    mockLLM.setResponses([]mockResponse{{content: "Reviewing...", toolCall: nil}})
    events, err := conv.SendMessage(context.Background(), "Review code", nil)
    require.NoError(t, err)
    drainEvents(events)

    req := mockLLM.getLastRequest()
    toolNames := extractToolNames(req.Tools)
    assert.NotContains(t, toolNames, "write_file", "review mode should not have write tools")
}
```

#### Test 6: Invalid Mode Handling

**Purpose**: Prove that invalid mode names are handled gracefully.

```go
func TestConversation_Integration_InvalidModeHandling(t *testing.T) {
    // Setup
    mockLLM := newMockLLM()
    agent := setupTestAgentWithLLM(t, mockLLM)
    conv := setupTestConversation(t, agent)

    // Try to set invalid mode
    err := conv.SetTaskMode("invalid-mode-name")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid task mode")

    // Verify mode unchanged (still default)
    assert.Equal(t, "regular", conv.GetTaskMode())

    // Verify subsequent turn works normally
    mockLLM.setResponses([]mockResponse{{content: "Works", toolCall: nil}})
    events, err := conv.SendMessage(context.Background(), "Test", nil)
    require.NoError(t, err)
    drainEvents(events)

    // Should use regular mode
    req := mockLLM.getLastRequest()
    toolNames := extractToolNames(req.Tools)
    assert.Contains(t, toolNames, "write_file", "should still have all tools")
}
```

#### Test 7: All Task Modes End-to-End

**Purpose**: Comprehensive test cycling through all 4 modes.

```go
func TestConversation_Integration_AllTaskModes(t *testing.T) {
    // Setup
    mockLLM := newMockLLM()
    agent := setupTestAgentWithLLM(t, mockLLM)
    conv := setupTestConversation(t, agent)

    testCases := []struct {
        mode              string
        expectedTools     []string
        forbiddenTools    []string
        expectedMaxTokens int
    }{
        {
            mode:              "regular",
            expectedTools:     []string{"read_file", "write_file", "execute_command"},
            forbiddenTools:    []string{},
            expectedMaxTokens: 16384,
        },
        {
            mode:              "review",
            expectedTools:     []string{"read_file", "list_directory", "get_context"},
            forbiddenTools:    []string{"write_file", "execute_command"},
            expectedMaxTokens: 12288,
        },
        {
            mode:              "compact",
            expectedTools:     []string{"read_file", "get_context", "file_search"},
            forbiddenTools:    []string{"write_file", "execute_command"},
            expectedMaxTokens: 4096,
        },
        {
            mode:              "planning",
            expectedTools:     []string{"get_context", "file_search", "git_context"},
            forbiddenTools:    []string{"read_file", "write_file", "execute_command"},
            expectedMaxTokens: 4096,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.mode, func(t *testing.T) {
            // Set mode
            err := conv.SetTaskMode(tc.mode)
            require.NoError(t, err)
            assert.Equal(t, tc.mode, conv.GetTaskMode())

            // Execute turn
            mockLLM.setResponses([]mockResponse{
                {content: fmt.Sprintf("Testing %s mode", tc.mode), toolCall: nil},
            })
            events, err := conv.SendMessage(context.Background(), "Test "+tc.mode, nil)
            require.NoError(t, err)
            drainEvents(events)

            // Verify tools
            req := mockLLM.getLastRequest()
            toolNames := extractToolNames(req.Tools)

            for _, expectedTool := range tc.expectedTools {
                assert.Contains(t, toolNames, expectedTool,
                    "mode %s should have %s", tc.mode, expectedTool)
            }

            for _, forbiddenTool := range tc.forbiddenTools {
                assert.NotContains(t, toolNames, forbiddenTool,
                    "mode %s should NOT have %s", tc.mode, forbiddenTool)
            }

            // Verify token budget
            assert.Equal(t, tc.expectedMaxTokens, req.MaxTokens,
                "mode %s should have %d tokens", tc.mode, tc.expectedMaxTokens)
        })
    }
}
```

### Test Helpers

```go
// setupTestAgentWithLLM creates a test agent with a mock LLM provider
func setupTestAgentWithLLM(t *testing.T, mockLLM llm.Provider) *Agent {
    t.Helper()

    executor := NewExecutor()
    validator := NewValidator()
    ctx := &Environment{WorkDir: "/tmp/test"}
    emitter := NewEventEmitter()

    agent, err := NewAgent(mockLLM, executor, validator, ctx, emitter)
    require.NoError(t, err)

    // Register test tools
    agent.toolRegistry.Register(/* register relevant tools */)

    return agent
}

// setupTestConversation creates a test conversation
func setupTestConversation(t *testing.T, agent *Agent) *Conversation {
    t.Helper()

    history := NewHistory()
    emitter := NewEventEmitter()

    return NewConversation(agent, history, emitter)
}

// setupTestManagerWithLLM creates a test manager
func setupTestManagerWithLLM(t *testing.T, mockLLM llm.Provider) *Manager {
    t.Helper()

    config := &Config{
        // ... test config ...
    }

    manager, err := NewManager(config, WithLLMProvider(mockLLM))
    require.NoError(t, err)

    return manager
}
```

## Test Coverage Analysis

### Target Coverage

- **New Test File**: 100% of integration test code
- **conversation.go**: ≥85% overall (maintain existing + new code)
- **Specific Methods**:
  - SetTaskMode(): 100%
  - GetTaskMode(): 100%
  - getCurrentTask(): 100%
  - sendMessageInternal() (modified): maintain existing coverage

### Coverage Verification

```bash
# Run tests with coverage
go test -race -coverprofile=coverage.out ./internal/core/
go tool cover -func=coverage.out | grep conversation.go

# Verify ≥85% coverage
go tool cover -func=coverage.out | grep "total:" | awk '{print $3}' | sed 's/%//' | \
    awk '{if ($1 >= 85) print "PASS: Coverage is", $1"%"; else print "FAIL: Coverage is", $1"%"}'
```

## Performance Considerations

### Test Execution Time

**Target**: < 10 seconds for all integration tests

**Optimization Strategies**:
1. Use t.Parallel() for independent tests
2. Minimize turn execution count (1-3 turns per test)
3. Use fast mock LLM (no network I/O)
4. Skip heavy setup where possible

### Benchmark Integration Tests

Not required for P2.3, but if tests are slow:

```go
func BenchmarkConversation_Integration_ModeSwitchAffectsTools(b *testing.B) {
    mockLLM := newMockLLM()
    agent := setupTestAgentWithLLM(b, mockLLM)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        conv := setupTestConversation(b, agent)
        _ = conv.SetTaskMode("review")
        // ... execute turn ...
    }
}
// Target: < 100ms per iteration
```

## Migration & Compatibility

### Backward Compatibility

✅ **No Breaking Changes**:
- Tests are additive (new test file or added to existing)
- No changes to production code (only test code)
- Existing tests continue to pass

### Test Organization

**Option 1**: Add to existing `conversation_test.go`
```
internal/core/
├── conversation.go
└── conversation_test.go (add integration tests here)
```

**Option 2**: Create new test file `conversation_integration_test.go`
```
internal/core/
├── conversation.go
├── conversation_test.go (unit tests)
└── conversation_integration_test.go (NEW: integration tests)
```

**Recommendation**: Option 1 (add to existing file) for simpler organization.

## Security Considerations

### Test Isolation

- Each test creates fresh agent/conversation instances
- Tests use isolated temp directories
- No shared state between tests

### Mock LLM Security

- Mock LLM does not call real API (no credential leakage)
- Mock responses are deterministic (no randomness)
- No network I/O (fully local testing)

## Open Questions

1. **Q**: Should we test mode switching during active turn execution?
   **A**: Yes, Test 4 covers concurrent switches. Separate test for "switch during turn" is future work.

2. **Q**: Should we mock tool execution or use real tools?
   **A**: Mock tools for speed. Real tool testing is e2e (Phase 5).

3. **Q**: Should we test event stream contents?
   **A**: Yes, verify EventTypeSystemInfo is emitted on mode switch.

## Acceptance Criteria

- [ ] Test 1: Mode switch affects tool availability ✅
- [ ] Test 2: Mode switch affects token budget ✅
- [ ] Test 3: Mode persists across turns ✅
- [ ] Test 4: Concurrent switches are safe ✅
- [ ] Test 5: Manager.NewConversationWithTask() works ✅
- [ ] Test 6: Invalid mode handling works ✅
- [ ] Test 7: All 4 modes work end-to-end ✅
- [ ] All tests pass with go test -race
- [ ] Test coverage ≥85% for conversation.go
- [ ] Tests complete in < 10 seconds
- [ ] make lint passes with zero errors

## Dependencies

### Blocked By
- P2.1: Add Task Mode to Conversation (complete)
- P2.2: Update Manager for Task Support (complete)

### Blocks
- P3.1: Add Global Task Mode Flag (can start in parallel)
- Phase 3: CLI Integration (needs P2.3 for confidence)

### Related
- P1.5: Integration Tests for Core Agent (similar approach)

## References

- [specs/task-modes/ROADMAP.md](../../task-modes/ROADMAP.md) - Phase 2 roadmap
- [specs/task-modes/specification.md](../../task-modes/specification.md) - Full spec
- [internal/core/conversation.go](../../../internal/core/conversation.go) - Implementation file
- [internal/core/conversation_test.go](../../../internal/core/conversation_test.go) - Test file
- [AGENTS.md](../../../AGENTS.md) - Development standards

## Approval

**Author**: Spin Agent
**Reviewed**: TBD
**Approved**: TBD
**Status**: Draft → Pending Review

---

**Last Updated**: 2025-10-12
