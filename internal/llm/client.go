package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

// HTTPClient wraps http.Client with retry logic and exponential backoff for regular (non-streaming) calls.
type HTTPClient struct {
	client     *http.Client
	maxRetries int
	retryDelay time.Duration
}

// NewHTTPClient creates an HTTP client with retry logic.
// Defaults: Timeout=5m, MaxRetries=3, RetryDelay=1s
func NewHTTPClient(opts ...ClientOption) *HTTPClient {
	// Create custom transport with no response header timeout for streaming
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// No ResponseHeaderTimeout - this is critical for streaming with slow models
	}

	c := &HTTPClient{
		client: &http.Client{
			Timeout:   5 * time.Minute,
			Transport: transport,
		},
		maxRetries: 3,
		retryDelay: time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Do executes HTTP request with retry logic (for non-streaming requests).
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	bodyBytes, err := c.bufferRequestBody(req)
	if err != nil {
		return nil, err
	}

	return c.executeWithRetries(req, bodyBytes)
}

// executeWithRetries executes the request with retry logic.
func (c *HTTPClient) executeWithRetries(req *http.Request, bodyBytes []byte) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := c.waitForRetry(req.Context(), attempt); err != nil {
			return nil, err
		}

		resp, err := c.executeSingleAttempt(req, bodyBytes)
		if err != nil {
			lastErr = err
			continue
		}

		if !c.isRetryable(resp.StatusCode) {
			return resp, nil
		}

		result, err := c.handleRetryableResponse(req, resp, bodyBytes)
		if err != nil {
			lastErr = err
			continue
		}
		if result != nil {
			return result, nil
		}

		resp.Body.Close()
		lastErr = fmt.Errorf("http %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// handleRetryableResponse handles retryable responses and returns a successful response or error.
func (c *HTTPClient) handleRetryableResponse(req *http.Request, resp *http.Response, bodyBytes []byte) (*http.Response, error) {
	newResp, err := c.handleRetryAfter(req, resp, bodyBytes)
	if err != nil {
		return nil, err
	}
	return newResp, nil
}

// executeSingleAttempt executes a single HTTP request attempt.
func (c *HTTPClient) executeSingleAttempt(req *http.Request, bodyBytes []byte) (*http.Response, error) {
	c.restoreRequestBody(req, bodyBytes)
	return c.client.Do(req)
}

// DoStream executes an HTTP request intended for a long-lived streaming response.
// - Single attempt, no retries
// - No request body buffering
// - The provided context controls lifetime
func (c *HTTPClient) DoStream(req *http.Request) (*http.Response, error) {
	// Ensure the underlying client has no global timeout for streams.
	// If a timeout is set on c.client, users should build with NewStreamingHTTPClient.
	return c.client.Do(req)
}

// --- internals for Do (non-stream) ---

// bufferRequestBody reads and buffers request body for retries (non-streaming only).
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

func (c *HTTPClient) calculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	shift := attempt - 1
	if shift > 30 { // prevent overflow
		shift = 30
	}
	return c.retryDelay * time.Duration(1<<uint(shift))
}

func (c *HTTPClient) restoreRequestBody(req *http.Request, bodyBytes []byte) {
	if bodyBytes != nil {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
}

func (c *HTTPClient) handleRetryAfter(req *http.Request, resp *http.Response, bodyBytes []byte) (*http.Response, error) {
	if resp.StatusCode != http.StatusTooManyRequests {
		return nil, nil
	}

	waitDuration := c.parseRetryAfter(resp.Header.Get("Retry-After"))
	if waitDuration == 0 {
		return nil, nil
	}

	resp.Body.Close()

	if err := c.waitForDuration(req.Context(), waitDuration); err != nil {
		return nil, err
	}

	return c.retryRequest(req, bodyBytes)
}

// parseRetryAfter parses the Retry-After header value.
func (c *HTTPClient) parseRetryAfter(retryAfter string) time.Duration {
	if retryAfter == "" {
		return 0
	}
	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// waitForDuration waits for the specified duration or context cancellation.
func (c *HTTPClient) waitForDuration(ctx context.Context, wait time.Duration) error {
	select {
	case <-time.After(wait):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// retryRequest retries the request after waiting.
func (c *HTTPClient) retryRequest(req *http.Request, bodyBytes []byte) (*http.Response, error) {
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

func (c *HTTPClient) isRetryable(code int) bool {
	return code == http.StatusTooManyRequests ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout
}

// --- options ---

type ClientOption func(*HTTPClient)

func WithTimeout(d time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.client.Timeout = d
	}
}
func WithMaxRetries(n int) ClientOption {
	return func(c *HTTPClient) {
		c.maxRetries = n
	}
}
func WithRetryDelay(d time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.retryDelay = d
	}
}
