package security

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dmytrogajewski/spin/internal/events"
)

// ApprovalService provides centralized approval handling for various operations.
// It can be used by executors, agents, tools, and other components that need
// user approval for potentially dangerous operations.
type ApprovalService struct {
	handler   ApprovalHandler
	emitter   *events.EventEmitter
	validator *Validator
	store     PolicyStore
	// default TTLs used when persisting approvals without explicit TTL.
	sessionDefaultTTL time.Duration
	globalDefaultTTL  time.Duration
}

// ApprovalServiceConfig configures the approval service.
type ApprovalServiceConfig struct {
	Handler           ApprovalHandler
	Emitter           *events.EventEmitter
	Validator         *Validator
	Store             PolicyStore
	SessionDefaultTTL time.Duration
	GlobalDefaultTTL  time.Duration
}

// NewApprovalServiceWithConfig creates a new approval service with full configuration.
func NewApprovalServiceWithConfig(cfg ApprovalServiceConfig) *ApprovalService {
	return &ApprovalService{
		handler:           cfg.Handler,
		emitter:           cfg.Emitter,
		validator:         cfg.Validator,
		store:             cfg.Store,
		sessionDefaultTTL: cfg.SessionDefaultTTL,
		globalDefaultTTL:  cfg.GlobalDefaultTTL,
	}
}

// SetHandler updates the approval handler.
// This allows runtime-specific handlers (ACP, TUI, etc.) to be swapped after service creation.
func (s *ApprovalService) SetHandler(handler ApprovalHandler) {
	s.handler = handler
}

// RequestApproval requests approval for an operation with full event emission.
// Returns the request ID, whether approved, and any error.
func (s *ApprovalService) RequestApproval(ctx context.Context, operation Operation) (reqID string, approved bool, err error) {
	// Generate unique request ID.
	reqID = uuid.New().String()

	// Create approval request.
	req := ApprovalRequest{
		ID:         reqID,
		Command:    operation.Command,
		Reason:     operation.Reason,
		WorkDir:    operation.WorkDir,
		Timestamp:  time.Now(),
		ToolCallID: operation.ToolCallID,
	}

	// Short-circuit via persisted policy if available (session first, then global).
	if s.store != nil && operation.Command != nil {
		key := NewPolicyKey(operation.Command.Program, operation.Command.Args, operation.WorkDir)
		if p, ok, _ := s.store.Get(ctx, key, ScopeSession); ok {
			if s.emitter != nil {
				s.emitter.Emit(events.Event{
					Type:      events.EventPolicyApplied,
					Timestamp: time.Now(),
					Data: events.SystemEventData{
						Level:   "info",
						Message: "approval short-circuited by session policy",
					},
				})
				s.emitApprovalApproved(reqID, operation.Command, "approved via policy (session)")
			}

			return reqID, p.Decision == DecisionAllow, nil
		}

		if p, ok, _ := s.store.Get(ctx, key, ScopeGlobal); ok {
			if s.emitter != nil {
				s.emitter.Emit(events.Event{
					Type:      events.EventPolicyApplied,
					Timestamp: time.Now(),
					Data: events.SystemEventData{
						Level:   "info",
						Message: "approval short-circuited by global policy",
					},
				})
				s.emitApprovalApproved(reqID, operation.Command, "approved via policy (global)")
			}

			return reqID, p.Decision == DecisionAllow, nil
		}
	}

	// Emit approval request event if emitter available.
	if s.emitter != nil {
		s.emitApprovalRequest(req)
	}

	// If no handler, auto-deny.
	if s.handler == nil {
		if s.emitter != nil {
			s.emitApprovalDenied(reqID, operation.Command, "no approval handler configured")
		}

		return reqID, false, errors.New("no approval handler configured")
	}

	// Invoke approval handler.
	resp, canceled := s.invokeHandler(ctx, req)
	if canceled {
		if s.emitter != nil {
			s.emitApprovalDenied(reqID, operation.Command, "context canceled")
		}

		return reqID, false, errors.New("context canceled")
	}

	// Validate response.
	if resp.RequestID != reqID {
		if s.emitter != nil {
			s.emitApprovalDenied(reqID, operation.Command, "response request ID mismatch")
		}

		return reqID, false, errors.New("request ID mismatch")
	}

	// Handle command modification if needed.
	if resp.Approved && resp.ModifiedCommand != "" {
		return s.handleModifiedCommand(ctx, reqID, operation.Command, resp)
	}

	// Process approval decision.
	if resp.Approved {
		// Persist policy if scope requires it.
		if s.store != nil && operation.Command != nil && (resp.Scope == ScopeSession || resp.Scope == ScopeGlobal) {
			var expiresAt *time.Time

			if resp.TTL != nil {
				t := time.Now().Add(*resp.TTL)
				expiresAt = &t
			} else {
				// Apply defaults per scope if configured.
				var ttl time.Duration
				if resp.Scope == ScopeSession && s.sessionDefaultTTL > 0 {
					ttl = s.sessionDefaultTTL
				}

				if resp.Scope == ScopeGlobal && s.globalDefaultTTL > 0 {
					ttl = s.globalDefaultTTL
				}

				if ttl > 0 {
					t := time.Now().Add(ttl)
					expiresAt = &t
				}
			}

			p := Policy{
				Version:    "1",
				Scope:      resp.Scope,
				Key:        NewPolicyKey(operation.Command.Program, operation.Command.Args, operation.WorkDir),
				Decision:   DecisionAllow,
				PolicyNote: resp.PolicyNote,
				CreatedAt:  time.Now(),
				ExpiresAt:  expiresAt,
			}

			_ = s.store.Save(ctx, p)
			if s.emitter != nil {
				s.emitter.Emit(events.Event{
					Type:      events.EventPolicySaved,
					Timestamp: time.Now(),
					Data: events.SystemEventData{
						Level:   "info",
						Message: "approval policy persisted: " + resp.Scope,
					},
				})
			}
		}

		if s.emitter != nil {
			s.emitApprovalApproved(reqID, operation.Command, resp.Reason)
		}

		return reqID, true, nil
	}

	if s.emitter != nil {
		s.emitApprovalDenied(reqID, operation.Command, resp.Reason)
	}

	return reqID, false, nil
}

// invokeHandler invokes the approval handler.
func (s *ApprovalService) invokeHandler(ctx context.Context, req ApprovalRequest) (ApprovalResponse, bool) {
	// Invoke handler in goroutine with context.
	respChan := make(chan ApprovalResponse, 1)

	go func() {
		resp := s.handler(ctx, req)
		respChan <- resp
	}()

	// Wait for response or context cancellation.
	select {
	case resp := <-respChan:
		return resp, false
	case <-ctx.Done():
		return ApprovalResponse{}, true
	}
}

// handleModifiedCommand handles approval with a modified command.
func (s *ApprovalService) handleModifiedCommand(_ context.Context, reqID string, originalCmd *Command, resp ApprovalResponse) (string, bool, error) {
	// Parse modified command.
	modCmd, err := ParseCommand(resp.ModifiedCommand)
	if err != nil {
		if s.emitter != nil {
			s.emitApprovalDenied(reqID, originalCmd, "modified command parse error: "+err.Error())
		}

		return reqID, false, fmt.Errorf("parse error: %w", err)
	}

	// Re-validate modified command if validator available.
	if s.validator != nil {
		result, err := s.validator.Classify(modCmd)
		if err != nil {
			if s.emitter != nil {
				s.emitApprovalDenied(reqID, originalCmd, "modified command validation error: "+err.Error())
			}

			return reqID, false, fmt.Errorf("validation error: %w", err)
		}

		// Check if modified command is not safe.
		if result.Classification != CommandSafe {
			if s.emitter != nil {
				s.emitApprovalDenied(reqID, originalCmd, "modified command failed validation: "+result.Classification.String())
			}

			return reqID, false, fmt.Errorf("modified command not safe: %s", result.Classification)
		}
	}

	// Update original command with modified version.
	*originalCmd = *modCmd

	// Emit approved event with modification info.
	if s.emitter != nil {
		s.emitter.Emit(events.Event{
			Type:      events.EventCommandApproved,
			Timestamp: time.Now(),
			Data: events.ApprovalEventData{
				RequestID:       reqID,
				Command:         modCmd.Program,
				WorkDir:         modCmd.WorkDir,
				Reason:          resp.Reason,
				Status:          events.ApprovalStatusApproved,
				ModifiedCommand: resp.ModifiedCommand,
				Timestamp:       time.Now(),
			},
		})
	}

	return reqID, true, nil
}

// emitApprovalRequest emits the approval request event.
func (s *ApprovalService) emitApprovalRequest(req ApprovalRequest) {
	s.emitter.Emit(events.Event{
		Type:      events.EventCommandApproval,
		Timestamp: req.Timestamp,
		Data: events.ApprovalEventData{
			RequestID: req.ID,
			Command:   req.Command.Program,
			WorkDir:   req.WorkDir,
			Reason:    req.Reason,
			Status:    events.ApprovalStatusPending,
			Timestamp: req.Timestamp,
		},
	})
}

// emitApprovalDenied emits the approval denied event.
func (s *ApprovalService) emitApprovalDenied(reqID string, cmd *Command, reason string) {
	s.emitter.Emit(events.Event{
		Type:      events.EventCommandDenied,
		Timestamp: time.Now(),
		Data: events.ApprovalEventData{
			RequestID: reqID,
			Command:   cmd.Program,
			WorkDir:   cmd.WorkDir,
			Reason:    reason,
			Status:    events.ApprovalStatusDenied,
			Timestamp: time.Now(),
		},
	})
}

// emitApprovalApproved emits the approval approved event.
func (s *ApprovalService) emitApprovalApproved(reqID string, cmd *Command, reason string) {
	s.emitter.Emit(events.Event{
		Type:      events.EventCommandApproved,
		Timestamp: time.Now(),
		Data: events.ApprovalEventData{
			RequestID: reqID,
			Command:   cmd.Program,
			WorkDir:   cmd.WorkDir,
			Reason:    reason,
			Status:    events.ApprovalStatusApproved,
			Timestamp: time.Now(),
		},
	})
}

// Operation represents an operation that may require approval.
type Operation struct {
	// Command is the command to be executed (if applicable).
	Command *Command

	// Reason explains why approval is needed.
	Reason string

	// WorkDir is the working directory for the operation.
	WorkDir string

	// ToolCallID is the LLM tool call ID (e.g., "call-0") when this operation
	// is associated with a tool call. This allows approval notifications to
	// use the same tool call ID as the tool call events.
	ToolCallID string
}

// NewOperation creates a new Operation with the given command, reason, and work directory.
//
// This helper function standardizes Operation construction across the codebase,
// ensuring consistent behavior and simplifying future changes.
func NewOperation(cmd *Command, reason string, workDir string) Operation {
	return Operation{
		Command: cmd,
		Reason:  reason,
		WorkDir: workDir,
	}
}

// NewOperationWithToolCallID creates a new Operation with the given command, reason,
// work directory, and tool call ID. This is used when the operation is associated
// with a specific LLM tool call.
func NewOperationWithToolCallID(cmd *Command, reason string, workDir string, toolCallID string) Operation {
	return Operation{
		Command:    cmd,
		Reason:     reason,
		WorkDir:    workDir,
		ToolCallID: toolCallID,
	}
}

// ApprovalServiceOption is a functional option for ApprovalService.
type ApprovalServiceOption func(*ApprovalService) error
