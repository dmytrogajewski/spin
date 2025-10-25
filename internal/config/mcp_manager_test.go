package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestMCPManager_List_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	// Create empty config
	err := os.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	mgr := NewMCPManager(loader)
	servers, err := mgr.List()

	require.NoError(t, err)
	assert.Empty(t, servers)
}

func TestMCPManager_List_Multiple(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	// Create config with two servers
	yamlContent := `
mcp:
  servers:
    - name: filesystem
      command: npx
      args:
        - -y
        - "@modelcontextprotocol/server-filesystem"
        - /workspace
    - name: github
      command: mcp-server-github
      args:
        - --token-file
        - ~/.github-token
`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	mgr := NewMCPManager(loader)
	servers, err := mgr.List()

	require.NoError(t, err)
	require.Len(t, servers, 2)
	assert.Equal(t, "filesystem", servers[0].Name)
	assert.Equal(t, "github", servers[1].Name)
}

func TestMCPManager_Get_Found(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	yamlContent := `
mcp:
  servers:
    - name: filesystem
      command: npx
      args:
        - -y
        - "@modelcontextprotocol/server-filesystem"
        - /workspace
`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	mgr := NewMCPManager(loader)
	server, err := mgr.Get("filesystem")

	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Equal(t, "filesystem", server.Name)
	assert.Equal(t, "npx", server.Command)
}

func TestMCPManager_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	err := os.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	mgr := NewMCPManager(loader)
	server, err := mgr.Get("nonexistent")

	require.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "not found")
}

func TestMCPManager_Add_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	// Create empty config
	err := os.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	mgr := NewMCPManager(loader)
	mgr.configFile = configFile // Set config file path

	newServer := MCPServer{
		Name:    "filesystem",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/workspace"},
	}

	err = mgr.Add(newServer)
	require.NoError(t, err)

	// Verify added
	server, err := mgr.Get("filesystem")
	require.NoError(t, err)
	assert.Equal(t, "filesystem", server.Name)
}

func TestMCPManager_Add_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	yamlContent := `
mcp:
  servers:
    - name: filesystem
      command: npx
`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	mgr := NewMCPManager(loader)
	mgr.configFile = configFile

	newServer := MCPServer{
		Name:    "filesystem",
		Command: "different-cmd",
	}

	err = mgr.Add(newServer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestMCPManager_Add_EmptyName(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	err := os.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	mgr := NewMCPManager(loader)
	mgr.configFile = configFile

	newServer := MCPServer{
		Name:    "",
		Command: "npx",
	}

	err = mgr.Add(newServer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestMCPManager_Add_EmptyCommand(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	err := os.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	mgr := NewMCPManager(loader)
	mgr.configFile = configFile

	newServer := MCPServer{
		Name:    "filesystem",
		Command: "",
	}

	err = mgr.Add(newServer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
}

func TestMCPManager_Remove_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	yamlContent := `
mcp:
  servers:
    - name: filesystem
      command: npx
    - name: github
      command: mcp-server-github
`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	mgr := NewMCPManager(loader)
	mgr.configFile = configFile

	err = mgr.Remove("filesystem")
	require.NoError(t, err)

	// Verify removed
	_, err = mgr.Get("filesystem")
	require.Error(t, err)

	// Verify other server still exists
	server, err := mgr.Get("github")
	require.NoError(t, err)
	assert.Equal(t, "github", server.Name)
}

func TestMCPManager_Remove_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	err := os.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	err = loader.LoadFromFile(configFile)
	require.NoError(t, err)

	mgr := NewMCPManager(loader)
	mgr.configFile = configFile

	err = mgr.Remove("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}



func TestMCPManager_ConfigFile_DefaultCreation(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "spin.yaml")

	// Don't create file - test auto-creation
	loader := NewLoader()
	// Load without error (file not found is ok)
	_ = loader.Load("")

	mgr := NewMCPManager(loader)
	mgr.configFile = configFile

	newServer := MCPServer{
		Name:    "filesystem",
		Command: "npx",
	}

	err := mgr.Add(newServer)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(configFile)
	require.NoError(t, err)
}
