package snapshot_test

// Journey: specs/journeys/JOURNEY-R5.2.md.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/agent/middleware/snapshot"
)

// errSnapshotFailed is a test sentinel error.
var errSnapshotFailed = errors.New("snapshot failed")

// mockSnapshotter implements [snapshot.Snapshotter] for testing.
type mockSnapshotter struct {
	hash  string
	err   error
	calls int
}

func (m *mockSnapshotter) TakeSnapshot() (string, error) {
	m.calls++

	return m.hash, m.err
}

const testHash = "abc123def456"

func TestMiddleware_AfterExecution_TakesSnapshot(t *testing.T) {
	t.Parallel()

	mock := &mockSnapshotter{hash: testHash}

	mw := snapshot.NewMiddleware(mock, nil)

	mw.AfterExecution(context.Background(), &harness.IterationContext{}, &harness.Response{})

	require.Equal(t, 1, mock.calls)
}

func TestMiddleware_AfterExecution_ErrorDoesNotPanic(t *testing.T) {
	t.Parallel()

	mock := &mockSnapshotter{err: errSnapshotFailed}

	mw := snapshot.NewMiddleware(mock, nil)

	// Should not panic.
	mw.AfterExecution(context.Background(), &harness.IterationContext{}, &harness.Response{})

	require.Equal(t, 1, mock.calls)
}

func TestMiddleware_BeforeTurn_IsNoOp(t *testing.T) {
	t.Parallel()

	mock := &mockSnapshotter{}
	mw := snapshot.NewMiddleware(mock, nil)

	// Should not panic or call snapshot.
	mw.BeforeTurn(context.Background(), &harness.IterationContext{})

	require.Equal(t, 0, mock.calls)
}
