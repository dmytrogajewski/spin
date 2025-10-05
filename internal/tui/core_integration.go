package tui

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/llm"
)

// CoreManager wraps core functionality for TUI.
// It manages the Manager and active Conversation, handling lifecycle and communication.
type CoreManager struct {
	manager *core.Manager
	conv    *core.Conversation
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewCoreManager creates a new CoreManager for TUI.
// It initializes the core Manager with the given configuration and provider.
func NewCoreManager(cfg *core.Config, provider llm.Provider) (*CoreManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create manager with provider
	mgr, err := core.NewManager(
		cfg,
		core.WithLLM(provider),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create manager: %w", err)
	}

	return &CoreManager{
		manager: mgr,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// StartConversation creates a new conversation and returns the event channel.
func (cm *CoreManager) StartConversation() (<-chan core.Event, error) {
	// Use empty string to use default workdir from config
	conv, err := cm.manager.NewConversation(cm.ctx, "")
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}

	cm.conv = conv
	return conv.Stream(), nil
}

// SendMessage sends a user message to the active conversation.
func (cm *CoreManager) SendMessage(content string) error {
	if cm.conv == nil {
		return fmt.Errorf("no active conversation")
	}

	return cm.conv.RunTurn(cm.ctx, content)
}

// Stop cancels the current turn.
func (cm *CoreManager) Stop() error {
	if cm.conv == nil {
		return nil
	}
	return cm.conv.Stop(cm.ctx)
}

// Pause pauses the current turn.
func (cm *CoreManager) Pause() error {
	if cm.conv == nil {
		return nil
	}
	return cm.conv.Pause()
}

// Resume resumes a paused turn.
func (cm *CoreManager) Resume() error {
	if cm.conv == nil {
		return nil
	}
	return cm.conv.Resume()
}

// Close cleanup resources.
func (cm *CoreManager) Close() error {
	cm.cancel()
	// Note: Manager doesn't have a Close method in current implementation
	// If it's added later, uncomment:
	// if cm.manager != nil {
	//     return cm.manager.Close()
	// }
	return nil
}
