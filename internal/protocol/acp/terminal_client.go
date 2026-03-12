package acp

import (
	"context"
	"errors"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/agent/runtime"
)

// ACPTerminalClient implements runtime.TerminalClient using ACP connection.
type ACPTerminalClient struct {
	connection *acp.AgentSideConnection
}

// NewACPTerminalClient creates a new terminal client.
func NewACPTerminalClient(conn *acp.AgentSideConnection) *ACPTerminalClient {
	return &ACPTerminalClient{
		connection: conn,
	}
}

// Create creates a new terminal and executes a command.
func (c *ACPTerminalClient) Create(ctx context.Context, cmd string, args []string, env []runtime.EnvVar, cwd string, limit int) (string, error) {
	if c.connection == nil {
		return "", errors.New("ACP connection not available")
	}

	// Convert env vars.
	acpEnv := make([]acp.EnvVariable, len(env))
	for i, e := range env {
		acpEnv[i] = acp.EnvVariable{
			Name:  e.Name,
			Value: e.Value,
		}
	}

	// Limit cannot be negative.
	byteLimit := max(limit, 0)

	sessionID := runtime.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return "", errors.New("session ID not found in context")
	}

	params := acp.CreateTerminalRequest{
		SessionId:       acp.SessionId(sessionID),
		Command:         cmd,
		Args:            args,
		Env:             acpEnv,
		Cwd:             &cwd,
		OutputByteLimit: &byteLimit,
	}

	resp, err := c.connection.CreateTerminal(ctx, params)
	if err != nil {
		return "", err
	}

	return string(resp.TerminalId), nil
}

// WaitForExit waits for the terminal command to complete.
func (c *ACPTerminalClient) WaitForExit(ctx context.Context, terminalID string) (int, *string, error) {
	if c.connection == nil {
		return -1, nil, errors.New("ACP connection not available")
	}

	sessionID := runtime.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return -1, nil, errors.New("session ID not found in context")
	}

	params := acp.WaitForTerminalExitRequest{
		SessionId:  acp.SessionId(sessionID),
		TerminalId: terminalID,
	}

	resp, err := c.connection.WaitForTerminalExit(ctx, params)
	if err != nil {
		return -1, nil, err
	}

	var exitCode int
	if resp.ExitCode != nil {
		exitCode = int(*resp.ExitCode)
	}

	return exitCode, resp.Signal, nil
}

// GetOutput retrieves the current terminal output.
func (c *ACPTerminalClient) GetOutput(ctx context.Context, terminalID string) (string, bool, *runtime.ExitStatus, error) {
	if c.connection == nil {
		return "", false, nil, errors.New("ACP connection not available")
	}

	sessionID := runtime.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return "", false, nil, errors.New("session ID not found in context")
	}

	params := acp.TerminalOutputRequest{
		SessionId:  acp.SessionId(sessionID),
		TerminalId: terminalID,
	}

	resp, err := c.connection.TerminalOutput(ctx, params)
	if err != nil {
		return "", false, nil, err
	}

	var exitStatus *runtime.ExitStatus

	if resp.ExitStatus != nil {
		var code *int

		if resp.ExitStatus.ExitCode != nil {
			c := int(*resp.ExitStatus.ExitCode)
			code = &c
		}

		exitStatus = &runtime.ExitStatus{
			ExitCode: code,
			Signal:   resp.ExitStatus.Signal,
		}
	}

	return resp.Output, resp.Truncated, exitStatus, nil
}

// Release releases terminal resources.
func (c *ACPTerminalClient) Release(ctx context.Context, terminalID string) error {
	if c.connection == nil {
		return errors.New("ACP connection not available")
	}

	sessionID := runtime.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return errors.New("session ID not found in context")
	}

	params := acp.ReleaseTerminalRequest{
		SessionId:  acp.SessionId(sessionID),
		TerminalId: terminalID,
	}

	_, err := c.connection.ReleaseTerminal(ctx, params)

	return err
}
