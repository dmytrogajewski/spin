package session

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/dmytrogajewski/spin/pkg/alg/ds"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/storage"
)

var (
	// ErrTranscriptWriterClosed is returned when Append is called after Close.
	ErrTranscriptWriterClosed = errors.New("transcript writer is closed")
	// ErrTranscriptNotFound is returned when the transcript file does not exist.
	ErrTranscriptNotFound = errors.New("transcript file not found")
)

// TranscriptWriter provides append-only JSONL persistence for conversation messages.
// Each message is serialized as a single JSON line and appended under advisory file lock.
// Thread-safe via the underlying [ds.JSONLWriter] mutex.
type TranscriptWriter struct {
	jsonl *ds.JSONLWriter[message.Message]
}

// NewTranscriptWriter creates or opens a JSONL transcript file at the given path.
func NewTranscriptWriter(path string) (*TranscriptWriter, error) {
	writer, err := ds.NewJSONLWriter[message.Message](path)
	if err != nil {
		return nil, fmt.Errorf("open transcript file: %w", err)
	}

	return &TranscriptWriter{jsonl: writer}, nil
}

// Append serializes msg as a single JSON line and appends it under exclusive file lock.
// The context controls the lock acquisition timeout.
func (w *TranscriptWriter) Append(ctx context.Context, msg message.Message) error {
	file := w.jsonl.File()
	if file == nil {
		return ErrTranscriptWriterClosed
	}

	// Acquire exclusive lock for the append.
	fd := storage.SafeFlockFd(file.Fd())
	if lockErr := storage.FlockExclusiveWithContext(ctx, fd); lockErr != nil {
		return fmt.Errorf("lock transcript file: %w", lockErr)
	}

	defer flockUnlockIgnore(fd)

	if err := w.jsonl.Append(msg); err != nil {
		return fmt.Errorf("append transcript: %w", err)
	}

	return nil
}

// ReadAll reads all valid messages from the transcript file.
// Corrupted or empty lines are silently skipped.
// The context controls the lock acquisition timeout.
func (w *TranscriptWriter) ReadAll(ctx context.Context) ([]message.Message, error) {
	// Use the jsonl path to open a separate read handle with shared lock.
	path := w.jsonl.Path()

	file, err := openForSharedRead(ctx, path)
	if errors.Is(err, ErrTranscriptNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	defer closeSharedRead(file)

	msgs, readErr := w.jsonl.ReadAll()
	if readErr != nil {
		return msgs, fmt.Errorf("read transcript: %w", readErr)
	}

	return msgs, nil
}

// Count returns the number of messages appended in this writer's lifetime.
func (w *TranscriptWriter) Count() int {
	return w.jsonl.Count()
}

// Close closes the underlying file handle. Safe to call multiple times.
func (w *TranscriptWriter) Close() error {
	if err := w.jsonl.Close(); err != nil {
		return fmt.Errorf("close transcript: %w", err)
	}

	return nil
}

// openForSharedRead opens a file for reading with a shared advisory lock.
// Returns [ErrTranscriptNotFound] if the file does not exist.
func openForSharedRead(ctx context.Context, path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrTranscriptNotFound
		}

		return nil, fmt.Errorf("open transcript for reading: %w", err)
	}

	fd := storage.SafeFlockFd(file.Fd())
	if lockErr := storage.FlockSharedWithContext(ctx, fd); lockErr != nil {
		_ = file.Close()

		return nil, fmt.Errorf("shared lock transcript file: %w", lockErr)
	}

	return file, nil
}

// closeSharedRead releases the shared lock and closes the file.
func closeSharedRead(file *os.File) {
	fd := storage.SafeFlockFd(file.Fd())

	// Best-effort cleanup; errors are not actionable in defer paths.
	handleCleanupError(storage.FlockUnlock(fd))
	handleCleanupError(file.Close())
}

// flockUnlockIgnore releases a file lock in a deferred context.
func flockUnlockIgnore(fd int) {
	handleCleanupError(storage.FlockUnlock(fd))
}

// handleCleanupError consumes an error from a cleanup operation.
// Errors during cleanup (unlock, close) are not actionable.
func handleCleanupError(_ error) {}
