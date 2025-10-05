package tui

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
)

// ErrorSeverity represents the severity level of an error.
type ErrorSeverity int

const (
	// SeverityInfo represents informational messages (low severity).
	SeverityInfo ErrorSeverity = iota
	// SeverityWarning represents warnings (recoverable issues).
	SeverityWarning
	// SeverityError represents errors (operation failed).
	SeverityError
	// SeverityCritical represents critical errors (system failure).
	SeverityCritical
)

// String returns the string representation of ErrorSeverity.
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Icon returns the emoji icon for the severity level.
func (s ErrorSeverity) Icon() string {
	switch s {
	case SeverityInfo:
		return "ℹ️"
	case SeverityWarning:
		return "⚠️"
	case SeverityError:
		return "❌"
	case SeverityCritical:
		return "🔥"
	default:
		return "❓"
	}
}

// ErrorDisplay represents an error for display in the TUI.
// It contains all information needed to render the error appropriately
// based on severity, with support for auto-dismissal and user interaction.
type ErrorDisplay struct {
	// Message is the user-friendly error message.
	Message string
	// Code is the error code from core.ErrorCode.
	Code string
	// Details contains technical details and stack traces.
	Details string
	// Operation is the operation that failed (e.g., "Agent.RunTurn").
	Operation string
	// Severity is the severity level of the error.
	Severity ErrorSeverity
	// Timestamp is when the error occurred.
	Timestamp time.Time
	// Dismissible indicates if the user can manually dismiss the error.
	Dismissible bool
	// Dismissed indicates if the user has dismissed the error.
	Dismissed bool
	// AutoDismiss is the duration after which to auto-dismiss (0 = never).
	AutoDismiss time.Duration
}

// ClassifySeverity maps core error codes to severity levels.
// This determines how the error is displayed in the TUI.
func ClassifySeverity(errData core.ErrorData) ErrorSeverity {
	switch errData.Code {
	// Critical errors - require immediate attention, show modal
	case core.ErrCodeInternal.String():
		return SeverityCritical

	// Regular errors - show inline in transcript
	case core.ErrCodePermissionDenied.String(),
		core.ErrCodeExternal.String(),
		core.ErrCodeUnknown.String():
		return SeverityError

	// Warnings - show in status bar with auto-dismiss
	case core.ErrCodeInvalidInput.String(),
		core.ErrCodeNotFound.String(),
		core.ErrCodeAlreadyExists.String(),
		core.ErrCodeTimeout.String():
		return SeverityWarning

	// Info - show in status bar with shorter auto-dismiss
	case core.ErrCodeCancelled.String():
		return SeverityInfo

	// Default to error for unknown/empty codes
	default:
		return SeverityError
	}
}

// NewErrorDisplay creates an ErrorDisplay from core.ErrorData.
// It automatically sets severity, auto-dismiss duration, and other properties
// based on the error code.
func NewErrorDisplay(errData core.ErrorData) ErrorDisplay {
	severity := ClassifySeverity(errData)

	// Determine auto-dismiss duration based on severity
	var autoDismiss time.Duration
	switch severity {
	case SeverityInfo:
		autoDismiss = 3 * time.Second
	case SeverityWarning:
		autoDismiss = 5 * time.Second
	case SeverityError, SeverityCritical:
		autoDismiss = 0 // No auto-dismiss for errors and critical
	}

	return ErrorDisplay{
		Message:     errData.Message,
		Code:        errData.Code,
		Details:     errData.Details,
		Severity:    severity,
		Timestamp:   time.Now(),
		Dismissible: true, // All errors are dismissible
		Dismissed:   false,
		AutoDismiss: autoDismiss,
	}
}

// ShouldShowInModal returns true if the error should display in a modal overlay.
// Currently, only critical errors show in modals.
func (e ErrorDisplay) ShouldShowInModal() bool {
	return e.Severity == SeverityCritical
}

// ShouldShowInStatusBar returns true if the error should display in the status bar.
// Info and warning severity errors are transient and show in the status bar.
func (e ErrorDisplay) ShouldShowInStatusBar() bool {
	return e.Severity == SeverityInfo || e.Severity == SeverityWarning
}

// ShouldShowInTranscript returns true if the error should display inline in transcript.
// Error and critical severity errors show in the conversation transcript.
func (e ErrorDisplay) ShouldShowInTranscript() bool {
	return e.Severity == SeverityError || e.Severity == SeverityCritical
}

// IsExpired returns true if the error has passed its auto-dismiss duration.
// If AutoDismiss is 0, it never expires.
func (e ErrorDisplay) IsExpired(now time.Time) bool {
	if e.AutoDismiss == 0 {
		return false
	}
	return now.Sub(e.Timestamp) >= e.AutoDismiss
}

// TimeUntilDismiss returns the remaining time until auto-dismiss.
// Returns 0 if already expired or no auto-dismiss is set.
func (e ErrorDisplay) TimeUntilDismiss(now time.Time) time.Duration {
	if e.AutoDismiss == 0 {
		return 0
	}
	elapsed := now.Sub(e.Timestamp)
	remaining := e.AutoDismiss - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Dismiss marks the error as dismissed by the user.
func (e *ErrorDisplay) Dismiss() {
	e.Dismissed = true
}
