// Package strutil provides string manipulation utilities for code processing.
//
// This package offers advanced string operations needed for AI-driven file
// modifications, including line manipulation, indentation detection, fuzzy
// matching, and case conversion utilities.
//
// Key Features:
//   - Line operations: Split, join, and trim lines with mixed line endings
//   - Indentation detection: Auto-detect tabs vs spaces and indentation size
//   - Whitespace normalization: Handle different line endings and spacing
//   - Similarity algorithms: Levenshtein distance and fuzzy matching
//   - Case conversion: snake_case, camelCase, PascalCase utilities
//
// Performance:
//   - SplitLines: <50μs for 1000-line files
//   - LevenshteinDistance: <100μs for 100-character strings
//   - DetectIndentation: <200μs for 100-line files
//   - Zero external dependencies
//
// Example usage:
//
//	// Split text with mixed line endings
//	lines := strutil.SplitLines("line1\r\nline2\nline3\r")
//	// Returns: ["line1", "line2", "line3"]
//
//	// Detect indentation style
//	useTabs, size := strutil.DetectIndentation(sourceCode)
//	// Returns: (false, 4) for 4-space indentation
//
//	// Calculate string similarity
//	similarity := strutil.Similarity("kitten", "sitting")
//	// Returns: 0.571 (57.1% similar)
//
//	// Convert case
//	snake := strutil.ToSnakeCase("MyVariableName")
//	// Returns: "my_variable_name"
package strutil
