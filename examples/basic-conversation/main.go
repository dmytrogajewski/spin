// Package main demonstrates basic conversation usage with the Spin core package.
//
// This example shows how to:
//   - Create a conversation manager
//   - Start a conversation
//   - Send messages to the agent
//   - Stream events and responses
//
// Run this example:
//
//	go run examples/basic-conversation/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func main() {
	// Create configuration
	cfg := &core.Config{
		MaxTurns:    10,
		Temperature: 0.7,
		MaxTokens:   2048,
		Model:       "claude-3-5-sonnet-20241022",
		Debug:       true,
	}

	// Create a mock LLM provider for demonstration
	// In production, use a real provider like Anthropic
	provider := &MockLLMProvider{}

	// Create tool registry
	registry := tools.NewRegistry()

	// Create manager with options
	mgr, err := core.NewManager(
		cfg,
		core.WithLLMProvider(provider),
		core.WithToolRegistry(registry),
	)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Start a new conversation
	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx)
	if err != nil {
		log.Fatalf("Failed to create conversation: %v", err)
	}

	fmt.Println("🤖 Spin AI Agent - Basic Conversation Example")
	fmt.Println("=" + string(make([]byte, 48)))
	fmt.Println()

	// Send a message and handle streaming events
	userMessage := "Hello! Can you help me understand what you can do?"

	fmt.Printf("👤 User: %s\n\n", userMessage)
	fmt.Println("🤖 Assistant:")

	// Stream the response
	eventChan, err := conv.SendMessage(ctx, userMessage)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	// Process events
	for event := range eventChan {
		switch event.Type {
		case core.EventTypeStreamContent:
			// Print content as it streams
			fmt.Print(event.Data)

		case core.EventTypeTurnComplete:
			// Turn completed
			fmt.Println("\n\n✓ Turn completed")

		case core.EventTypeError:
			// Handle error
			fmt.Fprintf(os.Stderr, "\n✗ Error: %v\n", event.Data)
		}
	}

	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("Example completed successfully!")
}

// MockLLMProvider is a simple mock for demonstration purposes
type MockLLMProvider struct{}

func (m *MockLLMProvider) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	// In a real implementation, this would call an actual LLM API
	return &llm.CompletionResponse{
		Content: "Hello! I'm Spin, an AI coding agent. I can help you with:\n" +
			"- Writing and reviewing code\n" +
			"- Explaining code and concepts\n" +
			"- Running commands and executing tasks\n" +
			"- Managing sessions and conversations\n\n" +
			"How can I assist you today?",
		StopReason:   llm.StopReasonEndTurn,
		TokensInput:  50,
		TokensOutput: 75,
	}, nil
}

func (m *MockLLMProvider) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamEvent, error) {
	eventChan := make(chan llm.StreamEvent, 10)

	go func() {
		defer close(eventChan)

		content := "Hello! I'm Spin, an AI coding agent. I can help you with various tasks."

		// Simulate streaming by sending chunks
		for i := 0; i < len(content); i += 10 {
			end := i + 10
			if end > len(content) {
				end = len(content)
			}

			eventChan <- llm.StreamEvent{
				Type:    llm.StreamEventTypeContent,
				Content: content[i:end],
			}
		}

		// Send completion event
		eventChan <- llm.StreamEvent{
			Type:       llm.StreamEventTypeDone,
			StopReason: llm.StopReasonEndTurn,
		}
	}()

	return eventChan, nil
}
