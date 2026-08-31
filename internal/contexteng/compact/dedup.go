package compact

import (
	"strconv"
)

const timesMark = " ×"

func compactDedup(stdout []byte) []byte {
	return encodeLines(dedupLines(decodeLines(stdout)))
}

func dedupLines(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}

	out := make([]string, 0, len(lines))
	prev := lines[0]
	count := 1

	for _, line := range lines[1:] {
		if line == prev {
			count++

			continue
		}

		out = append(out, formatDup(prev, count))
		prev = line
		count = 1
	}

	return append(out, formatDup(prev, count))
}

func formatDup(line string, count int) string {
	if count <= 1 {
		return line
	}

	return line + timesMark + strconv.Itoa(count)
}
