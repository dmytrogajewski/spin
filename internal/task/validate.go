package task

import "fmt"

// validateMaxTokens validates the max tokens budget.
// Returns ErrMaxTokensMustBePositive if maxTokens is non-positive,
// or ErrMaxTokensExceedsMaximumAllowed if it exceeds MaxAllowedTokens.
func validateMaxTokens(maxTokens int) error {
	if maxTokens <= 0 {
		return ErrMaxTokensMustBePositive
	}

	if maxTokens > MaxAllowedTokens {
		return fmt.Errorf("max tokens %d exceeds maximum allowed %d: %w", maxTokens, MaxAllowedTokens, ErrMaxTokensExceedsMaximumAllowed)
	}

	return nil
}
