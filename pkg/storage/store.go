// Package storage provides unified storage implementations for persistence.
// Domain packages (session, history) define their own interfaces;
// this package provides reusable implementations.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dmytrogajewski/spin/pkg/alg/pathx"
)

var (
	// ErrPathExistsButIsNotA is a sentinel error.
	ErrPathExistsButIsNotA = errors.New("path exists but is not a directory")
	// ErrKeyCannotBeEmpty is a sentinel error.
	ErrKeyCannotBeEmpty = errors.New("key cannot be empty")
	// ErrNotFound is a sentinel error.
	ErrNotFound = errors.New("not found")
)

// Store is a generic key-value store interface.
// Domain packages can use this or define their own specialized interfaces.
type Store[T any] interface {
	// Save persists data with the given key.
	Save(ctx context.Context, key string, data T) error

	// Load retrieves data by key.
	Load(ctx context.Context, key string) (T, error)

	// Delete removes data by key.
	Delete(ctx context.Context, key string) error

	// Exists checks if key exists.
	Exists(ctx context.Context, key string) (bool, error)

	// List returns all keys.
	List(ctx context.Context) ([]string, error)
}

// FileStore implements Store using the filesystem.
// It handles JSON serialization, atomic writes, and concurrent access.
type FileStore[T any] struct {
	baseDir string
	suffix  string       // File suffix (e.g., ".json", ".history.json").
	mu      sync.RWMutex // Concurrent access protection.
}

// FileStoreConfig configures a FileStore instance.
type FileStoreConfig struct {
	// BaseDir is the directory where files are stored (required).
	BaseDir string

	// Suffix is the file extension/suffix (default: ".json").
	Suffix string
}

// NewFileStore creates a new file-based store.
func NewFileStore[T any](cfg FileStoreConfig) (*FileStore[T], error) {
	baseDir := cfg.BaseDir

	// Expand home directory if needed.
	expanded, err := pathx.ExpandHome(baseDir)
	if err != nil {
		return nil, fmt.Errorf("expand home directory: %w", err)
	}

	baseDir = expanded

	// Check if path exists and is a file (not directory).
	info, err := os.Stat(baseDir)
	if err == nil && !info.IsDir() {
		return nil, fmt.Errorf("path exists but is not a directory: %s: %w", baseDir, ErrPathExistsButIsNotA)
	}

	// Create directory if it doesn't exist.
	err = os.MkdirAll(baseDir, 0o700)
	if err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	suffix := cfg.Suffix
	if suffix == "" {
		suffix = ".json"
	}

	return &FileStore[T]{
		baseDir: baseDir,
		suffix:  suffix,
	}, nil
}

// Save persists data with atomic write.
func (fs *FileStore[T]) Save(ctx context.Context, key string, data T) error {
	if key == "" {
		return ErrKeyCannotBeEmpty
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Serialize to JSON.
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize data: %w", err)
	}

	return AtomicWriteFile(ctx, fs.filePath(key), jsonData, DefaultFilePerm)
}

// Load retrieves data by key.
func (fs *FileStore[T]) Load(ctx context.Context, key string) (T, error) {
	var zero T

	if key == "" {
		return zero, ErrKeyCannotBeEmpty
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return zero, fmt.Errorf("load %s: %w", key, err)
	}

	path := fs.filePath(key)

	jsonData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return zero, fmt.Errorf("not found: %s: %w", key, ErrNotFound)
		}

		return zero, fmt.Errorf("read file: %w", err)
	}

	var data T

	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return zero, fmt.Errorf("deserialize data: %w", err)
	}

	return data, nil
}

// Delete removes data by key.
func (fs *FileStore[T]) Delete(ctx context.Context, key string) error {
	if key == "" {
		return ErrKeyCannotBeEmpty
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}

	path := fs.filePath(key)

	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

// Exists checks if key exists.
func (fs *FileStore[T]) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, ErrKeyCannotBeEmpty
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("exists %s: %w", key, err)
	}

	path := fs.filePath(key)

	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, fmt.Errorf("stat file: %w", err)
}

// List returns all keys.
func (fs *FileStore[T]) List(ctx context.Context) ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}

	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var keys []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, fs.suffix) && !strings.HasSuffix(name, ".tmp") {
			// Extract key from filename.
			key := strings.TrimSuffix(name, fs.suffix)
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// filePath returns the file path for a key.
func (fs *FileStore[T]) filePath(key string) string {
	return filepath.Join(fs.baseDir, key+fs.suffix)
}

// GetBaseDir returns the base directory (useful for testing).
func (fs *FileStore[T]) GetBaseDir() string {
	return fs.baseDir
}
