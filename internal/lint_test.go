package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoTypesPackage ensures we never have a package named "types", "common", "util", etc.
// These are anti-patterns per Go Code Review Comments.
// Packages should be named after what they do, not what they contain.
//
// See: https://go.dev/doc/effective_go#package-names
// See: https://github.com/golang/go/wiki/CodeReviewComments#package-names
func TestNoTypesPackage(t *testing.T) {
	t.Parallel()

	prohibitedNames := []string{
		"types",
		"common",
		"util",
		"utils",
		"helpers",
		"lib",
		"misc",
		"base",
	}

	internalDir := "."

	err := filepath.Walk(internalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		// Skip hidden directories and vendor.
		baseName := filepath.Base(path)
		if strings.HasPrefix(baseName, ".") || baseName == "vendor" {
			return filepath.SkipDir
		}

		// Check if directory name is prohibited.
		for _, prohibited := range prohibitedNames {
			if baseName == prohibited {
				t.Errorf("Found prohibited package name: %q at %q\n"+
					"Packages should be named after what they DO, not what they ARE.\n"+
					"Examples:\n"+
					"  - Instead of 'types', use specific names like 'toolparams', 'messages', etc.\n"+
					"  - Instead of 'util', use 'encoding', 'validation', etc.\n"+
					"  - Instead of 'helpers', use the domain name like 'stringutil', 'pathutil', etc.\n"+
					"See: https://go.dev/doc/effective_go#package-names",
					prohibited, path)
			}
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

	prohibitedFileNames := []string{
		"types.go",
		"common.go",
		"util.go",
		"utils.go",
		"helpers.go",
		"base.go",
	}

	internalDir := "."

	err := filepath.Walk(internalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Only check .go files (not _test.go).
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		baseName := filepath.Base(path)
		for _, prohibited := range prohibitedFileNames {
			if baseName == prohibited {
				t.Errorf("Found prohibited file name: %q at %q\n"+
					"Files should be named after their primary type or functionality.\n"+
					"Examples:\n"+
					"  - Instead of 'types.go', use 'message.go', 'request.go', etc.\n"+
					"  - Instead of 'util.go', use 'encoding.go', 'validation.go', etc.\n"+
					"See: https://go.dev/doc/effective_go#package-names",
					prohibited, path)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk internal directory: %v", err)
	}
}
