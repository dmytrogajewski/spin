package lsp

import (
	"context"
	"encoding/json"
	"fmt"
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
}

// NewServer creates a server wrapping the given transport for the specified language.
func NewServer(transport Transport, langConfig LanguageConfig) *Server {
	return &Server{
		transport:  transport,
		langConfig: langConfig,
		alive:      true,
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

// FindDefinition sends textDocument/definition and returns locations.
func (s *Server) FindDefinition(ctx context.Context, uri string, line, character int) ([]Location, error) {
	if checkErr := s.checkReady(); checkErr != nil {
		return nil, checkErr
	}

	params := buildPositionParams(uri, line, character)

	result, sendErr := s.transport.Send(ctx, "textDocument/definition", params)
	if sendErr != nil {
		return nil, fmt.Errorf("find definition: %w", sendErr)
	}

	return parseLocations(result)
}

// FindReferences sends textDocument/references and returns locations.
func (s *Server) FindReferences(ctx context.Context, uri string, line, character int) ([]Location, error) {
	if checkErr := s.checkReady(); checkErr != nil {
		return nil, checkErr
	}

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
