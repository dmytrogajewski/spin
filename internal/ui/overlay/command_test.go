package overlay

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errExecutionFailed = errors.New("execution failed")

func TestNewCommandRegistry(t *testing.T) {
	t.Parallel()

	registry := NewCommandRegistry()
	assert.NotNil(t, registry)
	assert.Empty(t, registry.Commands())
}

func TestCommandRegistry_Register(t *testing.T) {
	t.Parallel()

	registry := NewCommandRegistry()
	cmd := NewSimpleCommand("Test", "A test command", "Test", '🧪', nil)

	registry.Register(cmd)

	assert.Len(t, registry.Commands(), 1)
	assert.Equal(t, "Test", registry.Commands()[0].Name())
}

func TestCommandRegistry_RegisterMultiple(t *testing.T) {
	t.Parallel()

	registry := NewCommandRegistry()
	cmd1 := NewSimpleCommand("First", "First command", "Test", 'A', nil)
	cmd2 := NewSimpleCommand("Second", "Second command", "Test", 'B', nil)
	cmd3 := NewSimpleCommand("Third", "Third command", "Test", 'C', nil)

	registry.Register(cmd1)
	registry.Register(cmd2)
	registry.Register(cmd3)

	assert.Len(t, registry.Commands(), 3)
	assert.Equal(t, "First", registry.Commands()[0].Name())
	assert.Equal(t, "Second", registry.Commands()[1].Name())
	assert.Equal(t, "Third", registry.Commands()[2].Name())
}

func TestSimpleCommand_Fields(t *testing.T) {
	t.Parallel()

	cmd := NewSimpleCommand(
		"Run...",
		"Execute shell command",
		"Edit",
		'▶',
		nil,
	)

	assert.Equal(t, "Run...", cmd.Name())
	assert.Equal(t, "Execute shell command", cmd.Description())
	assert.Equal(t, "Edit", cmd.Category())
	assert.Equal(t, '▶', cmd.Icon())
}

func TestSimpleCommand_ExecuteNil(t *testing.T) {
	t.Parallel()

	cmd := NewSimpleCommand("Test", "Test", "Test", 'T', nil)

	err := cmd.Execute(context.Background())

	assert.NoError(t, err)
}

func TestSimpleCommand_Execute(t *testing.T) {
	t.Parallel()

	executed := false
	cmd := NewSimpleCommand("Test", "Test", "Test", 'T', func(_ context.Context) error {
		executed = true

		return nil
	})

	err := cmd.Execute(context.Background())

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestSimpleCommand_ExecuteError(t *testing.T) {
	t.Parallel()

	expectedErr := errExecutionFailed
	cmd := NewSimpleCommand("Test", "Test", "Test", 'T', func(_ context.Context) error {
		return expectedErr
	})

	err := cmd.Execute(context.Background())

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestSimpleCommand_ExecuteWithContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	var receivedCtx context.Context

	cmd := NewSimpleCommand("Test", "Test", "Test", 'T', func(execCtx context.Context) error {
		receivedCtx = execCtx

		return execCtx.Err()
	})

	err := cmd.Execute(ctx)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.NotNil(t, receivedCtx)
}
