package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// TestNewAgent tests agent creation with various configurations
func TestNewAgent(t *testing.T) {
	tests := []struct {
		name    string
		llm     llm.Provider
		wantErr bool
	}{
		{
			name:    "valid agent with mock LLM",
			llm:     llm.NewMockProvider("test response"),
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
			ctx := &Environment{WorkDir: t.TempDir()}
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
				return
			}

			if agent.llm != tt.llm {
				t.Error("NewAgent() LLM not set correctly")
			}
		})
	}
}

// TestNewAgent_WithOptions tests agent creation with functional options
func TestNewAgent_WithOptions(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
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
	llm := llm.NewMockProvider("Here are the files: file1.go, file2.go")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
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
	t.Skip("Skipping timeout test until mock provider supports delay")
	// Create mock LLM
	llm := llm.NewMockProvider("response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
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

	// Create context with realistic timeout
	ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

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
	llm := llm.NewMockProvider("mock")
	llm.SetError(llmErr)
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
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
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
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
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
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
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
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

// TestAgent_EventEmission tests that agent emits appropriate events
func TestAgent_EventEmission(t *testing.T) {
	llm := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
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

	// Wait for events with timeout
	eventTimeout := time.After(100 * time.Millisecond)
drainLoop:
	for {
		select {
		case event, ok := <-eventsChan:
			if !ok {
				break drainLoop
			}
			events = append(events, event)
		case <-eventTimeout:
			break drainLoop
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
	llm := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
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

// =============================================================================
// Feature 6.2: Tool Call Processing Tests
// =============================================================================

// TestAgent_validateToolCall tests tool call validation
func TestAgent_validateToolCall(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	tests := []struct {
		name    string
		call    *ToolCall
		wantErr bool
	}{
		{
			name: "valid tool call",
			call: &ToolCall{
				ID:   "call_123",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "execute_command",
					Arguments: `{"command": "ls"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			call: &ToolCall{
				ID:   "",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "execute_command",
					Arguments: `{"command": "ls"}`,
				},
			},
			wantErr: true,
		},
		{
			name: "missing function name",
			call: &ToolCall{
				ID:   "call_123",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "",
					Arguments: `{"command": "ls"}`,
				},
			},
			wantErr: true,
		},
		{
			name:    "nil tool call",
			call:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := agent.validateToolCall(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateToolCall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAgent_parseToolArguments tests argument parsing
func TestAgent_parseToolArguments(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	tests := []struct {
		name      string
		call      *ToolCall
		wantArgs  map[string]interface{}
		wantErr   bool
		checkArgs func(*testing.T, map[string]interface{})
	}{
		{
			name: "valid JSON args",
			call: &ToolCall{
				ID:   "call_123",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "execute_command",
					Arguments: `{"command": "ls -la", "workdir": "/tmp"}`,
				},
			},
			wantErr: false,
			checkArgs: func(t *testing.T, args map[string]interface{}) {
				if args["command"] != "ls -la" {
					t.Errorf("command = %v, want 'ls -la'", args["command"])
				}
				if args["workdir"] != "/tmp" {
					t.Errorf("workdir = %v, want '/tmp'", args["workdir"])
				}
			},
		},
		{
			name: "empty JSON",
			call: &ToolCall{
				ID:   "call_123",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "execute_command",
					Arguments: `{}`,
				},
			},
			wantErr: false,
			checkArgs: func(t *testing.T, args map[string]interface{}) {
				if len(args) != 0 {
					t.Errorf("expected empty args, got %v", args)
				}
			},
		},
		{
			name: "invalid JSON",
			call: &ToolCall{
				ID:   "call_123",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "execute_command",
					Arguments: `{invalid json}`,
				},
			},
			wantErr: true,
		},
		{
			name: "empty arguments string",
			call: &ToolCall{
				ID:   "call_123",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "execute_command",
					Arguments: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := agent.parseToolArguments(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseToolArguments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkArgs != nil {
				tt.checkArgs(t, args)
			}
		})
	}
}

// TestAgent_executeCommand tests command execution
func TestAgent_executeCommand(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctx := &Environment{WorkDir: workDir}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		check   func(*testing.T, *ToolResult)
	}{
		{
			name: "simple command",
			args: map[string]interface{}{
				"command": "echo hello",
			},
			wantErr: false,
			check: func(t *testing.T, result *ToolResult) {
				if !result.Success {
					t.Error("expected success")
				}
				if result.ExitCode != 0 {
					t.Errorf("ExitCode = %d, want 0", result.ExitCode)
				}
			},
		},
		{
			name:    "missing command",
			args:    map[string]interface{}{},
			wantErr: false, // Returns result with error, not nil
			check: func(t *testing.T, result *ToolResult) {
				if result.Success {
					t.Error("expected failure for missing command")
				}
				if result.Error == nil {
					t.Error("expected error in result")
				}
			},
		},
		{
			name: "invalid command type",
			args: map[string]interface{}{
				"command": 123, // Should be string
			},
			wantErr: false,
			check: func(t *testing.T, result *ToolResult) {
				if result.Success {
					t.Error("expected failure for invalid command type")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := agent.executeCommand(context.Background(), "test_call", tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				t.Fatal("executeCommand() returned nil result")
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// TestAgent_executeCommand_WithApproval tests command execution with approval
func TestAgent_executeCommand_WithApproval(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctx := &Environment{WorkDir: workDir}
	emitter := NewEventEmitter(100)

	// Create agent with approval required
	agent, err := NewAgent(llm, executor, validator, ctx, emitter,
		WithRequireApproval(true),
	)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Set up approval handler
	approved := true
	agent.approvalHandler = func(req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{
			RequestID: req.ID,
			Approved:  approved,
			Timestamp: time.Now(),
		}
	}

	tests := []struct {
		name     string
		args     map[string]interface{}
		approved bool
		wantErr  bool
		check    func(*testing.T, *ToolResult)
	}{
		{
			name: "approved dangerous command",
			args: map[string]interface{}{
				"command": "chmod +x script.sh",
			},
			approved: true,
			wantErr:  false,
			check: func(t *testing.T, result *ToolResult) {
				// Command will fail (file doesn't exist) but approval worked
				if result == nil {
					t.Fatal("expected result")
				}
			},
		},
		{
			name: "denied dangerous command",
			args: map[string]interface{}{
				"command": "chmod +x script.sh",
			},
			approved: false,
			wantErr:  false,
			check: func(t *testing.T, result *ToolResult) {
				if result.Success {
					t.Error("expected failure for denied command")
				}
				if result.Error == nil {
					t.Error("expected error for denied command")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			approved = tt.approved

			result, err := agent.executeCommand(context.Background(), "test_call", tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// NOTE: File operation tests (read_file, write_file, list_directory) have been moved
// to internal/tools/builtin_test.go where the actual implementation now lives.
// The Agent delegates to the tool registry for these operations.

// TestAgent_ProcessToolCall_Complete tests the complete ProcessToolCall method
func TestAgent_ProcessToolCall_Complete(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctx := &Environment{WorkDir: workDir}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	// Create test file for read test
	testFile := filepath.Join(workDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		call    *ToolCall
		wantErr bool
		check   func(*testing.T, *ToolResult)
	}{
		{
			name: "execute_command",
			call: &ToolCall{
				ID:   "call_cmd",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "execute_command",
					Arguments: `{"command": "echo test"}`,
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *ToolResult) {
				if result.ID != "call_cmd" {
					t.Errorf("ID = %s, want call_cmd", result.ID)
				}
			},
		},
		{
			name: "read_file",
			call: &ToolCall{
				ID:   "call_read",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "read_file",
					Arguments: fmt.Sprintf(`{"path": "%s"}`, testFile),
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *ToolResult) {
				if !result.Success {
					t.Errorf("expected success, got error: %v", result.Error)
				}
				if result.Output != "hello" {
					t.Errorf("Output = %q, want %q", result.Output, "hello")
				}
			},
		},
		{
			name: "write_file",
			call: &ToolCall{
				ID:   "call_write",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "write_file",
					Arguments: fmt.Sprintf(`{"path": "%s", "content": "new content"}`, filepath.Join(workDir, "new.txt")),
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *ToolResult) {
				if !result.Success {
					t.Errorf("expected success, got error: %v", result.Error)
				}
			},
		},
		{
			name: "list_directory",
			call: &ToolCall{
				ID:   "call_list",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "list_directory",
					Arguments: fmt.Sprintf(`{"path": "%s"}`, workDir),
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *ToolResult) {
				if !result.Success {
					t.Errorf("expected success, got error: %v", result.Error)
				}
			},
		},
		{
			name: "unknown tool",
			call: &ToolCall{
				ID:   "call_unknown",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "unknown_tool",
					Arguments: `{}`,
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *ToolResult) {
				if result.Success {
					t.Error("expected failure for unknown tool")
				}
				if result.Error == nil {
					t.Error("expected error for unknown tool")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := agent.ProcessToolCall(context.Background(), tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessToolCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				t.Fatal("ProcessToolCall() returned nil result")
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// TestAgent_ProcessToolCall_Events tests event emission during tool execution
func TestAgent_ProcessToolCall_Events(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctx := &Environment{WorkDir: workDir}
	emitter := NewEventEmitter(100)

	// Subscribe to events
	_, eventsChan, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe() error: %v", err)
	}

	agent, err := NewAgent(llm, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	call := &ToolCall{
		ID:   "call_test",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "execute_command",
			Arguments: `{"command": "echo test"}`,
		},
	}

	// Execute in goroutine
	done := make(chan error)
	go func() {
		_, err := agent.ProcessToolCall(context.Background(), call)
		done <- err
	}()

	// Wait for completion
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ProcessToolCall() error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}

	// Collect events with timeout
	events := []Event{}
	timeout := time.After(50 * time.Millisecond)
	for {
		select {
		case event := <-eventsChan:
			events = append(events, event)
		case <-timeout:
			goto checkEvents
		default:
			goto checkEvents
		}
	}

checkEvents:
	// Should have start and complete events
	hasStart := false
	hasComplete := false
	for _, event := range events {
		if event.Type == EventToolCallStart {
			hasStart = true
		}
		if event.Type == EventToolCallComplete {
			hasComplete = true
		}
	}

	if !hasStart {
		t.Error("Expected EventToolCallStart")
	}
	if !hasComplete {
		t.Error("Expected EventToolCallComplete")
	}
}

// TestNewToolsRegistered verifies that the new Phase 5.1 tools are registered
func TestNewToolsRegistered(t *testing.T) {
	// Create agent
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)
	mockLLM := llm.NewMockProvider("test")

	agent, err := NewAgent(mockLLM, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Check that new tools are registered
	expectedTools := []string{
		"read_file",
		"write_file",
		"list_directory",
		"execute_command",
		"get_context",
		"apply_patch", // Phase 5.1
		"file_search", // Phase 5.1
		"git_context", // Phase 5.1
	}

	registeredTools := agent.toolRegistry.List()
	toolNames := make(map[string]bool)
	for _, tool := range registeredTools {
		toolNames[tool.Name()] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool %q to be registered, but it was not found", expected)
		}
	}

	// Verify total count (should be 8 tools)
	if len(registeredTools) != 8 {
		t.Errorf("Expected 8 tools to be registered, got %d", len(registeredTools))
		t.Logf("Registered tools:")
		for _, tool := range registeredTools {
			t.Logf("  - %s", tool.Name())
		}
	}
}

// TestAgent_ResolveTask_ExplicitTask tests that explicit Task field takes precedence
func TestAgent_ResolveTask_ExplicitTask(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Create explicit task
	explicitTask, err := agent.taskRegistry.Get("review")
	if err != nil {
		t.Fatalf("Failed to get review task: %v", err)
	}

	// Test: Explicit task should take precedence over TaskName
	req := &AgentRequest{
		Input:    "test input",
		Task:     explicitTask,
		TaskName: "compact", // Should be ignored
	}

	resolved, err := agent.resolveTask(req)
	if err != nil {
		t.Fatalf("resolveTask() unexpected error: %v", err)
	}

	if resolved == nil {
		t.Fatal("resolveTask() returned nil task")
	}

	if resolved.Name() != "review" {
		t.Errorf("resolveTask() = %q, want %q", resolved.Name(), "review")
	}

	// Verify it's the exact same instance
	if resolved != explicitTask {
		t.Error("resolveTask() did not return the exact same task instance")
	}
}

// TestAgent_ResolveTask_ByName tests task resolution by name
func TestAgent_ResolveTask_ByName(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Test: Resolve task by name
	req := &AgentRequest{
		Input:    "test input",
		TaskName: "compact",
	}

	resolved, err := agent.resolveTask(req)
	if err != nil {
		t.Fatalf("resolveTask() unexpected error: %v", err)
	}

	if resolved == nil {
		t.Fatal("resolveTask() returned nil task")
	}

	if resolved.Name() != "compact" {
		t.Errorf("resolveTask() = %q, want %q", resolved.Name(), "compact")
	}
}

// TestAgent_ResolveTask_Default tests default task fallback
func TestAgent_ResolveTask_Default(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Test: Neither Task nor TaskName set should use default
	req := &AgentRequest{
		Input: "test input",
		// Neither Task nor TaskName set
	}

	resolved, err := agent.resolveTask(req)
	if err != nil {
		t.Fatalf("resolveTask() unexpected error: %v", err)
	}

	if resolved == nil {
		t.Fatal("resolveTask() returned nil task")
	}

	// Default should be "regular" mode
	if resolved.Name() != "regular" {
		t.Errorf("resolveTask() = %q, want %q (default)", resolved.Name(), "regular")
	}
}

// TestAgent_ResolveTask_InvalidName tests error handling for invalid task names
func TestAgent_ResolveTask_InvalidName(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Test: Invalid task name should return error
	req := &AgentRequest{
		Input:    "test input",
		TaskName: "nonexistent",
	}

	resolved, err := agent.resolveTask(req)
	if err == nil {
		t.Fatal("resolveTask() expected error for invalid task name, got nil")
	}

	if resolved != nil {
		t.Errorf("resolveTask() expected nil task for error case, got %v", resolved)
	}

	// Verify error message is clear
	errMsg := err.Error()
	if !contains(errMsg, "task resolution failed") {
		t.Errorf("Error message should contain 'task resolution failed', got: %s", errMsg)
	}
	if !contains(errMsg, "nonexistent") {
		t.Errorf("Error message should contain task name 'nonexistent', got: %s", errMsg)
	}
	if !contains(errMsg, "not found") {
		t.Errorf("Error message should contain 'not found', got: %s", errMsg)
	}
}

// TestAgent_ResolveTask_AllBuiltinModes tests that all 4 built-in modes can be resolved
func TestAgent_ResolveTask_AllBuiltinModes(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Test all 4 built-in modes
	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			req := &AgentRequest{
				Input:    "test input",
				TaskName: mode,
			}

			resolved, err := agent.resolveTask(req)
			if err != nil {
				t.Fatalf("resolveTask(%q) unexpected error: %v", mode, err)
			}

			if resolved == nil {
				t.Fatalf("resolveTask(%q) returned nil task", mode)
			}

			if resolved.Name() != mode {
				t.Errorf("resolveTask(%q) = %q, want %q", mode, resolved.Name(), mode)
			}
		})
	}
}

// TestAgent_ResolveTask_EmptyTaskName tests that empty TaskName falls back to default
func TestAgent_ResolveTask_EmptyTaskName(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Test: Explicitly empty TaskName should use default
	req := &AgentRequest{
		Input:    "test input",
		TaskName: "", // Explicitly empty
	}

	resolved, err := agent.resolveTask(req)
	if err != nil {
		t.Fatalf("resolveTask() unexpected error: %v", err)
	}

	if resolved == nil {
		t.Fatal("resolveTask() returned nil task")
	}

	// Should get default (regular)
	if resolved.Name() != "regular" {
		t.Errorf("resolveTask() with empty TaskName = %q, want %q (default)", resolved.Name(), "regular")
	}
}

// TestAgent_BuildToolsForTask_RegularMode tests that regular mode has all tools
func TestAgent_BuildToolsForTask_RegularMode(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	regularTask, err := agent.taskRegistry.Get("regular")
	if err != nil {
		t.Fatalf("Failed to get regular task: %v", err)
	}

	tools, err := agent.BuildToolsForTask(regularTask)
	if err != nil {
		t.Fatalf("BuildToolsForTask() unexpected error: %v", err)
	}

	if tools == nil {
		t.Fatal("BuildToolsForTask() returned nil tools")
	}

	// Regular mode should have all 8 tools
	if len(tools) != 8 {
		t.Errorf("buildToolsForTask(regular) = %d tools, want 8", len(tools))
	}

	// Extract tool names
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}

	// Verify all expected tools are present
	expected := []string{"execute_command", "read_file", "write_file", "list_directory",
		"get_context", "apply_patch", "file_search", "git_context"}
	for _, name := range expected {
		found := false
		for _, toolName := range toolNames {
			if toolName == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool %q not found in regular mode", name)
		}
	}
}

// TestAgent_BuildToolsForTask_ReviewMode tests that review mode has only read-only tools
func TestAgent_BuildToolsForTask_ReviewMode(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	reviewTask, err := agent.taskRegistry.Get("review")
	if err != nil {
		t.Fatalf("Failed to get review task: %v", err)
	}

	tools, err := agent.BuildToolsForTask(reviewTask)
	if err != nil {
		t.Fatalf("BuildToolsForTask() unexpected error: %v", err)
	}

	if tools == nil {
		t.Fatal("BuildToolsForTask() returned nil tools")
	}

	// Extract tool names
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}

	// Review mode should only have read-only tools
	expectedAllowed := []string{"read_file", "list_directory", "get_context",
		"file_search", "git_context"}
	for _, name := range expectedAllowed {
		found := false
		for _, toolName := range toolNames {
			if toolName == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected read-only tool %q not found in review mode", name)
		}
	}

	// Should NOT have write tools
	forbidden := []string{"bash", "write_file", "apply_patch"}
	for _, name := range forbidden {
		for _, toolName := range toolNames {
			if toolName == name {
				t.Errorf("Forbidden tool %q found in review mode", name)
			}
		}
	}
}

// TestAgent_BuildToolsForTask_CompactMode tests that compact mode has minimal tools
func TestAgent_BuildToolsForTask_CompactMode(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	compactTask, err := agent.taskRegistry.Get("compact")
	if err != nil {
		t.Fatalf("Failed to get compact task: %v", err)
	}

	tools, err := agent.BuildToolsForTask(compactTask)
	if err != nil {
		t.Fatalf("BuildToolsForTask() unexpected error: %v", err)
	}

	if len(tools) != 3 {
		t.Errorf("buildToolsForTask(compact) = %d tools, want 3", len(tools))
	}

	// Extract tool names
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}

	// Verify expected minimal tools
	expected := []string{"read_file", "get_context", "file_search"}
	for _, name := range expected {
		found := false
		for _, toolName := range toolNames {
			if toolName == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool %q not found in compact mode", name)
		}
	}
}

// TestAgent_BuildToolsForTask_PlanningMode tests that planning mode has only context tools
func TestAgent_BuildToolsForTask_PlanningMode(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	planningTask, err := agent.taskRegistry.Get("planning")
	if err != nil {
		t.Fatalf("Failed to get planning task: %v", err)
	}

	tools, err := agent.BuildToolsForTask(planningTask)
	if err != nil {
		t.Fatalf("BuildToolsForTask() unexpected error: %v", err)
	}

	if len(tools) != 3 {
		t.Errorf("buildToolsForTask(planning) = %d tools, want 3", len(tools))
	}

	// Extract tool names
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}

	// Verify expected context tools
	expected := []string{"get_context", "file_search", "git_context"}
	for _, name := range expected {
		found := false
		for _, toolName := range toolNames {
			if toolName == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool %q not found in planning mode", name)
		}
	}
}

// TestAgent_BuildToolsForTask_NilRegistry tests handling of nil tool registry
func TestAgent_BuildToolsForTask_NilRegistry(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Simulate nil registry
	agent.toolRegistry = nil

	task, _ := agent.taskRegistry.Get("regular")
	tools, err := agent.BuildToolsForTask(task)

	if err != nil {
		t.Errorf("BuildToolsForTask() with nil registry should not error, got: %v", err)
	}

	if tools != nil {
		t.Errorf("BuildToolsForTask() with nil registry should return nil, got: %v", tools)
	}
}

// TestAgent_BuildToolsForTask_SchemaPreserved tests that tool schemas are preserved correctly
func TestAgent_BuildToolsForTask_SchemaPreserved(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test response")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	task, _ := agent.taskRegistry.Get("regular")
	tools, err := agent.BuildToolsForTask(task)

	if err != nil {
		t.Fatalf("BuildToolsForTask() unexpected error: %v", err)
	}

	// Find execute_command tool
	var execTool *llm.Tool
	for i := range tools {
		if tools[i].Function.Name == "execute_command" {
			execTool = &tools[i]
			break
		}
	}

	if execTool == nil {
		t.Fatal("execute_command tool should be present in regular mode")
	}

	if execTool.Type != "function" {
		t.Errorf("execute_command tool type = %q, want %q", execTool.Type, "function")
	}

	if execTool.Function.Description == "" {
		t.Error("execute_command tool should have a description")
	}

	if execTool.Function.Parameters == nil {
		t.Error("execute_command tool should have parameters")
	}
}

// =============================================================================
// P1.5: Integration Tests for Core Agent
// =============================================================================

// TestAgent_Integration_EndToEnd_RegularMode tests complete agent execution in regular mode
func TestAgent_Integration_EndToEnd_RegularMode(t *testing.T) {
	// Create test agent with mock LLM
	llmProvider := llm.NewMockProvider("I have successfully completed the task.")
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctx := &Environment{WorkDir: workDir}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Create request with regular task
	req := &AgentRequest{
		Input:    "List files in current directory",
		WorkDir:  workDir,
		TaskName: "regular",
	}

	// Execute: resolve → filter → execute
	resp, err := agent.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Execute() returned nil response")
	}

	if resp.Content == "" {
		t.Error("Execute() should return content")
	}

	if resp.TurnsUsed < 1 {
		t.Errorf("Execute() TurnsUsed = %d, want >= 1", resp.TurnsUsed)
	}

	// Verify task resolution worked by checking tools were correctly filtered
	// Regular mode should have all 8 tools available to the LLM
	regularTask, _ := agent.taskRegistry.Get("regular")
	tools, _ := agent.BuildToolsForTask(regularTask)
	if len(tools) != 8 {
		t.Errorf("Regular mode should have 8 tools, got %d", len(tools))
	}
}

// TestAgent_Integration_EndToEnd_ReviewMode tests complete agent execution in review mode
func TestAgent_Integration_EndToEnd_ReviewMode(t *testing.T) {
	// Create test agent with mock LLM
	llmProvider := llm.NewMockProvider("I have analyzed the code.")
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctx := &Environment{WorkDir: workDir}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Create request with review task
	req := &AgentRequest{
		Input:    "Review this code",
		WorkDir:  workDir,
		TaskName: "review",
	}

	// Execute: resolve → filter → execute
	resp, err := agent.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Execute() returned nil response")
	}

	// Verify tool filtering was applied (review mode should only have read-only tools)
	reviewTask, _ := agent.taskRegistry.Get("review")
	tools, _ := agent.BuildToolsForTask(reviewTask)

	// Verify no write tools present
	for _, tool := range tools {
		if tool.Function.Name == "write_file" || tool.Function.Name == "bash" || tool.Function.Name == "apply_patch" {
			t.Errorf("Review mode should not have write tool: %s", tool.Function.Name)
		}
	}

	// Review mode should have 5 read-only tools
	if len(tools) != 5 {
		t.Errorf("Review mode should have 5 tools, got %d", len(tools))
	}
}

// TestAgent_Integration_EndToEnd_CompactMode tests complete agent execution in compact mode
func TestAgent_Integration_EndToEnd_CompactMode(t *testing.T) {
	// Create test agent with mock LLM
	llmProvider := llm.NewMockProvider("Quick answer: yes.")
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctx := &Environment{WorkDir: workDir}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Create request with compact task
	req := &AgentRequest{
		Input:    "What is 2+2?",
		WorkDir:  workDir,
		TaskName: "compact",
	}

	// Execute: resolve → filter → execute
	resp, err := agent.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Execute() returned nil response")
	}

	// Verify minimal tool set (should be only 3 tools)
	compactTask, _ := agent.taskRegistry.Get("compact")
	tools, _ := agent.BuildToolsForTask(compactTask)

	if len(tools) != 3 {
		t.Errorf("Compact mode should have exactly 3 tools, got %d", len(tools))
	}

	// Verify execution completed successfully
	if resp.TurnsUsed < 1 {
		t.Errorf("Execute() TurnsUsed = %d, want >= 1", resp.TurnsUsed)
	}
}

// TestAgent_Integration_EndToEnd_PlanningMode tests complete agent execution in planning mode
func TestAgent_Integration_EndToEnd_PlanningMode(t *testing.T) {
	// Create test agent with mock LLM
	llmProvider := llm.NewMockProvider("Here is the plan: 1. Step one, 2. Step two")
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctx := &Environment{WorkDir: workDir}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Create request with planning task
	req := &AgentRequest{
		Input:    "Plan how to implement feature X",
		WorkDir:  workDir,
		TaskName: "planning",
	}

	// Execute: resolve → filter → execute
	resp, err := agent.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Execute() returned nil response")
	}

	// Verify only context tools present
	planningTask, _ := agent.taskRegistry.Get("planning")
	tools, _ := agent.BuildToolsForTask(planningTask)

	if len(tools) != 3 {
		t.Errorf("Planning mode should have exactly 3 tools, got %d", len(tools))
	}

	// Verify execution completed successfully
	if resp.TurnsUsed < 1 {
		t.Errorf("Execute() TurnsUsed = %d, want >= 1", resp.TurnsUsed)
	}
}

// TestAgent_Integration_ConcurrentTaskRegistry tests concurrent access to task registry with 100 goroutines
func TestAgent_Integration_ConcurrentTaskRegistry(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test")
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctx := &Environment{WorkDir: workDir}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Run 100 goroutines concurrently accessing the task registry
	const concurrency = 100
	done := make(chan error, concurrency)
	modes := []string{"regular", "review", "compact", "planning"}

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			// Each goroutine picks a mode based on its ID
			mode := modes[id%len(modes)]

			// Test 1: resolveTask
			req := &AgentRequest{
				Input:    "test",
				TaskName: mode,
			}
			task, err := agent.resolveTask(req)
			if err != nil {
				done <- fmt.Errorf("goroutine %d resolveTask failed: %w", id, err)
				return
			}
			if task.Name() != mode {
				done <- fmt.Errorf("goroutine %d got task %s, want %s", id, task.Name(), mode)
				return
			}

			// Test 2: buildToolsForTask
			_, err = agent.BuildToolsForTask(task)
			if err != nil {
				done <- fmt.Errorf("goroutine %d buildToolsForTask failed: %w", id, err)
				return
			}

			// Test 3: GetTaskRegistry
			registry := agent.GetTaskRegistry()
			if registry == nil {
				done <- fmt.Errorf("goroutine %d GetTaskRegistry returned nil", id)
				return
			}

			// Test 4: ListTaskModes
			modes := agent.ListTaskModes()
			if len(modes) != 4 {
				done <- fmt.Errorf("goroutine %d ListTaskModes returned %d modes, want 4", id, len(modes))
				return
			}

			done <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	timeout := time.After(5 * time.Second)
	for i := 0; i < concurrency; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-timeout:
			t.Fatal("Test timed out waiting for goroutines")
		}
	}
}

// TestAgent_Integration_InvalidTaskHandling tests error handling in end-to-end flow
func TestAgent_Integration_InvalidTaskHandling(t *testing.T) {
	// Create test agent
	llmProvider := llm.NewMockProvider("test")
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctx := &Environment{WorkDir: workDir}
	emitter := NewEventEmitter(100)

	agent, err := NewAgent(llmProvider, executor, validator, ctx, emitter)
	if err != nil {
		t.Fatalf("NewAgent() failed: %v", err)
	}

	// Test with invalid task name
	req := &AgentRequest{
		Input:    "test",
		WorkDir:  workDir,
		TaskName: "nonexistent",
	}

	_, err = agent.Execute(context.Background(), req)
	if err == nil {
		t.Error("Execute() with invalid task should return error")
	}

	// Verify error message is clear
	errMsg := err.Error()
	if !contains(errMsg, "task resolution failed") || !contains(errMsg, "nonexistent") {
		t.Errorf("Error message should be clear, got: %s", errMsg)
	}
}
