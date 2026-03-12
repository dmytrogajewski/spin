//go:build e2e_mcp_test

package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/mcp"
)

// testLogger returns a logger that discards all output.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// mockSSEServer creates a mock SSE MCP server for testing.
func mockSSEServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for expected headers
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Logf("Received Authorization header: %s", auth[:min(len(auth), 20)]+"...")
		}

		// Respond with SSE events
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		// Send initial connection event
		fmt.Fprintf(w, "event: open\ndata: {\"status\":\"connected\"}\n\n")
		flusher.Flush()

		// Keep connection open briefly for test
		time.Sleep(100 * time.Millisecond)
	}))
}

// mockStreamableHTTPServer creates a mock streamable HTTP MCP server.
func mockStreamableHTTPServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for expected headers
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			t.Logf("Received X-API-Key header")
		}

		// Return mock MCP response
		w.Header().Set("Content-Type", "application/json")

		// Mock initialize response
		if strings.Contains(r.URL.Path, "initialize") || r.Method == "POST" {
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]interface{}{},
					"serverInfo": map[string]interface{}{
						"name":    "mock-server",
						"version": "1.0.0",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Default response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]interface{}{},
		})
	}))
}

// TestRegistry_SSE_Creation tests SSE registry creation.
func TestRegistry_SSE_Creation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := mockSSEServer(t)
	defer server.Close()

	registry, err := mcp.NewRemoteRegistry(mcp.RemoteRegistryConfig{
		Name:      "test-sse",
		Transport: mcp.TransportSSE,
		URL:       server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
		},
		Logger: testLogger(),
	})
	require.NoError(t, err)
	require.NotNil(t, registry)
	defer registry.Close()

	assert.Equal(t, "test-sse", registry.Name())
	assert.False(t, registry.IsConnected()) // Not initialized yet
}

// TestRegistry_StreamableHTTP_Creation tests streamable HTTP registry creation.
func TestRegistry_StreamableHTTP_Creation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := mockStreamableHTTPServer(t)
	defer server.Close()

	registry, err := mcp.NewRemoteRegistry(mcp.RemoteRegistryConfig{
		Name:      "test-http",
		Transport: mcp.TransportStreamableHTTP,
		URL:       server.URL,
		Headers: map[string]string{
			"X-API-Key": "test-key",
		},
		Logger: testLogger(),
	})
	require.NoError(t, err)
	require.NotNil(t, registry)
	defer registry.Close()

	assert.Equal(t, "test-http", registry.Name())
}

// TestRegistry_Local_Creation tests local stdio registry creation.
func TestRegistry_Local_Creation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	registry, err := mcp.NewLocalRegistry(mcp.LocalRegistryConfig{
		Name:    "test-local",
		Command: "/bin/echo",
		Args:    []string{"test"},
		Logger:  testLogger(),
	})
	require.NoError(t, err)
	require.NotNil(t, registry)
	defer registry.Close()

	assert.Equal(t, "test-local", registry.Name())
}

// TestRegistryManager_MultipleRegistries tests managing multiple registries.
func TestRegistryManager_MultipleRegistries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	manager := mcp.NewDefaultRegistryManager(testLogger())
	defer manager.Close()

	// Create local registry
	localReg, err := mcp.NewLocalRegistry(mcp.LocalRegistryConfig{
		Name:    "local-server",
		Command: "/bin/echo",
		Args:    []string{"test"},
		Logger:  testLogger(),
	})
	require.NoError(t, err)
	require.NoError(t, manager.Register(localReg))

	// Verify registration
	assert.Equal(t, 1, manager.RegistryCount())

	reg, exists := manager.Get("local-server")
	assert.True(t, exists)
	assert.Equal(t, "local-server", reg.Name())
}

// TestMCPServerConfig_Validation tests configuration validation.
func TestMCPServerConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  mcp.MCPServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid stdio config",
			config: mcp.MCPServerConfig{
				Name:    "test",
				Command: "/bin/echo",
			},
			wantErr: false,
		},
		{
			name: "valid sse config",
			config: mcp.MCPServerConfig{
				Name:      "test",
				Transport: mcp.TransportSSE,
				URL:       "https://example.com/mcp",
			},
			wantErr: false,
		},
		{
			name: "valid streamable-http config",
			config: mcp.MCPServerConfig{
				Name:      "test",
				Transport: mcp.TransportStreamableHTTP,
				URL:       "https://example.com/mcp",
			},
			wantErr: false,
		},
		{
			name: "sse with oauth",
			config: mcp.MCPServerConfig{
				Name:      "test",
				Transport: mcp.TransportSSE,
				URL:       "https://example.com/mcp",
				OAuth: &mcp.OAuthConfig{
					ClientID: "test-client",
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: mcp.MCPServerConfig{
				Command: "/bin/echo",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "stdio missing command",
			config: mcp.MCPServerConfig{
				Name: "test",
			},
			wantErr: true,
			errMsg:  "command is required",
		},
		{
			name: "sse missing url",
			config: mcp.MCPServerConfig{
				Name:      "test",
				Transport: mcp.TransportSSE,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "streamable-http missing url",
			config: mcp.MCPServerConfig{
				Name:      "test",
				Transport: mcp.TransportStreamableHTTP,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "stdio with url",
			config: mcp.MCPServerConfig{
				Name:    "test",
				Command: "/bin/echo",
				URL:     "https://example.com",
			},
			wantErr: true,
			errMsg:  "url is not allowed",
		},
		{
			name: "sse with command",
			config: mcp.MCPServerConfig{
				Name:      "test",
				Transport: mcp.TransportSSE,
				URL:       "https://example.com/mcp",
				Command:   "/bin/echo",
			},
			wantErr: true,
			errMsg:  "command is not allowed",
		},
		{
			name: "stdio with oauth",
			config: mcp.MCPServerConfig{
				Name:    "test",
				Command: "/bin/echo",
				OAuth: &mcp.OAuthConfig{
					ClientID: "test-client",
				},
			},
			wantErr: true,
			errMsg:  "oauth is not allowed",
		},
		{
			name: "oauth missing client_id",
			config: mcp.MCPServerConfig{
				Name:      "test",
				Transport: mcp.TransportSSE,
				URL:       "https://example.com/mcp",
				OAuth:     &mcp.OAuthConfig{},
			},
			wantErr: true,
			errMsg:  "oauth client_id is required",
		},
		{
			name: "invalid transport",
			config: mcp.MCPServerConfig{
				Name:      "test",
				Transport: "websocket",
			},
			wantErr: true,
			errMsg:  "invalid transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMCPTransportType tests transport type methods.
func TestMCPTransportType(t *testing.T) {
	t.Run("IsValid", func(t *testing.T) {
		tests := []struct {
			transport mcp.TransportType
			valid     bool
		}{
			{"", true},
			{mcp.TransportStdio, true},
			{mcp.TransportSSE, true},
			{mcp.TransportStreamableHTTP, true},
			{"websocket", false},
			{"invalid", false},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.valid, tt.transport.IsValid(), "transport: %s", tt.transport)
		}
	})

	t.Run("IsRemote", func(t *testing.T) {
		tests := []struct {
			transport mcp.TransportType
			remote    bool
		}{
			{"", false},
			{mcp.TransportStdio, false},
			{mcp.TransportSSE, true},
			{mcp.TransportStreamableHTTP, true},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.remote, tt.transport.IsRemote(), "transport: %s", tt.transport)
		}
	})
}

// TestService_Creation tests Service creation with registry manager.
func TestService_Creation(t *testing.T) {
	manager := mcp.NewDefaultRegistryManager(testLogger())
	service := mcp.NewService(manager)
	require.NotNil(t, service)
	defer service.Close()

	// Empty manager should return no tools
	tools := service.GetTools()
	assert.Empty(t, tools)
}

// TestService_Search tests tool search functionality.
func TestService_Search(t *testing.T) {
	manager := mcp.NewDefaultRegistryManager(testLogger())
	service := mcp.NewService(manager)
	require.NotNil(t, service)
	defer service.Close()

	// Search on empty registry should return empty
	results := service.Search(nil, "test", 10)
	assert.Empty(t, results)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
