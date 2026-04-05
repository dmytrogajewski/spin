---
name: evolve
description: Full pipeline — generalize codebase, create roadmap from findings, then implement all features
---

# Agent Instruction: Evolve — Generalize → Roadmap → Implement All

You are an orchestration agent that runs three phases sequentially. Each phase must fully complete before the next begins.

[ ALL GIT OPERATIONS PROHIBITED . NEVER USE GIT AT ANY COST, DONT call git ]

---

## Phase 1: Generalize

Run the `/generalize` skill exactly as defined. This will:

1. Scan all `*.go` files (excluding tests)
2. Analyze every function for reuse potential (cross-package, generics, decomposable, replaceable)
3. Produce `specs/ref/LIST.md` (updated after every file)
4. Produce `specs/ref/SPEC.md` (clustered by problem domain)

**Exit criterion:** Both `specs/ref/LIST.md` and `specs/ref/SPEC.md` exist and are complete.

---

## Phase 2: Roadmap

Run the `/roadmap` skill over the spec produced in Phase 1 (`specs/ref/SPEC.md`). This will:

1. Analyze the codebase for existing implementations
2. Create a progressive decomposition where every step is independently valuable and testable
3. Produce a roadmap at `specs/ref/ROADMAP.md`

Each roadmap item must be scoped to a single user journey with DoD/DoR/Description.

**Exit criterion:** `specs/ref/ROADMAP.md` exists with numbered, testable items.

---

## Phase 3: Implement ALL Features

Iterate through **every** item in the roadmap produced in Phase 2. For each item, run the `/implement` skill workflow:

1. Read the roadmap item
2. Write journey document → `specs/journeys/JOURNEY-{id}.md`
3. Write tests (min 90% coverage)
4. Write implementation
5. Run `go vet ./...`
6. Run `make lint`
7. Fix all issues — no nolint, no linter config changes
8. Iterate until all tests pass
9. Close the roadmap item in `specs/ref/ROADMAP.md`
10. Update documentation in `docs/`
11. Update `AGENTS.md` if needed
12. Add traceability links (journey → implementation, roadmap → journey, test → journey)

Use the Small Change Fast Path when applicable (< 15 lines, no new public API, no architectural impact).

Use micro-TDD for all non-trivial changes.

**Exit criterion:** Every roadmap item is marked done with evidence links. All tests pass. Lint is clean.


**Remove specs/ref/ dir after completing roadmap. Add summary of session into LOG.md**
---

## Progress Tracking

Maintain a progress summary after each phase transition:

```
## Evolve Progress

### Phase 1: Generalize
Status: {✅ | ⌛ | [ ]}
Files analyzed: {n}/{total}
Reuse candidates found: {count}

### Phase 2: Roadmap
Status: {✅ | ⌛ | [ ]}
Roadmap items created: {count}

### Phase 3: Implement
Status: {✅ | ⌛ | [ ]}
Items completed: {n}/{total}
Current item: {description}
```

---

## Rules

1. **Sequential phases** — never start Phase 2 before Phase 1 completes, never start Phase 3 before Phase 2 completes.
2. **No skipping** — implement every roadmap item, not just the easy ones.
3. **No git operations** — destructive or otherwise.
4. **Quality gates** — every implementation must pass `go vet` and `make lint` with zero errors.
5. **Traceability** — every implemented feature must link back through journey → roadmap → spec.
6. **Respect AGENTS.md** — read and follow any project-level agent instructions.
