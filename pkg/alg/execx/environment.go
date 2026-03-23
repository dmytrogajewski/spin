package execx

import "strings"

// minEnvSplitParts is the minimum number of parts from splitting KEY=VALUE.
const minEnvSplitParts = 2

// FilterEnvironment filters environment variables, excluding entries whose
// key matches any sensitive prefix (case-insensitive) or contains any
// sensitive substring (case-insensitive). Returns a map of safe key-value pairs.
func FilterEnvironment(env []string, sensitivePrefix, sensitiveSubstr []string) map[string]string {
	filtered := make(map[string]string)

	for _, entry := range env {
		parts := strings.SplitN(entry, "=", minEnvSplitParts)
		if len(parts) != minEnvSplitParts {
			continue
		}

		key := parts[0]
		if !isSensitiveKey(key, sensitivePrefix, sensitiveSubstr) {
			filtered[key] = parts[1]
		}
	}

	return filtered
}

// isSensitiveKey checks if a key matches any sensitive prefix or contains any sensitive substring.
func isSensitiveKey(key string, prefixes, substrings []string) bool {
	upper := strings.ToUpper(key)

	for _, prefix := range prefixes {
		if strings.HasPrefix(upper, strings.ToUpper(prefix)) {
			return true
		}
	}

	for _, substr := range substrings {
		if strings.Contains(upper, strings.ToUpper(substr)) {
			return true
		}
	}

	return false
}
