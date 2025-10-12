package pathutil

import (
	"path/filepath"
	"testing"
)

// BenchmarkValidateRelativePath benchmarks simple path validation
func BenchmarkValidateRelativePath(b *testing.B) {
	path := "src/internal/core/agent.go"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateRelativePath(path)
	}
}

// BenchmarkValidateRelativePath_Complex benchmarks deep path validation
func BenchmarkValidateRelativePath_Complex(b *testing.B) {
	path := "a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/file.txt"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateRelativePath(path)
	}
}

// BenchmarkValidateRelativePath_WithTraversal benchmarks path with parent references
func BenchmarkValidateRelativePath_WithTraversal(b *testing.B) {
	path := "a/b/../c/d/../e/f"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateRelativePath(path)
	}
}

// BenchmarkNormalizePath benchmarks path normalization
func BenchmarkNormalizePath(b *testing.B) {
	path := "./src/./internal/../core/agent.go"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NormalizePath(path)
	}
}

// BenchmarkNormalizePath_Simple benchmarks simple path (should be zero alloc)
func BenchmarkNormalizePath_Simple(b *testing.B) {
	path := "src/main.go"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NormalizePath(path)
	}
}

// BenchmarkSafeJoin benchmarks safe path joining
func BenchmarkSafeJoin(b *testing.B) {
	root := "/workspace"
	relPath := "src/internal/core/agent.go"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SafeJoin(root, relPath)
	}
}

// BenchmarkIsWithinRoot benchmarks workspace containment check
func BenchmarkIsWithinRoot(b *testing.B) {
	root := "/workspace"
	path := "/workspace/src/main.go"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsWithinRoot(root, path)
	}
}

// BenchmarkRelativePath benchmarks relative path computation
func BenchmarkRelativePath(b *testing.B) {
	root := "/workspace"
	path := "/workspace/src/internal/core/agent.go"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = RelativePath(root, path)
	}
}

// BenchmarkFilepathClean is a baseline to compare our NormalizePath against
func BenchmarkFilepathClean(b *testing.B) {
	path := "./src/./internal/../core/agent.go"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filepath.Clean(path)
	}
}
