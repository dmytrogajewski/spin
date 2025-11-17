package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCommandContext is a mock implementation of CommandContext for testing.
type mockCommandContext struct {
	currentMode string
	workDir     string
	setModeErr  error
}

func (m *mockCommandContext) GetCurrentMode() string {
	return m.currentMode
}

func (m *mockCommandContext) SetMode(mode string) error {
	if m.setModeErr != nil {
		return m.setModeErr
	}
	m.currentMode = mode
	return nil
}

func (m *mockCommandContext) GetWorkDir() string {
	return m.workDir
}

// TestParseCommand tests command parsing.
func TestParseCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCmd   string
		wantArgs  []string
		wantIsCmd bool
	}{
		{
			name:      "slash command with no args",
			input:     "/mode",
			wantCmd:   "/mode",
			wantArgs:  []string{},
			wantIsCmd: true,
		},
		{
			name:      "slash command with one arg",
			input:     "/mode review",
			wantCmd:   "/mode",
			wantArgs:  []string{"review"},
			wantIsCmd: true,
		},
		{
			name:      "slash command with multiple args",
			input:     "/mode review test",
			wantCmd:   "/mode",
			wantArgs:  []string{"review", "test"},
			wantIsCmd: true,
		},
		{
			name:      "command with leading/trailing whitespace",
			input:     "  /help  ",
			wantCmd:   "/help",
			wantArgs:  []string{},
			wantIsCmd: true,
		},
		{
			name:      "regular message",
			input:     "Write a test for the auth module",
			wantCmd:   "",
			wantArgs:  nil,
			wantIsCmd: false,
		},
		{
			name:      "message with slash in middle",
			input:     "Use the /api endpoint to fetch data",
			wantCmd:   "",
			wantArgs:  nil,
			wantIsCmd: false,
		},
		{
			name:      "empty input",
			input:     "",
			wantCmd:   "",
			wantArgs:  nil,
			wantIsCmd: false,
		},
		{
			name:      "just slash",
			input:     "/",
			wantCmd:   "",
			wantArgs:  nil,
			wantIsCmd: false,
		},
		{
			name:      "case insensitive command",
			input:     "/MODE REVIEW",
			wantCmd:   "/mode",
			wantArgs:  []string{"review"},
			wantIsCmd: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args, isCmd := ParseCommand(tt.input)
			assert.Equal(t, tt.wantIsCmd, isCmd)
			if tt.wantIsCmd {
				assert.Equal(t, tt.wantCmd, cmd)
				assert.Equal(t, tt.wantArgs, args)
			}
		})
	}
}

// TestModeCommand tests the /mode command.
func TestModeCommand(t *testing.T) {
	cmd := &ModeCommand{}
	assert.Equal(t, "/mode", cmd.Name())
	assert.Equal(t, "Show current mode or switch to a different mode", cmd.Description())

	t.Run("show current mode", func(t *testing.T) {
		ctx := &mockCommandContext{currentMode: "regular"}
		result, err := cmd.Execute(context.Background(), []string{}, ctx)
		require.NoError(t, err)
		assert.Contains(t, result, "Current mode: regular")
	})

	t.Run("switch mode", func(t *testing.T) {
		ctx := &mockCommandContext{currentMode: "regular"}
		result, err := cmd.Execute(context.Background(), []string{"review"}, ctx)
		require.NoError(t, err)
		assert.Contains(t, result, "Switched to review mode")
		assert.Equal(t, "review", ctx.GetCurrentMode())
	})

	t.Run("invalid mode", func(t *testing.T) {
		ctx := &mockCommandContext{currentMode: "regular"}
		_, err := cmd.Execute(context.Background(), []string{"invalid"}, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mode")
	})
}

// TestHelpCommand tests the /help command.
func TestHelpCommand(t *testing.T) {
	cmd := &HelpCommand{}
	assert.Equal(t, "/help", cmd.Name())
	assert.Equal(t, "Show this help message", cmd.Description())

	ctx := &mockCommandContext{}
	result, err := cmd.Execute(context.Background(), []string{}, ctx)
	require.NoError(t, err)
	assert.Contains(t, result, "Available commands:")
	assert.Contains(t, result, "/mode")
	assert.Contains(t, result, "/help")
	assert.Contains(t, result, "Available modes:")
	assert.Contains(t, result, "regular")
	assert.Contains(t, result, "review")
	assert.Contains(t, result, "compact")
	assert.Contains(t, result, "planning")
}

// TestExitCommand tests the /exit command.
func TestExitCommand(t *testing.T) {
	cmd := &ExitCommand{}
	assert.Equal(t, "/exit", cmd.Name())

	ctx := &mockCommandContext{}
	_, err := cmd.Execute(context.Background(), []string{}, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available via ACP")
}

// TestQuitCommand tests the /quit command.
func TestQuitCommand(t *testing.T) {
	cmd := &QuitCommand{}
	assert.Equal(t, "/quit", cmd.Name())

	ctx := &mockCommandContext{}
	_, err := cmd.Execute(context.Background(), []string{}, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available via ACP")
}

// TestExecuteCommand tests command execution.
func TestExecuteCommand(t *testing.T) {
	ctx := &mockCommandContext{currentMode: "regular"}

	t.Run("valid command", func(t *testing.T) {
		result, err := ExecuteCommand(context.Background(), "/mode", []string{}, ctx)
		require.NoError(t, err)
		assert.Contains(t, result, "Current mode")
	})

	t.Run("unknown command", func(t *testing.T) {
		_, err := ExecuteCommand(context.Background(), "/unknown", []string{}, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command")
	})
}

// TestListCommands tests command listing.
func TestListCommands(t *testing.T) {
	commands := ListCommands()
	assert.Greater(t, len(commands), 0)

	// Check that expected commands are registered
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	assert.True(t, commandNames["/mode"], "should have /mode command")
	assert.True(t, commandNames["/help"], "should have /help command")
	assert.True(t, commandNames["/exit"], "should have /exit command")
	assert.True(t, commandNames["/quit"], "should have /quit command")
}

