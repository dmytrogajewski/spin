package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dmytrogajewski/spin/internal/syncmap"
	"github.com/dmytrogajewski/spin/internal/tools"
)

var (
	// ErrRegistryAlreadyRegistered is a sentinel error.
	ErrRegistryAlreadyRegistered = errors.New("registry already registered")
	// ErrRegistryNotFound is a sentinel error.
	ErrRegistryNotFound = errors.New("registry not found")
)

// DefaultRegistryManager is the standard RegistryManager implementation.
type DefaultRegistryManager struct {
	registries *syncmap.Map[string, Registry]
	logger     *slog.Logger
}

// NewDefaultRegistryManager creates a new DefaultRegistryManager.
func NewDefaultRegistryManager(logger *slog.Logger) *DefaultRegistryManager {
	return &DefaultRegistryManager{
		registries: syncmap.New[string, Registry](),
		logger:     logger,
	}
}

// Register adds a registry to the manager.
func (m *DefaultRegistryManager) Register(registry Registry) error {
	name := registry.Name()

	if !m.registries.SetIfAbsent(name, registry) {
		return fmt.Errorf("registry already registered: %s: %w", name, ErrRegistryAlreadyRegistered)
	}

	if m.logger != nil {
		m.logger.Debug("registry registered", "name", name)
	}

	return nil
}

// Unregister removes a registry by name.
func (m *DefaultRegistryManager) Unregister(name string) error {
	registry, ok := m.registries.Pop(name)
	if !ok {
		return fmt.Errorf("registry not found: %s: %w", name, ErrRegistryNotFound)
	}

	// Close the registry after atomic removal from the map.
	err := registry.Close()
	if err != nil && m.logger != nil {
		m.logger.Warn("error closing registry", "name", name, "err", err)
	}

	if m.logger != nil {
		m.logger.Debug("registry unregistered", "name", name)
	}

	return nil
}

// Get retrieves a registry by name.
func (m *DefaultRegistryManager) Get(name string) (Registry, bool) {
	return m.registries.Get(name)
}

// All returns all registered registries.
func (m *DefaultRegistryManager) All() []Registry {
	return m.registries.Values()
}

// AllTools returns tools from all registries.
func (m *DefaultRegistryManager) AllTools() []tools.Tool {
	var result []tools.Tool

	m.registries.Range(func(_ string, registry Registry) bool {
		result = append(result, registry.List()...)

		return true
	})

	return result
}

// Search searches across all registries.
// ctx is for cancellation and timeouts; searchCtx provides additional search options (can be nil).
func (m *DefaultRegistryManager) Search(ctx context.Context, searchCtx *SearchContext, query string, maxResults int) []tools.Tool {
	var allResults []tools.Tool

	m.registries.Range(func(_ string, registry Registry) bool {
		results := registry.Search(ctx, searchCtx, query, maxResults)
		allResults = append(allResults, results...)

		return true
	})

	// Apply max limit if specified.
	if maxResults > 0 && len(allResults) > maxResults {
		allResults = allResults[:maxResults]
	}

	return allResults
}

// Tool finds a tool by name.
// Supports qualified names (registry:tool) for explicit registry targeting.
func (m *DefaultRegistryManager) Tool(name string) tools.Tool {
	// Check for qualified name (registry:tool).
	if idx := strings.Index(name, ":"); idx > 0 {
		registryName := name[:idx]
		toolName := name[idx+1:]

		registry, exists := m.registries.Get(registryName)
		if !exists {
			return nil
		}

		return registry.Tool(toolName)
	}

	// Try to find in all registries by full qualified name (mcp_registry_tool).
	var found tools.Tool

	m.registries.Range(func(_ string, registry Registry) bool {
		for _, t := range registry.List() {
			if t.Name() == name {
				found = t

				return false
			}
		}

		return true
	})

	if found != nil {
		return found
	}

	// Try to find by raw tool name in any registry.
	m.registries.Range(func(_ string, registry Registry) bool {
		if t := registry.Tool(name); t != nil {
			found = t

			return false
		}

		return true
	})

	return found
}

// Close closes all registries.
func (m *DefaultRegistryManager) Close() error {
	var lastErr error

	m.registries.Range(func(name string, registry Registry) bool {
		err := registry.Close()
		if err != nil {
			if m.logger != nil {
				m.logger.Warn("error closing registry", "name", name, "err", err)
			}

			lastErr = err
		}

		return true
	})

	m.registries.Clear()

	if m.logger != nil {
		m.logger.Info("registry manager closed")
	}

	return lastErr
}

// Count returns the total number of tools across all registries.
func (m *DefaultRegistryManager) Count() int {
	count := 0

	m.registries.Range(func(_ string, registry Registry) bool {
		count += registry.Count()

		return true
	})

	return count
}

// RegistryCount returns the number of registered registries.
func (m *DefaultRegistryManager) RegistryCount() int {
	return m.registries.Len()
}
