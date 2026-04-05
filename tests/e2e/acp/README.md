# ACP E2E Tests

End-to-end tests for the Agent Client Protocol (ACP) implementation.

## Overview

These tests verify the complete ACP protocol flow by:
- Starting Spin ACP agent as a subprocess
- Using ACP SDK's `ClientSideConnection` to communicate with the agent
- Testing all protocol methods end-to-end
- Verifying protocol compliance

## Test Structure

```
tests/e2e/acp/
├── test_helpers.go      # Test infrastructure (process management, client creation)
├── initialize_test.go   # Initialization flow tests
├── session_test.go      # Session management tests
├── prompt_test.go       # Prompt processing tests
└── README.md            # This file
```

## Running Tests

### Prerequisites

- Spin binary must be built: `go build -o bin/spin cmd/spin`
- Ollama must be running (for LLM provider)
- Test model available (e.g., `qwen3:0.6b`)

### Run All Tests

```bash
# From project root
go test ./tests/e2e/acp/... -v

# With race detection
go test ./tests/e2e/acp/... -v -race

# Skip slow tests
go test ./tests/e2e/acp/... -v -short
```

### Run Specific Test

```bash
go test ./tests/e2e/acp/... -v -run TestACP_Initialize
```

## Test Infrastructure

### Helper Functions

- `startACPAgent()` - Starts `spin acp` subprocess with stdio pipes
- `createACPClient()` - Creates ACP client-side connection
- `cleanupAgent()` - Gracefully stops agent process
- `createTestWorkspace()` - Creates temporary directory for testing
- `waitForInitialization()` - Waits for agent to be ready

### Test Client

The `testClient` struct implements the `acp.Client` interface:
- Auto-approves permission requests (for testing)
- Handles session update notifications
- Provides stub implementations for unused methods

## Test Coverage

### Initialization Tests
- ✅ Protocol version negotiation
- ✅ Capability advertisement
- ✅ Agent info exchange
- ✅ Client capability storage
- ✅ Timeout handling

### Session Management Tests
- ✅ NewSession with working directory
- ✅ NewSession with MCP servers
- ✅ Session mode state
- ✅ Error handling
- ✅ Concurrent session creation
- ✅ LoadSession (basic)

### Prompt Processing Tests
- ✅ Basic prompt with text blocks
- ✅ Prompt with image/audio blocks
- ✅ Mixed content blocks
- ✅ Invalid session handling

### Tool Call Tests
- ✅ Tool execution and notifications
- ✅ Tool call notification structure
- ✅ Tool call updates

### Cancellation Tests
- ✅ Cancel during prompt execution
- ✅ Cancel with invalid session
- ✅ Cancel when no prompt active

### Permission Request Tests
- ✅ Permission request flow
- ✅ Invalid session handling
- ✅ All permission option kinds

### Session Mode Tests
- ✅ Set session mode
- ✅ All available modes
- ✅ Invalid session/mode handling
- ✅ Mode change notifications

## Future Enhancements

Additional tests that could be added:
- Plan and command tests (integration with planning system)
- More complex tool call scenarios
- Concurrent prompt execution
- Session persistence tests

## Notes

- Tests use real LLM (Ollama with qwen3:0.6b) for realistic testing
- Tests create isolated workspaces per test
- Process cleanup is handled automatically
- Tests skip in short mode (`-short` flag)

## Troubleshooting

**Agent process doesn't start:**
- Check that `bin/spin` exists and is executable
- Verify Ollama is running
- Check model is available: `ollama list`

**Tests timeout:**
- Increase `testTimeout` in `test_helpers.go`
- Check agent logs in stderr
- Verify LLM provider is responding

**Connection errors:**
- Ensure stdio pipes are properly connected
- Check for process startup delays
- Verify ACP SDK version compatibility

## See Also

- [ACP Roadmap](../../../specs/acp/ROADMAP.md)
- [ACP Protocol Implementation](../../../docs/packages/protocol-acp.md)
- [ACP SDK Integration](../../../docs/packages/acp-sdk-integration.md)

