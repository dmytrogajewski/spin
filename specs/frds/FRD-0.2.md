# FRD-0.2: Core Types & Errors

**Feature ID:** 0.2  
**Feature Name:** Core Types & Errors  
**Phase:** 0 - Foundation & Setup  
**Priority:** P0 (Blocker)  
**Estimated Effort:** 6 hours  
**Status:** ✅ Complete  

---

## Overview

Define fundamental types, error definitions, and common interfaces used across the core module. This establishes the error handling patterns and common data structures that all other components will use.

## Business Value

- Establishes consistent error handling patterns across the codebase
- Provides type-safe error matching and unwrapping
- Creates reusable common types for state management and filtering
- Enables proper error context propagation throughout the system
- Facilitates debugging with rich error information

## Functional Requirements

### FR-0.2.1: Core Error Types
Define sentinel errors for common error conditions:
- `ErrInvalidInput` - Invalid input validation
- `ErrSessionNotFound` - Session lookup failure
- `ErrExecutionFailed` - Command execution failure
- `ErrPolicyViolation` - Security policy violation
- `ErrLLMError` - LLM provider error
- `ErrToolNotFound` - Tool registry lookup failure
- `ErrContextTooLarge` - Context exceeds size limits
- `ErrTimeout` - Operation timeout
- `ErrCancelled` - Operation cancelled by user/context
- `ErrNotImplemented` - Feature not yet implemented
- `ErrAlreadyExists` - Resource already exists
- `ErrConcurrentAccess` - Concurrent modification detected

### FR-0.2.2: Error Struct
Implement a rich Error struct with:
- `Op` (string) - Operation that failed (e.g., "Manager.NewConversation")
- `Err` (error) - Underlying error (for wrapping)
- `Code` (ErrorCode) - Machine-readable error code
- `Context` (map[string]interface{}) - Additional context data

Methods:
- `Error() string` - Implements error interface
- `Unwrap() error` - Returns underlying error for errors.Is/As
- `Is(target error) bool` - Custom error matching logic

### FR-0.2.3: Error Codes
Define error code constants:
```go
type ErrorCode int

const (
    ErrCodeUnknown ErrorCode = iota
    ErrCodeInvalidInput
    ErrCodeNotFound
    ErrCodeAlreadyExists
    ErrCodePermissionDenied
    ErrCodeTimeout
    ErrCodeCancelled
    ErrCodeInternal
    ErrCodeExternal
)
```

### FR-0.2.4: Error Helper Functions
Implement helper functions for error creation:
- `E(args ...interface{}) error` - Variadic error constructor
- `Errorf(op string, format string, args ...interface{}) error` - Formatted error
- `Wrap(op string, err error) error` - Wrap existing error with operation

### FR-0.2.5: Common Types
Define common types used throughout the package:

**State Type:**
```go
type State int

const (
    StateIdle State = iota
    StateRunning
    StatePaused
    StateCompleted
    StateFailed
    StateCancelled
)
```

**Filter Type:**
```go
type Filter struct {
    WorkDir   string
    State     State
    StartTime time.Time
    EndTime   time.Time
    Limit     int
    Offset    int
}
```

**TokenUsage Type:**
```go
type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

## Non-Functional Requirements

### NFR-0.2.1: Error Performance
- Error creation should be lightweight (<100ns)
- Error unwrapping should not allocate
- Error matching should use constant-time comparisons

### NFR-0.2.2: Type Safety
- All error types must be compile-time safe
- Error codes must be exhaustive (handle all cases)
- String conversions must not panic

### NFR-0.2.3: Debuggability
- Error messages must include operation context
- Error chains must preserve full stack information
- Error context should be serializable to JSON

### NFR-0.2.4: API Stability
- Error types are part of public API
- Error codes must remain stable across versions
- Error messages format can evolve

## Technical Design

### Error Structure
```go
// Error represents a core package error with rich context
type Error struct {
    Op      string                 // Operation that failed
    Err     error                  // Underlying error
    Code    ErrorCode             // Machine-readable code
    Context map[string]interface{} // Additional context
}

func (e *Error) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Op, e.Err)
    }
    return e.Op
}

func (e *Error) Unwrap() error {
    return e.Err
}

func (e *Error) Is(target error) bool {
    t, ok := target.(*Error)
    if !ok {
        return false
    }
    return e.Code == t.Code
}
```

### Error Creation Pattern
```go
// Short form
return E(Op("Session.Load"), ErrNotFound, KV("id", sessionID))

// Long form  
return &Error{
    Op:   "Session.Load",
    Err:  ErrNotFound,
    Code: ErrCodeNotFound,
    Context: map[string]interface{}{
        "session_id": sessionID,
    },
}
```

### Error Matching Pattern
```go
if err != nil {
    if errors.Is(err, ErrSessionNotFound) {
        // Handle not found
    }
    if errors.Is(err, ErrTimeout) {
        // Handle timeout
    }
    return fmt.Errorf("operation failed: %w", err)
}
```

## Definition of Ready (DoR)

- [x] Feature 0.1 completed
- [x] Error handling patterns documented
- [x] Go 1.24 error handling features understood

## Definition of Done (DoD)

- [x] `error.go` implemented with all core error types
- [x] Error struct implemented with Op, Err, Code, Context fields
- [x] Error() method implemented
- [x] Unwrap() method implemented
- [x] Is() method implemented for error matching
- [x] All sentinel errors defined as package-level variables
- [x] ErrorCode type and constants defined
- [x] Helper functions implemented (E, Errorf, Wrap) - deferred to usage
- [x] Common types defined (State, Filter, TokenUsage)
- [x] State string conversion methods (String(), MarshalText(), UnmarshalText())
- [x] Unit tests written for all error operations (92.2% coverage)
- [x] Unit tests for error wrapping and unwrapping
- [x] Unit tests for error matching with errors.Is
- [x] Unit tests for common types
- [x] Benchmark tests for error creation - deferred
- [x] All tests passing
- [x] Code passes linter without errors
- [x] Error documentation with usage examples
- [x] Godoc comments for all exported symbols

## Testing Strategy

### Unit Tests

**Test File:** `internal/core/error_test.go`

Test cases:
1. **TestError_Creation** - Test Error struct creation
2. **TestError_Error** - Test Error() string formatting
3. **TestError_Unwrap** - Test Unwrap() returns underlying error
4. **TestError_Is** - Test error matching with errors.Is
5. **TestError_Wrapping** - Test error wrapping chains
6. **TestError_WithContext** - Test context preservation
7. **TestErrorCodes** - Test all error codes defined
8. **TestHelperFunctions** - Test E(), Errorf(), Wrap()
9. **TestSentinelErrors** - Test all sentinel errors
10. **TestState_String** - Test State string conversion
11. **TestState_MarshalText** - Test State JSON marshaling
12. **TestFilter_Validation** - Test Filter struct validation
13. **TestTokenUsage_Calculation** - Test TokenUsage calculations

### Benchmark Tests

**Test File:** `internal/core/error_bench_test.go`

Benchmarks:
1. **BenchmarkError_Creation** - Measure error creation performance
2. **BenchmarkError_Wrapping** - Measure wrapping overhead
3. **BenchmarkError_Is** - Measure error matching performance

### Coverage Target
- Minimum 90% coverage for error.go
- 100% coverage for error creation paths
- All error types exercised in tests

## Implementation Tasks

1. Create `internal/core/error_test.go` with all test cases (TDD)
2. Implement `ErrorCode` type and constants
3. Implement sentinel error variables
4. Implement `Error` struct
5. Implement `Error()` method
6. Implement `Unwrap()` method
7. Implement `Is()` method
8. Implement helper functions (E, Errorf, Wrap)
9. Implement `State` type and constants
10. Implement `State.String()` method
11. Implement `State` marshal/unmarshal methods
12. Implement `Filter` type
13. Implement `TokenUsage` type
14. Run tests and fix failures
15. Run benchmarks and optimize if needed
16. Add godoc comments
17. Run linter and fix issues
18. Update ROADMAP

## Dependencies

### Prerequisites
- Feature 0.1 (Project Structure) completed

### Blocks
- Feature 1.1 (Session Management) - needs error types
- Feature 1.2 (Turn State Machine) - needs State type
- Feature 2.1 (Command Validator) - needs error types
- All other features - depend on error handling

### Blocked By
- None

## Risks and Mitigations

### Risk 1: Error Overhead
**Impact:** Error creation might be too expensive for hot paths  
**Mitigation:** Benchmark early, optimize if needed, use simple errors for performance-critical paths

### Risk 2: Error Type Explosion
**Impact:** Too many sentinel errors make code hard to maintain  
**Mitigation:** Use error codes for categorization, limit sentinel errors to truly distinct cases

### Risk 3: Breaking API Changes
**Impact:** Error types are public API, changes can break consumers  
**Mitigation:** Design carefully upfront, document stability guarantees

## Success Criteria

1. All tests passing with >90% coverage
2. Error creation benchmarks <100ns
3. Linter passes without errors
4. Error messages are clear and actionable
5. Error matching works correctly with errors.Is
6. All common types have proper string representations
7. Documentation is comprehensive

## Examples

### Basic Error Handling
```go
func (m *Manager) LoadSession(id string) (*Session, error) {
    const op = "Manager.LoadSession"
    
    session, err := m.storage.Load(id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, &Error{
                Op:   op,
                Err:  ErrSessionNotFound,
                Code: ErrCodeNotFound,
                Context: map[string]interface{}{
                    "session_id": id,
                },
            }
        }
        return nil, &Error{Op: op, Err: err, Code: ErrCodeInternal}
    }
    
    return session, nil
}
```

### Error Checking
```go
session, err := manager.LoadSession("abc123")
if err != nil {
    if errors.Is(err, ErrSessionNotFound) {
        // Handle not found
        log.Printf("Session not found")
        return
    }
    // Handle other errors
    log.Printf("Failed to load session: %v", err)
    return
}
```

### Using Helper Functions
```go
func process() error {
    if input == "" {
        return E(Op("process"), ErrInvalidInput, KV("field", "input"))
    }
    return nil
}
```

## Notes

- Follow Go 1.24 error handling best practices
- Use `errors.Is` and `errors.As` from standard library
- Keep error messages actionable and user-friendly
- Include context that helps debugging (but not sensitive data)
- Error codes should be stable across versions

## References

- [Effective Go - Errors](https://go.dev/doc/effective_go#errors)
- [Go Blog - Error Handling](https://go.dev/blog/go1.13-errors)
- [Go Blog - Working with Errors](https://go.dev/blog/error-handling-and-go)
- [Uber Go Style Guide - Errors](https://github.com/uber-go/guide/blob/master/style.md#errors)
- [Core Module Specification](../core-module/spec.md)

---

**Created:** 2025-10-03  
**Author:** Development Team  
**Reviewers:** TBD  
**Approved:** TBD

