package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// statefulMockProvider is a test provider that changes behavior based on call count
type statefulMockProvider struct {
	name         string
	callCount    int
	verifyFunc   func(t *testing.T, callNum int, req llm.CompletionRequest) *llm.CompletionResponse
	capabilities llm.Capabilities
}

func (p *statefulMockProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return nil, fmt.Errorf("Complete not implemented, use Stream")
}

func (p *statefulMockProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	p.callCount++
	resp := p.verifyFunc(nil, p.callCount, req)

	chunks := make(chan llm.StreamChunk, 10)
	go func() {
		defer close(chunks)

		if resp.Content != "" {
			chunks <- llm.StreamChunk{
				Type:    llm.ChunkTypeContentDelta,
				Content: resp.Content,
			}
		}

		for _, tc := range resp.ToolCalls {
			tcCopy := tc
			chunks <- llm.StreamChunk{
				Type:     llm.ChunkTypeToolCallStart,
				ToolCall: &tcCopy,
			}
		}

		chunks <- llm.StreamChunk{
			Type:         llm.ChunkTypeDone,
			FinishReason: resp.FinishReason,
		}
	}()

	return chunks, nil
}

func (p *statefulMockProvider) Models(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{{ID: "test-model", Name: "Test Model", Description: "Test", ContextSize: 4096}}, nil
}

func (p *statefulMockProvider) Capabilities() llm.Capabilities {
	return p.capabilities
}

func (p *statefulMockProvider) Name() string {
	return p.name
}

func (p *statefulMockProvider) Close() error {
	return nil
}

// TestConversation_ToolMessagesInHistory verifies that tool call and result messages
// are properly added to conversation history (BUG-20251014030000 fix).
func TestConversation_ToolMessagesInHistory(t *testing.T) {
	// Setup
	cfg := &Config{
		Provider:    "test",
		Model:       "test-model",
		WorkDir:     t.TempDir(),
		SessionDir:  t.TempDir(),
		MaxTurns:    10,
		Timeout:     30 * time.Second,
		SandboxMode: "workspace-only",
		MaxTokens:   4096,
	}

	// Create a stateful provider that returns a tool call on first turn
	// and a text response on second turn (after receiving tool result)
	mockProvider := &statefulMockProvider{
		name: "test-provider",
		capabilities: llm.Capabilities{
			Streaming:       true,
			FunctionCalling: true,
		},
		verifyFunc: func(testRef *testing.T, callNum int, req llm.CompletionRequest) *llm.CompletionResponse {
			if callNum == 1 {
				// First turn: request to use write_file tool
				return &llm.CompletionResponse{
					Content: "I'll write the file now.",
					ToolCalls: []llm.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "write_file",
								Arguments: `{"path":"test.txt","content":"Hello, World!"}`,
							},
						},
					},
					Usage: llm.Usage{
						PromptTokens:     100,
						CompletionTokens: 20,
						TotalTokens:      120,
					},
					FinishReason: "tool_calls",
				}
			}

			// Second turn: verify we received tool result in history
			foundToolResult := false
			for _, msg := range req.Messages {
				if msg.Role == "tool" && msg.ToolCallID == "call_1" {
					foundToolResult = true
					// Verify tool result contains success message
					if msg.Content == "" {
						t.Error("Tool result message has empty content")
					}
					break
				}
			}
			if !foundToolResult {
				t.Error("Tool result message not found in second turn history")
			}

			// Return final response
			return &llm.CompletionResponse{
				Content:      "File written successfully!",
				Usage:        llm.Usage{TotalTokens: 50},
				FinishReason: "stop",
			}
		},
	}

	emitter := NewEventEmitter(100)
	mgr, err := NewManager(cfg, WithLLM(mockProvider), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, cfg.WorkDir)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	// Execute a turn that uses a tool
	err = conv.RunTurn(ctx, "Write a test file")
	if err != nil {
		t.Fatalf("failed to run turn: %v", err)
	}

	// Small delay to ensure all messages are processed
	time.Sleep(50 * time.Millisecond)

	// Verify history contains all expected messages
	messages := conv.history.Messages()

	// Debug: print all messages
	t.Logf("History has %d messages:", len(messages))
	for i, msg := range messages {
		t.Logf("  [%d] role=%s content=%q tool_calls=%d tool_call_id=%q",
			i, msg.Role, msg.Content, len(msg.ToolCalls), msg.ToolCallID)
	}

	// Expected structure:
	// 1. System message
	// 2. User message: "Write a test file"
	// 3. Assistant message with tool_calls
	// 4. Tool result message
	// 5. Assistant message: "File written successfully!"

	if len(messages) < 5 {
		t.Fatalf("expected at least 5 messages in history, got %d", len(messages))
	}

	// Verify user message
	if messages[1].Role != RoleUser || messages[1].Content != "Write a test file" {
		t.Errorf("message[1] should be user message, got role=%s content=%s",
			messages[1].Role, messages[1].Content)
	}

	// Verify assistant message with tool calls
	foundToolCall := false
	toolCallIndex := -1
	for i := 2; i < len(messages); i++ {
		if messages[i].Role == RoleAssistant && len(messages[i].ToolCalls) > 0 {
			foundToolCall = true
			toolCallIndex = i
			if messages[i].ToolCalls[0].Function.Name != "write_file" {
				t.Errorf("expected write_file tool call, got %s",
					messages[i].ToolCalls[0].Function.Name)
			}
			break
		}
	}
	if !foundToolCall {
		t.Error("assistant message with tool calls not found in history")
	}

	// Verify tool result message follows assistant message with tool call
	if toolCallIndex >= 0 && toolCallIndex+1 < len(messages) {
		toolResultMsg := messages[toolCallIndex+1]
		if toolResultMsg.Role != RoleTool {
			t.Errorf("expected tool result message after assistant tool call, got role=%s",
				toolResultMsg.Role)
		}
		if toolResultMsg.ToolCallID != "call_1" {
			t.Errorf("expected tool_call_id='call_1', got '%s'",
				toolResultMsg.ToolCallID)
		}
		if toolResultMsg.Content == "" {
			t.Error("tool result message has empty content")
		}
	} else {
		t.Error("tool result message not found after tool call")
	}

	// Verify final assistant message
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != RoleAssistant {
		t.Errorf("expected final message to be assistant, got %s", lastMsg.Role)
	}
	if lastMsg.Content != "File written successfully!" {
		t.Errorf("expected final content, got: %s", lastMsg.Content)
	}

	t.Logf("✓ All %d messages properly added to history", len(messages))
}

// TestConversation_MultipleToolCalls verifies that multiple tool calls
// in a single turn are all properly added to history.
func TestConversation_MultipleToolCalls(t *testing.T) {
	// Setup
	cfg := &Config{
		Provider:    "test",
		Model:       "test-model",
		WorkDir:     t.TempDir(),
		SessionDir:  t.TempDir(),
		MaxTurns:    10,
		Timeout:     30 * time.Second,
		SandboxMode: "workspace-only",
		MaxTokens:   4096,
	}

	mockProvider := &statefulMockProvider{
		name: "test-provider",
		capabilities: llm.Capabilities{
			Streaming:       true,
			FunctionCalling: true,
		},
		verifyFunc: func(testRef *testing.T, callNum int, req llm.CompletionRequest) *llm.CompletionResponse {
			// Check if this is the first call (no tool results in history)
			hasToolResults := false
			for _, msg := range req.Messages {
				if msg.Role == "tool" {
					hasToolResults = true
					break
				}
			}

			if !hasToolResults {
				// First turn: request multiple tools
				return &llm.CompletionResponse{
					Content: "I'll read and write files.",
					ToolCalls: []llm.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "read_file",
								Arguments: `{"path":"input.txt"}`,
							},
						},
						{
							ID:   "call_2",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "write_file",
								Arguments: `{"path":"output.txt","content":"data"}`,
							},
						},
					},
					Usage:        llm.Usage{TotalTokens: 120},
					FinishReason: "tool_calls",
				}
			}

			// Second turn: verify we have both tool results
			toolResults := 0
			for _, msg := range req.Messages {
				if msg.Role == "tool" {
					toolResults++
				}
			}
			if toolResults != 2 {
				t.Errorf("expected 2 tool results in history, found %d", toolResults)
			}

			return &llm.CompletionResponse{
				Content:      "Done processing files.",
				Usage:        llm.Usage{TotalTokens: 50},
				FinishReason: "stop",
			}
		},
	}

	emitter := NewEventEmitter(100)
	mgr, err := NewManager(cfg, WithLLM(mockProvider), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, cfg.WorkDir)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	// Execute turn with multiple tool calls
	err = conv.RunTurn(ctx, "Process files")
	if err != nil {
		t.Fatalf("failed to run turn: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Verify history
	messages := conv.history.Messages()

	// Count tool result messages
	toolResultCount := 0
	for _, msg := range messages {
		if msg.Role == RoleTool {
			toolResultCount++
		}
	}

	if toolResultCount != 2 {
		t.Errorf("expected 2 tool result messages in history, got %d", toolResultCount)
	}

	t.Logf("✓ Multiple tool calls properly tracked in history")
}
