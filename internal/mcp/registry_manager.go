package mcp

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/dmytrogajewski/spin/internal/tools"
)

var (
	ErrRegistryAlreadyRegistered = errors.New("registry already registered")
	ErrRegistryNotFound = errors.New("registry not found")
)

// DefaultRegistryManager is the standard RegistryManager implementation.
type DefaultRegistryManager struct {
	registries map[string]Registry
	logger     *slog.Logger
	mu         sync.RWMutex
}

// NewDefaultRegistryManager creates a new DefaultRegistryManager.
func NewDefaultRegistryManager(logger *slog.Logger) *DefaultRegistryManager {
	return &DefaultRegistryManager{
		registries: make(map[string]Registry),
		logger:     logger,
	}
}

// Register adds a registry to the manager.
func (m *DefaultRegistryManager) Register(registry Registry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := registry.Name()
	if _, exists := m.registries[name]; exists {
return fmt.Errorf("registry already registered: %s: %w", name, ErrRegistryAlreadyRegistered)
	}

	m.registries[name] = registry

	if m.logger != nil {
		m.logger.Debug("registry registered", "name", name)
	}

	return nil
}

// Unregister removes a registry by name.
func (m *DefaultRegistryManager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	registry, exists := m.registries[name]
	if !exists {
return fmt.Errorf("registry not found: %s: %w", name, ErrRegistryNotFound)
	}

	// Close the registry.
	err := registry.Close()
	if err != nil {
		if m.logger != nil {
			m.logger.Warn("error closing registry", "name", name, "err", err)
		}
	}

	delete(m.registries, name)

	if m.logger != nil {
		m.logger.Debug("registry unregistered", "name", name)
	}

	return nil
}

// Get retrieves a registry by name.
func (m *DefaultRegistryManager) Get(name string) (Registry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	registry, exists := m.registries[name]

	return registry, exists
}

// All returns all registered registries.
func (m *DefaultRegistryManager) All() []Registry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Registry, 0, len(m.registries))
	for _, registry := range m.registries {
		result = append(result, registry)
	}

	return result
}

// AllTools returns tools from all registries.
func (m *DefaultRegistryManager) AllTools() []tools.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []tools.Tool
	for _, registry := range m.registries {
		result = append(result, registry.List()...)
	}

	return result
}

// Search searches across all registries.
// ctx can be nil for simple searches; required for dynamic registries that call APIs.
func (m *DefaultRegistryManager) Search(ctx *SearchContext, query string, maxResults int) []tools.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allResults []tools.Tool

	// Search each registry.
	for _, registry := range m.registries {
		results := registry.Search(ctx, query, maxResults)
		allResults = append(allResults, results...)
	}

	// Apply max limit if specified.
	if maxResults > 0 && len(allResults) > maxResults {
		allResults = allResults[:maxResults]
	}

	return allResults
}

// Tool finds a tool by name.
// Supports qualified names (registry:tool) for explicit registry targeting.
func (m *DefaultRegistryManager) Tool(name string) tools.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for qualified name (registry:tool).
	if idx := strings.Index(name, ":"); idx > 0 {
		registryName := name[:idx]
		toolName := name[idx+1:]

		registry, exists := m.registries[registryName]
		if !exists {
			return nil
		}

		return registry.Tool(toolName)
	}

	// Try to find in all registries by full qualified name (mcp_registry_tool).
	for _, registry := range m.registries {
		for _, t := range registry.List() {
			if t.Name() == name {
				return t
			}
		}
	}

	// Try to find by raw tool name in any registry.
	for _, registry := range m.registries {
		if t := registry.Tool(name); t != nil {
			return t
		}
	}

	return nil
}

// Close closes all registries.
func (m *DefaultRegistryManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error

	for name, registry := range m.registries {
		err := registry.Close()
		if err != nil {
			if m.logger != nil {
				m.logger.Warn("error closing registry", "name", name, "err", err)
			}

			lastErr = err
		}
	}

	m.registries = make(map[string]Registry)

	if m.logger != nil {
		m.logger.Info("registry manager closed")
	}

	return lastErr
}

// Count returns the total number of tools across all registries.
func (m *DefaultRegistryManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, registry := range m.registries {
		count += registry.Count()
	}

	return count
}

// RegistryCount returns the number of registered registries.
func (m *DefaultRegistryManager) RegistryCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.registries)
}
