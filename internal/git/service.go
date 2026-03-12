package git

import (
	"context"
	"log/slog"
)

// Service wraps Integration to provide a clean service interface
// following the dependency injection pattern used in the tools package.
type Service struct {
	integration *Integration
}

// NewService creates a new Git service and initializes it.
// If enabled is false, the service is created but not initialized.
func NewService(enabled bool, workDir string, logger *slog.Logger) (*Service, error) {
	integration := NewIntegration(enabled, workDir, logger)

	if enabled {
		err := integration.Initialize(context.Background())
		if err != nil {
			return nil, err
		}
	}

	return &Service{
		integration: integration,
	}, nil
}

// GetIntegration returns the underlying Integration for use with tools.
func (s *Service) GetIntegration() *Integration {
	return s.integration
}

// GetContextInfo returns Git context information for the agent.
func (s *Service) GetContextInfo() ContextInfo {
	return s.integration.GetContextInfo()
}

// IsRepository returns true if the working directory is a Git repository.
func (s *Service) IsRepository() bool {
	return s.integration.IsRepository()
}

// Close cleans up Git service resources.
func (s *Service) Close() error {
	return s.integration.Close()
}
