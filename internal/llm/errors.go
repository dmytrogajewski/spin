package llm

import (
	"github.com/dmytrogajewski/spin/internal/errors"
)

// Common errors for LLM operations.
//
// These are now structured errors with proper error codes for type-safe handling.
var (
	// ErrProviderNotFound is returned when a requested provider is not found.
	ErrProviderNotFound = errors.New(errors.CodeNotFound, "LLM", "provider not found", nil)

	// ErrInvalidRequest is returned when the completion request is invalid.
	// This can occur due to missing required fields, invalid parameter values,
	// or other request validation failures.
	ErrInvalidRequest = errors.New(errors.CodeValidation, "LLM", "invalid request", nil)

	// ErrRateLimited is returned when the provider's rate limit is exceeded.
	// Callers should implement exponential backoff and retry logic.
	ErrRateLimited = errors.New(errors.CodeLLM, "LLM", "rate limited", nil)

	// ErrContextLengthExceeded is returned when the request exceeds the model's
	// maximum context length. Callers should reduce the number of messages or tokens.
	ErrContextLengthExceeded = errors.New(errors.CodeLLM, "LLM", "context length exceeded", nil)

	// ErrModelNotFound is returned when the requested model is not available
	// or does not exist for the provider.
	ErrModelNotFound = errors.New(errors.CodeNotFound, "LLM", "model not found", nil)
)
