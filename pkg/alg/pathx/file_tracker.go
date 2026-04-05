package pathx

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// modTimeTolerance is the tolerance window for filesystem timestamp granularity.
const modTimeTolerance = 50 * time.Millisecond

// ErrFileModifiedSinceRead is returned when a file has been modified since the last read.
var ErrFileModifiedSinceRead = errors.New("file modified since last read")

// FileTracker tracks file read timestamps to detect stale reads.
// It records when each file was last read and can assert that a file
// has not been modified since the last read, preventing overwrites
// of externally modified files.
type FileTracker struct {
	mu    sync.RWMutex
	reads map[string]time.Time
}

// NewFileTracker creates a new file tracker.
func NewFileTracker() *FileTracker {
	return &FileTracker{
		reads: make(map[string]time.Time),
	}
}

// RecordRead records the current modification time of a file as the last-read timestamp.
func (ft *FileTracker) RecordRead(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file for read tracking: %w", err)
	}

	ft.mu.Lock()
	ft.reads[path] = info.ModTime()
	ft.mu.Unlock()

	return nil
}

// AssertFresh checks that a file has not been modified since the last recorded read.
// If the file was never read through the tracker, the write is allowed — the agent
// may know about the file through other means (e.g., it created the file via a shell
// command). The tracker only blocks writes when a read WAS recorded and the file has
// since been modified externally, preventing stale overwrites.
func (ft *FileTracker) AssertFresh(path string) error {
	ft.mu.RLock()
	readTime, exists := ft.reads[path]
	ft.mu.RUnlock()

	// No recorded read — the agent never read this file through read_file.
	// Allow the write: the file may be new, or created/known via shell commands.
	if !exists {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file for freshness check: %w", err)
	}

	modTime := info.ModTime()
	if modTime.Sub(readTime) > modTimeTolerance {
		return fmt.Errorf("%s: %w; re-read the file first", path, ErrFileModifiedSinceRead)
	}

	return nil
}
