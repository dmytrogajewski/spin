# BUG-20251014030000: Tool Response Messages Missing from Conversation History

**Status:** ✅ Fixed
**Severity:** P0 - Critical
**Reported:** 2025-10-14 03:00:00
**Fixed:** 2025-10-14 03:50:00

## Summary

The agent executes tool calls successfully, but the tool call messages (assistant with tool_calls) and tool result messages (role="tool") are never added to the conversation history. This breaks multi-turn conversations as the LLM cannot see what tools were called or their results on subsequent turns.

## Impact

- **Agent cannot see tool responses:** On the next turn, the LLM has no context about previous tool calls
- **Broken conversation flow:** Agent repeats tool calls or provides incorrect responses
- **User sees errors:** UI shows "Failed to write file" even though write succeeded
- **Data loss:** Tool interaction history is completely lost

## Root Cause

In `internal/core/conversation.go` lines 245-254, `RunTurn` only adds:
1. User message (`history.AddUserMessage`)
2. Final assistant text response (`history.AddAssistantMessage`)

It does NOT add:
1. Assistant messages with tool calls (role="assistant" with `tool_calls` field)
2. Tool result messages (role="tool" with `tool_call_id` field)

Meanwhile, in `internal/core/agent.go` lines 509-594, the agent:
- Adds assistant messages with tool calls to LOCAL `messages` slice (line 519)
- Adds tool result messages to LOCAL `messages` slice (lines 554-560, 581-586)
- Uses these in the agent loop to provide context to LLM

But these messages are never persisted to `c.history`.

## Reproduction

1. Start a conversation
2. Ask agent to use a tool (e.g., "create a file")
3. Tool executes successfully
4. Ask a follow-up question referencing the tool result
5. Agent has no knowledge of previous tool call

Example user report:
```
│   WRITE   tetris_game.txt
 ⤷ Failed to write file.
  [[0 for _ in range(10)] for _ in range(10)], [1, 1, 1, 1]...
  ● Failed
 ⤷ Failed to write file.
```

File was actually written successfully, but agent never received confirmation.

## Expected Behavior

After a turn completes with tool calls:
1. User message should be added to history
2. For each tool interaction:
   - Assistant message with `tool_calls` should be added
   - Tool result message (role="tool") should be added
3. Final assistant message (if any text response) should be added

Example proper flow:
```json
[
  {"role": "user", "content": "create a file"},
  {"role": "assistant", "content": "", "tool_calls": [{"id": "1", "function": {"name": "write_file", ...}}]},
  {"role": "tool", "tool_call_id": "1", "content": "File written successfully"},
  {"role": "assistant", "content": "I've created the file for you."}
]
```

## Proposed Solution

### Option 1: Return Full Message History from Agent

Modify `AgentResponse` to include all messages generated during the turn:

```go
type AgentResponse struct {
    Content      string
    Messages     []Message  // NEW: Full message history from turn
    ToolCalls    []*ToolCall
    ToolResults  []*ToolResult
    TurnsUsed    int
    TokensUsed   int
    FinishReason string
    Error        error
}
```

Then in `conversation.go`, add all messages to history:

```go
respMu.Lock()
if resp != nil {
    // Add user message
    _ = c.history.AddUserMessage(req.Input)

    // Add all turn messages (assistant + tool + assistant final)
    for _, msg := range resp.Messages {
        if msg.Role != RoleUser { // Skip user message (already added)
            _ = c.history.AddMessage(msg)
        }
    }
}
respMu.Unlock()
```

### Option 2: Track Messages Separately

Have agent track messages generated and return them separately from content.

**Recommendation:** Option 1 is cleaner and more explicit.

## Files Affected

- `internal/core/agent.go` - Agent.Execute() must populate Messages field
- `internal/core/conversation.go` - RunTurn() must add all messages to history
- `internal/core/history.go` - May need helper for bulk message addition

## Testing Requirements

1. **Unit test:** Verify tool messages are added to history
2. **Integration test:** Multi-turn conversation with tools verifies context preservation
3. **E2E test:** Full conversation with multiple tool calls confirms LLM receives context

## Implementation

### Changes Made

1. **Modified `AgentResponse` struct** (`internal/core/agent.go:165`)
   - Added `Messages []Message` field to capture all messages generated during turn execution
   - This includes assistant messages with tool_calls, tool result messages, and final assistant response

2. **Updated `Agent.Execute()` method** (`internal/core/agent.go:429`)
   - Track starting history length before turn execution
   - Capture all new messages (after history) into `resp.Messages` before returning
   - Extract final assistant message addition into `addFinalMessage()` helper
   - Extract tool processing logic into `processToolCalls()` helper (reduces cyclomatic complexity)

3. **Updated `Conversation.RunTurn()` method** (`internal/core/conversation.go:245`)
   - Add user message to history (existing)
   - **NEW:** Iterate through `resp.Messages` and add all non-user messages to history
   - This ensures tool calls, tool results, and final assistant messages are all persisted

4. **Added comprehensive tests** (`internal/core/conversation_toolhistory_test.go`)
   - `TestConversation_ToolMessagesInHistory`: Verifies single tool call flow
   - `TestConversation_MultipleToolCalls`: Verifies multiple tool calls in one turn
   - Both tests verify that tool result messages appear in subsequent LLM requests

### Code Quality

- ✅ All existing tests pass
- ✅ New tests added and passing
- ✅ Linter clean (gofmt, gocyclo ≤15)
- ✅ No race conditions (tested with `-race`)
- ✅ Cyclomatic complexity reduced through refactoring

## Acceptance Criteria

- [x] Tool call messages (assistant with tool_calls) are added to history
- [x] Tool result messages (role="tool") are added to history
- [x] Multi-turn conversations maintain full tool context
- [x] All tests pass
- [x] No regression in existing functionality
- [x] Linter clean
- [x] Cyclomatic complexity ≤15

## Related

- User report: "agent do not get responses from tools"
- Observable symptom: File writes succeed but show as "Failed"
- Root cause: Messages only used in agent loop, never persisted to conversation history

## Testing

```bash
# Run specific tests
go test -v -race ./internal/core -run "TestConversation_ToolMessagesInHistory|TestConversation_MultipleToolCalls"

# Run all core tests
go test -race ./internal/core/...

# Run linter
make lint
```

---

**Fixed:** 2025-10-14 03:50:00
**Commit:** (to be added)
