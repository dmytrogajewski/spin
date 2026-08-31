// Package tasks is the parent-side A2A task registry plus a unified view over shell processes.
package tasks

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/session"
)

const pollInterval = 5 * time.Millisecond

const (
	// StateWorking is an in-flight A2A task.
	StateWorking = "working"
	// StateCompleted is a successful terminal state.
	StateCompleted = "completed"
	// StateFailed is a failed terminal state.
	StateFailed = "failed"
	// StateCanceled is a canceled terminal state.
	StateCanceled = "canceled"
)

// ErrNotFound indicates the task id is not in the registry.
var ErrNotFound = errors.New("tasks: not found")

// ErrAmbiguous indicates an untyped id exists in both agent and shell stores.
var ErrAmbiguous = errors.New("tasks: ambiguous id; use agent:<id> or shell:<id>")

// Record is one A2A task row: id, spec, and lifecycle state.
type Record struct {
	ID    string
	Spec  string
	State string
}

// Handle is the live A2A peer for Get / Cancel / SIGTERM.
type Handle interface {
	Get(ctx context.Context) (state string, err error)
	Cancel(ctx context.Context) error
	SignalTERM() error
}

type entry struct {
	rec    Record
	handle Handle
}

// Registry holds in-memory A2A task records for the parent session.
type Registry struct {
	mu    sync.RWMutex
	items map[string]*entry
	sess  *session.Session
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{items: make(map[string]*entry)}
}

// Restore builds a registry from session metadata (handles are not restored).
func Restore(sess *session.Session) *Registry {
	reg := New()
	if sess == nil {
		return reg
	}

	reg.sess = sess
	for _, row := range sess.Metadata.AgentTasks {
		reg.items[row.ID] = &entry{rec: Record{ID: row.ID, Spec: row.Spec, State: row.State}}
	}

	return reg
}

// Bind attaches the session so mutations persist AgentTasks metadata.
func (r *Registry) Bind(sess *session.Session) {
	r.mu.Lock()
	r.sess = sess
	r.persistLocked()
	r.mu.Unlock()
}

// List returns registered rows. Order is unspecified.
func (r *Registry) List() []Record {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Record, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item.rec)
	}

	slices.SortFunc(out, func(left, right Record) int {
		return cmp.Compare(left.ID, right.ID)
	})

	return out
}

// Register stores a row. handle may be nil for already-terminal snapshots.
func (r *Registry) Register(id, spec, state string, handle Handle) Record {
	rec := Record{ID: id, Spec: spec, State: state}

	r.mu.Lock()
	r.items[id] = &entry{rec: rec, handle: handle}
	r.persistLocked()
	r.mu.Unlock()

	return rec
}

// Cancel calls tasks/cancel then SIGTERM and marks the row canceled.
func (r *Registry) Cancel(ctx context.Context, id string) error {
	item, err := r.lookup(id)
	if err != nil {
		return err
	}

	if Terminal(item.rec.State) {
		return nil
	}

	if item.handle != nil {
		if cancelErr := item.handle.Cancel(ctx); cancelErr != nil {
			return fmt.Errorf("tasks cancel: %w", cancelErr)
		}

		if termErr := item.handle.SignalTERM(); termErr != nil {
			return fmt.Errorf("tasks sigterm: %w", termErr)
		}
	}

	r.setState(id, StateCanceled)

	return nil
}

// CancelAll sends tasks/cancel then SIGTERM to every non-terminal row.
func (r *Registry) CancelAll(ctx context.Context) error {
	if r == nil {
		return nil
	}

	for _, rec := range r.List() {
		_ = r.Cancel(ctx, rec.ID)
	}

	return nil
}

// Wait returns when the task is terminal or ctx is canceled.
// It must not acquire the spawn semaphore — see TestWait_DoesNotAcquireSemaphore.
func (r *Registry) Wait(ctx context.Context, id string) (Record, error) {
	item, err := r.lookup(id)
	if err != nil {
		return Record{}, err
	}

	if Terminal(item.rec.State) {
		return item.rec, nil
	}

	if item.handle == nil {
		<-ctx.Done()

		return item.rec, fmt.Errorf("tasks wait: %w", ctx.Err())
	}

	return r.poll(ctx, id, item.handle)
}

func (r *Registry) poll(ctx context.Context, id string, handle Handle) (Record, error) {
	for {
		if err := ctx.Err(); err != nil {
			rec, _ := r.mustLookup(id)

			return rec, fmt.Errorf("tasks wait: %w", err)
		}

		state, getErr := handle.Get(ctx)
		if getErr != nil {
			rec, _ := r.mustLookup(id)

			return rec, fmt.Errorf("tasks get: %w", getErr)
		}

		r.setState(id, state)

		if Terminal(state) {
			return r.mustLookup(id)
		}

		select {
		case <-ctx.Done():
			rec, _ := r.mustLookup(id)

			return rec, fmt.Errorf("tasks wait: %w", ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

func (r *Registry) lookup(id string) (*entry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}

	return item, nil
}

func (r *Registry) mustLookup(id string) (Record, error) {
	item, err := r.lookup(id)
	if err != nil {
		return Record{}, err
	}

	return item.rec, nil
}

func (r *Registry) setState(id, state string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if item, ok := r.items[id]; ok {
		item.rec.State = state
	}

	r.persistLocked()
}

func (r *Registry) persistLocked() {
	if r.sess == nil {
		return
	}

	rows := make([]session.AgentTask, 0, len(r.items))
	for _, item := range r.items {
		rows = append(rows, session.AgentTask{
			ID:    item.rec.ID,
			Spec:  item.rec.Spec,
			State: item.rec.State,
		})
	}

	_ = r.sess.UpdateMetadata(func(m *session.Metadata) {
		m.AgentTasks = rows
	})
}

// Terminal reports whether state cannot accept further work.
func Terminal(state string) bool {
	switch state {
	case StateCompleted, StateFailed, StateCanceled:
		return true
	default:
		return false
	}
}
