package pathx

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadFileWithLimit reads a file up to maxBytes bytes.
// Returns (content, truncated, error). If maxBytes <= 0, the entire file
// is read without limit.
func ReadFileWithLimit(path string, maxBytes int64) (content string, truncated bool, err error) {
	if maxBytes <= 0 {
		var data []byte

		data, err = os.ReadFile(path)
		if err != nil {
			return "", false, fmt.Errorf("read file: %w", err)
		}

		return string(data), false, nil
	}

	var info os.FileInfo

	info, err = os.Stat(path)
	if err != nil {
		return "", false, fmt.Errorf("stat file: %w", err)
	}

	truncated = info.Size() > maxBytes
	readSize := info.Size()

	if truncated {
		readSize = maxBytes
	}

	var file *os.File

	file, err = os.Open(path)
	if err != nil {
		return "", false, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	buf := make([]byte, readSize)

	var bytesRead int

	bytesRead, err = io.ReadFull(file, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", false, fmt.Errorf("read file: %w", err)
	}

	return string(buf[:bytesRead]), truncated, nil
}

// ReadLastLines reads a file and returns the last n lines joined by newline.
// If the file has fewer than n lines, all lines are returned.
// Empty trailing lines from a trailing newline are excluded.
func ReadLastLines(path string, n int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var lines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return "", fmt.Errorf("read file: %w", scanErr)
	}

	if len(lines) == 0 {
		return "", nil
	}

	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return strings.Join(lines, "\n"), nil
}
