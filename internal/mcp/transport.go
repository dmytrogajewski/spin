package mcp

import (
	"errors"
	"fmt"
	"net/url"
)

var (
	// ErrNameIsRequired is a sentinel error.
	ErrNameIsRequired = errors.New("name is required")
	// ErrInvalidTransport is a sentinel error.
	ErrInvalidTransport = errors.New("invalid transport")
	// ErrCommandIsRequiredForStdioTransport is a sentinel error.
	ErrCommandIsRequiredForStdioTransport = errors.New("command is required for stdio transport")
	// ErrURLIsNotAllowedForStdio is a sentinel error.
	ErrURLIsNotAllowedForStdio = errors.New("url is not allowed for stdio transport")
	// ErrOauthIsNotAllowedForStdio is a sentinel error.
	ErrOauthIsNotAllowedForStdio = errors.New("oauth is not allowed for stdio transport")
	// ErrURLIsRequiredForTransport is a sentinel error.
	ErrURLIsRequiredForTransport = errors.New("url is required for  transport")
	// ErrInvalidURL is a sentinel error.
	ErrInvalidURL = errors.New("invalid url")
	// ErrCommandIsNotAllowedForRemote is a sentinel error.
	ErrCommandIsNotAllowedForRemote = errors.New("command is not allowed for remote transport")
	// ErrOauthClientIDIsRequired is a sentinel error.
	ErrOauthClientIDIsRequired = errors.New("oauth client_id is required")
	// ErrTransportRequired is returned when a plugin mcp.json server omits type.
	ErrTransportRequired = errors.New("mcp transport type is required")
)

// ParsePluginTransport maps an Agent Plugins mcp.json type to a TransportType.
// Empty type is rejected; transport is never guessed.
func ParsePluginTransport(value string) (TransportType, error) {
	switch value {
	case string(TransportStdio):
		return TransportStdio, nil
	case string(TransportStreamableHTTP):
		return TransportStreamableHTTP, nil
	case string(TransportSSE):
		return TransportSSE, nil
	case "":
		return "", ErrTransportRequired
	default:
		return "", fmt.Errorf("invalid transport: %s: %w", value, ErrUnsupportedTransport)
	}
}

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
	case TransportStdio, "":
		return false
	default:
		return false
	}
}

// Validate validates the MCP server configuration.
func (c *ServerConfig) Validate() error {
	// Name is always required.
	if c.Name == "" {
		return ErrNameIsRequired
	}

	// Validate transport type.
	if !c.Transport.IsValid() {
		return fmt.Errorf("invalid transport: %s: %w", c.Transport, ErrInvalidTransport)
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
func (c *ServerConfig) validateStdio() error {
	// Command is required for stdio.
	if c.Command == "" {
		return ErrCommandIsRequiredForStdioTransport
	}

	// URL is not allowed for stdio.
	if c.URL != "" {
		return ErrURLIsNotAllowedForStdio
	}

	// OAuth is not allowed for stdio.
	if c.OAuth != nil {
		return ErrOauthIsNotAllowedForStdio
	}

	return nil
}

// validateRemote validates remote transport configuration.
func (c *ServerConfig) validateRemote(transport TransportType) error {
	// URL is required for remote transports.
	if c.URL == "" {
		return fmt.Errorf("url is required for %s transport: %w", transport, ErrURLIsRequiredForTransport)
	}

	// Validate URL format.
	parsedURL, err := url.Parse(c.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid url: %s: %w", c.URL, ErrInvalidURL)
	}

	// Command is not allowed for remote transports.
	if c.Command != "" {
		return ErrCommandIsNotAllowedForRemote
	}

	// Validate OAuth if provided.
	if c.OAuth != nil {
		if c.OAuth.ClientID == "" {
			return ErrOauthClientIDIsRequired
		}
	}

	return nil
}
