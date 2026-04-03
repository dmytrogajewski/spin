package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/dmytrogajewski/spin/pkg/storage"
)

var (
	// ErrMcpServerNotFound is a sentinel error.
	ErrMcpServerNotFound = errors.New("mcp server not found")
	// ErrMcpServerAlreadyExists is a sentinel error.
	ErrMcpServerAlreadyExists = errors.New("mcp server already exists")
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
	SmitheryAPIKey string `json:"smithery_api_key,omitempty" mapstructure:"smithery_api_key" yaml:"smithery_api_key,omitempty"`

	SmitheryNamespace string `json:"smithery_namespace,omitempty" mapstructure:"smithery_namespace" yaml:"smithery_namespace,omitempty"`

	// DynamicLoadout enables dynamic tool discovery via search.
	DynamicLoadout bool `json:"dynamic_loadout,omitempty" mapstructure:"dynamic_loadout" yaml:"dynamic_loadout,omitempty"`
}

// toConfigV2 converts an MCPServer to an MCPServerConfigV2 for validation.
func (s MCPServer) toConfigV2() *MCPServerConfigV2 {
	return &MCPServerConfigV2{
		Name:              s.Name,
		Transport:         s.Transport,
		Command:           s.Command,
		Args:              s.Args,
		Env:               s.Env,
		URL:               s.URL,
		Headers:           s.Headers,
		OAuth:             s.OAuth,
		SmitheryAPIKey:    s.SmitheryAPIKey,
		SmitheryNamespace: s.SmitheryNamespace,
		DynamicLoadout:    s.DynamicLoadout,
	}
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

// validate validates an MCP server configuration by delegating to MCPServerConfigV2.Validate.
func (m *MCPConfigStore) validate(server MCPServer) error {
	return server.toConfigV2().Validate()
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

		spinDir := filepath.Join(homeDir, ".spin")
		configFile = filepath.Join(spinDir, "spin.yaml")
		m.configFile = configFile

		// Create directory if needed.
		err = os.MkdirAll(spinDir, 0o700)
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
	err = storage.AtomicWriteFile(context.Background(), configFile, data, storage.DefaultFilePerm)
	if err != nil {
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
	case MCPTransportSSE, MCPTransportStreamableHTTP:
		return "remote"
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
