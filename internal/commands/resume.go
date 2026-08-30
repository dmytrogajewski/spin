package commands

import (
	"context"
	"errors"
	"fmt"
)

// ErrResumeCommandIsNotAvailableVia is returned when /resume runs without a TUI browser.
var ErrResumeCommandIsNotAvailableVia = errors.New("resume command is not available via ACP protocol")

// SessionBrowser is implemented by TUI command context to list and restore sessions.
type SessionBrowser interface {
	ResumeCatalog(ctx context.Context) string
	ResumeBySelector(ctx context.Context, selector string) (string, error)
}

// ResumeCommand handles /resume.
type ResumeCommand struct{}

// Name implements Command.
func (c *ResumeCommand) Name() string {
	return "/resume"
}

// Description implements Command.
func (c *ResumeCommand) Description() string {
	return "List or continue a previous session"
}

// Execute lists sessions or restores the selected one.
func (c *ResumeCommand) Execute(ctx context.Context, args []string, cmdCtx CommandContext) (string, error) {
	browser, ok := cmdCtx.(SessionBrowser)
	if !ok {
		return "", ErrResumeCommandIsNotAvailableVia
	}

	if len(args) == 0 {
		return browser.ResumeCatalog(ctx), nil
	}

	return browser.ResumeBySelector(ctx, args[0])
}

// IsTUIOnly reports commands that ACP must not advertise.
func IsTUIOnly(name string) bool {
	switch name {
	case "/exit", "/quit", "/resume":
		return true
	default:
		return false
	}
}

// FormatResumeSuccess is the TUI confirmation after a restore.
func FormatResumeSuccess(shortID string, messageCount int, preview string) string {
	line := fmt.Sprintf("✓ Resumed %s (%d messages)", shortID, messageCount)
	if preview != "" {
		return line + "\n  " + preview
	}

	return line
}
