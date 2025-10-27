package security

import (
	"context"
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/google/uuid"
)

// ApprovalService provides centralized approval handling for various operations.
// It can be used by executors, agents, tools, and other components that need
// user approval for potentially dangerous operations.
type ApprovalService struct {
	handler   ApprovalHandler
	emitter   *events.EventEmitter
	validator *Validator
}

// ApprovalServiceConfig configures the approval service.
type ApprovalServiceConfig struct {
	Handler   ApprovalHandler
	Emitter   *events.EventEmitter
	Validator *Validator
}

// NewApprovalService creates a new approval service with the given handler.
func NewApprovalService(handler ApprovalHandler, emitter *events.EventEmitter, validator *Validator) *ApprovalService {
	return &ApprovalService{
		handler:   handler,
		emitter:   emitter,
		validator: validator,
	}
}

// NewApprovalServiceWithConfig creates a new approval service with full configuration.
func NewApprovalServiceWithConfig(cfg ApprovalServiceConfig) *ApprovalService {
	return &ApprovalService{
		handler:   cfg.Handler,
		emitter:   cfg.Emitter,
		validator: cfg.Validator,
	}
}

// RequestApproval requests approval for an operation with full event emission.
// Returns the request ID, whether approved, and any error.
func (s *ApprovalService) RequestApproval(ctx context.Context, operation Operation) (reqID string, approved bool, err error) {
	// Generate unique request ID
	reqID = uuid.New().String()

	// Create approval request
	req := ApprovalRequest{
		ID:        reqID,
		Command:   operation.Command,
		Reason:    operation.Reason,
		WorkDir:   operation.WorkDir,
		Timestamp: time.Now(),
	}

	// Emit approval request event if emitter available
	if s.emitter != nil {
		s.emitApprovalRequest(req)
	}

	// If no handler, auto-deny
	if s.handler == nil {
		if s.emitter != nil {
			s.emitApprovalDenied(reqID, operation.Command, "no approval handler configured")
		}
		return reqID, false, fmt.Errorf("no approval handler configured")
	}

	// Invoke approval handler
	resp, cancelled := s.invokeHandler(ctx, req)
	if cancelled {
		if s.emitter != nil {
			s.emitApprovalDenied(reqID, operation.Command, "context cancelled")
		}
		return reqID, false, fmt.Errorf("context cancelled")
	}

	// Validate response
	if resp.RequestID != reqID {
		if s.emitter != nil {
			s.emitApprovalDenied(reqID, operation.Command, "response request ID mismatch")
		}
		return reqID, false, fmt.Errorf("request ID mismatch")
	}

	// Handle command modification if needed
	if resp.Approved && resp.ModifiedCommand != "" {
		return s.handleModifiedCommand(ctx, reqID, operation.Command, resp)
	}

	// Process approval decision
	if resp.Approved {
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
	// Invoke handler in goroutine
	respChan := make(chan ApprovalResponse, 1)
	go func() {
		resp := s.handler(req)
		respChan <- resp
	}()

	// Wait for response or context cancellation
	select {
	case resp := <-respChan:
		return resp, false
	case <-ctx.Done():
		return ApprovalResponse{}, true
	}
}

// handleModifiedCommand handles approval with a modified command.
func (s *ApprovalService) handleModifiedCommand(_ context.Context, reqID string, originalCmd *Command, resp ApprovalResponse) (string, bool, error) {
	// Parse modified command
	modCmd, err := ParseCommand(resp.ModifiedCommand)
	if err != nil {
		if s.emitter != nil {
			s.emitApprovalDenied(reqID, originalCmd, "modified command parse error: "+err.Error())
		}
		return reqID, false, fmt.Errorf("parse error: %w", err)
	}

	// Re-validate modified command if validator available
	if s.validator != nil {
		result, err := s.validator.Classify(modCmd)
		if err != nil {
			if s.emitter != nil {
				s.emitApprovalDenied(reqID, originalCmd, "modified command validation error: "+err.Error())
			}
			return reqID, false, fmt.Errorf("validation error: %w", err)
		}

		// Check if modified command is not safe
		if result.Classification != CommandSafe {
			if s.emitter != nil {
				s.emitApprovalDenied(reqID, originalCmd, "modified command failed validation: "+result.Classification.String())
			}
			return reqID, false, fmt.Errorf("modified command not safe: %s", result.Classification)
		}
	}

	// Update original command with modified version
	*originalCmd = *modCmd

	// Emit approved event with modification info
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

// RequestApprovalWithValidator requests approval for a command using a validator.
// This is a convenience method that checks if approval is needed before requesting it.
func (s *ApprovalService) RequestApprovalWithValidator(ctx context.Context, cmd *Command, validator *Validator, workDir string) (bool, error) {
	if validator == nil {
		// No validator - approve by default
		return true, nil
	}

	// Check if approval is needed
	if !validator.NeedsApproval(cmd) {
		return true, nil
	}

	// Determine reason based on classification
	result, err := validator.Classify(cmd)
	if err != nil {
		return false, fmt.Errorf("failed to classify command: %w", err)
	}

	reason := fmt.Sprintf("Command classified as %s", result.Classification.String())
	if result.Reason != "" {
		reason += fmt.Sprintf(" - %s", result.Reason)
	}

	// Request approval
	operation := Operation{
		Command: cmd,
		Reason:  reason,
		WorkDir: workDir,
	}

	_, approved, err := s.RequestApproval(ctx, operation)
	return approved, err
}

// Operation represents an operation that may require approval.
type Operation struct {
	// Command is the command to be executed (if applicable)
	Command *Command

	// Reason explains why approval is needed
	Reason string

	// WorkDir is the working directory for the operation
	WorkDir string
}

// ApprovalServiceOption is a functional option for ApprovalService.
type ApprovalServiceOption func(*ApprovalService) error
