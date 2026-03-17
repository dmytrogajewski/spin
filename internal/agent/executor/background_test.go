//go:build !windows

package executor_test

// Journey: specs/journeys/JOURNEY-R3.1.md.

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
)

const (
	defaultMaxOutputLines = 100
	testSleepDuration     = 100 * time.Millisecond
	startupOutputWait     = 2 * time.Second
	// testStartupTimeout is a short startup timeout for tests using sleep commands.
	testStartupTimeout = 200 * time.Millisecond
)

func TestTaskState_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state executor.TaskState
		want  string
	}{
		{executor.TaskRunning, "running"},
		{executor.TaskCompleted, "completed"},
		{executor.TaskFailed, "failed"},
		{executor.TaskKilled, "killed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestBackgroundTask_StartsAndReportsRunning(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	taskID, _, err := mgr.Start("sleep", []string{"30"}, os.Environ(), t.TempDir())
	require.NoError(t, err)
	require.Len(t, taskID, executor.TaskIDLength)

	tasks := mgr.List()
	require.Len(t, tasks, 1)
	require.Equal(t, executor.TaskRunning, tasks[0].State)
	require.Equal(t, taskID, tasks[0].ID)

	t.Cleanup(func() { mgr.Cleanup() })
}

func TestBackgroundTask_CapturesOutput(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	taskID, initialOutput, err := mgr.Start(
		"sh", []string{"-c", "echo hello && echo world"},
		os.Environ(), t.TempDir(),
	)
	require.NoError(t, err)

	// Wait for process to complete.
	waitForState(t, mgr, taskID, executor.TaskCompleted)

	// Either initial output or file output should have the content.
	output, getErr := mgr.GetOutput(taskID, defaultMaxOutputLines)
	require.NoError(t, getErr)

	combined := initialOutput + output
	require.Contains(t, combined, "hello")
	require.Contains(t, combined, "world")
}

func TestBackgroundTask_CompletedState(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	taskID, _, err := mgr.Start("true", nil, os.Environ(), t.TempDir())
	require.NoError(t, err)

	waitForState(t, mgr, taskID, executor.TaskCompleted)

	info := findTask(t, mgr, taskID)
	require.Equal(t, executor.TaskCompleted, info.State)
	require.Equal(t, 0, info.ExitCode)
}

func TestBackgroundTask_FailedState(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	taskID, _, err := mgr.Start("false", nil, os.Environ(), t.TempDir())
	require.NoError(t, err)

	waitForState(t, mgr, taskID, executor.TaskFailed)

	info := findTask(t, mgr, taskID)
	require.Equal(t, executor.TaskFailed, info.State)
	require.NotEqual(t, 0, info.ExitCode)
}

func TestBackgroundTask_GracefulKill(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	// Use a process that handles SIGTERM (sh traps it by default).
	taskID, _, err := mgr.Start("sleep", []string{"300"}, os.Environ(), t.TempDir())
	require.NoError(t, err)

	// Give process time to start.
	time.Sleep(testSleepDuration)

	killErr := mgr.Kill(taskID)
	require.NoError(t, killErr)

	info := findTask(t, mgr, taskID)
	require.Equal(t, executor.TaskKilled, info.State)
}

func TestBackgroundTask_WaitStartup(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	start := time.Now()

	taskID, initial, err := mgr.Start(
		"sh", []string{"-c", "echo started && sleep 300"},
		os.Environ(), t.TempDir(),
	)
	require.NoError(t, err)

	elapsed := time.Since(start)

	// Should have returned quickly (not waited full 20s).
	require.Less(t, elapsed, startupOutputWait)
	require.Contains(t, initial, "started")

	t.Cleanup(func() {
		_ = mgr.Kill(taskID)
	})
}

func TestBackgroundTaskManager_MaxConcurrent(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	// Start max tasks.
	for range executor.MaxConcurrentTasks {
		_, _, err := mgr.Start("sleep", []string{"300"}, os.Environ(), t.TempDir())
		require.NoError(t, err)
	}

	// Next one should fail.
	_, _, err := mgr.Start("sleep", []string{"300"}, os.Environ(), t.TempDir())
	require.ErrorIs(t, err, executor.ErrMaxConcurrentTasks)

	t.Cleanup(func() { mgr.Cleanup() })
}

func TestBackgroundTaskManager_ListTasks(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	// Start two tasks.
	idOne, _, err := mgr.Start("sleep", []string{"300"}, os.Environ(), t.TempDir())
	require.NoError(t, err)

	idTwo, _, err := mgr.Start("sleep", []string{"300"}, os.Environ(), t.TempDir())
	require.NoError(t, err)

	tasks := mgr.List()
	require.Len(t, tasks, 2)

	ids := make(map[string]bool)
	for _, task := range tasks {
		ids[task.ID] = true
	}

	require.True(t, ids[idOne])
	require.True(t, ids[idTwo])

	t.Cleanup(func() { mgr.Cleanup() })
}

func TestBackgroundTaskManager_GetOutput_InvalidID(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	_, err := mgr.GetOutput("invalid", defaultMaxOutputLines)
	require.ErrorIs(t, err, executor.ErrTaskNotFound)
}

func TestBackgroundTaskManager_Kill_InvalidID(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	err := mgr.Kill("invalid")
	require.ErrorIs(t, err, executor.ErrTaskNotFound)
}

func TestBackgroundTaskManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	const goroutineCount = 5

	var wg sync.WaitGroup

	wg.Add(goroutineCount)

	taskIDs := make([]string, goroutineCount)
	errs := make([]error, goroutineCount)

	for idx := range goroutineCount {
		go func() {
			defer wg.Done()

			taskID, _, startErr := mgr.Start("sleep", []string{"300"}, os.Environ(), t.TempDir())
			taskIDs[idx] = taskID
			errs[idx] = startErr
		}()
	}

	wg.Wait()

	for idx, startErr := range errs {
		require.NoError(t, startErr, "goroutine %d failed", idx)
	}

	tasks := mgr.List()
	require.Len(t, tasks, goroutineCount)

	t.Cleanup(func() { mgr.Cleanup() })
}

func TestBackgroundTaskManager_Cleanup(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	_, _, err := mgr.Start("sleep", []string{"300"}, os.Environ(), t.TempDir())
	require.NoError(t, err)

	_, _, err = mgr.Start("sleep", []string{"300"}, os.Environ(), t.TempDir())
	require.NoError(t, err)

	mgr.Cleanup()

	// After cleanup, all tasks should be killed.
	tasks := mgr.List()
	for _, task := range tasks {
		require.Equal(t, executor.TaskKilled, task.State)
	}
}

func TestBackgroundTaskManager_GetOutput_MaxLines(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	// Generate 10 lines of output.
	taskID, _, err := mgr.Start(
		"sh", []string{"-c", "for i in $(seq 1 10); do echo line$i; done"},
		os.Environ(), t.TempDir(),
	)
	require.NoError(t, err)

	waitForState(t, mgr, taskID, executor.TaskCompleted)

	// Request only last 3 lines.
	const maxThreeLines = 3

	output, getErr := mgr.GetOutput(taskID, maxThreeLines)
	require.NoError(t, getErr)

	lines := strings.Split(output, "\n")
	require.LessOrEqual(t, len(lines), maxThreeLines)
}

func TestBackgroundTask_KillNonRunning(t *testing.T) {
	t.Parallel()

	mgr := newTestManager(t)

	taskID, _, err := mgr.Start("true", nil, os.Environ(), t.TempDir())
	require.NoError(t, err)

	waitForState(t, mgr, taskID, executor.TaskCompleted)

	killErr := mgr.Kill(taskID)
	require.ErrorIs(t, killErr, executor.ErrTaskNotRunning)
}

// newTestManager creates a BackgroundTaskManager with a short startup timeout.
func newTestManager(t *testing.T) *executor.BackgroundTaskManager {
	t.Helper()

	outputDir := filepath.Join(t.TempDir(), "tasks")

	mgr, err := executor.NewBackgroundTaskManager(outputDir)
	require.NoError(t, err)

	mgr.SetStartupTimeout(testStartupTimeout)

	return mgr
}

// waitForState polls until a task reaches the expected state or times out.
func waitForState(t *testing.T, mgr *executor.BackgroundTaskManager, taskID string, want executor.TaskState) {
	t.Helper()

	const (
		pollInterval = 50 * time.Millisecond
		maxWait      = 10 * time.Second
	)

	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		info := findTask(t, mgr, taskID)
		if info.State == want {
			return
		}

		time.Sleep(pollInterval)
	}

	t.Fatalf("task %s did not reach state %s within timeout", taskID, want)
}

// findTask locates a task by ID in the manager's list.
func findTask(t *testing.T, mgr *executor.BackgroundTaskManager, taskID string) executor.TaskInfo {
	t.Helper()

	for _, task := range mgr.List() {
		if task.ID == taskID {
			return task
		}
	}

	t.Fatalf("task %s not found", taskID)

	return executor.TaskInfo{}
}
