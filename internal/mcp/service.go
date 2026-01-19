package mcp

import (
	"context"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// Service wraps MCPServerManager to provide a clean service interface
// following the dependency injection pattern used in the tools package.
type Service struct {
	manager *MCPServerManager
}

// NewService creates a new MCP service and initializes it.
// If config.EnableMCP is false, the service is created but not initialized.
func NewService(config *Config, logger *slog.Logger) (*Service, error) {
	manager := NewMCPServerManager(config, logger)

	if config.EnableMCP {
		if err := manager.Initialize(context.Background()); err != nil {
			return nil, err
		}
	}

	return &Service{
		manager: manager,
	}, nil
}

// GetTools returns all registered MCP tools as tool registry entries.
func (s *Service) GetTools() []tools.Tool {
	return s.manager.GetTools()
}

// Close closes all MCP connections.
func (s *Service) Close() error {
	return s.manager.Close()
}
