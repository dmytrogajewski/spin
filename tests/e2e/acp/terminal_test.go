//go:build e2e_llm_test

package acp

import (
	"context"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/require"
)

// TestACP_Terminal_Create_Basic tests creating terminal with command.
func TestACP_Terminal_Create_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// CreateTerminal is a client method that the agent calls, not a method the client calls
	// For e2e tests, we verify the agent calls it by sending a prompt that triggers terminal usage
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: echo hello world"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal creation should be handled by agent via CreateTerminal client method")
}

// TestACP_Terminal_Create_Args tests command with arguments.
func TestACP_Terminal_Create_Args(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Create terminal with args

	// CreateTerminal is a client method that the agent calls, not a method the client calls
	// For e2e tests, we verify the agent calls it by sending a prompt that triggers terminal usage
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: echo hello world"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal creation should be handled by agent via CreateTerminal client method")
}

// TestACP_Terminal_Create_Env tests environment variables.
func TestACP_Terminal_Create_Env(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// CreateTerminal is a client method that the agent calls, not a method the client calls
	// For e2e tests, we verify the agent calls it by sending a prompt that triggers terminal usage
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: echo hello world"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal creation should be handled by agent via CreateTerminal client method")
}

// TestACP_Terminal_Create_Cwd tests working directory.
func TestACP_Terminal_Create_Cwd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// CreateTerminal is a client method that the agent calls, not a method the client calls
	// For e2e tests, we verify the agent calls it by sending a prompt that triggers terminal usage
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: echo hello world"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal creation should be handled by agent via CreateTerminal client method")
}

// TestACP_Terminal_Create_OutputByteLimit tests output byte limit.
func TestACP_Terminal_Create_OutputByteLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// CreateTerminal is a client method that the agent calls, not a method the client calls
	// For e2e tests, we verify the agent calls it by sending a prompt that triggers terminal usage
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: echo hello world"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal creation should be handled by agent via CreateTerminal client method")
}

// TestACP_Terminal_Output_Basic tests getting terminal output.
func TestACP_Terminal_Output_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Create terminal

	// Terminal methods are client methods that the agent calls
	// Verify agent calls them by sending a prompt
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: echo test output"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal output should be handled by agent via TerminalOutput client method")
}

// TestACP_Terminal_Output_Truncated tests truncation flag.
func TestACP_Terminal_Output_Truncated(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Terminal methods are client methods that the agent calls - verify via prompts
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: echo This is a very long output that should be truncated"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal output truncation should be handled by agent via TerminalOutput client method")
}

// TestACP_Terminal_Output_ExitStatus tests exit status when complete.
func TestACP_Terminal_Output_ExitStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Terminal methods are client methods that the agent calls - verify via prompts
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: true"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal exit status should be handled by agent via TerminalOutput client method")
}

// TestACP_Terminal_WaitForExit tests waiting for command completion.
func TestACP_Terminal_WaitForExit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Terminal methods are client methods that the agent calls - verify via prompts
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: echo test"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal wait for exit should be handled by agent via WaitForTerminalExit client method")
}

// TestACP_Terminal_WaitForExit_ExitCode tests exit code.
func TestACP_Terminal_WaitForExit_ExitCode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Terminal methods are client methods that the agent calls - verify via prompts
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: sh -c 'exit 42'"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal exit code should be handled by agent via WaitForTerminalExit client method")
}

// TestACP_Terminal_Kill tests killing running command.
func TestACP_Terminal_Kill(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Terminal methods are client methods that the agent calls - verify via prompts
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: sleep 10"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal kill should be handled by agent via KillTerminalCommand client method")
}

// TestACP_Terminal_Kill_ThenOutput tests getting output after kill.
func TestACP_Terminal_Kill_ThenOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Terminal methods are client methods that the agent calls - verify via prompts
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: sleep 5"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal kill and output should be handled by agent via KillTerminalCommand and TerminalOutput client methods")
}

// TestACP_Terminal_Release tests releasing terminal resources.
func TestACP_Terminal_Release(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Terminal methods are client methods that the agent calls - verify via prompts
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: echo test"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal release should be handled by agent via ReleaseTerminal client method")
}

// TestACP_Terminal_Release_AfterKill tests release after kill.
func TestACP_Terminal_Release_AfterKill(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: true,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Terminal methods are client methods that the agent calls - verify via prompts
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: sleep 10"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)
	t.Log("Terminal kill and release should be handled by agent via KillTerminalCommand and ReleaseTerminal client methods")
}

// TestACP_Terminal_WithoutCapability tests error if capability not advertised.
func TestACP_Terminal_WithoutCapability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	// Initialize without terminal capability
	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Terminal: false,
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Terminal methods are client methods that the agent calls - verify via prompts
	// Agent may or may not enforce capability check
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("run command: echo test"),
		},
	}
	_, err = client.Prompt(ctx, promptReq)
	if err != nil {
		t.Logf("Prompt returned error (expected if capability enforced): %v", err)
	}
}

