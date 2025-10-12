package pathutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestValidateRelativePath tests path validation
func TestValidateRelativePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		// Valid relative paths
		{
			name:    "valid simple path",
			path:    "src/main.go",
			wantErr: nil,
		},
		{
			name:    "valid current directory",
			path:    ".",
			wantErr: nil,
		},
		{
			name:    "valid path with dot prefix",
			path:    "./src/main.go",
			wantErr: nil,
		},
		{
			name:    "valid path with parent staying in workspace",
			path:    "src/../lib/util.go",
			wantErr: nil,
		},
		{
			name:    "valid path with dot component",
			path:    "src/./main.go",
			wantErr: nil,
		},
		{
			name:    "valid path with trailing slash",
			path:    "src/",
			wantErr: nil,
		},
		{
			name:    "valid path with double slashes",
			path:    "src//main.go",
			wantErr: nil,
		},
		{
			name:    "valid nested path",
			path:    "a/b/c/d/e/f/file.txt",
			wantErr: nil,
		},

		// Invalid absolute paths
		{
			name:    "invalid absolute unix path",
			path:    "/etc/passwd",
			wantErr: ErrAbsolutePath,
		},
		{
			name:    "invalid absolute home path",
			path:    "/home/user/file.txt",
			wantErr: ErrAbsolutePath,
		},
		{
			name:    "invalid root path",
			path:    "/",
			wantErr: ErrAbsolutePath,
		},

		// Invalid path traversal
		{
			name:    "invalid triple parent traversal",
			path:    "../../../etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "invalid hidden traversal",
			path:    "foo/../../../bar",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "invalid multiple parents",
			path:    "../../..",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "invalid traversal from subdirectory",
			path:    "src/../../..",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "invalid single parent",
			path:    "..",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "valid parent at end (resolves to current dir)",
			path:    "src/..",
			wantErr: nil,
		},

		// Edge cases
		{
			name:    "invalid empty path",
			path:    "",
			wantErr: ErrEmptyPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRelativePath(tt.path)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateRelativePath(%q) = %v, want %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

// TestNormalizePath tests path normalization
func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "simple path unchanged",
			path: "src/main.go",
			want: "src/main.go",
		},
		{
			name: "remove dot prefix",
			path: "./src/main.go",
			want: "src/main.go",
		},
		{
			name: "remove double slashes",
			path: "src//main.go",
			want: "src/main.go",
		},
		{
			name: "remove dot component",
			path: "src/./main.go",
			want: "src/main.go",
		},
		{
			name: "resolve parent",
			path: "src/../lib/util.go",
			want: "lib/util.go",
		},
		{
			name: "remove trailing slash",
			path: "src/",
			want: "src",
		},
		{
			name: "current directory",
			path: ".",
			want: ".",
		},
		{
			name: "complex mixed",
			path: "./src/./foo/../bar//baz",
			want: "src/bar/baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePath(tt.path)
			if got != tt.want {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestSafeJoin tests safe path joining
func TestSafeJoin(t *testing.T) {
	// Create temp directory for tests
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		root    string
		relPath string
		want    string
		wantErr error
	}{
		{
			name:    "valid simple join",
			root:    tmpDir,
			relPath: "src/main.go",
			want:    filepath.Join(tmpDir, "src/main.go"),
			wantErr: nil,
		},
		{
			name:    "valid current directory",
			root:    tmpDir,
			relPath: ".",
			want:    tmpDir,
			wantErr: nil,
		},
		{
			name:    "valid with normalization",
			root:    tmpDir,
			relPath: "./src/./main.go",
			want:    filepath.Join(tmpDir, "src/main.go"),
			wantErr: nil,
		},
		{
			name:    "valid with parent staying in workspace",
			root:    tmpDir,
			relPath: "src/../lib/util.go",
			want:    filepath.Join(tmpDir, "lib/util.go"),
			wantErr: nil,
		},
		{
			name:    "invalid path traversal escape",
			root:    tmpDir,
			relPath: "../../../etc/passwd",
			want:    "",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "invalid absolute path",
			root:    tmpDir,
			relPath: "/etc/passwd",
			want:    "",
			wantErr: ErrAbsolutePath,
		},
		{
			name:    "invalid empty path",
			root:    tmpDir,
			relPath: "",
			want:    "",
			wantErr: ErrEmptyPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeJoin(tt.root, tt.relPath)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("SafeJoin(%q, %q) error = %v, wantErr %v", tt.root, tt.relPath, err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("SafeJoin(%q, %q) = %q, want %q", tt.root, tt.relPath, got, tt.want)
			}
		})
	}
}

// TestSafeJoinWithSymlinks tests symlink handling
func TestSafeJoinWithSymlinks(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	outsideDir := t.TempDir()

	// Create a file inside workspace
	insideFile := filepath.Join(tmpDir, "inside.txt")
	if err := os.WriteFile(insideFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a file outside workspace
	outsideFile := filepath.Join(outsideDir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create symlink inside workspace pointing to file inside workspace
	symlinkInsideToInside := filepath.Join(tmpDir, "link_inside")
	if err := os.Symlink(insideFile, symlinkInsideToInside); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Create symlink inside workspace pointing to file outside workspace
	symlinkInsideToOutside := filepath.Join(tmpDir, "link_outside")
	if err := os.Symlink(outsideFile, symlinkInsideToOutside); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	tests := []struct {
		name    string
		root    string
		relPath string
		wantErr error
	}{
		{
			name:    "valid symlink inside workspace",
			root:    tmpDir,
			relPath: "link_inside",
			wantErr: nil,
		},
		{
			name:    "invalid symlink escaping workspace",
			root:    tmpDir,
			relPath: "link_outside",
			wantErr: ErrSymlinkEscape,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SafeJoin(tt.root, tt.relPath)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("SafeJoin(%q, %q) error = %v, wantErr %v", tt.root, tt.relPath, err, tt.wantErr)
			}
		})
	}
}

// TestIsWithinRoot tests workspace containment check
func TestIsWithinRoot(t *testing.T) {
	tmpDir := t.TempDir()
	outsideDir := t.TempDir()

	tests := []struct {
		name string
		root string
		path string
		want bool
	}{
		{
			name: "file inside root",
			root: tmpDir,
			path: filepath.Join(tmpDir, "src/main.go"),
			want: true,
		},
		{
			name: "root itself",
			root: tmpDir,
			path: tmpDir,
			want: true,
		},
		{
			name: "file outside root",
			root: tmpDir,
			path: filepath.Join(outsideDir, "file.txt"),
			want: false,
		},
		{
			name: "file in parent",
			root: tmpDir,
			path: filepath.Dir(tmpDir),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWithinRoot(tt.root, tt.path)
			if got != tt.want {
				t.Errorf("IsWithinRoot(%q, %q) = %v, want %v", tt.root, tt.path, got, tt.want)
			}
		})
	}
}

// TestRelativePath tests computing relative paths
func TestRelativePath(t *testing.T) {
	tests := []struct {
		name    string
		root    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "simple relative",
			root:    "/workspace",
			path:    "/workspace/src/main.go",
			want:    "src/main.go",
			wantErr: false,
		},
		{
			name:    "same directory",
			root:    "/workspace",
			path:    "/workspace",
			want:    ".",
			wantErr: false,
		},
		{
			name:    "nested directory",
			root:    "/workspace",
			path:    "/workspace/a/b/c/d/file.txt",
			want:    "a/b/c/d/file.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RelativePath(tt.root, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("RelativePath(%q, %q) error = %v, wantErr %v", tt.root, tt.path, err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("RelativePath(%q, %q) = %q, want %q", tt.root, tt.path, got, tt.want)
			}
		})
	}
}

// TestIsWithinRoot_EdgeCases tests edge cases for IsWithinRoot
func TestIsWithinRoot_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		root string
		path string
		want bool
	}{
		{
			name: "both relative paths",
			root: "workspace",
			path: "workspace/src",
			want: true,
		},
		{
			name: "workspace prefix but different dir",
			root: "/workspace",
			path: "/workspace2/src",
			want: false,
		},
		{
			name: "exact match",
			root: "/workspace/src",
			path: "/workspace/src",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWithinRoot(tt.root, tt.path)
			if got != tt.want {
				t.Errorf("IsWithinRoot(%q, %q) = %v, want %v", tt.root, tt.path, got, tt.want)
			}
		})
	}
}

// TestSafeJoin_NonExistentPath tests SafeJoin with non-existent paths
func TestSafeJoin_NonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Path that doesn't exist yet - should still be valid
	result, err := SafeJoin(tmpDir, "nonexistent/path/to/file.txt")
	if err != nil {
		t.Errorf("SafeJoin with non-existent path failed: %v", err)
	}

	expected := filepath.Join(tmpDir, "nonexistent/path/to/file.txt")
	if result != expected {
		t.Errorf("SafeJoin returned %q, want %q", result, expected)
	}
}

// TestSafeJoin_BrokenSymlink tests SafeJoin with broken symlinks
func TestSafeJoin_BrokenSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a broken symlink (pointing to non-existent file)
	brokenLink := filepath.Join(tmpDir, "broken_link")
	if err := os.Symlink("/nonexistent/target", brokenLink); err != nil {
		t.Fatalf("Failed to create broken symlink: %v", err)
	}

	// Should be allowed - the broken symlink itself is in the workspace
	relPath := "broken_link"
	result, err := SafeJoin(tmpDir, relPath)
	if err != nil {
		t.Errorf("SafeJoin with broken symlink failed: %v", err)
	}

	expected := filepath.Join(tmpDir, "broken_link")
	if result != expected {
		t.Errorf("SafeJoin returned %q, want %q", result, expected)
	}
}

// TestValidateRelativePath_Depth tests depth tracking algorithm
func TestValidateRelativePath_Depth(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name:    "depth stays positive: a/b/c",
			path:    "a/b/c",
			wantErr: nil,
		},
		{
			name:    "depth reaches zero: a/b/../c",
			path:    "a/b/../c",
			wantErr: nil,
		},
		{
			name:    "depth goes negative: a/../..",
			path:    "a/../..",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "depth goes negative immediately: ..",
			path:    "..",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "complex positive depth: a/b/c/../d/e/../f",
			path:    "a/b/c/../d/e/../f",
			wantErr: nil,
		},
		{
			name:    "complex negative depth: a/b/../../..",
			path:    "a/b/../../..",
			wantErr: ErrPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRelativePath(tt.path)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateRelativePath(%q) = %v, want %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
