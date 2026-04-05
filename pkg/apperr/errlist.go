package apperr

import "errors"

// ErrorList accumulates multiple errors for batch validation.
// The zero value is ready to use.
type ErrorList []error

// Add appends a non-nil error to the list. Nil errors are silently skipped.
func (el *ErrorList) Add(err error) {
	if err != nil {
		*el = append(*el, err)
	}
}

// HasErrors returns true if any errors have been added.
func (el *ErrorList) HasErrors() bool {
	return len(*el) > 0
}

// Err returns nil if the list is empty, otherwise returns a combined error
// via [errors.Join] that supports [errors.Is] and [errors.As] for each element.
func (el *ErrorList) Err() error {
	if len(*el) == 0 {
		return nil
	}

	return errors.Join(*el...)
}
