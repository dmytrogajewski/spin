# BUG-20251029001449: Approval Dialog Not Visible in TUI

**Status**: Open  
**Priority**: High  
**Component**: UI/TUI Approval System  
**Created**: 2025-10-29  

## Summary

Approval dialogs in TUI mode are not visible to users. The keyboard shortcuts work (Shift+A approves, Shift+D denies), but users cannot see the approval prompt in the status bar, making the system unusable.

## Root Cause

Race condition between approval prompt rendering and event-driven status bar updates:

1. `ShowApprovalDialog()` calls `showApprovalStatus(req)` which renders: `"Executing: \"command\" [A]pprove [D]eny"`
2. ApprovalService emits `EventCommandApproval` event (`security/approval.go:184`)
3. Event is processed by TUI event loop → `ui.ProcessEvent()` (`tui.go`)
4. `ProcessEvent()` calls `statusAggregator.ProcessEvent(event)` (`puretty.go:397`)
5. Aggregator sets agent state to `"Waiting approval"` (`status/aggregator.go:95`)
6. `ProcessEvent()` calls `updateStatusBar()` which gets status from StatusManager
7. **`updateStatusBar()` overwrites the approval prompt with regular status bar format**

The approval prompt is rendered correctly but immediately overwritten by the event-driven status update.

## Code Locations

- **Approval rendering**: `/home/dmitriy/sources/spin/internal/ui/adapters/puretty.go:683` (`showApprovalStatus`)
- **Event emission**: `/home/dmitriy/sources/spin/internal/security/approval.go:184` (`emitApprovalRequest`)
- **Event processing**: `/home/dmitriy/sources/spin/internal/ui/adapters/puretty.go:397` (`ProcessEvent`)
- **Aggregator**: `/home/dmitriy/sources/spin/internal/ui/status/aggregator.go:95` (sets "Waiting approval")
- **Status update**: `/home/dmitriy/sources/spin/internal/ui/adapters/puretty.go:425` (`updateStatusBar`)

## Expected Behavior

When approval is requested, the status bar should display:
```
Executing: "rm -rf /tmp/build" [A]pprove [D]eny
```

And remain visible until user responds.

## Actual Behavior

The approval prompt is rendered but immediately overwritten by the regular status bar showing agent state "Waiting approval".

## Proposed Fix

Add mode-aware status rendering in `updateStatusBar()`:

```go
func (u *PureTTY) updateStatusBar() {
    if u.statusManager == nil || u.statusRenderer == nil {
        return
    }

    // Skip status updates when in approval mode
    // The approval dialog manages its own status bar display
    u.mu.Lock()
    mode := u.mode
    u.mu.Unlock()
    
    if mode == ModeApproval {
        return
    }

    // ... rest of existing code
}
```

This prevents the status bar from being overwritten while approval dialog is active.

## Impact

- **Severity**: High - approval system is unusable without visual feedback
- **Users affected**: All TUI users with approval-required commands
- **Workaround**: None (users cannot see what they're approving)

## Acceptance Criteria

- [ ] Approval prompt visible in status bar when approval is requested
- [ ] Status bar shows: `Executing: "<command>" [A]pprove [D]eny`
- [ ] Keyboard shortcuts work (A/a to approve, D/d to deny, ESC to cancel)
- [ ] After approval/denial, status bar returns to normal display
- [ ] Status bar updates from events do not overwrite approval prompt
- [ ] Add test verifying approval prompt is not overwritten by events

## Related Files

- `internal/ui/adapters/puretty.go` - Status bar and approval dialog integration
- `internal/ui/overlay/approval.go` - Approval dialog component
- `internal/security/approval.go` - Approval service and event emission
- `internal/ui/status/aggregator.go` - Event-driven status updates
- `cmd/spin/tui.go` - TUI mode initialization
