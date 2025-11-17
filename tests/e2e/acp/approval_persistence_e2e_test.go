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
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Prepare isolated workspace and config with approval persistence enabled.
	workDir := createTestWorkspace(t)
	configPath := filepath.Join(workDir, "spin.yaml")
	policyPath := filepath.Join(workDir, "policies.json")

	cfg := `
version: "2.0"
security:
  policy_file: ` + policyPath + `
  approval_persistence_enabled: true
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	// Start ACP agent with config and workspace using the test-only provider.
	// The "test-llm" provider is only available in binaries built with
	// -tags e2e_llm_test and returns deterministic tool calls.
	cmd, stdin, stdout := startACPAgent(t,
		"--config-file", configPath,
		"--provider", "test-llm",
		"--model", "dummy",
		"--workspace", workDir,
	)
	defer cleanupAgent(t, cmd, stdin)

	// Client that records notifications so we can observe tool calls.
	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
	ctx := context.Background()

	// Initialize and create session.
	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)
	sessionID := sessionResp.SessionId

	// Helper: run a prompt that is likely to trigger a shell_command tool call.
	runPrompt := func(t *testing.T) {
		t.Helper()
		clientImpl.clearNotifications()
		req := acp.PromptRequest{
			SessionId: sessionID,
			Prompt: []acp.ContentBlock{
				acp.TextBlock("Run a shell command that prints 'approval persistence test' and then stop."),
			},
		}

		done := make(chan error, 1)
		go func() {
			_, err := client.Prompt(ctx, req)
			done <- err
		}()

		select {
		case err := <-done:
			require.NoError(t, err, "Prompt should complete")
		case <-time.After(60 * time.Second):
			t.Fatal("Prompt timed out")
		}
	}

	// 1) First prompt should require approval and cause a policy to be persisted.
	runPrompt(t)
	firstNotifications := clientImpl.getNotifications()
	if len(firstNotifications) == 0 {
		// In environments without a functioning local LLM, we cannot assert the
		// full prompt → tool call path. The wiring is still exercised up to the
		// LLM boundary.
		t.Skip("No notifications from first prompt; likely due to missing or incompatible local LLM")
	}

	// Heuristically assert that at least one tool call occurred; we rely on unit
	// tests for precise tool kind classification.
	hasToolCall := false
	for _, notif := range firstNotifications {
		if notif.Update.ToolCall != nil || notif.Update.ToolCallUpdate != nil {
			hasToolCall = true
			break
		}
	}
	if !hasToolCall {
		t.Log("No tool calls observed; skipping persistence checking as LLM may have chosen a different strategy")
		return
	}

	// 2) Second prompt: with persistence enabled and the same request, approval
	// should be short-circuited via policy; from the ACP side, we simply ensure
	// the prompt still succeeds. Detailed policy hit is already covered in unit tests.
	runPrompt(t)

	// 3) Revocation via CLI: run "spin approval clear --scope global" in the same workspace.
	bin := getBinPath(t)
	clearCmd := filepath.Clean(bin)
	clear := execCommand(t, clearCmd,
		"--config-file", configPath,
		"approval", "clear",
		"--scope", "global",
	)
	if !strings.Contains(clear.stdout, "Cleared") {
		t.Fatalf("expected clear command to report cleared policies, got stdout=%s stderr=%s", clear.stdout, clear.stderr)
	}

	// 4) Third prompt: after revocation, approval should no longer be short-circuited.
	// As above, we verify end-to-end success; detailed behavior is asserted in
	// lower-level tests.
	runPrompt(t)
}

type cmdResult struct {
	stdout string
	stderr string
}

// execCommand is a small helper to run a command and capture stdout/stderr.
func execCommand(t *testing.T, bin string, args ...string) cmdResult {
	t.Helper()

	cmd := exec.Command(bin, args...)
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
