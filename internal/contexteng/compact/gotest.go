package compact

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	actionPass   = "pass"
	actionFail   = "fail"
	actionOutput = "output"
	failPrefix   = "FAIL "
	okPrefix     = "ok "
)

type goTestEvent struct {
	Action  string
	Package string
	Test    string
	Output  string
}

func compactGoTest(stdout []byte) []byte {
	lines := decodeLines(stdout)
	if !hasNDJSON(lines) {
		return encodeLines(failureLines(lines))
	}

	return encodeLines(summarizeGoTest(lines))
}

func hasNDJSON(lines []string) bool {
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "{") {
			return true
		}
	}

	return false
}

func summarizeGoTest(lines []string) []string {
	pass := 0
	fails := make([]string, 0)
	outputs := make(map[string][]string)
	seen := make(map[string]bool)

	for _, line := range lines {
		event, ok := parseGoEvent(line)
		if !ok {
			continue
		}

		pass, fails = applyGoEvent(event, pass, fails, outputs, seen)
	}

	out := []string{okPrefix + strconv.Itoa(pass)}
	for _, name := range fails {
		out = append(out, failPrefix+name)
		out = append(out, outputs[name]...)
	}

	return out
}

func parseGoEvent(line string) (event goTestEvent, ok bool) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return goTestEvent{}, false
	}

	return goTestEvent{
		Action:  jsonString(raw["Action"]),
		Package: jsonString(raw["Package"]),
		Test:    jsonString(raw["Test"]),
		Output:  jsonString(raw["Output"]),
	}, true
}

func jsonString(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}

	return text
}

func applyGoEvent(
	event goTestEvent,
	pass int,
	fails []string,
	outputs map[string][]string,
	seen map[string]bool,
) (nextPass int, nextFails []string) {
	nextPass, nextFails = pass, fails

	switch event.Action {
	case actionPass:
		if event.Test != "" {
			return nextPass + 1, nextFails
		}
	case actionFail:
		if event.Test != "" && !seen[event.Test] {
			seen[event.Test] = true
			nextFails = append(nextFails, event.Test)
		}
	case actionOutput:
		appendGoOutput(outputs, event)
	default:
	}

	return nextPass, nextFails
}

func appendGoOutput(outputs map[string][]string, event goTestEvent) {
	if event.Test == "" {
		return
	}

	text := strings.TrimRight(event.Output, "\n")
	trim := strings.TrimSpace(text)

	if text == "" || skipGoOutput(trim) {
		return
	}

	outputs[event.Test] = append(outputs[event.Test], text)
}

func skipGoOutput(trim string) bool {
	return strings.HasPrefix(trim, "=== RUN") ||
		strings.HasPrefix(trim, "--- PASS") ||
		strings.HasPrefix(trim, "--- FAIL")
}

func compactFailures(stdout []byte) []byte {
	return encodeLines(failureLines(decodeLines(stdout)))
}

func failureLines(lines []string) []string {
	out := make([]string, 0)

	for _, line := range lines {
		if isFailureLine(line) {
			out = append(out, line)
		}
	}

	return out
}

func isFailureLine(line string) bool {
	trim := strings.TrimSpace(line)
	if trim == "" {
		return false
	}

	lower := strings.ToLower(trim)

	switch {
	case strings.Contains(lower, "failed"),
		strings.Contains(lower, "failure"),
		strings.Contains(line, "FAIL"),
		strings.Contains(line, "Error"),
		strings.Contains(line, "×"),
		strings.Contains(line, "✘"),
		strings.HasPrefix(trim, "E "),
		strings.HasPrefix(trim, "E\t"):
		return true
	default:
		return false
	}
}
