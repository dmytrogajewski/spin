package core

import (
	"context"
)

// requestApproval requests user approval for a command.
// It delegates to the centralized ApprovalService which handles
// event emission, timeouts, validation, and modified command processing.
func (a *Agent) requestApproval(ctx context.Context, cmd *Command, reason string) bool {
	_, approved, _ := a.approvalService.RequestApproval(ctx, Operation{
		Command: cmd,
		Reason:  reason,
		WorkDir: a.context.WorkDir,
	})
	return approved
}
