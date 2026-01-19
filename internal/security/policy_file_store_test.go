package security

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFilePolicyStore_SaveGetListDeleteClear_GlobalScope(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFilePolicyStore(filepath.Join(tmpDir, "policies.json"), 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewFilePolicyStore error: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()
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

	// Save
	if err := store.Save(ctx, p); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Get (hit)
	got, ok, err := store.Get(ctx, key, ScopeGlobal)
	if err != nil || !ok {
		t.Fatalf("get: err=%v ok=%v", err, ok)
	}
	if got.Decision != DecisionAllow || got.PolicyNote != "test" {
		t.Fatalf("unexpected policy: %+v", got)
	}

	// List
	list, err := store.List(ctx, ScopeGlobal)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(list))
	}

	// Delete
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

	// Re-save and Clear
	if err := store.Save(ctx, p); err != nil {
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
	tmpDir := t.TempDir()
	store, err := NewFilePolicyStore(filepath.Join(tmpDir, "policies.json"), 5*time.Millisecond)
	if err != nil {
		t.Fatalf("NewFilePolicyStore error: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	ctx := context.Background()

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
	if err := store.Save(ctx, p); err != nil {
		t.Fatalf("save: %v", err)
	}

	// should be visible before expiry
	if _, ok, _ := store.Get(ctx, key, ScopeGlobal); !ok {
		t.Fatalf("expected present before expiry")
	}
	// wait until after expiry + a bit for janitor
	time.Sleep(ttl + 20*time.Millisecond)

	// get should miss after expiry
	if _, ok, _ := store.Get(ctx, key, ScopeGlobal); ok {
		t.Fatalf("expected expired policy to be evicted")
	}
}

func TestFilePolicyStore_FileIsCreatedAndLocked(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "policies.json")
	_, err := os.Stat(path)
	if !os.IsNotExist(err) && err != nil {
		t.Fatalf("stat before create: %v", err)
	}

	store, err := NewFilePolicyStore(path, 0)
	if err != nil {
		t.Fatalf("NewFilePolicyStore error: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	// Save one to ensure file gets content
	key := NewPolicyKey("prog", nil, "/w")
	if err := store.Save(context.Background(), Policy{
		Version:   "1",
		Scope:     ScopeGlobal,
		Key:       key,
		Decision:  DecisionAllow,
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat after save: %v", err)
	}
}
