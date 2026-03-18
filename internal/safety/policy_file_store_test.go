package safety

// Journey: specs/journeys/JOURNEY-CTX-2.3.md.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFilePolicyStore_SaveGetListDeleteClear_GlobalScope(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	store, err := NewFilePolicyStore(t.Context(), filepath.Join(tmpDir, "policies.json"), 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewFilePolicyStore error: %v", err)
	}

	t.Cleanup(func() { store.Close() })

	ctx := t.Context()
	key := NewPolicyKey("/bin/echo", []string{"hello", "world"}, "/tmp")
	now := time.Now()
	p := Policy{
		Version:    "1",
		Scope:      ScopeGlobal,
		Key:        key,
		Decision:   DecisionAllow,
		PolicyNote: "test",
		CreatedAt:  now,
		ExpiresAt:  nil,
	}

	// Save.
	err = store.Save(ctx, p)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	// Get (hit).
	got, ok, err := store.Get(ctx, key, ScopeGlobal)
	if err != nil || !ok {
		t.Fatalf("get: err=%v ok=%v", err, ok)
	}

	if got.Decision != DecisionAllow || got.PolicyNote != "test" {
		t.Fatalf("unexpected policy: %+v", got)
	}

	// List.
	list, err := store.List(ctx, ScopeGlobal)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(list))
	}

	// Delete.
	deleted, err := store.Delete(ctx, key, ScopeGlobal)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	if !deleted {
		t.Fatalf("expected deleted=true")
	}

	list, _ = store.List(ctx, ScopeGlobal)
	if len(list) != 0 {
		t.Fatalf("expected empty after delete, got %d", len(list))
	}

	// Re-save and Clear.
	err = store.Save(ctx, p)
	if err != nil {
		t.Fatalf("re-save: %v", err)
	}

	n, err := store.Clear(ctx, ScopeGlobal)
	if err != nil {
		t.Fatalf("clear: %v", err)
	}

	if n != 1 {
		t.Fatalf("expected cleared=1, got %d", n)
	}
}

func TestFilePolicyStore_ExpiryEviction(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	store, err := NewFilePolicyStore(t.Context(), filepath.Join(tmpDir, "policies.json"), 5*time.Millisecond)
	if err != nil {
		t.Fatalf("NewFilePolicyStore error: %v", err)
	}

	t.Cleanup(func() { store.Close() })

	ctx := t.Context()

	ttl := 20 * time.Millisecond
	exp := time.Now().Add(ttl)

	key := NewPolicyKey("/usr/bin/rm", []string{"-rf", "/tmp/x"}, "/home/user")

	p := Policy{
		Version:   "1",
		Scope:     ScopeGlobal,
		Key:       key,
		Decision:  DecisionAllow,
		CreatedAt: time.Now(),
		ExpiresAt: &exp,
	}

	err = store.Save(ctx, p)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	// should be visible before expiry.
	if _, ok, _ := store.Get(ctx, key, ScopeGlobal); !ok {
		t.Fatalf("expected present before expiry")
	}
	// wait until after expiry + a bit for janitor.
	time.Sleep(ttl + 20*time.Millisecond)

	// get should miss after expiry.
	if _, ok, _ := store.Get(ctx, key, ScopeGlobal); ok {
		t.Fatalf("expected expired policy to be evicted")
	}
}

func TestFilePolicyStore_FileIsCreatedAndLocked(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "policies.json")

	_, err := os.Stat(path)
	if !os.IsNotExist(err) && err != nil {
		t.Fatalf("stat before create: %v", err)
	}

	store, err := NewFilePolicyStore(t.Context(), path, 0)
	if err != nil {
		t.Fatalf("NewFilePolicyStore error: %v", err)
	}

	t.Cleanup(func() { store.Close() })
	// Save one to ensure file gets content.
	key := NewPolicyKey("prog", nil, "/w")

	err = store.Save(t.Context(), Policy{
		Version:   "1",
		Scope:     ScopeGlobal,
		Key:       key,
		Decision:  DecisionAllow,
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	_, err = os.Stat(path)
	if err != nil {
		t.Fatalf("stat after save: %v", err)
	}
}

func TestFilePolicyStore_Save_CanceledContext(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	store, err := NewFilePolicyStore(t.Context(), filepath.Join(tmpDir, "policies.json"), 0)
	if err != nil {
		t.Fatalf("NewFilePolicyStore error: %v", err)
	}

	t.Cleanup(func() { store.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	key := NewPolicyKey("prog", nil, "/w")

	saveErr := store.Save(ctx, Policy{
		Version:   "1",
		Scope:     ScopeGlobal,
		Key:       key,
		Decision:  DecisionAllow,
		CreatedAt: time.Now(),
	})
	if saveErr == nil {
		t.Fatal("expected error from Save with canceled context")
	}

	if !isContextError(saveErr) {
		t.Fatalf("expected context error, got: %v", saveErr)
	}
}

func TestFilePolicyStore_Get_CanceledContext(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	store, err := NewFilePolicyStore(t.Context(), filepath.Join(tmpDir, "policies.json"), 0)
	if err != nil {
		t.Fatalf("NewFilePolicyStore error: %v", err)
	}

	t.Cleanup(func() { store.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	key := NewPolicyKey("prog", nil, "/w")

	_, _, getErr := store.Get(ctx, key, ScopeGlobal)
	if getErr == nil {
		t.Fatal("expected error from Get with canceled context")
	}

	if !isContextError(getErr) {
		t.Fatalf("expected context error, got: %v", getErr)
	}
}

// isContextError checks if the error chain includes [context.Canceled] or [context.DeadlineExceeded].
func isContextError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
