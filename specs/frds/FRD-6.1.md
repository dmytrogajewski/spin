# FRD-6.1: Agent Orchestration

**Feature ID:** 6.1  
**Feature Name:** Agent Orchestration  
**Phase:** 6 - Agent Core  
**Priority:** P0 (Blocker)  
**Estimated Effort:** 20 hours  
**Actual Effort:** ~6 hours  
**Status:** ✅ Complete  

---

## Overview

Implement the core agent decision loop that orchestrates LLM interactions, tool execution, and conversation flow. The Agent is the central coordinator that processes user requests through multiple turns of LLM calls and tool executions until the task is complete or limits are reached.

## Problem Statement

The Spin agent needs a robust orchestration layer that:
- Manages the interaction loop between the LLM and tool execution
- Handles streaming responses from the LLM
- Coordinates tool call processing and approval workflows
- Enforces safety policies and execution limits
- Provides context-aware prompt construction
- Handles errors gracefully and provides meaningful feedback

Without proper orchestration, the agent cannot effectively coordinate between LLM reasoning and tool execution, limiting its ability to accomplish complex multi-step tasks.

## Goals

1. **Core Agent Loop:** Implement the main agent execution loop that coordinates LLM and tools
2. **Tool Integration:** Process and execute tool calls from LLM responses
3. **Approval System:** Implement command approval logic for dangerous operations
4. **Context Management:** Build prompts with appropriate context and history
5. **Streaming Support:** Handle streaming LLM responses for real-time feedback
6. **Safety Enforcement:** Integrate with validator and executor for safe operations
7. **Turn Management:** Implement multi-turn conversations with proper limits

## Non-Goals

- Full LLM provider implementation (Phase 8.1)
- Complete tool registry implementation (Phase 8.2)
- Security sandbox implementation (Phase 8.3)
- MCP integration (Phase 8.4)
- Advanced caching or optimization (Phase 8.6)

## Definition of Ready (DoR)

- [x] Feature 2.1 (Command Validator) completed
- [x] Feature 2.2 (Command Executor) completed
- [x] Feature 3.1 (Environment Context) completed
- [x] Feature 4.1 (Event Infrastructure) completed
- [x] Mock LLM provider interface available
- [x] Mock tools registry interface available
- [x] Agent loop algorithm defined (see Technical Design)
- [x] Error handling patterns established (Feature 0.2)
- [x] Event system available (Feature 4.1)

## Definition of Done (DoD)

### Code Implementation
- [ ] `agent.go` implemented with Agent struct
- [ ] `NewAgent()` constructor with dependency injection
- [ ] `Execute()` method with main agent loop
- [ ] `ProcessToolCall()` for tool invocations
- [ ] `ShouldApprove()` for approval decision logic
- [ ] `buildPrompt()` for context construction
- [ ] LLM streaming integration
- [ ] Tool call accumulation and execution
- [ ] Multi-turn loop with max turns limit
- [ ] Timeout enforcement via context
- [ ] Finish reason detection
- [ ] Proper error handling and propagation

### Testing
- [ ] Unit tests for Agent struct (>85% coverage)
- [ ] Agent loop flow tests
- [ ] Tool call processing tests
- [ ] Approval logic tests
- [ ] Context building tests
- [ ] Error handling tests
- [ ] Timeout tests
- [ ] Max turns limit tests
- [ ] Integration tests with mock LLM
- [ ] Integration tests with mock tools
- [ ] Concurrent execution tests (race detector clean)

### Quality
- [ ] All tests passing
- [ ] Race detector clean (`go test -race`)
- [ ] Linter passing (`make lint`)
- [ ] Code analyzed with uast/herr (complexity ≤15)
- [ ] Godoc comments for all exported symbols
- [ ] Error handling follows project patterns

### Documentation
- [ ] Godoc comments complete
- [ ] Agent loop flow documented
- [ ] Tool call processing documented
- [ ] Approval system documented
- [ ] Example usage provided
- [ ] FRD updated with actual results

## Technical Design

### Agent Structure

```go
// Agent implements the core agent logic and decision-making loop.
type Agent struct {
    llm       LLMProvider        // LLM provider interface (mock for now)
    tools     ToolsRegistry      // Tools registry interface (mock for now)
    executor  *Executor          // Command executor
    validator *Validator         // Command validator
    context   *Context           // Environment context
    emitter   *EventEmitter      // Event emission
    config    *AgentConfig       // Agent configuration
}

// AgentConfig contains agent configuration
type AgentConfig struct {
    MaxTurns        int           // Maximum agent turns (default: 50)
    Timeout         time.Duration // Execution timeout (default: 5min)
    Temperature     float64       // LLM temperature (default: 0.7)
    MaxTokens       int           // Max tokens per call (default: 4096)
    RequireApproval bool          // Require approval for dangerous commands
}
```

### Key Methods

```go
// NewAgent creates a new agent with dependencies
func NewAgent(llm LLMProvider, tools ToolsRegistry, executor *Executor, 
              validator *Validator, opts ...AgentOption) *Agent

// Execute runs the agent loop for a request
func (a *Agent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error)

// ProcessToolCall processes a single tool call from LLM
func (a *Agent) ProcessToolCall(ctx context.Context, call *ToolCall) (*ToolResult, error)

// ShouldApprove determines if a command needs user approval
func (a *Agent) ShouldApprove(cmd *Command) (bool, string)

// buildPrompt constructs the LLM prompt with context
func (a *Agent) buildPrompt(req *AgentRequest, history []Message) []Message
```

### Data Structures

```go
// AgentRequest represents a request to the agent
type AgentRequest struct {
    Input       string            // User input
    History     []Message         // Conversation history
    Context     *Context          // Environment context
    Task        Task              // Task mode (regular, review, compact)
    WorkDir     string            // Working directory
}

// AgentResponse represents the agent's response
type AgentResponse struct {
    Content     string            // Response content
    ToolCalls   []*ToolCall       // Tools called
    ToolResults []*ToolResult     // Tool execution results
    TurnsUsed   int               // Number of turns used
    TokensUsed  int               // Tokens consumed
    FinishReason string           // Reason for completion
    Error       error             // Error if failed
}

// ToolCall represents an LLM tool invocation
type ToolCall struct {
    ID          string            // Tool call ID
    Name        string            // Tool name
    Arguments   map[string]interface{} // Tool arguments
}

// ToolResult represents tool execution result
type ToolResult struct {
    ID          string            // Matches ToolCall.ID
    Success     bool              // Execution success
    Output      string            // Tool output
    Error       error             // Error if failed
    ExitCode    int               // Exit code (for commands)
}

// Command represents a command to execute
type Command struct {
    Cmd     string   // Command to execute
    Args    []string // Command arguments
    WorkDir string   // Working directory
    Env     []string // Environment variables
}
```

### Agent Loop Algorithm

```
FUNCTION Execute(ctx, req):
    1. Initialize response and turn counter
    2. Build initial prompt with context
    3. WHILE turn < MaxTurns AND !done:
        a. Check context timeout/cancellation
        b. Call LLM with current prompt + history
        c. FOR EACH chunk in LLM stream:
            i.   IF ContentDelta: emit event, accumulate content
            ii.  IF ToolCall: accumulate tool call
            iii. IF ToolCallComplete:
                - Process tool call
                - Check if needs approval
                - IF approved: execute tool
                - Add result to context
            iv.  IF Done: check finish reason
        d. IF finish_reason == "stop": done = true
        e. IF finish_reason == "tool_calls": continue loop
        f. Increment turn counter
    4. Return response with all results
```

### Tool Call Processing Flow

```
FUNCTION ProcessToolCall(ctx, call):
    1. Look up tool in registry
    2. Extract and validate arguments
    3. IF tool is command execution:
        a. Parse command from arguments
        b. Classify command safety (validator)
        c. Check if needs approval
        d. IF needs approval:
            - Emit approval request event
            - Wait for user decision
            - IF denied: return error result
        e. Execute command via executor
        f. Return execution result
    4. ELSE (other tool types):
        a. Execute tool via registry
        b. Return tool result
    5. Handle errors gracefully
```

### Approval Decision Logic

```
FUNCTION ShouldApprove(cmd):
    1. Classify command via validator
    2. IF CommandSafe: return false, ""
    3. IF CommandInteractive: 
        return true, "This command may modify files"
    4. IF CommandDangerous: 
        return true, "WARNING: Dangerous operation"
    5. IF CommandForbidden: 
        return false, "BLOCKED: Forbidden command"
    6. DEFAULT: return true, "Unknown command, approval required"
```

### Context Building

```
FUNCTION buildPrompt(req, history):
    1. Create system message with:
        - Task mode system prompt
        - Environment context summary
        - Available tools list
        - Safety guidelines
    2. Add conversation history
    3. Add user input
    4. Truncate if exceeds token budget
    5. Return message list
```

## Implementation Plan

### Step 1: Core Agent Structure (2 hours)
1. Define `Agent` struct with all fields
2. Define `AgentConfig` with defaults
3. Implement `NewAgent()` constructor
4. Add functional options pattern
5. Write constructor tests

### Step 2: Request/Response Types (1 hour)
1. Define `AgentRequest` struct
2. Define `AgentResponse` struct
3. Define `ToolCall` and `ToolResult` structs
4. Define `Command` struct
5. Add validation methods

### Step 3: Basic Execute Loop (4 hours)
1. Implement `Execute()` skeleton
2. Add turn counter and timeout enforcement
3. Add LLM call integration
4. Implement basic loop structure
5. Add finish reason detection
6. Write loop flow tests

### Step 4: Tool Call Processing (4 hours)
1. Implement `ProcessToolCall()` method
2. Add tool registry lookup
3. Add argument parsing
4. Integrate with executor
5. Handle errors gracefully
6. Write tool call tests

### Step 5: Approval System (3 hours)
1. Implement `ShouldApprove()` method
2. Integrate with validator
3. Add approval event emission
4. Handle approval responses
5. Write approval tests

### Step 6: Context Building (2 hours)
1. Implement `buildPrompt()` method
2. Add system message construction
3. Add context summarization
4. Add history integration
5. Add token budget awareness
6. Write prompt building tests

### Step 7: Event Integration (2 hours)
1. Add event emission throughout loop
2. Emit ContentDelta events
3. Emit ToolCallStart/Progress/Complete events
4. Emit approval request events
5. Emit error events
6. Write event tests

### Step 8: Error Handling (2 hours)
1. Add error wrapping with context
2. Handle LLM errors
3. Handle tool execution errors
4. Handle timeout errors
5. Write error handling tests

## Test Plan

### Unit Tests

```go
// TestNewAgent tests agent creation
func TestNewAgent(t *testing.T)

// TestAgent_Execute_SingleTurn tests single turn execution
func TestAgent_Execute_SingleTurn(t *testing.T)

// TestAgent_Execute_MultiTurn tests multi-turn execution
func TestAgent_Execute_MultiTurn(t *testing.T)

// TestAgent_Execute_MaxTurns tests turn limit enforcement
func TestAgent_Execute_MaxTurns(t *testing.T)

// TestAgent_Execute_Timeout tests timeout enforcement
func TestAgent_Execute_Timeout(t *testing.T)

// TestAgent_ProcessToolCall tests tool call processing
func TestAgent_ProcessToolCall(t *testing.T)

// TestAgent_ProcessToolCall_Approval tests approval flow
func TestAgent_ProcessToolCall_Approval(t *testing.T)

// TestAgent_ShouldApprove tests approval decision logic
func TestAgent_ShouldApprove(t *testing.T)

// TestAgent_buildPrompt tests prompt construction
func TestAgent_buildPrompt(t *testing.T)
```

### Integration Tests

```go
// TestAgent_Integration_Complete tests complete agent flow
func TestAgent_Integration_Complete(t *testing.T)

// TestAgent_Integration_WithMockLLM tests with mock LLM
func TestAgent_Integration_WithMockLLM(t *testing.T)

// TestAgent_Integration_WithExecutor tests tool execution
func TestAgent_Integration_WithExecutor(t *testing.T)
```

### Test Coverage Targets
- **Agent core logic:** >90% coverage
- **Error paths:** 100% coverage
- **Overall package:** >85% coverage

## Dependencies

### Internal Dependencies
- `internal/core/validator.go` (Feature 2.1) ✅
- `internal/core/executor.go` (Feature 2.2) ✅
- `internal/core/context.go` (Feature 3.1) ✅
- `internal/core/event.go` (Feature 4.1) ✅
- `internal/core/error.go` (Feature 0.2) ✅
- `internal/core/task/task.go` (Feature 5.1) ✅
- `internal/core/testing/mock_llm.go` ✅

### External Dependencies
- `context` (stdlib)
- `time` (stdlib)
- `sync` (stdlib)

### Future Integration Points
- `internal/llm` package (Phase 8.1) - will replace mock
- `internal/tools` package (Phase 8.2) - will replace mock
- `internal/security` package (Phase 8.3) - will enhance executor

## Risk Assessment

### High Risk
- **Agent loop complexity:** The agent loop has many edge cases and state transitions
  - *Mitigation:* Incremental implementation, extensive testing, state diagram documentation
  
- **Tool call coordination:** Coordinating LLM streaming and tool execution is complex
  - *Mitigation:* Clear separation of concerns, well-defined interfaces, integration tests

### Medium Risk
- **Timeout handling:** Context cancellation must propagate correctly
  - *Mitigation:* Consistent context.Context usage, timeout tests

- **Error recovery:** Errors must be handled gracefully without leaving partial state
  - *Mitigation:* Transaction-like patterns, error wrapping, cleanup handlers

### Low Risk
- **Mock integration:** Working with mocks before real implementations
  - *Mitigation:* Follow interface contracts exactly, document differences from real implementation

## Success Criteria

1. ✅ Agent can execute single-turn requests
2. ✅ Agent can execute multi-turn requests
3. ✅ Tool calls are processed correctly
4. ✅ Approval system works for dangerous commands
5. ✅ Timeout and max turns limits enforced
6. ✅ Events emitted for all major operations
7. ✅ >85% test coverage achieved
8. ✅ All tests passing with race detector clean
9. ✅ Linter passing with no warnings
10. ✅ Code complexity ≤15 for all functions

## Example Usage

```go
// Create agent with dependencies
llm := testing.NewMockProvider("Hello! I'll help you with that.")
tools := testing.NewMockToolsRegistry()
executor := NewExecutor(validator, nil, workDir, 30*time.Second)
validator := NewValidator()

agent := NewAgent(llm, tools, executor, validator,
    WithMaxTurns(50),
    WithTimeout(5*time.Minute),
    WithRequireApproval(true),
)

// Create request
req := &AgentRequest{
    Input:   "List files in current directory",
    Context: ctx,
    WorkDir: "/home/user/project",
}

// Execute agent
resp, err := agent.Execute(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Response: %s\n", resp.Content)
fmt.Printf("Turns used: %d\n", resp.TurnsUsed)
fmt.Printf("Tokens used: %d\n", resp.TokensUsed)
```

## Open Questions

1. **Q: How should we handle partial tool execution in multi-tool scenarios?**
   - A: Execute tools sequentially, propagate errors immediately, provide partial results in error case

2. **Q: Should we implement tool call parallelization?**
   - A: Not in this phase. Keep sequential execution for simplicity and safety.

3. **Q: How do we handle approval timeouts?**
   - A: Use context with timeout. If approval not received within timeout, treat as denial.

4. **Q: Should we cache tool results?**
   - A: Not in this phase. Defer to Phase 8.6 (Performance Optimization).

## Related Documents

- [Core Module Spec](../core-module/spec.md)
- [Architecture Overview](../architecture-overview.md)
- [ROADMAP](../core-module/ROADMAP.md)
- [FRD-2.1: Command Validator](./FRD-2.1.md)
- [FRD-2.2: Command Executor](./FRD-2.2.md)
- [FRD-3.1: Environment Context](./FRD-3.1.md)
- [FRD-4.1: Event Infrastructure](./FRD-4.1.md)

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [SOLID Principles](https://en.wikipedia.org/wiki/SOLID)

---

**Created:** October 3, 2025  
**Author:** AI Development Agent  
**Reviewers:** Development Team  
**Last Updated:** October 3, 2025

