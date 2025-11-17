package security

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// filePolicyStore persists global-scope policies to a single JSON file with
// atomic writes and advisory locking. Non-global scopes are kept in-memory only.
type filePolicyStore struct {
	path     string
	mu       sync.RWMutex
	byScope  map[string]map[string]Policy // scope -> keyStr -> policy (in-memory view)
	stopCh   chan struct{}
	interval time.Duration
}

// NewFilePolicyStore creates a file-backed policy store.
// The janitor evictionInterval controls how often expired entries are pruned from memory.
// If interval <= 0, a default of 30s is used.
func NewFilePolicyStore(path string, evictionInterval time.Duration) (PolicyStore, error) {
	if path == "" {
		return nil, errors.New("path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if evictionInterval <= 0 {
		evictionInterval = 30 * time.Second
	}
	s := &filePolicyStore{
		path:     path,
		byScope:  make(map[string]map[string]Policy),
		stopCh:   make(chan struct{}),
		interval: evictionInterval,
	}
	// Best-effort load existing file
	err := s.loadFromDisk()
	if err != nil {
		return nil, err
	}
	go s.janitor()
	return s, nil
}

func (s *filePolicyStore) janitor() {
	t := time.NewTicker(s.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			s.removeExpired()
		case <-s.stopCh:
			return
		}
	}
}

func (s *filePolicyStore) removeExpired() {
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
	// Persist global scope after eviction
	_ = s.persistGlobalLocked()
}

func (s *filePolicyStore) Save(_ context.Context, p Policy) error {
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
		return s.persistGlobalLocked()
	}
	return nil
}

func (s *filePolicyStore) Get(_ context.Context, key PolicyKey, scope string) (Policy, bool, error) {
	keyStr := keyString(key)
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.byScope[scope]
	if m == nil {
		return Policy{}, false, nil
	}
	p, ok := m[keyStr]
	if !ok {
		return Policy{}, false, nil
	}
	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		return Policy{}, false, nil
	}
	return p, true, nil
}

func (s *filePolicyStore) List(_ context.Context, scope string) ([]Policy, error) {
	now := time.Now()
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.byScope[scope]
	if m == nil {
		return []Policy{}, nil
	}
	out := make([]Policy, 0, len(m))
	for _, p := range m {
		if p.ExpiresAt != nil && now.After(*p.ExpiresAt) {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *filePolicyStore) Delete(_ context.Context, key PolicyKey, scope string) (bool, error) {
	keyStr := keyString(key)
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.byScope[scope]
	if m == nil {
		return false, nil
	}
	if _, ok := m[keyStr]; ok {
		delete(m, keyStr)
		if len(m) == 0 {
			delete(s.byScope, scope)
		}
		if scope == ScopeGlobal {
			if err := s.persistGlobalLocked(); err != nil {
				return true, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (s *filePolicyStore) Clear(_ context.Context, scope string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.byScope[scope]
	if m == nil {
		return 0, nil
	}
	n := len(m)
	delete(s.byScope, scope)
	if scope == ScopeGlobal {
		if err := s.persistGlobalLocked(); err != nil {
			return n, err
		}
	}
	return n, nil
}

// persistGlobalLocked writes the global scope policies to disk atomically
// with an advisory lock. Caller must hold s.mu.
func (s *filePolicyStore) persistGlobalLocked() error {
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
		return err
	}
	tmp := s.path + ".tmp"

	// Acquire advisory lock on the target file (create if not exists)
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN) // nolint:errcheck

	// Write temp, then rename over target for atomicity
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// loadFromDisk loads global scope from disk into memory (best-effort).
func (s *filePolicyStore) loadFromDisk() error {
	// Open with shared lock to read
	f, err := os.OpenFile(s.path, os.O_RDONLY, 0o644)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN) // nolint:errcheck

	var payload struct {
		Global map[string]Policy `json:"global"`
	}
	dec := json.NewDecoder(f)
	if err := dec.Decode(&payload); err != nil {
		return err
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
