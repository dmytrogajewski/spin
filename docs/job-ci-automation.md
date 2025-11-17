## NAME

CI Automation - running Spin in non-interactive mode for CI/CD and scripts

## WHEN TO USE

Use `spin exec` when you want to:

- Keep branches green by automatically running tests and applying small fixes
- Enforce coding standards or refactor patterns across a codebase
- Run the agent as part of a CI job, cron, or one-off script without an interactive TUI

This mode is designed for headless environments: CI runners, containers, and shell scripts.

## PREREQUISITES

- Spin binary built and available in `PATH` or at `./bin/spin`
- Configured provider:
  - Environment variables (e.g. `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`) or
  - Local provider like Ollama/LM Studio with a small model (e.g. `qwen3:0.6b`)
- Optional config file (v2 format):

  ```yaml
  version: "2.0"
  provider: ollama
  model: qwen3:0.6b
  ```

## FLOW 1: ONE-OFF EXECUTION IN A SHELL

Goal: run a single prompt non-interactively and see the result in the same terminal.

### Steps

1. From your project root, run:

   ```bash
   ./bin/spin exec "run go test ./... and summarize failures"
   ```

   or, piping from stdin:

   ```bash
   echo "scan this repo for TODO comments and list files and line numbers" | ./bin/spin exec
   ```

2. Spin will:

   - Load configuration (from `--config-file`, `SPIN_CONFIG`, or defaults)
   - Build an LLM provider via `internal/llm/builder`
   - Create a conversation configured for exec mode (sandbox, tools, approvals)
   - Execute the prompt and stream a summary to stdout

3. When the turn is complete, `spin exec` exits with:

   - Exit code `0` on success
   - Non-zero exit code if execution fails and `--exit-on-error` is `true` (default)

### What this proves

- `spin exec` can be used as a simple CLI tool in scripts and terminals.
- Configuration loading and provider creation work without a TUI.
- Exit codes reflect success/failure for downstream tooling.

### Test coverage

- `cmd/spin/exec.go`
  - Unit tests (in `cmd/spin/exec_test.go`, if present) cover prompt parsing and flag handling
- `tests/e2e/e2e_test.go`
  - End-to-end “core functionality” tests validate config resolution and exec flows

> Note: If a specific end-to-end test for `spin exec` does not exist yet, this flow
> is a candidate for a dedicated e2e test that runs the binary with `exec` and asserts on exit code and output.

## FLOW 2: KEEPING A BRANCH GREEN IN CI

Goal: add a CI job that asks Spin to run tests and apply safe, auto-approved fixes.

### Steps

1. Ensure your provider is available in CI:

   - For remote providers, configure `OPENAI_API_KEY`/`ANTHROPIC_API_KEY` etc.
   - For local providers like Ollama, run them as a sidecar/container and point Spin at the correct base URL via config.

2. Add a job to your CI configuration (example: GitHub Actions):

   ```yaml
   jobs:
     spin-keep-green:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
         - uses: actions/setup-go@v5
           with:
             go-version: '1.23'
         - name: Build spin
           run: |
             go build -o bin/spin ./cmd/spin
         - name: Run spin exec to keep tests green
           env:
             OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
           run: |
             ./bin/spin exec \
               --auto-approve \
               --timeout 15m \
               "run go test ./...; fix any test failures; keep changes minimal and focused"
   ```

3. Configure Git to allow commits from the CI user if you want Spin to commit changes:

   ```bash
   git config user.name "spin-ci"
   git config user.email "ci@example.com"
   ```

4. Optionally, run Spin with `--sandbox workspace-write` to restrict writes to the repo.

### What this proves

- Spin can run in a fully headless environment and still use tools.
- Auto-approval can be used in controlled CI contexts to allow code changes.
- Timeouts prevent runs from blocking the pipeline indefinitely.

### Test coverage

- `tests/e2e/approval_cli_e2e_test.go`
  - `TestApprovalCLI_ListAndClear_Empty` – validates the approval policy file and CLI management
  - `TestApprovalCLI_Revoke_NonExistent` – validates revocation behavior for policy entries
- `tests/e2e/approval_persistence_e2e_test.go`
  - Verifies persistence of approval decisions across executions
  - Validates that tools execute correctly when called by the LLM

> Gaps: a dedicated e2e test that runs `spin exec` with `--auto-approve` in a temp repo and asserts
> that files (and optionally commits) are created would strengthen coverage for this flow.

## FLOW 3: READ-ONLY ANALYSIS IN CI (NO AUTO-APPROVE)

Goal: use Spin to analyze or review code in CI without making changes, even when tools are available.

### Steps

1. Run Spin in `exec` mode without `--auto-approve`:

   ```bash
   ./bin/spin exec \
     --timeout 5m \
     "analyze security of the auth/ package and summarize high-risk findings only"
   ```

2. In this mode:

   - Read-only tools (e.g. `read_file`, `list_directory`) execute normally.
   - Write operations and dangerous shell commands are denied by the approval handler in exec mode.

3. Consume the textual summary from stdout as part of your CI logs or artifacts.

### What this proves

- Exec mode can be used purely for analysis without risk of silent modifications.
- Approval handler for exec mode enforces a conservative policy when `--auto-approve` is not set.

### Test coverage

- `cmd/spin/exec.go`
  - `createConversationForExec` configures an approval handler that denies dangerous operations without `--auto-approve`.
- `tests/e2e/security/*` (if present)
  - Cover security classifications and forbidden commands

> Gaps: A focused e2e test that invokes `spin exec` without `--auto-approve` and asserts that
> a write attempt is denied (by checking exit code and stderr) would directly cover this flow.

## FLOW 4: MACHINE-CONSUMABLE OUTPUT WITH JSON

Goal: consume Spin’s output programmatically in CI or scripts.

### Steps

1. Run `spin exec` with JSON output:

   ```bash
   ./bin/spin exec \
     --format json \
     "summarize go test ./... output into a JSON object with fields: passed, failed, flakes"
   ```

2. Pipe the output into a JSON processor:

   ```bash
   ./bin/spin exec --format json "..." | jq '.failed'
   ```

3. Use the processed values to gate subsequent steps (e.g., fail the job if `.failed > 0`).

### What this proves

- Exec mode can shape output for downstream tools instead of just humans.
- JSON output can be wired into CI logic with standard tools like `jq`.

### Test coverage

- `cmd/spin/exec.go`
  - Flag handling and format selection are covered by unit tests (or should be added).

> Gaps: A dedicated unit or e2e test that verifies JSON output structure for a simple prompt
> should be added if not present.

## FLOW 5: SCHEDULING SPIN IN CRON OR BATCH JOBS

Goal: run Spin periodically to perform maintenance tasks on a repo.

### Steps

1. Create a small script, e.g. `scripts/spin-maintenance.sh`:

   ```bash
   #!/usr/bin/env bash
   set -euo pipefail

   cd /srv/repos/my-service

   ./bin/spin exec \
     --auto-approve \
     --timeout 20m \
     "run gofmt on changed files; update simple linter findings; do not introduce new dependencies"
   ```

2. Make it executable:

   ```bash
   chmod +x scripts/spin-maintenance.sh
   ```

3. Schedule it with cron (example):

   ```cron
   0 3 * * 1 /srv/repos/my-service/scripts/spin-maintenance.sh >> /var/log/spin-maintenance.log 2>&1
   ```

### What this proves

- Spin can act as a background maintenance agent, not only as an interactive assistant.
- Timeouts and exit codes integrate with traditional UNIX scheduling tools.

### Test coverage

- Same as Flow 2 (CI) – the underlying behavior is `spin exec` in a headless environment.

> Gaps: No additional behavior beyond Flow 2; cron-specific aspects (scheduling, logging) are not covered by automated tests.

## RELATED DOCUMENTS

- `docs/job-local-agent.md` – interactive TUI usage in a developer terminal
- `docs/job-acp-ide.md` – using Spin from an ACP-compatible IDE/editor
- `docs/testing-mapping.md` – mapping of all flows to tests


