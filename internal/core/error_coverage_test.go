package core

import (
	"errors"
	"testing"
)

func TestError_Error_AllCodes(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "with op only",
			err: &Error{
				Code: ErrCodeInternal,
				Op:   "internal error occurred",
			},
			want: "internal error occurred",
		},
		{
			name: "with wrapped error",
			err: &Error{
				Code: ErrCodeInternal,
				Op:   "internal error",
				Err:  errors.New("wrapped error"),
			},
			want: "internal error: wrapped error",
		},
		{
			name: "empty op with wrapped error",
			err: &Error{
				Code: ErrCodeInternal,
				Op:   "",
				Err:  errors.New("base error"),
			},
			want: ": base error", // Empty op results in ": error"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.want {
				t.Errorf("Error() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestError_Is_AllCases(t *testing.T) {
	tests := []struct {
		name   string
		err    *Error
		target error
		want   bool
	}{
		{
			name: "matches by code",
			err: &Error{
				Code: ErrCodeInvalidInput,
				Op:   "invalid",
			},
			target: &Error{
				Code: ErrCodeInvalidInput,
				Op:   "also invalid",
			},
			want: true,
		},
		{
			name: "does not match different code",
			err: &Error{
				Code: ErrCodeInternal,
				Op:   "internal",
			},
			target: &Error{
				Code: ErrCodeInvalidInput,
				Op:   "invalid",
			},
			want: false,
		},
		{
			name: "does not match non-Error type",
			err: &Error{
				Code: ErrCodeInternal,
				Op:   "internal",
			},
			target: errors.New("different error"),
			want:   false,
		},
		{
			name: "unknown code does not match",
			err: &Error{
				Code: ErrCodeUnknown,
				Op:   "unknown",
			},
			target: &Error{
				Code: ErrCodeUnknown,
				Op:   "also unknown",
			},
			want: false,
		},
		{
			name: "matches timeout code",
			err: &Error{
				Code: ErrCodeTimeout,
				Op:   "timeout1",
			},
			target: &Error{
				Code: ErrCodeTimeout,
				Op:   "timeout2",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Is(tt.target)
			if result != tt.want {
				t.Errorf("Is() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestErrorCode_String_AllCodes(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want string
	}{
		{ErrCodeUnknown, "unknown"},
		{ErrCodeInternal, "internal"},
		{ErrCodeInvalidInput, "invalid_input"},
		{ErrCodeNotFound, "not_found"},
		{ErrCodeTimeout, "timeout"},
		{ErrCodeCancelled, "cancelled"},
		{ErrCodePermissionDenied, "permission_denied"},
		{ErrCodeExternal, "external"},
		{ErrCodeAlreadyExists, "already_exists"},
		{ErrorCode(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			result := tt.code.String()
			if result != tt.want {
				t.Errorf("String() = %q, want %q", result, tt.want)
			}
		})
	}
}
