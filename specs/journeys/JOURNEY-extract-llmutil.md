# JOURNEY-extract-llmutil: Extract cleanJSONResponse to internal/llmutil

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 2.2 — Extract `cleanJSONResponse` to `internal/llmutil`
- Cluster: 16 (SPEC.md) | LIST.md finding 50

## 1. Journey

When **a developer parsing LLM responses in ACE modules** I want to **call a single, tested `llmutil.CleanJSONResponse` function** so I can **avoid maintaining two identical copies of the same markdown-stripping logic across curator and reflector**.

## 2. CJM

Two production packages independently implement `cleanJSONResponse(response string) string`:
- `internal/ace/curator/curator.go` (1 call-site)
- `internal/ace/reflector/reflector.go` (2 call-sites)

Both strip markdown code block wrappers (` ```json ` and ` ``` `) from LLM responses to extract raw JSON.

The curator version is slightly more robust (uses `CutSuffix` for trailing backticks). The unified version adopts the curator's approach.

### Phase 1: Create llmutil package

**Actions:**
1. Create `internal/llmutil/json.go` with `CleanJSONResponse(response string) string`.
2. Write comprehensive unit tests.

**Success Signal:** All edge cases covered: plain JSON, ```json wrapper, ``` wrapper, whitespace, empty.

### Phase 2: Migrate call-sites

**Actions:**
1. Replace 2 production `cleanJSONResponse` functions with `llmutil.CleanJSONResponse`.
2. Update test file in reflector to use `llmutil.CleanJSONResponse`.

**Success Signal:** All ACE tests pass, no duplicate functions remain.

### Phase 3: Verification

**Actions:** `go vet`, `make lint`, `go test -race ./internal/llmutil/... ./internal/ace/curator/... ./internal/ace/reflector/...`

**Success Signal:** Zero issues, all tests green.

### North Star Summary

A single `llmutil.CleanJSONResponse` function serves all LLM response cleanup needs. No duplicate implementations remain.

## 3. Tests

### TC-01: plain JSON passes through unchanged
### TC-02: ```json wrapper stripped
### TC-03: ``` wrapper stripped
### TC-04: extra whitespace trimmed
### TC-05: empty string returns empty
### TC-06: whitespace-only returns empty
### TC-07: nested backticks handled correctly

## Implementation

- Created: `internal/llmutil/json.go` — `CleanJSONResponse`
- Created: `internal/llmutil/json_test.go` — comprehensive unit tests
- Modified: `internal/ace/curator/curator.go` — uses `llmutil.CleanJSONResponse`
- Modified: `internal/ace/reflector/reflector.go` — uses `llmutil.CleanJSONResponse`
- Modified: `internal/ace/reflector/reflector_test.go` — uses `llmutil.CleanJSONResponse`
