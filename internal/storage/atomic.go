package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultFilePerm is the default file permission for atomic writes (owner read/write).
const DefaultFilePerm = os.FileMode(0o600)

// AtomicWriteFile writes data to path atomically using a staging-file + rename
// strategy. The file is first written to a staging file in the same
// directory, then renamed to the final path. This ensures that readers
// never see a partially-written file. The perm argument sets the file
// mode on the staging file before rename.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmpFile, err := os.CreateTemp(dir, ".atomic-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	// Write data to temp file.
	_, err = tmpFile.Write(data)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)

		return fmt.Errorf("write temp file: %w", err)
	}

	// Set permissions before rename.
	err = tmpFile.Chmod(perm)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)

		return fmt.Errorf("set file permissions: %w", err)
	}

	err = tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)

		return fmt.Errorf("close temp file: %w", err)
	}

	// Atomic rename.
	err = os.Rename(tmpPath, path)
	if err != nil {
		os.Remove(tmpPath)

		return fmt.Errorf("atomic rename: %w", err)
	}

	return nil
}
