package bullet

import (
	"fmt"
)

const (
	// MaxContentLength is the maximum allowed length for bullet content.
	MaxContentLength = 2048
)

// validate checks if the bullet meets all validation requirements.
func validate(b *Bullet) error {
	if len(b.Content) > MaxContentLength {
		return fmt.Errorf("content length %d exceeds maximum %d", len(b.Content), MaxContentLength)
	}

	if b.HelpfulCount < 0 {
		return fmt.Errorf("helpful count cannot be negative: %d", b.HelpfulCount)
	}

	if b.HarmfulCount < 0 {
		return fmt.Errorf("harmful count cannot be negative: %d", b.HarmfulCount)
	}

	return nil
}
