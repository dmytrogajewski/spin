// Package main demonstrates task mode switching with the Spin core package.
//
// This example shows how to:
//   - Create different task modes (regular, review, compact)
//   - Switch between modes during execution
//   - Understand mode-specific behaviors and constraints
//
// Run this example:
//
//	go run examples/task-modes/main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/core/task"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func main() {
	fmt.Println("🎯 Spin AI Agent - Task Mode Example")
	fmt.Println("=" + string(make([]byte, 48)))
	fmt.Println()

	// Create task registry
	taskRegistry := task.NewRegistry()

	// Create base configuration
	cfg := &core.Config{
		MaxTurns:    10,
		Temperature: 0.7,
		MaxTokens:   2048,
		Model:       "claude-3-5-sonnet-20241022",
		Debug:       false,
	}

	// Register different task modes
	taskRegistry.Register("regular", task.NewRegular(cfg))
	taskRegistry.Register("review", task.NewReview(cfg))
	taskRegistry.Register("compact", task.NewCompact(cfg))
	taskRegistry.SetDefault("regular")

	// Demonstrate each mode
	demonstrateRegularMode(taskRegistry)
	demonstrateReviewMode(taskRegistry)
	demonstrateCompactMode(taskRegistry)

	fmt.Println("\nExample completed successfully!")
}

func demonstrateRegularMode(registry *task.Registry) {
	fmt.Println("📝 Regular Mode")
	fmt.Println("-" + string(make([]byte, 24)))
	fmt.Println("Full interactive coding mode with:")

	mode, _ := registry.Get("regular")

	fmt.Printf("  • Name: %s\n", mode.Name())
	fmt.Printf("  • Description: %s\n", mode.Description())
	fmt.Printf("  • System Prompt: %s\n", truncate(mode.SystemPrompt(), 60))
	fmt.Printf("  • Auto-approve: %v\n", mode.ShouldAutoApprove())
	fmt.Printf("  • Max tokens: %d\n", mode.MaxTokens())

	allowedTools := mode.AllowedTools()
	if allowedTools == nil {
		fmt.Println("  • Tools: All tools allowed")
	} else {
		fmt.Printf("  • Tools: %d specific tools\n", len(allowedTools))
	}

	fmt.Println("\n  Use cases:")
	fmt.Println("    - Full coding sessions")
	fmt.Println("    - Complex multi-step tasks")
	fmt.Println("    - Interactive development")
	fmt.Println()
}

func demonstrateReviewMode(registry *task.Registry) {
	fmt.Println("🔍 Review Mode")
	fmt.Println("-" + string(make([]byte, 24)))
	fmt.Println("Read-only code review mode with:")

	mode, _ := registry.Get("review")

	fmt.Printf("  • Name: %s\n", mode.Name())
	fmt.Printf("  • Description: %s\n", mode.Description())
	fmt.Printf("  • System Prompt: %s\n", truncate(mode.SystemPrompt(), 60))
	fmt.Printf("  • Auto-approve: %v\n", mode.ShouldAutoApprove())
	fmt.Printf("  • Max tokens: %d\n", mode.MaxTokens())

	allowedTools := mode.AllowedTools()
	if allowedTools != nil {
		fmt.Printf("  • Tools: %d allowed (read-only)\n", len(allowedTools))
		fmt.Printf("    Examples: %v\n", allowedTools[:min(3, len(allowedTools))])
	}

	fmt.Println("\n  Use cases:")
	fmt.Println("    - Code reviews")
	fmt.Println("    - Security audits")
	fmt.Println("    - Documentation review")
	fmt.Println()
}

func demonstrateCompactMode(registry *task.Registry) {
	fmt.Println("⚡ Compact Mode")
	fmt.Println("-" + string(make([]byte, 24)))
	fmt.Println("Lightweight mode with minimal context:")

	mode, _ := registry.Get("compact")

	fmt.Printf("  • Name: %s\n", mode.Name())
	fmt.Printf("  • Description: %s\n", mode.Description())
	fmt.Printf("  • System Prompt: %s\n", truncate(mode.SystemPrompt(), 60))
	fmt.Printf("  • Auto-approve: %v\n", mode.ShouldAutoApprove())
	fmt.Printf("  • Max tokens: %d (reduced for efficiency)\n", mode.MaxTokens())

	fmt.Println("\n  Use cases:")
	fmt.Println("    - Quick queries")
	fmt.Println("    - Simple tasks")
	fmt.Println("    - Cost-optimized operations")
	fmt.Println()
}

func demonstrateModeSwitch(ctx context.Context) {
	fmt.Println("🔄 Task Mode Switching")
	fmt.Println("-" + string(make([]byte, 24)))
	fmt.Println()

	// Create configuration
	cfg := &core.Config{
		MaxTurns:    5,
		Temperature: 0.7,
		Model:       "claude-3-5-sonnet-20241022",
	}

	// Create manager
	mgr, err := core.NewManager(
		cfg,
		core.WithLLMProvider(&MockLLMProvider{}),
		core.WithToolRegistry(tools.NewRegistry()),
	)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Start with regular mode
	conv, _ := mgr.NewConversation(ctx)
	fmt.Println("Started in Regular mode")

	// Simulate task execution
	fmt.Println("  → Performing full coding task...")
	time.Sleep(100 * time.Millisecond)

	// Switch to review mode
	fmt.Println("\nSwitching to Review mode")
	fmt.Println("  → Reviewing changes (read-only)...")
	time.Sleep(100 * time.Millisecond)

	// Switch to compact mode
	fmt.Println("\nSwitching to Compact mode")
	fmt.Println("  → Quick status check...")
	time.Sleep(100 * time.Millisecond)

	_ = conv // Suppress unused warning
	fmt.Println("\n✓ Mode switching completed")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MockLLMProvider for demonstration
type MockLLMProvider struct{}

func (m *MockLLMProvider) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{
		Content:      "Mode demonstration response",
		StopReason:   llm.StopReasonEndTurn,
		TokensInput:  20,
		TokensOutput: 30,
	}, nil
}

func (m *MockLLMProvider) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamEvent, error) {
	eventChan := make(chan llm.StreamEvent, 1)
	go func() {
		defer close(eventChan)
		eventChan <- llm.StreamEvent{
			Type:       llm.StreamEventTypeDone,
			StopReason: llm.StopReasonEndTurn,
		}
	}()
	return eventChan, nil
}
