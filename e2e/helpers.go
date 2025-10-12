// Package e2e provides end-to-end testing utilities for the Spin AI coding agent.
//
// This package contains test helpers, mocks, and fixtures for comprehensive
// integration testing of full user workflows including:
//   - Full conversation flows (user → LLM → tools → response)
//   - Tool chain integration (multiple tools working together)
//   - Multi-turn conversations with context preservation
//   - Error recovery and graceful failure handling
//   - Security boundary testing (path traversal, injection attacks)
//   - Performance validation under load
//   - Chaos testing (concurrent operations, resource exhaustion)
package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/stretchr/testify/require"
)

// TestAgent is a fully configured agent for E2E testing.
type TestAgent struct {
	Agent     *core.Agent
	LLM       *MockLLM
	Executor  *core.Executor
	Validator *core.Validator
	Context   *core.Environment
	Emitter   *core.EventEmitter
	Workspace string
	cleanup   func()
}

// TestAgentOptions configures a TestAgent.
type TestAgentOptions struct {
	// LLMResponses are the mock LLM responses to return
	LLMResponses []MockResponse

	// WorkspaceFiles are files to create in the workspace
	WorkspaceFiles map[string]string

	// GitRepo if true, initializes a git repository in workspace
	GitRepo bool

	// Timeout for agent operations (default: 30s)
	Timeout time.Duration
}

// NewTestAgent creates a fully configured agent for E2E testing.
func NewTestAgent(t *testing.T, opts TestAgentOptions) *TestAgent {
	t.Helper()

	// Create temporary workspace
	workspace, err := os.MkdirTemp("", "spin-e2e-*")
	require.NoError(t, err, "failed to create temp workspace")

	// Create workspace files
	for path, content := range opts.WorkspaceFiles {
		fullPath := filepath.Join(workspace, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	// Initialize git repo if requested
	if opts.GitRepo {
		InitGitRepo(t, workspace)
	}

	// Create mock LLM
	mockLLM := NewMockLLM(opts.LLMResponses)

	// Create command validator
	validator := core.NewValidator()

	// Create command executor
	executor, err := core.NewExecutor(workspace)
	require.NoError(t, err, "failed to create executor")

	// Create environment context
	ctx := &core.Environment{
		WorkDir: workspace,
	}

	// Create event emitter
	emitter := core.NewEventEmitter(100)

	// Create agent
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	agent, err := core.NewAgent(mockLLM, executor, validator, ctx, emitter,
		core.WithAgentTimeout(timeout),
	)
	require.NoError(t, err, "failed to create agent")

	cleanup := func() {
		os.RemoveAll(workspace)
	}

	return &TestAgent{
		Agent:     agent,
		LLM:       mockLLM,
		Executor:  executor,
		Validator: validator,
		Context:   ctx,
		Emitter:   emitter,
		Workspace: workspace,
		cleanup:   cleanup,
	}
}

// Cleanup cleans up the test agent resources.
func (ta *TestAgent) Cleanup() {
	if ta.cleanup != nil {
		ta.cleanup()
	}
}

// NewTestWorkspace creates a temporary workspace with test files.
func NewTestWorkspace(t *testing.T, files map[string]string) (workspace string, cleanup func()) {
	t.Helper()

	workspace, err := os.MkdirTemp("", "spin-e2e-workspace-*")
	require.NoError(t, err)

	for path, content := range files {
		fullPath := filepath.Join(workspace, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	cleanup = func() {
		os.RemoveAll(workspace)
	}

	return workspace, cleanup
}

// InitGitRepo initializes a git repository in the given directory.
func InitGitRepo(t *testing.T, dir string) {
	t.Helper()

	// Create .git directory (minimal setup for testing)
	gitDir := filepath.Join(dir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))

	// Create minimal git config
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[user]
	name = Test User
	email = test@example.com
`
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(configContent), 0644))

	// Create HEAD
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644))

	// Create refs directory
	require.NoError(t, os.MkdirAll(filepath.Join(gitDir, "refs", "heads"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(gitDir, "objects"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(gitDir, "refs", "tags"), 0755))
}

// AssertEventSequence verifies that events were emitted in the expected order.
func AssertEventSequence(t *testing.T, events <-chan core.Event, expectedTypes []core.EventType) {
	t.Helper()

	var actualTypes []core.EventType
	timeout := time.After(5 * time.Second)

	for i := 0; i < len(expectedTypes); i++ {
		select {
		case event := <-events:
			actualTypes = append(actualTypes, event.Type)
		case <-timeout:
			require.FailNow(t, "timeout waiting for events",
				"expected %d events, got %d: %v", len(expectedTypes), len(actualTypes), actualTypes)
		}
	}

	require.Equal(t, expectedTypes, actualTypes, "event sequence mismatch")
}

// AssertNoErrors verifies that no error events were emitted.
func AssertNoErrors(t *testing.T, events []core.Event) {
	t.Helper()

	for _, event := range events {
		if event.Type == core.EventError {
			require.FailNow(t, "unexpected error event",
				"error: %v", event.Data)
		}
	}
}

// CollectEvents collects all events from a channel until it closes or timeout.
func CollectEvents(events <-chan core.Event, timeout time.Duration) []core.Event {
	var collected []core.Event
	timeoutChan := time.After(timeout)

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return collected
			}
			collected = append(collected, event)
		case <-timeoutChan:
			return collected
		}
	}
}

// FindEvent finds the first event of the given type.
func FindEvent(events []core.Event, eventType core.EventType) (core.Event, bool) {
	for _, event := range events {
		if event.Type == eventType {
			return event, true
		}
	}
	return core.Event{}, false
}

// FindEvents finds all events of the given type.
func FindEvents(events []core.Event, eventType core.EventType) []core.Event {
	var found []core.Event
	for _, event := range events {
		if event.Type == eventType {
			found = append(found, event)
		}
	}
	return found
}

// CreateTestFile creates a file in the workspace with the given content.
func CreateTestFile(t *testing.T, workspace, path, content string) string {
	t.Helper()

	fullPath := filepath.Join(workspace, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))

	return fullPath
}

// ReadTestFile reads a file from the workspace.
func ReadTestFile(t *testing.T, workspace, path string) string {
	t.Helper()

	fullPath := filepath.Join(workspace, path)
	content, err := os.ReadFile(fullPath)
	require.NoError(t, err, "failed to read file %s", path)

	return string(content)
}

// AssertFileExists asserts that a file exists in the workspace.
func AssertFileExists(t *testing.T, workspace, path string) {
	t.Helper()

	fullPath := filepath.Join(workspace, path)
	_, err := os.Stat(fullPath)
	require.NoError(t, err, "file should exist: %s", path)
}

// AssertFileNotExists asserts that a file does not exist in the workspace.
func AssertFileNotExists(t *testing.T, workspace, path string) {
	t.Helper()

	fullPath := filepath.Join(workspace, path)
	_, err := os.Stat(fullPath)
	require.True(t, os.IsNotExist(err), "file should not exist: %s", path)
}

// AssertFileContent asserts that a file has the expected content.
func AssertFileContent(t *testing.T, workspace, path, expectedContent string) {
	t.Helper()

	actualContent := ReadTestFile(t, workspace, path)
	require.Equal(t, expectedContent, actualContent, "file content mismatch: %s", path)
}

// MeasureTime measures the execution time of a function.
func MeasureTime(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// AssertDuration asserts that a duration is within acceptable bounds.
func AssertDuration(t *testing.T, duration, max time.Duration, operation string) {
	t.Helper()

	require.LessOrEqual(t, duration, max,
		"%s took %v, expected <%v", operation, duration, max)
}

// WaitForCondition waits for a condition to become true or timeout.
func WaitForCondition(timeout time.Duration, condition func() bool) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for condition")
}
