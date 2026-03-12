package acp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/security"
)

type mockACPConnection struct {
	requests []acp.RequestPermissionRequest
	resp     acp.RequestPermissionResponse
	err      error
}

func (m *mockACPConnection) SessionUpdate(_ context.Context, _ acp.SessionNotification) error {
	return nil
}

func (m *mockACPConnection) RequestPermission(_ context.Context, req acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	m.requests = append(m.requests, req)

	return m.resp, m.err
}

func TestACPApprovalHandler_NoActiveSession(t *testing.T) {
	handler := NewACPApprovalHandler(&SpinACPAgent{}, time.Second)

	resp := handler.HandleApprovalRequest(context.Background(), security.ApprovalRequest{
		ID: "test-no-session",
	})

	if resp.Approved {
		t.Fatalf("expected request to be denied when no active session")
	}

	if resp.Reason == "" {
		t.Fatalf("expected reason to be set")
	}
}

func TestACPApprovalHandler_MapsAllowOnceAndAlways(t *testing.T) {
	conn := &mockACPConnection{
		resp: acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected(acp.PermissionOptionId("allow_once")),
		},
	}

	agent := &SpinACPAgent{
		connection: conn,
	}
	handler := NewACPApprovalHandler(agent, 5*time.Second)
	handler.SetActiveSession(acp.SessionId("sess-1"))

	// First: allow_once.
	resp := handler.HandleApprovalRequest(context.Background(), security.ApprovalRequest{
		ID: "req-allow-once",
	})
	if !resp.Approved || resp.Scope != security.ScopeOnce {
		t.Fatalf("expected allow_once to approve with ScopeOnce, got approved=%v scope=%q", resp.Approved, resp.Scope)
	}

	// Second: allow_always.
	conn.resp = acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeSelected(acp.PermissionOptionId("allow_always")),
	}

	resp = handler.HandleApprovalRequest(context.Background(), security.ApprovalRequest{
		ID: "req-allow-always",
	})
	if !resp.Approved || resp.Scope != security.ScopeGlobal {
		t.Fatalf("expected allow_always to approve with ScopeGlobal, got approved=%v scope=%q", resp.Approved, resp.Scope)
	}

	if len(conn.requests) == 0 {
		t.Fatalf("expected RequestPermission to be called at least once")
	}
}

func TestACPApprovalHandler_DenyAndCancelPaths(t *testing.T) {
	conn := &mockACPConnection{
		resp: acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected(acp.PermissionOptionId("deny")),
		},
	}

	agent := &SpinACPAgent{
		connection: conn,
	}
	handler := NewACPApprovalHandler(agent, 10*time.Millisecond)
	handler.SetActiveSession(acp.SessionId("sess-2"))

	// Deny path.
	resp := handler.HandleApprovalRequest(context.Background(), security.ApprovalRequest{
		ID: "req-deny",
	})
	if resp.Approved {
		t.Fatalf("expected deny option to result in not approved")
	}

	// Cancel / error path.
	conn.err = errors.New("boom")

	resp = handler.HandleApprovalRequest(context.Background(), security.ApprovalRequest{
		ID: "req-error",
	})
	if resp.Approved {
		t.Fatalf("expected error path to result in not approved")
	}

	if resp.Reason == "" {
		t.Fatalf("expected error reason to be populated")
	}
}
