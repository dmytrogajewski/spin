# JOURNEY-R2: Tool Block Visibility for All Tool Types

## Overview

Verify that every tool type renders its expected TUI block header in exec mode output.
The mapper (`internal/tui/mapper.go`) has distinct code paths per tool name — each must
be exercised by a fixture test.

## Phases

### Phase 1: write_file → WRITE block (TV-3)
- Fixture: read_file (to satisfy FileTracker) → write_file → response
- Assert: block visible, file created on disk

### Phase 2: edit_file → block visible (TV-4)
- Fixture: read_file (to satisfy FileTracker) → edit_file → response
- Setup: workDir with `src.txt` containing "foo"
- Assert: block visible, file content changed to "bar"

### Phase 3: list_directory → EXECUTE block (TV-5)
- Fixture: list_directory → response
- Assert: EXECUTE block visible, file name in output

### Phase 4: file_search → GREP block (TV-6)
- Fixture: file_search → response
- Assert: file match in output

## Tests

- `TestFixture_WriteFileBlockVisible` — TV-3
- `TestFixture_EditFileBlockVisible` — TV-4
- `TestFixture_ListDirectoryBlockVisible` — TV-5
- `TestFixture_FileSearchBlockVisible` — TV-6

## Implementation

**Files created:**
- `tests/e2e/fixtures/write_file_block.jsonl` — TV-3 fixture (read→write→response)
- `tests/e2e/fixtures/edit_file_block.jsonl` — TV-4 fixture (read→edit→response)
- `tests/e2e/fixtures/list_directory_block.jsonl` — TV-5 fixture
- `tests/e2e/fixtures/file_search_block.jsonl` — TV-6 fixture

**Files modified:**
- `tests/e2e/fixture_exec_test.go` — 4 new test functions

**Roadmap:** [specs/testing/ROADMAP.md](../testing/ROADMAP.md) — R2 checklist complete

**Finding:** write_file and edit_file require a prior read_file call to satisfy
FileTracker.AssertFresh(). Fixtures use a 3-turn pattern: read→mutate→response.
file_search results depend on workspace indexing; test asserts tool execution
rather than specific search results.
