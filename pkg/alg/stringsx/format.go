package stringsx

import (
	"fmt"
	"strings"
)

// Column defines a single column for table formatting.
type Column struct {
	Name  string
	Width int
}

// FormatNumberedList formats items as a 1-indexed numbered list.
// The header function receives the item count and returns a header line.
// The formatter function receives the 1-based index and item, returning the line text.
func FormatNumberedList[T any](items []T, header func(int) string, formatter func(int, T) string) string {
	var buf strings.Builder

	buf.WriteString(header(len(items)))

	for idx, item := range items {
		fmt.Fprintf(&buf, "\n%d. %s", idx+1, formatter(idx+1, item))
	}

	return buf.String()
}

// FormatGrouped formats items grouped by a key function.
// The header function receives (totalItems, groupCount).
// The groupFormatter formats each item within a group.
func FormatGrouped[T any, Key comparable](
	items []T,
	keyFunc func(T) Key,
	header func(int, int) string,
	groupHeader func(Key) string,
	itemFormatter func(T) string,
) string {
	groups := make(map[Key][]T)

	var order []Key

	for _, item := range items {
		key := keyFunc(item)
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}

		groups[key] = append(groups[key], item)
	}

	var buf strings.Builder

	buf.WriteString(header(len(items), len(groups)))

	for _, key := range order {
		buf.WriteString("\n")
		buf.WriteString(groupHeader(key))
		buf.WriteString("\n")

		for _, item := range groups[key] {
			buf.WriteString("  ")
			buf.WriteString(itemFormatter(item))
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// FormatTable formats items as an aligned table with column headers and separators.
// The rowFormatter extracts cell values from each item.
func FormatTable[T any](items []T, columns []Column, rowFormatter func(T) []string) string {
	var buf strings.Builder

	// Header row.
	for i, col := range columns {
		if i > 0 {
			buf.WriteString(" | ")
		}

		fmt.Fprintf(&buf, "%-*s", col.Width, col.Name)
	}

	buf.WriteString("\n")

	// Separator row.
	for i, col := range columns {
		if i > 0 {
			buf.WriteString("-+-")
		}

		buf.WriteString(strings.Repeat("-", col.Width))
	}

	buf.WriteString("\n")

	// Data rows.
	for _, item := range items {
		values := rowFormatter(item)

		for i, col := range columns {
			if i > 0 {
				buf.WriteString(" | ")
			}

			val := ""
			if i < len(values) {
				val = values[i]
			}

			fmt.Fprintf(&buf, "%-*s", col.Width, val)
		}

		buf.WriteString("\n")
	}

	return buf.String()
}
