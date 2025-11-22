# FRD: Approval Persistence – F1 Core Types and Stores

ID: FRD-20251116-approval-persistence-F1-core-types-and-stores  
Date: 2025-11-16  
Owner: Spin  
Status: Ready for implementation

---

## Objective
Introduce approval scope to security types and add a concurrency-safe PolicyStore with deterministic command normalization. This enables later features to persist “Allow once/session/global” decisions and to short-circuit future approvals.

---

## Requirements
- Extend `internal/security.ApprovalResponse` with:
  - `Scope string` // "once" | "session" | "global"
  - `TTL *time.Duration`
  - `PolicyNote string`
- Define and implement:
  - `PolicyKey` normalization: `(program, args, workdir)` with exact-match semantics
  - `Policy` struct (versioned JSON shape)
  - `PolicyStore` interface and in-memory implementation with TTL and janitor
  - Optional file-backed global store to come in a later feature (F1 implements the in-memory baseline)

---

## Non-Goals
- ApprovalService integration, ACP endpoints, and TUI changes (covered in later features).
- Pattern-based matching; only exact-match normalization in F1.

---

## Design
### Data Types
- `PolicyKey`:
  - `Program string`
  - `Args []string`
  - `WorkDir string`
  - Normalization rules: trim whitespace, collapse internal spaces for args, keep exact values otherwise.
- `Policy`:
  - `Version string` (e.g., "1")
  - `Scope string` ("session"|"global")
  - `Key PolicyKey`
  - `Decision string` (initially "allow")
  - `PolicyNote string`
  - `CreatedAt time.Time`
  - `ExpiresAt *time.Time`
  - `Meta map[string]string` (optional)

### Store Interface
```
type PolicyStore interface {
    Save(ctx context.Context, p Policy) error
    Get(ctx context.Context, key PolicyKey, scope string) (Policy, bool, error)
    List(ctx context.Context, scope string) ([]Policy, error)
    Delete(ctx context.Context, key PolicyKey, scope string) (bool, error)
    Clear(ctx context.Context, scope string) (int, error)
}
```

### Concurrency & TTL
- RWMutex protection.
- TTL enforcement on read and via periodic janitor goroutine (configurable interval).

---

## Test Plan
- Unit tests:
  - Normalization: various arg whitespace cases; workdir pass-through; program trimming
  - Store: Save/Get/List/Delete/Clear; expiry path with TTL
  - Concurrency smoke: parallel Save/Get
- Coverage ≥90% for new code.

---

## Acceptance
- Types and store compile.
- Tests pass; linter clean.









