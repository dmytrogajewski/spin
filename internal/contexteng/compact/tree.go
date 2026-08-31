package compact

import (
	"slices"
	"strings"
)

const treeIndent = "  "

func filterTree(_ string, stdout, stderr []byte) (compactedStdout, compactedStderr []byte, err error) {
	return compactTree(stdout), stderr, nil
}

func compactTree(stdout []byte) []byte {
	paths := collectPaths(stdout)
	if len(paths) == 0 {
		return nil
	}

	root := buildTree(paths)

	return encodeLines(renderTree(root))
}

type treeEntry struct {
	kids map[string]*treeEntry
	file bool
}

func newEntry() *treeEntry {
	return &treeEntry{kids: make(map[string]*treeEntry)}
}

func collectPaths(stdout []byte) []string {
	lines := decodeLines(stdout)
	if looksLikeTree(lines) {
		return pathsFromTree(lines)
	}

	return pathsFromListing(lines)
}

func looksLikeTree(lines []string) bool {
	for _, line := range lines {
		if strings.Contains(line, "── ") {
			return true
		}
	}

	return false
}

func pathsFromListing(lines []string) []string {
	paths := make([]string, 0, len(lines))

	for _, line := range lines {
		path, ok := listingPath(line)
		if !ok {
			continue
		}

		paths = append(paths, path)
	}

	return paths
}

func listingPath(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed == "." || strings.HasPrefix(trimmed, "total ") {
		return "", false
	}

	if isUnixMode(trimmed) {
		return modeListingPath(trimmed)
	}

	trimmed = strings.TrimPrefix(trimmed, "./")
	if trimmed == "" || trimmed == "." {
		return "", false
	}

	return trimmed, true
}

const (
	unixModeWidth   = 10
	lsLongMinFields = 8
)

func isUnixMode(line string) bool {
	if len(line) < unixModeWidth {
		return false
	}

	switch line[0] {
	case '-', 'd', 'l', 'b', 'c', 'p', 's':
		return strings.ContainsAny(line[:unixModeWidth], "rwx-")
	default:
		return false
	}
}

func modeListingPath(line string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) < lsLongMinFields {
		return "", false
	}

	name := fields[len(fields)-1]
	if fields[0][0] == 'd' && !strings.HasSuffix(name, "/") {
		name += "/"
	}

	return name, true
}

func pathsFromTree(lines []string) []string {
	var (
		stack []string
		paths []string
	)

	for _, line := range lines {
		depth, name, ok := parseTreeLine(line)
		if !ok {
			continue
		}

		if depth < len(stack) {
			stack = stack[:depth]
		}

		path := name
		if len(stack) > 0 {
			path = strings.Join(stack, "/") + "/" + name
		}

		paths = append(paths, path)
		stack = append(stack, name)
	}

	return paths
}

func parseTreeLine(line string) (depth int, name string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed == "." {
		return 0, "", false
	}

	const marker = "── "

	prefix, rest, found := strings.Cut(line, marker)
	if !found {
		if strings.ContainsAny(line, "│├└─") {
			return 0, "", false
		}

		return 0, trimmed, true
	}

	name = strings.TrimSpace(rest)
	if name == "" {
		return 0, "", false
	}

	return treeDepth(prefix), name, true
}

func treeDepth(prefix string) int {
	width := 0

	for _, runeValue := range prefix {
		switch runeValue {
		case '│', '├', '└', ' ':
			width++
		default:
		}
	}

	const unit = 4

	return width / unit
}

func buildTree(paths []string) *treeEntry {
	root := newEntry()
	for _, path := range paths {
		addPath(root, path)
	}

	return root
}

func addPath(root *treeEntry, path string) {
	dirOnly := strings.HasSuffix(path, "/")
	path = strings.Trim(strings.TrimPrefix(path, "./"), "/")

	if path == "" {
		return
	}

	node := root
	parts := strings.Split(path, "/")

	for idx, part := range parts {
		child := node.kids[part]
		if child == nil {
			child = newEntry()
			node.kids[part] = child
		}

		node = child
		if idx == len(parts)-1 && !dirOnly {
			node.file = true
		}
	}
}

func (entry *treeEntry) fileCount() int {
	if entry.file && len(entry.kids) == 0 {
		return 1
	}

	total := 0
	for _, child := range entry.kids {
		total += child.fileCount()
	}

	return total
}

func renderTree(root *treeEntry) []string {
	kids := renderKids(root, "")
	lines := make([]string, 0, 1+len(kids))
	lines = append(lines, ". ("+countLabel(root.fileCount())+")")
	lines = append(lines, kids...)

	return lines
}

func renderKids(node *treeEntry, indent string) []string {
	names := make([]string, 0, len(node.kids))
	for name := range node.kids {
		names = append(names, name)
	}

	slices.Sort(names)

	lines := make([]string, 0, len(names))

	for _, name := range names {
		child := node.kids[name]
		if len(child.kids) > 0 {
			lines = append(lines, indent+name+"/ ("+countLabel(child.fileCount())+")")
			lines = append(lines, renderKids(child, indent+treeIndent)...)

			continue
		}

		lines = append(lines, indent+name)
	}

	return lines
}
