// Package llmutil provides shared utilities for processing LLM responses.
package llmutil

import "strings"

// CleanJSONResponse strips markdown code block wrappers from an LLM response
// to extract raw JSON. It handles ```json and ``` prefixes, trailing ```
// suffixes, and surrounding whitespace.
func CleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)

	// Handle markdown code blocks.
	if after, ok := strings.CutPrefix(response, "```json"); ok {
		response = after
		response = strings.TrimSpace(response)
	} else if afterPlain, okPlain := strings.CutPrefix(response, "```"); okPlain {
		response = afterPlain
		response = strings.TrimSpace(response)
	}

	// Remove trailing code block markers.
	if before, ok := strings.CutSuffix(response, "```"); ok {
		response = before
		response = strings.TrimSpace(response)
	}

	return response
}
