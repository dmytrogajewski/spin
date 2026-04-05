package acp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/safety"
)

var errBoom = errors.New("boom")

// newOutcomeSelected creates a RequestPermissionOutcome with the Selected variant and the given OptionId.
func newOutcomeSelected(id acp.PermissionOptionId) acp.RequestPermissionOutcome {
	outcome := acp.NewRequestPermissionOutcomeSelected()
	outcome.Selected.OptionId = id

	return outcome
}

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

func TestApprovalHandler_NoActiveSession(t *testing.T) {
	t.Parallel()

	handler := NewApprovalHandler(&SpinACPAgent{}, time.Second)

	resp := handler.HandleApprovalRequest(context.Background(), safety.ApprovalRequest{
		ID: "test-no-session",
	})

	if resp.Approved {
		t.Fatalf("expected request to be denied when no active session")
	}

	if resp.Reason == "" {
		t.Fatalf("expected reason to be set")
	}
}

func TestApprovalHandler_MapsAllowOnceAndAlways(t *testing.T) {
	t.Parallel()

	conn := &mockACPConnection{
		resp: acp.RequestPermissionResponse{
			Outcome: newOutcomeSelected(acp.PermissionOptionId("allow_once")),
		},
	}

	agent := &SpinACPAgent{
		connection: conn,
	}
	handler := NewApprovalHandler(agent, 5*time.Second)
	handler.SetActiveSession(acp.SessionId("sess-1"))

	// First: allow_once.
	resp := handler.HandleApprovalRequest(context.Background(), safety.ApprovalRequest{
		ID: "req-allow-once",
	})
	if !resp.Approved || resp.Scope != safety.ScopeOnce {
		t.Fatalf("expected allow_once to approve with ScopeOnce, got approved=%v scope=%q", resp.Approved, resp.Scope)
	}

	// Second: allow_always.
	conn.resp = acp.RequestPermissionResponse{
		Outcome: newOutcomeSelected(acp.PermissionOptionId("allow_always")),
	}

	resp = handler.HandleApprovalRequest(context.Background(), safety.ApprovalRequest{
		ID: "req-allow-always",
	})
	if !resp.Approved || resp.Scope != safety.ScopeGlobal {
		t.Fatalf("expected allow_always to approve with ScopeGlobal, got approved=%v scope=%q", resp.Approved, resp.Scope)
	}

	if len(conn.requests) == 0 {
		t.Fatalf("expected RequestPermission to be called at least once")
	}
}

func TestApprovalHandler_DenyAndCancelPaths(t *testing.T) {
	t.Parallel()

	conn := &mockACPConnection{
		resp: acp.RequestPermissionResponse{
			Outcome: newOutcomeSelected(acp.PermissionOptionId("deny")),
		},
	}

	agent := &SpinACPAgent{
		connection: conn,
	}
	handler := NewApprovalHandler(agent, 10*time.Millisecond)
	handler.SetActiveSession(acp.SessionId("sess-2"))

	// Deny path.
	resp := handler.HandleApprovalRequest(context.Background(), safety.ApprovalRequest{
		ID: "req-deny",
	})
	if resp.Approved {
		t.Fatalf("expected deny option to result in not approved")
	}

	// Cancel / error path.
	conn.err = errBoom

	resp = handler.HandleApprovalRequest(context.Background(), safety.ApprovalRequest{
		ID: "req-error",
	})
	if resp.Approved {
		t.Fatalf("expected error path to result in not approved")
	}

	if resp.Reason == "" {
		t.Fatalf("expected error reason to be populated")
	}
}
