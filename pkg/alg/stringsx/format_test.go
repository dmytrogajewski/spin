package stringsx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatNumberedList(t *testing.T) {
	t.Parallel()

	t.Run("empty_list", func(t *testing.T) {
		t.Parallel()

		got := FormatNumberedList(
			[]string{},
			func(n int) string { return fmt.Sprintf("Found %d item(s):", n) },
			func(_ int, s string) string { return s },
		)
		require.Equal(t, "Found 0 item(s):", got)
	})

	t.Run("single_item", func(t *testing.T) {
		t.Parallel()

		got := FormatNumberedList(
			[]string{"alpha"},
			func(n int) string { return fmt.Sprintf("Found %d item(s):", n) },
			func(_ int, s string) string { return s },
		)
		require.Equal(t, "Found 1 item(s):\n1. alpha", got)
	})

	t.Run("multiple_items", func(t *testing.T) {
		t.Parallel()

		got := FormatNumberedList(
			[]string{"alpha", "beta", "gamma"},
			func(n int) string { return fmt.Sprintf("Found %d item(s):", n) },
			func(_ int, s string) string { return s },
		)
		require.Equal(t, "Found 3 item(s):\n1. alpha\n2. beta\n3. gamma", got)
	})

	t.Run("index_passed_to_formatter", func(t *testing.T) {
		t.Parallel()

		got := FormatNumberedList(
			[]string{"a"},
			func(_ int) string { return "Items:" },
			func(idx int, s string) string { return fmt.Sprintf("[%d] %s", idx, s) },
		)
		require.Equal(t, "Items:\n1. [1] a", got)
	})
}

func TestFormatGrouped(t *testing.T) {
	t.Parallel()

	type ref struct {
		file string
		line int
	}

	t.Run("empty_items", func(t *testing.T) {
		t.Parallel()

		got := FormatGrouped(
			[]ref{},
			func(r ref) string { return r.file },
			func(total, groups int) string { return fmt.Sprintf("%d refs in %d files:", total, groups) },
			func(k string) string { return k + ":" },
			func(r ref) string { return fmt.Sprintf("line %d", r.line) },
		)
		require.Equal(t, "0 refs in 0 files:", got)
	})

	t.Run("single_group", func(t *testing.T) {
		t.Parallel()

		refs := []ref{
			{file: "main.go", line: 10},
			{file: "main.go", line: 20},
		}

		got := FormatGrouped(
			refs,
			func(r ref) string { return r.file },
			func(total, groups int) string { return fmt.Sprintf("%d refs in %d files:", total, groups) },
			func(k string) string { return k + ":" },
			func(r ref) string { return fmt.Sprintf("line %d", r.line) },
		)
		require.Equal(t, "2 refs in 1 files:\nmain.go:\n  line 10\n  line 20\n", got)
	})

	t.Run("multiple_groups_preserves_order", func(t *testing.T) {
		t.Parallel()

		refs := []ref{
			{file: "b.go", line: 1},
			{file: "a.go", line: 2},
			{file: "b.go", line: 3},
		}

		got := FormatGrouped(
			refs,
			func(r ref) string { return r.file },
			func(total, groups int) string { return fmt.Sprintf("%d refs in %d files:", total, groups) },
			func(k string) string { return k + ":" },
			func(r ref) string { return fmt.Sprintf("line %d", r.line) },
		)
		require.Equal(t, "3 refs in 2 files:\nb.go:\n  line 1\n  line 3\n\na.go:\n  line 2\n", got)
	})
}

func TestFormatTable(t *testing.T) {
	t.Parallel()

	type proc struct {
		id      string
		command string
		state   string
	}

	columns := []Column{
		{Name: "ID", Width: 8},
		{Name: "Command", Width: 12},
		{Name: "State", Width: 8},
	}

	t.Run("empty_table", func(t *testing.T) {
		t.Parallel()

		got := FormatTable(
			[]proc{},
			columns,
			func(p proc) []string { return []string{p.id, p.command, p.state} },
		)
		expected := "ID       | Command      | State   \n" +
			"---------+--------------+---------\n"
		require.Equal(t, expected, got)
	})

	t.Run("single_row", func(t *testing.T) {
		t.Parallel()

		got := FormatTable(
			[]proc{{id: "1", command: "ls", state: "done"}},
			columns,
			func(p proc) []string { return []string{p.id, p.command, p.state} },
		)
		require.Contains(t, got, "1        | ls           | done    \n")
	})

	t.Run("multiple_rows", func(t *testing.T) {
		t.Parallel()

		procs := []proc{
			{id: "1", command: "ls", state: "done"},
			{id: "2", command: "cat", state: "running"},
		}

		got := FormatTable(procs, columns, func(p proc) []string {
			return []string{p.id, p.command, p.state}
		})
		require.Contains(t, got, "1        | ls           | done")
		require.Contains(t, got, "2        | cat          | running")
	})
}
