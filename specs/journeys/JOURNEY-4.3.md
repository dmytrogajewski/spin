# JOURNEY-4.3: ACP Event Processing

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 4.3 |
| Title | Wire ACP Event Processing |
| User Story | As an ACP client, content deltas and plan notifications stream in real-time. |
| Paper Section | Entry layer — ACP protocol event streaming |
| Roadmap Item | JOURNEY-4.3 (10 functions) |

## Implementation

### Files Modified
- `internal/protocol/acp/agent.go` — Wire `processEvents()` goroutine in `promptWithConversation()` via second event subscription; wire `sendPlanNotifications()` after turn completion with conversation output
