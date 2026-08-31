package compact

import (
	"strings"

	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

const grepLineMax = 80

func compactGrep(stdout []byte) []byte {
	groups, order := groupGrep(decodeLines(stdout))

	return encodeLines(renderGrep(groups, order))
}

type grepHit struct {
	line string
	text string
}

const grepColonParts = 3

func groupGrep(lines []string) (groups map[string][]grepHit, order []string) {
	groups = make(map[string][]grepHit)
	order = make([]string, 0)

	for _, line := range lines {
		file, lineno, text, ok := parseGrepLine(line)
		if !ok {
			continue
		}

		if _, seen := groups[file]; !seen {
			order = append(order, file)
		}

		groups[file] = append(groups[file], grepHit{line: lineno, text: text})
	}

	return groups, order
}

func parseGrepLine(line string) (file, lineno, text string, ok bool) {
	parts := strings.SplitN(line, ":", grepColonParts)
	if len(parts) < grepColonParts || !isDigits(parts[1]) {
		return "", "", "", false
	}

	file, lineno, rest := parts[0], parts[1], parts[2]
	if col, body, found := strings.Cut(rest, ":"); found && isDigits(col) {
		return file, lineno, body, true
	}

	return file, lineno, rest, true
}

func renderGrep(groups map[string][]grepHit, order []string) []string {
	lines := make([]string, 0)

	for _, file := range order {
		lines = append(lines, file)

		for _, hit := range groups[file] {
			text := stringsx.TruncateWithEllipsis(hit.text, grepLineMax)
			lines = append(lines, "  "+hit.line+": "+text)
		}
	}

	return lines
}
