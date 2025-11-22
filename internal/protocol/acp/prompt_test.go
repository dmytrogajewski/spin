package acp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/planning"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpinACPAgent_Prompt_InvalidSession tests Prompt with invalid session ID.
func TestSpinACPAgent_Prompt_InvalidSession(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	req := acp.PromptRequest{
		SessionId: acp.SessionId("non-existent-session"),
		Prompt:    []acp.ContentBlock{acp.TextBlock("test")},
	}

	_, err = acpAgent.Prompt(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

// TestSpinACPAgent_Prompt_EmptyPrompt tests Prompt with empty prompt blocks.
func TestSpinACPAgent_Prompt_EmptyPrompt(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create a session first
	sessionReq := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt:    []acp.ContentBlock{}, // Empty prompt
	}

	_, err = acpAgent.Prompt(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt cannot be empty")
}

// TestSpinACPAgent_Prompt_Success tests successful prompt execution.
func TestSpinACPAgent_Prompt_Success(t *testing.T) {
	agentInstance := createTestAgent(t)
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create a session first
	sessionReq := acp.NewSessionRequest{
		Cwd: t.TempDir(),
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("test prompt")},
	}

	resp, err := acpAgent.Prompt(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, acp.StopReasonEndTurn, resp.StopReason)
}

// TestAgentEmitterStreams ensures the underlying agent emits content delta events.
func TestAgentEmitterStreams(t *testing.T) {
	agentInstance, emitter := createTestAgentWithEmitter(t)

	subID, ch, err := emitter.Subscribe()
	require.NoError(t, err)
	defer emitter.Unsubscribe(subID)

	req := &agent.AgentRequest{
		Input:   "hello emitter",
		Task:    task.DefaultTask(),
		History: []message.Message{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_, _ = agentInstance.Execute(ctx, req)
		close(done)
	}()

	require.Eventually(t, func() bool {
		select {
		case event := <-ch:
			if event.Type == events.EventContentDelta {
				return true
			}
		default:
		}
		return false
	}, time.Second, 10*time.Millisecond, "expected EventContentDelta from agent emitter")
	<-done
}

// TestEmitterManualSubscribe ensures subscribing before execution captures events.
func TestEmitterManualSubscribe(t *testing.T) {
	agentInstance, emitter := createTestAgentWithEmitter(t)

	subID, ch, err := emitter.Subscribe()
	require.NoError(t, err)
	defer emitter.Unsubscribe(subID)

	req := &agent.AgentRequest{
		Input:   "manual subscribe",
		Task:    task.DefaultTask(),
		History: []message.Message{},
	}

	go func() {
		_, _ = agentInstance.Execute(context.Background(), req)
	}()

	require.Eventually(t, func() bool {
		select {
		case event := <-ch:
			return event.Type == events.EventContentDelta
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond, "expected EventContentDelta event")
}

// createTestAgent creates a test agent with mock LLM provider.
func createTestAgent(t *testing.T) *agent.Agent {
	t.Helper()
	agentInstance, _ := createTestAgentWithEmitter(t)
	return agentInstance
}

// createTestAgentWithEmitter returns a test agent along with its emitter.
func createTestAgentWithEmitter(t *testing.T) (*agent.Agent, *events.EventEmitter) {
	t.Helper()

	mockProvider := llm.NewMockProvider("test response")
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewSecurityService(validator, approvalService)
	detectionService := detection.NewDetectionService(cycle.NewDetector(cycle.Config{Enabled: false}), nil)
	toolRuntime := agent.NewToolRuntime(agent.ToolRuntimeConfig{
		Registry:        tools.NewRegistry(),
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         t.TempDir(),
	})
	planningService := planning.NewPlanningService(mockProvider)

	agentInstance, err := agent.NewAgent(
		mockProvider,
		securityService,
		detectionService,
		toolRuntime,
		planningService,
		&agent.Environment{WorkDir: t.TempDir()},
		emitter,
	)
	require.NoError(t, err)
	return agentInstance, emitter
}

// TestSpinACPAgent_Prompt_SendsNotifications ensures prompt execution emits agent message chunks.
func TestSpinACPAgent_Prompt_SendsNotifications(t *testing.T) {
	agentInstance, emitter := createTestAgentWithEmitter(t)
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	// Create a session to obtain session ID
	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd: t.TempDir(),
	})
	require.NoError(t, err)

	// Execute prompt
	_, err = acpAgent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	})
	require.NoError(t, err)

	// Ensure at least one agent_message_chunk notification is sent
	require.Eventually(t, func() bool {
		for _, n := range mockConn.GetNotifications() {
			if chunk := n.Update.AgentMessageChunk; chunk != nil {
				if chunk.Content.Text != nil && strings.TrimSpace(chunk.Content.Text.Text) != "" {
					return true
				}
			}
		}
		return false
	}, time.Second, 10*time.Millisecond, "expected agent message chunk notification")
}

// TestSpinACPAgent_EndToEndNotifications verifies notifications over JSON-RPC connection.
func TestSpinACPAgent_EndToEndNotifications(t *testing.T) {
	agentInstance, emitter := createTestAgentWithEmitter(t)
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	agentReader, clientWriter := io.Pipe()
	clientReader, agentWriter := io.Pipe()

	agentConn := acp.NewAgentSideConnection(acpAgent, agentWriter, agentReader)
	acpAgent.SetConnection(agentConn)

	testClient := &stubClient{}
	clientConn := acp.NewClientSideConnection(testClient, clientWriter, clientReader)

	ctx := context.Background()
	_, err = clientConn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersion(1),
		ClientInfo: &acp.Implementation{
			Name:    "test-client",
			Version: "0.1.0",
		},
	})
	require.NoError(t, err)

	sessionResp, err := clientConn.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        t.TempDir(),
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	_, err = clientConn.Prompt(ctx, acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello over rpc")},
	})
	require.NoError(t, err)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		for _, n := range testClient.Notifications() {
			if chunk := n.Update.AgentMessageChunk; chunk != nil {
				if chunk.Content.Text != nil && strings.TrimSpace(chunk.Content.Text.Text) != "" {
					return
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	notifs := testClient.Notifications()
	t.Fatalf("expected agent message chunk notification via connection, got %d notifications: %+v", len(notifs), notifs)
}

// stubClient captures session notifications for assertions.
type stubClient struct {
	mu            sync.Mutex
	notifications []acp.SessionNotification
}

func (c *stubClient) Notifications() []acp.SessionNotification {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]acp.SessionNotification, len(c.notifications))
	copy(result, c.notifications)
	return result
}

func (c *stubClient) ReadTextFile(ctx context.Context, params acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, fmt.Errorf("fs.readTextFile not supported in stub client")
}

func (c *stubClient) WriteTextFile(ctx context.Context, params acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, fmt.Errorf("fs.writeTextFile not supported in stub client")
}

func (c *stubClient) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	// Always cancel to keep tests deterministic.
	return acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeCancelled(),
	}, nil
}

func (c *stubClient) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notifications = append(c.notifications, params)
	return nil
}

func (c *stubClient) CreateTerminal(ctx context.Context, params acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, fmt.Errorf("terminal capability not enabled in stub client")
}

func (c *stubClient) KillTerminalCommand(ctx context.Context, params acp.KillTerminalCommandRequest) (acp.KillTerminalCommandResponse, error) {
	return acp.KillTerminalCommandResponse{}, fmt.Errorf("terminal capability not enabled in stub client")
}

func (c *stubClient) TerminalOutput(ctx context.Context, params acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, fmt.Errorf("terminal capability not enabled in stub client")
}

func (c *stubClient) ReleaseTerminal(ctx context.Context, params acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, fmt.Errorf("terminal capability not enabled in stub client")
}

func (c *stubClient) WaitForTerminalExit(ctx context.Context, params acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, fmt.Errorf("terminal capability not enabled in stub client")
}

// TestSpinACPAgent_Prompt_ContentBlockConversion tests content block conversion.
func TestSpinACPAgent_Prompt_ContentBlockConversion(t *testing.T) {
	agentInstance := createTestAgent(t)
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create a session
	sessionReq := acp.NewSessionRequest{
		Cwd: t.TempDir(),
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	tests := []struct {
		name    string
		blocks  []acp.ContentBlock
		wantErr bool
	}{
		{
			name:    "text block",
			blocks:  []acp.ContentBlock{acp.TextBlock("hello")},
			wantErr: false,
		},
		{
			name: "resource link",
			blocks: []acp.ContentBlock{
				{
					ResourceLink: &acp.ContentBlockResourceLink{
						Name: "test.txt",
						Uri:  "file:///tmp/test.txt",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple text blocks",
			blocks: []acp.ContentBlock{
				acp.TextBlock("first"),
				acp.TextBlock("second"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := acp.PromptRequest{
				SessionId: sessionResp.SessionId,
				Prompt:    tt.blocks,
			}

			resp, err := acpAgent.Prompt(context.Background(), req)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

// TestConvertACPContentBlocksToMessages tests content block conversion directly.
func TestConvertACPContentBlocksToMessages(t *testing.T) {
	tests := []struct {
		name    string
		blocks  []acp.ContentBlock
		wantErr bool
	}{
		{
			name:    "text block",
			blocks:  []acp.ContentBlock{acp.TextBlock("test")},
			wantErr: false,
		},
		{
			name:    "empty blocks",
			blocks:  []acp.ContentBlock{},
			wantErr: true,
		},
		{
			name: "resource link",
			blocks: []acp.ContentBlock{
				{
					ResourceLink: &acp.ContentBlockResourceLink{
						Name: "file.txt",
						Uri:  "file:///tmp/file.txt",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "embedded resource with text",
			blocks: []acp.ContentBlock{
				{
					Resource: &acp.ContentBlockResource{
						Resource: acp.EmbeddedResourceResource{
							TextResourceContents: &acp.TextResourceContents{
								Text: "embedded text",
								Uri:  "file:///tmp/test.txt",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "embedded resource with blob",
			blocks: []acp.ContentBlock{
				{
					Resource: &acp.ContentBlockResource{
						Resource: acp.EmbeddedResourceResource{
							BlobResourceContents: &acp.BlobResourceContents{
								Blob: "base64data",
								MimeType: func() *string {
									s := "image/png"
									return &s
								}(),
								Uri: "file:///tmp/image.png",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "mixed blocks",
			blocks: []acp.ContentBlock{
				acp.TextBlock("first"),
				{
					ResourceLink: &acp.ContentBlockResourceLink{
						Name: "file.txt",
						Uri:  "file:///tmp/file.txt",
					},
				},
				acp.TextBlock("second"),
			},
			wantErr: false,
		},
		{
			name: "image block",
			blocks: []acp.ContentBlock{
				acp.ImageBlock("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==", "image/png"),
			},
			wantErr: false,
		},
		{
			name: "image block with default mime type",
			blocks: []acp.ContentBlock{
				{
					Image: &acp.ContentBlockImage{
						Data: "base64imagedata",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "audio block",
			blocks: []acp.ContentBlock{
				acp.AudioBlock("UklGRiQAAABXQVZFZm10IBAAAAABAAEAQB8AAAB9AAACABAAZGF0YQAAAAA=", "audio/wav"),
			},
			wantErr: false,
		},
		{
			name: "audio block with default mime type",
			blocks: []acp.ContentBlock{
				{
					Audio: &acp.ContentBlockAudio{
						Data: "base64audiodata",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "mixed content types including image and audio",
			blocks: []acp.ContentBlock{
				acp.TextBlock("text content"),
				acp.ImageBlock("base64image", "image/jpeg"),
				acp.AudioBlock("base64audio", "audio/mpeg"),
				{
					ResourceLink: &acp.ContentBlockResourceLink{
						Name: "file.txt",
						Uri:  "file:///tmp/file.txt",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := convertACPContentBlocksToMessages(tt.blocks)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, messages)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, messages)
				// Verify all messages have user role
				for _, msg := range messages {
					assert.Equal(t, message.RoleUser, msg.Role)
					assert.NotEmpty(t, msg.Content)
				}
			}
		})
	}
}

// TestConvertACPContentBlocksToMessages_ImageAudio tests image and audio block conversion with content verification.
func TestConvertACPContentBlocksToMessages_ImageAudio(t *testing.T) {
	tests := []struct {
		name           string
		block          acp.ContentBlock
		wantContentSub string // Substring that should be in the converted content
	}{
		{
			name:           "image block with mime type",
			block:          acp.ImageBlock("base64imagedata123", "image/jpeg"),
			wantContentSub: "[Image: image/jpeg",
		},
		{
			name: "image block without mime type",
			block: acp.ContentBlock{
				Image: &acp.ContentBlockImage{
					Data: "base64imagedata456",
				},
			},
			wantContentSub: "[Image: image/png", // Default mime type
		},
		{
			name:           "audio block with mime type",
			block:          acp.AudioBlock("base64audiodata789", "audio/wav"),
			wantContentSub: "[Audio: audio/wav",
		},
		{
			name: "audio block without mime type",
			block: acp.ContentBlock{
				Audio: &acp.ContentBlockAudio{
					Data: "base64audiodata012",
				},
			},
			wantContentSub: "[Audio: audio/mpeg", // Default mime type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := convertACPContentBlocksToMessages([]acp.ContentBlock{tt.block})
			require.NoError(t, err)
			require.Len(t, messages, 1)

			msg := messages[0]
			assert.Equal(t, message.RoleUser, msg.Role)
			assert.Contains(t, msg.Content, tt.wantContentSub)
			// Verify it includes byte count
			assert.Contains(t, msg.Content, "bytes]")
		})
	}
}

// TestConvertACPContentBlocksToMessages_EnhancedResources tests enhanced resource block handling with names.
func TestConvertACPContentBlocksToMessages_EnhancedResources(t *testing.T) {
	tests := []struct {
		name           string
		block          acp.ContentBlock
		wantContentSub string
	}{
		{
			name: "text resource with URI",
			block: acp.ContentBlock{
				Resource: &acp.ContentBlockResource{
					Resource: acp.EmbeddedResourceResource{
						TextResourceContents: &acp.TextResourceContents{
							Uri:  "file:///tmp/config.yaml",
							Text: "key: value",
						},
					},
				},
			},
			wantContentSub: "[Resource: config.yaml]",
		},
		{
			name: "blob resource with URI and mime type",
			block: acp.ContentBlock{
				Resource: &acp.ContentBlockResource{
					Resource: acp.EmbeddedResourceResource{
						BlobResourceContents: &acp.BlobResourceContents{
							Uri: "file:///tmp/image.png",
							MimeType: func() *string {
								s := "image/png"
								return &s
							}(),
							Blob: "base64data",
						},
					},
				},
			},
			wantContentSub: "[Resource: image.png]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := convertACPContentBlocksToMessages([]acp.ContentBlock{tt.block})
			require.NoError(t, err)
			require.Len(t, messages, 1)

			msg := messages[0]
			assert.Equal(t, message.RoleUser, msg.Role)
			assert.Contains(t, msg.Content, tt.wantContentSub)
		})
	}
}

// TestMapStopReason tests stop reason mapping.
func TestMapStopReason(t *testing.T) {
	tests := []struct {
		name         string
		finishReason string
		want         acp.StopReason
	}{
		// Spin agent finish reasons
		{"timeout", "timeout", acp.StopReasonCancelled},
		{"error", "error", acp.StopReasonEndTurn},
		{"empty_response", "empty_response", acp.StopReasonEndTurn},
		{"max_tokens", "max_tokens", acp.StopReasonMaxTokens},
		{"max_turns", "max_turns", acp.StopReasonMaxTurnRequests},
		{"cancelled", "cancelled", acp.StopReasonCancelled},
		{"refusal", "refusal", acp.StopReasonRefusal},
		// OpenAI finish reasons
		{"stop", "stop", acp.StopReasonEndTurn},
		{"length", "length", acp.StopReasonMaxTokens},
		{"tool_calls", "tool_calls", acp.StopReasonEndTurn},
		{"content_filter", "content_filter", acp.StopReasonRefusal},
		{"function_call", "function_call", acp.StopReasonEndTurn},
		// Default/unknown
		{"unknown", "unknown", acp.StopReasonEndTurn},
		{"empty", "", acp.StopReasonEndTurn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapStopReason(tt.finishReason)
			assert.Equal(t, tt.want, got, "finishReason: %s", tt.finishReason)
		})
	}
}

// TestExtractPathFromURI tests URI path extraction.
func TestExtractPathFromURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{"file URI", "file:///tmp/test.txt", "/tmp/test.txt"},
		{"plain path", "/tmp/test.txt", "/tmp/test.txt"},
		{"relative path", "test.txt", "test.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathFromURI(tt.uri)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestMapStopReasonFromError tests error to stop reason mapping.
func TestMapStopReasonFromError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		resp *agent.AgentResponse
		want acp.StopReason
	}{
		{
			name: "context cancelled",
			err:  context.Canceled,
			resp: nil,
			want: acp.StopReasonCancelled,
		},
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			resp: nil,
			want: acp.StopReasonCancelled,
		},
		{
			name: "error with response",
			err:  fmt.Errorf("some error"),
			resp: &agent.AgentResponse{
				FinishReason: "max_tokens",
			},
			want: acp.StopReasonMaxTokens,
		},
		{
			name: "error without response",
			err:  fmt.Errorf("some error"),
			resp: nil,
			want: acp.StopReasonEndTurn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapStopReasonFromError(tt.err, tt.resp)
			assert.Equal(t, tt.want, got)
		})
	}
}
