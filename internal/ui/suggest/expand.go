package suggest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/dmytrogajewski/spin/internal/commands"
	"github.com/dmytrogajewski/spin/internal/skills"
)

// Result is a submitted prompt after slash/skill/@ expansion.
type Result struct {
	Prompt    string
	Command   string
	Args      []string
	IsCommand bool
}

// Expand turns a submitted line into a command or an enriched prompt.
func Expand(line, workDir string, catalog skills.Catalog) Result {
	trimmed := strings.TrimSpace(line)
	if res, ok := expandSlash(trimmed, line, workDir, catalog); ok {
		return res
	}

	return Result{Prompt: attachMentions(trimmed, workDir)}
}

func expandSlash(trimmed, raw, workDir string, catalog skills.Catalog) (Result, bool) {
	cmd, args, isCmd := commands.ParseCommand(trimmed)
	if !isCmd {
		return Result{}, false
	}

	if _, exists := commands.GetCommand(cmd); exists {
		return Result{IsCommand: true, Command: cmd, Args: args}, true
	}

	name := strings.TrimPrefix(cmd, "/")

	act, err := skills.Activate(catalog, name)
	if err != nil {
		return Result{IsCommand: true, Command: cmd, Args: args}, true
	}

	rest := remainderAfterFirst(raw)
	body := formatSkill(act, rest)

	return Result{Prompt: attachMentions(body, workDir)}, true
}

func remainderAfterFirst(raw string) string {
	trimmed := strings.TrimSpace(raw)

	idx := strings.IndexFunc(trimmed, unicode.IsSpace)
	if idx < 0 {
		return ""
	}

	return strings.TrimSpace(trimmed[idx:])
}

func formatSkill(act skills.Activation, rest string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<skill name=%q source=%q>\n", act.Name, act.Source)
	b.WriteString(act.Body)

	if !strings.HasSuffix(act.Body, "\n") {
		b.WriteByte('\n')
	}

	b.WriteString("</skill>\n")

	if rest != "" {
		b.WriteByte('\n')
		b.WriteString(rest)
	}

	return b.String()
}

func attachMentions(prompt, workDir string) string {
	if workDir == "" || prompt == "" {
		return prompt
	}

	paths := mentionPaths(prompt)
	blocks := make([]string, 0, len(paths))

	for _, rel := range paths {
		block, ok := readAttach(workDir, rel)
		if ok {
			blocks = append(blocks, block)
		}
	}

	if len(blocks) == 0 {
		return prompt
	}

	return strings.Join(blocks, "\n") + "\n\n" + prompt
}

func mentionPaths(prompt string) []string {
	fields := strings.Fields(prompt)
	seen := make(map[string]struct{})
	out := make([]string, 0, len(fields))

	for _, field := range fields {
		rel, ok := mentionRel(field)
		if !ok {
			continue
		}

		if _, dup := seen[rel]; dup {
			continue
		}

		seen[rel] = struct{}{}
		out = append(out, rel)
	}

	return out
}

func mentionRel(field string) (string, bool) {
	if !strings.HasPrefix(field, "@") {
		return "", false
	}

	rel := strings.TrimPrefix(field, "@")
	rel = strings.TrimRight(rel, ",.;:)")

	if rel == "" || strings.Contains(rel, "..") {
		return "", false
	}

	return strings.ReplaceAll(rel, "\\", "/"), true
}

func readAttach(workDir, rel string) (string, bool) {
	abs, ok := containedFile(workDir, rel)
	if !ok {
		return "", false
	}

	info, err := os.Stat(abs)
	if err != nil || !info.Mode().IsRegular() {
		return skipNote(rel, "not a file"), true
	}

	if info.Size() > MaxAttachBytes {
		return skipNote(rel, "too large"), true
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return skipNote(rel, "unreadable"), true
	}

	if bytes.ContainsRune(data, 0) {
		return skipNote(rel, "binary"), true
	}

	var b strings.Builder
	fmt.Fprintf(&b, "<attached path=%q>\n", rel)
	b.Write(data)

	if len(data) == 0 || data[len(data)-1] != '\n' {
		b.WriteByte('\n')
	}

	b.WriteString("</attached>")

	return b.String(), true
}

func skipNote(rel, reason string) string {
	return fmt.Sprintf("<attached path=%q skipped=%q />", rel, reason)
}

func containedFile(workDir, rel string) (string, bool) {
	if workDir == "" || rel == "" || filepath.IsAbs(rel) {
		return "", false
	}

	root, err := filepath.Abs(workDir)
	if err != nil {
		return "", false
	}

	joined := filepath.Join(root, filepath.FromSlash(rel))

	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", false
	}

	relToRoot, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(relToRoot, "..") {
		return "", false
	}

	return abs, true
}
