//go:build !windows

package executor

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dmytrogajewski/spin/internal/process"
)

// Background task manager constants.
const (
	// MaxConcurrentTasks is the maximum number of background tasks allowed.
	MaxConcurrentTasks = 10
	// TaskIDLength is the number of hex characters in a task ID.
	TaskIDLength = 7
	// StartupWaitTimeout is the maximum time to wait for initial output.
	StartupWaitTimeout = 20 * time.Second
	// GracefulKillTimeout is the time to wait after SIGTERM before SIGKILL.
	GracefulKillTimeout = 5 * time.Second
	// outputDirPerm is the permission for the output directory.
	outputDirPerm = 0o750
	// taskIDRandomBytes is the number of random bytes for task ID generation.
	taskIDRandomBytes = 4
)

// Background task manager errors.
var (
	// ErrMaxConcurrentTasks is returned when the task limit is reached.
	ErrMaxConcurrentTasks = errors.New("maximum concurrent tasks reached")
	// ErrTaskNotFound is returned when a task ID does not exist.
	ErrTaskNotFound = errors.New("task not found")
	// ErrTaskNotRunning is returned when trying to kill a non-running task.
	ErrTaskNotRunning = errors.New("task is not running")
)

// backgroundTask is the internal representation of a running background task.
type backgroundTask struct {
	mu         sync.Mutex
	id         string
	command    string
	state      TaskState
	startedAt  time.Time
	exitCode   int
	cmd        *exec.Cmd
	outputFile *os.File
	outputPath string
	done       chan struct{}
}

// info returns a read-only snapshot of the task.
func (t *backgroundTask) info() TaskInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	return TaskInfo{
		ID:        t.id,
		Command:   t.command,
		State:     t.state,
		StartedAt: t.startedAt,
		ExitCode:  t.exitCode,
	}
}

// BackgroundTaskManager manages background processes with output capture.
type BackgroundTaskManager struct {
	mu             sync.Mutex
	tasks          map[string]*backgroundTask
	outputDir      string
	startupTimeout time.Duration
}

// NewBackgroundTaskManager creates a manager that stores output in the given directory.
func NewBackgroundTaskManager(outputDir string) (*BackgroundTaskManager, error) {
	if err := os.MkdirAll(outputDir, outputDirPerm); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	return &BackgroundTaskManager{
		tasks:          make(map[string]*backgroundTask),
		outputDir:      outputDir,
		startupTimeout: StartupWaitTimeout,
	}, nil
}

// SetStartupTimeout overrides the default startup wait timeout.
func (m *BackgroundTaskManager) SetStartupTimeout(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startupTimeout = d
}

// Start launches a command in the background and returns the task ID and initial output.
// It waits up to [StartupWaitTimeout] for the first output before returning.
func (m *BackgroundTaskManager) Start(
	program string, args []string, env []string, workDir string,
) (string, string, error) {
	m.mu.Lock()

	if m.runningCount() >= MaxConcurrentTasks {
		m.mu.Unlock()

		return "", "", ErrMaxConcurrentTasks
	}

	taskID := generateTaskID()
	outputPath := filepath.Join(m.outputDir, taskID+".output")

	outFile, err := os.Create(outputPath)
	if err != nil {
		m.mu.Unlock()

		return "", "", fmt.Errorf("create output file: %w", err)
	}

	task := &backgroundTask{
		id:         taskID,
		command:    buildFullCommand(program, args),
		state:      TaskRunning,
		startedAt:  time.Now(),
		exitCode:   -1,
		outputFile: outFile,
		outputPath: outputPath,
		done:       make(chan struct{}),
	}

	cmd := exec.CommandContext(context.Background(), program, args...)
	cmd.Dir = workDir
	cmd.Env = env

	process.SetGroup(cmd)

	// Pipe stdout+stderr to both the output file and a startup reader.
	startupReader, startupWriter := io.Pipe()
	multiWriter := io.MultiWriter(outFile, startupWriter)

	cmd.Stdout = multiWriter
	cmd.Stderr = multiWriter

	task.cmd = cmd

	if startErr := cmd.Start(); startErr != nil {
		outFile.Close()
		os.Remove(outputPath)
		m.mu.Unlock()

		return "", "", fmt.Errorf("start command: %w", startErr)
	}

	m.tasks[taskID] = task
	m.mu.Unlock()

	// Monitor the process in background.
	go m.monitor(task, startupWriter)

	// Wait for initial output.
	m.mu.Lock()
	timeout := m.startupTimeout
	m.mu.Unlock()

	initialOutput := m.waitStartup(startupReader, timeout)

	return taskID, initialOutput, nil
}

// List returns a snapshot of all tasks.
func (m *BackgroundTaskManager) List() []TaskInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]TaskInfo, 0, len(m.tasks))
	for _, task := range m.tasks {
		result = append(result, task.info())
	}

	return result
}

// GetOutput reads the last maxLines lines from a task's output file.
func (m *BackgroundTaskManager) GetOutput(taskID string, maxLines int) (string, error) {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	m.mu.Unlock()

	if !ok {
		return "", fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}

	return readLastLines(task.outputPath, maxLines)
}

// Kill sends SIGTERM to the task, then SIGKILL after [GracefulKillTimeout].
func (m *BackgroundTaskManager) Kill(taskID string) error {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}

	task.mu.Lock()
	if task.state != TaskRunning {
		task.mu.Unlock()

		return fmt.Errorf("%w: %s (state: %s)", ErrTaskNotRunning, taskID, task.state)
	}
	task.mu.Unlock()

	return m.killProcess(task)
}

// Cleanup kills all running tasks.
func (m *BackgroundTaskManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range m.tasks {
		task.mu.Lock()
		if task.state == TaskRunning {
			task.mu.Unlock()
			// Best-effort kill; ignore errors during cleanup.
			_ = m.killProcess(task)
		} else {
			task.mu.Unlock()
		}
	}
}

// runningCount returns the number of currently running tasks.
// Caller must hold m.mu.
func (m *BackgroundTaskManager) runningCount() int {
	count := 0

	for _, task := range m.tasks {
		task.mu.Lock()
		if task.state == TaskRunning {
			count++
		}
		task.mu.Unlock()
	}

	return count
}

// monitor waits for the process to exit and updates task state.
func (m *BackgroundTaskManager) monitor(task *backgroundTask, startupWriter *io.PipeWriter) {
	defer close(task.done)

	err := task.cmd.Wait()

	// Close the startup writer so waitStartup unblocks.
	startupWriter.Close()

	task.mu.Lock()
	defer task.mu.Unlock()

	// Close the output file.
	task.outputFile.Close()

	// Only update state if not already killed.
	if task.state == TaskKilled {
		return
	}

	if err != nil {
		task.state = TaskFailed

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			task.exitCode = exitErr.ExitCode()
		}

		return
	}

	task.state = TaskCompleted
	task.exitCode = 0
}

// waitStartup reads initial output up to the given timeout.
// It returns as soon as the first line is received or the timeout expires.
func (m *BackgroundTaskManager) waitStartup(reader *io.PipeReader, timeout time.Duration) string {
	firstLine := make(chan string, 1)

	go func() {
		scanner := bufio.NewScanner(reader)
		if scanner.Scan() {
			firstLine <- scanner.Text()
		}

		close(firstLine)

		// Drain remaining data so the pipe writer doesn't block.
		// The monitor goroutine will close the writer, which will end this loop.
		for scanner.Scan() {
			// Discard — output is captured via the multiwriter to file.
		}

		reader.Close()
	}()

	select {
	case line, ok := <-firstLine:
		if ok {
			return line + "\n"
		}

		return ""
	case <-time.After(timeout):
		return ""
	}
}

// killProcess sends SIGTERM then escalates to SIGKILL.
func (m *BackgroundTaskManager) killProcess(task *backgroundTask) error {
	if task.cmd.Process == nil {
		return process.ErrProcessNotStarted
	}

	pid := task.cmd.Process.Pid

	// Send SIGTERM to process group (best-effort, process may have already exited).
	_ = syscall.Kill(-pid, syscall.SIGTERM)

	// Wait for graceful shutdown or escalate to SIGKILL.
	select {
	case <-task.done:
	case <-time.After(GracefulKillTimeout):
		// Escalate to SIGKILL.
		_ = syscall.Kill(-pid, syscall.SIGKILL)

		<-task.done
	}

	task.mu.Lock()
	task.state = TaskKilled
	task.mu.Unlock()

	return nil
}

// generateTaskID creates a 7-char hex task identifier.
func generateTaskID() string {
	buf := make([]byte, taskIDRandomBytes)

	_, _ = rand.Read(buf)

	return hex.EncodeToString(buf)[:TaskIDLength]
}

// readLastLines reads the last n lines from a file.
func readLastLines(path string, maxLines int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open output file: %w", err)
	}
	defer file.Close()

	var lines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return "", fmt.Errorf("read output file: %w", scanErr)
	}

	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	return strings.Join(lines, "\n"), nil
}
