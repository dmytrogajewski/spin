package session

import (
	"github.com/dmytrogajewski/spin/pkg/storage"
)

// Storage is a type alias for the generic store with Session type.
// This allows the session package to use the unified storage without
// defining its own interface.
type Storage = storage.Store[Session]

// NewFileStorage creates file-based session storage.
func NewFileStorage(baseDir string) (*storage.FileStore[Session], error) {
	return storage.NewFileStore[Session](storage.FileStoreConfig{
		BaseDir: baseDir,
		Suffix:  ".json",
	})
}
