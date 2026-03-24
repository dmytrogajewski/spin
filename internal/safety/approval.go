// Package safety provides security approval handling.
package safety

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/pkg/alg/concurrency"
)

var (
	// ErrNoApprovalHandlerConfigured is a sentinel error.
	ErrNoApprovalHandlerConfigured = errors.New("no approval handler configured")
	// ErrContextCanceled is a sentinel error.
	ErrContextCanceled = errors.New("context canceled")
	// ErrRequestIDMismatch is a sentinel error.
	ErrRequestIDMismatch = errors.New("request ID mismatch")
	// ErrModifiedCommandNotSafe is a sentinel error.
	ErrModifiedCommandNotSafe = errors.New("modified command not safe")
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
	reqID = uuid.New().String()

	req := ApprovalRequest{
		ID:         reqID,
		Command:    operation.Command,
		Reason:     operation.Reason,
		WorkDir:    operation.WorkDir,
		Timestamp:  time.Now(),
		ToolCallID: operation.ToolCallID,
	}

	// Short-circuit via persisted policy.
	if ok, policyApproved := s.checkPolicyShortCircuit(ctx, reqID, operation); ok {
		return reqID, policyApproved, nil
	}

	s.emitIfPresent(func() { s.emitApprovalRequest(req) })

	if s.handler == nil {
		s.emitIfPresent(func() { s.emitApprovalDenied(reqID, operation.Command, "no approval handler configured") })

		return reqID, false, ErrNoApprovalHandlerConfigured
	}

	handler := s.handler
	resp, ok := concurrency.CallWithContext(ctx, func() ApprovalResponse {
		return handler(ctx, req)
	})

	if !ok {
		s.emitIfPresent(func() { s.emitApprovalDenied(reqID, operation.Command, "context canceled") })

		return reqID, false, ErrContextCanceled
	}

	if resp.RequestID != reqID {
		s.emitIfPresent(func() { s.emitApprovalDenied(reqID, operation.Command, "response request ID mismatch") })

		return reqID, false, ErrRequestIDMismatch
	}

	if resp.Approved && resp.ModifiedCommand != "" {
		return s.handleModifiedCommand(ctx, reqID, operation.Command, resp)
	}

	if resp.Approved {
		s.persistApprovalPolicy(ctx, operation, resp)
		s.emitIfPresent(func() { s.emitApprovalApproved(reqID, operation.Command, resp.Reason) })

		return reqID, true, nil
	}

	s.emitIfPresent(func() { s.emitApprovalDenied(reqID, operation.Command, resp.Reason) })

	return reqID, false, nil
}

// emitIfPresent calls fn only if the emitter is configured.
func (s *ApprovalService) emitIfPresent(fn func()) {
	if s.emitter != nil {
		fn()
	}
}

// checkPolicyShortCircuit checks persisted policies for a short-circuit decision.
// Returns (true, decision) if a policy was found, (false, false) otherwise.
func (s *ApprovalService) checkPolicyShortCircuit(ctx context.Context, reqID string, operation Operation) (found, approved bool) {
	if s.store == nil || operation.Command == nil {
		return false, false
	}

	key := NewPolicyKey(operation.Command.Program, operation.Command.Args, operation.WorkDir)

	if policy, ok, getErr := s.store.Get(ctx, key, ScopeSession); ok && getErr == nil {
		s.emitPolicyApplied("approval short-circuited by session policy")
		s.emitIfPresent(func() { s.emitApprovalApproved(reqID, operation.Command, "approved via policy (session)") })

		return true, policy.Decision == DecisionAllow
	}

	if policy, ok, getErr := s.store.Get(ctx, key, ScopeGlobal); ok && getErr == nil {
		s.emitPolicyApplied("approval short-circuited by global policy")
		s.emitIfPresent(func() { s.emitApprovalApproved(reqID, operation.Command, "approved via policy (global)") })

		return true, policy.Decision == DecisionAllow
	}

	return false, false
}

// emitPolicyApplied emits a policy applied event.
func (s *ApprovalService) emitPolicyApplied(message string) {
	if s.emitter == nil {
		return
	}

	s.emitter.Emit(events.Event{
		Type:      events.EventPolicyApplied,
		Timestamp: time.Now(),
		Data: events.SystemEventData{
			Level:   "info",
			Message: message,
		},
	})
}

// persistApprovalPolicy saves an approval policy if scope requires it.
func (s *ApprovalService) persistApprovalPolicy(ctx context.Context, operation Operation, resp ApprovalResponse) {
	if s.store == nil || operation.Command == nil {
		return
	}

	if resp.Scope != ScopeSession && resp.Scope != ScopeGlobal {
		return
	}

	expiresAt := s.resolveExpiry(resp)

	policy := Policy{
		Version:    "1",
		Scope:      resp.Scope,
		Key:        NewPolicyKey(operation.Command.Program, operation.Command.Args, operation.WorkDir),
		Decision:   DecisionAllow,
		PolicyNote: resp.PolicyNote,
		CreatedAt:  time.Now(),
		ExpiresAt:  expiresAt,
	}

	if saveErr := s.store.Save(ctx, policy); saveErr != nil {
		return
	}

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

// resolveExpiry computes the expiry time for a policy.
func (s *ApprovalService) resolveExpiry(resp ApprovalResponse) *time.Time {
	if resp.TTL != nil {
		t := time.Now().Add(*resp.TTL)

		return &t
	}

	var ttl time.Duration
	if resp.Scope == ScopeSession && s.sessionDefaultTTL > 0 {
		ttl = s.sessionDefaultTTL
	}

	if resp.Scope == ScopeGlobal && s.globalDefaultTTL > 0 {
		ttl = s.globalDefaultTTL
	}

	if ttl > 0 {
		t := time.Now().Add(ttl)

		return &t
	}

	return nil
}

// handleModifiedCommand handles approval with a modified command.
func (s *ApprovalService) handleModifiedCommand(
	_ context.Context, reqID string, originalCmd *Command, resp ApprovalResponse,
) (modifiedCmd string, approved bool, err error) {
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
		if validatedID, validationErr := s.validateModifiedCommand(reqID, originalCmd, modCmd); validationErr != nil {
			return validatedID, false, validationErr
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

// validateModifiedCommand validates a modified command and returns an error if it's not safe.
func (s *ApprovalService) validateModifiedCommand(reqID string, originalCmd, modCmd *Command) (id string, validationErr error) {
	classifyResult, classErr := s.validator.Classify(modCmd)
	if classErr != nil {
		s.emitIfPresent(func() {
			s.emitApprovalDenied(reqID, originalCmd, "modified command validation error: "+classErr.Error())
		})

		return reqID, fmt.Errorf("validation error: %w", classErr)
	}

	if classifyResult.Classification != CommandSafe {
		s.emitIfPresent(func() {
			s.emitApprovalDenied(reqID, originalCmd, "modified command failed validation: "+classifyResult.Classification.String())
		})

		return reqID, fmt.Errorf("modified command not safe: %s: %w", classifyResult.Classification, ErrModifiedCommandNotSafe)
	}

	return reqID, nil
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
func NewOperation(cmd *Command, reason, workDir string) Operation {
	return Operation{
		Command: cmd,
		Reason:  reason,
		WorkDir: workDir,
	}
}

// NewOperationWithToolCallID creates a new Operation with the given command, reason,
// work directory, and tool call ID. This is used when the operation is associated
// with a specific LLM tool call.
func NewOperationWithToolCallID(cmd *Command, reason, workDir, toolCallID string) Operation {
	return Operation{
		Command:    cmd,
		Reason:     reason,
		WorkDir:    workDir,
		ToolCallID: toolCallID,
	}
}

// ApprovalServiceOption is a functional option for ApprovalService.
type ApprovalServiceOption func(*ApprovalService) error
