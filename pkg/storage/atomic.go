package storage

import (
	"context"
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
//
// The context is checked before each I/O step. If the context is canceled,
// any partially-written temp file is cleaned up and the target file remains
// untouched.
func AtomicWriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("atomic write %s: %w", filepath.Base(path), err)
	}

	dir := filepath.Dir(path)

	tmpFile, err := os.CreateTemp(dir, ".atomic-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	// On any failure, close the temp file (safe even if already closed)
	// and remove it so readers never see a partial write.
	success := false

	defer func() {
		if !success {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("atomic write %s: %w", filepath.Base(path), ctxErr)
	}

	if _, err = tmpFile.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err = tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("set file permissions: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("atomic write %s: %w", filepath.Base(path), ctxErr)
	}

	if err = os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("atomic rename: %w", err)
	}

	success = true

	return nil
}
