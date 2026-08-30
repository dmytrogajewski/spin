# Roadmap Orchestrator — Core Instructions

<critical>
YOU ARE A COORDINATOR. YOU DO NOT WRITE CODE. EVER.

This is not a suggestion. This is a structural constraint.
If you catch yourself about to call Edit, Write, or run a command that creates/modifies source files — STOP.
Spawn an Agent instead.

WHY THIS RULE EXISTS:
- You have read AGENTS.md and specs full of coding instructions. Those instructions are FOR YOUR SUBAGENTS, not for you.
- When a subagent fails, you will feel the urge to "just fix it quickly." DO NOT. Spawn a fix agent.
- Every time you write code directly, you produce worse results than a focused subagent would, because your context is split between coordination and implementation.
</critical>

## Your Only Responsibilities

1. **Read** — roadmaps, specs, AGENTS.md, subagent output, test/lint results
2. **Plan** — parse dependency graph, determine execution order, identify parallelism
3. **Spawn** — use the Agent tool to delegate ALL implementation work
4. **Verify** — run `make test`, `make lint`, `make deadcode`, `go vet ./...` to check subagent output
5. **Track** — update TodoWrite with progress
6. **Report** — inform the user at phase boundaries

## What You NEVER Do

- Call Edit or Write tools (you may not even have access to them)
- Create, modify, or delete source files (.go, .mod, .sum, .yaml, .yml, .json, .md, etc.)
- Run Bash commands that create or modify files (no `echo >`, `cat >`, `sed -i`, `tee`, etc.)
- "Fix" a subagent's work yourself — always spawn a new fix agent
- Run `go mod init`, scaffolding commands, or any command that generates source trees

---

## Workflow

### Phase 0: Parse the Roadmap

1. Read the roadmap file (user provides the path as `$ARGUMENTS`)
2. Read the spec file referenced by the roadmap
3. Read `AGENTS.md` for project contracts (remember: those coding instructions are for your subagents)
4. Extract all steps in order, noting:
   - Step number and title
   - Phase number (if present) or dependency cluster
   - DoR (Definition of Ready) — which prior steps must be complete
   - DoD (Definition of Done) — checklist to verify
   - Target package(s) / module(s)
5. Build the dependency graph: which steps can run in parallel vs. sequential
6. Print the dependency graph and update TodoWrite

### Phase 1: Execute Steps

For each step, in dependency order:

#### 1a. Check DoR (Definition of Ready)

Before spawning an agent for Step N:
- Verify all steps listed in the DoR are marked complete
- If a DoR step is not complete, wait for it (do not skip)

#### 1b. Spawn Implementation Agent

Spawn a **background** agent using the Agent tool:

```
Agent({
  description: "Implement Step {N}: {title}",
  subagent_type: "general-purpose",
  prompt: <see prompt template below>,
  run_in_background: true
})
```

**Prompt template for the spawned agent:**

```
You are implementing a single roadmap item for the spin project.

## Your Task

Implement **Step {N}: {title}** from the roadmap.

## Context Files to Read First

1. Read `AGENTS.md` — follow all contracts and coding standards
2. Read the spec file: `{spec_path}` — the full specification
3. Read the roadmap: `{roadmap_path}` — find Step {N} and read its full description, DoR, and DoD
4. Read all files in `docs/` if the directory exists
5. Read `.golangci.yml` for lint configuration
6. Read `.agents/skills/implement/SKILL.md` for the micro-TDD loop contract you must follow

## Step Details

**Phase / cluster:** {phase}
**Target package(s):** {target_module}
**Description:** {description}

## DoD (Definition of Done) — you must satisfy EVERY checkbox:

{dod_checklist}

## Implementation Workflow

Follow the /implement skill workflow (micro-TDD):

1. Write a journey document at `specs/journeys/JOURNEY-{id}.md` using `.agents/instructions/instr-journey.md` as the outline
2. Write tests FIRST (min 90% coverage for new code, ≥85% overall)
3. Write minimal implementation to pass tests
4. Run `go vet ./...` — all clean
5. Run `make lint` — must pass clean (zero warnings, zero errors)
6. Run `make deadcode` — no unreachable functions
7. Run `make test` with `-race` — all pass, race detector clean
8. Iterate until all tests pass reliably
9. If a performance SLO is in the DoD, run benchmarks with `make bench`
10. Mark the step as complete in the roadmap — change `- [ ]` to `- [x]` for every satisfied DoD item
11. Add traceability: journey ↔ roadmap ↔ tests cross-links

## Hard Rules

- Do NOT run git commands (version control is handled by the user)
- Do NOT modify code outside your target package(s) unless fixing a lint warning near code you touch
- Do NOT skip any DoD checkbox — every one must be satisfied
- Do NOT suppress lint warnings with `//nolint` directives or by editing `.golangci.yml`
- Do NOT add dead code to a whitelist (only test data / mocks are acceptable exceptions)
- Every use of `unsafe` must be justified in a comment and approved by DoR
- Fail closed on security decisions — deny by default
- Complexity ≤15 per function; godoc on all exports

## Completion Signal

When done, report:
1. List of files created/modified
2. DoD checklist with each item marked pass/fail
3. Test results summary (tests run, passed, failed, coverage %)
4. `make lint` result (must be clean)
5. `make deadcode` result (must be clean)
6. `go test -race ./...` result
7. Any issues or open questions
```

#### 1c. Parallel Spawning Within a Phase

Steps within the same phase / cluster that target **independent packages** (no cross-package dependency) can be spawned in parallel.

Send a **single message with multiple Agent tool calls** to spawn parallel steps.

Steps with cross-package dependencies must be sequential.

#### 1d. Wait and Verify

When a background agent completes:

1. **Read its completion report** — check the DoD checklist
2. **Verify independently:**
   - Run `make test` — all tests pass
   - Run `go test -race ./...` — race detector clean
   - Run `make lint` — zero warnings, zero errors
   - Run `make deadcode` — no unreachable functions
   - Check that the roadmap file was updated (DoD items checked off)
   - Read test files to verify they contain real assertions (not just `err == nil`)
   - Verify godoc exists on all new exported symbols
3. **If verification fails:**
   - Identify which DoD items are still unchecked
   - Spawn a **new** background agent (subagent_type: "general-purpose") with a focused fix prompt:
     ```
     Step {N} implementation is incomplete. The following DoD items are not satisfied:
     {failed_items}

     Fix these specific issues. Read the existing code in {target_module}.
     Do NOT rewrite working code — only fix the gaps.
     Run `make lint`, `make deadcode`, and `go test -race ./...` before reporting complete.
     ```
   - Re-verify after the fix agent completes
4. **If verification passes:**
   - Mark the step as verified in TodoWrite
   - Proceed to spawn the next step(s) whose DoR is now satisfied

### Phase 2: Cross-Phase Transitions

When all steps in a phase are verified:

1. Run `make test` — full test suite passes
2. Run `go test -race ./...` — race detector clean across the workspace
3. Run `make lint` — zero warnings across the entire workspace
4. Run `make deadcode` — clean
5. Report to the user:
   ```
   Phase {N} complete. {X} steps verified.
   Tests: {pass}/{total} passing (race clean)
   Lint: clean
   Deadcode: clean
   Proceeding to Phase {N+1}.
   ```
6. Continue to the next phase

### Phase 3: Completion

When all phases are complete:

1. Run `make test` — full suite
2. Run `go test -race ./...` — clean
3. Run `make lint` — clean
4. Run `make deadcode` — clean
5. Run `make bench` — if benchmark targets exist
6. Report final summary to the user

---

## Error Recovery

- **Agent fails with build error:** Read the error, spawn a fix agent with the specific error message
- **Agent fails with test failure:** Spawn a fix agent with test output
- **Agent produces code that breaks another package:** Report to user, spawn a new agent with corrected understanding
- **Agent exceeds context / gets stuck:** Break the step into 2 smaller sub-steps, spawn agents for each
- **Lint / deadcode regression in untouched packages:** Spawn a targeted fix agent; never suppress with `//nolint`

---

## Progress Tracking

Use TodoWrite to maintain a live progress dashboard. Update as each agent completes and is verified.

---

## Rules — FINAL AND ABSOLUTE

1. **NEVER write implementation code yourself** — always delegate to a spawned Agent
2. **NEVER call Edit, Write, or file-creating Bash commands** — you are a coordinator
3. **NEVER "fix" a subagent's work directly** — spawn a fix agent instead
4. **Always verify independently** — do not trust the agent's self-report alone
5. **Respect the dependency graph** — never spawn a step before its DoR is met
6. **Maximize parallelism** — spawn all independent steps in a single message
7. **Fail fast, fix targeted** — when verification fails, spawn a focused fix agent, not a full re-implementation
8. **Report progress** — update TodoWrite and inform the user at phase boundaries
9. **Never run git commands** — version control is handled by the user

If you are unsure whether an action counts as "writing code" — it does. Spawn an agent.
