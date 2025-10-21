package llm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ClientOption
		wantErr bool
	}{
		{
			name:    "default configuration",
			opts:    nil,
			wantErr: false,
		},
		{
			name: "with custom timeout",
			opts: []ClientOption{
				WithTimeout(30 * time.Second),
			},
			wantErr: false,
		},
		{
			name: "with custom max retries",
			opts: []ClientOption{
				WithMaxRetries(5),
			},
			wantErr: false,
		},
		{
			name: "with custom retry delay",
			opts: []ClientOption{
				WithRetryDelay(2 * time.Second),
			},
			wantErr: false,
		},
		{
			name: "with multiple options",
			opts: []ClientOption{
				WithTimeout(1 * time.Minute),
				WithMaxRetries(10),
				WithRetryDelay(500 * time.Millisecond),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewHTTPClient(tt.opts...)
			if client == nil {
				t.Fatal("NewHTTPClient() returned nil")
			}
			if client.client == nil {
				t.Error("HTTPClient.client is nil")
			}
		})
	}
}

func TestHTTPClient_Do_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Do() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	if string(body) != "success" {
		t.Errorf("Do() body = %q, want %q", body, "success")
	}
}

func TestHTTPClient_Do_RetryOnRetryableErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCodes    []int
		wantAttempts   int32
		wantFinalError bool
	}{
		{
			name:           "retry on 429 then succeed",
			statusCodes:    []int{http.StatusTooManyRequests, http.StatusOK},
			wantAttempts:   2,
			wantFinalError: false,
		},
		{
			name:           "retry on 503 then succeed",
			statusCodes:    []int{http.StatusServiceUnavailable, http.StatusOK},
			wantAttempts:   2,
			wantFinalError: false,
		},
		{
			name:           "retry on 504 then succeed",
			statusCodes:    []int{http.StatusGatewayTimeout, http.StatusOK},
			wantAttempts:   2,
			wantFinalError: false,
		},
		{
			name:           "max retries exceeded",
			statusCodes:    []int{http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusTooManyRequests},
			wantAttempts:   4, // initial + 3 retries
			wantFinalError: true,
		},
		{
			name:           "no retry on 400",
			statusCodes:    []int{http.StatusBadRequest},
			wantAttempts:   1,
			wantFinalError: false,
		},
		{
			name:           "no retry on 401",
			statusCodes:    []int{http.StatusUnauthorized},
			wantAttempts:   1,
			wantFinalError: false,
		},
		{
			name:           "no retry on 404",
			statusCodes:    []int{http.StatusNotFound},
			wantAttempts:   1,
			wantFinalError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attempts int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				idx := int(atomic.AddInt32(&attempts, 1)) - 1
				if idx < len(tt.statusCodes) {
					w.WriteHeader(tt.statusCodes[idx])
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer server.Close()

			client := NewHTTPClient(
				WithMaxRetries(3),
				WithRetryDelay(10*time.Millisecond),
			)

			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if tt.wantFinalError {
				if err == nil {
					t.Error("Do() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Do() unexpected error = %v", err)
				}
				if resp != nil {
					resp.Body.Close()
				}
			}

			if attempts != tt.wantAttempts {
				t.Errorf("Do() attempts = %d, want %d", attempts, tt.wantAttempts)
			}
		})
	}
}

func TestHTTPClient_Do_RetryAfterHeader(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(
		WithMaxRetries(3),
		WithRetryDelay(10*time.Millisecond),
	)

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if attempts != 2 {
		t.Errorf("Do() attempts = %d, want 2", attempts)
	}

	// Should have waited at least 1 second for Retry-After
	if elapsed < time.Second {
		t.Errorf("Do() elapsed = %v, want >= 1s", elapsed)
	}
}

func TestHTTPClient_Do_ExponentialBackoff(t *testing.T) {
	var attempts int32
	var timestamps []time.Time

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamps = append(timestamps, time.Now())
		if atomic.AddInt32(&attempts, 1) <= 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	baseDelay := 50 * time.Millisecond
	client := NewHTTPClient(
		WithMaxRetries(3),
		WithRetryDelay(baseDelay),
	)

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if attempts != 4 {
		t.Errorf("Do() attempts = %d, want 4", attempts)
	}

	// Verify exponential backoff: delays should be approximately baseDelay * 2^(attempt-1)
	// First retry: ~50ms, second: ~100ms, third: ~200ms
	for i := 1; i < len(timestamps); i++ {
		delay := timestamps[i].Sub(timestamps[i-1])
		expectedDelay := baseDelay * time.Duration(1<<uint(i-1))

		// Allow 100% tolerance for timing variations to reduce flakiness
		minDelay := expectedDelay / 2
		maxDelay := expectedDelay * 3

		if delay < minDelay || delay > maxDelay {
			t.Errorf("retry %d delay = %v, want between %v and %v", i, delay, minDelay, maxDelay)
		}
	}
}

func TestHTTPClient_Do_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = client.Do(req)
	if err == nil {
		t.Error("Do() expected context deadline error, got nil")
	}
}

func TestHTTPClient_isRetryable(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"429 Too Many Requests", http.StatusTooManyRequests, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
		{"504 Gateway Timeout", http.StatusGatewayTimeout, true},
		{"200 OK", http.StatusOK, false},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"401 Unauthorized", http.StatusUnauthorized, false},
		{"403 Forbidden", http.StatusForbidden, false},
		{"404 Not Found", http.StatusNotFound, false},
		{"500 Internal Server Error", http.StatusInternalServerError, false},
		{"502 Bad Gateway", http.StatusBadGateway, false},
	}

	client := NewHTTPClient()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.isRetryable(tt.statusCode)
			if got != tt.want {
				t.Errorf("isRetryable(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestHTTPClient_Do_RequestBodyBuffering(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "test payload" {
			t.Errorf("request body = %q, want %q", body, "test payload")
		}

		if atomic.AddInt32(&attempts, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(
		WithMaxRetries(3),
		WithRetryDelay(10*time.Millisecond),
	)

	req, err := http.NewRequest("POST", server.URL, strings.NewReader("test payload"))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if attempts != 2 {
		t.Errorf("Do() attempts = %d, want 2", attempts)
	}
}


func TestHTTPClient_Do_NetworkError(t *testing.T) {
	client := NewHTTPClient(
		WithMaxRetries(2),
		WithRetryDelay(10*time.Millisecond),
	)

	// Create request to non-existent server using a reserved port that should fail
	req, err := http.NewRequest("GET", "http://127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = client.Do(req)
	if err == nil {
		t.Error("Do() expected network error, got nil")
	}
}

func TestHTTPClient_Do_RequestBodyReadError(t *testing.T) {
	client := NewHTTPClient()

	req, err := http.NewRequest("POST", "http://localhost:8080", io.NopCloser(&errorReader{}))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = client.Do(req)
	if err == nil {
		t.Error("Do() expected body read error, got nil")
	}
}

func TestHTTPClient_Do_NoRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient()
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Do() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHTTPClient_Do_InvalidRetryAfterHeader(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.Header().Set("Retry-After", "invalid")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(
		WithMaxRetries(3),
		WithRetryDelay(10*time.Millisecond),
	)

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if attempts != 2 {
		t.Errorf("Do() attempts = %d, want 2", attempts)
	}
}

func TestHTTPClient_Do_RetryAfterWithRetryableResponse(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(
		WithMaxRetries(3),
		WithRetryDelay(10*time.Millisecond),
	)

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if attempts != 3 {
		t.Errorf("Do() attempts = %d, want 3", attempts)
	}
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}
