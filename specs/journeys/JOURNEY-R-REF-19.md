# JOURNEY-R-REF-19: Create pkg/alg/diff Package

**Roadmap Item:** R-REF-19

## Summary

Created `pkg/alg/diff` package with unified diff generation and parsing. Types: `LineType`, `LineChange`, `Hunk`. Functions: `Generate(filePath, oldText, newText)`, `Parse(diffText) (filename, []Hunk, error)`.

## Implementation

- **Created:** `pkg/alg/diff/diff.go` — Generate, Parse, LineType, LineChange, Hunk
- **Created:** `pkg/alg/diff/diff_test.go` — 10 tests, 92.9% coverage
