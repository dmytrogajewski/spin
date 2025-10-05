package core

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/google/uuid"
)

// Simplified approval tests focusing on core logic without event verification

// TestApprovalFlow_Approved tests the happy path
func TestApprovalFlow_Approved(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	handler := func(req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Reason:    "user approved",
			Timestamp: time.Now(),
		}
	}

	agent, _ := NewAgent(llm, executor, validator, ctx, emitter,
		WithApprovalHandler(handler),
	)

	cmd := &Command{Program: "test", Args: []string{}, Raw: "test"}
	approved := agent.requestApproval(context.Background(), cmd, "test reason")

	if !approved {
		t.Error("requestApproval() should return true for approved command")
	}
}

// TestApprovalFlow_Denied tests denial path
func TestApprovalFlow_Denied(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	handler := func(req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{
			RequestID: req.ID,
			Approved:  false,
			Reason:    "user denied",
			Timestamp: time.Now(),
		}
	}

	agent, _ := NewAgent(llm, executor, validator, ctx, emitter,
		WithApprovalHandler(handler),
	)

	cmd := &Command{Program: "test", Args: []string{}, Raw: "test"}
	approved := agent.requestApproval(context.Background(), cmd, "test reason")

	if approved {
		t.Error("requestApproval() should return false for denied command")
	}
}

// TestApprovalFlow_NoHandler tests auto-deny when no handler
func TestApprovalFlow_NoHandler(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	// No approval handler set
	agent, _ := NewAgent(llm, executor, validator, ctx, emitter)

	cmd := &Command{Program: "test", Args: []string{}, Raw: "test"}
	approved := agent.requestApproval(context.Background(), cmd, "test reason")

	if approved {
		t.Error("requestApproval() should return false when no handler is set")
	}
}

// TestApprovalFlow_Timeout tests timeout handling
func TestApprovalFlow_Timeout(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	handler := func(req ApprovalRequest) ApprovalResponse {
		time.Sleep(200 * time.Millisecond) // Longer than timeout
		return ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Timestamp: time.Now(),
		}
	}

	agent, _ := NewAgent(llm, executor, validator, ctx, emitter,
		WithApprovalHandler(handler),
	)
	agent.config.ApprovalTimeout = 50 * time.Millisecond

	cmd := &Command{Program: "test", Args: []string{}, Raw: "test"}

	start := time.Now()
	approved := agent.requestApproval(context.Background(), cmd, "test")
	duration := time.Since(start)

	if approved {
		t.Error("requestApproval() should return false on timeout")
	}

	if duration > 150*time.Millisecond {
		t.Errorf("requestApproval() took %v, expected timeout at ~50ms", duration)
	}
}

// TestApprovalFlow_ContextCancellation tests context cancellation
func TestApprovalFlow_ContextCancellation(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	handler := func(req ApprovalRequest) ApprovalResponse {
		time.Sleep(200 * time.Millisecond)
		return ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Timestamp: time.Now(),
		}
	}

	agent, _ := NewAgent(llm, executor, validator, ctx, emitter,
		WithApprovalHandler(handler),
	)

	approvalCtx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	cmd := &Command{Program: "test", Args: []string{}, Raw: "test"}
	approved := agent.requestApproval(approvalCtx, cmd, "test")

	if approved {
		t.Error("requestApproval() should return false on context cancellation")
	}
}

// TestApprovalFlow_RequestIDMismatch tests mismatched request ID
func TestApprovalFlow_RequestIDMismatch(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	handler := func(req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{
			RequestID: "wrong-id-" + uuid.New().String(), // Wrong ID!
			Approved:  true,
			Timestamp: time.Now(),
		}
	}

	agent, _ := NewAgent(llm, executor, validator, ctx, emitter,
		WithApprovalHandler(handler),
	)

	cmd := &Command{Program: "test", Args: []string{}, Raw: "test"}
	approved := agent.requestApproval(context.Background(), cmd, "test")

	if approved {
		t.Error("requestApproval() should return false for mismatched request ID")
	}
}

// TestApprovalFlow_WithCommandModification tests command modification
func TestApprovalFlow_WithCommandModification(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	handler := func(req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{
			RequestID:       req.ID,
			Approved:        true,
			Reason:          "approved with modification",
			ModifiedCommand: "ls -la", // Modified to safe command
			Timestamp:       time.Now(),
		}
	}

	agent, _ := NewAgent(llm, executor, validator, ctx, emitter,
		WithApprovalHandler(handler),
	)

	cmd := &Command{
		Program: "rm",
		Args:    []string{"-rf", "/tmp/test"},
		Raw:     "rm -rf /tmp/test",
	}

	approved := agent.requestApproval(context.Background(), cmd, "dangerous command")

	if !approved {
		t.Error("requestApproval() should return true for approved modified command")
	}

	if cmd.Raw != "ls -la" {
		t.Errorf("command.Raw = %q, want %q", cmd.Raw, "ls -la")
	}
	if cmd.Program != "ls" {
		t.Errorf("command.Program = %q, want %q", cmd.Program, "ls")
	}
}

// TestApprovalFlow_ModifiedCommandValidationFails tests modified command validation failure
func TestApprovalFlow_ModifiedCommandValidationFails(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	handler := func(req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{
			RequestID:       req.ID,
			Approved:        true,
			Reason:          "approved with modification",
			ModifiedCommand: "rm -rf /", // Modified to FORBIDDEN command
			Timestamp:       time.Now(),
		}
	}

	agent, _ := NewAgent(llm, executor, validator, ctx, emitter,
		WithApprovalHandler(handler),
	)

	cmd := &Command{
		Program: "chmod",
		Args:    []string{"+x", "script.sh"},
		Raw:     "chmod +x script.sh",
	}

	approved := agent.requestApproval(context.Background(), cmd, "interactive command")

	if approved {
		t.Error("requestApproval() should return false when modified command fails validation")
	}
}

// TestApprovalFlow_ModifiedCommandParseError tests modified command parse error
func TestApprovalFlow_ModifiedCommandParseError(t *testing.T) {
	llm := llm.NewMockProvider("test")
	validator := NewValidator()
	executor, _ := NewExecutor(t.TempDir())
	ctx := &Environment{WorkDir: t.TempDir()}
	emitter := NewEventEmitter(100)

	handler := func(req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{
			RequestID:       req.ID,
			Approved:        true,
			ModifiedCommand: "\"unclosed quote", // Invalid command
			Timestamp:       time.Now(),
		}
	}

	agent, _ := NewAgent(llm, executor, validator, ctx, emitter,
		WithApprovalHandler(handler),
	)

	cmd := &Command{Program: "test", Args: []string{}, Raw: "test"}
	approved := agent.requestApproval(context.Background(), cmd, "test")

	if approved {
		t.Error("requestApproval() should return false when modified command fails to parse")
	}
}
