// Package validate provides composable validation chains for configuration
// and input validation, replacing hand-rolled validation patterns across the codebase.
package validate

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Sentinel errors for validation failures.
var (
	ErrRequired    = errors.New("is required")
	ErrPositive    = errors.New("must be positive")
	ErrNonNegative = errors.New("must be non-negative")
	ErrOutOfRange  = errors.New("out of range")
	ErrNotOneOf    = errors.New("must be one of")
	ErrMinLength   = errors.New("minimum length not met")
)

// Chain accumulates validation errors.
// Add checks with the provided methods, then call Err() to get the combined result.
type Chain struct {
	errs []error
}

// NewChain creates an empty validation chain.
func NewChain() *Chain {
	return &Chain{}
}

// Required checks that the string value is non-empty.
func (c *Chain) Required(field, value string) *Chain {
	if strings.TrimSpace(value) == "" {
		c.errs = append(c.errs, fmt.Errorf("%s %w", field, ErrRequired))
	}

	return c
}

// Positive checks that the int value is positive (> 0).
func (c *Chain) Positive(field string, value int) *Chain {
	if value <= 0 {
		c.errs = append(c.errs, fmt.Errorf("%s %w, got %d", field, ErrPositive, value))
	}

	return c
}

// NonNegative checks that the int value is non-negative (>= 0).
func (c *Chain) NonNegative(field string, value int) *Chain {
	if value < 0 {
		c.errs = append(c.errs, fmt.Errorf("%s %w, got %d", field, ErrNonNegative, value))
	}

	return c
}

// InRange checks that the value is within [min, max] inclusive.
func (c *Chain) InRange(field string, value, minVal, maxVal int) *Chain {
	if value < minVal || value > maxVal {
		c.errs = append(c.errs, fmt.Errorf("%s %w: must be between %d and %d, got %d", field, ErrOutOfRange, minVal, maxVal, value))
	}

	return c
}

// InRangeFloat checks that the float value is within [min, max] inclusive.
func (c *Chain) InRangeFloat(field string, value, minVal, maxVal float64) *Chain {
	if value < minVal || value > maxVal {
		c.errs = append(c.errs, fmt.Errorf("%s %w: must be between %g and %g, got %g", field, ErrOutOfRange, minVal, maxVal, value))
	}

	return c
}

// OneOf checks that the value matches one of the allowed values.
func (c *Chain) OneOf(field, value string, allowed ...string) *Chain {
	if slices.Contains(allowed, value) {
		return c
	}

	c.errs = append(c.errs, fmt.Errorf("%s %w [%s], got %q", field, ErrNotOneOf, strings.Join(allowed, ", "), value))

	return c
}

// MinLength checks that the string has at least minLen characters.
func (c *Chain) MinLength(field, value string, minLen int) *Chain {
	if len(value) < minLen {
		c.errs = append(c.errs, fmt.Errorf("%s %w: at least %d characters, got %d", field, ErrMinLength, minLen, len(value)))
	}

	return c
}

// Check adds a custom validation. If the error is non-nil, it is appended.
func (c *Chain) Check(err error) *Chain {
	if err != nil {
		c.errs = append(c.errs, err)
	}

	return c
}

// Err returns a combined error if any validations failed, or nil.
func (c *Chain) Err() error {
	if len(c.errs) == 0 {
		return nil
	}

	return errors.Join(c.errs...)
}

// HasErrors returns true if any validation has failed.
func (c *Chain) HasErrors() bool {
	return len(c.errs) > 0
}
