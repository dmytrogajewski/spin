package bullet

import (
	"errors"
	"fmt"
)

var (
	ErrContentLengthExceedsMaximum = errors.New("content length  exceeds maximum")
	ErrHelpfulCountCannotBeNegative = errors.New("helpful count cannot be negative")
	ErrHarmfulCountCannotBeNegative = errors.New("harmful count cannot be negative")
)

const (
	// MaxContentLength is the maximum allowed length for bullet content.
	MaxContentLength = 2048
)

// validate checks if the bullet meets all validation requirements.
func validate(b *Bullet) error {
	if len(b.Content) > MaxContentLength {
return fmt.Errorf("content length %d exceeds maximum %d: %w", len(b.Content), MaxContentLength, ErrContentLengthExceedsMaximum)
	}

	if b.HelpfulCount < 0 {
return fmt.Errorf("helpful count cannot be negative: %d: %w", b.HelpfulCount, ErrHelpfulCountCannotBeNegative)
	}

	if b.HarmfulCount < 0 {
return fmt.Errorf("harmful count cannot be negative: %d: %w", b.HarmfulCount, ErrHarmfulCountCannotBeNegative)
	}

	return nil
}
