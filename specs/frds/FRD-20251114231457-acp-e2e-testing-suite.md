# FRD: ACP E2E Testing Suite

**Feature ID**: FRD-20251114231457  
**Feature**: E2E Testing Suite for ACP Protocol  
**Roadmap Item**: Feature 10.1  
**Status**: In Progress  
**Created**: 2025-11-14

## Overview

Create comprehensive end-to-end (E2E) tests for the ACP protocol implementation. These tests will verify full protocol compliance by using the ACP SDK's client-side connection to test against a running Spin ACP agent.

## Background

Currently, the ACP implementation has:
- ✅ Comprehensive unit tests for all methods
- ✅ Integration tests for individual components
- ❌ No E2E tests that verify the complete protocol flow

E2E tests are critical to ensure:
- Protocol compliance
- End-to-end functionality works correctly
- No regressions in protocol behavior
- Interoperability with ACP clients

## Requirements

### Functional Requirements

1. **E2E Test Framework**
   - Create test framework in `tests/e2e/acp/` directory
   - Use ACP SDK's `acp.NewClientSideConnection()` for client-side testing
   - Start Spin ACP agent as subprocess using `spin acp` command
   - Handle stdio communication (JSON-RPC 2.0)
   - Clean up resources (processes, temp files) after tests

2. **Initialization Flow Tests**
   - Test protocol version negotiation
   - Test capability advertisement
   - Test agent info exchange
   - Test client capability storage

3. **Session Management Tests**
   - Test `NewSession` with working directory
   - Test `NewSession` with MCP servers (stdio transport)
   - Test `LoadSession` (if session storage available)
   - Test session ID generation and storage
   - Test error handling for invalid sessions

4. **Prompt Processing Tests**
   - Test `Prompt` with text content blocks
   - Test `Prompt` with image/audio content blocks
   - Test `Prompt` with resource links
   - Test real-time notifications (agent_message_chunk, tool_call, etc.)
   - Test stop reason mapping
   - Test cancellation during prompt execution

5. **Tool Call Tests**
   - Test tool call notifications
   - Test tool call updates (progress, completion)
   - Test tool call content (diffs, output)
   - Test tool kind mapping
   - Test file location extraction

6. **Cancellation Tests**
   - Test `Cancel` method
   - Test cancellation during prompt execution
   - Test cancellation of permission requests
   - Verify `cancelled` stop reason returned

7. **Permission Request Tests**
   - Test `RequestPermission` method
   - Test permission option handling (allow_once, allow_always, etc.)
   - Test cancellation of permission requests
   - Test integration with ApprovalService

8. **Session Mode Tests**
   - Test `SetSessionMode` method
   - Test mode state in `NewSessionResponse`
   - Test mode update notifications

9. **Plan and Commands Tests**
   - Test plan notifications
   - Test command execution via prompts
   - Test `available_commands_update` notifications

### Technical Requirements

1. **Test Infrastructure**
   - Use `os/exec` to start `spin acp` subprocess
   - Use `acp.NewClientSideConnection()` with subprocess stdio
   - Handle process lifecycle (start, stop, cleanup)
   - Use test fixtures and temporary directories
   - Mock LLM provider for deterministic tests (or use fast model)

2. **Test Reliability**
   - No flaky tests - all tests must be deterministic
   - Proper timeout handling
   - Graceful cleanup on test failure
   - Isolated test environments (separate temp dirs per test)

3. **Test Coverage**
   - Cover all ACP protocol methods
   - Cover error cases and edge cases
   - Cover notification flows
   - Cover concurrent operations (if applicable)

4. **CI Integration**
   - Tests should run in CI environment
   - Fast execution (use mock LLM or fast model)
   - Proper test isolation
   - Clear test output and error messages

## Design

### Test Structure

```
tests/e2e/acp/
├── acp_e2e_test.go          # Main E2E test suite
├── client_test.go           # Client connection tests
├── session_test.go          # Session management tests
├── prompt_test.go           # Prompt processing tests
├── tool_call_test.go        # Tool call tests
├── cancel_test.go           # Cancellation tests
├── permission_test.go       # Permission request tests
├── mode_test.go             # Session mode tests
├── test_helpers.go          # Test utilities and helpers
└── README.md                # Test documentation
```

### Test Helper Functions

```go
// startACPAgent starts spin acp as subprocess
func startACPAgent(t *testing.T, args ...string) (*exec.Cmd, io.ReadWriteCloser, io.ReadWriteCloser)

// createACPClient creates ACP client connection
func createACPClient(t *testing.T, stdin io.Reader, stdout io.Writer) *acp.ClientSideConnection

// waitForInitialization waits for agent to be ready
func waitForInitialization(t *testing.T, client *acp.ClientSideConnection) error

// cleanupAgent stops and cleans up agent process
func cleanupAgent(t *testing.T, cmd *exec.Cmd)
```

### Test Pattern

```go
func TestACP_Initialize(t *testing.T) {
    // Start agent
    cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b")
    defer cleanupAgent(t, cmd)
    
    // Create client
    client := createACPClient(t, stdin, stdout)
    
    // Test initialization
    resp, err := client.Initialize(ctx, acp.InitializeRequest{
        ProtocolVersion: acp.ProtocolVersionNumber,
        ClientCapabilities: acp.ClientCapabilities{},
    })
    require.NoError(t, err)
    assert.Equal(t, acp.ProtocolVersionNumber, resp.ProtocolVersion)
    assert.True(t, resp.AgentCapabilities.PromptCapabilities.Image)
}
```

## Implementation Plan

1. **Phase 1: Test Infrastructure**
   - Create test directory structure
   - Implement helper functions for starting agent and creating client
   - Implement cleanup and resource management
   - Write basic initialization test

2. **Phase 2: Core Protocol Tests**
   - Implement session management tests
   - Implement prompt processing tests
   - Implement cancellation tests

3. **Phase 3: Advanced Feature Tests**
   - Implement tool call tests
   - Implement permission request tests
   - Implement session mode tests
   - Implement plan and command tests

4. **Phase 4: CI Integration**
   - Ensure tests run in CI
   - Add test documentation
   - Verify test reliability

## Acceptance Criteria

- [ ] E2E test framework created in `tests/e2e/acp/`
- [ ] Tests use `acp.NewClientSideConnection()` for client-side testing
- [ ] Tests start Spin ACP agent as subprocess
- [ ] Initialization flow fully tested
- [ ] Session management fully tested
- [ ] Prompt processing fully tested (including notifications)
- [ ] Tool calls fully tested
- [ ] Cancellation fully tested
- [ ] Permission requests fully tested
- [ ] Session modes fully tested
- [ ] All tests passing reliably (no flaky tests)
- [ ] Tests run in CI environment
- [ ] Test documentation complete

## Testing Strategy

### Test Organization

1. **Basic Protocol Tests** (`acp_e2e_test.go`)
   - Initialize
   - NewSession (basic)
   - Prompt (basic)

2. **Session Tests** (`session_test.go`)
   - NewSession with MCP servers
   - LoadSession
   - Session error handling

3. **Prompt Tests** (`prompt_test.go`)
   - Text content blocks
   - Image/audio content blocks
   - Resource links
   - Notifications
   - Stop reasons

4. **Tool Call Tests** (`tool_call_test.go`)
   - Tool call lifecycle
   - Tool call content
   - Tool call diffs

5. **Cancellation Tests** (`cancel_test.go`)
   - Cancel during prompt
   - Cancel permission request

6. **Permission Tests** (`permission_test.go`)
   - RequestPermission
   - Permission options
   - Approval integration

7. **Mode Tests** (`mode_test.go`)
   - SetSessionMode
   - Mode notifications

### Test Data

- Use minimal test config
- Use fast/small LLM model (qwen3:0.6b or mock)
- Use temporary directories for workspace
- Use mock MCP servers (if needed)

### Test Reliability

- Use deterministic test data
- Proper timeout handling
- Graceful cleanup on failure
- Isolated test environments
- No shared state between tests

## Dependencies

- Feature 8.1 (ACP Connection Handler) - ✅ Completed
- All core ACP features - ✅ Completed
- ACP SDK v0.6.3 - ✅ Available
- Test infrastructure (existing E2E test patterns)

## Risks

1. **Test Flakiness**
   - LLM responses may be non-deterministic
   - Solution: Use mock LLM or very fast/small model
   - Solution: Test for patterns, not exact responses

2. **Process Management**
   - Agent process may not start/stop cleanly
   - Solution: Proper cleanup and timeout handling
   - Solution: Use test helpers for process management

3. **CI Environment**
   - Tests may fail in CI due to environment differences
   - Solution: Use hermetic test fixtures
   - Solution: Proper error messages for debugging

## Open Questions

1. Should we use mock LLM or real LLM for tests?
   - Option A: Mock LLM (fast, deterministic)
   - Option B: Real LLM with fast model (more realistic, slower)
   - Recommendation: Start with real LLM (qwen3:0.6b), add mock option later

2. How to handle MCP servers in tests?
   - Option A: Mock MCP servers
   - Option B: Real MCP servers (if available)
   - Option C: Skip MCP tests if servers not available
   - Recommendation: Option C for now, add mock servers later

3. Should tests be in `tests/e2e/acp/` or `internal/protocol/acp/e2e_test.go`?
   - Recommendation: `tests/e2e/acp/` (follows existing E2E test pattern)

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md) - Feature 10.1
- [ACP SDK Integration](../../docs/packages/acp-sdk-integration.md)
- [ACP Protocol Implementation](../../docs/packages/protocol-acp.md)
- [E2E Test README](../../tests/e2e/README.md)
- [ACP SDK Examples](https://github.com/coder/acp-go-sdk) - `example_client_test.go`

## Notes

- E2E tests should complement unit tests, not replace them
- Focus on protocol compliance and end-to-end flows
- Use existing E2E test patterns from `tests/e2e/`
- Tests should be fast enough to run in CI
- Consider using `-short` flag for quick tests vs full E2E tests

