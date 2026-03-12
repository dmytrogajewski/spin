package llm

import (
	"github.com/dmytrogajewski/spin/internal/apperr"
)

// Common errors for LLM operations.
//
// These are now structured errors with proper error codes for type-safe handling.
var (
	// ErrProviderNotFound is returned when a requested provider is not found.
	ErrProviderNotFound = apperr.New(apperr.CodeNotFound, "LLM", "provider not found", nil)

	// ErrInvalidRequest is returned when the completion request is invalid.
	// This can occur due to missing required fields, invalid parameter values,
	// or other request validation failures.
	ErrInvalidRequest = apperr.New(apperr.CodeValidation, "LLM", "invalid request", nil)

	// ErrRateLimited is returned when the provider's rate limit is exceeded.
	// Callers should implement exponential backoff and retry logic.
	ErrRateLimited = apperr.New(apperr.CodeLLM, "LLM", "rate limited", nil)

	// ErrContextLengthExceeded is returned when the request exceeds the model's
	// maximum context length. Callers should reduce the number of messages or tokens.
	ErrContextLengthExceeded = apperr.New(apperr.CodeLLM, "LLM", "context length exceeded", nil)

	// ErrModelNotFound is returned when the requested model is not available
	// or does not exist for the provider.
	ErrModelNotFound = apperr.New(apperr.CodeNotFound, "LLM", "model not found", nil)
)
