package acp

import (
	"context"
	"errors"
	"fmt"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
)

var (
	// ErrAcpConnectionNotAvailable is a sentinel error.
	ErrAcpConnectionNotAvailable = errors.New("ACP connection not available")
	// ErrSessionIDNotFoundInContext is a sentinel error.
	ErrSessionIDNotFoundInContext = errors.New("session ID not found in context")
)

// FilesystemClient implements filesystem operations using ACP connection.
type FilesystemClient struct {
	connection *acp.AgentSideConnection
}

// NewFilesystemClient creates a new filesystem client.
func NewFilesystemClient(conn *acp.AgentSideConnection) *FilesystemClient {
	return &FilesystemClient{
		connection: conn,
	}
}

// ReadTextFile reads a text file using ACP fs/read_text_file protocol.
func (c *FilesystemClient) ReadTextFile(ctx context.Context, path string, line, limit *int) (string, error) {
	if c.connection == nil {
		return "", ErrAcpConnectionNotAvailable
	}

	sessionID := executor.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return "", ErrSessionIDNotFoundInContext
	}

	params := acp.ReadTextFileRequest{
		SessionId: acp.SessionId(sessionID),
		Path:      path,
		Line:      line,
		Limit:     limit,
	}

	resp, err := c.connection.ReadTextFile(ctx, params)
	if err != nil {
		return "", fmt.Errorf("acp read text file: %w", err)
	}

	return resp.Content, nil
}

// WriteTextFile writes a text file using ACP fs/write_text_file protocol.
func (c *FilesystemClient) WriteTextFile(ctx context.Context, path, content string) error {
	if c.connection == nil {
		return ErrAcpConnectionNotAvailable
	}

	sessionID := executor.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return ErrSessionIDNotFoundInContext
	}

	params := acp.WriteTextFileRequest{
		SessionId: acp.SessionId(sessionID),
		Path:      path,
		Content:   content,
	}

	_, err := c.connection.WriteTextFile(ctx, params)
	if err != nil {
		return fmt.Errorf("acp write text file: %w", err)
	}

	return nil
}
