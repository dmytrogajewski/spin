package llm

import (
	"bytes"
	"context"
	"crypto/tls"
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

// NewStreamingHTTPClient creates an HTTP client suitable for long-lived streaming responses.
// - No global client timeout (ctx governs lifetime)
// - No automatic retries (streams should not be retried mid-body)
// - Transport tuned for keep-alive, no compression (avoid proxy buffering), HTTP/1.1 only
func NewStreamingHTTPClient() *HTTPClient {
	transport := &http.Transport{
		// Force HTTP/1.1 only (avoids some H2 buffering behaviors in proxies/middlwares).
		// For HTTPS, disabling HTTP/2:
		TLSNextProto: make(map[string]func(string, *tls.Conn) http.RoundTripper),
		// Reasonable dial timeouts; no read deadline on body (we want long streams).
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// Keep connections warm for repeated calls to same host.
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		// If the server sends compressed NDJSON, some proxies will buffer; disable compression.
		DisableCompression: true,
		// No ResponseHeaderTimeout here; Ollama responds quickly with headers.
		// If you run through a slow proxy, you can set e.g. 15s.
	}

	return &HTTPClient{
		client: &http.Client{
			Timeout:   0, // no global deadline; the request's context controls it
			Transport: transport,
		},
		maxRetries: 0, // do not retry streams
		retryDelay: 0, // unused
	}
}

// Do executes HTTP request with retry logic (for non-streaming requests).
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
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return nil, nil
	}
	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		return nil, nil
	}

	wait := time.Duration(seconds) * time.Second
	resp.Body.Close()

	select {
	case <-time.After(wait):
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
func WithTransport(transport *http.Transport) ClientOption {
	return func(c *HTTPClient) {
		c.client.Transport = transport
	}
}
