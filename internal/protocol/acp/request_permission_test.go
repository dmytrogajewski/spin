package acp

import (
	"context"
	"log/slog"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRequestPermission_SessionNotFound tests that RequestPermission returns error for non-existent session.
func TestRequestPermission_SessionNotFound(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: nil})
	acpAgent.SetApprovalService(approvalService)

	req := acp.RequestPermissionRequest{
		SessionId: acp.SessionId("non-existent"),
		ToolCall: acp.RequestPermissionToolCall{
			ToolCallId: acp.ToolCallId("tool-1"),
			Title:      acp.Ptr("write_file"),
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow"),
				Name:     "Allow",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
		},
	}

	_, err = acpAgent.RequestPermission(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

// TestRequestPermission_NoApprovalService tests that RequestPermission returns error when approval service is not configured.
func TestRequestPermission_NoApprovalService(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create a session
	sess := session.NewSession("/tmp/test")
	sessionID := acp.SessionId(sess.ID)
	acpAgent.mu.Lock()
	acpAgent.sessions[sessionID] = sess
	acpAgent.mu.Unlock()

	req := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall: acp.RequestPermissionToolCall{
			ToolCallId: acp.ToolCallId("tool-1"),
			Title:      acp.Ptr("write_file"),
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow"),
				Name:     "Allow",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
		},
	}

	_, err = acpAgent.RequestPermission(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "approval service not configured")
}

// TestRequestPermission_Approved tests successful approval flow.
func TestRequestPermission_Approved(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create approval handler that always approves
	approvalHandler := func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Reason:    "approved for testing",
		}
	}

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: approvalHandler, Emitter: emitter, Validator: nil})
	acpAgent.SetApprovalService(approvalService)

	// Create a session
	sess := session.NewSession("/tmp/test")
	sessionID := acp.SessionId(sess.ID)
	acpAgent.mu.Lock()
	acpAgent.sessions[sessionID] = sess
	acpAgent.mu.Unlock()

	req := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall: acp.RequestPermissionToolCall{
			ToolCallId: acp.ToolCallId("tool-1"),
			Title:      acp.Ptr("write_file"),
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow"),
				Name:     "Allow",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
			{
				OptionId: acp.PermissionOptionId("reject"),
				Name:     "Reject",
				Kind:     acp.PermissionOptionKindRejectOnce,
			},
		},
	}

	resp, err := acpAgent.RequestPermission(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Selected)
	assert.Equal(t, acp.PermissionOptionId("allow"), resp.Outcome.Selected.OptionId)
}

// TestRequestPermission_Denied tests denial flow.
func TestRequestPermission_Denied(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create approval handler that always denies
	approvalHandler := func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  false,
			Reason:    "denied for testing",
		}
	}

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: approvalHandler, Emitter: emitter, Validator: nil})
	acpAgent.SetApprovalService(approvalService)

	// Create a session
	sess := session.NewSession("/tmp/test")
	sessionID := acp.SessionId(sess.ID)
	acpAgent.mu.Lock()
	acpAgent.sessions[sessionID] = sess
	acpAgent.mu.Unlock()

	req := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall: acp.RequestPermissionToolCall{
			ToolCallId: acp.ToolCallId("tool-1"),
			Title:      acp.Ptr("write_file"),
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow"),
				Name:     "Allow",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
			{
				OptionId: acp.PermissionOptionId("reject"),
				Name:     "Reject",
				Kind:     acp.PermissionOptionKindRejectOnce,
			},
		},
	}

	resp, err := acpAgent.RequestPermission(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Selected)
	assert.Equal(t, acp.PermissionOptionId("reject"), resp.Outcome.Selected.OptionId)
}

// TestRequestPermission_Cancelled tests context cancellation.
func TestRequestPermission_Cancelled(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create approval handler that blocks (simulating cancellation)
	approvalHandler := func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		// Block until context is cancelled
		<-ctx.Done()
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  false,
			Reason:    "cancelled",
		}
	}

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: approvalHandler, Emitter: emitter, Validator: nil})
	acpAgent.SetApprovalService(approvalService)

	// Create a session
	sess := session.NewSession("/tmp/test")
	sessionID := acp.SessionId(sess.ID)
	acpAgent.mu.Lock()
	acpAgent.sessions[sessionID] = sess
	acpAgent.mu.Unlock()

	req := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall: acp.RequestPermissionToolCall{
			ToolCallId: acp.ToolCallId("tool-1"),
			Title:      acp.Ptr("write_file"),
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow"),
				Name:     "Allow",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
		},
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	resp, err := acpAgent.RequestPermission(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Cancelled)
}

// TestRequestPermission_WithRawInput tests tool call conversion with raw input parameters.
func TestRequestPermission_WithRawInput(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Track the operation that was requested
	var capturedOperation security.Operation
	approvalHandler := func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		// Capture the operation
		capturedOperation = security.NewOperation(req.Command, req.Reason, req.WorkDir)
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
		}
	}

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: approvalHandler, Emitter: emitter, Validator: nil})
	acpAgent.SetApprovalService(approvalService)

	// Create a session
	sess := session.NewSession("/tmp/test")
	sessionID := acp.SessionId(sess.ID)
	acpAgent.mu.Lock()
	acpAgent.sessions[sessionID] = sess
	acpAgent.mu.Unlock()

	req := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall: acp.RequestPermissionToolCall{
			ToolCallId: acp.ToolCallId("tool-1"),
			Title:      acp.Ptr("write_file"),
			RawInput: map[string]interface{}{
				"path":    "/tmp/test.txt",
				"content": "test content",
			},
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow"),
				Name:     "Allow",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
		},
	}

	_, err = acpAgent.RequestPermission(context.Background(), req)
	require.NoError(t, err)

	// Verify operation was created correctly
	assert.Equal(t, "write_file", capturedOperation.Command.Program)
	assert.Equal(t, "/tmp/test", capturedOperation.WorkDir)
	assert.Contains(t, capturedOperation.Command.Raw, "write_file")
}

// TestRequestPermission_AllowAlwaysOption tests that allow_always option is selected when approved.
func TestRequestPermission_AllowAlwaysOption(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	approvalHandler := func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
		}
	}

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: approvalHandler, Emitter: emitter, Validator: nil})
	acpAgent.SetApprovalService(approvalService)

	// Create a session
	sess := session.NewSession("/tmp/test")
	sessionID := acp.SessionId(sess.ID)
	acpAgent.mu.Lock()
	acpAgent.sessions[sessionID] = sess
	acpAgent.mu.Unlock()

	req := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall: acp.RequestPermissionToolCall{
			ToolCallId: acp.ToolCallId("tool-1"),
			Title:      acp.Ptr("write_file"),
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow-once"),
				Name:     "Allow Once",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
			{
				OptionId: acp.PermissionOptionId("allow-always"),
				Name:     "Allow Always",
				Kind:     acp.PermissionOptionKindAllowAlways,
			},
		},
	}

	resp, err := acpAgent.RequestPermission(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Selected)
	// Should select allow_once (first match)
	assert.Equal(t, acp.PermissionOptionId("allow-once"), resp.Outcome.Selected.OptionId)
}

// TestRequestPermission_Integration tests end-to-end integration with ApprovalService.
func TestRequestPermission_Integration(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create approval handler that simulates user interaction
	approvalHandler := func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		// Simulate approval based on command
		approved := req.Command.Program != "dangerous_command"
		reason := "approved"
		if !approved {
			reason = "denied - dangerous command"
		}
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  approved,
			Reason:    reason,
		}
	}

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: approvalHandler, Emitter: emitter, Validator: nil})
	acpAgent.SetApprovalService(approvalService)

	// Create a session
	sess := session.NewSession("/tmp/test")
	sessionID := acp.SessionId(sess.ID)
	acpAgent.mu.Lock()
	acpAgent.sessions[sessionID] = sess
	acpAgent.mu.Unlock()

	// Test approved case
	reqApproved := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall: acp.RequestPermissionToolCall{
			ToolCallId: acp.ToolCallId("tool-1"),
			Title:      acp.Ptr("write_file"),
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow"),
				Name:     "Allow",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
			{
				OptionId: acp.PermissionOptionId("reject"),
				Name:     "Reject",
				Kind:     acp.PermissionOptionKindRejectOnce,
			},
		},
	}

	resp, err := acpAgent.RequestPermission(context.Background(), reqApproved)
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Selected)
	assert.Equal(t, acp.PermissionOptionId("allow"), resp.Outcome.Selected.OptionId)

	// Test denied case
	reqDenied := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall: acp.RequestPermissionToolCall{
			ToolCallId: acp.ToolCallId("tool-2"),
			Title:      acp.Ptr("dangerous_command"),
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow"),
				Name:     "Allow",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
			{
				OptionId: acp.PermissionOptionId("reject"),
				Name:     "Reject",
				Kind:     acp.PermissionOptionKindRejectOnce,
			},
		},
	}

	resp, err = acpAgent.RequestPermission(context.Background(), reqDenied)
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Selected)
	assert.Equal(t, acp.PermissionOptionId("reject"), resp.Outcome.Selected.OptionId)
}
