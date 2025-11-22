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
// - "for now", "not yet", "will be", "later" patterns that indicate deferred work
func TestNoTODOsOrDeferredPatterns(t *testing.T) {
	// Patterns to detect
	patterns := []struct {
		name    string
		pattern *regexp.Regexp
		desc    string
	}{
		{
			name:    "TODO",
			pattern: regexp.MustCompile(`(?i)\bTODO\b`),
			desc:    "TODO comments indicate incomplete work",
		},
		{
			name:    "FIXME",
			pattern: regexp.MustCompile(`(?i)\bFIXME\b`),
			desc:    "FIXME comments indicate known issues",
		},
		{
			name:    "XXX",
			pattern: regexp.MustCompile(`(?i)\bXXX\b`),
			desc:    "XXX comments indicate problematic code",
		},
		{
			name:    "HACK",
			pattern: regexp.MustCompile(`(?i)\bHACK\b`),
			desc:    "HACK comments indicate workarounds",
		},
		{
			name:    "for now",
			pattern: regexp.MustCompile(`(?i)\bfor now\b`),
			desc:    "\"for now\" indicates temporary implementation",
		},
		{
			name:    "not yet",
			pattern: regexp.MustCompile(`(?i)\bnot yet\b`),
			desc:    "\"not yet\" indicates deferred implementation",
		},
		{
			name:    "will be",
			pattern: regexp.MustCompile(`(?i)\bwill be\b`),
			desc:    "\"will be\" indicates future work",
		},
		{
			name:    "later",
			pattern: regexp.MustCompile(`(?i)\blater\b`),
			desc:    "\"later\" indicates deferred work",
		},
		{
			name:    "temporary",
			pattern: regexp.MustCompile(`(?i)\btemporary\b`),
			desc:    "\"temporary\" indicates non-permanent solution",
		},
		{
			name:    "eventually",
			pattern: regexp.MustCompile(`(?i)\beventually\b`),
			desc:    "\"eventually\" indicates deferred implementation",
		},
	}

	// Get workspace root
	workspaceRoot, err := getWorkspaceRoot()
	if err != nil {
		t.Fatalf("Failed to find workspace root: %v", err)
	}

	// Directories to scan (all Go code, excluding tests and vendor)
	dirs := []string{
		"internal",
		"cmd",
	}

	var violations []violation

	for _, dir := range dirs {
		fullPath := filepath.Join(workspaceRoot, dir)
		err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Skip vendor and test files
			if strings.Contains(path, "/vendor/") || strings.Contains(path, "/.git/") {
				return nil
			}
			// Only check .go files
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			// Skip test files (they may have examples)
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			return checkFile(t, path, patterns, &violations)
		})
		if err != nil {
			t.Fatalf("Failed to walk %s: %v", fullPath, err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("Found %d code quality violations:\n", len(violations))

		// Group violations by pattern type for better reporting
		byPattern := make(map[string][]violation)
		for _, v := range violations {
			byPattern[v.pattern] = append(byPattern[v.pattern], v)
		}

		// Print detailed violations
		for _, v := range violations {
			t.Errorf("  %s:%d: %s - %s", v.file, v.line, v.pattern, v.desc)
			t.Errorf("    %s", strings.TrimSpace(v.context))
		}

		// Print implementation instructions to stdout (for AI agents)
		printImplementationInstructions(os.Stdout, byPattern, violations)

		t.Fatalf("Code quality test failed: found %d violations", len(violations))
	}
}

// getWorkspaceRoot finds the workspace root by looking for go.mod
func getWorkspaceRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
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

func checkFile(t *testing.T, filePath string, patterns []struct {
	name    string
	pattern *regexp.Regexp
	desc    string
}, violations *[]violation) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		// Skip lines that are only comments with these patterns (they might be in documentation)
		// But catch them if they're in actual code comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			// Check comment lines
			for _, p := range patterns {
				if p.pattern.MatchString(line) {
					// Get context (previous and next line if available)
					context := line
					if lineNum > 0 {
						context = lines[lineNum-1] + "\n" + context
					}
					if lineNum < len(lines)-1 {
						context = context + "\n" + lines[lineNum+1]
					}

					*violations = append(*violations, violation{
						file:    filePath,
						line:    lineNum + 1,
						pattern: p.name,
						desc:    p.desc,
						context: context,
					})
				}
			}
		} else {
			// Check code lines (might have inline comments)
			for _, p := range patterns {
				if p.pattern.MatchString(line) {
					// Get context
					context := line
					if lineNum > 0 {
						context = lines[lineNum-1] + "\n" + context
					}
					if lineNum < len(lines)-1 {
						context = context + "\n" + lines[lineNum+1]
					}

					*violations = append(*violations, violation{
						file:    filePath,
						line:    lineNum + 1,
						pattern: p.name,
						desc:    p.desc,
						context: context,
					})
				}
			}
		}
	}

	return nil
}

// printImplementationInstructions writes actionable instructions to help AI agents fix violations.
func printImplementationInstructions(w io.Writer, byPattern map[string][]violation, allViolations []violation) {
	separator := strings.Repeat("=", 80)
	fmt.Fprintf(w, "\n%s\n", separator)
	fmt.Fprintf(w, "IMPLEMENTATION INSTRUCTIONS FOR AI AGENTS\n")
	fmt.Fprintf(w, "%s\n\n", separator)

	fmt.Fprintf(w, "Found %d code quality violations that require implementation.\n", len(allViolations))
	fmt.Fprintf(w, "DO NOT simply remove comments or change wording. IMPLEMENT the missing functionality.\n\n")

	// Group by file for easier navigation
	byFile := make(map[string][]violation)
	for _, v := range allViolations {
		byFile[v.file] = append(byFile[v.file], v)
	}

	// Sort files for consistent output
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	fmt.Fprintf(w, "VIOLATIONS BY FILE:\n")
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 80))

	for _, file := range files {
		viols := byFile[file]
		fmt.Fprintf(w, "\n%s (%d violations):\n", file, len(viols))

		for _, v := range viols {
			fmt.Fprintf(w, "  Line %d [%s]: %s\n", v.line, v.pattern, v.desc)
			fmt.Fprintf(w, "    Context: %s\n", strings.TrimSpace(v.context))

			// Provide specific guidance based on pattern type
			switch v.pattern {
			case "TODO":
				fmt.Fprintf(w, "    ACTION: Implement the incomplete work described in the TODO comment.\n")
				fmt.Fprintf(w, "    DO NOT: Remove the TODO or change it to a different comment.\n")
			case "FIXME":
				fmt.Fprintf(w, "    ACTION: Fix the known issue described in the FIXME comment.\n")
				fmt.Fprintf(w, "    DO NOT: Remove the FIXME or ignore the issue.\n")
			case "XXX":
				fmt.Fprintf(w, "    ACTION: Resolve the problematic code marked with XXX.\n")
				fmt.Fprintf(w, "    DO NOT: Leave problematic code in place.\n")
			case "HACK":
				fmt.Fprintf(w, "    ACTION: Replace the workaround with a proper implementation.\n")
				fmt.Fprintf(w, "    DO NOT: Keep temporary workarounds.\n")
			case "for now":
				fmt.Fprintf(w, "    ACTION: Replace the temporary implementation with a permanent solution.\n")
				fmt.Fprintf(w, "    DO NOT: Change 'for now' to different wording - implement properly.\n")
			case "not yet":
				fmt.Fprintf(w, "    ACTION: Implement the deferred functionality.\n")
				fmt.Fprintf(w, "    DO NOT: Change 'not yet' to 'not implemented' - actually implement it.\n")
			case "will be":
				fmt.Fprintf(w, "    ACTION: Implement the future work described.\n")
				fmt.Fprintf(w, "    DO NOT: Change 'will be' to present tense - make it happen now.\n")
			case "later":
				fmt.Fprintf(w, "    ACTION: Implement the deferred work now.\n")
				fmt.Fprintf(w, "    DO NOT: Change 'later' to different wording - do it now.\n")
			case "temporary":
				fmt.Fprintf(w, "    ACTION: Replace the temporary solution with a permanent one.\n")
				fmt.Fprintf(w, "    DO NOT: Keep temporary solutions.\n")
			case "eventually":
				fmt.Fprintf(w, "    ACTION: Implement the deferred functionality now.\n")
				fmt.Fprintf(w, "    DO NOT: Change wording - implement the feature.\n")
			}
			fmt.Fprintf(w, "\n")
		}
	}

	fmt.Fprintf(w, "\n%s\n", separator)
	fmt.Fprintf(w, "IMPLEMENTATION CHECKLIST:\n")
	fmt.Fprintf(w, "%s\n\n", separator)

	fmt.Fprintf(w, "For each violation:\n")
	fmt.Fprintf(w, "1. Read the violation context to understand what needs to be implemented\n")
	fmt.Fprintf(w, "2. Check related code and dependencies\n")
	fmt.Fprintf(w, "3. Implement the missing functionality completely\n")
	fmt.Fprintf(w, "4. Add tests to verify the implementation\n")
	fmt.Fprintf(w, "5. Remove the violation comment/pattern (it's now implemented)\n")
	fmt.Fprintf(w, "6. Run tests to ensure everything works\n")
	fmt.Fprintf(w, "7. Verify this test passes\n\n")

	fmt.Fprintf(w, "REMEMBER:\n")
	fmt.Fprintf(w, "- Implementation > Comments\n")
	fmt.Fprintf(w, "- Tests are required for new functionality\n")
	fmt.Fprintf(w, "- No shortcuts or temporary solutions\n")
	fmt.Fprintf(w, "- Complete the work, don't defer it\n\n")

	fmt.Fprintf(w, "%s\n\n", separator)
}
