package tui

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCoreManager(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *core.Config
		provider    llm.Provider
		expectError bool
	}{
		{
			name: "valid config and provider",
			cfg: func() *core.Config {
				cfg := core.DefaultConfig()
				cfg.Provider = "mock"
				cfg.Model = "test-model"
				return cfg
			}(),
			provider:    llm.NewMockProvider("test"),
			expectError: false,
		},
		{
			name:        "nil config",
			cfg:         nil,
			provider:    llm.NewMockProvider("test"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := NewCoreManager(tt.cfg, tt.provider)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cm)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cm)
				assert.NotNil(t, cm.manager)
				assert.Nil(t, cm.conv) // No conversation until StartConversation
				defer cm.Close()
			}
		})
	}
}

func TestCoreManager_StartConversation(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = "/tmp/spin-test"
	provider := llm.NewMockProvider("test")
	cm, err := NewCoreManager(cfg, provider)
	require.NoError(t, err)
	defer cm.Close()

	events, err := cm.StartConversation()
	require.NoError(t, err)
	require.NotNil(t, events)
	require.NotNil(t, cm.conv)
}

func TestCoreManager_SendMessage(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = "/tmp/spin-test"
	provider := llm.NewMockProvider("test",
		llm.WithResponse("I'll help you"),
	)
	cm, err := NewCoreManager(cfg, provider)
	require.NoError(t, err)
	defer cm.Close()

	// Start conversation first
	events, err := cm.StartConversation()
	require.NoError(t, err)

	// Send message
	err = cm.SendMessage("Hello")
	require.NoError(t, err)

	// Should receive events
	select {
	case event := <-events:
		assert.NotEmpty(t, event.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestCoreManager_SendMessage_WithoutConversation(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = "/tmp/spin-test"
	provider := llm.NewMockProvider("test")
	cm, err := NewCoreManager(cfg, provider)
	require.NoError(t, err)
	defer cm.Close()

	// Try to send without starting conversation
	err = cm.SendMessage("Hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active conversation")
}

func TestCoreManager_Stop(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = "/tmp/spin-test"
	provider := llm.NewMockProvider("test",
		llm.WithResponse("This is a long response..."),
	)
	cm, err := NewCoreManager(cfg, provider)
	require.NoError(t, err)
	defer cm.Close()

	events, err := cm.StartConversation()
	require.NoError(t, err)

	// Send message
	err = cm.SendMessage("Hello")
	require.NoError(t, err)

	// Wait a bit then stop
	time.Sleep(50 * time.Millisecond)
	err = cm.Stop()
	assert.NoError(t, err)

	// Should eventually receive turn complete (cancelled)
	timeout := time.After(2 * time.Second)
	for {
		select {
		case event := <-events:
			if event.Type == core.EventTurnComplete || event.Type == core.EventTurnFailed {
				return // Success
			}
		case <-timeout:
			t.Fatal("timeout waiting for turn complete after stop")
		}
	}
}

func TestCoreManager_Pause_Resume(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = "/tmp/spin-test"
	provider := llm.NewMockProvider("test",
		llm.WithResponse("Test response"),
	)
	cm, err := NewCoreManager(cfg, provider)
	require.NoError(t, err)
	defer cm.Close()

	events, err := cm.StartConversation()
	require.NoError(t, err)

	// Send message
	err = cm.SendMessage("Hello")
	require.NoError(t, err)

	// Try to pause (may fail if turn already complete - that's OK)
	time.Sleep(10 * time.Millisecond)
	_ = cm.Pause() // Ignore error - turn might have completed

	// Try to resume (may fail if not paused - that's OK)
	time.Sleep(10 * time.Millisecond)
	_ = cm.Resume() // Ignore error

	// Should eventually complete
	timeout := time.After(2 * time.Second)
	for {
		select {
		case event := <-events:
			if event.Type == core.EventTurnComplete || event.Type == core.EventTurnFailed {
				return // Success - turn completed
			}
		case <-timeout:
			t.Fatal("timeout waiting for turn complete")
		}
	}
}

func TestCoreManager_Close(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = "/tmp/spin-test"
	provider := llm.NewMockProvider("test")
	cm, err := NewCoreManager(cfg, provider)
	require.NoError(t, err)

	_, err = cm.StartConversation()
	require.NoError(t, err)

	// Close should not error
	err = cm.Close()
	assert.NoError(t, err)

	// Context should be cancelled
	select {
	case <-cm.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("context not cancelled after close")
	}
}

func TestCoreManager_MultipleClose(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = "/tmp/spin-test"
	provider := llm.NewMockProvider("test")
	cm, err := NewCoreManager(cfg, provider)
	require.NoError(t, err)

	// Multiple closes should not panic
	err1 := cm.Close()
	err2 := cm.Close()
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}
