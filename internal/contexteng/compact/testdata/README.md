# Compact filter goldens

Fixtures are hand-authored from `specs/agent-harness/SPEC.md` (RTK R1–R10 plus the per-command table). They are **not** captured from a live `rtk` binary. Behavior matches the spec table.

Each directory is one apply:

| File | Role |
|------|------|
| `cmd` | Command string passed to `Apply` |
| `exit` | Process exit code (must be unchanged) |
| `raw` | Stdin-like command stdout |
| `compact` | Expected compacted stdout |

| Directory | Spec row | RTK |
|-----------|----------|-----|
| `ls`, `tree` | `ls` / `tree` | R10 |
| `read-none`, `read-minimal`, `read-aggressive` | `cat` / `read` | R8 |
| `grep`, `rg` | `grep` / `rg` | R2+R3 |
| `git-status` | `git status` | R5 |
| `git-diff` | `git diff` | R3 |
| `git-log` | `git log` | R5 |
| `git-add`, `git-commit`, `git-push`, `git-pull` | `git add/commit/push/pull` | confirm line |
| `gotest` | `go test` NDJSON | R9 |
| `pytest`, `jest`, `vitest`, `playwright`, `cargo-test`, `npm-test` | test runners | R6+R9 |
| `ruff` | `ruff check` | R2 |
| `docker-ps` | `docker ps` | essential fields |
| `dedup` | logs | R4 |
| `json` | structure-only JSON | R7 |
