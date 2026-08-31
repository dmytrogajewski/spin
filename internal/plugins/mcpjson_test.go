package plugins_test

// Journey: specs/journeys/JOURNEY-005-load-plugins-merge-skills.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/plugins"
)

func TestParseMCP_EmptyServersValid(t *testing.T) {
	t.Parallel()

	plugin, err := plugins.Load(filepath.Join(fixtureRoot, fixtureSample))
	require.NoError(t, err)
	require.True(t, plugin.MCPValid)
	require.Equal(t, plugins.MCPSchemaV1, plugin.MCP.Schema)
	require.Empty(t, plugin.MCP.Servers)
	require.Len(t, plugin.Skills, 1)
}

func TestParseMCP_ExplicitTransports(t *testing.T) {
	t.Parallel()

	payload := `{
		"$schema":"` + plugins.MCPSchemaV1 + `",
		"mcpServers":{
			"local":{"type":"stdio","command":"true"},
			"http":{"type":"streamable-http","url":"https://example.com/mcp"},
			"legacy":{"type":"sse","url":"https://example.com/sse"}
		}
	}`

	file, warnings, err := plugins.ParseMCP([]byte(payload))
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, file.Servers, 3)

	byName := map[string]plugins.MCPServer{}
	for _, server := range file.Servers {
		byName[server.Name] = server
	}

	require.Equal(t, "stdio", byName["local"].Transport)
	require.Equal(t, "true", byName["local"].Command)
	require.Equal(t, "streamable-http", byName["http"].Transport)
	require.Equal(t, "https://example.com/mcp", byName["http"].URL)
	require.Equal(t, "sse", byName["legacy"].Transport)

	configs := plugins.ServerConfigs(t.TempDir(), file)
	require.Len(t, configs, 3)

	cfgByName := map[string]mcp.ServerConfig{}
	for _, cfg := range configs {
		cfgByName[cfg.Name] = cfg
	}

	require.Equal(t, mcp.TransportStdio, cfgByName["local"].Transport)
	require.Equal(t, mcp.TransportStreamableHTTP, cfgByName["http"].Transport)
	require.Equal(t, mcp.TransportSSE, cfgByName["legacy"].Transport)
}

func TestParseMCP_MissingTypeNotGuessed(t *testing.T) {
	t.Parallel()

	payload := `{
		"$schema":"` + plugins.MCPSchemaV1 + `",
		"mcpServers":{
			"guessed":{"command":"true"}
		}
	}`

	file, warnings, err := plugins.ParseMCP([]byte(payload))
	require.NoError(t, err)
	require.Empty(t, file.Servers)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "guessed")
	require.Empty(t, plugins.ServerConfigs(t.TempDir(), file))
}

func TestParseMCP_InvalidFileKeepsPluginSkills(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writePluginTree(t, root, "keep-skills", "kept",
		`{"$schema":"`+plugins.MCPSchemaV1+`","mcpServers":[]}`)

	plugin, err := plugins.Load(root)
	require.NoError(t, err)
	require.False(t, plugin.MCPValid)
	require.Len(t, plugin.Skills, 1)
	require.Equal(t, "kept", plugin.Skills[0].Name)
	require.NotEmpty(t, plugin.Warnings)
}

func TestParseMCP_SchemaRequired(t *testing.T) {
	t.Parallel()

	_, _, err := plugins.ParseMCP([]byte(`{"mcpServers":{}}`))
	require.ErrorIs(t, err, plugins.ErrMissingMCPSchema)
}

func writePluginTree(t *testing.T, root, pluginName, skillName, mcpJSON string) {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.ManifestFile), minimalManifest(pluginName), filePerm))

	skillDir := filepath.Join(root, plugins.SkillsDir, skillName)
	require.NoError(t, os.MkdirAll(skillDir, dirPerm))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), skillMarkdown(skillName), filePerm))
	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.MCPFileName), []byte(mcpJSON), filePerm))
}
