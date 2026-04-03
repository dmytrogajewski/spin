// Package llmutil provides shared utilities for processing LLM responses.
package llmutil

import "github.com/dmytrogajewski/spin/pkg/alg/stringsx"

// CleanJSONResponse strips markdown code block wrappers from an LLM response
// to extract raw JSON. It handles ```json and ``` prefixes, trailing ```
// suffixes, and surrounding whitespace.
func CleanJSONResponse(response string) string {
	content, _ := stringsx.StripCodeFence(response)
	return content
}
