# JOURNEY-extract-atomic-write: Extract Atomic File Write to internal/storage

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 2.4 — Extract Atomic File Write to `internal/storage.AtomicWriteFile`
- Cluster: 2 (SPEC.md) | LIST.md findings 1, 10, 22, 48

## 1. Journey

When **a developer persisting data to disk** I want to **call a single, tested `storage.AtomicWriteFile` function** so I can **avoid maintaining four identical temp-file + rename patterns across storage, config, memory, and playbook packages**.

## 2. CJM

Four production packages independently implement the atomic write pattern:
- `internal/storage/store.go:Save` — `os.WriteFile` + `os.Rename` with `.tmp` suffix
- `internal/config/mcp_manager.go:writeConfig` — `os.WriteFile` + `os.Rename` with `.tmp` suffix
- `internal/memory/persistent.go:Put` — `os.WriteFile` + `os.Rename` with `.tmp` suffix
- `internal/ace/playbook/storage.go:Save` — `os.CreateTemp` + `os.Rename` with unique temp name

All follow the same pattern: write to temp file, rename to final path, clean up temp on error.

### Phase 1: Create AtomicWriteFile

**Actions:**
1. Add `AtomicWriteFile(path string, data []byte, perm os.FileMode) error` to `internal/storage`.
2. Write comprehensive unit tests.

**Success Signal:** All edge cases covered: success, directory creation, rename cleanup.

### Phase 2: Migrate call-sites

**Actions:**
1. Refactor `FileStore.Save` to use `AtomicWriteFile` internally.
2. Update `config.MCPConfigStore.writeConfig` to use `storage.AtomicWriteFile`.
3. Update `memory.PersistentStore.Put` to use `storage.AtomicWriteFile`.
4. Update `ace/playbook.Save` to use `storage.AtomicWriteFile`.

**Success Signal:** All tests pass, no duplicate atomic write patterns remain.

### Phase 3: Verification

**Actions:** `go vet`, `make lint`, `go test -race ./internal/storage/... ./internal/config/... ./internal/memory/... ./internal/ace/playbook/...`

**Success Signal:** Zero issues, all tests green.

### North Star Summary

A single `storage.AtomicWriteFile` function serves all atomic file persistence needs. No duplicate patterns remain.

## 3. Tests

### TC-01: successful atomic write creates file with correct content
### TC-02: successful atomic write applies correct permissions
### TC-03: no temp file left on success
### TC-04: temp file cleaned up on rename failure (simulated via read-only dir)
### TC-05: error when parent directory does not exist
### TC-06: overwrite existing file atomically
### TC-07: empty data writes empty file

## Implementation

- Created: `internal/storage/atomic.go` — `AtomicWriteFile`, `DefaultFilePerm`
- Created: `internal/storage/atomic_test.go` — comprehensive unit tests
- Modified: `internal/storage/store.go` — `FileStore.Save` uses `AtomicWriteFile`
- Modified: `internal/config/mcp_manager.go` — `writeConfig` uses `storage.AtomicWriteFile`
- Modified: `internal/memory/persistent.go` — `Put` uses `storage.AtomicWriteFile`
- Modified: `internal/ace/playbook/storage.go` — `Save` uses `storage.AtomicWriteFile`
