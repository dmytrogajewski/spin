package acp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/runtime"
	"github.com/dmytrogajewski/spin/internal/agent/sanitizer"
	"github.com/dmytrogajewski/spin/internal/commands"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/planning"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/version"
)

var (
	// ErrNilAgent is returned when agent is nil.
	ErrNilAgent = errors.New("agent cannot be nil")
	// ErrNilMCPManager is returned when MCP manager is nil.
	ErrNilMCPManager = errors.New("mcp manager cannot be nil")
	// ErrNilEmitter is returned when emitter is nil.
	ErrNilEmitter = errors.New("emitter cannot be nil")
	// ErrNotImplemented is returned for methods that are not implemented.
	ErrNotImplemented = errors.New("not implemented")
)

// SpinACPAgent implements the acp.Agent interface, adapting Spin's components
// to the Agent Client Protocol.
//
// SpinACPAgent acts as an adapter between ACP protocol requests and Spin's
// internal agent execution, session management, and tool orchestration.
type SpinACPAgent struct {
	agent           *agent.Agent
	mcpManager      *mcp.MCPManager
	emitter         *events.EventEmitter
	approvalService *security.ApprovalService // Optional approval service for permission requests
	approvalHandler *ACPApprovalHandler       // ACP-specific approval handler
	clientCaps      *acp.ClientCapabilities   // Stored after Initialize
	sessions        map[acp.SessionId]*session.Session
	sessionModes    map[acp.SessionId]acp.SessionModeId  // Current mode per session
	storage         session.Storage                      // Optional session storage for persistence
	connection      notificationSender                   // Optional connection for sending notifications
	cancels         map[acp.SessionId]context.CancelFunc // Cancel functions for in-progress prompt executions
	mu              sync.RWMutex                         // Protects sessions map, sessionModes, connection, and cancels
}

// NewSpinACPAgentWithStorage creates a new ACP agent adapter with optional session storage.
//
// The adapter requires:
//   - agent: Core agent for execution
//   - mcpManager: MCP server management
//   - emitter: Event emission for notifications
//
// Optional:
//   - storage: Session storage for persistence (if nil, LoadSession will not be available)
//
// Returns an error if any required component is nil.
// If storage is nil, session persistence features (LoadSession) will not be available.
func NewSpinACPAgentWithStorage(
	agent *agent.Agent,
	mcpManager *mcp.MCPManager,
	emitter *events.EventEmitter,
	storage session.Storage,
) (*SpinACPAgent, error) {
	if agent == nil {
		return nil, fmt.Errorf("%w", ErrNilAgent)
	}
	if mcpManager == nil {
		return nil, fmt.Errorf("%w", ErrNilMCPManager)
	}
	if emitter == nil {
		return nil, fmt.Errorf("%w", ErrNilEmitter)
	}

	return &SpinACPAgent{
		agent:           agent,
		mcpManager:      mcpManager,
		emitter:         emitter,
		approvalService: nil, // Optional - set via SetApprovalService() if needed
		storage:         storage,
		connection:      nil, // Set via SetConnection() after connection is created
		sessions:        make(map[acp.SessionId]*session.Session),
		sessionModes:    make(map[acp.SessionId]acp.SessionModeId),
		cancels:         make(map[acp.SessionId]context.CancelFunc),
	}, nil
}

// notificationSender is an interface for sending ACP session notifications and requesting permissions.
// This allows for easier testing and abstraction.
type notificationSender interface {
	SessionUpdate(ctx context.Context, notification acp.SessionNotification) error
	RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error)
}

// SetConnection sets the ACP connection for sending notifications.
// This should be called after the connection is created (e.g., in cmd/spin/acp.go).
func (a *SpinACPAgent) SetConnection(conn *acp.AgentSideConnection) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.connection = conn
}

// SetNotificationSender sets a notification sender (for testing).
// This allows using a mock connection in tests.
// This is exported for testing purposes only.
func (a *SpinACPAgent) SetNotificationSender(sender notificationSender) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.connection = sender
}

// SetApprovalService sets the approval service for permission requests.
// This should be called after the agent is created if approval service is available.
// It also creates and sets up the ACP approval handler.
func (a *SpinACPAgent) SetApprovalService(service *security.ApprovalService) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.approvalService = service
}

// SetApprovalHandler sets the ACP approval handler used for session tracking.
// The handler must be the same instance wired into the approval service so the
// active session tracking lines up with actual permission requests.
func (a *SpinACPAgent) SetApprovalHandler(handler *ACPApprovalHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.approvalHandler = handler
}

// Initialize establishes connection and negotiates capabilities.
//
// Negotiates protocol version, advertises agent capabilities based on Spin's
// features, stores client capabilities, and exchanges client/agent info.
func (a *SpinACPAgent) Initialize(ctx context.Context, req acp.InitializeRequest) (acp.InitializeResponse, error) {
	// Negotiate protocol version
	negotiatedVersion := a.negotiateProtocolVersion(req.ProtocolVersion)

	// Build agent capabilities
	agentCaps := a.buildAgentCapabilities()

	// Store client capabilities
	a.clientCaps = &req.ClientCapabilities

	// Build agent info
	agentInfo := &acp.Implementation{
		Name:    "spin",
		Version: version.ShortVersion(),
	}

	// Build response
	resp := acp.InitializeResponse{
		ProtocolVersion:   negotiatedVersion,
		AgentCapabilities: agentCaps,
		AgentInfo:         agentInfo,
		AuthMethods:       []acp.AuthMethod{}, // No auth methods initially
	}

	return resp, nil
}

// negotiateProtocolVersion negotiates the protocol version with the client.
// Currently only supports version 1. Returns version 1 if client requests
// a supported version, otherwise returns the latest supported version.
func (a *SpinACPAgent) negotiateProtocolVersion(clientVersion acp.ProtocolVersion) acp.ProtocolVersion {
	// Currently only support version 1
	if clientVersion == acp.ProtocolVersionNumber {
		return acp.ProtocolVersionNumber
	}
	// Return latest supported version (currently only version 1)
	return acp.ProtocolVersionNumber
}

// buildAgentCapabilities builds agent capabilities based on Spin's features.
func (a *SpinACPAgent) buildAgentCapabilities() acp.AgentCapabilities {
	return acp.AgentCapabilities{
		LoadSession: a.hasSessionPersistence(),
		PromptCapabilities: acp.PromptCapabilities{
			Image:           true, // Image blocks supported (converted to text description for agent processing)
			Audio:           true, // Audio blocks supported (converted to text description for agent processing)
			EmbeddedContext: true, // Embedded resources fully supported (text and blob)
		},
		McpCapabilities: acp.McpCapabilities{
			Http: false, // MCP manager currently only supports stdio
			Sse:  false, // MCP manager currently only supports stdio
		},
	}
}

// hasSessionPersistence checks if session persistence is available.
// Returns true if storage is configured and available.
func (a *SpinACPAgent) hasSessionPersistence() bool {
	return a.storage != nil
}

// NewSession creates a new session with working directory and MCP servers.
//
// Creates a session using the working directory, connects MCP servers if provided,
// and returns the session ID.
func (a *SpinACPAgent) NewSession(ctx context.Context, req acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	// Validate working directory
	if req.Cwd == "" {
		return acp.NewSessionResponse{}, fmt.Errorf("working directory is required")
	}

	// Create session
	sess := session.NewSession(req.Cwd)

	// Convert session ID to ACP format
	sessionID := acp.SessionId(sess.ID)

	// Validate and convert MCP servers synchronously (to catch conversion errors)
	// Connection happens in background to avoid blocking
	if len(req.McpServers) > 0 {
		configs := make([]mcp.MCPServerConfig, 0, len(req.McpServers))
		for _, server := range req.McpServers {
			config, err := convertACPMcpServerToSpin(server)
			if err != nil {
				return acp.NewSessionResponse{}, fmt.Errorf("invalid MCP server config: %w", err)
			}
			configs = append(configs, config)
		}

		// Store session before connecting MCP servers
		a.mu.Lock()
		a.sessions[sessionID] = sess
		a.mu.Unlock()

		// Connect servers in background (non-blocking - connection failures don't prevent session creation)
		go func() {
			for _, config := range configs {
				if err := a.mcpManager.ConnectServer(ctx, config); err != nil {
					// Log error but don't fail session creation
					// In a real implementation, we'd use a logger here
					_ = err // Error logged but session still created
				}
			}
		}()
	} else {
		// Store session
		a.mu.Lock()
		a.sessions[sessionID] = sess
		a.mu.Unlock()
	}

	// Initialize default mode for session
	defaultMode := getDefaultMode()
	a.mu.Lock()
	a.sessionModes[sessionID] = defaultMode
	a.mu.Unlock()

	// Build response with mode state
	resp := acp.NewSessionResponse{
		SessionId: sessionID,
		Modes: &acp.SessionModeState{
			AvailableModes: getAvailableModes(),
			CurrentModeId:  defaultMode,
		},
	}

	// Send available commands notification
	if err := a.sendAvailableCommandsUpdate(ctx, sessionID); err != nil {
		// Log error but don't fail session creation
		_ = err
	}

	return resp, nil
}

// convertACPMcpServerToSpin converts an ACP McpServer to Spin MCPServerConfig.
func convertACPMcpServerToSpin(acpServer acp.McpServer) (mcp.MCPServerConfig, error) {
	if acpServer.Stdio != nil {
		// Convert environment variables
		env := make(map[string]string)
		for _, envVar := range acpServer.Stdio.Env {
			env[envVar.Name] = envVar.Value
		}

		return mcp.MCPServerConfig{
			Name:    acpServer.Stdio.Name,
			Command: acpServer.Stdio.Command,
			Args:    acpServer.Stdio.Args,
			Env:     env,
		}, nil
	}

	// HTTP and SSE transports are not supported
	if acpServer.Http != nil {
		return mcp.MCPServerConfig{}, fmt.Errorf("HTTP transport is not supported")
	}

	if acpServer.Sse != nil {
		return mcp.MCPServerConfig{}, fmt.Errorf("SSE transport is not supported")
	}

	return mcp.MCPServerConfig{}, fmt.Errorf("no transport specified")
}

// Prompt processes a user prompt and executes the agent loop.
func (a *SpinACPAgent) Prompt(ctx context.Context, req acp.PromptRequest) (acp.PromptResponse, error) {
	// Validate session exists
	a.mu.RLock()
	_, exists := a.sessions[req.SessionId]
	a.mu.RUnlock()
	if !exists {
		return acp.PromptResponse{}, fmt.Errorf("session not found: %s", req.SessionId)
	}

	// Validate prompt is not empty
	if len(req.Prompt) == 0 {
		return acp.PromptResponse{}, fmt.Errorf("prompt cannot be empty")
	}

	// Create cancellable context for this prompt execution
	// This allows the Cancel method to cancel in-progress executions
	promptCtx, cancel := context.WithCancel(ctx)

	// Add session ID to context so it's available for tools (e.g. TerminalExecutor)
	promptCtx = runtime.ContextWithSessionID(promptCtx, string(req.SessionId))

	// Store cancel function so Cancel method can cancel this execution
	a.mu.Lock()
	// Cancel any existing in-progress execution for this session
	if existingCancel, exists := a.cancels[req.SessionId]; exists {
		existingCancel()
	}
	a.cancels[req.SessionId] = cancel
	a.mu.Unlock()

	// Clean up cancel function when prompt completes
	defer func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		// Remove cancel function (may have been removed by Cancel, but that's ok)
		delete(a.cancels, req.SessionId)
	}()

	// Convert ACP content blocks to Spin messages
	messages, err := convertACPContentBlocksToMessages(req.Prompt)
	if err != nil {
		return acp.PromptResponse{}, fmt.Errorf("failed to convert content blocks: %w", err)
	}

	// Extract text input from messages
	input := extractTextFromMessages(messages)

	// Check if input is a command
	if cmd, cmdArgs, isCmd := commands.ParseCommand(input); isCmd {
		// Execute command
		result, err := a.executeCommand(promptCtx, cmd, cmdArgs, req.SessionId)
		if err != nil {
			// Return error response
			return acp.PromptResponse{
				StopReason: acp.StopReasonRefusal,
			}, fmt.Errorf("command execution failed: %w", err)
		}

		// Send command output as agent message chunk notification
		a.mu.RLock()
		conn := a.connection
		a.mu.RUnlock()

		if conn != nil {
			update := acp.UpdateAgentMessageText(result)
			notification := acp.SessionNotification{
				SessionId: req.SessionId,
				Update:    update,
			}
			_ = conn.SessionUpdate(promptCtx, notification) // Log error but don't fail
		}

		// Return success response with command output
		return acp.PromptResponse{
			StopReason: acp.StopReasonEndTurn,
		}, nil
	}

	// Note: We do NOT send user_message_chunk notifications here because:
	// 1. The client already knows what they sent in the session/prompt request
	// 2. Sending it back would cause duplication (client shows both request and notification)
	// 3. user_message_chunk is only needed when replaying history in LoadSession

	// Get connection for event processing
	a.mu.RLock()
	conn := a.connection
	approvalHandler := a.approvalHandler
	a.mu.RUnlock()

	// Set active session in approval handler for this prompt execution
	if approvalHandler != nil {
		approvalHandler.SetActiveSession(req.SessionId)
		defer approvalHandler.ClearActiveSession()
	}

	// Create agent request
	agentReq := &agent.AgentRequest{
		Input:   input,
		Task:    task.DefaultTask(),
		History: []message.Message{},
	}

	// Subscribe to events for real-time notifications
	var (
		subID       string
		eventCh     <-chan events.Event
		unsubscribe func()
		eventsDone  chan struct{}
	)

	if conn != nil {
		var err error
		subID, eventCh, err = a.emitter.Subscribe()
		if err != nil {
			// Log error but continue without notifications
			_ = err
		} else {
			unsubscribe = func() {
				a.emitter.Unsubscribe(subID)
			}
			eventsDone = make(chan struct{})
			go func() {
				defer close(eventsDone)
				a.processEvents(promptCtx, req.SessionId, eventCh)
			}()
		}
	}

	defer func() {
		if unsubscribe != nil {
			unsubscribe()
		}
		if eventsDone != nil {
			<-eventsDone
		}
		cancel()
	}()

	// Execute agent with cancellable context
	agentResp, err := a.agent.Execute(promptCtx, agentReq)
	if err != nil {
		// Map error to stop reason
		stopReason := mapStopReasonFromError(err, agentResp)
		return acp.PromptResponse{
			StopReason: stopReason,
		}, nil // Return response with stop reason, not error
	}

	// Send plan notifications if a plan is available
	// Basic plan detection - checks orchestration service for plan (if accessible)
	// Full plan system integration deferred to Feature 9.2
	if conn != nil && agentResp != nil {
		if err := a.sendPlanNotifications(promptCtx, req.SessionId, agentResp); err != nil {
			// Log error but continue - plan notifications are non-critical
			_ = err
		}
	}

	// Map finish reason to ACP stop reason
	stopReason := mapStopReason(agentResp.FinishReason)

	return acp.PromptResponse{
		StopReason: stopReason,
	}, nil
}

// convertACPContentBlocksToMessages converts ACP ContentBlock[] to Spin message.Message[].
func convertACPContentBlocksToMessages(blocks []acp.ContentBlock) ([]message.Message, error) {
	var messages []message.Message

	for _, block := range blocks {
		if block.Text != nil {
			messages = append(messages, message.Message{
				Role:      message.RoleUser,
				Content:   block.Text.Text,
				Timestamp: time.Now(),
			})
		} else if block.ResourceLink != nil {
			// Extract file path from URI (basic implementation)
			uri := block.ResourceLink.Uri
			path := extractPathFromURI(uri)
			messages = append(messages, message.Message{
				Role:      message.RoleUser,
				Content:   fmt.Sprintf("File: %s", path),
				Timestamp: time.Now(),
			})
		} else if block.Resource != nil {
			// Embedded resource - extract text if available
			if block.Resource.Resource.TextResourceContents != nil {
				// Extract resource name from URI if available
				uri := block.Resource.Resource.TextResourceContents.Uri
				resourceName := extractResourceNameFromURI(uri)
				content := block.Resource.Resource.TextResourceContents.Text
				if resourceName != "" {
					content = fmt.Sprintf("[Resource: %s]\n%s", resourceName, content)
				}
				messages = append(messages, message.Message{
					Role:      message.RoleUser,
					Content:   content,
					Timestamp: time.Now(),
				})
			} else if block.Resource.Resource.BlobResourceContents != nil {
				// Blob resource - reference by MIME type
				mimeType := "unknown"
				if block.Resource.Resource.BlobResourceContents.MimeType != nil {
					mimeType = *block.Resource.Resource.BlobResourceContents.MimeType
				}
				// Extract resource name from URI if available
				uri := block.Resource.Resource.BlobResourceContents.Uri
				resourceName := extractResourceNameFromURI(uri)
				content := fmt.Sprintf("Resource (blob, %s)", mimeType)
				if resourceName != "" {
					content = fmt.Sprintf("[Resource: %s] %s", resourceName, content)
				}
				messages = append(messages, message.Message{
					Role:      message.RoleUser,
					Content:   content,
					Timestamp: time.Now(),
				})
			}
		} else if block.Image != nil {
			// Image block - convert to descriptive text since message.Message only supports text
			// MimeType is a string, not a pointer
			mimeType := block.Image.MimeType
			if mimeType == "" {
				mimeType = "image/png" // Default
			}
			// Include image data length in description
			dataLen := len(block.Image.Data)
			content := fmt.Sprintf("[Image: %s, %d bytes]", mimeType, dataLen)
			messages = append(messages, message.Message{
				Role:      message.RoleUser,
				Content:   content,
				Timestamp: time.Now(),
			})
		} else if block.Audio != nil {
			// Audio block - convert to descriptive text since message.Message only supports text
			// MimeType is a string, not a pointer
			mimeType := block.Audio.MimeType
			if mimeType == "" {
				mimeType = "audio/mpeg" // Default
			}
			// Include audio data length in description
			dataLen := len(block.Audio.Data)
			content := fmt.Sprintf("[Audio: %s, %d bytes]", mimeType, dataLen)
			messages = append(messages, message.Message{
				Role:      message.RoleUser,
				Content:   content,
				Timestamp: time.Now(),
			})
		}
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no valid content blocks found")
	}

	return messages, nil
}

// extractTextFromMessages extracts text content from messages.
func extractTextFromMessages(messages []message.Message) string {
	var parts []string
	for _, msg := range messages {
		if msg.Content != "" {
			parts = append(parts, msg.Content)
		}
	}
	return strings.Join(parts, "\n")
}

// extractPathFromURI extracts file path from URI.
// Basic implementation - handles file:// URIs.
func extractPathFromURI(uri string) string {
	if strings.HasPrefix(uri, "file://") {
		return strings.TrimPrefix(uri, "file://")
	}
	return uri
}

// extractResourceNameFromURI extracts resource name from URI.
// Extracts the filename from a URI (e.g., "file:///tmp/config.yaml" -> "config.yaml").
func extractResourceNameFromURI(uri string) string {
	path := extractPathFromURI(uri)
	// Extract filename from path
	if idx := strings.LastIndex(path, "/"); idx >= 0 && idx < len(path)-1 {
		return path[idx+1:]
	}
	return path
}

// mapStopReason maps Spin finish reason to ACP stop reason.
//
// Maps finish reasons from both Spin agent and OpenAI LLM responses:
// - Spin agent: "timeout", "error", "empty_response", "max_tokens", "max_turns", "cancelled", "refusal"
// - OpenAI: "stop", "length", "tool_calls", "content_filter", "function_call"
//
// Mapping rules:
// - "timeout" → cancelled (context cancellation)
// - "error" → end_turn (execution error, but turn completed)
// - "empty_response" → end_turn (empty response, but turn completed)
// - "max_tokens" → max_tokens (token limit reached)
// - "max_turns" → max_turn_requests (turn limit reached)
// - "cancelled" → cancelled (explicit cancellation)
// - "refusal" → refusal (agent refusal)
// - "length" (OpenAI) → max_tokens (token limit reached)
// - "content_filter" (OpenAI) → refusal (content filtered)
// - "stop" (OpenAI) → end_turn (normal completion)
// - "tool_calls" (OpenAI) → end_turn (tool calls are normal, execution continues)
// - "function_call" (OpenAI, deprecated) → end_turn (same as tool_calls)
// - default → end_turn (unknown reasons default to end_turn)
func mapStopReason(finishReason string) acp.StopReason {
	switch finishReason {
	// Spin agent finish reasons
	case "timeout":
		return acp.StopReasonCancelled
	case "error":
		return acp.StopReasonEndTurn
	case "empty_response":
		return acp.StopReasonEndTurn
	case "max_tokens":
		return acp.StopReasonMaxTokens
	case "max_turns":
		return acp.StopReasonMaxTurnRequests
	case "cancelled":
		return acp.StopReasonCancelled
	case "refusal":
		return acp.StopReasonRefusal
	// OpenAI finish reasons
	case "stop":
		return acp.StopReasonEndTurn
	case "length":
		return acp.StopReasonMaxTokens
	case "tool_calls":
		return acp.StopReasonEndTurn
	case "content_filter":
		return acp.StopReasonRefusal
	case "function_call":
		return acp.StopReasonEndTurn
	// Default: unknown finish reasons default to end_turn
	default:
		return acp.StopReasonEndTurn
	}
}

// mapStopReasonFromError maps agent error to ACP stop reason.
func mapStopReasonFromError(err error, resp *agent.AgentResponse) acp.StopReason {
	// Check if error is context cancellation (including wrapped errors)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return acp.StopReasonCancelled
	}
	if resp != nil {
		return mapStopReason(resp.FinishReason)
	}
	return acp.StopReasonEndTurn
}

// processEvents processes events from the event emitter and sends ACP notifications.
// This runs in a goroutine and continues until the context is cancelled or the channel is closed.
// Tracks thinking blocks across multiple content deltas and file content for diff generation.
func (a *SpinACPAgent) processEvents(ctx context.Context, sessionID acp.SessionId, eventCh <-chan events.Event) {
	// Track file content for diff generation
	fileTracker := newFileContentTracker()

	// Track accumulated content for plan detection
	var accumulatedContent string

	// Get connection once
	a.mu.RLock()
	conn := a.connection
	a.mu.RUnlock()

	if conn == nil {
		// No connection, can't send notifications
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				// Channel closed
				return
			}

			// Reset content on turn start
			if event.Type == events.EventTurnStart {
				accumulatedContent = ""
			}

			// Check for plan detection before tool call starts
			if event.Type == events.EventToolCallStart {
				// If we have content, try to detect plan
				if accumulatedContent != "" && a.agent != nil && a.agent.GetPlanner() == nil {
					plan := planning.DetectPlanFromText(accumulatedContent)
					if plan != nil {
						a.agent.SetPlanner(plan)
						// Send plan notification immediately
						planEntries := convertOrchestrationPlanToACP(plan)
						planUpdate := acp.UpdatePlan(planEntries...)
						notification := acp.SessionNotification{
							SessionId: sessionID,
							Update:    planUpdate,
						}
						if err := conn.SessionUpdate(ctx, notification); err != nil {
							_ = err
						}
					}
				}
			}

			// Handle content delta
			if event.Type == events.EventContentDelta {
				data, ok := event.ContentDeltaData()
				if !ok || data.Role != "assistant" {
					continue
				}

				accumulatedContent += data.Content

				update := acp.UpdateAgentMessageText(data.Content)
				notification := acp.SessionNotification{
					SessionId: sessionID,
					Update:    update,
				}
				if err := conn.SessionUpdate(ctx, notification); err != nil {
					_ = err
				}
				continue
			}

			// Handle thinking delta
			if event.Type == events.EventThinkingDelta {
				data, ok := event.ThinkingDeltaData()
				if !ok {
					continue
				}

				update := acp.UpdateAgentThoughtText(data.Content)
				notification := acp.SessionNotification{
					SessionId: sessionID,
					Update:    update,
				}
				if err := conn.SessionUpdate(ctx, notification); err != nil {
					_ = err
				}
				continue
			}

			// Handle plan updates
			if event.Type == events.EventPlanUpdate {
				data, ok := event.PlanUpdateData()
				if !ok {
					continue
				}

				// Convert plan to ACP entries
				planEntries := convertOrchestrationPlanToACP(data.Plan)
				if len(planEntries) == 0 {
					continue
				}

				// Send plan update notification
				planUpdate := acp.UpdatePlan(planEntries...)
				notification := acp.SessionNotification{
					SessionId: sessionID,
					Update:    planUpdate,
				}

				if err := conn.SessionUpdate(ctx, notification); err != nil {
					_ = err // Log but continue
				}
				continue
			}

			// Convert other events to ACP notification (with file tracker for diff generation)
			update, ok := convertEventToSessionUpdate(event, fileTracker)
			if !ok {
				// Event not mapped to ACP notification, skip
				continue
			}

			// Extract terminal ID from event metadata if this is a tool call complete event
			// We need to release the terminal AFTER sending the notification (per ACP spec)
			var terminalIDToRelease string
			if event.Type == events.EventToolCallComplete {
				if data, ok := event.ToolCallCompleteData(); ok {
					if terminalID, ok := data.Metadata["terminal_id"].(string); ok && terminalID != "" {
						terminalIDToRelease = terminalID
					}
				}
			}

			// Send notification via connection
			notification := acp.SessionNotification{
				SessionId: sessionID,
				Update:    update,
			}
			// Send notification (errors are logged but don't fail execution)
			if err := conn.SessionUpdate(ctx, notification); err != nil {
				// Log error but continue processing
				_ = err
			}

			// Release terminal AFTER notification is sent (per ACP spec requirement)
			if terminalIDToRelease != "" {
				// Type assert to get concrete connection type for terminal client
				if acpConn, ok := conn.(*acp.AgentSideConnection); ok {
					terminalClient := NewACPTerminalClient(acpConn)
					_ = terminalClient.Release(ctx, terminalIDToRelease)
				}
			}
		}
	}
}

// LoadSession loads an existing session from storage.
func (a *SpinACPAgent) LoadSession(ctx context.Context, req acp.LoadSessionRequest) (acp.LoadSessionResponse, error) {
	// Check if storage is available
	if a.storage == nil {
		return acp.LoadSessionResponse{}, fmt.Errorf("session persistence not available")
	}

	// Load session from storage
	sess, err := a.storage.Load(string(req.SessionId))
	if err != nil {
		return acp.LoadSessionResponse{}, fmt.Errorf("failed to load session: %w", err)
	}

	// Validate session
	if err := sess.Validate(); err != nil {
		return acp.LoadSessionResponse{}, fmt.Errorf("loaded session is invalid: %w", err)
	}

	// Convert session ID to ACP format
	sessionID := acp.SessionId(sess.ID)

	// Validate and convert MCP servers synchronously (to catch conversion errors)
	if len(req.McpServers) > 0 {
		configs := make([]mcp.MCPServerConfig, 0, len(req.McpServers))
		for _, server := range req.McpServers {
			config, err := convertACPMcpServerToSpin(server)
			if err != nil {
				return acp.LoadSessionResponse{}, fmt.Errorf("invalid MCP server config: %w", err)
			}
			configs = append(configs, config)
		}

		// Store session before connecting MCP servers
		a.mu.Lock()
		a.sessions[sessionID] = sess
		a.mu.Unlock()

		// Connect servers in background (non-blocking - connection failures don't prevent session loading)
		go func() {
			for _, config := range configs {
				if err := a.mcpManager.ConnectServer(ctx, config); err != nil {
					// Log error but don't fail session loading
					_ = err // Error logged but session still loaded
				}
			}
		}()
	} else {
		// Store session
		a.mu.Lock()
		a.sessions[sessionID] = sess
		a.mu.Unlock()
	}

	// Replay conversation history if connection is available
	a.mu.RLock()
	conn := a.connection
	a.mu.RUnlock()

	if conn != nil {
		// Send conversation history as notifications
		// This allows clients to see the full conversation when loading a session
		// Access exported fields - Session.Turns is exported but may need external synchronization
		// Since we're in LoadSession which has already loaded the session, we can safely access fields
		for _, turn := range sess.Turns {
			if turn == nil {
				continue
			}
			// Access exported fields directly - Turn fields are exported
			userInput := turn.UserInput
			aiResponse := turn.AIResponse

			// Send user message
			if userInput != "" {
				userUpdate := acp.UpdateUserMessageText(userInput)
				notification := acp.SessionNotification{
					SessionId: sessionID,
					Update:    userUpdate,
				}
				if err := conn.SessionUpdate(ctx, notification); err != nil {
					// Log error but continue replaying
					_ = err
				}
			}

			// Send AI response
			if aiResponse != "" {
				// Parse thinking blocks and filter protocol artifacts from AI response
				s := sanitizer.New()
				content, thought := s.Process(aiResponse)

				// Send thinking content if available
				if thought != "" {
					thinkUpdate := acp.UpdateAgentThoughtText(thought)
					notification := acp.SessionNotification{
						SessionId: sessionID,
						Update:    thinkUpdate,
					}
					if err := conn.SessionUpdate(ctx, notification); err != nil {
						_ = err
					}
				}

				// Send message content if available
				if content != "" {
					messageUpdate := acp.UpdateAgentMessageText(content)
					notification := acp.SessionNotification{
						SessionId: sessionID,
						Update:    messageUpdate,
					}
					if err := conn.SessionUpdate(ctx, notification); err != nil {
						_ = err
					}
				}
			} else {
				// No AI response content (e.g. tool call only)
			}
		}
	}

	// Build response
	resp := acp.LoadSessionResponse{
		// Models and Modes are optional and are not implemented
		Models: nil,
		Modes:  nil,
	}

	return resp, nil
}

// sendPlanNotifications sends plan notifications if a plan is detected.
// First checks for structured planning.Plan, then falls back to text-based detection.
func (a *SpinACPAgent) sendPlanNotifications(ctx context.Context, sessionID acp.SessionId, agentResp *agent.AgentResponse) error {
	// Get connection
	a.mu.RLock()
	conn := a.connection
	a.mu.RUnlock()

	if conn == nil {
		return nil // No connection, can't send notifications
	}

	if agentResp == nil {
		return nil
	}

	var planEntries []acp.PlanEntry

	// First, try to get structured plan from agent
	if a.agent != nil {
		agentPlan := a.agent.GetPlanner()
		if agentPlan != nil {
			// Convert agent plan to ACP plan entries
			planEntries = convertOrchestrationPlanToACP(agentPlan)
		}
	}

	// Fallback to text-based detection if no structured plan found
	if len(planEntries) == 0 && agentResp.Output != "" {
		plan := planning.DetectPlanFromText(agentResp.Output)
		if plan != nil {
			planEntries = convertOrchestrationPlanToACP(plan)
		}
	}

	if len(planEntries) == 0 {
		return nil // No plan detected
	}

	// Send plan update notification
	planUpdate := acp.UpdatePlan(planEntries...)
	notification := acp.SessionNotification{
		SessionId: sessionID,
		Update:    planUpdate,
	}

	if err := conn.SessionUpdate(ctx, notification); err != nil {
		return fmt.Errorf("failed to send plan notification: %w", err)
	}

	return nil
}

// Cancel cancels ongoing operations for a session.
//
// This cancels any in-progress prompt execution for the specified session.
// The cancellation is done by canceling the context used for the prompt execution,
// which will cause the agent execution to stop and return a cancelled stop reason.
//
// If there is no in-progress execution for the session, this is a no-op.
func (a *SpinACPAgent) Cancel(ctx context.Context, notif acp.CancelNotification) error {
	// Validate session exists
	a.mu.RLock()
	_, exists := a.sessions[notif.SessionId]
	a.mu.RUnlock()
	if !exists {
		return fmt.Errorf("session not found: %s", notif.SessionId)
	}

	// Cancel in-progress execution for this session
	a.mu.Lock()
	defer a.mu.Unlock()

	if cancel, exists := a.cancels[notif.SessionId]; exists {
		// Cancel the context for this session's prompt execution
		cancel()
		// Remove cancel function (it is cleaned up by defer in Prompt, but remove it here too)
		delete(a.cancels, notif.SessionId)
	}

	// Note: Permission requests are cancelled automatically when the context is cancelled,
	// since the approval service checks ctx.Done() in invokeHandler.

	return nil
}

// getAvailableModes returns all available session modes.
func getAvailableModes() []acp.SessionMode {
	return []acp.SessionMode{
		{
			Id:          acp.SessionModeId("regular"),
			Name:        "Regular",
			Description: stringPtr("Full-featured interactive coding mode with access to all tools"),
		},
		{
			Id:          acp.SessionModeId("review"),
			Name:        "Review",
			Description: stringPtr("Read-only code analysis and review mode"),
		},
		{
			Id:          acp.SessionModeId("compact"),
			Name:        "Compact",
			Description: stringPtr("Quick queries with minimal context and tool access"),
		},
		{
			Id:          acp.SessionModeId("planning"),
			Name:        "Planning",
			Description: stringPtr("Task decomposition and planning mode with context-only tools"),
		},
	}
}

// getDefaultMode returns the default session mode.
func getDefaultMode() acp.SessionModeId {
	return acp.SessionModeId("regular")
}

// stringPtr returns a pointer to the string.
func stringPtr(s string) *string {
	return &s
}

// sendAvailableCommandsUpdate sends an available_commands_update notification.
func (a *SpinACPAgent) sendAvailableCommandsUpdate(ctx context.Context, sessionID acp.SessionId) error {
	// Get connection
	a.mu.RLock()
	conn := a.connection
	a.mu.RUnlock()

	if conn == nil {
		return nil // No connection, can't send notifications
	}

	// Get all registered commands
	allCommands := commands.ListCommands()

	// Convert to ACP AvailableCommand format
	availableCommands := make([]acp.AvailableCommand, 0, len(allCommands))
	for _, cmd := range allCommands {
		// Skip exit/quit commands as they're TUI-only
		if cmd.Name() == "/exit" || cmd.Name() == "/quit" {
			continue
		}

		availableCommands = append(availableCommands, acp.AvailableCommand{
			Name:        cmd.Name(),
			Description: cmd.Description(),
		})
	}

	// Create notification
	update := acp.SessionAvailableCommandsUpdate{
		AvailableCommands: availableCommands,
	}

	notification := acp.SessionNotification{
		SessionId: sessionID,
		Update: acp.SessionUpdate{
			AvailableCommandsUpdate: &update,
		},
	}

	if err := conn.SessionUpdate(ctx, notification); err != nil {
		return fmt.Errorf("failed to send available commands update: %w", err)
	}

	return nil
}

// SetSessionMode sets the session mode.
func (a *SpinACPAgent) SetSessionMode(ctx context.Context, req acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	// Validate session exists
	a.mu.RLock()
	_, exists := a.sessions[req.SessionId]
	a.mu.RUnlock()
	if !exists {
		return acp.SetSessionModeResponse{}, fmt.Errorf("session not found: %s", req.SessionId)
	}

	// Validate mode ID is in available modes
	availableModes := getAvailableModes()
	validMode := false
	for _, mode := range availableModes {
		if mode.Id == req.ModeId {
			validMode = true
			break
		}
	}
	if !validMode {
		return acp.SetSessionModeResponse{}, fmt.Errorf("invalid mode: %s (must be one of: regular, review, compact, planning)", req.ModeId)
	}

	// Update stored mode
	a.mu.Lock()
	a.sessionModes[req.SessionId] = req.ModeId
	a.mu.Unlock()

	// Send mode update notification
	if a.connection != nil {
		update := acp.SessionUpdate{
			CurrentModeUpdate: &acp.SessionCurrentModeUpdate{
				CurrentModeId: req.ModeId,
			},
		}
		notif := acp.SessionNotification{
			SessionId: req.SessionId,
			Update:    update,
		}
		if err := a.connection.SessionUpdate(ctx, notif); err != nil {
			// Log error but don't fail the mode change
			_ = err
		}
	}

	return acp.SetSessionModeResponse{}, nil
}

// Authenticate handles authentication requests.
// This method is not implemented.
func (a *SpinACPAgent) Authenticate(ctx context.Context, req acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, fmt.Errorf("Authenticate: %w", ErrNotImplemented)
}

// RequestPermission requests user permission for a tool call operation.
//
// This method integrates with Spin's ApprovalService to handle permission requests
// from ACP clients. It converts ACP permission requests to Spin security operations
// and returns ACP-formatted responses.
func (a *SpinACPAgent) RequestPermission(ctx context.Context, req acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	// Validate session exists and get approval service
	a.mu.RLock()
	_, exists := a.sessions[req.SessionId]
	approvalService := a.approvalService
	a.mu.RUnlock()

	if !exists {
		return acp.RequestPermissionResponse{}, fmt.Errorf("session not found: %s", req.SessionId)
	}

	if approvalService == nil {
		return acp.RequestPermissionResponse{}, fmt.Errorf("approval service not configured")
	}

	// The agent's RequestPermission method is called by the CLIENT.
	// This is the reverse flow: the client is calling the agent to request permission.
	// This happens when the client wants to proactively request permission for an operation.
	//
	// The normal flow (agent → client) is handled by the approval handler calling
	// connection.RequestPermission() directly.
	//
	// For this reverse flow (client → agent), we need to:
	// 1. Convert the ACP tool call to a security operation
	// 2. Request approval through the approval service
	// 3. Based on the approval response, select the appropriate option from the client's options
	// 4. Return the selected option

	// Get session to extract work directory
	a.mu.RLock()
	sess, exists := a.sessions[req.SessionId]
	a.mu.RUnlock()

	if !exists {
		return acp.RequestPermissionResponse{}, fmt.Errorf("session not found: %s", req.SessionId)
	}

	// Convert tool call to operation
	operation, err := a.convertToolCallToOperation(req.ToolCall, sess.WorkDir)
	if err != nil {
		return acp.RequestPermissionResponse{}, fmt.Errorf("failed to convert tool call: %w", err)
	}

	// Request approval through the approval service
	_, approved, err := approvalService.RequestApproval(ctx, operation)
	if err != nil {
		// If context was cancelled, return cancelled outcome
		if ctx.Err() != nil {
			return acp.RequestPermissionResponse{
				Outcome: acp.NewRequestPermissionOutcomeCancelled(),
			}, nil
		}
		return acp.RequestPermissionResponse{}, fmt.Errorf("approval request failed: %w", err)
	}

	// Select appropriate option based on approval response
	if approved {
		// Find first allow option
		for _, opt := range req.Options {
			if opt.Kind == acp.PermissionOptionKindAllowOnce || opt.Kind == acp.PermissionOptionKindAllowAlways {
				return acp.RequestPermissionResponse{
					Outcome: acp.NewRequestPermissionOutcomeSelected(opt.OptionId),
				}, nil
			}
		}
	} else {
		// Find first reject/deny option
		for _, opt := range req.Options {
			if opt.Kind == acp.PermissionOptionKindRejectOnce || opt.Kind == acp.PermissionOptionKindRejectAlways {
				return acp.RequestPermissionResponse{
					Outcome: acp.NewRequestPermissionOutcomeSelected(opt.OptionId),
				}, nil
			}
		}
	}

	// No matching option found, return cancelled
	return acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeCancelled(),
	}, nil
}

// convertToolCallToOperation converts an ACP tool call to a Spin security operation.
func (a *SpinACPAgent) convertToolCallToOperation(toolCall acp.RequestPermissionToolCall, workDir string) (security.Operation, error) {
	// Extract tool name from title
	toolName := "unknown"
	if toolCall.Title != nil {
		toolName = *toolCall.Title
	}

	// Extract reason from tool call (use tool name as reason if no other reason available)
	reason := fmt.Sprintf("Tool call: %s", toolName)

	// Create command from tool call
	// Create a basic command structure
	// Additional details may be extracted from RawInput if needed
	cmd := &security.Command{
		Program: toolName,
		Args:    []string{},
		Raw:     toolName,
		WorkDir: workDir,
	}

	// Try to extract parameters from RawInput if available
	if toolCall.RawInput != nil {
		if rawInputMap, ok := toolCall.RawInput.(map[string]interface{}); ok {
			// Build args from raw input
			var args []string
			for key, value := range rawInputMap {
				args = append(args, fmt.Sprintf("--%s=%v", key, value))
			}
			cmd.Args = args
			// Update raw command string
			cmd.Raw = fmt.Sprintf("%s %s", toolName, strings.Join(args, " "))
		}
	}

	return security.NewOperation(cmd, reason, workDir), nil
}

// GetClientCapabilities returns the client capabilities stored after Initialize.
func (a *SpinACPAgent) GetClientCapabilities() *acp.ClientCapabilities {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.clientCaps
}
