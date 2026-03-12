package acp

import (
	"context"
	"fmt"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
)

// TerminalClient implements executor.TerminalClient using ACP connection.
type TerminalClient struct {
	connection *acp.AgentSideConnection
}

// NewTerminalClient creates a new terminal client.
func NewTerminalClient(conn *acp.AgentSideConnection) *TerminalClient {
	return &TerminalClient{
		connection: conn,
	}
}

// Create creates a new terminal and executes a command.
func (c *TerminalClient) Create(
	ctx context.Context, cmd string, args []string, env []executor.EnvVar, cwd string, limit int,
) (string, error) {
	if c.connection == nil {
		return "", ErrAcpConnectionNotAvailable
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

	sessionID := executor.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return "", ErrSessionIDNotFoundInContext
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
		return "", fmt.Errorf("acp create terminal: %w", err)
	}

	return string(resp.TerminalId), nil
}

// WaitForExit waits for the terminal command to complete.
func (c *TerminalClient) WaitForExit(ctx context.Context, terminalID string) (exitCode int, signal *string, err error) {
	if c.connection == nil {
		return -1, nil, ErrAcpConnectionNotAvailable
	}

	sessionID := executor.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return -1, nil, ErrSessionIDNotFoundInContext
	}

	params := acp.WaitForTerminalExitRequest{
		SessionId:  acp.SessionId(sessionID),
		TerminalId: terminalID,
	}

	resp, err := c.connection.WaitForTerminalExit(ctx, params)
	if err != nil {
		return -1, nil, fmt.Errorf("acp wait for terminal exit: %w", err)
	}

	if resp.ExitCode != nil {
		exitCode = int(*resp.ExitCode)
	}

	return exitCode, resp.Signal, nil
}

// GetOutput retrieves the current terminal output.
func (c *TerminalClient) GetOutput(
	ctx context.Context, terminalID string,
) (output string, truncated bool, exitStatus *executor.ExitStatus, err error) {
	if c.connection == nil {
		return "", false, nil, ErrAcpConnectionNotAvailable
	}

	sessionID := executor.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return "", false, nil, ErrSessionIDNotFoundInContext
	}

	params := acp.TerminalOutputRequest{
		SessionId:  acp.SessionId(sessionID),
		TerminalId: terminalID,
	}

	resp, err := c.connection.TerminalOutput(ctx, params)
	if err != nil {
		return "", false, nil, fmt.Errorf("acp terminal output: %w", err)
	}

	if resp.ExitStatus != nil {
		var code *int

		if resp.ExitStatus.ExitCode != nil {
			ec := int(*resp.ExitStatus.ExitCode)
			code = &ec
		}

		exitStatus = &executor.ExitStatus{
			ExitCode: code,
			Signal:   resp.ExitStatus.Signal,
		}
	}

	return resp.Output, resp.Truncated, exitStatus, nil
}

// Release releases terminal resources.
func (c *TerminalClient) Release(ctx context.Context, terminalID string) error {
	if c.connection == nil {
		return ErrAcpConnectionNotAvailable
	}

	sessionID := executor.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return ErrSessionIDNotFoundInContext
	}

	params := acp.ReleaseTerminalRequest{
		SessionId:  acp.SessionId(sessionID),
		TerminalId: terminalID,
	}

	_, err := c.connection.ReleaseTerminal(ctx, params)
	if err != nil {
		return fmt.Errorf("acp release terminal: %w", err)
	}

	return nil
}
