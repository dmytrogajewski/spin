package pathx

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

// ErrNotFound is returned when [WalkUpFind] reaches the filesystem root
// without finding a matching directory.
var ErrNotFound = errors.New("walk up find: no match found")

// WalkUpFind walks up the directory tree starting from start, calling
// predicate on each directory. Returns the first directory where predicate
// returns true. Returns [ErrNotFound] if the filesystem root is reached
// without a match. Returns the context error if ctx is canceled.
func WalkUpFind(ctx context.Context, start string, predicate func(dir string) bool) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	for {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("walk up find: %w", ctxErr)
		}

		if predicate(dir) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}

		dir = parent
	}
}
