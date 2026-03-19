# Testing Guide

## Overview

Spin uses a three-tier testing strategy: unit tests, integration tests, and fixture-driven E2E tests. All tiers run via `make test` and must pass before merge.

```
make test       # all tests
make lint       # golangci-lint + deadcode analysis
go test ./...   # Go tests only
```

## Fixture-Driven E2E Tests

Fixture tests run the real `spin exec` binary against pre-recorded LLM responses. This gives deterministic, reproducible E2E coverage without requiring an LLM API key.

### How It Works

1. **Fixture files** live in `tests/e2e/fixtures/*.jsonl`.
2. Each JSONL line represents one LLM response (one `Stream()` call).
3. The `test-llm` provider replays fixture lines sequentially.
4. Tests assert on terminal output, file system side-effects, and exit codes.

### JSONL Format

Each line is a JSON object with a `chunks` array:

```json
{"chunks":[{"id":"c1","model":"fix","object":"chat.completion.chunk","created":0,"choices":[{"index":0,"delta":{"role":"assistant","content":"Hello world."},"finish_reason":"stop"}]}]}
```

For tool calls:

```json
{"chunks":[{"id":"c1","model":"fix","object":"chat.completion.chunk","created":0,"choices":[{"index":0,"delta":{"role":"assistant","content":"Reading file.","tool_calls":[{"index":0,"id":"tc-1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"test.txt\"}"}}]},"finish_reason":"tool_calls"}]}]}
```

Multi-turn conversations use multiple lines (one per LLM response):

```jsonl
{"chunks":[...tool call response...]}
{"chunks":[...final text response after tool result...]}
```

Optional `delay_ms` field pauses before sending chunks (for timeout testing):

```json
{"chunks":[...],"delay_ms":5000}
```

### Recording Fixtures from Real Sessions

Use `--record-fixture` to capture a real LLM session as a replayable fixture:

```bash
spin exec "read test.txt and summarize it" \
  --record-fixture tests/e2e/fixtures/read_and_summarize.jsonl
```

This wraps the real provider with a recording layer that:
- Captures every streaming chunk from the LLM.
- Writes one JSONL line per `Stream()` call.
- Produces output in the exact format `FixtureProvider` expects.
- Does not affect the session — recording is transparent.

The recorded fixture can then be used in tests:

```go
func TestFixture_ReadAndSummarize(t *testing.T) {
    t.Parallel()

    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    workDir := setupFixtureWorkDir(t, map[string]string{
        "test.txt": "content to summarize",
    })

    r := runFixtureExec(t, "read_and_summarize.jsonl",
        "read test.txt and summarize it",
        withWorkDir(workDir), withAutoApprove())
    assertNoError(t, r)
    assertOutputContains(t, r, "summary")
}
```

### Writing Fixture Tests

All fixture tests live in `tests/e2e/fixture_exec_test.go`. Helper functions are in `tests/e2e/fixture_helpers_test.go`.

Key helpers:

| Helper | Purpose |
|--------|---------|
| `runFixtureExec(t, fixture, prompt, opts...)` | Run `spin exec` with a fixture file |
| `setupFixtureWorkDir(t, files)` | Create temp dir with files |
| `assertOutputContains(t, r, substr)` | Assert output contains text |
| `assertNoError(t, r)` | Assert clean exit |
| `withAutoApprove()` | Pass `--auto-approve` |
| `withWorkDir(dir)` | Set working directory |
| `withTimeout(d)` | Set test timeout |
| `withExecTimeout(val)` | Pass `--timeout` flag |

### Running Fixture Tests

```bash
# Build the test binary first (requires e2e_llm_test tag).
make build-test

# Run all fixture tests.
go test ./tests/e2e/... -v

# Run a specific fixture test.
go test ./tests/e2e/... -v -run TestFixture_SimpleResponse

# Run ACP protocol tests.
go test ./tests/e2e/acp/... -v
```

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `SPIN_TEST_FIXTURE` | Path to JSONL fixture file (used by test-llm provider) |

## Integration Tests

Integration tests verify wiring between components. They live alongside the code they test:

| Journey | Test File | What It Covers |
|---------|-----------|----------------|
| 0.1 DoomLoop | `internal/agent/harness/doomloop_integration_test.go` | Guard detection, reminder injection, reset |
| 1.1 Pipeline | `internal/agent/executor/adapter_pipeline_test.go` | Pipeline stages, halt behavior |
| 1.2 Blocklist | `internal/agent/executor/stage_blocklist_test.go` | Dangerous pattern blocking |
| 1.3 Hooks | `internal/safety/hooks/lifecycle_test.go` | SESSION_START, USER_PROMPT_SUBMIT, blocking |
| 2.1 Undo | `internal/undo/service_integration_test.go` | Full rollback flow, multi-snapshot |
| 2.3 Session | `internal/session/persistence_integration_test.go` | Index reopen, transcript write/read |
| 3.1 SubAgent | `internal/agent/subagent/manager_integration_test.go` | Builtins, spec lookup, override |
| 3.2 Context | `internal/contexteng/retrieval/pipeline_integration_test.go` | Multi-source assembly, bullet source |
| 3.3 Cache | `internal/llm/cache/persistence_integration_test.go` | Cross-instance persistence, staleness |
| 4.2 Web | `internal/tools/web_fetch_integration_test.go` | HTTP fetch + HTML conversion |

## Test Patterns

### Table-Driven Tests

```go
tests := []struct {
    name    string
    input   string
    want    string
}{
    {name: "basic", input: "hello", want: "HELLO"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel()
        got := Transform(tt.input)
        require.Equal(t, tt.want, got)
    })
}
```

### Test Helpers

- Use `t.Helper()` for all helper functions.
- Use `t.Parallel()` for all tests.
- Use `t.TempDir()` for file system tests.
- Use `testify/require` for assertions (fail-fast).
- Prefer interfaces + test doubles over mocking frameworks.

### Journey Comments

Every test file links to its journey spec:

```go
// Journey: specs/journeys/JOURNEY-1.1.md.
```

## Architecture

```
tests/
  e2e/
    fixtures/           # JSONL fixture files (recorded or hand-crafted)
    fixture_exec_test.go      # Fixture-driven exec tests
    fixture_helpers_test.go   # Test helpers (runFixtureExec, etc.)
    acp/                # ACP protocol E2E tests
  compliance/           # Protocol compliance tests

internal/
  llm/
    recorder/           # Recording provider wrapper
    testprovider/       # Fixture replay provider (build tag: e2e_llm_test)
    mock.go             # Mock provider for unit tests
```
