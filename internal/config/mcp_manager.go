package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// MCPServer represents an MCP server configuration.
type MCPServer struct {
	Name    string            `yaml:"name" toml:"name" json:"name" mapstructure:"name"`
	Command string            `yaml:"command" toml:"command" json:"command" mapstructure:"command"`
	Args    []string          `yaml:"args,omitempty" toml:"args,omitempty" json:"args,omitempty" mapstructure:"args"`
	Env     map[string]string `yaml:"env,omitempty" toml:"env,omitempty" json:"env,omitempty" mapstructure:"env"`
}

// MCPManager manages MCP server configurations.
type MCPManager struct {
	loader     *LoaderV2
	configFile string
}

// NewMCPManager creates a new MCP manager.
func NewMCPManager(loader *LoaderV2) *MCPManager {
	return &MCPManager{
		loader:     loader,
		configFile: loader.ConfigFileUsed(),
	}
}

// List returns all configured MCP servers.
func (m *MCPManager) List() ([]MCPServer, error) {
	var servers []MCPServer
	err := m.loader.UnmarshalKey("mcp.servers", &servers)
	if err != nil {
		// If key doesn't exist, return empty list
		return []MCPServer{}, nil
	}
	if servers == nil {
		return []MCPServer{}, nil
	}
	return servers, nil
}

// Get returns a specific MCP server by name.
func (m *MCPManager) Get(name string) (*MCPServer, error) {
	servers, err := m.List()
	if err != nil {
		return nil, err
	}

	for _, server := range servers {
		if server.Name == name {
			return &server, nil
		}
	}

	return nil, fmt.Errorf("mcp server '%s' not found", name)
}

// Add adds a new MCP server configuration.
func (m *MCPManager) Add(server MCPServer) error {
	// Validate
	if err := m.validate(server); err != nil {
		return err
	}

	// Check for duplicates
	existing, _ := m.Get(server.Name)
	if existing != nil {
		return fmt.Errorf("mcp server '%s' already exists", server.Name)
	}

	// Get current servers
	servers, err := m.List()
	if err != nil {
		return err
	}

	// Append new server
	servers = append(servers, server)

	// Update config
	m.loader.Set("mcp.servers", servers)

	// Write config
	return m.writeConfig()
}

// Remove removes an MCP server by name.
func (m *MCPManager) Remove(name string) error {
	// Get current servers
	servers, err := m.List()
	if err != nil {
		return err
	}

	// Find and remove
	found := false
	newServers := make([]MCPServer, 0, len(servers))
	for _, server := range servers {
		if server.Name == name {
			found = true
			continue
		}
		newServers = append(newServers, server)
	}

	if !found {
		return fmt.Errorf("mcp server '%s' not found", name)
	}

	// Update config
	m.loader.Set("mcp.servers", newServers)

	// Write config
	return m.writeConfig()
}

// validate validates an MCP server configuration.
func (m *MCPManager) validate(server MCPServer) error {
	if server.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if server.Command == "" {
		return fmt.Errorf("server command is required")
	}
	return nil
}

// writeConfig writes the configuration to file.
func (m *MCPManager) writeConfig() error {
	// Determine config file path
	configFile := m.configFile
	if configFile == "" {
		// Use default: ~/.spin/spin.yaml
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		spinDir := homeDir + "/.spin"
		configFile = spinDir + "/spin.yaml"
		m.configFile = configFile

		// Create directory if needed
		if err := os.MkdirAll(spinDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Get all settings
	settings := m.loader.AllSettings()

	// Marshal to YAML
	data, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file (atomic)
	tmpFile := configFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Rename (atomic operation)
	if err := os.Rename(tmpFile, configFile); err != nil {
		os.Remove(tmpFile) // Cleanup
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
