package compact

import (
	"slices"
	"strconv"
	"strings"
)

func ignoreCmd(transform func([]byte) []byte) Filter {
	return func(_ string, stdout, stderr []byte) (compactedStdout, compactedStderr []byte, err error) {
		return transform(stdout), stderr, nil
	}
}

func encodeLines(lines []string) []byte {
	if len(lines) == 0 {
		return nil
	}

	return []byte(strings.Join(lines, "\n") + "\n")
}

func decodeLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")

	if text == "" {
		return nil
	}

	return strings.Split(text, "\n")
}

func countLabel(nFiles int) string {
	if nFiles == 1 {
		return "1 file"
	}

	return strconv.Itoa(nFiles) + " files"
}

func isDigits(text string) bool {
	if text == "" {
		return false
	}

	for _, runeValue := range text {
		if runeValue < '0' || runeValue > '9' {
			return false
		}
	}

	return true
}

func collapseBlank(lines []string) []string {
	out := make([]string, 0, len(lines))
	prevBlank := true

	for _, line := range lines {
		blank := strings.TrimSpace(line) == ""
		if blank {
			if prevBlank {
				continue
			}

			prevBlank = true

			out = append(out, "")

			continue
		}

		prevBlank = false

		out = append(out, line)
	}

	if len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}

	return out
}

func renderGrouped(groups map[string][]string) []string {
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	const groupOverhead = 2

	lines := make([]string, 0, len(groups)*groupOverhead)

	for _, key := range keys {
		items := slices.Clone(groups[key])
		slices.Sort(items)
		lines = append(lines, key+" ("+strconv.Itoa(len(items))+")")

		for _, item := range items {
			lines = append(lines, "  "+item)
		}
	}

	return lines
}
