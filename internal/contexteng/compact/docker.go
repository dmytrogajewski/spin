package compact

import (
	"slices"
	"strings"
)

var dockerEssential = []string{"NAMES", "IMAGE", "STATUS"}

func compactDockerPS(stdout []byte) []byte {
	lines := decodeLines(stdout)
	if len(lines) == 0 {
		return nil
	}

	return encodeLines(reduceDockerPS(lines))
}

func reduceDockerPS(lines []string) []string {
	header := lines[0]
	rows := make([]string, 0, len(lines)-1)

	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := make([]string, 0, len(dockerEssential))
		for _, name := range dockerEssential {
			parts = append(parts, headerField(header, line, name))
		}

		rows = append(rows, strings.Join(parts, " "))
	}

	slices.Sort(rows)

	out := make([]string, 0, 1+len(rows))
	out = append(out, strings.Join(dockerEssential, " "))
	out = append(out, rows...)

	return out
}

func headerField(header, line, name string) string {
	start := strings.Index(header, name)
	if start < 0 || start >= len(line) {
		return ""
	}

	end := nextHeaderStart(header, start+len(name))
	if end < 0 || end > len(line) {
		end = len(line)
	}

	return strings.TrimSpace(line[start:end])
}

func nextHeaderStart(header string, after int) int {
	idx := after
	for idx < len(header) && header[idx] == ' ' {
		idx++
	}

	if idx >= len(header) {
		return -1
	}

	return idx
}
