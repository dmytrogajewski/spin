package acp

import (
	"context"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/commands"
)

// acpCommandContext implements commands.CommandContext for ACP protocol.
type acpCommandContext struct {
	agent     *SpinACPAgent
	sessionID acp.SessionId
}

// GetCurrentMode returns the current task mode for the session.
func (c *acpCommandContext) GetCurrentMode() string {
	c.agent.mu.RLock()
	defer c.agent.mu.RUnlock()

	modeID, exists := c.agent.sessionModes[c.sessionID]
	if !exists {
		return "regular" // Default mode.
	}

	return string(modeID)
}

// SetMode sets the task mode for the session.
func (c *acpCommandContext) SetMode(mode string) error {
	// Use SetSessionMode to change the mode.
	req := acp.SetSessionModeRequest{
		SessionId: c.sessionID,
		ModeId:    acp.SessionModeId(mode),
	}
	_, err := c.agent.SetSessionMode(context.Background(), req)

	return err
}

// GetWorkDir returns the working directory for the session.
func (c *acpCommandContext) GetWorkDir() string {
	c.agent.mu.RLock()
	defer c.agent.mu.RUnlock()

	sess, exists := c.agent.sessions[c.sessionID]
	if !exists {
		return ""
	}

	return sess.WorkDir
}

// executeCommand executes a command in the ACP context.
func (a *SpinACPAgent) executeCommand(ctx context.Context, commandName string, args []string, sessionID acp.SessionId) (string, error) {
	// Create command context.
	cmdCtx := &acpCommandContext{
		agent:     a,
		sessionID: sessionID,
	}

	// Execute command.
	return commands.ExecuteCommand(ctx, commandName, args, cmdCtx)
}
