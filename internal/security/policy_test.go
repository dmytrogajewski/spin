package security

import (
	"context"
	"testing"
	"time"
)

func TestNewPolicyKey_Normalization(t *testing.T) {
	key := NewPolicyKey("  /bin/echo  ", []string{"  hello", "world  ", "foo   bar", ""}, " /tmp ")
	if key.Program != "/bin/echo" {
		t.Fatalf("Program normalization failed: %q", key.Program)
	}
	if key.WorkDir != "/tmp" {
		t.Fatalf("WorkDir normalization failed: %q", key.WorkDir)
	}
	wantArgs := []string{"hello", "world", "foo bar", ""}
	if len(key.Args) != len(wantArgs) {
		t.Fatalf("Args len mismatch: got=%d want=%d", len(key.Args), len(wantArgs))
	}
	for i := range wantArgs {
		if key.Args[i] != wantArgs[i] {
			t.Fatalf("Arg[%d]=%q want=%q", i, key.Args[i], wantArgs[i])
		}
	}
}

func TestMemoryPolicyStore_SaveGetListDelete(t *testing.T) {
	store := NewMemoryPolicyStore(10 * time.Millisecond)
	t.Cleanup(func() { store.Close() })
	ctx := context.Background()

	key := NewPolicyKey("/bin/echo", []string{"hello"}, "/tmp")
	now := time.Now()
	p := Policy{
		Version:   "1",
		Scope:     ScopeSession,
		Key:       key,
		Decision:  DecisionAllow,
		CreatedAt: now,
	}
	if err := store.Save(ctx, p); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	got, ok, err := store.Get(ctx, key, ScopeSession)
	if err != nil || !ok {
		t.Fatalf("Get error=%v ok=%v", err, ok)
	}
	if got.Decision != DecisionAllow {
		t.Fatalf("Decision mismatch: %q", got.Decision)
	}
	list, err := store.List(ctx, ScopeSession)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List size got=%d want=1", len(list))
	}
	deleted, err := store.Delete(ctx, key, ScopeSession)
	if err != nil || !deleted {
		t.Fatalf("Delete error=%v deleted=%v", err, deleted)
	}
	list, _ = store.List(ctx, ScopeSession)
	if len(list) != 0 {
		t.Fatalf("List after delete got=%d want=0", len(list))
	}
}

func TestMemoryPolicyStore_TTLExpiry(t *testing.T) {
	store := NewMemoryPolicyStore(5 * time.Millisecond)
	t.Cleanup(func() { store.Close() })
	ctx := context.Background()
	key := NewPolicyKey("/bin/echo", []string{"x"}, "/tmp")
	exp := time.Now().Add(10 * time.Millisecond)
	p := Policy{
		Version:   "1",
		Scope:     ScopeSession,
		Key:       key,
		Decision:  DecisionAllow,
		CreatedAt: time.Now(),
		ExpiresAt: &exp,
	}
	if err := store.Save(ctx, p); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	// Should be present immediately
	if _, ok, _ := store.Get(ctx, key, ScopeSession); !ok {
		t.Fatalf("expected policy to be present before expiry")
	}
	// Wait for expiry + janitor tick
	time.Sleep(25 * time.Millisecond)
	if _, ok, _ := store.Get(ctx, key, ScopeSession); ok {
		t.Fatalf("expected policy to be expired")
	}
}

func TestMemoryPolicyStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryPolicyStore(5 * time.Millisecond)
	t.Cleanup(func() { store.Close() })
	ctx := context.Background()

	key := NewPolicyKey("/bin/echo", []string{"hello"}, "/tmp")
	p := Policy{
		Version:   "1",
		Scope:     ScopeSession,
		Key:       key,
		Decision:  DecisionAllow,
		CreatedAt: time.Now(),
	}
	if err := store.Save(ctx, p); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	const workers = 16
	done := make(chan struct{}, workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer func() { done <- struct{}{} }()

			for j := 0; j < 100; j++ {
				if _, _, err := store.Get(ctx, key, ScopeSession); err != nil {
					t.Errorf("Get error: %v", err)
				}
				if _, err := store.List(ctx, ScopeSession); err != nil {
					t.Errorf("List error: %v", err)
				}
			}
		}()
	}

	for i := 0; i < workers; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for workers")
		}
	}
}
