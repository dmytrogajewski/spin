//go:build windows

package executor

import (
	"context"
	"errors"
	"time"
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
)

// Background task manager errors.
var (
	// ErrMaxConcurrentTasks is returned when the task limit is reached.
	ErrMaxConcurrentTasks = errors.New("maximum concurrent tasks reached")
	// ErrTaskNotFound is returned when a task ID does not exist.
	ErrTaskNotFound = errors.New("task not found")
	// ErrTaskNotRunning is returned when trying to kill a non-running task.
	ErrTaskNotRunning = errors.New("task is not running")
	// ErrUnsupportedPlatform is returned on unsupported platforms.
	ErrUnsupportedPlatform = errors.New("background tasks are not supported on Windows")
)

// BackgroundTaskManager manages background processes with output capture.
// On Windows, background task management is not supported.
type BackgroundTaskManager struct{}

// NewBackgroundTaskManager returns an error on Windows.
func NewBackgroundTaskManager(_ string) (*BackgroundTaskManager, error) {
	return nil, ErrUnsupportedPlatform
}

// Start is not supported on Windows.
func (m *BackgroundTaskManager) Start(_ context.Context, _ string, _ []string, _ []string, _ string) (string, string, error) {
	return "", "", ErrUnsupportedPlatform
}

// List is not supported on Windows.
func (m *BackgroundTaskManager) List() []TaskInfo {
	return nil
}

// GetOutput is not supported on Windows.
func (m *BackgroundTaskManager) GetOutput(_ string, _ int) (string, error) {
	return "", ErrUnsupportedPlatform
}

// Kill is not supported on Windows.
func (m *BackgroundTaskManager) Kill(_ string) error {
	return ErrUnsupportedPlatform
}

// Cleanup is a no-op on Windows.
func (m *BackgroundTaskManager) Cleanup() {}
