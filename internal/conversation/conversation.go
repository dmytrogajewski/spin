package conversation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/frame"
	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/agent/tasks"
	"github.com/dmytrogajewski/spin/internal/contexteng/history"
	"github.com/dmytrogajewski/spin/internal/contexteng/retrieval"
	"github.com/dmytrogajewski/spin/internal/events"
	gitpkg "github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/lsp"
	mcppkg "github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
	"github.com/dmytrogajewski/spin/internal/session"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/pkg/tokenizer"
)

var (
	// ErrAgentIsRequired is a sentinel error.
	ErrAgentIsRequired = errors.New("agent is required")
	// ErrEmitterIsRequired is a sentinel error.
	ErrEmitterIsRequired = errors.New("emitter is required")
	// ErrWorkdirIsRequired is a sentinel error.
	ErrWorkdirIsRequired = errors.New("workDir is required")
	// ErrHarnessExecutorRequired is a sentinel error.
	ErrHarnessExecutorRequired = errors.New("harness executor is required")
	// ErrEmptyInput is a sentinel error.
	ErrEmptyInput = errors.New("input cannot be empty")
	// ErrBlockedByHook is a sentinel error for hook-blocked turns.
	ErrBlockedByHook = errors.New("blocked by hook")
)

// HarnessTurnExecutor is the interface for the harness execution path.
// All conversations use this executor for turn execution.
type HarnessTurnExecutor interface {
	Execute(
		ctx context.Context,
		query string,
		history []message.Message,
	) (output string, messages []message.Message, err error)
}

// Conversation represents an active conversation instance.
type Conversation struct {
	// Services (optional, can be nil).
	gitService    *gitpkg.Service
	shellService  *shellpkg.Service
	mcpService    *mcppkg.Service
	memoryService *MemoryService

	// Core components.
	agent    *agent.Agent
	history  *history.History
	emitter  *events.EventEmitter
	taskMode string       // Current task mode (regular, review, compact, planning).
	taskMu   sync.RWMutex // Protects taskMode.
	id       string       // Unified conversation ID (for both session and protocol).
	workDir  string       // Working directory for this conversation.

	// Harness executor for turn execution (required).
	harnessExecutor HarnessTurnExecutor

	// Hook runner for lifecycle events (optional, nil = no hooks).
	hookRunner interface {
		Execute(ctx context.Context, event hooks.Event, evtCtx hooks.EventContext) hooks.HookResult
	}

	// Transcript writer for JSONL persistence (optional, nil = no persistence).
	transcriptWriter *session.TranscriptWriter

	// Session index for fast session listing (optional, nil = no index).
	sessionIndex *session.Index

	// sessionDir is the on-disk root for transcripts and the session index.
	sessionDir string

	// SubAgent manager for spawning specialized subagents (optional).
	subagentManager *subagent.Manager

	// A2A task registry for background children (optional).
	taskRegistry *tasks.Registry

	// Shell background processes for the unified /tasks view (optional).
	shellTasks *tools.ShellAdapter

	// Context retrieval pipeline for assembling context fragments (optional).
	retrievalPipeline *retrieval.Pipeline

	// LSP manager for code navigation (optional, closed on conversation close).
	lspManager *lsp.Manager

	// Protocol-specific fields (optional, for protocol use).
	turnID      string             // Current turn ID.
	cancel      context.CancelFunc // Cancellation context.
	transformer EventTransformer   // Optional event transformer for protocol adapters.
	protocolMu  sync.RWMutex       // Protects protocol fields (turnID, cancel, transformer).
	closeOnce   sync.Once          // SESSION_END / resource close run once (Ctrl-C + defer).
}

// generateConversationID creates a new unique conversation ID.
func generateConversationID() string {
	return uuid.New().String()
}

// NewFromAgentConfig holds configuration for NewFromAgent.
type NewFromAgentConfig struct {
	// Agent is the pre-built agent instance (required).
	Agent *agent.Agent

	// HarnessExecutor is the harness turn executor (required).
	HarnessExecutor HarnessTurnExecutor

	// Emitter is the event emitter (required).
	Emitter *events.EventEmitter

	// WorkDir is the working directory (required).
	WorkDir string

	// ID is an optional conversation ID (generated if empty).
	ID string

	// History is an optional pre-existing history (new one created if nil).
	History *history.History

	// MaxTokens is the maximum tokens for history (optional, uses default if 0).
	MaxTokens int
}

// NewFromAgent creates a Conversation from an existing agent and harness executor.
// This is useful for modes like ACP where the agent is pre-built.
func NewFromAgent(cfg NewFromAgentConfig) (*Conversation, error) {
	if cfg.Agent == nil {
		return nil, ErrAgentIsRequired
	}

	if cfg.HarnessExecutor == nil {
		return nil, ErrHarnessExecutorRequired
	}

	if cfg.Emitter == nil {
		return nil, ErrEmitterIsRequired
	}

	if cfg.WorkDir == "" {
		return nil, ErrWorkdirIsRequired
	}

	hist := cfg.History
	if hist == nil {
		if cfg.MaxTokens > 0 {
			hist = history.NewHistory(cfg.MaxTokens, &tokenizer.SimpleTokenizer{})
		} else {
			hist = history.NewHistoryWithDefaults()
		}
	}

	id := cfg.ID
	if id == "" {
		id = generateConversationID()
	}

	return &Conversation{
		agent:           cfg.Agent,
		harnessExecutor: cfg.HarnessExecutor,
		history:         hist,
		emitter:         cfg.Emitter,
		taskMode:        "regular",
		id:              id,
		workDir:         cfg.WorkDir,
		taskRegistry:    tasks.New(),
	}, nil
}

// RunTurn executes a single turn in the conversation via the harness executor.
func (c *Conversation) RunTurn(ctx context.Context, input string) error {
	if strings.TrimSpace(input) == "" {
		return ErrEmptyInput
	}

	// Fire USER_PROMPT_SUBMIT hook — may block the turn.
	if c.hookRunner != nil {
		evtCtx := hooks.EventContext{
			SessionID: c.id,
			WorkDir:   c.workDir,
		}

		result := c.hookRunner.Execute(ctx, hooks.EventUserPromptSubmit, evtCtx)
		if result.Blocked {
			return fmt.Errorf("%w: %s", ErrBlockedByHook, result.Reason)
		}
	}

	if c.emitter != nil {
		c.emitter.Emit(events.Event{
			Type: events.EventTurnStart,
			Data: events.TurnEventData{},
		})
	}

	historyMessages := c.history.MessagesForLLM()
	c.bindRetrievalPipeline()

	_, messages, execErr := c.harnessExecutor.Execute(ctx, input, historyMessages)
	if execErr != nil {
		if err := c.history.AddUserMessage(ctx, input); err != nil {
			return fmt.Errorf("failed to add user message: %w", err)
		}

		errorMsg := message.Message{
			Role:    message.RoleAssistant,
			Content: fmt.Sprintf("Error: %v", execErr),
		}
		_ = c.history.AddMessage(ctx, errorMsg)

		return fmt.Errorf("harness execution failed: %w", execErr)
	}

	for _, msg := range messages {
		if err := c.history.AddMessage(ctx, msg); err != nil {
			return fmt.Errorf("failed to add message to history: %w", err)
		}

		// Persist to transcript (best-effort).
		if c.transcriptWriter != nil {
			_ = c.transcriptWriter.Append(ctx, msg)
		}

		c.touchSessionIndex(ctx)
	}

	return nil
}

// SetTaskMode sets the task mode for the conversation.
func (c *Conversation) SetTaskMode(mode string) error {
	err := agent.ValidateMode(mode)
	if err != nil {
		return err
	}

	c.taskMu.Lock()
	c.taskMode = mode
	c.taskMu.Unlock()

	// Emit mode switch event.
	if c.emitter != nil {
		c.emitter.Emit(events.Event{
			Type: events.EventInfo,
			Data: events.SystemEventData{
				Message: fmt.Sprintf("Switched to %s mode", mode),
			},
		})
	}

	return nil
}

// GetTaskMode returns the current task mode for the conversation.
func (c *Conversation) GetTaskMode() string {
	c.taskMu.RLock()
	defer c.taskMu.RUnlock()

	return c.taskMode
}

// CurrentFrame returns the TaskFrame for the session's /mode value.
func (c *Conversation) CurrentFrame() frame.TaskFrame {
	return frame.FromMode(c.GetTaskMode())
}

// GetTokenCount returns the total number of tokens used in the conversation history.
func (c *Conversation) GetTokenCount() int {
	return c.history.TokenCount()
}

// GetHistoryMessages returns the conversation history formatted for LLM use.
func (c *Conversation) GetHistoryMessages() []message.Message {
	return c.history.MessagesForLLM()
}

// AddHistoryMessage adds a message to the conversation history.
func (c *Conversation) AddHistoryMessage(ctx context.Context, msg message.Message) error {
	return c.history.AddMessage(ctx, msg)
}

// ID returns the conversation ID as a string.
func (c *Conversation) ID() string {
	return c.id
}

// GetAgent returns the underlying agent instance.
// This is useful for modes like ACP that need direct access to the agent.
func (c *Conversation) GetAgent() *agent.Agent {
	return c.agent
}

// GetEmitter returns the event emitter for this conversation.
// This is useful for modes like ACP that need direct access to the emitter.
func (c *Conversation) GetEmitter() *events.EventEmitter {
	return c.emitter
}

// SetID sets the conversation ID.
func (c *Conversation) SetID(id string) {
	c.id = id
}

// Stream returns the event stream for this conversation.
func (c *Conversation) Stream() <-chan events.Event {
	return c.emitter.Events()
}

// Close cancels running A2A tasks, fires STOP then SESSION_END, and
// cleans up resources. Services (git, shell, mcp) are owned by the
// application layer and are NOT closed here.
func (c *Conversation) Close(ctx context.Context) error {
	var err error

	c.closeOnce.Do(func() {
		_ = c.taskRegistry.CancelAll(context.WithoutCancel(ctx))
		c.fireStopThenSessionEnd(ctx)

		if c.lspManager != nil {
			_ = c.lspManager.Close(ctx)
		}

		if c.transcriptWriter != nil {
			err = c.transcriptWriter.Close()
		}
	})

	return err
}

func (c *Conversation) fireStopThenSessionEnd(ctx context.Context) {
	if c.hookRunner == nil {
		return
	}

	hookCtx := context.WithoutCancel(ctx)
	evtCtx := hooks.EventContext{SessionID: c.id, WorkDir: c.workDir}
	c.hookRunner.Execute(hookCtx, hooks.EventStop, evtCtx)
	c.hookRunner.Execute(hookCtx, hooks.EventSessionEnd, evtCtx)
}

// GetTurnID returns the current turn ID.
func (c *Conversation) GetTurnID() string {
	c.protocolMu.RLock()
	defer c.protocolMu.RUnlock()

	return c.turnID
}

// SetTurnID sets the current turn ID (thread-safe).
func (c *Conversation) SetTurnID(turnID string) {
	c.protocolMu.Lock()
	defer c.protocolMu.Unlock()

	c.turnID = turnID
}

// GetCancel returns the cancellation context function.
func (c *Conversation) GetCancel() context.CancelFunc {
	c.protocolMu.RLock()
	defer c.protocolMu.RUnlock()

	return c.cancel
}

// SetCancel sets the cancellation context function (thread-safe).
func (c *Conversation) SetCancel(cancel context.CancelFunc) {
	c.protocolMu.Lock()
	defer c.protocolMu.Unlock()

	c.cancel = cancel
}

// Cancel cancels the current turn (thread-safe).
func (c *Conversation) Cancel() {
	c.protocolMu.RLock()
	cancel := c.cancel
	c.protocolMu.RUnlock()

	if cancel != nil {
		cancel()
	}
}

// SetEventTransformer sets an event transformer for protocol-specific event handling.
// The transformer receives all events and can transform them to protocol-specific formats.
// Pass nil to remove the transformer.
func (c *Conversation) SetEventTransformer(transformer EventTransformer) {
	c.protocolMu.Lock()
	defer c.protocolMu.Unlock()

	c.transformer = transformer
}

// GetEventTransformer returns the current event transformer, if any.
func (c *Conversation) GetEventTransformer() EventTransformer {
	c.protocolMu.RLock()
	defer c.protocolMu.RUnlock()

	return c.transformer
}

// GetHistory returns the conversation's history instance.
// This is useful for persistence operations.
func (c *Conversation) GetHistory() *history.History {
	return c.history
}

// GetWorkDir returns the working directory for this conversation.
func (c *Conversation) GetWorkDir() string {
	return c.workDir
}

// GetMemoryService returns the memory service for this conversation.
// Returns nil if memory is not enabled.
func (c *Conversation) GetMemoryService() *MemoryService {
	return c.memoryService
}

// GetSubagentManager returns the subagent manager for spawning specialized subagents.
// Returns nil if subagents are not available.
func (c *Conversation) GetSubagentManager() *subagent.Manager {
	return c.subagentManager
}

// GetTaskRegistry returns the A2A task registry for wait/list/cancel.
func (c *Conversation) GetTaskRegistry() *tasks.Registry {
	if c == nil {
		return nil
	}

	return c.taskRegistry
}

// GetShellTasks returns the shell background adapter for the unified /tasks view.
func (c *Conversation) GetShellTasks() *tools.ShellAdapter {
	if c == nil {
		return nil
	}

	return c.shellTasks
}

// GetRetrievalPipeline returns the context retrieval pipeline.
// Returns nil if no retrieval sources are configured.
func (c *Conversation) GetRetrievalPipeline() *retrieval.Pipeline {
	return c.retrievalPipeline
}

type retrievalBinder interface {
	BindRetrieval(p *retrieval.Pipeline)
}

func (c *Conversation) bindRetrievalPipeline() {
	p := c.GetRetrievalPipeline()

	binder, ok := c.harnessExecutor.(retrievalBinder)
	if !ok {
		return
	}

	binder.BindRetrieval(p)
}

// GetSessionIndex returns the session index for session management operations
// (List, Remove, Count). Returns nil if index is unavailable.
func (c *Conversation) GetSessionIndex() *session.Index {
	return c.sessionIndex
}
