package task

// Journey: specs/journeys/JOURNEY-dedup-task-validate.md.

import (
	"errors"
	"testing"
)

func TestValidateMaxTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		maxTokens int
		wantErr   error
	}{
		{
			name:      "zero returns error",
			maxTokens: 0,
			wantErr:   ErrMaxTokensMustBePositive,
		},
		{
			name:      "negative returns error",
			maxTokens: -1,
			wantErr:   ErrMaxTokensMustBePositive,
		},
		{
			name:      "valid value returns nil",
			maxTokens: DefaultCompactMaxTokens,
			wantErr:   nil,
		},
		{
			name:      "exactly MaxAllowedTokens returns nil",
			maxTokens: MaxAllowedTokens,
			wantErr:   nil,
		},
		{
			name:      "exceeds MaxAllowedTokens returns error",
			maxTokens: MaxAllowedTokens + 1,
			wantErr:   ErrMaxTokensExceedsMaximumAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateMaxTokens(tt.maxTokens)

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validateMaxTokens(%d) unexpected error: %v", tt.maxTokens, err)
				}

				return
			}

			if err == nil {
				t.Errorf("validateMaxTokens(%d) = nil, want error wrapping %v", tt.maxTokens, tt.wantErr)

				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateMaxTokens(%d) = %v, want error wrapping %v", tt.maxTokens, err, tt.wantErr)
			}
		})
	}
}
