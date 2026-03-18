package session

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/storage"
)

const (
	// scannerMaxLineSize is the maximum line size for the JSONL scanner (10 MB).
	scannerMaxLineSize = 10 * 1024 * 1024
	// transcriptFilePerm is the file permission for transcript files.
	transcriptFilePerm = os.FileMode(0o600)
)

// ErrTranscriptWriterClosed is returned when Append is called after Close.
var ErrTranscriptWriterClosed = errors.New("transcript writer is closed")

// TranscriptWriter provides append-only JSONL persistence for conversation messages.
// Each message is serialized as a single JSON line and appended under advisory file lock.
// Thread-safe via [sync.Mutex].
type TranscriptWriter struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	count  int
	closed bool
}

// NewTranscriptWriter creates or opens a JSONL transcript file at the given path.
func NewTranscriptWriter(path string) (*TranscriptWriter, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, transcriptFilePerm)
	if err != nil {
		return nil, fmt.Errorf("open transcript file: %w", err)
	}

	return &TranscriptWriter{
		path: path,
		file: file,
	}, nil
}

// Append serializes msg as a single JSON line and appends it under exclusive file lock.
// The context controls the lock acquisition timeout.
func (w *TranscriptWriter) Append(ctx context.Context, msg message.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrTranscriptWriterClosed
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// Append newline delimiter.
	data = append(data, '\n')

	// Acquire exclusive lock for the append.
	fd := storage.SafeFlockFd(w.file.Fd())
	if lockErr := storage.FlockExclusiveWithContext(ctx, fd); lockErr != nil {
		return fmt.Errorf("lock transcript file: %w", lockErr)
	}

	defer func() { _ = storage.FlockUnlock(fd) }()

	if _, writeErr := w.file.Write(data); writeErr != nil {
		return fmt.Errorf("write transcript line: %w", writeErr)
	}

	w.count++

	return nil
}

// ReadAll reads all valid messages from the transcript file.
// Corrupted or empty lines are silently skipped.
// The context controls the lock acquisition timeout.
func (w *TranscriptWriter) ReadAll(ctx context.Context) ([]message.Message, error) {
	// Open a separate file handle for reading.
	file, err := os.Open(w.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("open transcript for reading: %w", err)
	}
	defer file.Close()

	// Acquire shared lock for reading.
	fd := storage.SafeFlockFd(file.Fd())
	if lockErr := storage.FlockSharedWithContext(ctx, fd); lockErr != nil {
		return nil, fmt.Errorf("shared lock transcript file: %w", lockErr)
	}

	defer func() { _ = storage.FlockUnlock(fd) }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), scannerMaxLineSize)

	var msgs []message.Message

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg message.Message

		if jsonErr := json.Unmarshal(line, &msg); jsonErr != nil {
			// Skip corrupted lines gracefully.
			continue
		}

		msgs = append(msgs, msg)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return msgs, fmt.Errorf("scan transcript file: %w", scanErr)
	}

	return msgs, nil
}

// Count returns the number of messages appended in this writer's lifetime.
func (w *TranscriptWriter) Count() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.count
}

// Close closes the underlying file handle. Safe to call multiple times.
func (w *TranscriptWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true

	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close transcript file: %w", err)
		}
	}

	return nil
}
