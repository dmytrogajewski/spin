package acp

import (
	"context"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// interceptingClient allows capturing requests from the server.
type interceptingClient struct {
	*testClient
	createTerminalFunc func(context.Context, acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error)
}

func (c *interceptingClient) CreateTerminal(ctx context.Context, params acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	if c.createTerminalFunc != nil {
		return c.createTerminalFunc(ctx, params)
	}

	return c.testClient.CreateTerminal(ctx, params)
}

func TestACPTerminalExecution(t *testing.T) {
	t.Parallel()

	// 1. Start the ACP agent process.
	cmd, stdin, stdout := startACPAgent(t)
	defer cleanupAgent(t, cmd, stdin)

	// 2. Setup client with interception.
	terminalCreated := make(chan bool, 1)

	handler := &interceptingClient{
		testClient: &testClient{},
		createTerminalFunc: func(_ context.Context, _ acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
			terminalCreated <- true

			return acp.CreateTerminalResponse{
				TerminalId: "term_1",
			}, nil
		},
	}

	// 3. Create connection.
	client := acp.NewClientSideConnection(handler, stdin, stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 4. Initialize with terminal capabilities.
	initReq := acp.InitializeRequest{
		ProtocolVersion: 1,
		ClientInfo: &acp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true, // Advertise terminal support.
		},
	}

	initResp, err := client.Initialize(ctx, initReq)
	require.NoError(t, err)
	assert.Equal(t, acp.ProtocolVersion(1), initResp.ProtocolVersion)

	// 5. Start a session.
	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        ".",
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	sessionID := sessionResp.SessionId

	// 6. Send a prompt that triggers a shell command
	// Use a command that definitely requires shell/execution.
	_, err = client.Prompt(ctx, acp.PromptRequest{
		SessionId: sessionID,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("execute 'echo hello' in the terminal"),
		},
	})
	require.NoError(t, err)

	// 7. Verify terminal/create is called.
	select {
	case <-terminalCreated:
		// Success!
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for terminal/create request. The agent likely fell back to local execution or failed.")
	}
}
