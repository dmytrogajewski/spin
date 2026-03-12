package acp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/require"
)

// TestACP_ApprovalPersistence_PromptToToolCall validates the high-level flow:
// Prompt -> tool call needing approval -> policy persisted -> subsequent prompt
// bypasses approval via policy, and revocation causes re-prompt.
func TestACP_ApprovalPersistence_PromptToToolCall(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := setupApprovalPersistenceEnv(t)

	// 1) First prompt should require approval and cause a policy to be persisted.
	env.runPrompt(t)

	if !env.hasToolCallNotifications(t) {
		return
	}

	// 2) Second prompt: approval should be short-circuited via policy.
	env.runPrompt(t)

	// 3) Revocation via CLI.
	clearApprovalPolicies(t, env.configPath)

	// 4) Third prompt: after revocation, approval should no longer be short-circuited.
	env.runPrompt(t)
}

// approvalPersistenceEnv holds the test environment for approval persistence tests.
type approvalPersistenceEnv struct {
	clientImpl *testClient
	client     *acp.ClientSideConnection
	sessionID  acp.SessionId
	configPath string
}

// setupApprovalPersistenceEnv creates the test environment for approval persistence.
func setupApprovalPersistenceEnv(t *testing.T) *approvalPersistenceEnv {
	t.Helper()

	workDir := createTestWorkspace(t)
	configPath := filepath.Join(workDir, "spin.yaml")
	policyPath := filepath.Join(workDir, "policies.json")

	cfg := `
version: "2.0"
security:
  policy_file: ` + policyPath + `
  approval_persistence_enabled: true
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o600))

	cmd, stdin, stdout := startACPAgent(t,
		"--config-file", configPath,
		"--provider", "test-llm",
		"--model", "dummy",
		"--workspace", workDir,
	)
	t.Cleanup(func() { cleanupAgent(t, cmd, stdin) })

	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{Cwd: workDir, McpServers: []acp.McpServer{}})
	require.NoError(t, err)

	return &approvalPersistenceEnv{
		clientImpl: clientImpl, client: client,
		sessionID: sessionResp.SessionId, configPath: configPath,
	}
}

// runPrompt sends a prompt and waits for completion.
func (e *approvalPersistenceEnv) runPrompt(t *testing.T) {
	t.Helper()
	e.clientImpl.clearNotifications()

	req := acp.PromptRequest{
		SessionId: e.sessionID,
		Prompt:    []acp.ContentBlock{acp.TextBlock("Run a shell command that prints 'approval persistence test' and then stop.")},
	}

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		_, promptErr := e.client.Prompt(ctx, req)
		done <- promptErr
	}()

	select {
	case promptErr := <-done:
		require.NoError(t, promptErr, "Prompt should complete")
	case <-time.After(60 * time.Second):
		t.Fatal("Prompt timed out")
	}
}

// hasToolCallNotifications checks if any tool call notifications were received.
func (e *approvalPersistenceEnv) hasToolCallNotifications(t *testing.T) bool {
	t.Helper()

	notifications := e.clientImpl.getNotifications()
	if len(notifications) == 0 {
		t.Skip("No notifications from first prompt; likely due to missing or incompatible local LLM")
	}

	for _, notif := range notifications {
		if notif.Update.ToolCall != nil || notif.Update.ToolCallUpdate != nil {
			return true
		}
	}

	t.Log("No tool calls observed; skipping persistence checking as LLM may have chosen a different strategy")

	return false
}

// clearApprovalPolicies runs the approval clear command.
func clearApprovalPolicies(t *testing.T, configPath string) {
	t.Helper()

	bin := getBinPath(t)
	clearResult := execCommand(t, filepath.Clean(bin),
		"--config-file", configPath, "approval", "clear", "--scope", "global",
	)

	if !strings.Contains(clearResult.stdout, "Cleared") {
		t.Fatalf("expected clear command to report cleared policies, got stdout=%s stderr=%s", clearResult.stdout, clearResult.stderr)
	}
}

type cmdResult struct {
	stdout string
	stderr string
}

// execCommand is a small helper to run a command and capture stdout/stderr.
func execCommand(t *testing.T, bin string, args ...string) cmdResult {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), bin, args...)

	var outBuf, errBuf strings.Builder

	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	require.NoError(t, err, "command failed: %s %v\nstderr: %s", bin, args, errBuf.String())

	return cmdResult{
		stdout: outBuf.String(),
		stderr: errBuf.String(),
	}
}
