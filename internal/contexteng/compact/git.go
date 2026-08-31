package compact

import (
	"strings"
)

const (
	shortHashLen     = 7
	porcelainMinLen  = 4
	diffGitMinFields = 4
	onelineMinFields = 2
	confirmOK        = "ok"
	stateUntrack     = "untracked"
	stateStaged      = "staged"
	stateMod         = "modified"
	stateDeleted     = "deleted"
	stateAdded       = "added"
	stateRename      = "renamed"
	stateMerge       = "unmerged"
)

func compactGitStatus(stdout []byte) []byte {
	lines := decodeLines(stdout)
	if porcelainStatus(lines) {
		return encodeLines(renderGrouped(parsePorcelain(lines)))
	}

	return encodeLines(renderGrouped(parseHumanStatus(lines)))
}

func porcelainStatus(lines []string) bool {
	for _, line := range lines {
		if len(line) >= porcelainMinLen && isPorcelainXY(line[:2]) && line[2] == ' ' {
			return true
		}
	}

	return false
}

func isPorcelainXY(pair string) bool {
	if pair == "??" || pair == "!!" {
		return true
	}

	return gitStatusCode(pair[0]) && gitStatusCode(pair[1])
}

func gitStatusCode(code byte) bool {
	switch code {
	case ' ', 'M', 'A', 'D', 'R', 'C', 'U', 'T', '!', '?':
		return true
	default:
		return false
	}
}

func parsePorcelain(lines []string) map[string][]string {
	groups := make(map[string][]string)

	for _, line := range lines {
		if len(line) < porcelainMinLen {
			continue
		}

		path := strings.TrimSpace(line[3:])
		if _, newPath, found := strings.Cut(path, " -> "); found {
			path = newPath
		}

		state := porcelainState(line[:2])
		groups[state] = append(groups[state], path)
	}

	return groups
}

func porcelainState(pair string) string {
	switch {
	case pair == "??":
		return stateUntrack
	case pair[0] == 'U' || pair[1] == 'U' || pair == "AA" || pair == "DD":
		return stateMerge
	case pair[0] == 'A':
		return stateAdded
	case pair[0] == 'D' || pair[1] == 'D':
		return stateDeleted
	case pair[0] == 'R':
		return stateRename
	case pair[0] != ' ':
		return stateStaged
	default:
		return stateMod
	}
}

func parseHumanStatus(lines []string) map[string][]string {
	state := ""
	groups := make(map[string][]string)

	for _, line := range lines {
		next, changed := humanState(line)
		if changed {
			state = next

			continue
		}

		appendHumanPath(groups, state, line)
	}

	return groups
}

func humanState(line string) (string, bool) {
	switch {
	case strings.Contains(line, "Untracked files"):
		return stateUntrack, true
	case strings.Contains(line, "Changes to be committed"):
		return stateStaged, true
	case strings.Contains(line, "Changes not staged"):
		return stateMod, true
	case strings.Contains(line, "Unmerged"):
		return stateMerge, true
	default:
		return "", false
	}
}

func appendHumanPath(groups map[string][]string, state, line string) {
	if state == "" {
		return
	}

	trim := strings.TrimSpace(line)
	if trim == "" || strings.HasPrefix(trim, "(") {
		return
	}

	if state != stateUntrack {
		_, path, found := strings.Cut(trim, ":")
		if !found {
			return
		}

		path = strings.TrimSpace(path)
		if path != "" {
			groups[state] = append(groups[state], path)
		}

		return
	}

	groups[state] = append(groups[state], trim)
}

func compactGitDiff(stdout []byte) []byte {
	return encodeLines(reduceDiff(decodeLines(stdout)))
}

func reduceDiff(lines []string) []string {
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		kept, ok := diffLine(line)
		if ok {
			out = append(out, kept)
		}
	}

	return out
}

func diffLine(line string) (string, bool) {
	switch {
	case strings.HasPrefix(line, "diff --git "):
		return diffPath(line), true
	case skipDiffHeader(line):
		return "", false
	case strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-"):
		return line, true
	default:
		return "", false
	}
}

func skipDiffHeader(line string) bool {
	prefixes := []string{
		"index ", "--- ", "+++ ", "old mode ", "new mode ",
		"similarity ", "rename ", "@@ ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	return false
}

func diffPath(line string) string {
	fields := strings.Fields(line)
	if len(fields) < diffGitMinFields {
		return strings.TrimPrefix(line, "diff --git ")
	}

	return strings.TrimPrefix(fields[3], "b/")
}

func compactGitLog(stdout []byte) []byte {
	return encodeLines(reduceLog(decodeLines(stdout)))
}

func reduceLog(lines []string) []string {
	state := logState{}

	for _, line := range lines {
		state.consume(line)
	}

	state.flush()

	return state.out
}

type logState struct {
	out     []string
	hash    string
	author  string
	subject string
}

func (state *logState) flush() {
	if state.hash == "" {
		return
	}

	state.out = append(state.out, joinLog(state.hash, state.author, state.subject))
	state.hash, state.author, state.subject = "", "", ""
}

func (state *logState) consume(line string) {
	switch {
	case strings.HasPrefix(line, "commit "):
		state.flush()
		state.hash = shortHash(strings.TrimPrefix(line, "commit "))
	case strings.HasPrefix(line, "Author:"):
		state.author = authorName(strings.TrimSpace(strings.TrimPrefix(line, "Author:")))
	case strings.HasPrefix(line, "Date:"):
	case strings.HasPrefix(line, "    ") && state.subject == "" && state.hash != "":
		state.subject = strings.TrimSpace(line)
	case state.hash == "" && onelineLog(line):
		state.out = append(state.out, line)
	default:
	}
}

func shortHash(raw string) string {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return ""
	}

	value := fields[0]
	if len(value) > shortHashLen {
		return value[:shortHashLen]
	}

	return value
}

func authorName(raw string) string {
	name, _, found := strings.Cut(raw, " <")
	if found {
		return strings.TrimSpace(name)
	}

	return strings.TrimSpace(raw)
}

func joinLog(hash, author, subject string) string {
	parts := []string{hash}
	if author != "" {
		parts = append(parts, author)
	}

	if subject != "" {
		parts = append(parts, subject)
	}

	return strings.Join(parts, " ")
}

func onelineLog(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < onelineMinFields {
		return false
	}

	return isHex(fields[0]) && len(fields[0]) >= shortHashLen
}

func isHex(text string) bool {
	if text == "" {
		return false
	}

	for _, runeValue := range text {
		hexDigit := (runeValue >= '0' && runeValue <= '9') ||
			(runeValue >= 'a' && runeValue <= 'f') ||
			(runeValue >= 'A' && runeValue <= 'F')
		if !hexDigit {
			return false
		}
	}

	return true
}

func compactGitConfirm(stdout []byte) []byte {
	for _, line := range decodeLines(stdout) {
		if strings.TrimSpace(line) == "" {
			continue
		}

		return encodeLines([]string{strings.TrimSpace(line)})
	}

	return encodeLines([]string{confirmOK})
}
