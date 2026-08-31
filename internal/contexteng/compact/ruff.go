package compact

import (
	"strings"
)

func compactRuff(stdout []byte) []byte {
	return encodeLines(renderGrouped(groupRuff(decodeLines(stdout))))
}

func groupRuff(lines []string) map[string][]string {
	groups := make(map[string][]string)

	for _, line := range lines {
		file, loc, rule, ok := parseRuff(line)
		if !ok {
			continue
		}

		groups[rule] = append(groups[rule], file+":"+loc)
	}

	return groups
}

const ruffColonParts = 4

func parseRuff(line string) (file, loc, rule string, ok bool) {
	parts := strings.SplitN(line, ":", ruffColonParts)
	if len(parts) < ruffColonParts {
		return "", "", "", false
	}

	rest := strings.TrimSpace(parts[3])
	fields := strings.Fields(rest)

	if len(fields) == 0 {
		return "", "", "", false
	}

	return parts[0], parts[1], fields[0], true
}
