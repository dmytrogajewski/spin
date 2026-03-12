package llm

import (
	"errors"
	"testing"
)

var (
	errContext = errors.New("context")
	errContext2 = errors.New("context")
	errContext3 = errors.New("context")
	errContext4 = errors.New("context")
	errContext5 = errors.New("context")
	errDifferent = errors.New("different")
)

func TestErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "ErrProviderNotFound",
			err:  ErrProviderNotFound,
			want: "LLM: provider not found",
		},
		{
			name: "ErrInvalidRequest",
			err:  ErrInvalidRequest,
			want: "LLM: invalid request",
		},
		{
			name: "ErrRateLimited",
			err:  ErrRateLimited,
			want: "LLM: rate limited",
		},
		{
			name: "ErrContextLengthExceeded",
			err:  ErrContextLengthExceeded,
			want: "LLM: context length exceeded",
		},
		{
			name: "ErrModelNotFound",
			err:  ErrModelNotFound,
			want: "LLM: model not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "wrapped ErrProviderNotFound",
			err:    errors.Join(errContext, ErrProviderNotFound),
			target: ErrProviderNotFound,
			want:   true,
		},
		{
			name:   "wrapped ErrInvalidRequest",
			err:    errors.Join(errContext2, ErrInvalidRequest),
			target: ErrInvalidRequest,
			want:   true,
		},
		{
			name:   "wrapped ErrRateLimited",
			err:    errors.Join(errContext3, ErrRateLimited),
			target: ErrRateLimited,
			want:   true,
		},
		{
			name:   "wrapped ErrContextLengthExceeded",
			err:    errors.Join(errContext4, ErrContextLengthExceeded),
			target: ErrContextLengthExceeded,
			want:   true,
		},
		{
			name:   "wrapped ErrModelNotFound",
			err:    errors.Join(errContext5, ErrModelNotFound),
			target: ErrModelNotFound,
			want:   true,
		},
		{
			name:   "different error",
			err:    errDifferent,
			target: ErrProviderNotFound,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorEquality(t *testing.T) {
	t.Parallel()

	// Verify all errors are distinct.
	errs := []error{
		ErrProviderNotFound,
		ErrInvalidRequest,
		ErrRateLimited,
		ErrContextLengthExceeded,
		ErrModelNotFound,
	}

	for i, err1 := range errs {
		for j, err2 := range errs {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("Error %d and %d are the same: %v == %v", i, j, err1, err2)
			}
		}
	}
}
