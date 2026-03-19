package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Server manages the lifecycle of a single language server process.
// It handles LSP initialization, request routing, and shutdown.
type Server struct {
	mu          sync.Mutex
	transport   Transport
	langConfig  LanguageConfig
	rootURI     string
	initialized bool
	alive       bool
	cache       *Cache
}

// NewServer creates a server wrapping the given transport for the specified language.
func NewServer(transport Transport, langConfig LanguageConfig) *Server {
	return &Server{
		transport:  transport,
		langConfig: langConfig,
		alive:      true,
		cache:      NewCache(),
	}
}

// Initialize performs the LSP initialize handshake with the server.
func (s *Server) Initialize(ctx context.Context, rootURI string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	params := buildInitializeParams(rootURI)

	_, sendErr := s.transport.Send(ctx, "initialize", params)
	if sendErr != nil {
		return fmt.Errorf("initialize: %w", sendErr)
	}

	if notifyErr := s.transport.Notify(ctx, "initialized", map[string]any{}); notifyErr != nil {
		return fmt.Errorf("initialized notification: %w", notifyErr)
	}

	s.rootURI = rootURI
	s.initialized = true

	return nil
}

// openFileForLSP reads a file from disk and sends textDocument/didOpen to the
// language server with the actual content. This is required because LSP servers
// need the full file text to provide accurate results.
func (s *Server) openFileForLSP(ctx context.Context, uri string) {
	filePath := strings.TrimPrefix(uri, "file://")

	content, err := os.ReadFile(filePath)
	if err != nil {
		// Best-effort — DidOpen with empty content as fallback.
		_ = s.DidOpen(ctx, uri, s.langConfig.ID, "")

		return
	}

	_ = s.DidOpen(ctx, uri, s.langConfig.ID, string(content))
}

// FindDefinition sends textDocument/definition and returns locations.
func (s *Server) FindDefinition(ctx context.Context, uri string, line, character int) ([]Location, error) {
	if checkErr := s.checkReady(); checkErr != nil {
		return nil, checkErr
	}

	// Ensure the file is opened in the language server with actual content.
	s.openFileForLSP(ctx, uri)

	// Check cache first.
	hash := ContentHash([]byte(uri))
	if cached := s.cache.GetRaw("textDocument/definition", uri, hash); cached != nil {
		return parseLocations(cached)
	}

	params := buildPositionParams(uri, line, character)

	result, sendErr := s.transport.Send(ctx, "textDocument/definition", params)
	if sendErr != nil {
		return nil, fmt.Errorf("find definition: %w", sendErr)
	}

	// Cache the result.
	s.cache.PutRaw("textDocument/definition", uri, hash, result)

	return parseLocations(result)
}

// FindReferences sends textDocument/references and returns locations.
func (s *Server) FindReferences(ctx context.Context, uri string, line, character int) ([]Location, error) {
	if checkErr := s.checkReady(); checkErr != nil {
		return nil, checkErr
	}

	// Ensure the file is opened with actual content.
	s.openFileForLSP(ctx, uri)

	params := buildPositionParams(uri, line, character)
	params["context"] = map[string]any{
		"includeDeclaration": true,
	}

	result, sendErr := s.transport.Send(ctx, "textDocument/references", params)
	if sendErr != nil {
		return nil, fmt.Errorf("find references: %w", sendErr)
	}

	return parseLocations(result)
}

// Rename sends textDocument/rename and returns a workspace edit.
func (s *Server) Rename(ctx context.Context, uri string, line, character int, newName string) (*WorkspaceEdit, error) {
	if checkErr := s.checkReady(); checkErr != nil {
		return nil, checkErr
	}

	// Ensure file is opened with actual content.
	s.openFileForLSP(ctx, uri)

	params := buildPositionParams(uri, line, character)
	params["newName"] = newName

	result, sendErr := s.transport.Send(ctx, "textDocument/rename", params)
	if sendErr != nil {
		return nil, fmt.Errorf("rename: %w", sendErr)
	}

	var edit WorkspaceEdit
	if unmarshalErr := json.Unmarshal(result, &edit); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal rename result: %w", unmarshalErr)
	}

	return &edit, nil
}

// SearchSymbols returns symbols matching the given pattern for the file URI.
// Results are cached via the L2 symbol cache and filtered via ParseMatcher.
func (s *Server) SearchSymbols(ctx context.Context, uri, pattern string) ([]Symbol, error) {
	if checkErr := s.checkReady(); checkErr != nil {
		return nil, checkErr
	}

	hash := ContentHash([]byte(uri))

	if cached := s.cache.GetSymbols(uri, hash); cached != nil {
		matcher := ParseMatcher(pattern)

		return FilterSymbols(cached, matcher), nil
	}

	result, sendErr := s.transport.Send(ctx, "textDocument/documentSymbol", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
	if sendErr != nil {
		return nil, fmt.Errorf("document symbols (%s): %w", SeverityError.String(), sendErr)
	}

	var symbols []Symbol
	if unmarshalErr := json.Unmarshal(result, &symbols); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal symbols: %w", unmarshalErr)
	}

	s.cache.PutSymbols(uri, hash, symbols)

	matcher := ParseMatcher(pattern)

	return FilterSymbols(symbols, matcher), nil
}

// DidOpen sends textDocument/didOpen notification.
func (s *Server) DidOpen(ctx context.Context, uri, languageID, text string) error {
	if checkErr := s.checkReady(); checkErr != nil {
		return checkErr
	}

	params := map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": languageID,
			"version":    1,
			"text":       text,
		},
	}

	if notifyErr := s.transport.Notify(ctx, "textDocument/didOpen", params); notifyErr != nil {
		return fmt.Errorf("did open: %w", notifyErr)
	}

	return nil
}

// DidChange sends textDocument/didChange notification with full content sync.
func (s *Server) DidChange(ctx context.Context, uri string, version int, text string) error {
	if checkErr := s.checkReady(); checkErr != nil {
		return checkErr
	}

	params := map[string]any{
		"textDocument": map[string]any{
			"uri":     uri,
			"version": version,
		},
		"contentChanges": []map[string]any{
			{"text": text},
		},
	}

	if notifyErr := s.transport.Notify(ctx, "textDocument/didChange", params); notifyErr != nil {
		return fmt.Errorf("did change: %w", notifyErr)
	}

	// Invalidate cache for changed file.
	s.cache.Invalidate(uri)

	return nil
}

// Shutdown sends the shutdown request and exit notification.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil
	}

	_, sendErr := s.transport.Send(ctx, "shutdown", nil)
	if sendErr != nil {
		return fmt.Errorf("shutdown request: %w", sendErr)
	}

	if notifyErr := s.transport.Notify(ctx, "exit", nil); notifyErr != nil {
		return fmt.Errorf("exit notification: %w", notifyErr)
	}

	s.initialized = false
	s.alive = false

	return nil
}

// IsAlive reports whether the server process is still running.
func (s *Server) IsAlive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.alive
}

// SetAlive sets the alive state of the server.
func (s *Server) SetAlive(alive bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.alive = alive
}

// Close shuts down the server and its transport.
func (s *Server) Close(ctx context.Context) error {
	shutdownErr := s.Shutdown(ctx)

	if closeErr := s.transport.Close(); closeErr != nil {
		if shutdownErr != nil {
			return fmt.Errorf("shutdown: %w, close: %w", shutdownErr, closeErr)
		}

		return fmt.Errorf("close transport: %w", closeErr)
	}

	return shutdownErr
}

// Language returns the language configuration for this server.
func (s *Server) Language() LanguageConfig {
	return s.langConfig
}

// checkReady returns an error if the server is not ready for requests.
func (s *Server) checkReady() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return ErrServerNotInitialized
	}

	if !s.alive {
		return ErrServerDead
	}

	return nil
}

// buildInitializeParams constructs the LSP initialize request params.
func buildInitializeParams(rootURI string) map[string]any {
	return map[string]any{
		"processId": nil,
		"rootUri":   rootURI,
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"definition": map[string]any{"dynamicRegistration": true},
				"references": map[string]any{"dynamicRegistration": true},
				"rename": map[string]any{
					"dynamicRegistration": true,
					"prepareSupport":      true,
				},
			},
		},
	}
}

// buildPositionParams constructs textDocument + position params.
func buildPositionParams(uri string, line, character int) map[string]any {
	return map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": line, "character": character},
	}
}

// parseLocations unmarshals a JSON result as either a single Location or an array.
func parseLocations(raw json.RawMessage) ([]Location, error) {
	var locations []Location
	if unmarshalErr := json.Unmarshal(raw, &locations); unmarshalErr == nil {
		return locations, nil
	}

	var single Location
	if unmarshalErr := json.Unmarshal(raw, &single); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal locations: %w", unmarshalErr)
	}

	return []Location{single}, nil
}
