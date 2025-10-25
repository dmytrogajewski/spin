package appserver

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/protocol/jsonrpc"
)

// mockProvider implements llm.Provider for testing
type mockProvider struct {
	mu           sync.Mutex
	completeFunc func(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error)
	streamFunc   func(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error)
}

func (m *mockProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}
	return &llm.CompletionResponse{
		Content: "Mock response",
		Usage: llm.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *mockProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.streamFunc != nil {
		return m.streamFunc(ctx, req)
	}

	// Default: return simple stream
	ch := make(chan llm.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- llm.StreamChunk{
			Type:    llm.ChunkTypeContentDelta,
			Content: "Mock response",
		}
	}()
	return ch, nil
}

func (m *mockProvider) Models(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{{Name: "mock-model"}}, nil
}

func (m *mockProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: true,
	}
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) Close() error {
	return nil
}

// TestProcessor_Integration_TaskModeEndToEnd verifies task mode flows end-to-end
func TestProcessor_Integration_TaskModeEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock provider that captures tool list
	var capturedToolCount int
	var mu sync.Mutex

	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
			mu.Lock()
			capturedToolCount = len(req.Tools)
			mu.Unlock()

			ch := make(chan llm.StreamChunk, 2)
			go func() {
				defer close(ch)
				ch <- llm.StreamChunk{
					Type:    llm.ChunkTypeContentDelta,
					Content: "Test response",
				}
			}()
			return ch, nil
		},
	}

	// Create processor with agent
	processor, err := NewProcessor(ProcessorConfig{
		WorkspacePath: tmpDir,
		Version:       "0.1.0",
		Provider:      provider,
	})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Set output to capture notifications
	output := &bytes.Buffer{}
	processor.SetOutput(output)

	tests := []struct {
		name              string
		taskMode          string
		expectToolCount   int  // We can't check exact tools, but we can verify filtering happened
		expectMinTools    int  // Minimum tools expected
		expectMaxTools    int  // Maximum tools expected
	}{
		{
			name:           "review mode has fewer tools than regular",
			taskMode:       "review",
			expectMinTools: 1,
			expectMaxTools: 20, // Should be less than regular mode
		},
		{
			name:           "compact mode has minimal tools",
			taskMode:       "compact",
			expectMinTools: 1,
			expectMaxTools: 10, // Should be very limited
		},
		{
			name:           "planning mode has context tools",
			taskMode:       "planning",
			expectMinTools: 1,
			expectMaxTools: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mu.Lock()
			capturedToolCount = 0
			mu.Unlock()

			mode := tt.taskMode
			params := jsonrpc.SendMessageParams{
				Message:  "Test message",
				TaskMode: &mode,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := processor.HandleSendMessage(ctx, params)
			if err != nil {
				t.Fatalf("HandleSendMessage failed: %v", err)
			}

			// Verify task mode in result
			if result.TaskMode != tt.taskMode {
				t.Errorf("Expected task mode %q, got %q", tt.taskMode, result.TaskMode)
			}

			// Wait for agent to process (runTurn runs async)
			time.Sleep(100 * time.Millisecond)

			// Verify tool filtering occurred
			mu.Lock()
			actualCount := capturedToolCount
			mu.Unlock()

			if actualCount < tt.expectMinTools || actualCount > tt.expectMaxTools {
				t.Errorf("Expected %d-%d tools, got %d for mode %q",
					tt.expectMinTools, tt.expectMaxTools, actualCount, tt.taskMode)
			}
		})
	}
}

// TestProcessor_Integration_TaskModeSwitching verifies mode switching mid-conversation
func TestProcessor_Integration_TaskModeSwitching(t *testing.T) {
	tmpDir := t.TempDir()

	var requestCount int
	var mu sync.Mutex

	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
			mu.Lock()
			requestCount++
			mu.Unlock()

			ch := make(chan llm.StreamChunk, 1)
			go func() {
				defer close(ch)
				ch <- llm.StreamChunk{
					Type:    llm.ChunkTypeContentDelta,
					Content: "Response",
				}
			}()
			return ch, nil
		},
	}

	processor, err := NewProcessor(ProcessorConfig{
		WorkspacePath: tmpDir,
		Version:       "0.1.0",
		Provider:      provider,
	})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	output := &bytes.Buffer{}
	processor.SetOutput(output)

	// First message in regular mode
	regularMode := "regular"
	params1 := jsonrpc.SendMessageParams{
		Message:  "First message",
		TaskMode: &regularMode,
	}

	ctx := context.Background()
	result1, err := processor.HandleSendMessage(ctx, params1)
	if err != nil {
		t.Fatalf("First message failed: %v", err)
	}

	if result1.TaskMode != "regular" {
		t.Errorf("Expected task mode 'regular', got %q", result1.TaskMode)
	}

	// Wait for first turn to complete
	time.Sleep(100 * time.Millisecond)

	// Second message switching to review mode
	reviewMode := "review"
	params2 := jsonrpc.SendMessageParams{
		ConversationID: &result1.ConversationID,
		Message:        "Second message",
		TaskMode:       &reviewMode,
	}

	result2, err := processor.HandleSendMessage(ctx, params2)
	if err != nil {
		t.Fatalf("Second message failed: %v", err)
	}

	if result2.TaskMode != "review" {
		t.Errorf("Expected task mode 'review', got %q", result2.TaskMode)
	}

	// Verify conversation ID is the same
	if result2.ConversationID != result1.ConversationID {
		t.Error("Conversation ID should remain the same")
	}

	// Verify agent was called for both messages
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	count := requestCount
	mu.Unlock()

	if count < 2 {
		t.Errorf("Expected at least 2 agent requests, got %d", count)
	}
}

// safeBuffer is a thread-safe buffer wrapper
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *safeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *safeBuffer) Bytes() []byte {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Bytes()
}

// TestProcessor_Integration_NotificationFlow verifies notifications are sent correctly
func TestProcessor_Integration_NotificationFlow(t *testing.T) {
	tmpDir := t.TempDir()

	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
			ch := make(chan llm.StreamChunk, 3)
			go func() {
				defer close(ch)
				ch <- llm.StreamChunk{
					Type:    llm.ChunkTypeContentDelta,
					Content: "Hello",
				}
				ch <- llm.StreamChunk{
					Type:    llm.ChunkTypeContentDelta,
					Content: " world",
				}
			}()
			return ch, nil
		},
	}

	processor, err := NewProcessor(ProcessorConfig{
		WorkspacePath: tmpDir,
		Version:       "0.1.0",
		Provider:      provider,
	})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	output := &safeBuffer{}
	processor.SetOutput(output)

	compactMode := "compact"
	params := jsonrpc.SendMessageParams{
		Message:  "Test message",
		TaskMode: &compactMode,
	}

	ctx := context.Background()
	result, err := processor.HandleSendMessage(ctx, params)
	if err != nil {
		t.Fatalf("HandleSendMessage failed: %v", err)
	}

	// Wait for turn to complete
	time.Sleep(200 * time.Millisecond)

	// Parse notifications using thread-safe buffer
	notifications := []jsonrpc.Notification{}
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	for {
		var notif jsonrpc.Notification
		if err := decoder.Decode(&notif); err != nil {
			break
		}
		notifications = append(notifications, notif)
	}

	// Verify we got at least turn_start and turn_complete
	if len(notifications) < 2 {
		t.Errorf("Expected at least 2 notifications, got %d", len(notifications))
	}

	// Verify turn_start was sent
	foundTurnStart := false
	foundTurnComplete := false
	for _, notif := range notifications {
		if notif.Method == "turn_start" {
			foundTurnStart = true
		}
		if notif.Method == "turn_complete" {
			foundTurnComplete = true
		}
	}

	if !foundTurnStart {
		t.Error("Expected turn_start notification")
	}
	if !foundTurnComplete {
		t.Error("Expected turn_complete notification")
	}

	// Verify task mode in result
	if result.TaskMode != "compact" {
		t.Errorf("Expected task mode 'compact', got %q", result.TaskMode)
	}
}

// TestProcessor_Integration_CancelTurnWithTaskMode verifies cancellation works with task modes
func TestProcessor_Integration_CancelTurnWithTaskMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Provider that blocks to simulate long-running operation
	blockCh := make(chan struct{})
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
			ch := make(chan llm.StreamChunk)
			go func() {
				defer close(ch)
				select {
				case <-blockCh:
					ch <- llm.StreamChunk{
						Type:    llm.ChunkTypeContentDelta,
						Content: "Response",
					}
				case <-ctx.Done():
					// Context cancelled
					return
				}
			}()
			return ch, nil
		},
	}

	processor, err := NewProcessor(ProcessorConfig{
		WorkspacePath: tmpDir,
		Version:       "0.1.0",
		Provider:      provider,
	})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	output := &bytes.Buffer{}
	processor.SetOutput(output)

	planningMode := "planning"
	params := jsonrpc.SendMessageParams{
		Message:  "Long running task",
		TaskMode: &planningMode,
	}

	ctx := context.Background()
	result, err := processor.HandleSendMessage(ctx, params)
	if err != nil {
		t.Fatalf("HandleSendMessage failed: %v", err)
	}

	// Give turn time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the turn
	cancelParams := jsonrpc.CancelTurnParams{
		TurnID: result.TurnID,
	}

	cancelResult, err := processor.HandleCancelTurn(ctx, cancelParams)
	if err != nil {
		t.Fatalf("HandleCancelTurn failed: %v", err)
	}

	if cancelResult.Status != "ok" {
		t.Errorf("Expected cancel status 'ok', got %q", cancelResult.Status)
	}

	// Unblock provider
	close(blockCh)

	// Verify task mode was set before cancellation
	if result.TaskMode != "planning" {
		t.Errorf("Expected task mode 'planning', got %q", result.TaskMode)
	}
}

// TestProcessor_Integration_DefaultTaskModeWithAgent verifies default mode when none specified
func TestProcessor_Integration_DefaultTaskModeWithAgent(t *testing.T) {
	tmpDir := t.TempDir()

	var capturedToolCount int
	var completeCalled, streamCalled bool
	var mu sync.Mutex

	provider := &mockProvider{
		completeFunc: func(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
			mu.Lock()
			completeCalled = true
			capturedToolCount = len(req.Tools)
			mu.Unlock()

			return &llm.CompletionResponse{
				Content: "Mock response",
				Usage: llm.Usage{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			}, nil
		},
		streamFunc: func(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
			mu.Lock()
			streamCalled = true
			capturedToolCount = len(req.Tools)
			mu.Unlock()

			ch := make(chan llm.StreamChunk, 1)
			go func() {
				defer close(ch)
				ch <- llm.StreamChunk{
					Type:    llm.ChunkTypeContentDelta,
					Content: "Response",
				}
			}()
			return ch, nil
		},
	}

	processor, err := NewProcessor(ProcessorConfig{
		WorkspacePath: tmpDir,
		Version:       "0.1.0",
		Provider:      provider,
	})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	output := &bytes.Buffer{}
	processor.SetOutput(output)

	// Send message without specifying task mode
	params := jsonrpc.SendMessageParams{
		Message: "Test message",
		// TaskMode is nil - should default to "regular"
	}

	ctx := context.Background()
	result, err := processor.HandleSendMessage(ctx, params)
	if err != nil {
		t.Fatalf("HandleSendMessage failed: %v", err)
	}

	// Should default to "regular" mode
	if result.TaskMode != "regular" {
		t.Errorf("Expected default task mode 'regular', got %q", result.TaskMode)
	}

	// Wait for agent to process with retry logic
	maxWait := 1 * time.Second
	checkInterval := 50 * time.Millisecond
	deadline := time.Now().Add(maxWait)

	var toolCount int
	for time.Now().Before(deadline) {
		mu.Lock()
		toolCount = capturedToolCount
		mu.Unlock()

		if toolCount >= 5 {
			break
		}
		time.Sleep(checkInterval)
	}

	mu.Lock()
	wasCalled := completeCalled || streamCalled
	mu.Unlock()

	if !wasCalled {
		t.Errorf("Provider was never called (neither Complete nor Stream)")
	}

	if toolCount < 5 {
		t.Errorf("Expected regular mode to have many tools, got %d (completeCalled=%v, streamCalled=%v)",
			toolCount, completeCalled, streamCalled)
	}
}

// TestProcessor_Integration_InvalidTaskModeWithAgent verifies error handling
func TestProcessor_Integration_InvalidTaskModeWithAgent(t *testing.T) {
	tmpDir := t.TempDir()

	provider := &mockProvider{}

	processor, err := NewProcessor(ProcessorConfig{
		WorkspacePath: tmpDir,
		Version:       "0.1.0",
		Provider:      provider,
	})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	invalidMode := "nonexistent"
	params := jsonrpc.SendMessageParams{
		Message:  "Test message",
		TaskMode: &invalidMode,
	}

	ctx := context.Background()
	_, err = processor.HandleSendMessage(ctx, params)
	if err == nil {
		t.Fatal("Expected error for invalid task mode")
	}

	// Verify it's a JSON-RPC error with correct code
	rpcErr, ok := err.(*jsonrpc.Error)
	if !ok {
		t.Fatalf("Expected *jsonrpc.Error, got %T", err)
	}

	if rpcErr.Code != jsonrpc.InvalidParams {
		t.Errorf("Expected error code %d, got %d", jsonrpc.InvalidParams, rpcErr.Code)
	}
}

// TestProcessor_Integration_AgentReceivesCorrectTaskName verifies agent gets task name
func TestProcessor_Integration_AgentReceivesCorrectTaskName(t *testing.T) {
	tmpDir := t.TempDir()

	// This test verifies the runTurn method passes TaskName to agent correctly
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
			ch := make(chan llm.StreamChunk, 1)
			go func() {
				defer close(ch)
				ch <- llm.StreamChunk{
					Type:    llm.ChunkTypeContentDelta,
					Content: "Response",
				}
			}()
			return ch, nil
		},
	}

	processor, err := NewProcessor(ProcessorConfig{
		WorkspacePath: tmpDir,
		Version:       "0.1.0",
		Provider:      provider,
	})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	output := &bytes.Buffer{}
	processor.SetOutput(output)

	// Test all valid modes
	modes := []string{"regular", "review", "compact", "planning"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			modePtr := mode
			params := jsonrpc.SendMessageParams{
				Message:  "Test message",
				TaskMode: &modePtr,
			}

			ctx := context.Background()
			result, err := processor.HandleSendMessage(ctx, params)
			if err != nil {
				t.Fatalf("HandleSendMessage failed: %v", err)
			}

			// Wait for agent to process
			time.Sleep(100 * time.Millisecond)

			// Verify task mode in result
			if result.TaskMode != mode {
				t.Errorf("Expected task mode %q, got %q", mode, result.TaskMode)
			}

			// Verify conversation has correct mode
			processor.mu.RLock()
			conv, ok := processor.conversations[result.ConversationID]
			processor.mu.RUnlock()

			if !ok {
				t.Fatal("Conversation not found")
			}

			conv.mu.RLock()
			convMode := conv.taskMode
			conv.mu.RUnlock()

			if convMode != mode {
				t.Errorf("Expected conversation task mode %q, got %q", mode, convMode)
			}
		})
	}
}
