package security

import (
	"strings"
)

// validator_matchers.go implements pattern matching logic for command classification.
//
// This file contains methods for:
//   - Checking special forbidden patterns (fork bombs, disk overwrite)
//   - Matching patterns against commands
//   - Checking pattern lists and maps

// checkSpecialForbiddenPatterns checks for special forbidden patterns that require raw command inspection.
func (v *Validator) checkSpecialForbiddenPatterns(cmd *Command) *ValidationResult {
	// Fork bomb check.
	if strings.Contains(cmd.Raw, ":(){ :|:& };:") || strings.Contains(cmd.Raw, ":()|:|&") {
		return &ValidationResult{
			Classification: CommandForbidden,
			Reason:         "Fork bomb detected",
			MatchedRule:    "fork-bomb",
			Confidence:     1.0,
		}
	}

	// Disk overwrite check.
	if strings.HasPrefix(cmd.Raw, ">") && strings.Contains(cmd.Raw, "/dev/") {
		return &ValidationResult{
			Classification: CommandForbidden,
			Reason:         "Attempting to overwrite system disk",
			MatchedRule:    "disk-overwrite",
			Confidence:     1.0,
		}
	}

	// mkfs check (any filesystem).
	if strings.HasPrefix(cmd.Program, "mkfs") {
		return &ValidationResult{
			Classification: CommandForbidden,
			Reason:         "Attempting to format filesystem",
			MatchedRule:    "mkfs",
			Confidence:     1.0,
		}
	}

	return nil
}

// checkPatternList checks a list of patterns (for forbidden patterns).
func (v *Validator) checkPatternList(cmd *Command, patterns []Pattern, classification CommandClass) *ValidationResult {
	for _, pattern := range patterns {
		if v.matchesPattern(cmd, pattern) {
			return &ValidationResult{
				Classification: classification,
				Reason:         pattern.Description,
				MatchedRule:    pattern.Program,
				Confidence:     1.0,
			}
		}
	}

	return nil
}

// checkPatternMap checks a map of patterns (for dangerous/interactive/safe patterns).
func (v *Validator) checkPatternMap(cmd *Command, patternMap map[string][]Pattern, classification CommandClass) *ValidationResult {
	patterns, ok := patternMap[cmd.Program]
	if !ok {
		return nil
	}

	for _, pattern := range patterns {
		if v.matchesPattern(cmd, pattern) {
			return &ValidationResult{
				Classification: classification,
				Reason:         pattern.Description,
				MatchedRule:    pattern.Program,
				Confidence:     1.0,
			}
		}
	}

	return nil
}

// matchesPattern checks if a command matches a given pattern.
func (v *Validator) matchesPattern(cmd *Command, pattern Pattern) bool {
	// Check program name.
	if cmd.Program != pattern.Program {
		return false
	}

	// Combine all args into a single string for pattern matching.
	argsStr := strings.Join(cmd.Args, " ")

	// Check forbidden patterns (must be present for forbidden matches).
	if len(pattern.ForbiddenPatterns) > 0 {
		for _, forbiddenPattern := range pattern.ForbiddenPatterns {
			if strings.Contains(argsStr, forbiddenPattern) {
				return true // Matches forbidden pattern.
			}
		}
		// If forbidden patterns specified but none found, no match.
		return false
	}

	// Check arg patterns (all must be present).
	for _, argPattern := range pattern.ArgPatterns {
		if !strings.Contains(argsStr, argPattern) {
			return false
		}
	}

	// If no arg patterns specified, any args match.
	if len(pattern.ArgPatterns) == 0 {
		return true
	}

	// All arg patterns matched.
	return true
}
