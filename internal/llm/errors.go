package llm

import "errors"

// Common errors for LLM operations.
var (
	// ErrProviderNotFound is returned when a requested provider is not found.
	ErrProviderNotFound = errors.New("provider not found")

	// ErrInvalidRequest is returned when the completion request is invalid.
	// This can occur due to missing required fields, invalid parameter values,
	// or other request validation failures.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrRateLimited is returned when the provider's rate limit is exceeded.
	// Callers should implement exponential backoff and retry logic.
	ErrRateLimited = errors.New("rate limited")

	// ErrContextLengthExceeded is returned when the request exceeds the model's
	// maximum context length. Callers should reduce the number of messages or tokens.
	ErrContextLengthExceeded = errors.New("context length exceeded")

	// ErrModelNotFound is returned when the requested model is not available
	// or does not exist for the provider.
	ErrModelNotFound = errors.New("model not found")
)
