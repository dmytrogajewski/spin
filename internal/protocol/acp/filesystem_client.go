package acp

import (
	"context"
	"errors"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/agent/runtime"
)

// ACPFilesystemClient implements filesystem operations using ACP connection.
type ACPFilesystemClient struct {
	connection *acp.AgentSideConnection
}

// NewACPFilesystemClient creates a new filesystem client.
func NewACPFilesystemClient(conn *acp.AgentSideConnection) *ACPFilesystemClient {
	return &ACPFilesystemClient{
		connection: conn,
	}
}

// ReadTextFile reads a text file using ACP fs/read_text_file protocol.
func (c *ACPFilesystemClient) ReadTextFile(ctx context.Context, path string, line *int, limit *int) (string, error) {
	if c.connection == nil {
		return "", errors.New("ACP connection not available")
	}

	sessionID := runtime.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return "", errors.New("session ID not found in context")
	}

	params := acp.ReadTextFileRequest{
		SessionId: acp.SessionId(sessionID),
		Path:      path,
		Line:      line,
		Limit:     limit,
	}

	resp, err := c.connection.ReadTextFile(ctx, params)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// WriteTextFile writes a text file using ACP fs/write_text_file protocol.
func (c *ACPFilesystemClient) WriteTextFile(ctx context.Context, path, content string) error {
	if c.connection == nil {
		return errors.New("ACP connection not available")
	}

	sessionID := runtime.GetSessionIDFromContext(ctx)
	if sessionID == "" {
		return errors.New("session ID not found in context")
	}

	params := acp.WriteTextFileRequest{
		SessionId: acp.SessionId(sessionID),
		Path:      path,
		Content:   content,
	}

	_, err := c.connection.WriteTextFile(ctx, params)

	return err
}
