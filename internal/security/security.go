package security

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrCommandCannotBeNil is a sentinel error.
	ErrCommandCannotBeNil = errors.New("command cannot be nil")
	// ErrValidatorNotConfigured is a sentinel error.
	ErrValidatorNotConfigured = errors.New("validator not configured")
	// ErrApprovalServiceNotConfigured is a sentinel error.
	ErrApprovalServiceNotConfigured = errors.New("approval service not configured")
)

// Service handles all security-related operations including
// command validation and approval management.
//
// This service centralizes security logic that was previously scattered
// across Agent and other components. It provides a clean interface for
// validating commands and requesting user approval for dangerous operations.
type Service struct {
	validator       *Validator
	approvalService *ApprovalService
}

// NewService creates a new security service with the given dependencies.
//
// Both validator and approvalService can be nil, though the service will
// return errors when methods requiring these dependencies are called.
func NewService(validator *Validator, approvalService *ApprovalService) *Service {
	return &Service{
		validator:       validator,
		approvalService: approvalService,
	}
}

// ValidateCommand validates a command and returns its classification.
//
// The classification determines whether the command is safe, neutral, or dangerous.
// Returns an error if the validator is not configured or if the command is nil.
func (s *Service) ValidateCommand(cmd *Command) (*ValidationResult, error) {
	if cmd == nil {
		return nil, ErrCommandCannotBeNil
	}

	if s.validator == nil {
		return nil, ErrValidatorNotConfigured
	}

	result, err := s.validator.Classify(cmd)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return result, nil
}

// NeedsApproval checks if a command requires user approval.
//
// Returns true if the command is classified as dangerous and requires approval.
// Returns false if the validator is not configured (fail-open for testing).
func (s *Service) NeedsApproval(cmd *Command) bool {
	if s.validator == nil {
		return false
	}

	return s.validator.NeedsApproval(cmd)
}

// RequestApproval requests user approval for an operation.
//
// This method delegates to the underlying ApprovalService. If no approval
// service is configured, it returns an error and denies the operation.
//
// Returns true if the operation was approved, false if denied, and an error
// if the approval process failed (timeout, validation error, etc.).
func (s *Service) RequestApproval(ctx context.Context, operation Operation) (bool, error) {
	if s.approvalService == nil {
		return false, ErrApprovalServiceNotConfigured
	}

	_, approved, err := s.approvalService.RequestApproval(ctx, operation)

	return approved, err
}

// ValidateAndApprove is a convenience method that validates a command and
// requests approval if needed.
//
// This combines validation and approval into a single call, making it easier
// to use in execution flows. It handles the following logic:
//  1. Validate the command
//  2. If safe, return approved=true without requesting approval
//  3. If forbidden, return approved=false (forbidden commands should be blocked)
//  4. If dangerous/interactive/unverified, request approval from user
//  5. Return the approval decision
//
// Returns true if the command is approved (either safe or explicitly approved),
// false if denied or forbidden, and an error if validation or approval fails.
func (s *Service) ValidateAndApprove(ctx context.Context, cmd *Command, workDir string) (bool, error) {
	// Validate the command first.
	result, err := s.ValidateCommand(cmd)
	if err != nil {
		return false, fmt.Errorf("validation failed: %w", err)
	}

	// If the command is safe, no approval needed.
	if result.Classification == CommandSafe {
		return true, nil
	}

	// If the command is forbidden, block it (don't even request approval).
	if result.Classification == CommandForbidden {
		return false, nil
	}

	// Check if approval is needed.
	if !s.NeedsApproval(cmd) {
		return true, nil
	}

	// Dangerous/interactive/unverified command that needs approval.
	operation := NewOperation(cmd, fmt.Sprintf("Command classified as %s: %s", result.Classification, result.Reason), workDir)

	approved, err := s.RequestApproval(ctx, operation)
	if err != nil {
		return false, fmt.Errorf("approval request failed: %w", err)
	}

	return approved, nil
}

// ApprovalService returns the approval service instance.
// This method allows access to the ApprovalService for components that need it directly,
// such as ToolRuntime.
func (s *Service) ApprovalService() *ApprovalService {
	return s.approvalService
}

// Validator returns the validator instance.
// This method allows access to the Validator for components that need it directly,
// such as ToolRuntime.
func (s *Service) Validator() *Validator {
	return s.validator
}

// SetApprovalService updates the approval service instance.
// This allows updating the approval service after creation, which is useful
// when the approval handler is configured (e.g., in ACP mode).
func (s *Service) SetApprovalService(service *ApprovalService) {
	s.approvalService = service
}
