package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/client"
)

// LocalRegistryConfig holds configuration for a local stdio MCP registry.
type LocalRegistryConfig struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
	Logger  *slog.Logger
}

// LocalRegistry wraps a local stdio MCP server as an Registry.
type LocalRegistry struct {
	baseRegistry

	config    LocalRegistryConfig
	sdkClient *client.Client
	logger    *slog.Logger
}

// NewLocalRegistry creates a new LocalRegistry for stdio MCP servers.
func NewLocalRegistry(config LocalRegistryConfig) (*LocalRegistry, error) {
	if config.Name == "" {
		return nil, ErrRegistryNameRequired
	}

	if config.Command == "" {
		return nil, ErrCommandRequiredForLocalRegistry
	}

	return &LocalRegistry{
		baseRegistry: baseRegistry{
			name:  config.Name,
			tools: make(map[string]*Tool),
			metadata: RegistryMetadata{
				Name: config.Name,
				Type: "local",
			},
		},
		config: config,
		logger: config.Logger,
	}, nil
}

// Initialize connects to the MCP server and discovers tools.
func (r *LocalRegistry) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connected {
		return nil
	}

	// Build environment slice.
	var env []string
	for k, v := range r.config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create stdio client.
	sdkClient, err := client.NewStdioMCPClient(r.config.Command, env, r.config.Args...)
	if err != nil {
		return fmt.Errorf("create stdio client: %w", err)
	}

	r.sdkClient = sdkClient
	r.mcpClient = &sdkClientWrapper{client: sdkClient}

	// Perform MCP handshake and discover tools.
	meta, toolsMap, err := initializeMCPConnection(ctx, r.mcpClient, r.name)
	if err != nil {
		r.mcpClient.Close()

		return err
	}

	r.applyHandshakeResult(meta, toolsMap)

	if r.logger != nil {
		r.logger.InfoContext(ctx, "local registry initialized",
			"name", r.name,
			"tools", len(r.tools))
	}

	return nil
}
