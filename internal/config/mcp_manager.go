package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	ErrMcpServerNotFound               = errors.New("mcp server not found")
	ErrMcpServerAlreadyExists          = errors.New("mcp server already exists")
	ErrServerNameIsRequired            = errors.New("server name is required")
	ErrServerCommandRequiredForStdio   = errors.New("server command is required for stdio transport")
)

// MCPServer represents an MCP server configuration.
type MCPServer struct {
	// Common fields.
	Name      string           `json:"name"                mapstructure:"name"      toml:"name"                yaml:"name"`
	Transport MCPTransportType `json:"transport,omitempty" mapstructure:"transport" toml:"transport,omitempty" yaml:"transport,omitempty"`

	// Stdio transport fields.
	Command string            `json:"command,omitempty" mapstructure:"command" toml:"command,omitempty" yaml:"command,omitempty"`
	Args    []string          `json:"args,omitempty"    mapstructure:"args"    toml:"args,omitempty"    yaml:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"     mapstructure:"env"     toml:"env,omitempty"     yaml:"env,omitempty"`

	// Remote transport fields.
	URL     string            `json:"url,omitempty"     mapstructure:"url"     toml:"url,omitempty"     yaml:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty" mapstructure:"headers" toml:"headers,omitempty" yaml:"headers,omitempty"`

	// OAuth configuration.
	OAuth *MCPOAuthConfigV2 `json:"oauth,omitempty" mapstructure:"oauth" toml:"oauth,omitempty" yaml:"oauth,omitempty"`

	// Smithery-specific fields.
	SmitheryAPIKey    string `json:"smithery_api_key,omitempty"   mapstructure:"smithery_api_key"   toml:"smithery_api_key,omitempty"   yaml:"smithery_api_key,omitempty"`
	SmitheryNamespace string `json:"smithery_namespace,omitempty" mapstructure:"smithery_namespace" toml:"smithery_namespace,omitempty" yaml:"smithery_namespace,omitempty"`

	// DynamicLoadout enables dynamic tool discovery via search.
	DynamicLoadout bool `json:"dynamic_loadout,omitempty" mapstructure:"dynamic_loadout" toml:"dynamic_loadout,omitempty" yaml:"dynamic_loadout,omitempty"`
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

	// Missing key is not an error — return empty list on unmarshal failure.
	_ = m.loader.UnmarshalKey("protocol.mcp_servers", &servers)

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

	return nil, fmt.Errorf("mcp server '%s' not found: %w", name, ErrMcpServerNotFound)
}

// Add adds a new MCP server configuration.
func (m *MCPConfigStore) Add(server MCPServer) error {
	// Validate.
	err := m.validate(server)
	if err != nil {
		return err
	}

	// Check for duplicates.
	existing, _ := m.Get(server.Name)
	if existing != nil {
		return fmt.Errorf("mcp server '%s' already exists: %w", server.Name, ErrMcpServerAlreadyExists)
	}

	// Get current servers.
	servers, err := m.List()
	if err != nil {
		return err
	}

	// Append new server.
	servers = append(servers, server)

	// Update config.
	m.loader.Set("protocol.mcp_servers", servers)

	// Write config.
	return m.writeConfig()
}

// Remove removes an MCP server by name.
func (m *MCPConfigStore) Remove(name string) error {
	// Get current servers.
	servers, err := m.List()
	if err != nil {
		return err
	}

	// Find and remove.
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
		return fmt.Errorf("mcp server '%s' not found: %w", name, ErrMcpServerNotFound)
	}

	// Update config.
	m.loader.Set("protocol.mcp_servers", newServers)

	// Write config.
	return m.writeConfig()
}

// validate validates an MCP server configuration.
func (m *MCPConfigStore) validate(server MCPServer) error {
	if server.Name == "" {
		return ErrServerNameIsRequired
	}

	// Validate transport type.
	if !server.Transport.IsValid() {
		return fmt.Errorf("invalid transport: %s: %w", server.Transport, ErrInvalidTransport)
	}

	// Determine effective transport (empty defaults to stdio).
	transport := server.Transport
	if transport == "" {
		transport = MCPTransportStdio
	}

	// Validate based on transport type.
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
		return ErrServerCommandRequiredForStdio
	}

	if server.URL != "" {
		return ErrURLNotAllowedForStdio
	}

	if server.OAuth != nil {
		return ErrOauthNotAllowedForStdio
	}

	return nil
}

// validateRemote validates remote transport configuration.
func (m *MCPConfigStore) validateRemote(server MCPServer, transport MCPTransportType) error {
	if server.URL == "" {
		return fmt.Errorf("url is required for %s transport: %w", transport, ErrURLRequiredForTransport)
	}

	if server.Command != "" {
		return ErrCommandNotAllowedForRemote
	}

	if server.OAuth != nil && server.OAuth.ClientID == "" {
		return ErrOauthClientIDRequired
	}

	return nil
}

// validateSmithery validates Smithery transport configuration.
func (m *MCPConfigStore) validateSmithery(server MCPServer) error {
	// API key is always required.
	if server.SmitheryAPIKey == "" {
		return ErrSmitheryAPIKeyRequired
	}
	// For static mode (URL provided), namespace is also required.
	if server.URL != "" && server.SmitheryNamespace == "" {
		return ErrSmitheryNamespaceRequired
	}
	// Command is not allowed.
	if server.Command != "" {
		return ErrCommandNotAllowedForSmithery
	}

	return nil
}

// writeConfig writes the configuration to file.
func (m *MCPConfigStore) writeConfig() error {
	// Determine config file path.
	configFile := m.configFile
	if configFile == "" {
		// Use default: ~/.spin/spin.yaml.
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		spinDir := homeDir + "/.spin"
		configFile = spinDir + "/spin.yaml"
		m.configFile = configFile

		// Create directory if needed.
		err = os.MkdirAll(spinDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Get all settings.
	settings := m.loader.AllSettings()

	// Marshal to YAML.
	data, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file (atomic).
	tmpFile := configFile + ".tmp"
	err = os.WriteFile(tmpFile, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Rename (atomic operation).
	err = os.Rename(tmpFile, configFile)
	if err != nil {
		os.Remove(tmpFile) // Cleanup.

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

// FormatSource returns the SOURCE column value: registry_name@loadout_type.
func FormatSource(server MCPServer) string {
	return server.Name + "@" + GetLoadoutType(server)
}
