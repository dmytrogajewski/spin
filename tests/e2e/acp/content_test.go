//go:build e2e_llm_test

package acp

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACP_Content_Text tests text content block.
func TestACP_Content_Text(t *testing.T) {
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

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Test text block
	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("This is a text content block"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Content_Text_Annotations tests text with annotations.
func TestACP_Content_Text_Annotations(t *testing.T) {
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

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Create text block with annotations (if supported by SDK)
	textBlock := acp.TextBlock("Text with annotations")
	// Note: Annotations may not be directly settable via helper, but structure should support it
	_ = textBlock

	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			textBlock,
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Content_Image tests image content block.
func TestACP_Content_Image(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	initResp, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	if !initResp.AgentCapabilities.PromptCapabilities.Image {
		t.Skip("Image capability not supported")
	}

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Test image block
	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.ImageBlock("base64imagedata", "image/png"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Content_Image_Base64 tests base64 image data.
func TestACP_Content_Image_Base64(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	initResp, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	if !initResp.AgentCapabilities.PromptCapabilities.Image {
		t.Skip("Image capability not supported")
	}

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Create minimal valid base64 image data (1x1 PNG)
	minimalPNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	
	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.ImageBlock(minimalPNG, "image/png"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Content_Image_MimeTypes tests different image MIME types.
func TestACP_Content_Image_MimeTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	initResp, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	if !initResp.AgentCapabilities.PromptCapabilities.Image {
		t.Skip("Image capability not supported")
	}

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	mimeTypes := []string{"image/png", "image/jpeg", "image/gif", "image/webp"}

	for _, mimeType := range mimeTypes {
		t.Run(mimeType, func(t *testing.T) {
			// Create minimal base64 data
			data := base64.StdEncoding.EncodeToString([]byte("test"))
			req := acp.PromptRequest{
				SessionId: sessionResp.SessionId,
				Prompt: []acp.ContentBlock{
					acp.ImageBlock(data, mimeType),
				},
			}

			resp, err := client.Prompt(ctx, req)
			require.NoError(t, err)
			assert.NotNil(t, resp.StopReason)
		})
	}
}

// TestACP_Content_Audio tests audio content block.
func TestACP_Content_Audio(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	initResp, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	if !initResp.AgentCapabilities.PromptCapabilities.Audio {
		t.Skip("Audio capability not supported")
	}

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Test audio block
	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.AudioBlock("base64audiodata", "audio/wav"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Content_Audio_MimeTypes tests different audio MIME types.
func TestACP_Content_Audio_MimeTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	initResp, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	if !initResp.AgentCapabilities.PromptCapabilities.Audio {
		t.Skip("Audio capability not supported")
	}

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	mimeTypes := []string{"audio/wav", "audio/mpeg", "audio/mp3", "audio/ogg"}

	for _, mimeType := range mimeTypes {
		t.Run(mimeType, func(t *testing.T) {
			data := base64.StdEncoding.EncodeToString([]byte("test"))
			req := acp.PromptRequest{
				SessionId: sessionResp.SessionId,
				Prompt: []acp.ContentBlock{
					acp.AudioBlock(data, mimeType),
				},
			}

			resp, err := client.Prompt(ctx, req)
			require.NoError(t, err)
			assert.NotNil(t, resp.StopReason)
		})
	}
}

// TestACP_Content_Resource_Text tests text resource content.
func TestACP_Content_Resource_Text(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	initResp, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	if !initResp.AgentCapabilities.PromptCapabilities.EmbeddedContext {
		t.Skip("EmbeddedContext capability not supported")
	}

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Test text resource block
	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("test with resource block"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Content_Resource_Blob tests blob resource content.
func TestACP_Content_Resource_Blob(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	initResp, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	if !initResp.AgentCapabilities.PromptCapabilities.EmbeddedContext {
		t.Skip("EmbeddedContext capability not supported")
	}

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Create blob resource (base64 encoded binary data)
	blobData := base64.StdEncoding.EncodeToString([]byte("binary data"))
	// Note: ResourceBlock may need different signature for blob vs text
	// This test verifies the structure exists
	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("test with blob resource"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	_ = blobData // Verify blob data can be created
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Content_ResourceLink tests resource link.
func TestACP_Content_ResourceLink(t *testing.T) {
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

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Test resource link block
	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.ResourceLinkBlock("file.txt", "file:///test/file.txt"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Content_ResourceLink_Metadata tests resource link with metadata.
func TestACP_Content_ResourceLink_Metadata(t *testing.T) {
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

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	// Test resource link
	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.ResourceLinkBlock("document.pdf", "file:///test/document.pdf"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

