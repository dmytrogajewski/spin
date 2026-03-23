package tools

import "github.com/dmytrogajewski/spin/pkg/alg/pathx"

// FileTracker is an alias for [pathx.FileTracker].
type FileTracker = pathx.FileTracker

// NewFileTracker creates a new file modification tracker.
func NewFileTracker() *FileTracker {
	return pathx.NewFileTracker()
}
