package e2e

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestNoTODOsOrDeferredPatterns ensures no TODO/FIXME/XXX/HACK or deferred implementation patterns exist.
// This test fails if any of these patterns are found in the codebase:
// - TODO, FIXME, XXX, HACK comments
// - "for now", "not yet", "will be", "later" patterns that indicate deferred work.
// codeQualityPatterns returns the patterns to detect in code quality checks.
var codeQualityPatterns = []struct {
	name    string
	pattern *regexp.Regexp
	desc    string
}{
	{name: "TODO", pattern: regexp.MustCompile(`(?i)\bTODO\b`), desc: "TODO comments indicate incomplete work"},
	{name: "FIXME", pattern: regexp.MustCompile(`(?i)\bFIXME\b`), desc: "FIXME comments indicate known issues"},
	{name: "XXX", pattern: regexp.MustCompile(`(?i)\bXXX\b`), desc: "XXX comments indicate problematic code"},
	{name: "HACK", pattern: regexp.MustCompile(`(?i)\bHACK\b`), desc: "HACK comments indicate workarounds"},
	{name: "for now", pattern: regexp.MustCompile(`(?i)\bfor now\b`), desc: "\"for now\" indicates temporary implementation"},
	{name: "not yet", pattern: regexp.MustCompile(`(?i)\bnot yet\b`), desc: "\"not yet\" indicates deferred implementation"},
	{name: "will be", pattern: regexp.MustCompile(`(?i)\bwill be\b`), desc: "\"will be\" indicates future work"},
	{name: "later", pattern: regexp.MustCompile(`(?i)\blater\b`), desc: "\"later\" indicates deferred work"},
	{name: "temporary", pattern: regexp.MustCompile(`(?i)\btemporary\b`), desc: "\"temporary\" indicates non-permanent solution"},
	{name: "eventually", pattern: regexp.MustCompile(`(?i)\beventually\b`), desc: "\"eventually\" indicates deferred implementation"},
}

func TestNoTODOsOrDeferredPatterns(t *testing.T) {
	t.Parallel()

	workspaceRoot, err := getWorkspaceRoot()
	if err != nil {
		t.Fatalf("Failed to find workspace root: %v", err)
	}

	violations := scanForViolations(t, workspaceRoot)
	if len(violations) == 0 {
		return
	}

	reportViolations(t, violations)
}

// scanForViolations walks the source directories and collects all violations.
func scanForViolations(t *testing.T, workspaceRoot string) []violation {
	t.Helper()

	var violations []violation

	for _, dir := range []string{"internal", "cmd"} {
		fullPath := filepath.Join(workspaceRoot, dir)

		walkErr := filepath.Walk(fullPath, func(path string, _ os.FileInfo, walkFnErr error) error {
			if walkFnErr != nil {
				return walkFnErr
			}

			if !isCheckableGoFile(path) {
				return nil
			}

			return checkFile(t, path, codeQualityPatterns, &violations)
		})
		if walkErr != nil {
			t.Fatalf("Failed to walk %s: %v", fullPath, walkErr)
		}
	}

	return violations
}

// isCheckableGoFile returns true if the path is a non-test Go file outside vendor.
func isCheckableGoFile(path string) bool {
	if strings.Contains(path, "/vendor/") || strings.Contains(path, "/.git/") {
		return false
	}

	if !strings.HasSuffix(path, ".go") {
		return false
	}

	return !strings.HasSuffix(path, "_test.go")
}

// reportViolations prints all violations and fails the test.
func reportViolations(t *testing.T, violations []violation) {
	t.Helper()

	t.Errorf("Found %d code quality violations:\n", len(violations))

	byPattern := make(map[string][]violation)
	for _, v := range violations {
		byPattern[v.pattern] = append(byPattern[v.pattern], v)
	}

	for _, v := range violations {
		t.Errorf("  %s:%d: %s - %s", v.file, v.line, v.pattern, v.desc)
		t.Errorf("    %s", strings.TrimSpace(v.context))
	}

	printImplementationInstructions(os.Stdout, byPattern, violations)
	t.Fatalf("Code quality test failed: found %d violations", len(violations))
}

// getWorkspaceRoot finds the workspace root by looking for go.mod.
func getWorkspaceRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		_, statErr := os.Stat(filepath.Join(dir, "go.mod"))
		if statErr == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}

		dir = parent
	}
}

type violation struct {
	file    string
	line    int
	pattern string
	desc    string
	context string
}

func checkFile(_ *testing.T, filePath string, patterns []struct {
	name    string
	pattern *regexp.Regexp
	desc    string
}, violations *[]violation) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", filePath, err)
	}

	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		// Check both comment and code lines for pattern violations.
		for _, p := range patterns {
			if p.pattern.MatchString(line) {
				// Get context (previous and next line if available).
				lineContext := line
				if lineNum > 0 {
					lineContext = lines[lineNum-1] + "\n" + lineContext
				}

				if lineNum < len(lines)-1 {
					lineContext = lineContext + "\n" + lines[lineNum+1]
				}

				*violations = append(*violations, violation{
					file:    filePath,
					line:    lineNum + 1,
					pattern: p.name,
					desc:    p.desc,
					context: lineContext,
				})
			}
		}
	}

	return nil
}

// patternGuidance maps pattern names to (action, doNot) guidance strings.
var patternGuidance = map[string][2]string{
	"TODO":      {"Implement the incomplete work described in the TODO comment.", "Remove the TODO or change it to a different comment."},
	"FIXME":     {"Fix the known issue described in the FIXME comment.", "Remove the FIXME or ignore the issue."},
	"XXX":       {"Resolve the problematic code marked with XXX.", "Leave problematic code in place."},
	"HACK":      {"Replace the workaround with a proper implementation.", "Keep temporary workarounds."},
	"for now":   {"Replace the temporary implementation with a permanent solution.", "Change 'for now' to different wording - implement properly."},
	"not yet":   {"Implement the deferred functionality.", "Change 'not yet' to 'not implemented' - actually implement it."},
	"will be":   {"Implement the future work described.", "Change 'will be' to present tense - make it happen now."},
	"later":     {"Implement the deferred work now.", "Change 'later' to different wording - do it now."},
	"temporary": {"Replace the temporary solution with a permanent one.", "Keep temporary solutions."},
	"eventually": {"Implement the deferred functionality now.", "Change wording - implement the feature."},
}

// printImplementationInstructions writes actionable instructions to help AI agents fix violations.
func printImplementationInstructions(w io.Writer, _ map[string][]violation, allViolations []violation) {
	separator := strings.Repeat("=", 80)
	fmt.Fprintf(w, "\n%s\nIMPLEMENTATION INSTRUCTIONS FOR AI AGENTS\n%s\n\n", separator, separator)
	fmt.Fprintf(w, "Found %d code quality violations that require implementation.\n", len(allViolations))
	fmt.Fprintf(w, "DO NOT simply remove comments or change wording. IMPLEMENT the missing functionality.\n\n")

	printViolationsByFile(w, allViolations)
	printChecklist(w, separator)
}

// printViolationsByFile groups and prints violations organized by file.
func printViolationsByFile(w io.Writer, allViolations []violation) {
	byFile := make(map[string][]violation)
	for _, v := range allViolations {
		byFile[v.file] = append(byFile[v.file], v)
	}

	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}

	sort.Strings(files)

	fmt.Fprintf(w, "VIOLATIONS BY FILE:\n%s\n", strings.Repeat("-", 80))

	for _, file := range files {
		viols := byFile[file]
		fmt.Fprintf(w, "\n%s (%d violations):\n", file, len(viols))

		for _, v := range viols {
			printViolationGuidance(w, v)
		}
	}
}

// printViolationGuidance prints a single violation with its guidance.
func printViolationGuidance(w io.Writer, v violation) {
	fmt.Fprintf(w, "  Line %d [%s]: %s\n", v.line, v.pattern, v.desc)
	fmt.Fprintf(w, "    Context: %s\n", strings.TrimSpace(v.context))

	if guidance, ok := patternGuidance[v.pattern]; ok {
		fmt.Fprintf(w, "    ACTION: %s\n", guidance[0])
		fmt.Fprintf(w, "    DO NOT: %s\n", guidance[1])
	}

	fmt.Fprintf(w, "\n")
}

// printChecklist prints the implementation checklist footer.
func printChecklist(w io.Writer, separator string) {
	fmt.Fprintf(w, "\n%s\nIMPLEMENTATION CHECKLIST:\n%s\n\n", separator, separator)
	fmt.Fprintf(w, "For each violation:\n")
	fmt.Fprintf(w, "1. Read the violation context to understand what needs to be implemented\n")
	fmt.Fprintf(w, "2. Check related code and dependencies\n")
	fmt.Fprintf(w, "3. Implement the missing functionality completely\n")
	fmt.Fprintf(w, "4. Add tests to verify the implementation\n")
	fmt.Fprintf(w, "5. Remove the violation comment/pattern (it's now implemented)\n")
	fmt.Fprintf(w, "6. Run tests to ensure everything works\n")
	fmt.Fprintf(w, "7. Verify this test passes\n\n")
	fmt.Fprintf(w, "REMEMBER:\n- Implementation > Comments\n- Tests are required for new functionality\n")
	fmt.Fprintf(w, "- No shortcuts or temporary solutions\n- Complete the work, don't defer it\n\n")
	fmt.Fprintf(w, "%s\n\n", separator)
}
