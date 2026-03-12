package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/memory"
)

const previewTruncateLen = 100

// storeList lists keys from a memory store matching a pattern.
func storeList(ctx context.Context, store memory.Store, params ToolParameters, label string) (ToolResult, error) {
	pattern := params.GetStringOr("pattern", "*")

	keys, listErr := store.List(ctx, pattern)
	if listErr != nil {
		return ErrToResultf("failed to list entries: %v", listErr)
	}

	if len(keys) == 0 {
		return NewToolResult(fmt.Sprintf("No entries found in %s", label)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d entries in %s:\n", len(keys), label)

	for _, key := range keys {
		fmt.Fprintf(&sb, "  - %s\n", key)
	}

	return NewToolResult(sb.String()), nil
}

// storeSearch searches a memory store for entries matching a query.
func storeSearch(
	ctx context.Context,
	store memory.Store,
	params ToolParameters,
	searchLimit int,
	maxPreview int,
	label string,
) (ToolResult, error) {
	query, _ := params.GetString("query")
	if query == "" {
		return NewToolError(ErrQueryParameterRequiredForSearch), nil
	}

	entries, searchErr := store.Search(ctx, query, searchLimit)
	if searchErr != nil {
		return ErrToResultf("failed to search entries: %v", searchErr)
	}

	if len(entries) == 0 {
		return NewToolResult(fmt.Sprintf("No entries found matching '%s' in %s", query, label)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d entries matching '%s':\n", len(entries), query)

	for _, entry := range entries {
		preview := entry.Value
		if len(preview) > maxPreview {
			preview = preview[:previewTruncateLen] + "..."
		}

		fmt.Fprintf(&sb, "  - %s: %s\n", entry.Key, preview)
	}

	return NewToolResult(sb.String()), nil
}
