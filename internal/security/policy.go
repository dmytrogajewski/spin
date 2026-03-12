package security

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
)

const policyEvictionInterval = 30 * time.Second

// PolicyKey identifies a decision using normalized command and context.
type PolicyKey struct {
	Program string
	Args    []string
	WorkDir string
}

// Policy is the persisted approval decision.
type Policy struct {
	Version    string
	Scope      string // "session" | "global".
	Key        PolicyKey
	Decision   string // "allow" (future-proof for "deny").
	PolicyNote string
	CreatedAt  time.Time
	ExpiresAt  *time.Time
	Meta       map[string]string
}

// Approval scope constants (kept in policy module as they define policy semantics).
const (
	ScopeOnce    = "once"
	ScopeSession = "session"
	ScopeGlobal  = "global"
)

// Policy decision constants.
const (
	DecisionAllow = "allow"
	DecisionDeny  = "deny"
)

// PolicyStore stores and retrieves policies with TTL and concurrency safety.
type PolicyStore interface {
	Save(ctx context.Context, p Policy) error
	Get(ctx context.Context, key PolicyKey, scope string) (Policy, bool, error)
	List(ctx context.Context, scope string) ([]Policy, error)
	Delete(ctx context.Context, key PolicyKey, scope string) (bool, error)
	Clear(ctx context.Context, scope string) (int, error)
	Close() error
}

// NewPolicyKey normalizes inputs into a PolicyKey (exact-match semantics).
// For shell commands executed via /bin/sh -c "command", uses the actual command
// string instead of the shell invocation, ensuring consistent policy keys.
func NewPolicyKey(program string, args []string, workDir string) PolicyKey {
	normalizedProgram := strings.TrimSpace(program)
	normalizedArgs := normalizeArgs(args)

	// For shell commands executed via /bin/sh -c "command" or /bin/bash -c "command",
	// use the actual command (second arg) instead of the shell invocation.
	// This ensures policy keys match for the same command regardless of shell wrapper.
	if (program == "/bin/sh" || program == "/bin/bash" || program == "sh" || program == "bash") &&
		len(args) == 2 && args[0] == "-c" {
		// Extract the actual command from the -c argument.
		actualCommand := strings.TrimSpace(args[1])
		// Use "shell" as program and the actual command as a single arg.
		normalizedProgram = "shell"
		normalizedArgs = []string{normalizeCommand(actualCommand)}
	}

	return PolicyKey{
		Program: normalizedProgram,
		Args:    normalizedArgs,
		WorkDir: strings.TrimSpace(workDir),
	}
}

// normalizeCommand normalizes a command string for consistent policy keys.
// Collapses whitespace and removes leading/trailing spaces.
func normalizeCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	// Collapse multiple spaces into one.
	cmd = wsCollapse.ReplaceAllString(cmd, " ")

	return cmd
}

var wsCollapse = regexp.MustCompile(`\s+`)

func normalizeArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	out := make([]string, 0, len(args))
	for _, a := range args {
		s := strings.TrimSpace(a)
		if s == "" {
			out = append(out, s)

			continue
		}

		s = wsCollapse.ReplaceAllString(s, " ")
		out = append(out, s)
	}

	return out
}

// MemoryPolicyStore is an in-memory PolicyStore with TTL eviction.
type MemoryPolicyStore struct {
	mu       sync.RWMutex
	byScope  map[string]map[string]Policy // scope -> keyStr -> policy.
	stopCh   chan struct{}
	interval time.Duration
}

// NewMemoryPolicyStore creates an in-memory store with a janitor running at interval.
func NewMemoryPolicyStore(evictionInterval time.Duration) *MemoryPolicyStore {
	if evictionInterval <= 0 {
		evictionInterval = policyEvictionInterval
	}

	s := &MemoryPolicyStore{
		byScope:  make(map[string]map[string]Policy),
		stopCh:   make(chan struct{}),
		interval: evictionInterval,
	}
	go s.janitor()

	return s
}

func (s *MemoryPolicyStore) janitor() {
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

func (s *MemoryPolicyStore) removeExpired() {
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
}

// Save implements the Save operation.
func (s *MemoryPolicyStore) Save(_ context.Context, p Policy) error {
	keyStr := keyString(p.Key)

	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.byScope[p.Scope]
	if m == nil {
		m = make(map[string]Policy)
		s.byScope[p.Scope] = m
	}

	m[keyStr] = p

	return nil
}

// getFromScopeMap retrieves a policy from a scope-keyed map under a read lock.
func getFromScopeMap(byScope map[string]map[string]Policy, key PolicyKey, scope string) (Policy, bool) {
	keyStr := keyString(key)

	m := byScope[scope]
	if m == nil {
		return Policy{}, false
	}

	p, ok := m[keyStr]
	if !ok {
		return Policy{}, false
	}

	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		return Policy{}, false
	}

	return p, true
}

// listFromScopeMap lists non-expired policies from a scope-keyed map.
func listFromScopeMap(byScope map[string]map[string]Policy, scope string) []Policy {
	now := time.Now()

	m := byScope[scope]
	if m == nil {
		return []Policy{}
	}

	out := make([]Policy, 0, len(m))
	for _, p := range m {
		if p.ExpiresAt != nil && now.After(*p.ExpiresAt) {
			continue
		}

		out = append(out, p)
	}

	return out
}

// Get implements the Get operation.
func (s *MemoryPolicyStore) Get(_ context.Context, key PolicyKey, scope string) (Policy, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := getFromScopeMap(s.byScope, key, scope)

	return p, ok, nil
}

// List implements the List operation.
func (s *MemoryPolicyStore) List(_ context.Context, scope string) ([]Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return listFromScopeMap(s.byScope, scope), nil
}

// Delete implements the Delete operation.
func (s *MemoryPolicyStore) Delete(_ context.Context, key PolicyKey, scope string) (bool, error) {
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

		return true, nil
	}

	return false, nil
}

// Clear implements the Clear operation.
func (s *MemoryPolicyStore) Clear(_ context.Context, scope string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.byScope[scope]
	if m == nil {
		return 0, nil
	}

	n := len(m)

	delete(s.byScope, scope)

	return n, nil
}

// Close stops the janitor goroutine and releases resources.
func (s *MemoryPolicyStore) Close() error {
	close(s.stopCh)

	return nil
}

func keyString(k PolicyKey) string {
	// Deterministic encoding with non-printable delimiters to avoid collisions.
	sep := "\x1F" // unit separator.

	return strings.Join([]string{
		k.Program,
		strings.Join(k.Args, sep),
		k.WorkDir,
	}, sep)
}
