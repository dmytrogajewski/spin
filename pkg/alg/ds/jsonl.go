// Package ds provides generic data structures for use across the codebase.
package ds

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

// jsonlFilePerm is the default file permission for JSONL files.
const jsonlFilePerm = os.FileMode(0o600)

// scannerMaxLineSize is the maximum JSONL line size (10 MB).
const scannerMaxLineSize = 10 * 1024 * 1024

// ErrWriterClosed is returned when [JSONLWriter.Append] is called after Close.
var ErrWriterClosed = errors.New("jsonl writer is closed")

// JSONLWriter provides append-only JSONL persistence for typed records.
// Each item is serialized as a single JSON line. Thread-safe via [sync.Mutex].
type JSONLWriter[Item any] struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	count  int
	closed bool
}

// NewJSONLWriter creates or opens a JSONL file at the given path.
func NewJSONLWriter[Item any](path string) (*JSONLWriter[Item], error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, jsonlFilePerm)
	if err != nil {
		return nil, fmt.Errorf("open jsonl file: %w", err)
	}

	return &JSONLWriter[Item]{
		path: path,
		file: file,
	}, nil
}

// Append serializes item as a single JSON line and appends it to the file.
func (w *JSONLWriter[Item]) Append(item Item) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWriterClosed
	}

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal jsonl item: %w", err)
	}

	data = append(data, '\n')

	if _, writeErr := w.file.Write(data); writeErr != nil {
		return fmt.Errorf("write jsonl line: %w", writeErr)
	}

	w.count++

	return nil
}

// ReadAll reads all valid items from the JSONL file.
// Corrupted or empty lines are silently skipped.
func (w *JSONLWriter[Item]) ReadAll() ([]Item, error) {
	file, err := os.Open(w.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("open jsonl for reading: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), scannerMaxLineSize)

	var items []Item

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var item Item

		if jsonErr := json.Unmarshal(line, &item); jsonErr != nil {
			continue // Skip corrupted lines.
		}

		items = append(items, item)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return items, fmt.Errorf("scan jsonl file: %w", scanErr)
	}

	return items, nil
}

// Count returns the number of items appended in this writer's lifetime.
func (w *JSONLWriter[Item]) Count() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.count
}

// Close closes the underlying file handle. Safe to call multiple times.
func (w *JSONLWriter[Item]) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true

	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close jsonl file: %w", err)
		}
	}

	return nil
}

// Path returns the file path of the JSONL writer.
func (w *JSONLWriter[Item]) Path() string {
	return w.path
}

// File returns the underlying file handle for use with file-locking.
// Returns nil if the writer is closed.
func (w *JSONLWriter[Item]) File() *os.File {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	return w.file
}
