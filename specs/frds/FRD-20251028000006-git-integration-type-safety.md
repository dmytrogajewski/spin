# FRD-20251028000006: Git Integration Type Safety

## Metadata
- **FRD ID**: FRD-20251028000006
- **Title**: Git Integration Type Safety
- **Created**: 2025-10-28
- **Status**: Approved
- **Related Phase**: 6.2 (Git Integration)
- **Parent Roadmap**: specs/ifacesroadmap.md

## Overview

Replace `map[string]interface{}` in `GetContextInfo()` with a strongly-typed `GitContextInfo` struct in the git integration package.

## Files Affected
- `internal/git/integration.go` - Define GitContextInfo and update GetContextInfo
- `internal/manager/manager.go` - Update addGitContext to use struct fields
- `internal/git/git_test.go` - Add tests for GitContextInfo

## Problem Statement

The `GetContextInfo()` method returns git context as `map[string]interface{}`, but the actual structure is well-defined. Using `map[string]interface{}` forces consumers to remember exact key names, perform manual type assertions, and lacks compile-time type safety.

## Proposed Solution

Define `GitContextInfo` struct with typed fields and update `GetContextInfo()` to return it.

## Benefits

1. **Type Safety**: Compile-time type checking
2. **IDE Support**: Autocomplete for struct fields
3. **Consistent**: Matches ShellContextInfo pattern (Phase 4.2)

## Interface{} Elimination Count

**Before**: 3 occurrences
**After**: 0 occurrences
**Net Reduction**: -3

## Success Criteria

- [ ] `GitContextInfo` struct defined
- [ ] `GetContextInfo()` returns `GitContextInfo`
- [ ] `addGitContext` uses struct fields
- [ ] All tests pass (90%+ coverage)
- [ ] `make lint` passes
