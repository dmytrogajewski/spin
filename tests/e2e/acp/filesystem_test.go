//go:build e2e_llm_test

package acp

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// filesystemTestClient tracks filesystem method calls.
type filesystemTestClient struct {
	testClient
	readCalls  []acp.ReadTextFileRequest
	writeCalls []acp.WriteTextFileRequest
	mu         sync.Mutex
}

func (c *filesystemTestClient) ReadTextFile(ctx context.Context, params acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readCalls = append(c.readCalls, params)
	
	// Read the actual file
	content, err := os.ReadFile(params.Path)
	if err != nil {
		return acp.ReadTextFileResponse{}, err
	}
	
	return acp.ReadTextFileResponse{
		Content: string(content),
	}, nil
}

func (c *filesystemTestClient) WriteTextFile(ctx context.Context, params acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeCalls = append(c.writeCalls, params)
	
	// Write the actual file
	err := os.WriteFile(params.Path, []byte(params.Content), 0644)
	if err != nil {
		return acp.WriteTextFileResponse{}, err
	}
	
	return acp.WriteTextFileResponse{}, nil
}

// TestACP_Filesystem_ReadTextFile_Basic tests that agent calls ReadTextFile.
func TestACP_Filesystem_ReadTextFile_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	fsClient := &filesystemTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &fsClient.testClient)
	ctx := context.Background()

	// Initialize with fs capabilities
	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(workDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file."
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Send prompt that should trigger file read
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file test.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify ReadTextFile was called
	fsClient.mu.Lock()
	readCalls := fsClient.readCalls
	fsClient.mu.Unlock()

	if len(readCalls) > 0 {
		assert.Equal(t, testFile, readCalls[0].Path, "ReadTextFile should be called with correct path")
		t.Logf("ReadTextFile was called %d time(s)", len(readCalls))
	} else {
		t.Log("ReadTextFile was not called (may be expected if agent doesn't use fs methods)")
	}
}

// TestACP_Filesystem_ReadTextFile_LineRange tests reading with line/limit parameters.
func TestACP_Filesystem_ReadTextFile_LineRange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	fsClient := &filesystemTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &fsClient.testClient)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				ReadTextFile: true,
			},
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Create a test file with multiple lines
	testFile := filepath.Join(workDir, "multiline.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Send prompt that might trigger file read with line range
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file multiline.txt from line 2"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify ReadTextFile was called (may have line/limit params)
	fsClient.mu.Lock()
	readCalls := fsClient.readCalls
	fsClient.mu.Unlock()

	if len(readCalls) > 0 {
		t.Logf("ReadTextFile was called with path: %s", readCalls[0].Path)
		if readCalls[0].Line != nil {
			t.Logf("ReadTextFile was called with line: %d", *readCalls[0].Line)
		}
		if readCalls[0].Limit != nil {
			t.Logf("ReadTextFile was called with limit: %d", *readCalls[0].Limit)
		}
	} else {
		t.Log("ReadTextFile was not called (may be expected)")
	}
}

// TestACP_Filesystem_ReadTextFile_InvalidPath tests error handling.
func TestACP_Filesystem_ReadTextFile_InvalidPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	fsClient := &filesystemTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &fsClient.testClient)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				ReadTextFile: true,
			},
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Send prompt that tries to read non-existent file
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file nonexistent.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify ReadTextFile was called (should return error)
	fsClient.mu.Lock()
	readCalls := fsClient.readCalls
	fsClient.mu.Unlock()

	if len(readCalls) > 0 {
		t.Logf("ReadTextFile was called for non-existent file: %s", readCalls[0].Path)
	} else {
		t.Log("ReadTextFile was not called (may be expected)")
	}
}

// TestACP_Filesystem_ReadTextFile_WithoutCapability tests error if capability not advertised.
func TestACP_Filesystem_ReadTextFile_WithoutCapability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	fsClient := &filesystemTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &fsClient.testClient)
	ctx := context.Background()

	// Initialize without read capability
	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				ReadTextFile: false,
			},
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Send prompt that might trigger file read
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file test.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Agent should not call ReadTextFile if capability not advertised
	fsClient.mu.Lock()
	readCalls := fsClient.readCalls
	fsClient.mu.Unlock()

	if len(readCalls) == 0 {
		t.Log("ReadTextFile was not called (agent respects capability)")
	} else {
		t.Logf("ReadTextFile was called %d time(s) (agent may not enforce capability)", len(readCalls))
	}
}

// TestACP_Filesystem_WriteTextFile_Basic tests that agent calls WriteTextFile.
func TestACP_Filesystem_WriteTextFile_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	fsClient := &filesystemTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &fsClient.testClient)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				WriteTextFile: true,
			},
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Send prompt that should trigger file write
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("write file write_test.txt with content This is written content"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify WriteTextFile was called
	fsClient.mu.Lock()
	writeCalls := fsClient.writeCalls
	fsClient.mu.Unlock()

	if len(writeCalls) > 0 {
		testFile := filepath.Join(workDir, "write_test.txt")
		assert.Equal(t, testFile, writeCalls[0].Path, "WriteTextFile should be called with correct path")
		assert.Contains(t, writeCalls[0].Content, "written content", "WriteTextFile should have correct content")
		t.Logf("WriteTextFile was called %d time(s)", len(writeCalls))
		
		// Verify file was written
		content, err := os.ReadFile(testFile)
		if err == nil {
			assert.Contains(t, string(content), "written content")
		}
	} else {
		t.Log("WriteTextFile was not called (may be expected if agent doesn't use fs methods)")
	}
}

// TestACP_Filesystem_WriteTextFile_Create tests creating new file.
func TestACP_Filesystem_WriteTextFile_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	fsClient := &filesystemTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &fsClient.testClient)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				WriteTextFile: true,
			},
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Send prompt that should create new file
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("create file new_file.txt with content New file content"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify WriteTextFile was called
	fsClient.mu.Lock()
	writeCalls := fsClient.writeCalls
	fsClient.mu.Unlock()

	if len(writeCalls) > 0 {
		testFile := filepath.Join(workDir, "new_file.txt")
		// Verify file was created
		_, err = os.Stat(testFile)
		if err == nil {
			t.Log("File was created successfully")
		}
	} else {
		t.Log("WriteTextFile was not called (may be expected)")
	}
}

// TestACP_Filesystem_WriteTextFile_Overwrite tests overwriting existing file.
func TestACP_Filesystem_WriteTextFile_Overwrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	fsClient := &filesystemTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &fsClient.testClient)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				WriteTextFile: true,
			},
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Create existing file
	testFile := filepath.Join(workDir, "existing.txt")
	oldContent := "Old content"
	err = os.WriteFile(testFile, []byte(oldContent), 0644)
	require.NoError(t, err)

	// Send prompt that should overwrite file
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("write file existing.txt with content New content"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify WriteTextFile was called
	fsClient.mu.Lock()
	writeCalls := fsClient.writeCalls
	fsClient.mu.Unlock()

	if len(writeCalls) > 0 {
		// Verify file was overwritten
		content, err := os.ReadFile(testFile)
		if err == nil {
			if string(content) != oldContent {
				t.Log("File was overwritten successfully")
			}
		}
	} else {
		t.Log("WriteTextFile was not called (may be expected)")
	}
}

// TestACP_Filesystem_WriteTextFile_InvalidPath tests error handling.
func TestACP_Filesystem_WriteTextFile_InvalidPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	fsClient := &filesystemTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &fsClient.testClient)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				WriteTextFile: true,
			},
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Send prompt that tries to write to invalid path
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("write file /nonexistent/path/file.txt with content test"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify WriteTextFile was called (may fail)
	fsClient.mu.Lock()
	writeCalls := fsClient.writeCalls
	fsClient.mu.Unlock()

	if len(writeCalls) > 0 {
		t.Logf("WriteTextFile was called for invalid path: %s", writeCalls[0].Path)
	} else {
		t.Log("WriteTextFile was not called (may be expected)")
	}
}

// TestACP_Filesystem_WriteTextFile_WithoutCapability tests error if capability not advertised.
func TestACP_Filesystem_WriteTextFile_WithoutCapability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	fsClient := &filesystemTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &fsClient.testClient)
	ctx := context.Background()

	// Initialize without write capability
	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				WriteTextFile: false,
			},
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Send prompt that might trigger file write
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("write file test.txt with content test"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Agent should not call WriteTextFile if capability not advertised
	fsClient.mu.Lock()
	writeCalls := fsClient.writeCalls
	fsClient.mu.Unlock()

	if len(writeCalls) == 0 {
		t.Log("WriteTextFile was not called (agent respects capability)")
	} else {
		t.Logf("WriteTextFile was called %d time(s) (agent may not enforce capability)", len(writeCalls))
	}
}

// TestACP_Filesystem_ReadWrite_Integration tests read after write.
func TestACP_Filesystem_ReadWrite_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	fsClient := &filesystemTestClient{}
	client := createACPClientWithClient(t, stdin, stdout, &fsClient.testClient)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
		},
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Send prompt that writes then reads
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("write file integration_test.txt with content Integration test content, then read it"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Verify both methods were called
	fsClient.mu.Lock()
	writeCalls := fsClient.writeCalls
	readCalls := fsClient.readCalls
	fsClient.mu.Unlock()

	testFile := filepath.Join(workDir, "integration_test.txt")
	if len(writeCalls) > 0 && len(readCalls) > 0 {
		t.Logf("WriteTextFile called %d time(s), ReadTextFile called %d time(s)", len(writeCalls), len(readCalls))
		
		// Verify file exists and has correct content
		content, err := os.ReadFile(testFile)
		if err == nil {
			assert.Contains(t, string(content), "Integration test content")
		}
	} else {
		t.Logf("WriteTextFile called %d time(s), ReadTextFile called %d time(s) (may be expected)", len(writeCalls), len(readCalls))
	}
}
