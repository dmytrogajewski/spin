## NAME

Spin Docs - guide for the Spin agent

## DESCRIPTION

This `docs/` tree explains Spin in terms of complete, realistic workflows instead of individual features.
Each guide follows a concrete value stream from the first command to a green test run,
and every documented scenario is backed by automated tests (or explicitly marked as needing one).

Spin focuses on three primary flows:

- Local autonomous coding agent in your terminal
- Headless automation for CI/CD and scripts
- ACP-compatible backend agent for your IDE

## DOCUMENT MAP

### Usage Guides

- **Local agent**: `docs/job-local-agent.md` – using Spin as an autonomous coding agent in your terminal
- **CI/CD automation**: `docs/job-ci-automation.md` – using Spin in CI/CD pipelines and automation scripts
- **IDE integration**: `docs/job-acp-ide.md` – using Spin with an ACP-compatible IDE/editor

### Configuration and Setup

- **Configuration**: `docs/configuration.md` – setting up Spin with your LLM provider and preferences
- **Troubleshooting**: `docs/troubleshooting.md` – solving common issues with Spin

### Reference

- **Test mapping**: `docs/testing-mapping.md` – how documented scenarios map to unit and e2e tests

## TEST-BACKED DOCUMENTATION

Every scenario described in this documentation must be executable and verifiable:

- End-to-end behavior is covered by tests under `tests/e2e/`
- Unit and integration behavior is covered by tests in `internal/` and `cmd/spin/`
- The `docs/testing-mapping.md` file lists each scenario and the test functions that exercise it

When you extend Spin or add a new user flow:

1. Add or update a scenario in the appropriate job document
2. Add or update the corresponding row in `docs/testing-mapping.md`
3. Implement or extend tests so that the scenario is fully covered

If a scenario is documented but still missing coverage, it must be marked as `needs-test`
in `docs/testing-mapping.md` until tests are added.

## SEE ALSO

- Top-level manual: `README.md`
- Examples: `examples/`
- Tests: `tests/`


