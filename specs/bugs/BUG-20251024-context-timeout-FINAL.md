# Context Timeout Fix - Final Summary

**Date:** 2025-10-24  
**Status:** Fixed

## Summary

Removed all hardcoded defaults in favor of config-only approach. The system now uses config defaults exclusively:
- LLM timeout uses `a.config.Timeout` (60 minutes from DefaultConfig)
- Agent timeout uses `a.config.Timeout` directly (no fallback to constants)
- Respects existing context deadlines when present

## Issue

**Original Code (INCORRECT):**
```go
func (a *Agent) callLLMWithTimeout(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error) {
    timeout := 30 * time.Second  // Hard-coded!
    // ...
}

func (a *Agent) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
    timeout := a.config.Timeout
    if timeout == 0 {
        timeout = DefaultAgentTimeout  // Hard-coded fallback!
    }
    return context.WithTimeout(ctx, timeout)
}
```

**Problems:**
1. Hard-coded 30-second timeout in callLLMWithTimeout
2. Hard-coded fallback to DefaultAgentTimeout constant
3. No respect for existing context deadlines
4. Mixed approach: config + hardcoded fallbacks

## Fix

**Updated Code (CORRECT):**
```go
func (a *Agent) callLLMWithTimeout(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error) {
    // Check if incoming context already has a deadline
    var effectiveTimeout time.Duration
    if deadline, ok := ctx.Deadline(); ok {
        effectiveTimeout = time.Until(deadline)
        slog.Debug("using existing context deadline", "timeout", effectiveTimeout)
    } else {
        // No existing deadline, use config timeout (initialized from DefaultConfig: 60 minutes)
        effectiveTimeout = a.config.Timeout
        slog.Debug("using config timeout", "timeout", effectiveTimeout)
    }
    // ...
}

func (a *Agent) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
    // Use config timeout (initialized from DefaultConfig: 60 minutes)
    return context.WithTimeout(ctx, a.config.Timeout)
}
```

## Behavior

### Case 1: Fresh Context (Production Use)
- No existing deadline
- Uses `a.config.Timeout` (60 minutes by default)
- Supports long-running tasks

### Case 2: Context with Deadline (Tests)
- Existing deadline present (e.g., 1 second test timeout)
- Uses the existing deadline
- Test timeout enforced immediately

### Case 3: Config Override
- If config sets shorter timeout, uses that
- If config sets longer timeout (up to 60 minutes), uses that
- Config always wins when no existing deadline

## Timeout Flow

```
User Context
  ↓
applyTimeout() → wraps with 60m timeout
  ↓
callLLMWithTimeout() → checks for existing deadline
  ↓
[Existing 60m deadline] → Use it
  ↓
LLM Call → Can run up to 60 minutes
```

## Files Modified

- `internal/agent/agent.go:995-1017` - Fixed timeout logic to use config only
- `internal/agent/agent.go:308-312` - Removed hardcoded fallback to DefaultAgentTimeout
- `internal/conversation/conversation.go:52-97` - Preserve history on error

## Verification

Production: LLM calls can run up to 60 minutes  
Tests: Test timeouts still work correctly  
Config: User-configured timeouts respected

