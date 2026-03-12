package mcp

import (
	"errors"
	"fmt"
	"net/url"
)

// TransportType defines the MCP server connection transport.
type TransportType string

// Transport type constants.
const (
	// TransportStdio uses stdio for local process communication.
	TransportStdio TransportType = "stdio"

	// TransportSSE uses Server-Sent Events for remote communication.
	TransportSSE TransportType = "sse"

	// TransportStreamableHTTP uses HTTP streaming for remote communication.
	TransportStreamableHTTP TransportType = "streamable-http"

	// TransportSmithery uses Smithery's connection-based API.
	TransportSmithery TransportType = "smithery"
)

// IsValid returns true if the transport type is valid.
// Empty string is valid and defaults to stdio.
func (t TransportType) IsValid() bool {
	switch t {
	case "", TransportStdio, TransportSSE, TransportStreamableHTTP, TransportSmithery:
		return true
	default:
		return false
	}
}

// IsRemote returns true if the transport requires a remote URL.
func (t TransportType) IsRemote() bool {
	switch t {
	case TransportSSE, TransportStreamableHTTP, TransportSmithery:
		return true
	default:
		return false
	}
}

// Validate validates the MCP server configuration.
func (c *MCPServerConfig) Validate() error {
	// Name is always required.
	if c.Name == "" {
		return errors.New("name is required")
	}

	// Validate transport type.
	if !c.Transport.IsValid() {
		return fmt.Errorf("invalid transport: %s", c.Transport)
	}

	// Determine effective transport (empty defaults to stdio).
	transport := c.Transport
	if transport == "" {
		transport = TransportStdio
	}

	// Validate based on transport type.
	if transport.IsRemote() {
		return c.validateRemote(transport)
	}

	return c.validateStdio()
}

// validateStdio validates stdio transport configuration.
func (c *MCPServerConfig) validateStdio() error {
	// Command is required for stdio.
	if c.Command == "" {
		return errors.New("command is required for stdio transport")
	}

	// URL is not allowed for stdio.
	if c.URL != "" {
		return errors.New("url is not allowed for stdio transport")
	}

	// OAuth is not allowed for stdio.
	if c.OAuth != nil {
		return errors.New("oauth is not allowed for stdio transport")
	}

	return nil
}

// validateRemote validates remote transport configuration.
func (c *MCPServerConfig) validateRemote(transport TransportType) error {
	// URL is required for remote transports.
	if c.URL == "" {
		return fmt.Errorf("url is required for %s transport", transport)
	}

	// Validate URL format.
	parsedURL, err := url.Parse(c.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid url: %s", c.URL)
	}

	// Command is not allowed for remote transports.
	if c.Command != "" {
		return errors.New("command is not allowed for remote transport")
	}

	// Validate OAuth if provided.
	if c.OAuth != nil {
		if c.OAuth.ClientID == "" {
			return errors.New("oauth client_id is required")
		}
	}

	return nil
}
