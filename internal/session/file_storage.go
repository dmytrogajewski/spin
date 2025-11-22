package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileStorage implements Storage using filesystem.
type FileStorage struct {
	baseDir string       // Base directory (e.g., ~/.spin/sessions)
	mu      sync.RWMutex // Concurrent access protection
}

// NewFileStorage creates file-based storage.
func NewFileStorage(baseDir string) (*FileStorage, error) {
	// Expand home directory if needed
	if strings.HasPrefix(baseDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		baseDir = filepath.Join(home, baseDir[2:])
	}

	// Check if path exists and is a file (not directory)
	info, err := os.Stat(baseDir)
	if err == nil && !info.IsDir() {
		return nil, fmt.Errorf("path exists but is not a directory: %s", baseDir)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	return &FileStorage{
		baseDir: baseDir,
	}, nil
}

// Save writes session to storage with atomic write.
func (fs *FileStorage) Save(s *Session) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Validate session
	if err := s.Validate(); err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize session: %w", err)
	}

	// Write to temp file first (atomic write pattern)
	tmpPath := fs.sessionPath(s.ID) + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// Atomic rename
	finalPath := fs.sessionPath(s.ID)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("atomic rename: %w", err)
	}

	return nil
}

// Load reads session from storage.
func (fs *FileStorage) Load(id string) (*Session, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := fs.sessionPath(id)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("deserialize session: %w", err)
	}

	// Validate loaded session
	if err := session.Validate(); err != nil {
		return nil, fmt.Errorf("loaded session is invalid: %w", err)
	}

	return &session, nil
}

// Delete removes session from storage.
func (fs *FileStorage) Delete(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.sessionPath(id)

	// Remove file (don't error if it doesn't exist)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete session file: %w", err)
	}

	return nil
}

// Exists checks if session exists.
func (fs *FileStorage) Exists(id string) (bool, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := fs.sessionPath(id)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// List returns all session IDs with optional filter.
func (fs *FileStorage) List(filter Filter) ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Read all session files
	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		// Skip directories and non-JSON files
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Extract session ID from filename
		id := strings.TrimSuffix(entry.Name(), ".json")

		// Apply filters (need to load session for filtering)
		if fs.matchesFilter(id, filter) {
			ids = append(ids, id)
		}
	}

	// Apply pagination
	return fs.paginate(ids, filter.Offset, filter.Limit), nil
}

// ListMetadata returns session metadata without loading full sessions.
func (fs *FileStorage) ListMetadata(filter Filter) ([]*Metadata, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := fs.readSessionEntries()
	if err != nil {
		return nil, err
	}

	metadataList := fs.collectMetadata(entries, filter)
	return fs.applyPagination(metadataList, filter), nil
}

// readSessionEntries reads all session files from the directory.
func (fs *FileStorage) readSessionEntries() ([]os.DirEntry, error) {
	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	return entries, nil
}

// collectMetadata collects metadata from session files.
func (fs *FileStorage) collectMetadata(entries []os.DirEntry, filter Filter) []*Metadata {
	var metadataList []*Metadata

	for _, entry := range entries {
		if !fs.isValidSessionFile(entry) {
			continue
		}

		id := fs.extractSessionID(entry.Name())
		if metadata := fs.loadSessionMetadata(id, filter); metadata != nil {
			metadataList = append(metadataList, metadata)
		}
	}

	return metadataList
}

// isValidSessionFile checks if an entry is a valid session file.
func (fs *FileStorage) isValidSessionFile(entry os.DirEntry) bool {
	return !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json")
}

// extractSessionID extracts session ID from filename.
func (fs *FileStorage) extractSessionID(filename string) string {
	return strings.TrimSuffix(filename, ".json")
}

// loadSessionMetadata loads and filters session metadata.
func (fs *FileStorage) loadSessionMetadata(id string, filter Filter) *Metadata {
	session, err := fs.loadSession(id)
	if err != nil {
		return nil
	}

	if !fs.sessionMatchesFilter(session, filter) {
		return nil
	}

	return &session.Metadata
}

// applyPagination applies pagination to the metadata list.
func (fs *FileStorage) applyPagination(metadataList []*Metadata, filter Filter) []*Metadata {
	start, end := fs.paginationRange(len(metadataList), filter.Offset, filter.Limit)
	if start >= len(metadataList) {
		return []*Metadata{}
	}
	return metadataList[start:end]
}

// sessionPath returns the file path for a session ID.
func (fs *FileStorage) sessionPath(id string) string {
	return filepath.Join(fs.baseDir, id+".json")
}

// matchesFilter checks if a session ID matches the filter (loads session).
func (fs *FileStorage) matchesFilter(id string, filter Filter) bool {
	// If no filter criteria, match all
	if filter.State == nil && filter.WorkDir == "" &&
		filter.CreatedAfter == nil && filter.CreatedBefore == nil &&
		len(filter.Tags) == 0 {
		return true
	}

	// Need to load session to check filters
	session, err := fs.loadSession(id)
	if err != nil {
		return false
	}

	return fs.sessionMatchesFilter(session, filter)
}

// sessionMatchesFilter checks if a loaded session matches filter criteria.
func (fs *FileStorage) sessionMatchesFilter(session *Session, filter Filter) bool {
	// State filter
	if filter.State != nil && session.State != *filter.State {
		return false
	}

	// WorkDir filter
	if filter.WorkDir != "" && session.WorkDir != filter.WorkDir {
		return false
	}

	// CreatedAfter filter
	if filter.CreatedAfter != nil && session.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}

	// CreatedBefore filter
	if filter.CreatedBefore != nil && session.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}

	// Tags filter (OR logic - session must have at least one of the filter tags)
	if len(filter.Tags) > 0 {
		hasTag := false
		for _, filterTag := range filter.Tags {
			for _, sessionTag := range session.Metadata.Tags {
				if sessionTag == filterTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	return true
}

// loadSession loads a session without acquiring locks (internal use).
func (fs *FileStorage) loadSession(id string) (*Session, error) {
	path := fs.sessionPath(id)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// paginate applies offset and limit to a slice of IDs.
func (fs *FileStorage) paginate(ids []string, offset, limit int) []string {
	start, end := fs.paginationRange(len(ids), offset, limit)
	if start >= len(ids) {
		return []string{}
	}
	return ids[start:end]
}

// paginationRange calculates start and end indices for pagination.
func (fs *FileStorage) paginationRange(total, offset, limit int) (start, end int) {
	start = offset
	if start < 0 {
		start = 0
	}
	if start >= total {
		return start, start
	}

	end = total
	if limit > 0 {
		end = start + limit
		if end > total {
			end = total
		}
	}

	return start, end
}
