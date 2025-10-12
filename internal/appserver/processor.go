package appserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/protocol"
	"github.com/dmytrogajewski/spin/internal/protocol/jsonrpc"
	"github.com/google/uuid"
)

// Processor handles app-server business logic
type Processor struct {
	mu            sync.RWMutex
	agent         *core.Agent
	emitter       *core.EventEmitter
	workspacePath string
	version       string
	conversations map[string]*Conversation
	output        io.Writer
}

// Conversation tracks a single conversation state
type Conversation struct {
	ID       protocol.ConversationID
	TurnID   string
	History  []core.Message
	cancel   context.CancelFunc
	taskMode string         // current task mode name
	mu       sync.RWMutex   // protects taskMode access
}

// ProcessorConfig contains processor configuration
type ProcessorConfig struct {
	WorkspacePath string
	Version       string
	Provider      llm.Provider
	Executor      *core.Executor
	Validator     *core.Validator
	Environment   *core.Environment
}

// NewProcessor creates a new processor
func NewProcessor(config ProcessorConfig) (*Processor, error) {
	// Create default dependencies if not provided
	executor := config.Executor
	if executor == nil {
		var err error
		executor, err = core.NewExecutor(config.WorkspacePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create executor: %w", err)
		}
	}

	validator := config.Validator
	if validator == nil {
		validator = core.NewValidator()
	}

	environment := config.Environment
	if environment == nil {
		var err error
		environment, err = core.GatherEnvironment(config.WorkspacePath)
		if err != nil {
			return nil, fmt.Errorf("failed to gather environment: %w", err)
		}
	}

	// Create event emitter
	emitter := core.NewEventEmitter(core.DefaultEventBufferSize)

	// Create agent if provider is provided
	var agent *core.Agent
	if config.Provider != nil {
		var err error
		agent, err = core.NewAgent(
			config.Provider,
			executor,
			validator,
			environment,
			emitter,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent: %w", err)
		}
	}

	return &Processor{
		agent:         agent,
		emitter:       emitter,
		workspacePath: config.WorkspacePath,
		version:       config.Version,
		conversations: make(map[string]*Conversation),
	}, nil
}

// SetOutput sets the output writer for notifications
func (p *Processor) SetOutput(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.output = w
}

// HandleInitialize sets up workspace and config
func (p *Processor) HandleInitialize(ctx context.Context, params jsonrpc.InitializeParams) (jsonrpc.InitializeResult, error) {
	// Apply config overrides if needed
	if params.Config != nil {
		// TODO: Apply config overrides
	}

	return jsonrpc.InitializeResult{
		Status:  "ok",
		Version: p.version,
	}, nil
}

// HandleSendMessage starts a conversation turn
func (p *Processor) HandleSendMessage(ctx context.Context, params jsonrpc.SendMessageParams) (jsonrpc.SendMessageResult, error) {
	p.mu.Lock()

	var conv *Conversation

	// Get or create conversation
	if params.ConversationID == nil {
		// New conversation
		convID := protocol.NewConversationID()

		conv = &Conversation{
			ID:       convID,
			History:  []core.Message{},
			taskMode: "regular", // default mode
		}
		p.conversations[convID.String()] = conv
	} else {
		// Existing conversation
		var ok bool
		conv, ok = p.conversations[*params.ConversationID]
		if !ok {
			p.mu.Unlock()
			return jsonrpc.SendMessageResult{},
				jsonrpc.NewError(jsonrpc.ConversationNotFound, "conversation not found")
		}
	}

	// Handle task mode switch
	if params.TaskMode != nil {
		taskMode := *params.TaskMode

		// Validate task mode
		if err := jsonrpc.ValidateTaskMode(taskMode); err != nil {
			p.mu.Unlock()
			return jsonrpc.SendMessageResult{},
				jsonrpc.NewError(jsonrpc.InvalidParams, err.Error())
		}

		// Apply task mode
		conv.mu.Lock()
		conv.taskMode = taskMode
		conv.mu.Unlock()
	}

	// Get current task mode for response
	conv.mu.RLock()
	currentMode := conv.taskMode
	if currentMode == "" {
		currentMode = "regular"
	}
	conv.mu.RUnlock()

	// Generate turn ID
	turnID := generateTurnID()
	conv.TurnID = turnID

	// Create cancellable context for this turn
	turnCtx, cancel := context.WithCancel(ctx)
	conv.cancel = cancel

	p.mu.Unlock()

	// Start turn in background
	go p.runTurn(turnCtx, conv, params.Message, turnID)

	return jsonrpc.SendMessageResult{
		ConversationID: conv.ID.String(),
		TurnID:         turnID,
		TaskMode:       currentMode,
	}, nil
}

// runTurn executes a conversation turn
func (p *Processor) runTurn(ctx context.Context, conv *Conversation, message string, turnID string) {
	// Send turn_start notification
	p.sendNotification("turn_start", protocol.TurnStart{
		TurnID:      turnID,
		UserMessage: message,
	})

	// If no agent is configured, just echo back
	if p.agent == nil {
		p.sendNotification("assistant_delta", protocol.AssistantDelta{
			Delta: "Agent not configured. Message received: " + message,
		})
		p.sendNotification("turn_complete", protocol.TurnComplete{
			TurnID:       turnID,
			FinalMessage: "Turn completed (no agent)",
		})
		return
	}

	// Get current task mode
	conv.mu.RLock()
	taskMode := conv.taskMode
	if taskMode == "" {
		taskMode = "regular"
	}
	conv.mu.RUnlock()

	// Create agent request with task mode
	req := &core.AgentRequest{
		Input:    message,
		History:  conv.History,
		TaskName: taskMode,
	}

	// Subscribe to agent events
	subscriptionID, eventChan, err := p.emitter.Subscribe()
	if err != nil {
		p.sendNotification("status_update", protocol.StatusUpdate{
			Message: fmt.Sprintf("Failed to subscribe to events: %v", err),
			Level:   protocol.StatusLevelError,
		})
		return
	}
	defer p.emitter.Unsubscribe(subscriptionID)

	// Execute agent in background
	resultChan := make(chan error, 1)
	go func() {
		_, err := p.agent.Execute(ctx, req)
		resultChan <- err
	}()

	// Stream events back as protocol messages
	for {
		select {
		case event := <-eventChan:
			if msg, ok := protocol.FromCoreEvent(event); ok {
				p.sendProtocolMessage(msg)
			}

		case err := <-resultChan:
			// Agent execution completed
			if err != nil {
				p.sendNotification("status_update", protocol.StatusUpdate{
					Message: fmt.Sprintf("Turn failed: %v", err),
					Level:   protocol.StatusLevelError,
				})
			}

			// Send turn_complete notification
			p.sendNotification("turn_complete", protocol.TurnComplete{
				TurnID:       turnID,
				FinalMessage: "Turn completed",
			})
			return

		case <-ctx.Done():
			// Context cancelled
			return
		}
	}
}

// sendNotification sends a notification to the client
func (p *Processor) sendNotification(method string, params interface{}) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.output == nil {
		return
	}

	paramsJSON, _ := json.Marshal(params)
	notif := jsonrpc.Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}
	json.NewEncoder(p.output).Encode(notif)
}

// sendProtocolMessage sends a protocol message
func (p *Processor) sendProtocolMessage(msg protocol.Message) {
	// Parse message to get the specific type
	data, err := protocol.ParseMessage(msg)
	if err != nil {
		return
	}

	// Send as notification using message type as method
	p.sendNotification(msg.Type, data)
}

// HandleApproveTool approves/rejects tool calls
func (p *Processor) HandleApproveTool(ctx context.Context, params jsonrpc.ApproveToolParams) (jsonrpc.ApproveToolResult, error) {
	// TODO: Forward approval to appropriate conversation/agent
	return jsonrpc.ApproveToolResult{Status: "ok"}, nil
}

// HandleCancelTurn cancels an in-progress turn
func (p *Processor) HandleCancelTurn(ctx context.Context, params jsonrpc.CancelTurnParams) (jsonrpc.CancelTurnResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Find conversation with this turn
	for _, conv := range p.conversations {
		if conv.TurnID == params.TurnID && conv.cancel != nil {
			conv.cancel()
			return jsonrpc.CancelTurnResult{Status: "ok"}, nil
		}
	}

	return jsonrpc.CancelTurnResult{},
		jsonrpc.NewError(jsonrpc.InvalidState, "turn not found or already completed")
}

// HandleSearchFiles searches for files in workspace
func (p *Processor) HandleSearchFiles(ctx context.Context, params jsonrpc.SearchFilesParams) (jsonrpc.SearchFilesResult, error) {
	// Perform file search
	files, err := SearchFiles(p.workspacePath, params.Query, params.Limit)
	if err != nil {
		return jsonrpc.SearchFilesResult{}, err
	}
	return jsonrpc.SearchFilesResult{Files: files}, nil
}

func generateTurnID() string {
	return "turn-" + uuid.New().String()
}
