package lsp

import (
	"context"
	"fmt"
	"sync"
)

// ServerFactory creates a new Server for the given language config and root URI.
// It handles process startup and transport creation.
type ServerFactory func(ctx context.Context, lang LanguageConfig, rootURI string) (*Server, error)

// Manager manages language server instances — one per language.
// Servers are started lazily on first request and restarted if they crash.
type Manager struct {
	mu      sync.Mutex
	servers map[string]*Server
	rootURI string
	factory ServerFactory
}

// NewManager creates a manager for the given workspace root.
// The factory function is called to create new server instances.
func NewManager(rootURI string, factory ServerFactory) *Manager {
	return &Manager{
		servers: make(map[string]*Server),
		rootURI: rootURI,
		factory: factory,
	}
}

// ForFile returns a running, initialized server for the given file path.
// It detects the language, lazily starts the server (double-check locking),
// and restarts dead servers.
func (m *Manager) ForFile(ctx context.Context, filePath string) (*Server, error) {
	lang, detectErr := DetectLanguage(filePath)
	if detectErr != nil {
		return nil, detectErr
	}

	return m.serverForLanguage(ctx, lang)
}

// Close shuts down all managed servers gracefully.
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error

	for _, srv := range m.servers {
		langID := srv.Language()
		if closeErr := srv.Close(ctx); closeErr != nil && firstErr == nil {
			firstErr = fmt.Errorf("close server %s: %w", langID, closeErr)
		}
	}

	clear(m.servers)

	return firstErr
}

// serverForLanguage returns or creates a server for the given language.
func (m *Manager) serverForLanguage(ctx context.Context, lang LanguageConfig) (*Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if srv, ok := m.servers[lang.ID]; ok {
		if srv.IsAlive() {
			return srv, nil
		}

		// Server died — mark it and create a fresh one.
		srv.SetAlive(false)
	}

	srv, createErr := m.factory(ctx, lang, m.rootURI)
	if createErr != nil {
		return nil, createErr
	}

	m.servers[lang.ID] = srv

	return srv, nil
}
