---
name: generalize
description: Find reusable code and document generalization opportunities
---

# Agent Instructions: Generalize — Find Reusable Code

You are a code analysis agent. Your goal is to find all potentially reusable code and document it for generalization.

---

## Workflow (micro-iterations)

### 1. Iterate Over Every Source File (Except Tests)

Scan all `*.go` files excluding `*_test.go`.

### 2. For Each File, Analyze Every Function

For each function ask: **is it reusable?**

**Criteria:**


- **(a) Cross-package potential** — it does something that could be needed in other packages if we generalize the function signature.
- **(b) Already generic** — it is already mostly generic; to make it fully generic we just have to rewrite it with Go generics (`type parameters`).
- **(c) Decomposable** — it could be decomposed into smaller generic functions.
- **(d) Replaceable** — it could be removed entirely if another known function from `specs/ref/LIST.md` is used instead.

### 3. Take Action

| Finding | Action |
|---------|--------|
| **(a), (b), (c)** | Document into `specs/ref/LIST.md` |
| **(d)** | Add info into `specs/ref/LIST.md` near the corresponding function that could replace it |

**Update `specs/ref/LIST.md` after every file — do NOT batch.**

### 4. Build the Spec

After completing all files:

1. Read `specs/ref/LIST.md`.
2. Write `specs/ref/SPEC.md` — organize findings into **clusters by problem domain**.

---

## LIST.md Format

```markdown
## Dedup opportunities

1. {path/to/package}

   Function: {function name}
   Position: {file}:{line}:{col}
   Findings: {What did you find during analysis}
   Could replace:
     - path/to/package:999:99:FunctionName
     ...

...
```

---

## Progress Tracking Graph

Track your progress as a graph. Always do **one movement at a time**.

```
[ {pkg1} [x] ] -> [ {pkg2} [✅] ] -> [ {pkg2.1} [🚫] ]
                                  -> [ {pkg2.2} [🚫] ]
                -> [ {pkg3} [⌛] ] -> [ {pkg3.1} [ ] ]
```

**Legend:**

| Symbol | Meaning |
|--------|---------|
| ✅ | Done, completed, or checked hypothesis |
| 🚫 | Canceled |
| ⌛ | Next / current work |
| [ ] | Not complete, not checked |

---

## Rules

1. **Micro-loops** — update `LIST.md` after every file, not at the end.
2. **One graph movement at a time** — do not skip ahead.
3. **Be specific** — include exact file paths, line numbers, and function signatures.
4. **Generics focus** — when criterion (b) applies, note which type parameters would be needed.
5. **No false positives** — only document genuinely reusable code, not every helper.
