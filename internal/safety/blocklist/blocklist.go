// Package blocklist provides an independent command blocklist checker (Layer 4).
// It blocks destructive commands regardless of approval configuration.
package blocklist

import "regexp"

// Rule defines a blocklist pattern with a human-readable description.
type Rule struct {
	// Pattern is the compiled regex to match against the full command string.
	Pattern *regexp.Regexp
	// Reason explains why the command is blocked.
	Reason string
}

// Checker evaluates command strings against blocklist rules.
// It operates independently of the approval system (Layer 4 defense-in-depth).
type Checker struct {
	rules   []Rule
	enabled bool
}

// NewChecker creates a Checker with default destructive-command rules.
func NewChecker(enabled bool) *Checker {
	return &Checker{
		rules:   defaultRules(),
		enabled: enabled,
	}
}

// Check evaluates a command string against the blocklist.
// Returns true and a reason if the command is blocked.
// Returns false and empty reason if the command is allowed.
// A disabled checker always returns false.
func (c *Checker) Check(command string) (blocked bool, reason string) {
	if !c.enabled {
		return false, ""
	}

	for _, rule := range c.rules {
		if rule.Pattern.MatchString(command) {
			return true, rule.Reason
		}
	}

	return false, ""
}

// Enabled returns whether the blocklist checker is active.
func (c *Checker) Enabled() bool {
	return c.enabled
}

// defaultRules returns the standard set of destructive command blocklist rules.
func defaultRules() []Rule {
	return []Rule{
		{
			Pattern: regexp.MustCompile(`\brm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+)?(-[a-zA-Z]*r[a-zA-Z]*\s+)?/\s*$`),
			Reason:  "blocked: rm targeting root filesystem",
		},
		{
			Pattern: regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*\s+)?(-[a-zA-Z]*f[a-zA-Z]*\s+)?/\s*$`),
			Reason:  "blocked: rm targeting root filesystem",
		},
		{
			Pattern: regexp.MustCompile(`\brm\s+-rf\s+/\*`),
			Reason:  "blocked: rm -rf /* targeting root contents",
		},
		{
			Pattern: regexp.MustCompile(`\brm\s+-rf\s+~`),
			Reason:  "blocked: rm -rf targeting home directory",
		},
		{
			Pattern: regexp.MustCompile(`\brm\s+-rf\s+\$HOME`),
			Reason:  "blocked: rm -rf targeting home directory",
		},
		{
			Pattern: regexp.MustCompile(`\bdd\s+.*\bof=/dev/[a-z]`),
			Reason:  "blocked: dd writing to device",
		},
		{
			Pattern: regexp.MustCompile(`\bmkfs\b`),
			Reason:  "blocked: filesystem formatting",
		},
		{
			Pattern: regexp.MustCompile(`:\(\)\s*\{`),
			Reason:  "blocked: fork bomb detected",
		},
		{
			Pattern: regexp.MustCompile(`\bchmod\s+(-R\s+)?777\s+/\s*$`),
			Reason:  "blocked: insecure permissions on root",
		},
		{
			Pattern: regexp.MustCompile(`\b(curl|wget)\s+.*\|\s*(bash|sh|zsh)\b`),
			Reason:  "blocked: piping download to shell",
		},
	}
}
