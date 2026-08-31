package tasks

import (
	"fmt"
	"strings"
)

const emptyList = "No agent tasks."

const emptyView = "No tasks."

// Format prints id, spec, and state — one row per task.
func Format(rows []Record) string {
	if len(rows) == 0 {
		return emptyList
	}

	var b strings.Builder
	for _, rec := range rows {
		fmt.Fprintf(&b, "%s  %s  %s\n", rec.ID, rec.Spec, rec.State)
	}

	return strings.TrimRight(b.String(), "\n")
}

// FormatView prints kind=agent|shell, typed id, spec, and state.
func FormatView(rows []Row) string {
	if len(rows) == 0 {
		return emptyView
	}

	var b strings.Builder
	for _, rec := range rows {
		fmt.Fprintf(&b, "kind=%s  %s  %s  %s\n", rec.Kind, rec.ID, rec.Spec, rec.State)
	}

	return strings.TrimRight(b.String(), "\n")
}
