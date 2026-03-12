package openai

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// mapError maps SDK errors to standard llm errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	// Context errors (pass-through).
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("timeout: %w", err)
	}

	// SDK API errors.
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401, 403:
			return fmt.Errorf("unauthorized: %w", err)
		case 429:
			return fmt.Errorf("%w: %v", llm.ErrRateLimited, err)
		case 400, 404, 422:
			return fmt.Errorf("%w: %v", llm.ErrInvalidRequest, err)
		case 500, 502, 503:
			return fmt.Errorf("server error: %w", err)
		default:
			return fmt.Errorf("openai api error (status %d): %w", apiErr.StatusCode, err)
		}
	}

	// Network errors.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Errorf("connection error: %w", err)
	}

	// Unknown error - wrap with context.
	return fmt.Errorf("openai: %w", err)
}
