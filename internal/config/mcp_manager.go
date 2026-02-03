package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// MCPServer represents an MCP server configuration.
type MCPServer struct {
	// Common fields
	Name      string           `yaml:"name" toml:"name" json:"name" mapstructure:"name"`
	Transport MCPTransportType `yaml:"transport,omitempty" toml:"transport,omitempty" json:"transport,omitempty" mapstructure:"transport"`

	// Stdio transport fields
	Command string            `yaml:"command,omitempty" toml:"command,omitempty" json:"command,omitempty" mapstructure:"command"`
	Args    []string          `yaml:"args,omitempty" toml:"args,omitempty" json:"args,omitempty" mapstructure:"args"`
	Env     map[string]string `yaml:"env,omitempty" toml:"env,omitempty" json:"env,omitempty" mapstructure:"env"`

	// Remote transport fields
	URL     string            `yaml:"url,omitempty" toml:"url,omitempty" json:"url,omitempty" mapstructure:"url"`
	Headers map[string]string `yaml:"headers,omitempty" toml:"headers,omitempty" json:"headers,omitempty" mapstructure:"headers"`

	// OAuth configuration
	OAuth *MCPOAuthConfigV2 `yaml:"oauth,omitempty" toml:"oauth,omitempty" json:"oauth,omitempty" mapstructure:"oauth"`

	// Smithery-specific fields
	SmitheryAPIKey    string `yaml:"smithery_api_key,omitempty" toml:"smithery_api_key,omitempty" json:"smithery_api_key,omitempty" mapstructure:"smithery_api_key"`
	SmitheryNamespace string `yaml:"smithery_namespace,omitempty" toml:"smithery_namespace,omitempty" json:"smithery_namespace,omitempty" mapstructure:"smithery_namespace"`

	// DynamicLoadout enables dynamic tool discovery via search
	DynamicLoadout bool `yaml:"dynamic_loadout,omitempty" toml:"dynamic_loadout,omitempty" json:"dynamic_loadout,omitempty" mapstructure:"dynamic_loadout"`
}

// MCPConfigStore manages MCP server configurations.
type MCPConfigStore struct {
	loader     *LoaderV2
	configFile string
}

// NewMCPConfigStore creates a new MCP config store.
func NewMCPConfigStore(loader *LoaderV2) *MCPConfigStore {
	return &MCPConfigStore{
		loader:     loader,
		configFile: loader.ConfigFileUsed(),
	}
}

// List returns all configured MCP servers.
func (m *MCPConfigStore) List() ([]MCPServer, error) {
	var servers []MCPServer
	err := m.loader.UnmarshalKey("protocol.mcp_servers", &servers)
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
func (m *MCPConfigStore) Get(name string) (*MCPServer, error) {
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
func (m *MCPConfigStore) Add(server MCPServer) error {
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
	m.loader.Set("protocol.mcp_servers", servers)

	// Write config
	return m.writeConfig()
}

// Remove removes an MCP server by name.
func (m *MCPConfigStore) Remove(name string) error {
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
	m.loader.Set("protocol.mcp_servers", newServers)

	// Write config
	return m.writeConfig()
}

// validate validates an MCP server configuration.
func (m *MCPConfigStore) validate(server MCPServer) error {
	if server.Name == "" {
		return fmt.Errorf("server name is required")
	}

	// Validate transport type
	if !server.Transport.IsValid() {
		return fmt.Errorf("invalid transport: %s", server.Transport)
	}

	// Determine effective transport (empty defaults to stdio)
	transport := server.Transport
	if transport == "" {
		transport = MCPTransportStdio
	}

	// Validate based on transport type
	if transport == MCPTransportSmithery {
		return m.validateSmithery(server)
	} else if transport.IsRemote() {
		return m.validateRemote(server, transport)
	}
	return m.validateStdio(server)
}

// validateStdio validates stdio transport configuration.
func (m *MCPConfigStore) validateStdio(server MCPServer) error {
	if server.Command == "" {
		return fmt.Errorf("server command is required for stdio transport")
	}
	if server.URL != "" {
		return fmt.Errorf("url is not allowed for stdio transport")
	}
	if server.OAuth != nil {
		return fmt.Errorf("oauth is not allowed for stdio transport")
	}
	return nil
}

// validateRemote validates remote transport configuration.
func (m *MCPConfigStore) validateRemote(server MCPServer, transport MCPTransportType) error {
	if server.URL == "" {
		return fmt.Errorf("url is required for %s transport", transport)
	}
	if server.Command != "" {
		return fmt.Errorf("command is not allowed for remote transport")
	}
	if server.OAuth != nil && server.OAuth.ClientID == "" {
		return fmt.Errorf("oauth client_id is required")
	}
	return nil
}

// validateSmithery validates Smithery transport configuration.
func (m *MCPConfigStore) validateSmithery(server MCPServer) error {
	// API key is always required
	if server.SmitheryAPIKey == "" {
		return fmt.Errorf("smithery_api_key is required for smithery transport")
	}
	// For static mode (URL provided), namespace is also required
	if server.URL != "" && server.SmitheryNamespace == "" {
		return fmt.Errorf("smithery_namespace is required when url is specified")
	}
	// Command is not allowed
	if server.Command != "" {
		return fmt.Errorf("command is not allowed for smithery transport")
	}
	return nil
}

// writeConfig writes the configuration to file.
func (m *MCPConfigStore) writeConfig() error {
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

// GetRegistryTypeName returns a human-readable type name for a registry.
func GetRegistryTypeName(server MCPServer) string {
	switch server.Transport {
	case MCPTransportStdio, "":
		return "local"
	case MCPTransportSmithery:
		return "smithery"
	default:
		return "remote"
	}
}

// GetLoadoutType returns "dynamic" or "static" based on DynamicLoadout setting.
func GetLoadoutType(server MCPServer) string {
	if server.DynamicLoadout {
		return "dynamic"
	}
	return "static"
}

// FormatSource returns the SOURCE column value: registry_name@loadout_type
func FormatSource(server MCPServer) string {
	return server.Name + "@" + GetLoadoutType(server)
}
