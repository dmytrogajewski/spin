# JOURNEY-replace-boolstring: Replace boolString with strconv.FormatBool

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 1.5 — Replace `boolString` with `strconv.FormatBool`
- Cluster: 19 (SPEC.md) | LIST.md finding 29

## 1. Journey

When **a developer reading the conversation builder code** I want to **see stdlib `strconv.FormatBool` instead of a custom `boolString` function** so I can **immediately recognize the intent without looking up a project-specific utility**.

## 2. CJM

`internal/conversation/builder.go` defines a trivial `boolString(b bool) string` that returns `"true"` or `"false"`. This is exactly what `strconv.FormatBool` does. The `strconv` package is already imported in the file.

### North Star Summary

Four call-sites replaced with `strconv.FormatBool`, custom function removed. Zero new code.

## 3. Tests

### TC-01: existing conversation tests pass

**Given** `boolString` is replaced with `strconv.FormatBool`.
**When** `go test ./internal/conversation/...` is run.
**Then** all tests pass.

## Implementation

- Modified: `internal/conversation/builder.go` — replaced 4 `boolString()` calls with `strconv.FormatBool()`, removed custom function
