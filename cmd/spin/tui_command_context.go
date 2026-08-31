package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
	"github.com/dmytrogajewski/spin/internal/commands"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

// ErrExitRequested is a sentinel error.
var ErrExitRequested = errors.New("exit requested")

// tuiCommandContext implements commands.CommandContext for TUI.
type tuiCommandContext struct {
	conv *conversation.Conversation
}

// GetCurrentMode returns the current task mode for the session.
func (c *tuiCommandContext) GetCurrentMode() string {
	return c.conv.GetTaskMode()
}

// SetMode sets the task mode for the session.
func (c *tuiCommandContext) SetMode(_ context.Context, mode string) error {
	return c.conv.SetTaskMode(mode)
}

// GetWorkDir returns the working directory for the session.
func (c *tuiCommandContext) GetWorkDir() string {
	return c.conv.GetWorkDir()
}

// AgentTasks implements commands.TaskSource.
func (c *tuiCommandContext) AgentTasks() *tasks.Registry {
	if c == nil || c.conv == nil {
		return nil
	}

	return c.conv.GetTaskRegistry()
}

// ShellTasks implements commands.ShellTaskSource.
func (c *tuiCommandContext) ShellTasks() *tools.ShellAdapter {
	if c == nil || c.conv == nil {
		return nil
	}

	return c.conv.GetShellTasks()
}

// ResumeCatalog implements commands.SessionBrowser.
func (c *tuiCommandContext) ResumeCatalog(ctx context.Context) string {
	return session.FormatResumeList(c.conv.ListResumableSessions(ctx), time.Now())
}

// ResumeBySelector implements commands.SessionBrowser.
func (c *tuiCommandContext) ResumeBySelector(ctx context.Context, selector string) (string, error) {
	cands := c.conv.ListResumableSessions(ctx)

	picked, err := session.ResolveSelector(cands, selector)
	if err != nil {
		return "", err
	}

	if resumeErr := c.conv.Resume(ctx, picked.ID); resumeErr != nil {
		return "", resumeErr
	}

	return commands.FormatResumeSuccess(
		session.ShortID(picked.ID),
		len(c.conv.GetHistoryMessages()),
		picked.Preview,
	), nil
}

// handleCommand processes slash commands using the command system.
// Returns an error if exit was requested ([ErrExitRequested]) or the command failed.
func handleCommand(
	ctx context.Context, ui *adapters.PureTTY,
	conv *conversation.Conversation, cmdName string, args []string,
) error {
	// Create command context.
	cmdCtx := &tuiCommandContext{conv: conv}

	// Special handling for exit/quit commands (TUI-only).
	if cmdName == "/exit" || cmdName == "/quit" {
		return ErrExitRequested
	}

	// Execute command via command system.
	result, err := commands.ExecuteCommand(ctx, cmdName, args, cmdCtx)
	if err != nil {
		// Check if it's an unknown command.
		if strings.Contains(err.Error(), "unknown command") {
			_ = ui.PrintLine(fmt.Sprintf("Unknown command: %s (type /help for available commands)\n", cmdName))

			return nil
		}
		// Other errors.
		_ = ui.PrintLine(fmt.Sprintf("Error: %v\n", err))

		return nil
	}

	if cmdName == "/resume" {
		ui.SetConversationID(conv.ID())
		ui.SetTokenCount(int64(conv.GetTokenCount()))
	}

	// Print command output.
	_ = ui.PrintLine(result)

	if !strings.HasSuffix(result, "\n") {
		_ = ui.PrintLine("")
	}

	return nil
}
