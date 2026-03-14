// Package pathutil provides shared utilities for file path operations.
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandHome expands a leading `~` or `~/` in path to the current user's
// home directory. Paths that do not start with `~` are returned unchanged.
// The `~user` form is not supported and is returned as-is.
func ExpandHome(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}

		return home, nil
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}

		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}
