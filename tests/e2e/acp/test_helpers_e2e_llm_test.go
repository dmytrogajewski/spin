//go:build e2e_llm_test

package acp

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/require"
)

const (
	testTimeout = 120 * time.Second
	binPath     = "../../../bin/spin"
)

// getBinPath returns the absolute path to the spin binary.
func getBinPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	// From tests/e2e/acp/ -> tests/e2e/ -> tests/ -> root
	root := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
	return filepath.Join(root, "bin", "spin")
}

// startACPAgent starts the spin acp command as a subprocess and returns the command,
// stdin pipe (for writing to agent), and stdout pipe (for reading from agent).
// When built with e2e_llm_test, we force the provider to "test-llm" so no
// external LLM is required.
func startACPAgent(t *testing.T, args ...string) (*exec.Cmd, io.WriteCloser, io.ReadCloser) {
	t.Helper()

	binPath := getBinPath(t)

	// Ensure we always end up with test-llm provider and a dummy model.
	// If callers already passed --provider/--model, these flags will be
	// overridden by the ones we append here (Cobra uses last occurrence).
	override := []string{"--provider", "test-llm", "--model", "dummy"}

	// Build args: "acp" + test-specific overrides + additional args
	cmdArgs := append([]string{"acp"}, args...)
	cmdArgs = append(cmdArgs, override...)

	cmd := exec.Command(binPath, cmdArgs...)

	// Get stdin/stdout pipes
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	// Stderr goes to os.Stderr for debugging
	cmd.Stderr = os.Stderr

	// Start the process
	err = cmd.Start()
	require.NoError(t, err, "Failed to start ACP agent")

	// Give agent a moment to initialize
	time.Sleep(50 * time.Millisecond)

	return cmd, stdin, stdout
}

// The rest of the helpers are identical to the non-tagged version.

// cleanupAgent stops and cleans up the agent process.
func cleanupAgent(t *testing.T, cmd *exec.Cmd, stdin io.WriteCloser) {
	t.Helper()

	if stdin != nil {
		stdin.Close()
	}

	if cmd == nil || cmd.Process == nil {
		return
	}

	// Try graceful shutdown first
	_ = cmd.Process.Signal(os.Interrupt)

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited
	case <-time.After(2 * time.Second):
		// Force kill after timeout
		_ = cmd.Process.Kill()
		<-done
	}
}

// createACPClient creates an ACP client-side connection.
// The client implementation is a simple wrapper that handles requests.
func createACPClient(t *testing.T, stdin io.Writer, stdout io.Reader) *acp.ClientSideConnection {
	t.Helper()
	return createACPClientWithClient(t, stdin, stdout, &testClient{})
}

// createACPClientWithClient creates an ACP client-side connection with a custom client.
func createACPClientWithClient(t *testing.T, stdin io.Writer, stdout io.Reader, client *testClient) *acp.ClientSideConnection {
	t.Helper()

	// Create client-side connection
	conn := acp.NewClientSideConnection(client, stdin, stdout)

	return conn
}

// testClient is a minimal ACP client implementation for testing.
// It implements the acp.Client interface with stub methods.
// It tracks notifications for verification in tests.
type testClient struct {
	notifications []acp.SessionNotification
	mu            sync.Mutex
}

// getNotifications returns all received notifications.
func (c *testClient) getNotifications() []acp.SessionNotification {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Return a copy
	result := make([]acp.SessionNotification, len(c.notifications))
	copy(result, c.notifications)
	return result
}

// clearNotifications clears all stored notifications.
func (c *testClient) clearNotifications() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notifications = nil
}

// ReadTextFile implements acp.Client interface (not used in basic tests).
func (c *testClient) ReadTextFile(ctx context.Context, params acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, nil
}

// WriteTextFile implements acp.Client interface (not used in basic tests).
func (c *testClient) WriteTextFile(ctx context.Context, params acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, nil
}

// RequestPermission implements acp.Client interface.
func (c *testClient) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	// For testing, we can auto-approve by selecting the first allow option
	// Find an allow_once or allow_always option
	var selectedOptionID acp.PermissionOptionId
	for _, option := range params.Options {
		if option.Kind == acp.PermissionOptionKindAllowOnce || option.Kind == acp.PermissionOptionKindAllowAlways {
			selectedOptionID = option.OptionId
			break
		}
	}

	// If no allow option found, use first option (for testing)
	if selectedOptionID == "" && len(params.Options) > 0 {
		selectedOptionID = params.Options[0].OptionId
	}

	return acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeSelected(selectedOptionID),
	}, nil
}

// SessionUpdate implements acp.Client interface.
// This is called by the agent to send notifications.
func (c *testClient) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
	// Store notifications for verification in tests
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notifications = append(c.notifications, params)
	return nil
}

// CreateTerminal implements acp.Client interface (not used in basic tests).
func (c *testClient) CreateTerminal(ctx context.Context, params acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{
		TerminalId: "test-terminal-1",
	}, nil
}

// KillTerminalCommand implements acp.Client interface (not used in basic tests).
func (c *testClient) KillTerminalCommand(ctx context.Context, params acp.KillTerminalCommandRequest) (acp.KillTerminalCommandResponse, error) {
	return acp.KillTerminalCommandResponse{}, nil
}

// TerminalOutput implements acp.Client interface (not used in basic tests).
func (c *testClient) TerminalOutput(ctx context.Context, params acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, nil
}

// ReleaseTerminal implements acp.Client interface (not used in basic tests).
func (c *testClient) ReleaseTerminal(ctx context.Context, params acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, nil
}

// WaitForTerminalExit implements acp.Client interface (not used in basic tests).
func (c *testClient) WaitForTerminalExit(ctx context.Context, params acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	// Return successful exit for test commands
	exitCode := 0
	return acp.WaitForTerminalExitResponse{
		ExitCode: &exitCode,
	}, nil
}

// waitForInitialization waits for the agent to be ready by attempting initialization.
func waitForInitialization(t *testing.T, conn *acp.ClientSideConnection) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to initialize - this verifies the connection is working
	_, err := conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion:    acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{},
		ClientInfo: &acp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})

	return err
}

// createTestWorkspace creates a temporary directory for testing.
func createTestWorkspace(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "spin-acp-e2e-*")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}

// waitForNotification waits for a specific notification type to arrive.
func waitForNotification(t *testing.T, client *testClient, timeout time.Duration, check func(acp.SessionNotification) bool) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		notifications := client.getNotifications()
		for _, notif := range notifications {
			if check(notif) {
				return true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// verifySessionUpdate verifies that a session/update notification was received.
func verifySessionUpdate(t *testing.T, client *testClient, check func(acp.SessionUpdate) bool) bool {
	t.Helper()

	notifications := client.getNotifications()
	for _, notif := range notifications {
		if check(notif.Update) {
			return true
		}
	}
	return false
}

// createTestFile creates a test file with the given content.
func createTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()

	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

// verifyFileContents verifies that a file has the expected content.
func verifyFileContents(t *testing.T, filePath, expectedContent string) {
	t.Helper()

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, expectedContent, string(content))
}

// createMockMCPServer creates a simple mock MCP server process for testing.
// Returns the command path and args that can be used in McpServer config.
func createMockMCPServer(t *testing.T) (string, []string) {
	t.Helper()

	// Use echo as a simple mock server
	// In real tests, you might want a more sophisticated mock
	return "/bin/echo", []string{"mcp-server-response"}
}

// waitForToolCall waits for a tool call notification.
func waitForToolCall(t *testing.T, client *testClient, timeout time.Duration) bool {
	t.Helper()

	return waitForNotification(t, client, timeout, func(notif acp.SessionNotification) bool {
		return notif.Update.ToolCall != nil
	})
}

// waitForPlanUpdate waits for a plan update notification.
func waitForPlanUpdate(t *testing.T, client *testClient, timeout time.Duration) bool {
	t.Helper()

	return waitForNotification(t, client, timeout, func(notif acp.SessionNotification) bool {
		return notif.Update.Plan != nil
	})
}

// waitForModeUpdate waits for a current_mode_update notification.
func waitForModeUpdate(t *testing.T, client *testClient, timeout time.Duration) bool {
	t.Helper()

	return waitForNotification(t, client, timeout, func(notif acp.SessionNotification) bool {
		return notif.Update.CurrentModeUpdate != nil
	})
}

// waitForCommandsUpdate waits for an available_commands_update notification.
func waitForCommandsUpdate(t *testing.T, client *testClient, timeout time.Duration) bool {
	t.Helper()

	return waitForNotification(t, client, timeout, func(notif acp.SessionNotification) bool {
		return notif.Update.AvailableCommandsUpdate != nil
	})
}
