# Context Timeout and History Issues - Fix Summary

**Date:** 2025-10-24  
**Severity:** High  
**Status:** Fixed

## Summary

Fixed two critical issues discovered during agent execution testing:
1. LLM calls timing out prematurely (30 seconds) despite 60-minute agent timeout
2. Conversation history not preserved on error, causing loss of context

## Issues Found

### 1. Premature LLM Timeout (CRITICAL)

**Location:** `internal/agent/agent.go:997`

**Problem:**
Hard-coded 30-second timeout for LLM calls caused premature cancellation even though agent timeout is 60 minutes:

```go
timeout := 30 * time.Second // Default timeout for LLM calls
```

**Impact:**
- Complex code generation tasks fail after 30 seconds
- Agent timeout setting (60 minutes) was ignored
- User-facing error: "context cancelled: context deadline exceeded"

**Fix:** ✅
Changed default LLM timeout to 2 minutes:

```go
// Use 2 minutes default for LLM calls (sufficient for most complex tasks)
// Allow agent config to override with shorter timeout if needed
timeout := 2 * time.Minute
```

**File:** `internal/agent/agent.go:995-1014`

### 2. History Not Preserved on Error (CRITICAL)

**Location:** `internal/conversation/conversation.go:52-68`

**Problem:**
Conversation history was only added AFTER successful execution, so errors caused complete context loss:

```go
// Execute agent
resp, err := c.agent.Execute(ctx, req)
if err != nil {
    return fmt.Errorf("agent execution failed: %w", err)  // No history saved!
}
// History added here - too late!
```

**Impact:**
- User messages lost on error
- Agent "forgets" previous conversation
- No context available for retry attempts

**Fix:** ✅
Add user message to history BEFORE execution and capture errors:

```go
// Add user message to history BEFORE execution so it's preserved even on error
err := c.history.AddUserMessage(input)
if err != nil {
    return fmt.Errorf("failed to add user message: %w", err)
}

// Execute agent
resp, err := c.agent.Execute(ctx, req)
if err != nil {
    // Add error message to history so it's preserved
    errorMsg := message.Message{
        Role:    message.RoleAssistant,
        Content: fmt.Sprintf("Error: %v", err),
    }
    _ = c.history.AddMessage(errorMsg)
    return fmt.Errorf("agent execution failed: %w", err)
}
```

**File:** `internal/conversation/conversation.go:52-97`

## User Experience Improvements

### Before
```
> Create simple tetris game in python
[30 seconds later]
✗ Error: agent execution failed: context cancelled: context deadline exceeded

> try again
I can't access your chat history...
```

### After
```
> Create simple tetris game in python
[2 minutes allowed for LLM call]
✓ Game created!

[On error:]
> Create simple tetris game in python
✗ Error: agent execution failed: ...
> try again
I'll continue working on the tetris game...
```

## Testing

✅ All conversation tests pass
✅ Agent timeout handling verified
✅ History preservation tested

## Root Causes

1. **LLM Timeout Too Short**: 30 seconds insufficient for complex code generation
2. **History Added After Success**: Lost context on any error
3. **No Error Capture**: Errors not recorded in history

## Files Modified

- `internal/agent/agent.go` - Increased LLM timeout to 2 minutes
- `internal/conversation/conversation.go` - Preserve history on error

## Verification

Run agent execution with complex tasks - no premature timeouts observed.
History preserved across failed turns - context maintained.

