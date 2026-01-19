package session

import (
	"github.com/dmytrogajewski/spin/internal/storage"
)

// Storage is a type alias for the generic store with Session type.
// This allows the session package to use the unified storage without
// defining its own interface.
type Storage = storage.Store[Session]

// NewFileStorage creates file-based session storage.
func NewFileStorage(baseDir string) (Storage, error) {
	return storage.NewFileStore[Session](storage.FileStoreConfig{
		BaseDir: baseDir,
		Suffix:  ".json",
	})
}
