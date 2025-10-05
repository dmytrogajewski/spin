package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// HTTPClient wraps http.Client with retry logic and exponential backoff.
// It automatically retries requests on retryable errors (429, 503, 504)
// with configurable retry parameters.
type HTTPClient struct {
	client     *http.Client
	maxRetries int
	retryDelay time.Duration
}

// NewHTTPClient creates an HTTP client with retry logic.
// Default configuration:
//   - Timeout: 5 minutes
//   - Max retries: 3
//   - Retry delay: 1 second (with exponential backoff)
//
// The client can be customized using functional options:
//
//	client := NewHTTPClient(
//	    WithTimeout(2 * time.Minute),
//	    WithMaxRetries(5),
//	    WithRetryDelay(2 * time.Second),
//	)
func NewHTTPClient(opts ...ClientOption) *HTTPClient {
	c := &HTTPClient{
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
		maxRetries: 3,
		retryDelay: time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Do executes HTTP request with retry logic and exponential backoff.
//
// Retry behavior:
//   - Retries on 429 (Too Many Requests), 503 (Service Unavailable), 504 (Gateway Timeout)
//   - Uses exponential backoff: delay * 2^(attempt-1)
//   - Respects Retry-After header for 429 responses
//   - Buffers request body for retries
//   - Respects context cancellation
//
// Example:
//
//	req, _ := http.NewRequest("GET", "https://api.example.com/v1/models", nil)
//	resp, err := client.Do(req)
//	if err != nil {
//	    return err
//	}
//	defer resp.Body.Close()
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	bodyBytes, err := c.bufferRequestBody(req)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := c.waitForRetry(req.Context(), attempt); err != nil {
			return nil, err
		}

		c.restoreRequestBody(req, bodyBytes)

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if !c.isRetryable(resp.StatusCode) {
			return resp, nil
		}

		newResp, err := c.handleRetryAfter(req, resp, bodyBytes)
		if err != nil {
			lastErr = err
			continue
		}

		if newResp != nil {
			return newResp, nil
		}

		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// bufferRequestBody reads and buffers request body for retries
func (c *HTTPClient) bufferRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	req.Body.Close()
	return bodyBytes, nil
}

// waitForRetry implements exponential backoff delay
func (c *HTTPClient) waitForRetry(ctx context.Context, attempt int) error {
	if attempt == 0 {
		return nil
	}

	delay := c.calculateBackoff(attempt)
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// calculateBackoff calculates exponential backoff delay
func (c *HTTPClient) calculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	// Calculate shift with overflow protection
	shift := attempt - 1
	if shift > 30 { // Prevent overflow
		shift = 30
	}
	// #nosec G115 -- shift is bounded to [0, 30] above
	return c.retryDelay * time.Duration(1<<uint(shift))
}

// restoreRequestBody restores buffered body to request
func (c *HTTPClient) restoreRequestBody(req *http.Request, bodyBytes []byte) {
	if bodyBytes != nil {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
}

// handleRetryAfter handles Retry-After header for rate limiting
func (c *HTTPClient) handleRetryAfter(req *http.Request, resp *http.Response, bodyBytes []byte) (*http.Response, error) {
	if resp.StatusCode != http.StatusTooManyRequests {
		return nil, nil
	}

	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return nil, nil
	}

	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		return nil, nil
	}

	waitDuration := time.Duration(seconds) * time.Second
	resp.Body.Close()

	select {
	case <-time.After(waitDuration):
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}

	c.restoreRequestBody(req, bodyBytes)
	newResp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if !c.isRetryable(newResp.StatusCode) {
		return newResp, nil
	}

	newResp.Body.Close()
	return nil, nil
}

// isRetryable returns true for status codes that should be retried.
// Retryable status codes:
//   - 429 Too Many Requests (rate limiting)
//   - 503 Service Unavailable (temporary server issue)
//   - 504 Gateway Timeout (upstream timeout)
func (c *HTTPClient) isRetryable(code int) bool {
	return code == http.StatusTooManyRequests ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout
}

// ClientOption configures HTTPClient.
type ClientOption func(*HTTPClient)

// WithTimeout sets the request timeout for the HTTP client.
// The timeout applies to the entire request including retries.
//
// Example:
//
//	client := NewHTTPClient(WithTimeout(2 * time.Minute))
func WithTimeout(d time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.client.Timeout = d
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
// The total number of attempts will be maxRetries + 1 (initial attempt).
//
// Example:
//
//	client := NewHTTPClient(WithMaxRetries(5))
func WithMaxRetries(n int) ClientOption {
	return func(c *HTTPClient) {
		c.maxRetries = n
	}
}

// WithRetryDelay sets the base delay for exponential backoff.
// Actual delay for retry N is: retryDelay * 2^(N-1)
//
// Example:
//
//	client := NewHTTPClient(WithRetryDelay(2 * time.Second))
//	// First retry: 2s, second: 4s, third: 8s, etc.
func WithRetryDelay(d time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.retryDelay = d
	}
}

// WithTransport sets a custom HTTP transport for the client.
// This allows configuring connection pooling, TLS settings, proxies, etc.
//
// Example:
//
//	transport := &http.Transport{
//	    MaxIdleConns:    100,
//	    IdleConnTimeout: 90 * time.Second,
//	}
//	client := NewHTTPClient(WithTransport(transport))
func WithTransport(transport *http.Transport) ClientOption {
	return func(c *HTTPClient) {
		c.client.Transport = transport
	}
}
