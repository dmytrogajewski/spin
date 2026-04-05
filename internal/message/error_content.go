package message

import (
	"strings"
)

// errorKeywords are case-insensitive indicators of error content.
var errorKeywords = []string{
	"error",
	"failed",
	"exception",
	"panic",
	"fatal",
	"cannot",
	"could not",
	"unable to",
}

// IsErrorContent reports whether content contains common error indicators.
// The check is case-insensitive.
func IsErrorContent(content string) bool {
	lower := strings.ToLower(content)

	for _, kw := range errorKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}
