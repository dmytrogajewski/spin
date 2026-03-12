package acp

import (
	"context"
	"log/slog"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
)

// TestRequestPermission_SessionNotFound tests that RequestPermission returns error for non-existent session.
func TestRequestPermission_SessionNotFound(t *testing.T) {
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{
		Handler: nil, Emitter: emitter, Validator: nil,
	})
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
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create a session.
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

// permissionDecisionCase describes a test case for request permission with a specific decision.
type permissionDecisionCase struct {
	name         string
	approved     bool
	reason       string
	wantOptionID acp.PermissionOptionId
}

func runPermissionDecisionTests(t *testing.T, cases []permissionDecisionCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			agentInstance := &agent.Agent{}
			mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
			emitter := events.NewEventEmitter(100)

			storage, err := session.NewFileStorage(t.TempDir())
			require.NoError(t, err)
			acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
			require.NoError(t, err)

			approvalHandler := func(_ context.Context, req security.ApprovalRequest) security.ApprovalResponse {
				return security.ApprovalResponse{
					RequestID: req.ID,
					Approved:  tt.approved,
					Reason:    tt.reason,
				}
			}

			approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{
				Handler: approvalHandler, Emitter: emitter, Validator: nil,
			})
			acpAgent.SetApprovalService(approvalService)

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
			assert.Equal(t, tt.wantOptionID, resp.Outcome.Selected.OptionId)
		})
	}
}

// TestRequestPermission_Approved tests successful approval flow.
func TestRequestPermission_Approved(t *testing.T) {
	t.Parallel()
	runPermissionDecisionTests(t, []permissionDecisionCase{
		{"approved", true, "approved for testing", acp.PermissionOptionId("allow")},
	})
}

// TestRequestPermission_Denied tests denial flow.
func TestRequestPermission_Denied(t *testing.T) {
	t.Parallel()
	runPermissionDecisionTests(t, []permissionDecisionCase{
		{"denied", false, "denied for testing", acp.PermissionOptionId("reject")},
	})
}

// TestRequestPermission_Canceled tests context cancellation.
func TestRequestPermission_Canceled(t *testing.T) {
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create approval handler that blocks (simulating cancellation).
	approvalHandler := func(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		// Block until context is canceled.
		<-ctx.Done()

		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  false,
			Reason:    "canceled",
		}
	}

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{
		Handler: approvalHandler, Emitter: emitter, Validator: nil,
	})
	acpAgent.SetApprovalService(approvalService)

	// Create a session.
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

	// Create cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	resp, err := acpAgent.RequestPermission(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Cancelled)
}

// TestRequestPermission_WithRawInput tests tool call conversion with raw input parameters.
func TestRequestPermission_WithRawInput(t *testing.T) {
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Track the operation that was requested.
	var capturedOperation security.Operation

	approvalHandler := func(_ context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		// Capture the operation.
		capturedOperation = security.NewOperation(req.Command, req.Reason, req.WorkDir)

		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
		}
	}

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{
		Handler: approvalHandler, Emitter: emitter, Validator: nil,
	})
	acpAgent.SetApprovalService(approvalService)

	// Create a session.
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
			RawInput: map[string]any{
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

	// Verify operation was created correctly.
	assert.Equal(t, "write_file", capturedOperation.Command.Program)
	assert.Equal(t, "/tmp/test", capturedOperation.WorkDir)
	assert.Contains(t, capturedOperation.Command.Raw, "write_file")
}

// TestRequestPermission_AllowAlwaysOption tests that allow_always option is selected when approved.
func TestRequestPermission_AllowAlwaysOption(t *testing.T) {
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	approvalHandler := func(_ context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
		}
	}

	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{
		Handler: approvalHandler, Emitter: emitter, Validator: nil,
	})
	acpAgent.SetApprovalService(approvalService)

	// Create a session.
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
	// Should select allow_once (first match).
	assert.Equal(t, acp.PermissionOptionId("allow-once"), resp.Outcome.Selected.OptionId)
}

func setupPermissionIntegrationAgent(t *testing.T) (*SpinACPAgent, acp.SessionId) {
	t.Helper()

	agentInstance := &agent.Agent{}
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	mcpSvc := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpSvc, emitter, storage)
	require.NoError(t, err)

	approvalHandler := func(_ context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		approved := req.Command.Program != "dangerous_command"

		reason := "approved"
		if !approved {
			reason = "denied - dangerous command"
		}

		return security.ApprovalResponse{RequestID: req.ID, Approved: approved, Reason: reason}
	}

	acpAgent.SetApprovalService(security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{
		Handler: approvalHandler, Emitter: emitter,
	}))

	sess := session.NewSession("/tmp/test")
	sessionID := acp.SessionId(sess.ID)

	acpAgent.mu.Lock()
	acpAgent.sessions[sessionID] = sess
	acpAgent.mu.Unlock()

	return acpAgent, sessionID
}

func newPermissionRequest(sessionID acp.SessionId, toolCallID, toolName string) acp.RequestPermissionRequest {
	return acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall:  acp.RequestPermissionToolCall{ToolCallId: acp.ToolCallId(toolCallID), Title: acp.Ptr(toolName)},
		Options: []acp.PermissionOption{
			{OptionId: "allow", Name: "Allow", Kind: acp.PermissionOptionKindAllowOnce},
			{OptionId: "reject", Name: "Reject", Kind: acp.PermissionOptionKindRejectOnce},
		},
	}
}

// TestRequestPermission_Integration tests end-to-end integration with ApprovalService.
func TestRequestPermission_Integration(t *testing.T) {
	t.Parallel()
	acpAgent, sessionID := setupPermissionIntegrationAgent(t)

	t.Run("approved", func(t *testing.T) {
		t.Parallel()

		resp, err := acpAgent.RequestPermission(context.Background(), newPermissionRequest(sessionID, "tool-1", "write_file"))
		require.NoError(t, err)
		require.NotNil(t, resp.Outcome.Selected)
		assert.Equal(t, acp.PermissionOptionId("allow"), resp.Outcome.Selected.OptionId)
	})

	t.Run("denied", func(t *testing.T) {
		t.Parallel()

		resp, err := acpAgent.RequestPermission(context.Background(), newPermissionRequest(sessionID, "tool-2", "dangerous_command"))
		require.NoError(t, err)
		require.NotNil(t, resp.Outcome.Selected)
		assert.Equal(t, acp.PermissionOptionId("reject"), resp.Outcome.Selected.OptionId)
	})
}
