package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

// RemoteRegistryConfig holds configuration for a remote MCP registry.
type RemoteRegistryConfig struct {
	Name      string
	Transport TransportType // sse or streamable-http.
	URL       string
	Headers   map[string]string
	OAuth     *OAuthConfig
	Logger    *slog.Logger
}

// RemoteRegistry wraps a remote MCP server (SSE or HTTP) as an Registry.
type RemoteRegistry struct {
	baseRegistry

	config    RemoteRegistryConfig
	sdkClient *client.Client
	logger    *slog.Logger
}

// NewRemoteRegistry creates a new RemoteRegistry for SSE or HTTP MCP servers.
func NewRemoteRegistry(config RemoteRegistryConfig) (*RemoteRegistry, error) {
	if config.Name == "" {
		return nil, ErrRegistryNameRequired
	}

	if config.URL == "" {
		return nil, ErrURLRequiredForRemoteRegistry
	}

	if config.Transport != TransportSSE && config.Transport != TransportStreamableHTTP {
		return nil, ErrTransportMustBeSseOrStreamable
	}

	return &RemoteRegistry{
		baseRegistry: baseRegistry{
			name:  config.Name,
			tools: make(map[string]*Tool),
			metadata: RegistryMetadata{
				Name: config.Name,
				Type: "remote",
			},
		},
		config: config,
		logger: config.Logger,
	}, nil
}

// Initialize connects to the MCP server and discovers tools.
func (r *RemoteRegistry) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connected {
		return nil
	}

	// Create and start the transport client.
	if err := r.connectTransport(ctx); err != nil {
		return err
	}

	// Perform MCP handshake and discover tools.
	meta, toolsMap, err := initializeMCPConnection(ctx, r.mcpClient, r.name)
	if err != nil {
		r.mcpClient.Close()

		return err
	}

	r.applyHandshakeResult(meta, toolsMap)

	if r.logger != nil {
		r.logger.InfoContext(ctx, "remote registry initialized",
			"name", r.name,
			"transport", r.config.Transport,
			"tools", len(r.tools))
	}

	return nil
}

// connectTransport creates the SDK client, starts its transport, and stores it.
func (r *RemoteRegistry) connectTransport(ctx context.Context) error {
	var (
		sdkClient *client.Client
		err       error
	)

	switch r.config.Transport {
	case TransportSSE:
		sdkClient, err = r.createSSEClient()
	case TransportStreamableHTTP:
		sdkClient, err = r.createStreamableHTTPClient()
	case TransportStdio, TransportSmithery, "":
		return fmt.Errorf("unsupported transport: %s: %w", r.config.Transport, ErrUnsupportedTransport)
	default:
		return fmt.Errorf("unsupported transport: %s: %w", r.config.Transport, ErrUnsupportedTransport)
	}

	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	if err = sdkClient.Start(ctx); err != nil {
		sdkClient.Close()

		return fmt.Errorf("start transport: %w", err)
	}

	r.sdkClient = sdkClient
	r.mcpClient = &sdkClientWrapper{client: sdkClient}

	return nil
}

// buildOAuthConfig extracts the OAuth configuration from the registry config.
func (r *RemoteRegistry) buildOAuthConfig() transport.OAuthConfig {
	return transport.OAuthConfig{
		ClientID:     r.config.OAuth.ClientID,
		ClientSecret: r.config.OAuth.ClientSecret,
		RedirectURI:  r.config.OAuth.RedirectURL,
		Scopes:       r.config.OAuth.Scopes,
	}
}

// createMCPClient creates an MCP client using the provided factory functions.
// This eliminates duplication between SSE and StreamableHTTP client creation paths.
func (r *RemoteRegistry) createMCPClient(
	newPlainClient func(url string) (*client.Client, error),
	newOAuthClient func(url string, oauth transport.OAuthConfig) (*client.Client, error),
	label string,
) (*client.Client, error) {
	if r.config.OAuth != nil {
		c, err := newOAuthClient(r.config.URL, r.buildOAuthConfig())
		if err != nil {
			return nil, fmt.Errorf("create OAuth %s client: %w", label, err)
		}

		return c, nil
	}

	c, err := newPlainClient(r.config.URL)
	if err != nil {
		return nil, fmt.Errorf("create %s client: %w", label, err)
	}

	return c, nil
}

// sseClientFactories returns plain and OAuth factory functions for SSE transport.
func sseClientFactories(headers map[string]string) (
	plainFactory func(string) (*client.Client, error),
	oauthFactory func(string, transport.OAuthConfig) (*client.Client, error),
) {
	var opts []transport.ClientOption
	if len(headers) > 0 {
		opts = append(opts, transport.WithHeaders(headers))
	}

	return func(url string) (*client.Client, error) {
			return client.NewSSEMCPClient(url, opts...)
		}, func(url string, oauth transport.OAuthConfig) (*client.Client, error) {
			return client.NewOAuthSSEClient(url, oauth, opts...)
		}
}

// createSSEClient creates an SSE MCP client.
func (r *RemoteRegistry) createSSEClient() (*client.Client, error) {
	plain, oauth := sseClientFactories(r.config.Headers)

	return r.createMCPClient(plain, oauth, "SSE")
}

// createStreamableHTTPClient creates a streamable HTTP MCP client.
func (r *RemoteRegistry) createStreamableHTTPClient() (*client.Client, error) {
	var opts []transport.StreamableHTTPCOption
	if len(r.config.Headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(r.config.Headers))
	}

	return r.createMCPClient(
		func(url string) (*client.Client, error) {
			return client.NewStreamableHttpClient(url, opts...)
		},
		func(url string, oauth transport.OAuthConfig) (*client.Client, error) {
			return client.NewOAuthStreamableHttpClient(url, oauth, opts...)
		},
		"streamable HTTP",
	)
}
