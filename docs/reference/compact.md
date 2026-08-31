# Compact reference

<!-- Reference template: Good Docs Project (CC-BY 4.0). Lookup only; not a how-to. -->

Tool-output compact (RTK-style). Default on. Applies to shell exec and built-in `read_file` / `grep` / glob / `list_directory`.

## Synopsis

```
SPIN_COMPACT=0
compact.enabled: false
compact.backend: "" | rtk
compact.read_level: none | minimal | aggressive
```

`/help` prints the same escape hatch.

## Description

The Go pipeline filters stdout/stderr after a command or built-in tool returns. Exit codes are never rewritten. Unknown commands pass through unchanged. Optional R11 rewrite prefixes PATH `rtk` when `compact.backend` is `rtk` and `rtk` is on `PATH`.

A `--no-compact` CLI flag appears in [SPEC.md](../../specs/agent-harness/SPEC.md) and is not shipped.

## Escape hatch

| Control | Effect |
|---------|--------|
| `SPIN_COMPACT=0` | Disables compact and R11 rewrite for the process |
| `compact.enabled: false` | Disables the Go pipeline (same as `/help`) |
| `compact.backend: rtk` | R11 argv prefix when `rtk` resolves on `PATH`; missing binary is a no-op |
| `compact.read_level` | Default R8 level for built-in read (`minimal` if unset) |

Status bar shows on/off and last-turn output-bytes reduction (`−N%`) from the R15 ledger.

## Command table

Landed `Default()` registry (prefix match on `cmd` plus space).

| Input | Compact form |
|-------|--------------|
| `ls` / `tree` / `find` | Tree plus per-directory file counts (R10) |
| `cat` / `read` / `smart` | R8 code filter: `none` / `minimal` (default, comments stripped) / `aggressive` (signatures). `-l` or `--level=` on the command overrides |
| `grep` / `rg` | Truncated lines, grouped by file |
| `git status` | Compact stat, grouped by state |
| `git diff` | Reduced context, headers stripped |
| `git log` | Hash, author, subject |
| `git add` / `git commit` / `git push` / `git pull` | One confirmation line |
| `go test` | NDJSON parsed; failures only (R9) |
| `cargo test` / `npm test` / `pytest` / `jest` / `vitest` / `playwright` | Failures only |
| `ruff` / `ruff check` | Grouped by rule and file |
| `docker ps` | Essential fields only |
| `log` / `dedup` | Adjacent duplicate lines collapsed to ` ×N` (R4) |
| `json` | Keys and types; large values stripped (R7) |

## Rules R12–R15

| Id | Name | Contract |
|----|------|----------|
| R12 | Fail-safe | Filter error or panic returns the original stdout/stderr |
| R13 | Exit-code preservation | `Result.ExitCode` is the input exit code; compact never swallows non-zero |
| R14 | Unknown passthrough | No registered command (and no prefix match) → identity bytes, strategy `R14`, 0% reduction |
| R15 | Token estimate | Ledger is `ceil(bytes/4)` on stdout+stderr. Not a tokenizer. Status chip uses byte reduction percent |

R1–R11 are the per-command filters and the optional argv rewrite above. R11 is argv-level (`RewriteArgv`), not a stdio filter.

## See also

- [How to write an agent skill](../how-to/agent-skills.md)
- [Hooks reference](hooks.md) (`PRE_COMPACT`)
