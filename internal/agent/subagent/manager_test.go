package subagent

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Journey: specs/journeys/JOURNEY-1.3.md.

const (
	testQuery   = "analyze the project structure"
	testSummary = "found 10 Go files in internal/"
)

// echoExecutor is a test executor that returns the query as summary.
func echoExecutor(_ context.Context, _ *Spec, query string) (string, error) {
	return "summary: " + query, nil
}

// TestNewManager_RegistersBuiltins tests that NewManager pre-registers builtin specs.
// Kills mutant: skipping builtin registration would make this test fail.
func TestNewManager_RegistersBuiltins(t *testing.T) {
	t.Parallel()

	mgr := NewManager(echoExecutor, DefaultMaxConcurrent)

	assert.NotNil(t, mgr.Spec(NameExplorer), "explorer should be registered")
	assert.NotNil(t, mgr.Spec(NamePlanner), "planner should be registered")
	assert.NotNil(t, mgr.Spec(NameReviewer), "reviewer should be registered")
	assert.NotNil(t, mgr.Spec(NameAskUser), "ask_user should be registered")
}

// TestNewManager_DefaultConcurrency tests that zero/negative concurrency uses default.
// Kills mutant: not applying default would make this test fail.
func TestNewManager_DefaultConcurrency(t *testing.T) {
	t.Parallel()

	mgr := NewManager(echoExecutor, 0)
	assert.Equal(t, DefaultMaxConcurrent, mgr.maxConcurrent)

	mgr2 := NewManager(echoExecutor, -5)
	assert.Equal(t, DefaultMaxConcurrent, mgr2.maxConcurrent)
}

// TestSpawn_Explorer tests that Spawn executes the explorer and returns a summary.
// Kills mutant: not calling the executor would make this test fail.
func TestSpawn_Explorer(t *testing.T) {
	t.Parallel()

	mgr := NewManager(echoExecutor, DefaultMaxConcurrent)

	summary, err := mgr.Spawn(context.Background(), NameExplorer, testQuery)
	require.NoError(t, err)
	assert.Contains(t, summary, testQuery)
}

// TestSpawn_UnknownSpec tests that Spawn returns error for unknown spec names.
// Kills mutant: accepting unknown specs would make this test fail.
func TestSpawn_UnknownSpec(t *testing.T) {
	t.Parallel()

	mgr := NewManager(echoExecutor, DefaultMaxConcurrent)

	_, err := mgr.Spawn(context.Background(), "nonexistent", testQuery)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSpecNotFound)
}

// TestSpawn_PanicRecovery tests that executor panics are recovered as errors.
// Kills mutant: removing panic recovery would make this test fail.
func TestSpawn_PanicRecovery(t *testing.T) {
	t.Parallel()

	panicExecutor := func(_ context.Context, _ *Spec, _ string) (string, error) {
		panic("executor crashed")
	}

	mgr := NewManager(panicExecutor, DefaultMaxConcurrent)

	summary, err := mgr.Spawn(context.Background(), NameExplorer, testQuery)
	require.ErrorIs(t, err, ErrPanicked)
	assert.Contains(t, err.Error(), "executor crashed")
	assert.Empty(t, summary)
}

// TestSpawn_ConcurrencyCap tests that maxConcurrent limits parallel execution.
// Kills mutant: removing semaphore would make this test fail.
func TestSpawn_ConcurrencyCap(t *testing.T) {
	t.Parallel()

	const maxConcurrent = 3

	const totalSpawns = 6

	var (
		running    atomic.Int32
		maxRunning atomic.Int32
	)

	slowExecutor := func(_ context.Context, _ *Spec, _ string) (string, error) {
		cur := running.Add(1)

		// Track the peak concurrency.
		for {
			old := maxRunning.Load()
			if cur <= old || maxRunning.CompareAndSwap(old, cur) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)
		running.Add(-1)

		return testSummary, nil
	}

	mgr := NewManager(slowExecutor, maxConcurrent)

	var wg sync.WaitGroup

	for range totalSpawns {
		wg.Go(func() {
			_, err := mgr.Spawn(context.Background(), NameExplorer, testQuery)
			assert.NoError(t, err)
		})
	}

	wg.Wait()

	assert.LessOrEqual(t, maxRunning.Load(), int32(maxConcurrent),
		"peak concurrency should not exceed maxConcurrent")
}

// TestSpawn_ContextCancellation tests that Spawn respects context cancellation.
// Kills mutant: ignoring context would make this test fail.
func TestSpawn_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Fill the semaphore so the next Spawn blocks.
	blockingExecutor := func(ctx context.Context, _ *Spec, _ string) (string, error) {
		<-ctx.Done()

		return "", ctx.Err()
	}

	mgr := NewManager(blockingExecutor, 1)

	// Spawn one that blocks.
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_, _ = mgr.Spawn(ctx, NameExplorer, testQuery)
	}()

	// Give the goroutine time to acquire the semaphore.
	time.Sleep(10 * time.Millisecond)

	// Second spawn should block on semaphore, then fail on context cancel.
	cancelCtx, cancelFn := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelFn()

	_, err := mgr.Spawn(cancelCtx, NameExplorer, testQuery)
	require.Error(t, err)

	cancel() // Clean up blocking goroutine.
}

// TestRegister_CustomSpec tests that Register adds a custom spec.
// Kills mutant: not storing the spec would make this test fail.
func TestRegister_CustomSpec(t *testing.T) {
	t.Parallel()

	mgr := NewManager(echoExecutor, DefaultMaxConcurrent)

	customSpec := &Spec{
		Name:          "custom",
		Description:   "A custom subagent.",
		SystemPrompt:  "You are custom.",
		AllowedTools:  []string{"read_file"},
		MaxIterations: 10,
	}

	err := mgr.Register(customSpec)
	require.NoError(t, err)

	// Spawn the custom spec.
	summary, err := mgr.Spawn(context.Background(), "custom", testQuery)
	require.NoError(t, err)
	assert.Contains(t, summary, testQuery)
}

// TestRegister_NilSpec tests that Register rejects nil specs.
// Kills mutant: accepting nil would make this test fail.
func TestRegister_NilSpec(t *testing.T) {
	t.Parallel()

	mgr := NewManager(echoExecutor, DefaultMaxConcurrent)

	err := mgr.Register(nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilSpec)
}

// TestRegister_EmptyName tests that Register rejects specs with empty names.
// Kills mutant: accepting empty names would make this test fail.
func TestRegister_EmptyName(t *testing.T) {
	t.Parallel()

	mgr := NewManager(echoExecutor, DefaultMaxConcurrent)

	err := mgr.Register(&Spec{Name: ""})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptySpecName)
}
