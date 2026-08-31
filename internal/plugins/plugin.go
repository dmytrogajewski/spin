// Package plugins loads Agent Plugins 1.0 packages and contains package paths.
package plugins

import (
	"encoding/json"
	"errors"

	"github.com/dmytrogajewski/spin/internal/skills"
)

// ManifestFile is the required manifest filename at the plugin root.
const ManifestFile = "plugin.json"

// SkillsDir is the fixed skills discovery directory.
const SkillsDir = "skills"

// MCPFileName is the fixed MCP configuration filename at the plugin root.
const MCPFileName = "mcp.json"

// RelPluginsDir is the search directory under workdir and home.
const RelPluginsDir = ".spin/plugins"

// SpinAgentExtension is the reverse-domain namespace for spin-specific extras.
const SpinAgentExtension = "com.spin.agent"

// SpinAgentHooksDir is the conventional hooks directory under the plugin root.
const SpinAgentHooksDir = "com.spin.agent/hooks"

// SchemaV1 is the canonical Agent Plugins 1.0.0 plugin.json schema identifier.
const SchemaV1 = "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json"

// MCPSchemaV1 is the canonical Agent Plugins 1.0.0 mcp.json schema identifier.
const MCPSchemaV1 = "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json"

const (
	// MaxNameLen is the Agent Plugins maximum length for name.
	MaxNameLen = 64

	relPathPrefix = "./"
	pathDotDot    = ".."

	fieldSchema      = "$schema"
	fieldName        = "name"
	fieldVersion     = "version"
	fieldDescription = "description"
	fieldAuthor      = "author"
	fieldHomepage    = "homepage"
	fieldRepository  = "repository"
	fieldLicense     = "license"
	fieldKeywords    = "keywords"
	fieldExtensions  = "extensions"

	fieldAuthorName  = "name"
	fieldAuthorEmail = "email"
	fieldAuthorURL   = "url"

	warnUnknownField   = "unknown field "
	warnExtensionsType = "extensions is not an object; ignored"
	warnSkillsNotDir   = "skills is not a directory; ignored"
	warnSkippedSkill   = "skipped skill "
	warnSkippedMCP     = "skipped mcp server "
	warnInvalidMCP     = "mcp.json disabled: "

	fieldMCPServers = "mcpServers"
	fieldType       = "type"
	fieldCommand    = "command"
	fieldArgs       = "args"
	fieldEnv        = "env"
	fieldURL        = "url"
	fieldHeaders    = "headers"
	fieldCwd        = "cwd"

	transportStdio          = "stdio"
	transportStreamableHTTP = "streamable-http"
	transportSSE            = "sse"
)

var (
	// ErrMissingManifest is returned when plugin.json is absent.
	ErrMissingManifest = errors.New("plugin.json not found")
	// ErrInvalidManifest is returned when plugin.json is not a valid object.
	ErrInvalidManifest = errors.New("invalid plugin.json")
	// ErrInvalidSchema is returned when $schema is missing or unsupported.
	ErrInvalidSchema = errors.New("unsupported plugin schema")
	// ErrInvalidName is returned when name violates Agent Plugins rules.
	ErrInvalidName = errors.New("invalid plugin name")
	// ErrInvalidField is returned when a known field has the wrong JSON type.
	ErrInvalidField = errors.New("invalid plugin field")
	// ErrPathEscape is returned when a package path leaves the plugin root.
	ErrPathEscape = errors.New("path escapes plugin root")
	// ErrNotPluginRelative is returned when a path does not start with ./.
	ErrNotPluginRelative = errors.New("path is not plugin-relative")
	// ErrEmptyPath is returned when Contain is called with an empty path.
	ErrEmptyPath = errors.New("plugin path is empty")
	// ErrInvalidMCP is returned when mcp.json fails top-level validation.
	ErrInvalidMCP = errors.New("invalid mcp.json")
	// ErrMissingMCPSchema is returned when mcp.json $schema is missing or unsupported.
	ErrMissingMCPSchema = errors.New("unsupported mcp schema")
	// ErrTransportRequired is returned when an MCP server omits type.
	ErrTransportRequired = errors.New("mcp transport type is required")
)

var permittedFields = map[string]struct{}{
	fieldSchema:      {},
	fieldName:        {},
	fieldVersion:     {},
	fieldDescription: {},
	fieldAuthor:      {},
	fieldHomepage:    {},
	fieldRepository:  {},
	fieldLicense:     {},
	fieldKeywords:    {},
	fieldExtensions:  {},
}

// Author is the optional plugin.json author object.
type Author struct {
	// Name is an optional author display name.
	Name string
	// Email is an optional author email.
	Email string
	// URL is an optional author URL.
	URL string
}

// Manifest is a parsed Agent Plugins 1.0 plugin.json object.
type Manifest struct {
	// Schema is the canonical $schema identifier.
	Schema string
	// Name is the human-readable plugin name.
	Name string
	// Version is an optional version string.
	Version string
	// Description is an optional short description.
	Description string
	// Author is an optional author object.
	Author *Author
	// Homepage is an optional homepage URL.
	Homepage string
	// Repository is an optional source repository URL.
	Repository string
	// License is an optional license identifier.
	License string
	// Keywords are optional search tags.
	Keywords []string
	// Extensions holds client-specific objects keyed by namespace.
	Extensions map[string]json.RawMessage
	// UnknownFields lists top-level keys that were reported and ignored.
	UnknownFields []string
	// ExtensionsIgnored is true when extensions was present but not an object.
	ExtensionsIgnored bool
}

// Plugin is a validated plugin root with discovered immediate skills.
type Plugin struct {
	// Root is the plugin directory passed to Load.
	Root string
	// Manifest is the validated plugin.json record.
	Manifest Manifest
	// Skills are immediate children of skills/ that parsed successfully.
	Skills []skills.Skill
	// MCP is the parsed mcp.json file when MCPValid is true.
	MCP MCPFile
	// MCPValid is true when mcp.json loaded (including empty mcpServers).
	MCPValid bool
	// Warnings are non-fatal reports (unknown fields, skipped skills, MCP).
	Warnings []string
}

// MCPFile is a parsed Agent Plugins mcp.json object.
type MCPFile struct {
	// Schema is the canonical $schema identifier.
	Schema string
	// Servers are valid mapped server entries in declaration order.
	Servers []MCPServer
}

// MCPServer is one mcpServers member after closed-variant validation.
type MCPServer struct {
	// Name is the mcpServers object key.
	Name string
	// Transport is the explicit type: stdio, streamable-http, or sse.
	Transport string
	// Command is the stdio executable token.
	Command string
	// Args are stdio arguments.
	Args []string
	// Env is the stdio environment overlay.
	Env map[string]string
	// URL is the remote endpoint.
	URL string
	// Headers are remote HTTP headers.
	Headers map[string]string
}
