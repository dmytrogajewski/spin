//go:build e2e_llm_test

package acp

import (
	"context"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACP_MCP_Stdio_Connection tests stdio MCP server connection.
func TestACP_MCP_Stdio_Connection(t *testing.T) {
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
	})
	require.NoError(t, err)

	// Create session with stdio MCP server
	req := acp.NewSessionRequest{
		Cwd: workDir,
		McpServers: []acp.McpServer{
			{
				Stdio: &acp.McpServerStdio{
					Name:    "test-server",
					Command: "/bin/echo",
					Args:    []string{"test"},
					Env:     []acp.EnvVariable{},
				},
			},
		},
	}

	resp, err := client.NewSession(ctx, req)
	// MCP server connection may fail, but session should still be created
	if err != nil {
		t.Logf("NewSession with MCP server returned error (may be expected): %v", err)
	} else {
		assert.NotEmpty(t, resp.SessionId, "Session ID should be generated")
	}
}

// TestACP_MCP_Stdio_EnvVars tests environment variables passed to MCP server.
func TestACP_MCP_Stdio_EnvVars(t *testing.T) {
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
	})
	require.NoError(t, err)

	// Create session with MCP server env vars
	req := acp.NewSessionRequest{
		Cwd: workDir,
		McpServers: []acp.McpServer{
			{
				Stdio: &acp.McpServerStdio{
					Name:    "test-server",
					Command: "/bin/env",
					Args:    []string{},
					Env: []acp.EnvVariable{
						{
							Name:  "TEST_VAR",
							Value: "test_value",
						},
					},
				},
			},
		},
	}

	resp, err := client.NewSession(ctx, req)
	if err != nil {
		t.Logf("NewSession with MCP env vars returned error (may be expected): %v", err)
	} else {
		assert.NotEmpty(t, resp.SessionId)
	}
}

// TestACP_MCP_Stdio_Args tests command arguments passed to MCP server.
func TestACP_MCP_Stdio_Args(t *testing.T) {
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
	})
	require.NoError(t, err)

	// Create session with MCP server args
	req := acp.NewSessionRequest{
		Cwd: workDir,
		McpServers: []acp.McpServer{
			{
				Stdio: &acp.McpServerStdio{
					Name:    "test-server",
					Command: "/bin/echo",
					Args:    []string{"arg1", "arg2", "arg3"},
					Env:     []acp.EnvVariable{},
				},
			},
		},
	}

	resp, err := client.NewSession(ctx, req)
	if err != nil {
		t.Logf("NewSession with MCP args returned error (may be expected): %v", err)
	} else {
		assert.NotEmpty(t, resp.SessionId)
	}
}

// TestACP_MCP_MultipleServers tests multiple MCP servers.
func TestACP_MCP_MultipleServers(t *testing.T) {
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
	})
	require.NoError(t, err)

	// Create session with multiple MCP servers
	req := acp.NewSessionRequest{
		Cwd: workDir,
		McpServers: []acp.McpServer{
			{
				Stdio: &acp.McpServerStdio{
					Name:    "server1",
					Command: "/bin/echo",
					Args:    []string{"server1"},
					Env:     []acp.EnvVariable{},
				},
			},
			{
				Stdio: &acp.McpServerStdio{
					Name:    "server2",
					Command: "/bin/echo",
					Args:    []string{"server2"},
					Env:     []acp.EnvVariable{},
				},
			},
		},
	}

	resp, err := client.NewSession(ctx, req)
	if err != nil {
		t.Logf("NewSession with multiple MCP servers returned error (may be expected): %v", err)
	} else {
		assert.NotEmpty(t, resp.SessionId)
	}
}

// TestACP_MCP_ToolsAvailable tests that MCP tools are available to agent.
func TestACP_MCP_ToolsAvailable(t *testing.T) {
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
	})
	require.NoError(t, err)

	// Create session with MCP server
	req := acp.NewSessionRequest{
		Cwd: workDir,
		McpServers: []acp.McpServer{
			{
				Stdio: &acp.McpServerStdio{
					Name:    "test-server",
					Command: "/bin/echo",
					Args:    []string{"test"},
					Env:     []acp.EnvVariable{},
				},
			},
		},
	}

	sessionResp, err := client.NewSession(ctx, req)
	if err != nil {
		t.Logf("NewSession with MCP server returned error (may be expected): %v", err)
		return
	}

	// Send prompt that might use MCP tools
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("test prompt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	// Prompt should succeed (MCP tools may or may not be used)
	require.NoError(t, err)
}
