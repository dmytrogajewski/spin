package filesearch

import "github.com/dmytrogajewski/spin/pkg/alg/pathx"

// IgnoreHandler is an alias for [pathx.IgnoreHandler].
type IgnoreHandler = pathx.IgnoreHandler

// NewIgnoreHandler creates an ignore handler for the given root directory.
func NewIgnoreHandler(rootDir string) (*IgnoreHandler, error) {
	return pathx.NewIgnoreHandler(rootDir)
}
