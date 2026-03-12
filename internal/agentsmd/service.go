package agentsmd

import (
	"errors"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

var ErrAgentsMdNotFoundAtCustom = errors.New("AGENTS.md not found at custom path")

// Service provides AGENTS.md content for system prompt injection.
type Service struct {
	config     *Config
	discoverer Discoverer
	logger     *slog.Logger
	workDir    string

	mu      sync.RWMutex
	content string
	path    string
	loaded  bool
}

// NewService creates a new AGENTS.md service.
// workDir is the working directory for discovery.
// gitRoot is optional; pass empty string if not in a git repository.
func NewService(cfg *Config, workDir, gitRoot string) *Service {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Service{
		config:     cfg,
		discoverer: NewDiscoverer(gitRoot),
		logger:     slog.Default(),
		workDir:    workDir,
	}
}

// Load discovers and loads AGENTS.md content.
// Returns nil if the file is not found (not an error condition).
// Returns an error for filesystem errors or if custom path doesn't exist.
func (s *Service) Load(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.DebugContext(ctx, "AGENTS.md loading disabled")

		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		path string
		err  error
	)

	// Use custom path if specified.

	if s.config.Path != "" {
		path = s.config.Path
		if !fileExists(path) {
return fmt.Errorf("AGENTS.md not found at custom path: %s: %w", path, ErrAgentsMdNotFoundAtCustom)
		}
	} else {
		// Auto-discover.
		path, err = s.discoverer.Discover(ctx, s.workDir)
		if err != nil {
			return fmt.Errorf("discover AGENTS.md: %w", err)
		}
	}

	// Not found is not an error.
	if path == "" {
		s.logger.DebugContext(ctx, "AGENTS.md not found")

		return nil
	}

	// Read file with size limit.
	content, err := s.readWithLimit(path)
	if err != nil {
		return fmt.Errorf("read AGENTS.md: %w", err)
	}

	s.content = content
	s.path = path
	s.loaded = true

	s.logger.InfoContext(ctx, "loaded AGENTS.md", "path", path, "size", len(content))

	return nil
}

// Content returns the cached AGENTS.md content.
// Returns empty string if not loaded or disabled.
func (s *Service) Content() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.content
}

// IsLoaded returns true if AGENTS.md was found and loaded.
func (s *Service) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loaded
}

// Path returns the path to the loaded AGENTS.md file.
// Returns empty string if not loaded.
func (s *Service) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.path
}

// Refresh re-reads the AGENTS.md file from disk.
// Useful for long-running sessions where the file may have changed.
func (s *Service) Refresh(ctx context.Context) error {
	s.mu.Lock()
	s.content = ""
	s.path = ""
	s.loaded = false
	s.mu.Unlock()

	return s.Load(ctx)
}

// readWithLimit reads file content up to MaxSize bytes.
func (s *Service) readWithLimit(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat agents.md: %w", err)
	}

	// Determine read size.
	readSize := info.Size()
	truncated := false

	if s.config.MaxSize > 0 && info.Size() > s.config.MaxSize {
		readSize = s.config.MaxSize
		truncated = true

		s.logger.Warn("AGENTS.md exceeds size limit, truncating",
			"path", path,
			"size", info.Size(),
			"limit", s.config.MaxSize)
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open agents.md: %w", err)
	}
	defer file.Close()

	// Read up to readSize bytes.
	buf := make([]byte, readSize)

	n, err := io.ReadFull(file, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("read agents.md: %w", err)
	}

	content := string(buf[:n])

	// Add truncation notice if needed.
	if truncated {
		content += "\n\n[Content truncated due to size limit]"
	}

	return content, nil
}
