package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/memory"
)

// entryFormatter appends tool-specific fields to the get output.
type entryFormatter func(entry *memory.Entry, sb *strings.Builder)

// storePut stores a key-value pair in a memory store.
func storePut(ctx context.Context, store memory.Store, params ToolParameters, label string) (ToolResult, error) {
	key := params.GetStringOr("key", "")
	if key == "" {
		return NewToolError(errKeyParameterRequiredForPut), nil
	}

	value := params.GetStringOr("value", "")
	if value == "" {
		return NewToolError(errValueParameterRequiredForPut), nil
	}

	opts := memory.PutOptions{Overwrite: true}

	ns := params.GetStringOr("namespace", "")
	if ns != "" {
		opts.Namespace = ns
	}

	if params.Has("tags") {
		var tags []string

		tagsErr := params.GetObject("tags", &tags)
		if tagsErr == nil {
			opts.Tags = tags
		}
	}

	putErr := store.Put(ctx, key, value, opts)
	if putErr != nil {
		return ErrToResultf("failed to store entry: %w", putErr)
	}

	return NewToolResult(fmt.Sprintf("Stored entry with key '%s' (%d bytes%s)", key, len(value), labelSuffix(label))), nil
}

// storeGet retrieves an entry from a memory store.
// The optional formatter adds tool-specific fields to the output.
func storeGet(
	ctx context.Context, store memory.Store, params ToolParameters,
	label string, formatter entryFormatter,
) (ToolResult, error) {
	key := params.GetStringOr("key", "")
	if key == "" {
		return NewToolError(errKeyParameterRequiredForGet), nil
	}

	entry, getErr := store.Get(ctx, key)
	if getErr != nil {
		if errors.Is(getErr, memory.ErrNotFound) {
			return NewToolError(fmt.Errorf("key '%s' not found in %s: %w", key, label, errKeyNotFound)), nil
		}

		return ErrToResultf("failed to get entry: %w", getErr)
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "Key: %s\n", entry.Key)
	fmt.Fprintf(&sb, "Namespace: %s\n", entry.Namespace)

	if len(entry.Tags) > 0 {
		fmt.Fprintf(&sb, "Tags: %s\n", strings.Join(entry.Tags, ", "))
	}

	if formatter != nil {
		formatter(entry, &sb)
	}

	fmt.Fprintf(&sb, "Value:\n%s", entry.Value)

	return NewToolResult(sb.String()), nil
}

// storeDelete removes an entry from a memory store.
func storeDelete(ctx context.Context, store memory.Store, params ToolParameters, label string) (ToolResult, error) {
	key := params.GetStringOr("key", "")
	if key == "" {
		return NewToolError(errKeyParameterRequiredForDel), nil
	}

	delErr := store.Delete(ctx, key)
	if delErr != nil {
		return ErrToResultf("failed to delete entry: %w", delErr)
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' deleted%s", key, labelSuffix(label))), nil
}

// labelSuffix returns a label suffix for messages (e.g., ", persistent" or "").
func labelSuffix(label string) string {
	if label == "" {
		return ""
	}

	return ", " + label
}

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
	query := params.GetStringOr("query", "")
	if query == "" {
		return NewToolError(errQueryParamRequiredForSearch), nil
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
			preview = preview[:maxPreview] + "..."
		}

		fmt.Fprintf(&sb, "  - %s: %s\n", entry.Key, preview)
	}

	return NewToolResult(sb.String()), nil
}
