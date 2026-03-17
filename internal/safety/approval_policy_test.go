package safety

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
)

func TestApprovalService_PolicyShortCircuit(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(10)
	store := NewMemoryPolicyStore(10 * time.Millisecond)

	t.Cleanup(func() { store.Close() })

	svc := NewApprovalServiceWithConfig(ApprovalServiceConfig{
		Handler:           nil, // should not be called on policy hit.
		Emitter:           emitter,
		Validator:         NewValidator(),
		Store:             store,
		SessionDefaultTTL: 0,
		GlobalDefaultTTL:  0,
	})

	cmd := &Command{Program: "/bin/echo", Args: []string{"hello"}, WorkDir: "/tmp"}
	key := NewPolicyKey(cmd.Program, cmd.Args, cmd.WorkDir)

	p := Policy{
		Version:   "1",
		Scope:     ScopeSession,
		Key:       key,
		Decision:  DecisionAllow,
		CreatedAt: time.Now(),
	}

	err := store.Save(context.Background(), p)
	if err != nil {
		t.Fatalf("save policy: %v", err)
	}

	_, ok, err := store.Get(context.Background(), key, ScopeSession)
	if err != nil || !ok {
		t.Fatalf("policy not present: err=%v ok=%v", err, ok)
	}

	_, approved, err := svc.RequestApproval(context.Background(), NewOperation(cmd, "test", "/tmp"))
	if err != nil {
		t.Fatalf("RequestApproval error: %v", err)
	}

	if !approved {
		t.Fatalf("expected approved via policy")
	}
}

func TestApprovalService_PersistOnApprove(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(10)
	store := NewMemoryPolicyStore(10 * time.Millisecond)

	t.Cleanup(func() { store.Close() })
	// Handler that approves with session scope and 50ms TTL.
	ttl := 50 * time.Millisecond
	handler := func(_ context.Context, req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Scope:     ScopeSession,
			TTL:       &ttl,
			Timestamp: time.Now(),
		}
	}
	svc := NewApprovalServiceWithConfig(ApprovalServiceConfig{
		Handler:           handler,
		Emitter:           emitter,
		Validator:         NewValidator(),
		Store:             store,
		SessionDefaultTTL: 0,
		GlobalDefaultTTL:  0,
	})

	cmd := &Command{Program: "/bin/echo", Args: []string{"ok"}, WorkDir: "/tmp"}
	// subscribe to events to ensure policy_saved is emitted.
	subID, ch, _ := emitter.Subscribe()
	defer emitter.Unsubscribe(subID)

	_, approved, err := svc.RequestApproval(context.Background(), NewOperation(cmd, "test", "/tmp"))
	if err != nil || !approved {
		t.Fatalf("approval failed: err=%v approved=%v", err, approved)
	}

	key := NewPolicyKey(cmd.Program, cmd.Args, cmd.WorkDir)
	if _, ok, _ := store.Get(context.Background(), key, ScopeSession); !ok {
		t.Fatalf("expected persisted session policy")
	}

	// Drain events and look for policy_saved.
	foundSaved := false

	drain := true
	for drain {
		select {
		case ev := <-ch:
			if ev.Type == events.EventPolicySaved {
				foundSaved = true
			}
		default:
			drain = false
		}
	}

	if !foundSaved {
		t.Fatalf("expected EventPolicySaved to be emitted")
	}
}

func TestApprovalService_ApproveOnceDoesNotPersist(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(10)
	store := NewMemoryPolicyStore(10 * time.Millisecond)

	t.Cleanup(func() { store.Close() })

	// Handler approves with scope=once (no persistence expected).
	handler := func(_ context.Context, req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Scope:     ScopeOnce,
		}
	}

	svc := NewApprovalServiceWithConfig(ApprovalServiceConfig{
		Handler:           handler,
		Emitter:           emitter,
		Validator:         NewValidator(),
		Store:             store,
		SessionDefaultTTL: 0,
		GlobalDefaultTTL:  0,
	})

	cmd := &Command{Program: "/bin/rm", Args: []string{"-rf", "/tmp/x"}, WorkDir: "/tmp"}

	_, approved, err := svc.RequestApproval(context.Background(), NewOperation(cmd, "test once", "/tmp"))
	if err != nil || !approved {
		t.Fatalf("approval failed: err=%v approved=%v", err, approved)
	}

	// ScopeOnce should NOT create any persisted policy.
	key := NewPolicyKey(cmd.Program, cmd.Args, cmd.WorkDir)
	if _, ok, _ := store.Get(context.Background(), key, ScopeSession); ok {
		t.Fatalf("ScopeOnce should not persist session policy")
	}

	if _, ok, _ := store.Get(context.Background(), key, ScopeGlobal); ok {
		t.Fatalf("ScopeOnce should not persist global policy")
	}
}

func TestApprovalService_OnceScopeReasks(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(10)
	store := NewMemoryPolicyStore(10 * time.Millisecond)

	t.Cleanup(func() { store.Close() })

	var calls int

	handler := func(_ context.Context, req ApprovalRequest) ApprovalResponse {
		calls++

		return ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Scope:     ScopeOnce,
		}
	}

	svc := NewApprovalServiceWithConfig(ApprovalServiceConfig{
		Handler:           handler,
		Emitter:           emitter,
		Validator:         NewValidator(),
		Store:             store,
		SessionDefaultTTL: 0,
		GlobalDefaultTTL:  0,
	})

	cmd := &Command{Program: "/bin/rm", Args: []string{"-rf", "/tmp/y"}, WorkDir: "/tmp"}
	op := NewOperation(cmd, "test once reask", "/tmp")

	// First approval should call handler.
	_, approved, err := svc.RequestApproval(context.Background(), op)
	if err != nil || !approved {
		t.Fatalf("first approval failed: err=%v approved=%v", err, approved)
	}

	if calls != 1 {
		t.Fatalf("expected handler to be called once, got %d", calls)
	}

	// Second approval for same operation should call handler again (no short-circuit).
	_, approved, err = svc.RequestApproval(context.Background(), op)
	if err != nil || !approved {
		t.Fatalf("second approval failed: err=%v approved=%v", err, approved)
	}

	if calls != 2 {
		t.Fatalf("expected handler to be called twice for ScopeOnce, got %d", calls)
	}
}

func TestApprovalService_GlobalScopePersistsAndShortCircuits(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(10)
	store := NewMemoryPolicyStore(10 * time.Millisecond)

	t.Cleanup(func() { store.Close() })

	// Track handler invocations to prove short-circuit on second call.
	var calls int

	handler := func(_ context.Context, req ApprovalRequest) ApprovalResponse {
		calls++

		return ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Scope:     ScopeGlobal,
		}
	}

	svc := NewApprovalServiceWithConfig(ApprovalServiceConfig{
		Handler:           handler,
		Emitter:           emitter,
		Validator:         NewValidator(),
		Store:             store,
		SessionDefaultTTL: 0,
		GlobalDefaultTTL:  30 * time.Minute,
	})

	cmd := &Command{Program: "/usr/bin/echo", Args: []string{"hello"}, WorkDir: "/workspace"}
	op := NewOperation(cmd, "global approve", "/workspace")

	// First approval should go through handler and persist policy.
	_, approved, err := svc.RequestApproval(context.Background(), op)
	if err != nil || !approved {
		t.Fatalf("first approval failed: err=%v approved=%v", err, approved)
	}

	if calls != 1 {
		t.Fatalf("expected handler to be called once, got %d", calls)
	}

	key := NewPolicyKey(cmd.Program, cmd.Args, cmd.WorkDir)
	if _, ok, _ := store.Get(context.Background(), key, ScopeGlobal); !ok {
		t.Fatalf("expected global policy to be persisted")
	}

	// Second approval for same operation should short-circuit via policy (no extra handler call).
	_, approved, err = svc.RequestApproval(context.Background(), op)
	if err != nil || !approved {
		t.Fatalf("second approval failed: err=%v approved=%v", err, approved)
	}

	if calls != 1 {
		t.Fatalf("expected handler not to be called on policy short-circuit, got %d calls", calls)
	}
}

func TestApprovalService_RevocationReasks(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(10)
	store := NewMemoryPolicyStore(10 * time.Millisecond)

	t.Cleanup(func() { store.Close() })

	var calls int

	handler := func(_ context.Context, req ApprovalRequest) ApprovalResponse {
		calls++

		return ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Scope:     ScopeGlobal,
		}
	}

	svc := NewApprovalServiceWithConfig(ApprovalServiceConfig{
		Handler:           handler,
		Emitter:           emitter,
		Validator:         NewValidator(),
		Store:             store,
		SessionDefaultTTL: 0,
		GlobalDefaultTTL:  30 * time.Minute,
	})

	cmd := &Command{Program: "/usr/bin/echo", Args: []string{"ok"}, WorkDir: "/workspace"}
	op := NewOperation(cmd, "global approve", "/workspace")
	key := NewPolicyKey(cmd.Program, cmd.Args, cmd.WorkDir)

	// First approval persists global policy.
	_, approved, err := svc.RequestApproval(context.Background(), op)
	if err != nil || !approved {
		t.Fatalf("first approval failed: err=%v approved=%v", err, approved)
	}

	if calls != 1 {
		t.Fatalf("expected handler to be called once, got %d", calls)
	}

	// Remove policy (revocation).
	deleted, err := store.Delete(context.Background(), key, ScopeGlobal)
	if err != nil || !deleted {
		t.Fatalf("expected global policy to be deleted, deleted=%v err=%v", deleted, err)
	}

	// Second approval should call handler again because policy was revoked.
	_, approved, err = svc.RequestApproval(context.Background(), op)
	if err != nil || !approved {
		t.Fatalf("second approval after revocation failed: err=%v approved=%v", err, approved)
	}

	if calls != 2 {
		t.Fatalf("expected handler to be called again after revocation, got %d calls", calls)
	}
}
