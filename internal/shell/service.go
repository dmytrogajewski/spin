package shell

import (
	"context"
	"log/slog"
	"time"
)

// Service wraps shell.Context to provide a clean service interface
// following the dependency injection pattern used in the tools package.
type Service struct {
	context *Context
}

// NewService creates a new Shell service and initializes it.
// If enabled is false, the service is created but not initialized.
func NewService(enabled bool, workDir string, logger *slog.Logger, timeout time.Duration) (*Service, error) {
	ctx := NewContext(enabled, workDir, logger, timeout)

	if enabled {
		if err := ctx.Initialize(context.Background()); err != nil {
			return nil, err
		}
	}

	return &Service{
		context: ctx,
	}, nil
}

// GetContext returns the underlying shell.Context for use with tools.
func (s *Service) GetContext() *Context {
	return s.context
}

// GetContextInfo returns shell context information for the agent.
func (s *Service) GetContextInfo() ContextInfo {
	return s.context.GetContextInfo()
}

// IsEnabled returns true if shell integration is enabled.
func (s *Service) IsEnabled() bool {
	return s.context.IsEnabled()
}

// Close cleans up shell service resources.
func (s *Service) Close() error {
	return s.context.Close()
}
