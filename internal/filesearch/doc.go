// Package filesearch provides file scanning and fuzzy matching functionality
// for the Spin TUI file picker.
//
// The package includes:
//   - Scanner: Scans directories recursively for files
//   - Matcher: Fuzzy matching algorithm for file paths
//
// Example usage:
//
//	scanner := filesearch.NewScanner(".", true)
//	files, _ := scanner.Scan()
//
//	matcher := filesearch.NewMatcher(false)
//	matches := matcher.Match("test", files)
package filesearch
