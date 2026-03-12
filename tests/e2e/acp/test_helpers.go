//go:build !e2e_llm_test

// Package acp provides end-to-end tests for the ACP protocol.
package acp

import (
	"context"
	"fmt"
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
	testSettleTime       = 50 * time.Millisecond
	testTimeoutShort     = 2 * time.Second
	testTimeoutLong      = 10 * time.Second
)

const (
	// testTimeout is the default timeout for ACP E2E tests.
	testTimeout = 120 * time.Second

	// binPath is the path to the spin binary (relative to test file).
	binPath = "../../../bin/spin"
)

// getBinPath returns the absolute path to the spin binary.
func getBinPath(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)
	// From tests/e2e/acp/ -> tests/e2e/ -> tests/ -> root.
	root := filepath.Dir(filepath.Dir(filepath.Dir(wd)))

	return filepath.Join(root, "bin", "spin")
}

// createTestConfig creates a minimal test config file without MCP servers.
// Returns the path to the config file.
func createTestConfig(t *testing.T) string {
	t.Helper()

	configContent := `version: "2.0"
llm:
  provider: test-llm
  model: dummy
protocol:
  enable_mcp: false
  enable_git: true
  enable_shell: true
`
	configPath := filepath.Join(t.TempDir(), "test-config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	return configPath
}

// startACPAgent starts the spin acp command as a subprocess and returns the command,
// stdin pipe (for writing to agent), and stdout pipe (for reading from agent).
// The provider is always overridden to "test-llm" so no external LLM is required.
func startACPAgent(t *testing.T, args ...string) (*exec.Cmd, io.WriteCloser, io.ReadCloser) {
	t.Helper()

	binPath := getBinPath(t)

	// Create a minimal test config to avoid loading user's global config with MCP servers.
	configPath := createTestConfig(t)

	// Ensure we always end up with test-llm provider and a dummy model.
	// If callers already passed --provider/--model, these flags will be
	// overridden by the ones we append here (Cobra uses last occurrence).
	override := []string{"--config-file", configPath, "--provider", "test-llm", "--model", "dummy"}

	// Build args: "acp" + test-specific args + overrides.
	cmdArgs := append([]string{"acp"}, args...)
	cmdArgs = append(cmdArgs, override...)

	cmd := exec.CommandContext(t.Context(), binPath, cmdArgs...)

	// Get stdin/stdout pipes.
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	// Stderr goes to os.Stderr for debugging.
	cmd.Stderr = os.Stderr

	// Start the process.
	err = cmd.Start()
	require.NoError(t, err, "Failed to start ACP agent")

	// Give agent a moment to initialize.
	time.Sleep(testSettleTime)

	return cmd, stdin, stdout
}

// cleanupAgent stops and cleans up the agent process.
func cleanupAgent(t *testing.T, cmd *exec.Cmd, stdin io.WriteCloser) {
	t.Helper()

	if stdin != nil {
		stdin.Close()
	}

	if cmd == nil || cmd.Process == nil {
		return
	}

	// Try graceful shutdown first.
	_ = cmd.Process.Signal(os.Interrupt)

	// Wait with timeout.
	done := make(chan error, 1)

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited.
	case <-time.After(testTimeoutShort):
		// Force kill after timeout.
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

	// Create client-side connection.
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
	// Return a copy.
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
func (c *testClient) ReadTextFile(_ context.Context, _ acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, nil
}

// WriteTextFile implements acp.Client interface (not used in basic tests).
func (c *testClient) WriteTextFile(_ context.Context, _ acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, nil
}

// RequestPermission implements acp.Client interface.
func (c *testClient) RequestPermission(_ context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	// For testing, we can auto-approve by selecting the first allow option
	// Find an allow_once or allow_always option.
	var selectedOptionID acp.PermissionOptionId

	for _, option := range params.Options {
		if option.Kind == acp.PermissionOptionKindAllowOnce || option.Kind == acp.PermissionOptionKindAllowAlways {
			selectedOptionID = option.OptionId

			break
		}
	}

	// If no allow option found, use first option (for testing).
	if selectedOptionID == "" && len(params.Options) > 0 {
		selectedOptionID = params.Options[0].OptionId
	}

	return acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeSelected(selectedOptionID),
	}, nil
}

// SessionUpdate implements acp.Client interface.
// This is called by the agent to send notifications.
func (c *testClient) SessionUpdate(_ context.Context, params acp.SessionNotification) error {
	// Store notifications for verification in tests.
	c.mu.Lock()
	defer c.mu.Unlock()

	c.notifications = append(c.notifications, params)

	return nil
}

// CreateTerminal implements acp.Client interface (not used in basic tests).
func (c *testClient) CreateTerminal(_ context.Context, _ acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, nil
}

// KillTerminalCommand implements acp.Client interface (not used in basic tests).
func (c *testClient) KillTerminalCommand(_ context.Context, _ acp.KillTerminalCommandRequest) (acp.KillTerminalCommandResponse, error) {
	return acp.KillTerminalCommandResponse{}, nil
}

// TerminalOutput implements acp.Client interface (not used in basic tests).
func (c *testClient) TerminalOutput(_ context.Context, _ acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, nil
}

// ReleaseTerminal implements acp.Client interface (not used in basic tests).
func (c *testClient) ReleaseTerminal(_ context.Context, _ acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, nil
}

// WaitForTerminalExit implements acp.Client interface (not used in basic tests).
func (c *testClient) WaitForTerminalExit(_ context.Context, _ acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, nil
}

// waitForInitialization waits for the agent to be ready by attempting initialization.
func waitForInitialization(t *testing.T, conn *acp.ClientSideConnection) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeoutLong)
	defer cancel()

	// Try to initialize - this verifies the connection is working.
	_, err := conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion:    acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{},
		ClientInfo: &acp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})
	if err != nil {
		return fmt.Errorf("acp initialize: %w", err)
	}

	return nil
}

// createTestWorkspace creates a temporary directory for testing.
func createTestWorkspace(t *testing.T) string {
	t.Helper()

	return t.TempDir()
}
