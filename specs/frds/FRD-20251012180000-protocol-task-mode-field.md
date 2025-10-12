# FRD: Protocol Task Mode Field Support

**ID**: FRD-20251012180000
**Status**: Implementation
**Created**: 2025-10-12
**Roadmap**: [P4.1] Update Protocol with Task Mode Field
**Phase**: 4 - AppServer Integration
**Priority**: HIGH
**Estimated Effort**: 1 hour

## Overview

Add `task_mode` field to JSON-RPC protocol messages to support task mode selection and reporting in the WebSocket/HTTP API. This enables clients (IDE extensions, web UIs) to specify and query task modes.

## Problem Statement

The task mode system is now integrated into core agent and conversation layers (Phases 1-3 complete), but the WebSocket/JSON-RPC protocol has no way to:

1. **Send** task mode preference from client to server
2. **Report** current task mode from server to client

Without protocol support, external clients (IDE extensions, web UI) cannot use task modes.

## Goals

1. Add optional `task_mode` field to `SendMessageParams` (client → server)
2. Add `task_mode` field to `SendMessageResult` (server → client)
3. Maintain **backward compatibility** (field is optional)
4. Support all 4 modes: `regular`, `review`, `compact`, `planning`
5. Provide clear error messages for invalid modes

## Non-Goals

- Protocol version bump (changes are additive)
- Client-side implementation (out of scope)
- Auto mode selection (future enhancement)

## Design

### 1. Protocol Changes

#### 1.1 SendMessageParams (Request)

**Current:**
```go
// SendMessageParams contains message sending parameters
type SendMessageParams struct {
	ConversationID *string `json:"conversation_id,omitempty"` // nil = new conversation
	Message        string  `json:"message"`
}
```

**New:**
```go
// SendMessageParams contains message sending parameters
type SendMessageParams struct {
	ConversationID *string `json:"conversation_id,omitempty"` // nil = new conversation
	Message        string  `json:"message"`
	TaskMode       *string `json:"task_mode,omitempty"`        // NEW: optional task mode
}
```

**JSON Example:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "send_message",
  "params": {
    "conversation_id": "conv-123",
    "message": "Review this code",
    "task_mode": "review"
  }
}
```

#### 1.2 SendMessageResult (Response)

**Current:**
```go
// SendMessageResult is the response to send_message
type SendMessageResult struct {
	ConversationID string `json:"conversation_id"`
	TurnID         string `json:"turn_id"`
}
```

**New:**
```go
// SendMessageResult is the response to send_message
type SendMessageResult struct {
	ConversationID string `json:"conversation_id"`
	TurnID         string `json:"turn_id"`
	TaskMode       string `json:"task_mode"` // NEW: current task mode
}
```

**JSON Example:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "conversation_id": "conv-123",
    "turn_id": "turn-456",
    "task_mode": "review"
  }
}
```

### 2. Validation Rules

**Valid Task Modes:**
- `regular` (default)
- `review`
- `compact`
- `planning`

**Validation:**
```go
// ValidTaskModes are the allowed task mode names
var ValidTaskModes = map[string]bool{
	"regular":  true,
	"review":   true,
	"compact":  true,
	"planning": true,
}

// ValidateTaskMode checks if a task mode name is valid
func ValidateTaskMode(mode string) error {
	if mode == "" {
		return nil // empty is ok (use default)
	}
	if !ValidTaskModes[mode] {
		return fmt.Errorf("invalid task mode: %s (valid: regular, review, compact, planning)", mode)
	}
	return nil
}
```

### 3. Backward Compatibility

**Missing Field Behavior:**

Request without `task_mode`:
```json
{
  "method": "send_message",
  "params": {
    "message": "Hello"
  }
}
```
→ Uses conversation's current mode (default: `regular`)

Response always includes `task_mode`:
```json
{
  "result": {
    "conversation_id": "conv-123",
    "turn_id": "turn-456",
    "task_mode": "regular"
  }
}
```

**Old Clients:**
- Can ignore `task_mode` in responses
- Don't need to send `task_mode` in requests
- Default behavior unchanged

**New Clients:**
- Can specify `task_mode` to switch modes
- Can read `task_mode` to display current mode
- Can validate mode before sending

## Implementation

### File: `internal/protocol/jsonrpc/jsonrpc.go`

**Changes Required:**

1. Add `TaskMode *string` field to `SendMessageParams`
2. Add `TaskMode string` field to `SendMessageResult`
3. Add validation helper `ValidateTaskMode()`
4. Update godoc comments

**Code:**

```go
// SendMessageParams contains message sending parameters
type SendMessageParams struct {
	// ConversationID identifies the conversation (nil = new conversation)
	ConversationID *string `json:"conversation_id,omitempty"`

	// Message is the user's input
	Message string `json:"message"`

	// TaskMode optionally specifies the task mode to use for this turn.
	// Valid values: "regular", "review", "compact", "planning"
	// If nil, uses the conversation's current mode (default: "regular")
	TaskMode *string `json:"task_mode,omitempty"`
}

// SendMessageResult is the response to send_message
type SendMessageResult struct {
	// ConversationID uniquely identifies the conversation
	ConversationID string `json:"conversation_id"`

	// TurnID uniquely identifies this turn
	TurnID string `json:"turn_id"`

	// TaskMode is the current task mode for the conversation
	TaskMode string `json:"task_mode"`
}

// ValidTaskModes are the allowed task mode names
var ValidTaskModes = map[string]bool{
	"regular":  true,
	"review":   true,
	"compact":  true,
	"planning": true,
}

// ValidateTaskMode checks if a task mode name is valid.
// Empty string is valid (means use default).
func ValidateTaskMode(mode string) error {
	if mode == "" {
		return nil
	}
	if !ValidTaskModes[mode] {
		return fmt.Errorf("invalid task mode: %s (valid: regular, review, compact, planning)", mode)
	}
	return nil
}
```

## Testing Strategy

### Unit Tests

#### Test 1: SendMessageParams with TaskMode Serialization
```go
func TestSendMessageParams_WithTaskMode(t *testing.T) {
	mode := "review"
	params := SendMessageParams{
		Message:  "Hello",
		TaskMode: &mode,
	}

	data, err := json.Marshal(params)
	require.NoError(t, err)

	var decoded SendMessageParams
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.NotNil(t, decoded.TaskMode)
	assert.Equal(t, "review", *decoded.TaskMode)
}
```

#### Test 2: SendMessageParams without TaskMode (Backward Compat)
```go
func TestSendMessageParams_WithoutTaskMode(t *testing.T) {
	params := SendMessageParams{
		Message: "Hello",
	}

	data, err := json.Marshal(params)
	require.NoError(t, err)

	var decoded SendMessageParams
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.TaskMode)
}
```

#### Test 3: SendMessageResult Serialization
```go
func TestSendMessageResult_WithTaskMode(t *testing.T) {
	result := SendMessageResult{
		ConversationID: "conv-123",
		TurnID:         "turn-456",
		TaskMode:       "compact",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded SendMessageResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "compact", decoded.TaskMode)
}
```

#### Test 4: ValidateTaskMode - Valid Modes
```go
func TestValidateTaskMode_Valid(t *testing.T) {
	tests := []string{"regular", "review", "compact", "planning", ""}

	for _, mode := range tests {
		t.Run(mode, func(t *testing.T) {
			err := ValidateTaskMode(mode)
			assert.NoError(t, err)
		})
	}
}
```

#### Test 5: ValidateTaskMode - Invalid Modes
```go
func TestValidateTaskMode_Invalid(t *testing.T) {
	tests := []string{"invalid", "REGULAR", "Review", "compact-mode"}

	for _, mode := range tests {
		t.Run(mode, func(t *testing.T) {
			err := ValidateTaskMode(mode)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid task mode")
		})
	}
}
```

#### Test 6: JSON Unmarshaling from Client
```go
func TestSendMessageParams_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantMode *string
		wantErr  bool
	}{
		{
			name:     "with task_mode",
			json:     `{"message":"Hello","task_mode":"review"}`,
			wantMode: strPtr("review"),
			wantErr:  false,
		},
		{
			name:     "without task_mode",
			json:     `{"message":"Hello"}`,
			wantMode: nil,
			wantErr:  false,
		},
		{
			name:     "task_mode null",
			json:     `{"message":"Hello","task_mode":null}`,
			wantMode: nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params SendMessageParams
			err := json.Unmarshal([]byte(tt.json), &params)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantMode == nil {
				assert.Nil(t, params.TaskMode)
			} else {
				require.NotNil(t, params.TaskMode)
				assert.Equal(t, *tt.wantMode, *params.TaskMode)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
```

### Coverage Target

- **SendMessageParams**: 100% coverage
- **SendMessageResult**: 100% coverage
- **ValidateTaskMode**: 100% coverage
- **Overall package**: Maintain ≥90% coverage

## Error Handling

### Invalid Task Mode

**Request:**
```json
{
  "method": "send_message",
  "params": {
    "message": "Hello",
    "task_mode": "invalid"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32602,
    "message": "invalid task mode: invalid (valid: regular, review, compact, planning)"
  }
}
```

**Error Code:** `-32602` (Invalid Params)

## Migration Path

**Phase 4.1 (This FRD):**
- ✅ Add fields to protocol
- ✅ Add validation
- ✅ Write tests

**Phase 4.2 (P4.2):**
- Use fields in processor
- Call `conversation.SetTaskMode()`
- Return current mode in result

**Phase 4.3 (P4.3):**
- Integration tests
- E2E tests with real WebSocket

## Performance Impact

**Minimal:**
- Two string fields per request/response
- Validation is O(1) map lookup
- No new allocations in hot path

## Security Considerations

**Validation:**
- Mode names must be in allowlist
- No arbitrary strings accepted
- Clear error messages don't leak info

**No New Attack Surface:**
- Task modes already validated in core
- Protocol just passes string through

## Documentation Updates

### Protocol Documentation

Update `docs/packages/protocol.md`:

**Add to "Conversation Methods" section:**

```markdown
#### task_mode Field

The `task_mode` field allows clients to specify or query the task mode:

**Valid Modes:**
- `regular`: Full-featured interactive coding (default)
- `review`: Read-only code analysis
- `compact`: Quick queries with minimal tools
- `planning`: Task decomposition

**Request Example:**
\```json
{
  "method": "send_message",
  "params": {
    "message": "Review this function",
    "task_mode": "review"
  }
}
\```

**Response:**
\```json
{
  "result": {
    "conversation_id": "conv-123",
    "turn_id": "turn-456",
    "task_mode": "review"
  }
}
\```

**Behavior:**
- If `task_mode` is specified, switches the conversation to that mode
- If `task_mode` is omitted, uses conversation's current mode
- Response always includes current `task_mode`
```

## Definition of Done

- [x] `TaskMode` field added to `SendMessageParams` (optional)
- [x] `TaskMode` field added to `SendMessageResult` (required)
- [x] `ValidateTaskMode()` function implemented
- [x] `ValidTaskModes` map defined
- [x] Unit tests written (6 tests covering all scenarios)
- [ ] All tests pass
- [ ] Test coverage ≥90% for new code
- [ ] `make lint` passes (zero errors)
- [ ] Race detector clean (`go test -race`)
- [ ] Godoc complete on all exports
- [ ] Backward compatibility verified

## Success Criteria

**Functional:**
1. Protocol accepts `task_mode` in requests
2. Protocol returns `task_mode` in responses
3. Validation rejects invalid modes
4. Old clients work without changes
5. New clients can use task modes

**Quality:**
1. Test coverage ≥90%
2. No lint errors
3. No race conditions
4. Clear godoc on all exports

**Performance:**
1. No measurable overhead
2. Validation < 1μs

## References

- [ROADMAP P4.1](../../task-modes/ROADMAP.md#p41-update-protocol-with-task-mode-field)
- [Task Modes Specification](../../task-modes/specification.md)
- [Protocol Package Docs](../../../docs/packages/protocol.md)
- [Core Package Docs](../../../docs/packages/core.md)

## Changelog

- **2025-10-12**: Initial version
