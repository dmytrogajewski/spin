package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dmytrogajewski/spin/internal/mcp"
)

var permittedMCPFields = map[string]struct{}{
	fieldSchema:     {},
	fieldMCPServers: {},
}

var stdioFields = map[string]struct{}{
	fieldType:    {},
	fieldCommand: {},
	fieldArgs:    {},
	fieldEnv:     {},
	fieldCwd:     {},
}

var remoteFields = map[string]struct{}{
	fieldType:    {},
	fieldURL:     {},
	fieldHeaders: {},
}

func loadMCP(root string) (MCPFile, []string, error) {
	path := filepath.Join(root, MCPFileName)

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MCPFile{}, nil, nil
		}

		return MCPFile{}, nil, fmt.Errorf("read %s: %w", path, err)
	}

	return ParseMCP(raw)
}

// ParseMCP decodes mcp.json bytes. Empty mcpServers is valid.
func ParseMCP(data []byte) (MCPFile, []string, error) {
	raw, err := decodeObject(data)
	if err != nil {
		return MCPFile{}, nil, fmt.Errorf("%w: %w", ErrInvalidMCP, err)
	}

	if unknown := unknownMCPTopLevel(raw); len(unknown) > 0 {
		return MCPFile{}, nil, fmt.Errorf("%w: unknown field %q", ErrInvalidMCP, unknown[0])
	}

	file := MCPFile{}
	if schemaErr := assignMCPSchema(&file, raw); schemaErr != nil {
		return MCPFile{}, nil, schemaErr
	}

	servers, warnings, serversErr := parseMCPServers(raw)
	if serversErr != nil {
		return MCPFile{}, nil, serversErr
	}

	file.Servers = servers

	return file, warnings, nil
}

func unknownMCPTopLevel(raw map[string]json.RawMessage) []string {
	unknown := make([]string, 0)

	for key := range raw {
		if _, ok := permittedMCPFields[key]; ok {
			continue
		}

		unknown = append(unknown, key)
	}

	slices.Sort(unknown)

	return unknown
}

func assignMCPSchema(file *MCPFile, raw map[string]json.RawMessage) error {
	value, ok := raw[fieldSchema]
	if !ok {
		return fmt.Errorf("%w: missing %s", ErrMissingMCPSchema, fieldSchema)
	}

	schema, err := decodeString(value, fieldSchema)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidMCP, err)
	}

	if schema != MCPSchemaV1 {
		return fmt.Errorf("%w: %q", ErrMissingMCPSchema, schema)
	}

	file.Schema = schema

	return nil
}

func parseMCPServers(raw map[string]json.RawMessage) ([]MCPServer, []string, error) {
	value, ok := raw[fieldMCPServers]
	if !ok {
		return nil, nil, fmt.Errorf("%w: missing %s", ErrInvalidMCP, fieldMCPServers)
	}

	obj, isObject := decodeObjectField(value)
	if !isObject {
		return nil, nil, fmt.Errorf("%w: %s must be an object", ErrInvalidMCP, fieldMCPServers)
	}

	if len(obj) == 0 {
		return nil, nil, nil
	}

	return loadServerEntries(obj)
}

func loadServerEntries(obj map[string]json.RawMessage) ([]MCPServer, []string, error) {
	names := make([]string, 0, len(obj))
	for name := range obj {
		names = append(names, name)
	}

	slices.Sort(names)

	servers := make([]MCPServer, 0, len(names))
	warnings := make([]string, 0)

	for _, name := range names {
		server, err := parseServerEntry(name, obj[name])
		if err != nil {
			warnings = append(warnings, warnSkippedMCP+name+": "+err.Error())

			continue
		}

		servers = append(servers, server)
	}

	return servers, warnings, nil
}

func parseServerEntry(name string, raw json.RawMessage) (MCPServer, error) {
	fields, isObject := decodeObjectField(raw)
	if !isObject {
		return MCPServer{}, fmt.Errorf("%w: server must be an object", ErrInvalidMCP)
	}

	transport, err := decodeRequiredTransport(fields)
	if err != nil {
		return MCPServer{}, err
	}

	if fieldErr := rejectForeignFields(fields, transport); fieldErr != nil {
		return MCPServer{}, fieldErr
	}

	server := MCPServer{Name: name, Transport: transport}
	if transport == transportStdio {
		return assignStdioServer(server, fields)
	}

	return assignRemoteServer(server, fields)
}

func decodeRequiredTransport(fields map[string]json.RawMessage) (string, error) {
	value, ok := fields[fieldType]
	if !ok {
		return "", ErrTransportRequired
	}

	transport, err := decodeString(value, fieldType)
	if err != nil {
		return "", err
	}

	switch transport {
	case transportStdio, transportStreamableHTTP, transportSSE:
		return transport, nil
	default:
		return "", fmt.Errorf("%w: %q", mcp.ErrUnsupportedTransport, transport)
	}
}

func rejectForeignFields(fields map[string]json.RawMessage, transport string) error {
	allowed := stdioFields
	if transport != transportStdio {
		allowed = remoteFields
	}

	for key := range fields {
		if _, ok := allowed[key]; ok {
			continue
		}

		return fmt.Errorf("%w: %s is not permitted for %s", ErrInvalidMCP, key, transport)
	}

	return nil
}

func assignStdioServer(server MCPServer, fields map[string]json.RawMessage) (MCPServer, error) {
	command, err := decodeRequiredString(fields, fieldCommand)
	if err != nil {
		return MCPServer{}, err
	}

	server.Command = command

	args, argsErr := decodeOptionalStringSlice(fields, fieldArgs)
	if argsErr != nil {
		return MCPServer{}, argsErr
	}

	server.Args = args

	env, envErr := decodeOptionalStringMap(fields, fieldEnv)
	if envErr != nil {
		return MCPServer{}, envErr
	}

	server.Env = env

	return server, nil
}

func assignRemoteServer(server MCPServer, fields map[string]json.RawMessage) (MCPServer, error) {
	url, err := decodeRequiredString(fields, fieldURL)
	if err != nil {
		return MCPServer{}, err
	}

	server.URL = url

	headers, headerErr := decodeOptionalStringMap(fields, fieldHeaders)
	if headerErr != nil {
		return MCPServer{}, headerErr
	}

	server.Headers = headers

	return server, nil
}

func decodeRequiredString(fields map[string]json.RawMessage, field string) (string, error) {
	value, ok := fields[field]
	if !ok {
		return "", fmt.Errorf("%w: missing %s", ErrInvalidMCP, field)
	}

	return decodeString(value, field)
}

func decodeOptionalStringSlice(fields map[string]json.RawMessage, field string) ([]string, error) {
	value, ok := fields[field]
	if !ok {
		return []string{}, nil
	}

	var items []string
	if err := json.Unmarshal(value, &items); err != nil {
		return nil, fmt.Errorf("%w: %s must be an array of strings", ErrInvalidField, field)
	}

	return items, nil
}

func decodeOptionalStringMap(fields map[string]json.RawMessage, field string) (map[string]string, error) {
	value, ok := fields[field]
	if !ok {
		return map[string]string{}, nil
	}

	var items map[string]string
	if err := json.Unmarshal(value, &items); err != nil {
		return nil, fmt.Errorf("%w: %s must be an object of strings", ErrInvalidField, field)
	}

	return items, nil
}

// ServerConfigs maps valid MCP servers onto the existing MCP manager types.
// Transport is always set; empty type never becomes stdio.
func ServerConfigs(root string, file MCPFile) []mcp.ServerConfig {
	configs := make([]mcp.ServerConfig, 0, len(file.Servers))

	for _, server := range file.Servers {
		cfg, err := mapServerConfig(root, server)
		if err != nil {
			continue
		}

		configs = append(configs, cfg)
	}

	return configs
}

func mapServerConfig(root string, server MCPServer) (mcp.ServerConfig, error) {
	transport, err := mcp.ParsePluginTransport(server.Transport)
	if err != nil {
		return mcp.ServerConfig{}, err
	}

	cfg := mcp.ServerConfig{
		Name:      server.Name,
		Transport: transport,
		Args:      server.Args,
		Env:       server.Env,
		URL:       server.URL,
		Headers:   server.Headers,
	}

	if transport != mcp.TransportStdio {
		return cfg, nil
	}

	command, cmdErr := resolveStdioCommand(root, server.Command)
	if cmdErr != nil {
		return mcp.ServerConfig{}, cmdErr
	}

	cfg.Command = command

	return cfg, nil
}

func resolveStdioCommand(root, command string) (string, error) {
	if !strings.HasPrefix(command, relPathPrefix) {
		return command, nil
	}

	return Contain(root, command)
}
