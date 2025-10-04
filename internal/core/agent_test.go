package core

import (
	"context"
	"errors"
	"testing"
	"time"

	coretesting "github.com/dmytrogajewski/spin/internal/core/testing"
)

// TestNewAgent tests agent creation with various configurations
func TestNewAgent(t *testing.T) {
	tests := []struct {
		name    string
		llm     coretesting.LLMProvider
		wantErr bool
	}{
		{
			name:    "valid agent with mock LLM",
			llm:     coretesting.NewMockProvider("test response"),
			wantErr: false,
		},
		{
			name:    "nil LLM should fail",
			llm:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create dependencies
			validator := NewValidator()
			executor, _ := NewExecutor(t.TempDir())
			ctx := &Context{WorkDir: t.TempDir()}
			emitter := NewEventEmitter(100)

			agent, err := NewAgent(tt.llm, executor, validator, ctx, emitter)

			if tt.wantErr {
				if err == nil {
					t.Error("NewAgent() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewAgent() unexpected error: %v", err)
				return
			}

			if agent == nil {
				t.Error("NewAgent() returned nil agent")
			}

			if agent.llm != tt.llm {
				t.Error("NewAgent() LLM not set correctly")
			}
		})
	}
}

// TestNewAgent_WithOptions tests agent creation with functional options
func TestNewAgent_WithOptions(t *testing.T) {
	llm := coretesting.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	tests := []struct {
		name        string
		options     []AgentOption
		checkConfig func(*testing.T, *AgentConfig)
	}{
		{
			name:    "default config",
			options: nil,
			checkConfig: func(t *testing.T, cfg *AgentConfig) {
				if cfg.MaxTurns != DefaultMaxTurns {
					t.Errorf("MaxTurns = %d, want %d", cfg.MaxTurns, DefaultMaxTurns)
				}
			},
		},
		{
			name: "custom max turns",
			options: []AgentOption{
				WithMaxTurns(10),
			},
			checkConfig: func(t *testing.T, cfg *AgentConfig) {
				if cfg.MaxTurns != 10 {
					t.Errorf("MaxTurns = %d, want 10", cfg.MaxTurns)
				}
			},
		},
		{
			name: "custom timeout",
			options: []AgentOption{
				WithAgentTimeout(1 * time.Minute),
			},
			checkConfig: func(t *testing.T, cfg *AgentConfig) {
				if cfg.Timeout != 1*time.Minute {
					t.Errorf("Timeout = %v, want 1m", cfg.Timeout)
				}
			},
		},
		{
			name: "require approval",
			options: []AgentOption{
				WithRequireApproval(true),
			},
			checkConfig: func(t *testing.T, cfg *AgentConfig) {
				if !cfg.RequireApproval {
					t.Error("RequireApproval should be true")
				}
			},
		},
		{
			name: "custom temperature",
			options: []AgentOption{
				WithTemperature(0.5),
			},
			checkConfig: func(t *testing.T, cfg *AgentConfig) {
				if cfg.Temperature != 0.5 {
					t.Errorf("Temperature = %f, want 0.5", cfg.Temperature)
				}
			},
		},
		{
			name: "custom max tokens",
			options: []AgentOption{
				WithMaxTokens(2048),
			},
			checkConfig: func(t *testing.T, cfg *AgentConfig) {
				if cfg.MaxTokens != 2048 {
					t.Errorf("MaxTokens = %d, want 2048", cfg.MaxTokens)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(llm, executor, validator, ctx, emitter, tt.options...)
			if err != nil {
				t.Fatalf("NewAgent() error: %v", err)
			}

			tt.checkConfig(t, agent.config)
		})
	}
}

// TestAgent_Execute_SingleTurn tests single turn agent execution
func TestAgent_Execute_SingleTurn(t *testing.T) {
	// Create mock LLM that returns a simple response
	llm := coretesting.NewMockProvider("Here are the files: file1.go, file2.go")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Create request
	req := &AgentRequest{
		Input:   "List files in current directory",
		WorkDir: t.TempDir(),
	}

	// Execute
	resp, err := agent.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if resp == nil {
		t.Fatal("Execute() returned nil response")
	}

	if resp.Content == "" {
		t.Error("Execute() response content is empty")
	}

	if resp.TurnsUsed != 1 {
		t.Errorf("Execute() TurnsUsed = %d, want 1", resp.TurnsUsed)
	}

	// Check that LLM was called
	if llm.Calls != 1 {
		t.Errorf("LLM calls = %d, want 1", llm.Calls)
	}
}

// TestAgent_Execute_MultiTurn tests multi-turn agent execution
func TestAgent_Execute_MultiTurn(t *testing.T) {
	// This test will be implemented when we have proper tool call support
	t.Skip("Skipping multi-turn test until tool call support is implemented")
}

// TestAgent_Execute_MaxTurnsLimit tests that agent respects max turns limit
func TestAgent_Execute_MaxTurnsLimit(t *testing.T) {
	// Skip for now - requires proper tool call support to trigger multi-turn behavior
	// Will be fully implemented in Feature 6.2
	t.Skip("Skipping max turns test until multi-turn/tool call support is implemented")
}

// TestAgent_Execute_Timeout tests that agent respects context timeout
func TestAgent_Execute_Timeout(t *testing.T) {
	// Create mock LLM that returns slowly (simulated)
	llm := coretesting.NewMockProvider("response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter,
		WithAgentTimeout(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	req := &AgentRequest{
		Input:   "Test request",
		WorkDir: t.TempDir(),
	}

	// Create context with immediate cancellation
	ctx2, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure timeout

	_, err = agent.Execute(ctx2, req)
	if err == nil {
		t.Error("Execute() expected timeout error but got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("Execute() error = %v, want context deadline/canceled error", err)
	}
}

// TestAgent_Execute_LLMError tests error handling for LLM failures
func TestAgent_Execute_LLMError(t *testing.T) {
	// Create mock LLM that returns error
	llmErr := errors.New("LLM connection failed")
	llm := coretesting.NewMockProviderWithError(llmErr)
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	req := &AgentRequest{
		Input:   "Test request",
		WorkDir: t.TempDir(),
	}

	_, err = agent.Execute(context.Background(), req)
	if err == nil {
		t.Error("Execute() expected error but got nil")
	}

	if !errors.Is(err, llmErr) {
		t.Errorf("Execute() error = %v, want %v", err, llmErr)
	}
}

// TestAgent_Execute_InvalidRequest tests validation of agent requests
func TestAgent_Execute_InvalidRequest(t *testing.T) {
	llm := coretesting.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	tests := []struct {
		name    string
		req     *AgentRequest
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty input",
			req: &AgentRequest{
				Input:   "",
				WorkDir: t.TempDir(),
			},
			wantErr: true,
		},
		{
			name: "valid request",
			req: &AgentRequest{
				Input:   "test",
				WorkDir: t.TempDir(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := agent.Execute(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAgent_ShouldApprove tests command approval decision logic
func TestAgent_ShouldApprove(t *testing.T) {
	llm := coretesting.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter,
		WithRequireApproval(true),
	)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	tests := []struct {
		name         string
		cmd          *Command
		wantApproval bool
		wantReason   string
	}{
		{
			name: "safe command - ls",
			cmd: &Command{
				Program: "ls",
				Args:    []string{"-la"},
			},
			wantApproval: false,
			wantReason:   "",
		},
		{
			name: "interactive command - mkdir",
			cmd: &Command{
				Program: "mkdir",
				Args:    []string{"newdir"},
			},
			wantApproval: true,
		},
		{
			name: "dangerous command - chmod",
			cmd: &Command{
				Program: "chmod",
				Args:    []string{"+x", "script.sh"},
			},
			wantApproval: true,
		},
		{
			name: "forbidden command - rm -rf /",
			cmd: &Command{
				Program: "rm",
				Args:    []string{"-rf", "/"},
			},
			// Forbidden commands return false because they should never execute
			wantApproval: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsApproval, reason := agent.ShouldApprove(tt.cmd)

			if needsApproval != tt.wantApproval {
				t.Errorf("ShouldApprove() approval = %v, want %v", needsApproval, tt.wantApproval)
			}

			if tt.wantReason != "" && reason != tt.wantReason {
				t.Errorf("ShouldApprove() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

// TestAgent_ShouldApprove_ApprovalDisabled tests that approval can be disabled
func TestAgent_ShouldApprove_ApprovalDisabled(t *testing.T) {
	llm := coretesting.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter,
		WithRequireApproval(false),
	)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Even dangerous commands should not require approval when disabled
	cmd := &Command{
		Program: "rm",
		Args:    []string{"-rf", "/tmp/test"},
	}

	needsApproval, _ := agent.ShouldApprove(cmd)
	if needsApproval {
		t.Error("ShouldApprove() should return false when approval is disabled")
	}
}

// TestAgent_buildPrompt tests prompt construction
func TestAgent_buildPrompt(t *testing.T) {
	llm := coretesting.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{
		WorkDir: t.TempDir(),
		OS: OSInfo{
			OS:   "linux",
			Arch: "amd64",
		},
	}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	req := &AgentRequest{
		Input:   "List files",
		WorkDir: t.TempDir(),
		History: []Message{
			{
				Role:    RoleUser,
				Content: "Previous message",
			},
		},
	}

	messages := agent.buildPrompt(req)

	// Should have system message + history + user input
	if len(messages) < 3 {
		t.Errorf("buildPrompt() returned %d messages, want at least 3", len(messages))
	}

	// First message should be system
	if messages[0].Role != RoleSystem {
		t.Errorf("buildPrompt() first message role = %v, want %v", messages[0].Role, RoleSystem)
	}

	// System message should contain context info
	if messages[0].Content == "" {
		t.Error("buildPrompt() system message content is empty")
	}

	// Last message should be user input
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != RoleUser {
		t.Errorf("buildPrompt() last message role = %v, want %v", lastMsg.Role, RoleUser)
	}

	if lastMsg.Content != "List files" {
		t.Errorf("buildPrompt() last message content = %q, want %q", lastMsg.Content, "List files")
	}
}

// TestAgent_buildPrompt_WithTask tests prompt construction with task mode
func TestAgent_buildPrompt_WithTask(t *testing.T) {
	llm := coretesting.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Create request with review task
	// Note: Normally we'd import the task package, but for this test
	// we'll just verify the system message is not empty
	req := &AgentRequest{
		Input:   "Review this code",
		WorkDir: t.TempDir(),
		Task:    nil, // Will use default prompt
	}

	messages := agent.buildPrompt(req)

	// System message should include task-specific prompt
	systemMsg := messages[0]
	if systemMsg.Role != RoleSystem {
		t.Errorf("buildPrompt() first message role = %v, want %v", systemMsg.Role, RoleSystem)
	}

	// Should contain review-specific instructions
	if systemMsg.Content == "" {
		t.Error("buildPrompt() system message should not be empty")
	}
}

// TestAgent_ProcessToolCall tests tool call processing
func TestAgent_ProcessToolCall(t *testing.T) {
	t.Skip("Skipping ProcessToolCall test until implementation is complete")
}

// TestAgent_ProcessToolCall_WithApproval tests tool call with approval flow
func TestAgent_ProcessToolCall_WithApproval(t *testing.T) {
	t.Skip("Skipping approval test until implementation is complete")
}

// TestAgent_EventEmission tests that agent emits appropriate events
func TestAgent_EventEmission(t *testing.T) {
	llm := coretesting.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	// Subscribe to events
	_, eventsChan, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe() error: %v", err)
	}
	defer emitter.Close()

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	req := &AgentRequest{
		Input:   "Test",
		WorkDir: t.TempDir(),
	}

	// Execute in goroutine and collect events
	done := make(chan error)
	go func() {
		_, err := agent.Execute(context.Background(), req)
		done <- err
	}()

	// Collect events
	events := []Event{}
	timeout := time.After(1 * time.Second)

	// Wait for execution to complete
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute() error: %v", err)
		}
	case <-timeout:
		t.Fatal("Test timed out waiting for execution")
	}

	// Give events time to be emitted
	time.Sleep(100 * time.Millisecond)

	// Drain event channel
drainLoop:
	for {
		select {
		case event, ok := <-eventsChan:
			if !ok {
				break drainLoop
			}
			events = append(events, event)
		default:
			break drainLoop
		}
	}

	// Should have received at least some events
	if len(events) == 0 {
		t.Error("Execute() should emit events but none were received")
	}

	// Check that we got expected event types
	hasTurnStart := false
	hasContent := false
	for _, event := range events {
		if event.Type == EventTurnStart {
			hasTurnStart = true
		}
		if event.Type == EventContentDelta || event.Type == EventTurnComplete {
			hasContent = true
		}
	}

	if !hasTurnStart {
		t.Error("Execute() should emit EventTurnStart")
	}
	if !hasContent {
		t.Error("Execute() should emit content or completion events")
	}
}

// TestAgent_ConcurrentExecute tests concurrent agent executions
func TestAgent_ConcurrentExecute(t *testing.T) {
	llm := coretesting.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Context{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Run multiple concurrent executions
	const concurrency = 5
	done := make(chan bool, concurrency)
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			req := &AgentRequest{
				Input:   "Test request",
				WorkDir: t.TempDir(),
			}

			_, err := agent.Execute(context.Background(), req)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < concurrency; i++ {
		select {
		case <-done:
			// Success
		case err := <-errors:
			t.Errorf("Concurrent execution failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}
}
