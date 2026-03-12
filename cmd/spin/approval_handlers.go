package main

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

// createAutoApproveHandler returns an approval handler that auto-approves all requests.
func createAutoApproveHandler() security.ApprovalHandler {
	return func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Reason:    "auto-approved",
		}
	}
}

// createDenyHandler returns an approval handler that denies all requests with given reason.
func createDenyHandler(reason string) security.ApprovalHandler {
	return func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  false,
			Reason:    reason,
		}
	}
}

// createTUIApprovalHandler returns an approval handler that shows TUI dialog.
func createTUIApprovalHandler(ui *adapters.PureTTY) security.ApprovalHandler {
	return func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		return ui.ShowApprovalDialog(ctx, req)
	}
}
