package llm

import (
	"fmt"
	"net/http"
	"strings"
)

// ErrorMapper provides standardized error mapping for all LLM providers.
// This ensures consistent error handling across different providers.
type ErrorMapper struct {
	providerName string
}

// NewErrorMapper creates a new error mapper for a specific provider.
func NewErrorMapper(providerName string) *ErrorMapper {
	return &ErrorMapper{
		providerName: providerName,
	}
}

// MapError maps provider-specific errors to standardized LLM errors.
func (em *ErrorMapper) MapError(err error) error {
	if err == nil {
		return nil
	}

	// Handle HTTP errors
	if httpErr, ok := err.(*HTTPError); ok {
		return em.mapHTTPError(httpErr)
	}

	// Handle common error patterns
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "rate limit"):
		return fmt.Errorf("%w: %s", ErrRateLimited, err)
	case strings.Contains(errStr, "context length"):
		return fmt.Errorf("%w: %s", ErrContextLengthExceeded, err)
	case strings.Contains(errStr, "model not found"):
		return fmt.Errorf("%w: %s", ErrModelNotFound, err)
	case strings.Contains(errStr, "invalid request"):
		return fmt.Errorf("%w: %s", ErrInvalidRequest, err)
	case strings.Contains(errStr, "unauthorized"):
		return fmt.Errorf("%w: %s", ErrInvalidRequest, err)
	case strings.Contains(errStr, "forbidden"):
		return fmt.Errorf("%w: %s", ErrInvalidRequest, err)
	case strings.Contains(errStr, "not found"):
		return fmt.Errorf("%w: %s", ErrProviderNotFound, err)
	default:
		return fmt.Errorf("%s error: %w", em.providerName, err)
	}
}

// mapHTTPError maps HTTP errors to standardized LLM errors.
func (em *ErrorMapper) mapHTTPError(httpErr *HTTPError) error {
	switch httpErr.StatusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrInvalidRequest, httpErr)
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", ErrInvalidRequest, httpErr)
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrInvalidRequest, httpErr)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrProviderNotFound, httpErr)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", ErrRateLimited, httpErr)
	case http.StatusInternalServerError:
		return fmt.Errorf("%s server error: %w", em.providerName, httpErr)
	case http.StatusBadGateway:
		return fmt.Errorf("%s gateway error: %w", em.providerName, httpErr)
	case http.StatusServiceUnavailable:
		return fmt.Errorf("%s service unavailable: %w", em.providerName, httpErr)
	default:
		return fmt.Errorf("%s HTTP error %d: %w", em.providerName, httpErr.StatusCode, httpErr)
	}
}

// HTTPError represents an HTTP error with status code and body.
type HTTPError struct {
	StatusCode int
	Body       string
	Message    string
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// ProviderError represents a provider-specific error with context.
type ProviderError struct {
	Provider string
	Code     string
	Message  string
	Details  map[string]interface{}
	Err      error
}
