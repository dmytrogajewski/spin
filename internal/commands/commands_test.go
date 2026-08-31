package commands

// Journey: specs/journeys/JOURNEY-002-discover-skill-catalog.md.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/skills"
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

func (m *mockCommandContext) SetMode(_ context.Context, mode string) error {
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
	t.Parallel()

	tests := []struct {
		name, input, wantCmd string
		wantArgs             []string
		wantIsCmd            bool
	}{
		{"slash command with no args", "/mode", "/mode", []string{}, true},
		{"slash command with one arg", "/mode review", "/mode", []string{"review"}, true},
		{"slash command with multiple args", "/mode review test", "/mode", []string{"review", "test"}, true},
		{"command with leading/trailing whitespace", "  /help  ", "/help", []string{}, true},
		{"regular message", "Write a test for the auth module", "", nil, false},
		{"message with slash in middle", "Use the /api endpoint to fetch data", "", nil, false},
		{"empty input", "", "", nil, false},
		{"just slash", "/", "", nil, false},
		{"case insensitive command", "/MODE REVIEW", "/mode", []string{"review"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

	cmd := &ModeCommand{}
	assert.Equal(t, "/mode", cmd.Name())
	assert.Equal(t, "Show current mode or switch to a different mode", cmd.Description())

	t.Run("show current mode", func(t *testing.T) {
		t.Parallel()

		ctx := &mockCommandContext{currentMode: "regular"}
		result, err := cmd.Execute(context.Background(), []string{}, ctx)
		require.NoError(t, err)
		assert.Contains(t, result, "Current mode: regular")
	})

	t.Run("switch mode", func(t *testing.T) {
		t.Parallel()

		ctx := &mockCommandContext{currentMode: "regular"}
		result, err := cmd.Execute(context.Background(), []string{"review"}, ctx)
		require.NoError(t, err)
		assert.Contains(t, result, "Switched to review mode")
		assert.Equal(t, "review", ctx.GetCurrentMode())
	})

	t.Run("invalid mode", func(t *testing.T) {
		t.Parallel()

		ctx := &mockCommandContext{currentMode: "regular"}
		_, err := cmd.Execute(context.Background(), []string{"invalid"}, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mode")
	})
}

// TestHelpCommand tests the /help command.
func TestHelpCommand(t *testing.T) {
	t.Parallel()

	cmd := &HelpCommand{}
	assert.Equal(t, "/help", cmd.Name())
	assert.Equal(t, "Show this help message", cmd.Description())

	ctx := &mockCommandContext{}
	result, err := cmd.Execute(context.Background(), []string{}, ctx)
	require.NoError(t, err)
	assert.Contains(t, result, "Available commands:")
	assert.Contains(t, result, "/mode")
	assert.Contains(t, result, "/help")
	assert.Contains(t, result, "/resume")
	assert.Contains(t, result, "/skills")
	assert.Contains(t, result, "/tasks")
	assert.Contains(t, result, "/agents")
	assert.Contains(t, result, "/task wait")
	assert.Contains(t, result, "Available modes:")
	assert.Contains(t, result, "regular")
	assert.Contains(t, result, "review")
	assert.Contains(t, result, "compact")
	assert.Contains(t, result, "planning")
	assert.Contains(t, result, "SPIN_COMPACT=0")
	assert.Contains(t, result, "compact.enabled")
	assert.Contains(t, result, "/<skill>")
	assert.Contains(t, result, "@path")
	assert.Contains(t, result, "Ctrl-V")
}

// TestExitCommand tests the /exit command.
func TestExitCommand(t *testing.T) {
	t.Parallel()

	cmd := &ExitCommand{}
	assert.Equal(t, "/exit", cmd.Name())

	ctx := &mockCommandContext{}
	_, err := cmd.Execute(context.Background(), []string{}, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available via ACP")
}

// TestQuitCommand tests the /quit command.
func TestQuitCommand(t *testing.T) {
	t.Parallel()

	cmd := &QuitCommand{}
	assert.Equal(t, "/quit", cmd.Name())

	ctx := &mockCommandContext{}
	_, err := cmd.Execute(context.Background(), []string{}, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available via ACP")
}

// mockSessionBrowser extends mockCommandContext with resume hooks.
type mockSessionBrowser struct {
	mockCommandContext

	catalog  string
	resumeTo string
}

func (m *mockSessionBrowser) ResumeCatalog(_ context.Context) string {
	return m.catalog
}

func (m *mockSessionBrowser) ResumeBySelector(_ context.Context, selector string) (string, error) {
	m.resumeTo = selector

	return "resumed " + selector, nil
}

func TestResumeCommand_ListsWithoutArgs(t *testing.T) {
	t.Parallel()

	cmd := &ResumeCommand{}
	assert.Equal(t, "/resume", cmd.Name())

	ctx := &mockSessionBrowser{catalog: "Resumable sessions:"}
	result, err := cmd.Execute(context.Background(), nil, ctx)
	require.NoError(t, err)
	assert.Equal(t, "Resumable sessions:", result)
}

func TestResumeCommand_Selects(t *testing.T) {
	t.Parallel()

	ctx := &mockSessionBrowser{}
	result, err := (&ResumeCommand{}).Execute(context.Background(), []string{"last"}, ctx)
	require.NoError(t, err)
	assert.Equal(t, "resumed last", result)
	assert.Equal(t, "last", ctx.resumeTo)
}

func TestResumeCommand_RequiresBrowser(t *testing.T) {
	t.Parallel()

	_, err := (&ResumeCommand{}).Execute(context.Background(), nil, &mockCommandContext{})
	require.ErrorIs(t, err, ErrResumeCommandIsNotAvailableVia)
}

func TestIsTUIOnly(t *testing.T) {
	t.Parallel()

	assert.True(t, IsTUIOnly("/resume"))
	assert.True(t, IsTUIOnly("/exit"))
	assert.False(t, IsTUIOnly("/mode"))
	assert.False(t, IsTUIOnly("/skills"))
	assert.False(t, IsTUIOnly("/tasks"))
	assert.False(t, IsTUIOnly("/agents"))
}

// TestExecuteCommand tests command execution.
func TestExecuteCommand(t *testing.T) {
	t.Parallel()

	ctx := &mockCommandContext{currentMode: "regular"}

	t.Run("valid command", func(t *testing.T) {
		t.Parallel()

		result, err := ExecuteCommand(context.Background(), "/mode", []string{}, ctx)
		require.NoError(t, err)
		assert.Contains(t, result, "Current mode")
	})

	t.Run("unknown command", func(t *testing.T) {
		t.Parallel()

		_, err := ExecuteCommand(context.Background(), "/unknown", []string{}, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command")
	})
}

// TestListCommands tests command listing.
func TestListCommands(t *testing.T) {
	t.Parallel()

	commands := ListCommands()
	assert.NotEmpty(t, commands)

	// Check that expected commands are registered.
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	assert.True(t, commandNames["/mode"], "should have /mode command")
	assert.True(t, commandNames["/help"], "should have /help command")
	assert.True(t, commandNames["/exit"], "should have /exit command")
	assert.True(t, commandNames["/quit"], "should have /quit command")
	assert.True(t, commandNames["/resume"], "should have /resume command")
	assert.True(t, commandNames["/skills"], "should have /skills command")
	assert.True(t, commandNames["/tasks"], "should have /tasks command")
	assert.True(t, commandNames["/task"], "should have /task command")
	assert.True(t, commandNames["/agents"], "should have /agents command")
}

func TestSkillsCommand_ListsCatalog(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	skillDir := filepath.Join(workDir, ".agents", "skills", "alpha")
	require.NoError(t, os.MkdirAll(skillDir, 0o750))

	content := "---\nname: alpha\ndescription: Project alpha description.\n---\n\n## MustNotAppearInCatalog\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, skills.FileName), []byte(content), 0o600))

	cmd := &SkillsCommand{
		options: func(dir string) skills.Options {
			return skills.Options{WorkDir: dir}
		},
	}
	assert.Equal(t, "/skills", cmd.Name())
	assert.NotEmpty(t, cmd.Description())

	result, err := cmd.Execute(context.Background(), nil, &mockCommandContext{workDir: workDir})
	require.NoError(t, err)
	assert.Contains(t, result, "alpha")
	assert.Contains(t, result, string(skills.SourceProject))
	assert.Contains(t, result, "Project alpha description.")
	assert.NotContains(t, result, "## MustNotAppearInCatalog")
}

func TestSkillsCommand_EmptyCatalog(t *testing.T) {
	t.Parallel()

	cmd := &SkillsCommand{
		options: func(dir string) skills.Options {
			return skills.Options{WorkDir: dir}
		},
	}
	result, err := cmd.Execute(context.Background(), nil, &mockCommandContext{workDir: t.TempDir()})
	require.NoError(t, err)
	assert.Equal(t, skills.EmptyCatalogMessage, result)
}
