ID: FRD-20251116-approval-persistence-finish  
Date: 2025-11-16  
Owner: Spin  
Status: In Progress  

---

## Objective

Finish the approval persistence roadmap by hardening core policy behavior, adding deterministic end-to-end coverage, and closing all remaining quality, risk, and documentation items for “allow once / session / global” approvals.

This FRD builds on:
- `FRD-20251116-approval-persistence-F1-core-types-and-stores`
- `PROPOSAL-20251116-approval-persistence-scopes`
- `specs/roadmaps/approval-persistence/ROADMAP.md`

---

## Current State vs Target

### Implemented
- Core types and in-memory `PolicyStore` (`internal/security/policy.go`) with normalization and TTL janitor.
- `ApprovalService` integration with policy short-circuit and persistence on approve (`internal/security/approval.go`).
- Basic policy- and scope-oriented tests in `internal/security/approval_policy_test.go`.
- File-backed global store implementation (`internal/security/policy_file_store.go`).
- TUI approval dialog with scope handling and key preview (`internal/ui/overlay/approval.go`, `internal/ui/adapters/puretty.go`).
- ACP approval handler options extended for scopes and exposed via `RequestPermission`.
- CLI scaffolding for `spin approval` commands under `cmd/spin/approval.go`.

### Not Yet Complete (This FRD)
- E2E coverage for scopes and revocation across ACP → approval service → executor/tooling.
- Full unit/integration coverage (≥90%) for:
  - Memory and file-backed `PolicyStore` (expiry, locking, corrupted file handling).
  - `ApprovalService` policy hit/miss, TTL defaulting, and cancellation behavior.
  - ACP approval handler scope round-trips.
  - TUI key handling and key/TTL preview.
  - CLI policy commands.
- Race detector validation for policy flows.
- Feature flag and config behavior for `security.approval.persistence.*`.
- Docs and roadmap completion.

---

## Requirements

### R1: PolicyStore & ApprovalService Quality Gates
1. **Policy TTL behavior**
   - Session/global default TTLs applied when `ApprovalResponse.TTL` is nil and defaults are non-zero.
   - Expired policies are never returned from `Get`/`List` and are eventually removed by janitor.
2. **File-backed global store**
   - Atomic write via temp file + rename.
   - Advisory locking to guard concurrent writers.
   - Robust handling of missing, empty, and corrupted policy files with deterministic errors.
3. **ApprovalService policy paths**
   - `RequestApproval` short-circuits on session policy hit before global.
   - Approved responses with `ScopeOnce` never persist.
   - Revocation of a policy (via store `Delete`) causes subsequent calls to re-enter handler.
4. **Race-safety**
   - No data races in `PolicyStore` or `ApprovalService` policy paths under concurrent `Save`/`Get`/`Delete`/`RequestApproval`.

### R2: ACP Integration
1. **Options and behavior**
   - ACP approval handler surfaces options equivalent to:
     - Allow once
     - Allow for session
     - Allow always (global)
     - Deny
   - Initialize does not advertise any non-spec custom capabilities.
2. **Round-trip semantics**
   - ACP responses for approval carry `scope`, optional `ttl`, and `policyNote` and map correctly onto `ApprovalResponse`.
   - Persisted policies reflect ACP-selected scope and TTL.

### R3: TUI & CLI UX
1. **TUI approval dialog**
   - Keyboard bindings:
     - `A` → scope `once`
     - `S` → scope `session`
     - `G` → scope `global`
     - `D` → deny
     - `ESC` → cancel
   - Dialog shows normalized key (program, args, workdir) and effective TTL before confirmation.
   - Behavior is stable under window resize and timeout.
2. **CLI policy management**
   - `spin approval list [--scope session|global]`
   - `spin approval clear [--scope session|global]`
   - `spin approval revoke --program <p> --args "<a...>" [--workdir <w>] [--scope session|global]`
   - Commands use the same `PolicyKey` normalization as the approval flow.
   - Safe behavior on empty/nonexistent policy files and when revoking non-existent keys.

### R4: E2E Coverage (F7)
1. **Approve once**
   - First ACP-driven command approval with scope `once` executes.
   - Second identical command in same session requires a new approval.
2. **Approve session/global**
   - First approval with scope `session` or `global` creates a persisted policy.
   - Second identical command in same session/workspace bypasses approval via policy short-circuit.
3. **Revocation**
   - After `spin approval revoke` for a given key, the next identical command re-prompts for approval.
4. **Determinism**
   - E2E tests run with fixed commands, workdirs, and TTL behavior.
   - No timing flakiness; tests stable under `-race`.

### R5: Configuration & Feature Flag
1. Config keys:
   - `security.approval.persistence.enabled` (bool; default false).
   - `security.approval.persistence.global_path` (string path).
   - `security.approval.persistence.global_ttl` (duration).
   - `security.approval.persistence.session_ttl` (duration).
2. Behavior:
   - When `enabled=false`, approval flow behaves as today (no persistence, no policy short-circuit).
   - When `enabled=true`, policies are checked and persisted according to scope.
   - Config examples in `configs/example.yaml` and relevant `examples/config-*.yaml` demonstrate both modes.

### R6: Docs & Roadmap
1. Docs:
   - `docs/packages/protocol-acp.md` describes approval persistence semantics and ACP fields.
   - New `docs/approval-persistence.md` explains:
     - Conceptual model for once/session/global.
     - TUI shortcuts and key preview.
     - CLI list/revoke/clear flows.
     - Config flags and safety guidance.
2. Roadmap:
   - All remaining unchecked items in `specs/roadmaps/approval-persistence/ROADMAP.md` are set to complete once requirements R1–R5 are met.

---

## Non-Goals
- Pattern-based or fuzzy policy matching.
- Cross-workspace global storage beyond the configured workspace file.
- Remote/clustered stores or shared policy services.

---

## Acceptance Criteria

1. New and updated tests achieve ≥90% coverage for approval persistence code paths in:
   - `internal/security` (policy + approval).
   - ACP approval path.
   - TUI approval overlay and adapter.
   - `spin approval` CLI commands.
2. E2E tests in `tests/e2e` exercise:
   - Once/session/global behavior.
   - Revocation and re-prompt.
   - Feature-flag enabled operation.
3. `go test ./...`, `go test -race ./...`, and `go test -cover ./...` all pass.
4. `make lint`, `make deadcode`, and complexity analysis are clean for changed files.
5. Roadmap and docs are updated and linked from this FRD.


