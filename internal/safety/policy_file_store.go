package safety

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/storage"
)

const fileStoreEvictionInterval = 30 * time.Second

// ErrPathIsRequired is a sentinel error.
var ErrPathIsRequired = errors.New("path is required")

// FilePolicyStore persists global-scope policies to a single JSON file with
// atomic writes and advisory locking. Non-global scopes are kept in-memory only.
type FilePolicyStore struct {
	path     string
	mu       sync.RWMutex
	byScope  map[string]map[string]Policy // scope -> keyStr -> policy (in-memory view).
	stopCh   chan struct{}
	interval time.Duration
}

// NewFilePolicyStore creates a file-backed policy store.
// The janitor evictionInterval controls how often expired entries are pruned from memory.
// If interval <= 0, a default of 30s is used.
// The context controls the initial load from disk.
func NewFilePolicyStore(ctx context.Context, path string, evictionInterval time.Duration) (*FilePolicyStore, error) {
	if path == "" {
		return nil, ErrPathIsRequired
	}

	err := os.MkdirAll(filepath.Dir(path), 0o750)
	if err != nil {
		return nil, fmt.Errorf("create policy store directory: %w", err)
	}

	if evictionInterval <= 0 {
		evictionInterval = fileStoreEvictionInterval
	}

	s := &FilePolicyStore{
		path:     path,
		byScope:  make(map[string]map[string]Policy),
		stopCh:   make(chan struct{}),
		interval: evictionInterval,
	}
	// Best-effort load existing file.
	err = s.loadFromDisk(ctx)
	if err != nil {
		return nil, err
	}

	go s.janitor(context.WithoutCancel(ctx))

	return s, nil
}

func (s *FilePolicyStore) janitor(ctx context.Context) {
	t := time.NewTicker(s.interval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			s.removeExpired(ctx)
		case <-s.stopCh:
			return
		}
	}
}

// Close stops the janitor goroutine and releases resources.
func (s *FilePolicyStore) Close() error {
	close(s.stopCh)

	return nil
}

func (s *FilePolicyStore) removeExpired(ctx context.Context) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for scope, m := range s.byScope {
		for k, p := range m {
			if p.ExpiresAt != nil && now.After(*p.ExpiresAt) {
				delete(m, k)
			}
		}

		if len(m) == 0 {
			delete(s.byScope, scope)
		}
	}
	// Persist global scope after eviction (best-effort).
	_ = s.persistGlobalLocked(ctx)
}

// Save implements the Save operation.
func (s *FilePolicyStore) Save(ctx context.Context, p Policy) error {
	keyStr := keyString(p.Key)

	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.byScope[p.Scope]
	if m == nil {
		m = make(map[string]Policy)
		s.byScope[p.Scope] = m
	}

	m[keyStr] = p
	if p.Scope == ScopeGlobal {
		return s.persistGlobalLocked(ctx)
	}

	return nil
}

// Get implements the Get operation.
func (s *FilePolicyStore) Get(ctx context.Context, key PolicyKey, scope string) (Policy, bool, error) {
	if err := ctx.Err(); err != nil {
		return Policy{}, false, fmt.Errorf("get policy: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := getFromScopeMap(s.byScope, key, scope)

	return p, ok, nil
}

// List implements the List operation.
func (s *FilePolicyStore) List(ctx context.Context, scope string) ([]Policy, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return listFromScopeMap(s.byScope, scope), nil
}

// Delete implements the Delete operation.
func (s *FilePolicyStore) Delete(ctx context.Context, key PolicyKey, scope string) (bool, error) {
	keyStr := keyString(key)

	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.byScope[scope]
	if m == nil {
		return false, nil
	}

	if _, ok := m[keyStr]; !ok {
		return false, nil
	}

	delete(m, keyStr)

	if len(m) == 0 {
		delete(s.byScope, scope)
	}

	if scope == ScopeGlobal {
		if err := s.persistGlobalLocked(ctx); err != nil {
			return true, err
		}
	}

	return true, nil
}

// Clear implements the Clear operation.
func (s *FilePolicyStore) Clear(ctx context.Context, scope string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.byScope[scope]
	if m == nil {
		return 0, nil
	}

	n := len(m)

	delete(s.byScope, scope)

	if scope == ScopeGlobal {
		err := s.persistGlobalLocked(ctx)
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

// persistGlobalLocked writes the global scope policies to disk atomically
// with an advisory lock. Caller must hold s.mu.
func (s *FilePolicyStore) persistGlobalLocked(ctx context.Context) error {
	global := s.byScope[ScopeGlobal]
	if global == nil {
		global = map[string]Policy{}
	}

	payload := struct {
		Global map[string]Policy `json:"global"`
	}{
		Global: global,
	}

	data, err := json.MarshalIndent(payload, "", "\t")
	if err != nil {
		return fmt.Errorf("marshal policies: %w", err)
	}

	tmp := s.path + ".tmp"

	// Acquire advisory lock on the target file (create if not exists).
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open policy file: %w", err)
	}
	defer f.Close()

	fd := storage.SafeFlockFd(f.Fd())

	err = storage.FlockExclusiveWithContext(ctx, fd)
	if err != nil {
		return fmt.Errorf("lock policy file: %w", err)
	}

	defer func() { _ = storage.FlockUnlock(fd) }()

	// Write temp, then rename over target for atomicity.
	err = os.WriteFile(tmp, data, 0o600)
	if err != nil {
		return fmt.Errorf("write temp policy file: %w", err)
	}

	err = os.Rename(tmp, s.path)
	if err != nil {
		return fmt.Errorf("rename policy file: %w", err)
	}

	return nil
}

// loadFromDisk loads global scope from disk into memory (best-effort).
func (s *FilePolicyStore) loadFromDisk(ctx context.Context) error {
	// Open with shared lock to read.
	f, err := os.OpenFile(s.path, os.O_RDONLY, 0o600)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("open policy file for reading: %w", err)
	}
	defer f.Close()

	fd := storage.SafeFlockFd(f.Fd())

	err = storage.FlockSharedWithContext(ctx, fd)
	if err != nil {
		return fmt.Errorf("shared lock policy file: %w", err)
	}

	defer func() { _ = storage.FlockUnlock(fd) }()

	var payload struct {
		Global map[string]Policy `json:"global"`
	}

	dec := json.NewDecoder(f)

	err = dec.Decode(&payload)
	if err != nil {
		return fmt.Errorf("decode policy file: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.byScope == nil {
		s.byScope = make(map[string]map[string]Policy)
	}

	if payload.Global != nil {
		s.byScope[ScopeGlobal] = payload.Global
	}

	return nil
}
