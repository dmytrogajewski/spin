package mcp

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// Service provides MCP tool management through a RegistryManager.
type Service struct {
	registryManager RegistryManager
}

// NewService creates a new MCP service with the given RegistryManager.
func NewService(registryManager RegistryManager) *Service {
	return &Service{
		registryManager: registryManager,
	}
}

// GetTools returns all registered MCP tools as tool registry entries.
func (s *Service) GetTools() []tools.Tool {
	if s.registryManager == nil {
		return nil
	}

	return s.registryManager.AllTools()
}

// Search searches for tools matching the query across all registries.
// ctx is for cancellation and timeouts; searchCtx provides additional search options (can be nil).
func (s *Service) Search(ctx context.Context, searchCtx *SearchContext, query string, maxResults int) []tools.Tool {
	if s.registryManager == nil {
		return nil
	}

	return s.registryManager.Search(ctx, searchCtx, query, maxResults)
}

// Tool returns a specific tool by name.
// Supports qualified names (registry:tool) for explicit registry targeting.
func (s *Service) Tool(name string) tools.Tool {
	if s.registryManager == nil {
		return nil
	}

	return s.registryManager.Tool(name)
}

// GetRegistryManager returns the underlying RegistryManager.
func (s *Service) GetRegistryManager() RegistryManager {
	return s.registryManager
}

// ConnectServer connects a new MCP server dynamically.
// This creates a registry for the server and registers it with the manager.
func (s *Service) ConnectServer(ctx context.Context, config ServerConfig) error {
	if s.registryManager == nil {
		return nil
	}

	// Check if already registered.
	if _, exists := s.registryManager.Get(config.Name); exists {
		return nil
	}

	// Create appropriate registry based on transport.
	registry, err := createRegistryFromConfig(config, nil)
	if err != nil {
		return err
	}

	// Register and initialize.
	err = s.registryManager.Register(registry)
	if err != nil {
		return err
	}

	return registry.Initialize(ctx)
}

// Close closes all MCP connections.
func (s *Service) Close() error {
	if s.registryManager == nil {
		return nil
	}

	return s.registryManager.Close()
}

// createRegistryFromConfig creates an Registry from ServerConfig.
func createRegistryFromConfig(config ServerConfig, _ any) (Registry, error) {
	transport := config.Transport
	if transport == "" {
		transport = TransportStdio
	}

	switch transport {
	case TransportStdio:
		return NewLocalRegistry(LocalRegistryConfig{
			Name:    config.Name,
			Command: config.Command,
			Args:    config.Args,
			Env:     config.Env,
		})

	case TransportSSE, TransportStreamableHTTP:
		return NewRemoteRegistry(RemoteRegistryConfig{
			Name:      config.Name,
			Transport: transport,
			URL:       config.URL,
			Headers:   config.Headers,
			OAuth:     config.OAuth,
		})

	case TransportSmithery:
		return NewSmitheryRegistry(SmitheryRegistryConfig{
			Name:      config.Name,
			APIKey:    config.SmitheryAPIKey,
			MCPURL:    config.URL,
			Namespace: config.SmitheryNamespace,
		})

	default:
		return nil, ErrUnsupportedTransport
	}
}
