package tui

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
)

// TestClassifySeverity tests error severity classification from core.ErrorData.
func TestClassifySeverity(t *testing.T) {
	tests := []struct {
		name     string
		errData  core.ErrorData
		expected ErrorSeverity
	}{
		{
			name: "internal error is critical",
			errData: core.ErrorData{
				Code: core.ErrCodeInternal.String(),
			},
			expected: SeverityCritical,
		},
		{
			name: "timeout is warning",
			errData: core.ErrorData{
				Code: core.ErrCodeTimeout.String(),
			},
			expected: SeverityWarning,
		},
		{
			name: "cancelled is info",
			errData: core.ErrorData{
				Code: core.ErrCodeCancelled.String(),
			},
			expected: SeverityInfo,
		},
		{
			name: "permission denied is error",
			errData: core.ErrorData{
				Code: core.ErrCodePermissionDenied.String(),
			},
			expected: SeverityError,
		},
		{
			name: "external is error",
			errData: core.ErrorData{
				Code: core.ErrCodeExternal.String(),
			},
			expected: SeverityError,
		},
		{
			name: "invalid input is warning",
			errData: core.ErrorData{
				Code: core.ErrCodeInvalidInput.String(),
			},
			expected: SeverityWarning,
		},
		{
			name: "not found is warning",
			errData: core.ErrorData{
				Code: core.ErrCodeNotFound.String(),
			},
			expected: SeverityWarning,
		},
		{
			name: "already exists is warning",
			errData: core.ErrorData{
				Code: core.ErrCodeAlreadyExists.String(),
			},
			expected: SeverityWarning,
		},
		{
			name: "unknown is error",
			errData: core.ErrorData{
				Code: core.ErrCodeUnknown.String(),
			},
			expected: SeverityError,
		},
		{
			name: "empty code is error",
			errData: core.ErrorData{
				Code: "",
			},
			expected: SeverityError,
		},
		{
			name: "unrecognized code is error",
			errData: core.ErrorData{
				Code: "invalid_code",
			},
			expected: SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifySeverity(tt.errData)
			if got != tt.expected {
				t.Errorf("ClassifySeverity() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestNewErrorDisplay tests ErrorDisplay creation from core.ErrorData.
func TestNewErrorDisplay(t *testing.T) {
	tests := []struct {
		name       string
		errData    core.ErrorData
		wantMsg    string
		wantCode   string
		wantSev    ErrorSeverity
		wantDismis bool
		wantAuto   time.Duration
	}{
		{
			name: "critical error - not dismissible, no auto-dismiss",
			errData: core.ErrorData{
				Message: "System failure",
				Code:    core.ErrCodeInternal.String(),
				Details: "Internal server error",
			},
			wantMsg:    "System failure",
			wantCode:   "internal",
			wantSev:    SeverityCritical,
			wantDismis: true, // User can dismiss critical errors
			wantAuto:   0,    // No auto-dismiss for critical
		},
		{
			name: "warning - dismissible with auto-dismiss",
			errData: core.ErrorData{
				Message: "File not found",
				Code:    core.ErrCodeNotFound.String(),
				Details: "config.yaml does not exist",
			},
			wantMsg:    "File not found",
			wantCode:   "not_found",
			wantSev:    SeverityWarning,
			wantDismis: true,
			wantAuto:   5 * time.Second,
		},
		{
			name: "info - dismissible with shorter auto-dismiss",
			errData: core.ErrorData{
				Message: "Operation cancelled",
				Code:    core.ErrCodeCancelled.String(),
			},
			wantMsg:    "Operation cancelled",
			wantCode:   "cancelled",
			wantSev:    SeverityInfo,
			wantDismis: true,
			wantAuto:   3 * time.Second,
		},
		{
			name: "error - not auto-dismiss",
			errData: core.ErrorData{
				Message: "Permission denied",
				Code:    core.ErrCodePermissionDenied.String(),
				Details: "Cannot access /etc/shadow",
			},
			wantMsg:    "Permission denied",
			wantCode:   "permission_denied",
			wantSev:    SeverityError,
			wantDismis: true,
			wantAuto:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewErrorDisplay(tt.errData)

			if got.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", got.Message, tt.wantMsg)
			}
			if got.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", got.Code, tt.wantCode)
			}
			if got.Severity != tt.wantSev {
				t.Errorf("Severity = %v, want %v", got.Severity, tt.wantSev)
			}
			if got.Dismissible != tt.wantDismis {
				t.Errorf("Dismissible = %v, want %v", got.Dismissible, tt.wantDismis)
			}
			if got.AutoDismiss != tt.wantAuto {
				t.Errorf("AutoDismiss = %v, want %v", got.AutoDismiss, tt.wantAuto)
			}
			if got.Details != tt.errData.Details {
				t.Errorf("Details = %q, want %q", got.Details, tt.errData.Details)
			}
			if got.Dismissed {
				t.Error("Dismissed should be false on creation")
			}
			if got.Timestamp.IsZero() {
				t.Error("Timestamp should be set")
			}
		})
	}
}

// TestErrorDisplay_ShouldShowInModal tests modal display criteria.
func TestErrorDisplay_ShouldShowInModal(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		want     bool
	}{
		{"critical shows in modal", SeverityCritical, true},
		{"error does not show in modal", SeverityError, false},
		{"warning does not show in modal", SeverityWarning, false},
		{"info does not show in modal", SeverityInfo, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ErrorDisplay{Severity: tt.severity}
			got := e.ShouldShowInModal()
			if got != tt.want {
				t.Errorf("ShouldShowInModal() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrorDisplay_ShouldShowInStatusBar tests status bar display criteria.
func TestErrorDisplay_ShouldShowInStatusBar(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		want     bool
	}{
		{"info shows in status bar", SeverityInfo, true},
		{"warning shows in status bar", SeverityWarning, true},
		{"error does not show in status bar", SeverityError, false},
		{"critical does not show in status bar", SeverityCritical, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ErrorDisplay{Severity: tt.severity}
			got := e.ShouldShowInStatusBar()
			if got != tt.want {
				t.Errorf("ShouldShowInStatusBar() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrorDisplay_ShouldShowInTranscript tests transcript display criteria.
func TestErrorDisplay_ShouldShowInTranscript(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		want     bool
	}{
		{"error shows in transcript", SeverityError, true},
		{"critical shows in transcript", SeverityCritical, true},
		{"warning does not show in transcript", SeverityWarning, false},
		{"info does not show in transcript", SeverityInfo, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ErrorDisplay{Severity: tt.severity}
			got := e.ShouldShowInTranscript()
			if got != tt.want {
				t.Errorf("ShouldShowInTranscript() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrorDisplay_IsExpired tests auto-dismiss expiration.
func TestErrorDisplay_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		autoDismiss time.Duration
		timestamp   time.Time
		currentTime time.Time
		want        bool
	}{
		{
			name:        "not expired - within time",
			autoDismiss: 5 * time.Second,
			timestamp:   now,
			currentTime: now.Add(3 * time.Second),
			want:        false,
		},
		{
			name:        "expired - past time",
			autoDismiss: 5 * time.Second,
			timestamp:   now,
			currentTime: now.Add(6 * time.Second),
			want:        true,
		},
		{
			name:        "no auto-dismiss - never expires",
			autoDismiss: 0,
			timestamp:   now,
			currentTime: now.Add(1 * time.Hour),
			want:        false,
		},
		{
			name:        "exactly at expiration - is expired",
			autoDismiss: 5 * time.Second,
			timestamp:   now,
			currentTime: now.Add(5 * time.Second),
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ErrorDisplay{
				AutoDismiss: tt.autoDismiss,
				Timestamp:   tt.timestamp,
			}
			got := e.IsExpired(tt.currentTime)
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrorDisplay_TimeUntilDismiss tests remaining time calculation.
func TestErrorDisplay_TimeUntilDismiss(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		autoDismiss time.Duration
		timestamp   time.Time
		currentTime time.Time
		want        time.Duration
	}{
		{
			name:        "3 seconds remaining",
			autoDismiss: 5 * time.Second,
			timestamp:   now,
			currentTime: now.Add(2 * time.Second),
			want:        3 * time.Second,
		},
		{
			name:        "already expired - zero remaining",
			autoDismiss: 5 * time.Second,
			timestamp:   now,
			currentTime: now.Add(6 * time.Second),
			want:        0,
		},
		{
			name:        "no auto-dismiss - zero remaining",
			autoDismiss: 0,
			timestamp:   now,
			currentTime: now.Add(1 * time.Hour),
			want:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ErrorDisplay{
				AutoDismiss: tt.autoDismiss,
				Timestamp:   tt.timestamp,
			}
			got := e.TimeUntilDismiss(tt.currentTime)
			if got != tt.want {
				t.Errorf("TimeUntilDismiss() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrorDisplay_Dismiss tests manual dismissal.
func TestErrorDisplay_Dismiss(t *testing.T) {
	e := ErrorDisplay{
		Dismissible: true,
		Dismissed:   false,
	}

	if e.Dismissed {
		t.Error("Should not be dismissed initially")
	}

	e.Dismiss()

	if !e.Dismissed {
		t.Error("Should be dismissed after Dismiss() called")
	}
}

// TestErrorSeverity_String tests severity string representation.
func TestErrorSeverity_String(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		want     string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{SeverityCritical, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.severity.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestErrorSeverity_Icon tests severity icon representation.
func TestErrorSeverity_Icon(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		want     string
	}{
		{SeverityInfo, "ℹ️"},
		{SeverityWarning, "⚠️"},
		{SeverityError, "❌"},
		{SeverityCritical, "🔥"},
	}

	for _, tt := range tests {
		t.Run(tt.severity.String(), func(t *testing.T) {
			got := tt.severity.Icon()
			if got != tt.want {
				t.Errorf("Icon() = %q, want %q", got, tt.want)
			}
		})
	}
}
