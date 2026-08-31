package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const pathDotDot = ".."

// Resolve joins rel onto skillRoot and returns an absolute path that stays
// inside the skill root. Any ".." path component is rejected before Clean.
func Resolve(skillRoot, rel string) (string, error) {
	if rel == "" {
		return "", ErrEmptyPath
	}

	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: absolute path", ErrPathEscape)
	}

	if hasDotDot(rel) {
		return "", fmt.Errorf("%w: %s", ErrPathEscape, rel)
	}

	absRoot, err := filepath.Abs(skillRoot)
	if err != nil {
		return "", fmt.Errorf("skill root: %w", err)
	}

	joined := filepath.Join(absRoot, filepath.Clean(rel))

	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("skill path: %w", err)
	}

	if !insideRoot(absRoot, absJoined) {
		return "", fmt.Errorf("%w: %s", ErrPathEscape, rel)
	}

	return absJoined, nil
}

// ReadResource resolves rel against skillRoot and reads that one file.
// It does not follow links inside the file.
func ReadResource(skillRoot, rel string) ([]byte, error) {
	path, err := Resolve(skillRoot, rel)
	if err != nil {
		return nil, err
	}

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("read %s: %w", rel, readErr)
	}

	return data, nil
}

func hasDotDot(rel string) bool {
	normalized := filepath.ToSlash(rel)
	for part := range strings.SplitSeq(normalized, "/") {
		if part == pathDotDot {
			return true
		}
	}

	return false
}

func insideRoot(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}

	return rel != pathDotDot && !strings.HasPrefix(rel, pathDotDot+string(os.PathSeparator))
}
