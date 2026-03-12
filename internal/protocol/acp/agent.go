// Package acp provides the ACP protocol implementation.
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
	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/agent/sanitizer"
	"github.com/dmytrogajewski/spin/internal/commands"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/planning"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/appinfo"
)

const (
	unknownValue  = "unknown"
	mimeImagePNG  = "image/png"
	roleAssistant = "assistant"
	toolWriteFile = "write_file"
)

var (
	// ErrNilAgent is returned when agent is nil.
	ErrNilAgent = errors.New("agent cannot be nil")
	// ErrNilMCPService is returned when MCP service is nil.
	ErrNilMCPService = errors.New("mcp service cannot be nil")
	// ErrNilEmitter is returned when emitter is nil.
	ErrNilEmitter = errors.New("emitter cannot be nil")
	// ErrNotImplemented is returned for methods that are not implemented.
	ErrNotImplemented = errors.New("not implemented")
	ErrWorkingDirectoryIsRequired = errors.New("working directory is required")
	ErrHTTPTransportIsNotSupported = errors.New("HTTP transport is not supported")
	ErrSseTransportIsNotSupported = errors.New("SSE transport is not supported")
	ErrNoTransportSpecified = errors.New("no transport specified")
	ErrSessionNotFound = errors.New("session not found")
	ErrPromptCannotBeEmpty = errors.New("prompt cannot be empty")
	ErrConversationManagerNotConfigured = errors.New("conversation manager not configured")
	ErrNoValidContentBlocksFound = errors.New("no valid content blocks found")
	ErrSessionPersistenceNotAvailable = errors.New("session persistence not available")
	ErrSessionNotFound2 = errors.New("session not found")
	ErrSessionNotFound3 = errors.New("session not found")
	ErrInvalidMode = errors.New("invalid mode")
	ErrSessionNotFound4 = errors.New("session not found")
	ErrApprovalServiceNotConfigured = errors.New("approval service not configured")
)

// SpinACPAgent implements the acp.Agent interface, adapting Spin's components
// to the Agent Client Protocol.
//
// SpinACPAgent acts as an adapter between ACP protocol requests and Spin's
// internal agent execution, session management, and tool orchestration.
//
// The agent uses ConversationManager for multi-session support, delegating
// conversation lifecycle and history management to the conversation package.
type SpinACPAgent struct {
	agent           *agent.Agent
	mcpService      *mcp.Service
	emitter         *events.EventEmitter
	approvalService *security.ApprovalService // Optional approval service for permission requests.
	approvalHandler *ApprovalHandler       // ACP-specific approval handler.
	clientCaps      *acp.ClientCapabilities   // Stored after Initialize.
	sessions        map[acp.SessionId]*session.Session
	sessionModes    map[acp.SessionId]acp.SessionModeId    // Current mode per session.
	storage         session.Storage                        // Optional session storage for persistence.
	histStorage     history.Storage                        // Optional history storage for persistence.
	connection      notificationSender                     // Optional connection for sending notifications.
	cancels         map[acp.SessionId]context.CancelFunc   // Cancel functions for in-progress prompt executions.
	convManager     *conversation.Manager                  // Manages conversations per session.
	transformers    map[acp.SessionId]*EventTransformer // Event transformers per session.
	acpRuntime      *executor.ACPRuntime                    // ACP runtime for tool registration.
	mu              sync.RWMutex                           // Protects sessions map, sessionModes, connection, cancels, and transformers.
}

// NewSpinACPAgentWithStorage creates a new ACP agent adapter with optional session storage.
//
// The adapter requires:
//   - agent: Core agent for execution
//   - mcpService: MCP service for tool management
//   - emitter: Event emission for notifications
//
// Optional:
//   - storage: Session storage for persistence (if nil, LoadSession will not be available)
//
// Returns an error if any required component is nil.
// If storage is nil, session persistence features (LoadSession) will not be available.
func NewSpinACPAgentWithStorage(
	spinAgent *agent.Agent,
	mcpService *mcp.Service,
	emitter *events.EventEmitter,
	storage session.Storage,
) (*SpinACPAgent, error) {
	if spinAgent == nil {
		return nil, fmt.Errorf("%w", ErrNilAgent)
	}

	if mcpService == nil {
		return nil, fmt.Errorf("%w", ErrNilMCPService)
	}

	if emitter == nil {
		return nil, fmt.Errorf("%w", ErrNilEmitter)
	}

	return &SpinACPAgent{
		agent:           spinAgent,
		mcpService:      mcpService,
		emitter:         emitter,
		approvalService: nil, // Optional - set via SetApprovalService() if needed.
		storage:         storage,
		histStorage:     nil, // Set via SetHistoryStorage() if needed.
		connection:      nil, // Set via SetConnection() after connection is created.
		sessions:        make(map[acp.SessionId]*session.Session),
		sessionModes:    make(map[acp.SessionId]acp.SessionModeId),
		cancels:         make(map[acp.SessionId]context.CancelFunc),
		convManager:     nil, // Set via SetConversationManager() if needed.
		transformers:    make(map[acp.SessionId]*EventTransformer),
	}, nil
}

// SetConversationManager sets the conversation manager for multi-session support.
// When set, the agent will use Conversation.RunTurn() instead of direct agent execution.
func (a *SpinACPAgent) SetConversationManager(mgr *conversation.Manager) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.convManager = mgr
}

// SetHistoryStorage sets the history storage for conversation persistence.
func (a *SpinACPAgent) SetHistoryStorage(storage history.Storage) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.histStorage = storage
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
func (a *SpinACPAgent) SetApprovalHandler(handler *ApprovalHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.approvalHandler = handler
}

// SetACPRuntime sets the ACP runtime for dynamic tool registration.
// This allows Initialize to update the runtime with client capabilities
// and re-register tools based on client support.
func (a *SpinACPAgent) SetACPRuntime(rt *executor.ACPRuntime) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.acpRuntime = rt
}

// Initialize establishes connection and negotiates capabilities.
//
// Negotiates protocol version, advertises agent capabilities based on Spin's
// features, stores client capabilities, and exchanges client/agent info.
func (a *SpinACPAgent) Initialize(_ context.Context, req acp.InitializeRequest) (acp.InitializeResponse, error) {
	// Negotiate protocol version.
	negotiatedVersion := a.negotiateProtocolVersion(req.ProtocolVersion)

	// Build agent capabilities.
	agentCaps := a.buildAgentCapabilities()

	// Store client capabilities.
	a.clientCaps = &req.ClientCapabilities

	// Update runtime with client capabilities and re-register tools
	// This is necessary because tools are registered before Initialize is called,
	// so filesystem tools are not registered if client capabilities were unknown.
	a.mu.RLock()
	rt := a.acpRuntime
	a.mu.RUnlock()

	if rt != nil {
		rt.SetClientCapabilities(&req.ClientCapabilities)
		// Re-register tools now that we know client capabilities.
		if toolRuntime := a.agent.GetToolRuntime(); toolRuntime != nil {
			rt.RegisterTools(toolRuntime.Registry())
		}
	}

	// Build agent info.
	agentInfo := &acp.Implementation{
		Name:    "spin",
		Version: appinfo.ShortVersion(),
	}

	// Build response.
	resp := acp.InitializeResponse{
		ProtocolVersion:   negotiatedVersion,
		AgentCapabilities: agentCaps,
		AgentInfo:         agentInfo,
		AuthMethods:       []acp.AuthMethod{}, // No auth methods initially.
	}

	return resp, nil
}

// negotiateProtocolVersion negotiates the protocol version with the client.
// Currently only supports version 1. Returns version 1 if client requests
// a supported version, otherwise returns the latest supported version.
func (a *SpinACPAgent) negotiateProtocolVersion(clientVersion acp.ProtocolVersion) acp.ProtocolVersion {
	// Currently only support version 1.
	if clientVersion == acp.ProtocolVersionNumber {
		return acp.ProtocolVersionNumber
	}
	// Return latest supported version (currently only version 1).
	return acp.ProtocolVersionNumber
}

// buildAgentCapabilities builds agent capabilities based on Spin's features.
func (a *SpinACPAgent) buildAgentCapabilities() acp.AgentCapabilities {
	return acp.AgentCapabilities{
		LoadSession: a.hasSessionPersistence(),
		PromptCapabilities: acp.PromptCapabilities{
			Image:           true, // Image blocks supported (converted to text description for agent processing).
			Audio:           true, // Audio blocks supported (converted to text description for agent processing).
			EmbeddedContext: true, // Embedded resources fully supported (text and blob).
		},
		McpCapabilities: acp.McpCapabilities{
			Http: false, // MCP manager currently only supports stdio.
			Sse:  false, // MCP manager currently only supports stdio.
		},
	}
}

// hasSessionPersistence checks if session persistence is available.
// Returns true if storage is configured and available.
func (a *SpinACPAgent) hasSessionPersistence() bool {
	return a.storage != nil
}

// NewSession creates a new session with working directory and MCP servers.
func (a *SpinACPAgent) NewSession(ctx context.Context, req acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	if req.Cwd == "" {
		return acp.NewSessionResponse{}, ErrWorkingDirectoryIsRequired
	}

	sess := session.NewSession(req.Cwd)
	sessionID := acp.SessionId(sess.ID)

	if err := a.storeSessionWithMCPServers(ctx, sessionID, sess, req.McpServers); err != nil {
		return acp.NewSessionResponse{}, err
	}

	defaultMode := getDefaultMode()
	a.mu.Lock()
	a.sessionModes[sessionID] = defaultMode
	a.mu.Unlock()

	resp := acp.NewSessionResponse{
		SessionId: sessionID,
		Modes: &acp.SessionModeState{
			AvailableModes: getAvailableModes(),
			CurrentModeId:  defaultMode,
		},
	}

	_ = a.sendAvailableCommandsUpdate(ctx, sessionID)

	return resp, nil
}

// convertACPMcpServerToSpin converts an ACP McpServer to Spin ServerConfig.
func convertACPMcpServerToSpin(acpServer acp.McpServer) (mcp.ServerConfig, error) {
	if acpServer.Stdio != nil {
		// Convert environment variables.
		env := make(map[string]string)
		for _, envVar := range acpServer.Stdio.Env {
			env[envVar.Name] = envVar.Value
		}

		return mcp.ServerConfig{
			Name:    acpServer.Stdio.Name,
			Command: acpServer.Stdio.Command,
			Args:    acpServer.Stdio.Args,
			Env:     env,
		}, nil
	}

	// HTTP and SSE transports are not supported.
	if acpServer.Http != nil {
		return mcp.ServerConfig{}, ErrHTTPTransportIsNotSupported
	}

	if acpServer.Sse != nil {
		return mcp.ServerConfig{}, ErrSseTransportIsNotSupported
	}

	return mcp.ServerConfig{}, ErrNoTransportSpecified
}

// Prompt processes a user prompt and executes the agent loop.
func (a *SpinACPAgent) Prompt(ctx context.Context, req acp.PromptRequest) (acp.PromptResponse, error) {
	sess, err := a.validatePromptRequest(req)
	if err != nil {
		return acp.PromptResponse{}, err
	}

	promptCtx, cancel := a.setupPromptContext(ctx, req.SessionId, sess)
	defer a.cleanupPromptCancel(req.SessionId)

	input, err := a.extractPromptInput(req.Prompt)
	if err != nil {
		return acp.PromptResponse{}, err
	}

	// Handle slash commands.
	if resp, handled, cmdErr := a.tryExecuteCommand(promptCtx, req.SessionId, input); handled {
		return resp, cmdErr
	}

	a.mu.RLock()
	approvalHandler := a.approvalHandler
	convManager := a.convManager
	sess = a.sessions[req.SessionId]
	a.mu.RUnlock()

	if approvalHandler != nil {
		approvalHandler.SetActiveSession(req.SessionId)
		defer approvalHandler.ClearActiveSession()
	}

	if convManager != nil && sess != nil {
		return a.promptWithConversation(promptCtx, req, input, sess.WorkDir, cancel)
	}

	return a.promptWithAgent(promptCtx, req, input, cancel)
}

// validatePromptRequest validates the prompt request and returns the session.
func (a *SpinACPAgent) validatePromptRequest(req acp.PromptRequest) (*session.Session, error) {
	a.mu.RLock()
	sess, exists := a.sessions[req.SessionId]
	a.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s: %w", req.SessionId, ErrSessionNotFound)
	}

	if len(req.Prompt) == 0 {
		return nil, ErrPromptCannotBeEmpty
	}

	return sess, nil
}

// setupPromptContext creates a cancellable context with session metadata.
func (a *SpinACPAgent) setupPromptContext(
	ctx context.Context, sessionID acp.SessionId, sess *session.Session,
) (context.Context, context.CancelFunc) {
	promptCtx, cancel := context.WithCancel(ctx)
	promptCtx = executor.ContextWithSessionID(promptCtx, string(sessionID))
	if sess != nil && sess.WorkDir != "" {
		promptCtx = executor.ContextWithWorkDir(promptCtx, sess.WorkDir)
	}

	a.mu.Lock()
	if existingCancel, cancelExists := a.cancels[sessionID]; cancelExists {
		existingCancel()
	}
	a.cancels[sessionID] = cancel
	a.mu.Unlock()

	return promptCtx, cancel
}

// cleanupPromptCancel removes the cancel function for the session.
func (a *SpinACPAgent) cleanupPromptCancel(sessionID acp.SessionId) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.cancels, sessionID)
}

// extractPromptInput converts content blocks to text input.
func (a *SpinACPAgent) extractPromptInput(blocks []acp.ContentBlock) (string, error) {
	messages, err := convertACPContentBlocksToMessages(blocks)
	if err != nil {
		return "", fmt.Errorf("failed to convert content blocks: %w", err)
	}
	return extractTextFromMessages(messages), nil
}

// tryExecuteCommand checks if input is a command and executes it.
// Returns (response, handled, error).
func (a *SpinACPAgent) tryExecuteCommand(ctx context.Context, sessionID acp.SessionId, input string) (acp.PromptResponse, bool, error) {
	cmd, cmdArgs, isCmd := commands.ParseCommand(input)
	if !isCmd {
		return acp.PromptResponse{}, false, nil
	}

	result, err := a.executeCommand(ctx, cmd, cmdArgs, sessionID)
	if err != nil {
		return acp.PromptResponse{StopReason: acp.StopReasonRefusal}, true, fmt.Errorf("command execution failed: %w", err)
	}

	a.mu.RLock()
	conn := a.connection
	a.mu.RUnlock()

	if conn != nil {
		a.sendSessionUpdate(ctx, sessionID, conn, acp.UpdateAgentMessageText(result))
	}

	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, true, nil
}


// promptWithConversation executes a prompt using ConversationManager.
// This is the new path that uses Conversation.RunTurn() for proper history management.
func (a *SpinACPAgent) promptWithConversation(
	ctx context.Context, req acp.PromptRequest, input, workDir string, cancel context.CancelFunc,
) (acp.PromptResponse, error) {
	a.mu.RLock()
	convManager := a.convManager
	conn := a.connection
	a.mu.RUnlock()

	if convManager == nil {
		return acp.PromptResponse{}, ErrConversationManagerNotConfigured
	}

	conv, err := convManager.GetOrCreate(ctx, string(req.SessionId), workDir)
	if err != nil {
		return acp.PromptResponse{}, fmt.Errorf("failed to get conversation: %w", err)
	}

	transformer := a.ensureTransformer(req.SessionId, conn, conv)

	conv.SetCancel(cancel)
	defer conv.SetCancel(nil)

	unsubscribe, eventsDone := a.subscribeTransformerEvents(ctx, conn, transformer)

	defer func() {
		if unsubscribe != nil {
			unsubscribe()
		}

		if eventsDone != nil {
			<-eventsDone
		}

		cancel()
	}()

	// Execute turn via conversation (manages history automatically).
	err = conv.RunTurn(ctx, input)
	if err != nil {
		// Map error to stop reason.
		stopReason := mapStopReasonFromError(err, nil)

		return acp.PromptResponse{
			StopReason: stopReason,
		}, nil
	}

	// Save history after successful turn (if storage configured).
	a.mu.RLock()
	histStorage := a.histStorage
	a.mu.RUnlock()

	if histStorage != nil {
		_ = conv.GetHistory().Save(histStorage, string(req.SessionId))
	}

	return acp.PromptResponse{
		StopReason: acp.StopReasonEndTurn,
	}, nil
}

// ensureTransformer sets up an event transformer for the session if needed.
func (a *SpinACPAgent) ensureTransformer(
	sessionID acp.SessionId, conn notificationSender, conv *conversation.Conversation,
) *EventTransformer {
	a.mu.Lock()
	defer a.mu.Unlock()

	transformer, exists := a.transformers[sessionID]
	if !exists && conn != nil {
		transformer = NewEventTransformer(sessionID, conn, a.agent)
		a.transformers[sessionID] = transformer
		conv.SetEventTransformer(transformer)
	}

	return transformer
}

// subscribeTransformerEvents subscribes to events and forwards them through the transformer.
// Returns an unsubscribe function and a done channel.
func (a *SpinACPAgent) subscribeTransformerEvents(
	ctx context.Context, conn notificationSender, transformer *EventTransformer,
) (cleanup func(), done chan struct{}) {
	if conn == nil || transformer == nil {
		return nil, nil
	}

	subID, eventCh, subErr := a.emitter.Subscribe()
	if subErr != nil {
		return nil, nil
	}

	eventsDone := make(chan struct{})

	go func() {
		defer close(eventsDone)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventCh:
				if !ok {
					return
				}
				transformer.Transform(ctx, event)
			}
		}
	}()

	unsubscribe := func() { a.emitter.Unsubscribe(subID) }
	return unsubscribe, eventsDone
}

// promptWithAgent executes a prompt using direct agent execution (legacy path).
// This is the fallback path when ConversationManager is not configured.
func (a *SpinACPAgent) promptWithAgent(
	ctx context.Context, req acp.PromptRequest, input string, cancel context.CancelFunc,
) (acp.PromptResponse, error) {
	a.mu.RLock()
	conn := a.connection
	a.mu.RUnlock()

	// Create agent request.
	agentReq := &agent.Request{
		Input:   input,
		Task:    task.DefaultTask(),
		History: []message.Message{},
	}

	// Subscribe to events for real-time notifications.
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
			// Log error but continue without notifications.
			_ = err
		} else {
			unsubscribe = func() {
				a.emitter.Unsubscribe(subID)
			}
			eventsDone = make(chan struct{})

			go func() {
				defer close(eventsDone)

				a.processEvents(ctx, req.SessionId, eventCh)
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

	// Execute agent with cancellable context.
	agentResp, err := a.agent.Execute(ctx, agentReq)
	if err != nil {
		// Map error to stop reason.
		stopReason := mapStopReasonFromError(err, agentResp)

		return acp.PromptResponse{
			StopReason: stopReason,
		}, nil // Return response with stop reason, not error.
	}

	// Send plan notifications if a plan is available.
	if conn != nil && agentResp != nil {
		planErr := a.sendPlanNotifications(ctx, req.SessionId, agentResp)
		if planErr != nil {
			// Log error but continue - plan notifications are non-critical.
			_ = planErr
		}
	}

	// Map finish reason to ACP stop reason.
	stopReason := mapStopReason(agentResp.FinishReason)

	return acp.PromptResponse{
		StopReason: stopReason,
	}, nil
}

// convertACPContentBlocksToMessages converts ACP ContentBlock[] to Spin message.Message[].
func convertACPContentBlocksToMessages(blocks []acp.ContentBlock) ([]message.Message, error) {
	var messages []message.Message

	for _, block := range blocks {
		if msg, ok := convertContentBlock(block); ok {
			messages = append(messages, msg)
		}
	}

	if len(messages) == 0 {
		return nil, ErrNoValidContentBlocksFound
	}

	return messages, nil
}

// convertContentBlock converts a single ACP ContentBlock to a message.
// Returns the message and true if conversion was successful.
func convertContentBlock(block acp.ContentBlock) (message.Message, bool) {
	switch {
	case block.Text != nil:
		return newUserMessage(block.Text.Text), true
	case block.ResourceLink != nil:
		path := extractPathFromURI(block.ResourceLink.Uri)
		return newUserMessage(fmt.Sprintf("File: %s", path)), true
	case block.Resource != nil:
		return convertResourceBlock(block.Resource)
	case block.Image != nil:
		return convertImageBlock(block.Image), true
	case block.Audio != nil:
		return convertAudioBlock(block.Audio), true
	default:
		return message.Message{}, false
	}
}

// newUserMessage creates a user message with the given content.
func newUserMessage(content string) message.Message {
	return message.Message{
		Role:      message.RoleUser,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// convertResourceBlock converts an embedded resource block to a message.
func convertResourceBlock(res *acp.ContentBlockResource) (message.Message, bool) {
	if res.Resource.TextResourceContents != nil {
		uri := res.Resource.TextResourceContents.Uri
		resourceName := extractResourceNameFromURI(uri)
		content := res.Resource.TextResourceContents.Text
		if resourceName != "" {
			content = fmt.Sprintf("[Resource: %s]\n%s", resourceName, content)
		}
		return newUserMessage(content), true
	}

	if res.Resource.BlobResourceContents != nil {
		mimeType := unknownValue
		if res.Resource.BlobResourceContents.MimeType != nil {
			mimeType = *res.Resource.BlobResourceContents.MimeType
		}
		uri := res.Resource.BlobResourceContents.Uri
		resourceName := extractResourceNameFromURI(uri)
		content := fmt.Sprintf("Resource (blob, %s)", mimeType)
		if resourceName != "" {
			content = fmt.Sprintf("[Resource: %s] %s", resourceName, content)
		}
		return newUserMessage(content), true
	}

	return message.Message{}, false
}

// convertImageBlock converts an image content block to a descriptive message.
func convertImageBlock(img *acp.ContentBlockImage) message.Message {
	mimeType := img.MimeType
	if mimeType == "" {
		mimeType = mimeImagePNG
	}
	return newUserMessage(fmt.Sprintf("[Image: %s, %d bytes]", mimeType, len(img.Data)))
}

// convertAudioBlock converts an audio content block to a descriptive message.
func convertAudioBlock(audio *acp.ContentBlockAudio) message.Message {
	mimeType := audio.MimeType
	if mimeType == "" {
		mimeType = "audio/mpeg"
	}
	return newUserMessage(fmt.Sprintf("[Audio: %s, %d bytes]", mimeType, len(audio.Data)))
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
	if after, ok := strings.CutPrefix(uri, "file://"); ok {
		return after
	}

	return uri
}

// extractResourceNameFromURI extracts resource name from URI.
// Extracts the filename from a URI (e.g., "file:///tmp/config.yaml" -> "config.yaml").
func extractResourceNameFromURI(uri string) string {
	path := extractPathFromURI(uri)
	// Extract filename from path.
	if idx := strings.LastIndex(path, "/"); idx >= 0 && idx < len(path)-1 {
		return path[idx+1:]
	}

	return path
}

// mapStopReason maps Spin finish reason to ACP stop reason.
//
// Maps finish reasons from both Spin agent and OpenAI LLM responses:
// - Spin agent: "timeout", "error", "empty_response", "max_tokens", "max_turns", "canceled", "refusal"
// - OpenAI: "stop", "length", "tool_calls", "content_filter", "function_call"
//
// Mapping rules:
// - "timeout" → canceled (context cancellation)
// - "error" → end_turn (execution error, but turn completed)
// - "empty_response" → end_turn (empty response, but turn completed)
// - "max_tokens" → max_tokens (token limit reached)
// - "max_turns" → max_turn_requests (turn limit reached)
// - "canceled" → canceled (explicit cancellation)
// - "refusal" → refusal (agent refusal)
// - "length" (OpenAI) → max_tokens (token limit reached)
// - "content_filter" (OpenAI) → refusal (content filtered)
// - "stop" (OpenAI) → end_turn (normal completion)
// - "tool_calls" (OpenAI) → end_turn (tool calls are normal, execution continues)
// - "function_call" (OpenAI, deprecated) → end_turn (same as tool_calls)
// - default → end_turn (unknown reasons default to end_turn).
func mapStopReason(finishReason string) acp.StopReason {
	switch finishReason {
	// Spin agent finish reasons.
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
	case "canceled":
		return acp.StopReasonCancelled
	case "refusal":
		return acp.StopReasonRefusal
		// OpenAI finish reasons.
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
		// Default: unknown finish reasons default to end_turn.
	default:
		return acp.StopReasonEndTurn
	}
}

// mapStopReasonFromError maps agent error to ACP stop reason.
func mapStopReasonFromError(err error, resp *agent.Response) acp.StopReason {
	// Check if error is context cancellation (including wrapped errors).
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return acp.StopReasonCancelled
	}

	if resp != nil {
		return mapStopReason(resp.FinishReason)
	}

	return acp.StopReasonEndTurn
}

// processEvents processes events from the event emitter and sends ACP notifications.
// This runs in a goroutine and continues until the context is canceled or the channel is closed.
func (a *SpinACPAgent) processEvents(ctx context.Context, sessionID acp.SessionId, eventCh <-chan events.Event) {
	fileTracker := newFileContentTracker()
	var accumulatedContent string

	a.mu.RLock()
	conn := a.connection
	a.mu.RUnlock()

	if conn == nil {
		return
	}

	proc := &eventProcessor{
		agent:       a,
		conn:        conn,
		sessionID:   sessionID,
		fileTracker: fileTracker,
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			accumulatedContent = proc.handleEvent(ctx, event, accumulatedContent)
		}
	}
}

// eventProcessor handles individual event processing for ACP notifications.
type eventProcessor struct {
	agent       *SpinACPAgent
	conn        notificationSender
	sessionID   acp.SessionId
	fileTracker *fileContentTracker
}

// handleEvent processes a single event and returns the updated accumulated content.
func (p *eventProcessor) handleEvent(ctx context.Context, event events.Event, accumulatedContent string) string {
	if event.Type == events.EventTurnStart {
		return ""
	}

	if event.Type == events.EventToolCallStart {
		p.detectAndSendPlan(ctx, accumulatedContent)
	}

	switch event.Type {
	case events.EventContentDelta:
		return p.handleContentDelta(ctx, event, accumulatedContent)
	case events.EventThinkingDelta:
		p.handleThinkingDelta(ctx, event)
	case events.EventPlanUpdate:
		p.handlePlanUpdate(ctx, event)
	default:
		p.handleGenericEvent(ctx, event)
	}

	return accumulatedContent
}

// detectAndSendPlan attempts to detect a plan from accumulated content and sends it.
func (p *eventProcessor) detectAndSendPlan(ctx context.Context, content string) {
	if content == "" || p.agent.agent == nil || p.agent.agent.GetPlanner() != nil {
		return
	}

	plan := planning.DetectPlanFromText(content)
	if plan == nil {
		return
	}

	p.agent.agent.SetPlanner(plan)
	planEntries := convertOrchestrationPlanToACP(plan)
	p.sendNotification(ctx, acp.UpdatePlan(planEntries...))
}

// handleContentDelta processes content delta events.
func (p *eventProcessor) handleContentDelta(ctx context.Context, event events.Event, accumulatedContent string) string {
	data, hasData := event.ContentDeltaData()
	if !hasData || data.Role != roleAssistant {
		return accumulatedContent
	}

	p.sendNotification(ctx, acp.UpdateAgentMessageText(data.Content))
	return accumulatedContent + data.Content
}

// handleThinkingDelta processes thinking delta events.
func (p *eventProcessor) handleThinkingDelta(ctx context.Context, event events.Event) {
	data, hasThinking := event.ThinkingDeltaData()
	if !hasThinking {
		return
	}
	p.sendNotification(ctx, acp.UpdateAgentThoughtText(data.Content))
}

// handlePlanUpdate processes plan update events.
func (p *eventProcessor) handlePlanUpdate(ctx context.Context, event events.Event) {
	data, hasPlan := event.PlanUpdateData()
	if !hasPlan {
		return
	}

	planEntries := convertOrchestrationPlanToACP(data.Plan)
	if len(planEntries) == 0 {
		return
	}

	p.sendNotification(ctx, acp.UpdatePlan(planEntries...))
}

// handleGenericEvent processes events that need conversion to ACP format.
func (p *eventProcessor) handleGenericEvent(ctx context.Context, event events.Event) {
	update, converted := convertEventToSessionUpdate(event, p.fileTracker)
	if !converted {
		return
	}

	terminalID := extractTerminalID(event)

	p.sendNotification(ctx, update)

	if terminalID != "" {
		if acpConn, isACPConn := p.conn.(*acp.AgentSideConnection); isACPConn {
			terminalClient := NewTerminalClient(acpConn)
			_ = terminalClient.Release(ctx, terminalID)
		}
	}
}

// sendNotification sends an ACP session notification.
func (p *eventProcessor) sendNotification(ctx context.Context, update acp.SessionUpdate) {
	notification := acp.SessionNotification{
		SessionId: p.sessionID,
		Update:    update,
	}
	_ = p.conn.SessionUpdate(ctx, notification)
}

// extractTerminalID extracts the terminal ID from a tool call complete event.
func extractTerminalID(event events.Event) string {
	if event.Type != events.EventToolCallComplete {
		return ""
	}
	data, hasComplete := event.ToolCallCompleteData()
	if !hasComplete {
		return ""
	}
	terminalID, isStr := data.Metadata["terminal_id"].(string)
	if !isStr {
		return ""
	}
	return terminalID
}

// LoadSession loads an existing session from storage.
func (a *SpinACPAgent) LoadSession(ctx context.Context, req acp.LoadSessionRequest) (acp.LoadSessionResponse, error) {
	if a.storage == nil {
		return acp.LoadSessionResponse{}, ErrSessionPersistenceNotAvailable
	}

	sessData, err := a.storage.Load(string(req.SessionId))
	if err != nil {
		return acp.LoadSessionResponse{}, fmt.Errorf("failed to load session: %w", err)
	}

	sess := &sessData
	if err = sess.Validate(); err != nil {
		return acp.LoadSessionResponse{}, fmt.Errorf("loaded session is invalid: %w", err)
	}

	sessionID := acp.SessionId(sess.ID)

	if err := a.storeSessionWithMCPServers(ctx, sessionID, sess, req.McpServers); err != nil {
		return acp.LoadSessionResponse{}, err
	}

	a.replayConversationHistory(ctx, sessionID)

	return acp.LoadSessionResponse{Models: nil, Modes: nil}, nil
}

// storeSessionWithMCPServers stores a session and connects MCP servers if provided.
func (a *SpinACPAgent) storeSessionWithMCPServers(
	ctx context.Context, sessionID acp.SessionId, sess *session.Session, mcpServers []acp.McpServer,
) error {
	if len(mcpServers) > 0 {
		configs, err := convertMCPServers(mcpServers)
		if err != nil {
			return err
		}

		a.mu.Lock()
		a.sessions[sessionID] = sess
		a.mu.Unlock()

		go a.connectMCPServersBackground(ctx, configs)
	} else {
		a.mu.Lock()
		a.sessions[sessionID] = sess
		a.mu.Unlock()
	}

	return nil
}

// convertMCPServers converts ACP MCP server configs to Spin format.
func convertMCPServers(servers []acp.McpServer) ([]mcp.ServerConfig, error) {
	configs := make([]mcp.ServerConfig, 0, len(servers))
	for _, server := range servers {
		config, err := convertACPMcpServerToSpin(server)
		if err != nil {
			return nil, fmt.Errorf("invalid MCP server config: %w", err)
		}
		configs = append(configs, config)
	}
	return configs, nil
}

// connectMCPServersBackground connects MCP servers in background.
func (a *SpinACPAgent) connectMCPServersBackground(ctx context.Context, configs []mcp.ServerConfig) {
	for _, config := range configs {
		_ = a.mcpService.ConnectServer(ctx, config)
	}
}

// replayConversationHistory replays stored conversation history as ACP notifications.
func (a *SpinACPAgent) replayConversationHistory(ctx context.Context, sessionID acp.SessionId) {
	a.mu.RLock()
	conn := a.connection
	histStorage := a.histStorage
	a.mu.RUnlock()

	if conn == nil || histStorage == nil {
		return
	}

	histData, loadErr := histStorage.Load(string(sessionID))
	if loadErr != nil {
		return
	}

	for _, msg := range histData.Messages {
		a.replayMessage(ctx, sessionID, conn, msg)
	}
}

// replayMessage replays a single message as ACP notification(s).
func (a *SpinACPAgent) replayMessage(ctx context.Context, sessionID acp.SessionId, conn notificationSender, msg message.Message) {
	if msg.Content == "" {
		return
	}

	switch msg.Role {
	case message.RoleUser:
		a.sendSessionUpdate(ctx, sessionID, conn, acp.UpdateUserMessageText(msg.Content))
	case message.RoleAssistant:
		a.replayAssistantMessage(ctx, sessionID, conn, msg.Content)
	}
}

// replayAssistantMessage replays an assistant message, separating thinking from content.
func (a *SpinACPAgent) replayAssistantMessage(ctx context.Context, sessionID acp.SessionId, conn notificationSender, content string) {
	s := sanitizer.New()
	cleanContent, thought := s.Process(content)

	if thought != "" {
		a.sendSessionUpdate(ctx, sessionID, conn, acp.UpdateAgentThoughtText(thought))
	}
	if cleanContent != "" {
		a.sendSessionUpdate(ctx, sessionID, conn, acp.UpdateAgentMessageText(cleanContent))
	}
}

// sendSessionUpdate sends a session update notification.
func (a *SpinACPAgent) sendSessionUpdate(ctx context.Context, sessionID acp.SessionId, conn notificationSender, update acp.SessionUpdate) {
	notification := acp.SessionNotification{
		SessionId: sessionID,
		Update:    update,
	}
	_ = conn.SessionUpdate(ctx, notification)
}

// sendPlanNotifications sends plan notifications if a plan is detected.
// First checks for structured planning.Plan, then falls back to text-based detection.
func (a *SpinACPAgent) sendPlanNotifications(ctx context.Context, sessionID acp.SessionId, agentResp *agent.Response) error {
	// Get connection.
	a.mu.RLock()
	conn := a.connection
	a.mu.RUnlock()

	if conn == nil {
		return nil // No connection, can't send notifications.
	}

	if agentResp == nil {
		return nil
	}

	var planEntries []acp.PlanEntry

	// First, try to get structured plan from agent.
	if a.agent != nil {
		agentPlan := a.agent.GetPlanner()
		if agentPlan != nil {
			// Convert agent plan to ACP plan entries.
			planEntries = convertOrchestrationPlanToACP(agentPlan)
		}
	}

	// Fallback to text-based detection if no structured plan found.
	if len(planEntries) == 0 && agentResp.Output != "" {
		plan := planning.DetectPlanFromText(agentResp.Output)
		if plan != nil {
			planEntries = convertOrchestrationPlanToACP(plan)
		}
	}

	if len(planEntries) == 0 {
		return nil // No plan detected.
	}

	// Send plan update notification.
	planUpdate := acp.UpdatePlan(planEntries...)
	notification := acp.SessionNotification{
		SessionId: sessionID,
		Update:    planUpdate,
	}

	err := conn.SessionUpdate(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to send plan notification: %w", err)
	}

	return nil
}

// Cancel cancels ongoing operations for a session.
//
// This cancels any in-progress prompt execution for the specified session.
// The cancellation is done by canceling the context used for the prompt execution,
// which will cause the agent execution to stop and return a canceled stop reason.
//
// If there is no in-progress execution for the session, this is a no-op.
func (a *SpinACPAgent) Cancel(_ context.Context, notif acp.CancelNotification) error {
	// Validate session exists.
	a.mu.RLock()
	_, exists := a.sessions[notif.SessionId]
	a.mu.RUnlock()

	if !exists {
return fmt.Errorf("session not found: %s: %w", notif.SessionId, ErrSessionNotFound2)
	}

	// Cancel in-progress execution for this session.
	a.mu.Lock()
	defer a.mu.Unlock()

	if cancel, hasCancel := a.cancels[notif.SessionId]; hasCancel {
		// Cancel the context for this session's prompt execution.
		cancel()
		// Remove cancel function (it is cleaned up by defer in Prompt, but remove it here too).
		delete(a.cancels, notif.SessionId)
	}

	// Note: Permission requests are canceled automatically when the context is canceled,
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
	// Get connection.
	a.mu.RLock()
	conn := a.connection
	a.mu.RUnlock()

	if conn == nil {
		return nil // No connection, can't send notifications.
	}

	// Get all registered commands.
	allCommands := commands.ListCommands()

	// Convert to ACP AvailableCommand format.
	availableCommands := make([]acp.AvailableCommand, 0, len(allCommands))
	for _, cmd := range allCommands {
		// Skip exit/quit commands as they're TUI-only.
		if cmd.Name() == "/exit" || cmd.Name() == "/quit" {
			continue
		}

		availableCommands = append(availableCommands, acp.AvailableCommand{
			Name:        cmd.Name(),
			Description: cmd.Description(),
		})
	}

	// Create notification.
	update := acp.SessionAvailableCommandsUpdate{
		AvailableCommands: availableCommands,
	}

	notification := acp.SessionNotification{
		SessionId: sessionID,
		Update: acp.SessionUpdate{
			AvailableCommandsUpdate: &update,
		},
	}

	err := conn.SessionUpdate(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to send available commands update: %w", err)
	}

	return nil
}

// SetSessionMode sets the session mode.
func (a *SpinACPAgent) SetSessionMode(ctx context.Context, req acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	// Validate session exists.
	a.mu.RLock()
	_, exists := a.sessions[req.SessionId]
	a.mu.RUnlock()

	if !exists {
return acp.SetSessionModeResponse{}, fmt.Errorf("session not found: %s: %w", req.SessionId, ErrSessionNotFound3)
	}

	// Validate mode ID is in available modes.
	availableModes := getAvailableModes()
	validMode := false

	for _, mode := range availableModes {
		if mode.Id == req.ModeId {
			validMode = true

			break
		}
	}

	if !validMode {
		return acp.SetSessionModeResponse{}, fmt.Errorf(
			"invalid mode: %s (must be one of: regular, review, compact, planning): %w",
			req.ModeId, ErrInvalidMode)
	}

	// Update stored mode.
	a.mu.Lock()
	a.sessionModes[req.SessionId] = req.ModeId
	a.mu.Unlock()

	// Send mode update notification.
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
		err := a.connection.SessionUpdate(ctx, notif)
		if err != nil {
			// Log error but don't fail the mode change.
			_ = err
		}
	}

	return acp.SetSessionModeResponse{}, nil
}

// Authenticate handles authentication requests.
// This method is not implemented.
func (a *SpinACPAgent) Authenticate(_ context.Context, _ acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, fmt.Errorf("Authenticate: %w", ErrNotImplemented)
}

// RequestPermission requests user permission for a tool call operation.
// This handles the reverse flow where the client calls the agent to request permission.
func (a *SpinACPAgent) RequestPermission(ctx context.Context, req acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	a.mu.RLock()
	sess, exists := a.sessions[req.SessionId]
	approvalService := a.approvalService
	a.mu.RUnlock()

	if !exists {
		return acp.RequestPermissionResponse{}, fmt.Errorf("session not found: %s: %w", req.SessionId, ErrSessionNotFound4)
	}

	if approvalService == nil {
		return acp.RequestPermissionResponse{}, ErrApprovalServiceNotConfigured
	}

	operation, err := a.convertToolCallToOperation(req.ToolCall, sess.WorkDir)
	if err != nil {
		return acp.RequestPermissionResponse{}, fmt.Errorf("failed to convert tool call: %w", err)
	}

	_, approved, approvalErr := approvalService.RequestApproval(ctx, operation)
	if approvalErr != nil {
		if ctx.Err() != nil {
			return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeCancelled()}, nil
		}
		return acp.RequestPermissionResponse{}, fmt.Errorf("approval request failed: %w", approvalErr)
	}

	return selectPermissionOption(approved, req.Options), nil
}

// selectPermissionOption finds the matching option based on the approval decision.
func selectPermissionOption(approved bool, options []acp.PermissionOption) acp.RequestPermissionResponse {
	for _, opt := range options {
		if approved && (opt.Kind == acp.PermissionOptionKindAllowOnce || opt.Kind == acp.PermissionOptionKindAllowAlways) {
			return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(opt.OptionId)}
		}
		if !approved && (opt.Kind == acp.PermissionOptionKindRejectOnce || opt.Kind == acp.PermissionOptionKindRejectAlways) {
			return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(opt.OptionId)}
		}
	}
	return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeCancelled()}
}

// convertToolCallToOperation converts an ACP tool call to a Spin security operation.
func (a *SpinACPAgent) convertToolCallToOperation(toolCall acp.RequestPermissionToolCall, workDir string) (security.Operation, error) {
	// Extract tool name from title.
	toolName := unknownValue
	if toolCall.Title != nil {
		toolName = *toolCall.Title
	}

	// Extract reason from tool call (use tool name as reason if no other reason available).
	reason := fmt.Sprintf("Tool call: %s", toolName)

	// Create command from tool call
	// Create a basic command structure
	// Additional details may be extracted from RawInput if needed.
	cmd := &security.Command{
		Program: toolName,
		Args:    []string{},
		Raw:     toolName,
		WorkDir: workDir,
	}

	// Try to extract parameters from RawInput if available.
	if toolCall.RawInput != nil {
		if rawInputMap, ok := toolCall.RawInput.(map[string]any); ok {
			// Build args from raw input.
			var args []string
			for key, value := range rawInputMap {
				args = append(args, fmt.Sprintf("--%s=%v", key, value))
			}

			cmd.Args = args
			// Update raw command string.
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
