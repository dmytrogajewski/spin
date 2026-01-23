package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServerConfig_Validate_Stdio(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid stdio config",
			config: MCPServerConfig{
				Name:      "test-server",
				Transport: TransportStdio,
				Command:   "npx",
				Args:      []string{"-y", "@modelcontextprotocol/server-filesystem"},
			},
			wantErr: false,
		},
		{
			name: "valid stdio config with empty transport",
			config: MCPServerConfig{
				Name:    "test-server",
				Command: "echo",
			},
			wantErr: false,
		},
		{
			name: "invalid stdio config missing command",
			config: MCPServerConfig{
				Name:      "test-server",
				Transport: TransportStdio,
			},
			wantErr: true,
			errMsg:  "command is required for stdio transport",
		},
		{
			name: "invalid stdio config with URL",
			config: MCPServerConfig{
				Name:      "test-server",
				Transport: TransportStdio,
				Command:   "echo",
				URL:       "https://example.com/mcp",
			},
			wantErr: true,
			errMsg:  "url is not allowed for stdio transport",
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

func TestMCPServerConfig_Validate_SSE(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid sse config",
			config: MCPServerConfig{
				Name:      "smithery-server",
				Transport: TransportSSE,
				URL:       "https://server.smithery.ai/sse",
			},
			wantErr: false,
		},
		{
			name: "valid sse config with headers",
			config: MCPServerConfig{
				Name:      "smithery-server",
				Transport: TransportSSE,
				URL:       "https://server.smithery.ai/sse",
				Headers: map[string]string{
					"Authorization": "Bearer token",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid sse config missing url",
			config: MCPServerConfig{
				Name:      "smithery-server",
				Transport: TransportSSE,
			},
			wantErr: true,
			errMsg:  "url is required for sse transport",
		},
		{
			name: "invalid sse config with command",
			config: MCPServerConfig{
				Name:      "smithery-server",
				Transport: TransportSSE,
				URL:       "https://server.smithery.ai/sse",
				Command:   "echo",
			},
			wantErr: true,
			errMsg:  "command is not allowed for remote transport",
		},
		{
			name: "invalid sse config with invalid url",
			config: MCPServerConfig{
				Name:      "smithery-server",
				Transport: TransportSSE,
				URL:       "not-a-valid-url",
			},
			wantErr: true,
			errMsg:  "invalid url",
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

func TestMCPServerConfig_Validate_StreamableHTTP(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid streamable-http config",
			config: MCPServerConfig{
				Name:      "remote-server",
				Transport: TransportStreamableHTTP,
				URL:       "https://mcp.example.com/v1",
			},
			wantErr: false,
		},
		{
			name: "invalid streamable-http missing url",
			config: MCPServerConfig{
				Name:      "remote-server",
				Transport: TransportStreamableHTTP,
			},
			wantErr: true,
			errMsg:  "url is required for streamable-http transport",
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

func TestMCPServerConfig_Validate_OAuth(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid sse config with oauth",
			config: MCPServerConfig{
				Name:      "protected-server",
				Transport: TransportSSE,
				URL:       "https://protected.example.com/mcp",
				OAuth: &OAuthConfig{
					ClientID: "my-client-id",
				},
			},
			wantErr: false,
		},
		{
			name: "valid streamable-http config with oauth",
			config: MCPServerConfig{
				Name:      "protected-server",
				Transport: TransportStreamableHTTP,
				URL:       "https://protected.example.com/mcp",
				OAuth: &OAuthConfig{
					ClientID:     "my-client-id",
					ClientSecret: "secret",
					Scopes:       []string{"read", "write"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid oauth with stdio transport",
			config: MCPServerConfig{
				Name:      "local-server",
				Transport: TransportStdio,
				Command:   "echo",
				OAuth: &OAuthConfig{
					ClientID: "my-client-id",
				},
			},
			wantErr: true,
			errMsg:  "oauth is not allowed for stdio transport",
		},
		{
			name: "invalid oauth missing client_id",
			config: MCPServerConfig{
				Name:      "protected-server",
				Transport: TransportSSE,
				URL:       "https://protected.example.com/mcp",
				OAuth:     &OAuthConfig{},
			},
			wantErr: true,
			errMsg:  "oauth client_id is required",
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

func TestMCPServerConfig_Validate_InvalidTransport(t *testing.T) {
	config := MCPServerConfig{
		Name:      "test-server",
		Transport: "websocket",
	}

	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transport")
}

func TestMCPServerConfig_Validate_MissingName(t *testing.T) {
	config := MCPServerConfig{
		Command: "echo",
	}

	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}
