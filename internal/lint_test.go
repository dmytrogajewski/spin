package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// prohibitedDirNames lists package names that violate Go naming conventions.
var prohibitedDirNames = []string{
	"types",
	"common",
	"util",
	"utils",
	"helpers",
	"lib",
	"misc",
	"base",
}

// prohibitedFileNames lists file names that violate Go naming conventions.
var prohibitedFileNames = []string{
	"types.go",
	"common.go",
	"util.go",
	"utils.go",
	"helpers.go",
	"base.go",
}

// isProhibited checks whether a name appears in a prohibited list.
func isProhibited(name string, prohibited []string) bool {
	for _, p := range prohibited {
		if name == p {
			return true
		}
	}

	return false
}

// isSkippableDir returns true for hidden directories and vendor.
func isSkippableDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor"
}

// TestNoTypesPackage ensures we never have a package named "types", "common", "util", etc.
// These are anti-patterns per Go Code Review Comments.
// Packages should be named after what they do, not what they contain.
//
// See: https://go.dev/doc/effective_go#package-names
// See: https://github.com/golang/go/wiki/CodeReviewComments#package-names
func TestNoTypesPackage(t *testing.T) {
	t.Parallel()

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		baseName := filepath.Base(path)
		if isSkippableDir(baseName) {
			return filepath.SkipDir
		}

		if isProhibited(baseName, prohibitedDirNames) {
			t.Errorf("Found prohibited package name: %q at %q\n"+
				"Packages should be named after what they DO, not what they ARE.\n"+
				"See: https://go.dev/doc/effective_go#package-names",
				baseName, path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk internal directory: %v", err)
	}
}

// TestNoTypesGoFile ensures we never have a file named "types.go" in the root of a package.
// This is another anti-pattern - files should be named after their primary type or functionality.
func TestNoTypesGoFile(t *testing.T) {
	t.Parallel()

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !isGoSourceFile(path) {
			return nil
		}

		baseName := filepath.Base(path)
		if isProhibited(baseName, prohibitedFileNames) {
			t.Errorf("Found prohibited file name: %q at %q\n"+
				"Files should be named after their primary type or functionality.\n"+
				"See: https://go.dev/doc/effective_go#package-names",
				baseName, path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk internal directory: %v", err)
	}
}

// isGoSourceFile returns true for .go files that are not test files.
func isGoSourceFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}
