package main

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

// createAutoApproveHandler returns an approval handler that auto-approves all requests.
func createAutoApproveHandler() safety.ApprovalHandler {
	return func(_ context.Context, req safety.ApprovalRequest) safety.ApprovalResponse {
		return safety.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Reason:    "auto-approved",
		}
	}
}

// createDenyHandler returns an approval handler that denies all requests with given reason.
func createDenyHandler(reason string) safety.ApprovalHandler {
	return func(_ context.Context, req safety.ApprovalRequest) safety.ApprovalResponse {
		return safety.ApprovalResponse{
			RequestID: req.ID,
			Approved:  false,
			Reason:    reason,
		}
	}
}

// createTUIApprovalHandler returns an approval handler that shows TUI dialog.
func createTUIApprovalHandler(ui *adapters.PureTTY) safety.ApprovalHandler {
	return func(ctx context.Context, req safety.ApprovalRequest) safety.ApprovalResponse {
		return ui.ShowApprovalDialog(ctx, req)
	}
}
