package filesearch

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Grep returns file:line:text hits for pattern under root. Fixtures only — no network.
func Grep(ctx context.Context, root, pattern string) (string, error) {
	if pattern == "" {
		return "", nil
	}

	files, err := NewScanner(root).ScanWithContext(ctx)
	if err != nil {
		return "", err
	}

	var hits []string

	for _, rel := range files {
		fileHits, grepErr := grepFile(filepath.Join(root, rel), filepath.ToSlash(rel), pattern)
		if grepErr != nil {
			continue
		}

		hits = append(hits, fileHits...)
	}

	return strings.Join(hits, "\n"), nil
}

func grepFile(abs, rel, pattern string) ([]string, error) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	if bytes.IndexByte(data, 0) >= 0 {
		return nil, nil
	}

	var hits []string

	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		if strings.Contains(line, pattern) {
			hits = append(hits, fmt.Sprintf("%s:%d:%s", rel, lineNo, line))
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return hits, fmt.Errorf("scan: %w", scanErr)
	}

	return hits, nil
}
