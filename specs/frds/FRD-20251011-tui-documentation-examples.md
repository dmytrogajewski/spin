# FRD-20251011: TUI Documentation & Examples

**Date:** 2025-10-11
**Author:** Spin Agent
**Phase:** 7.3 Documentation & Examples
**Status:** In Progress
**Roadmap:** [ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 7.3

---

## 1. Context

Phase 7.3 of the TUI implementation roadmap requires comprehensive user-facing documentation and working examples. All technical components (Phases 1-6, 7.1-7.2) are complete and tested. Users need clear documentation to understand and use the TUI.

**Dependencies:**
- ✅ Phase 1: Terminal Control Infrastructure (COMPLETED)
- ✅ Phase 2: Prompt Subsystem (COMPLETED)
- ✅ Phase 3: Output System (COMPLETED)
- ✅ Phase 4: Block System (COMPLETED)
- ✅ Phase 5.1: PureTTY Adapter (COMPLETED)
- ✅ Phase 6.1: Block Timeline UI Integration (COMPLETED)
- ✅ Phase 6.2: Command Palette Overlay (COMPLETED)
- ✅ Phase 7.1: E2E TUI Tests (COMPLETED)
- ✅ Phase 7.2: Performance Validation (COMPLETED)

---

## 2. Requirements

### 2.1 Documentation (`docs/tui.md`)

**Must include:**

1. **Overview**
   - What is the Spin TUI
   - Key features and design principles
   - Factory Droid philosophy (native scrollback)

2. **Architecture**
   - Component diagram
   - Package structure and responsibilities
   - Data flow (user input → blocks → rendering)

3. **Usage Guide**
   - Starting the TUI
   - Basic interaction (typing, entering commands)
   - Timeline navigation
   - Block operations

4. **Keymap Reference**
   - Input mode keys
   - Timeline navigation keys
   - Filter mode keys
   - Command palette keys
   - Global shortcuts

5. **Block Types**
   - Overview of all 9 block types
   - Visual examples of each
   - When each type appears

6. **Advanced Features**
   - Command palette (Ctrl-P)
   - Filtering blocks (/)
   - Block actions (copy, save, rerun)
   - Status indicators

7. **Troubleshooting**
   - Terminal compatibility issues
   - SSH/tmux edge cases
   - Unicode/emoji display
   - Cursor visibility issues
   - Performance degradation

8. **References**
   - Links to package docs
   - Links to FRDs
   - Links to examples

### 2.2 Examples

All examples must:
- Compile without errors
- Run successfully (verified with `go run`)
- Include inline comments explaining key concepts
- Follow Go best practices
- Be self-contained (no external dependencies beyond Spin packages)

**Required examples:**

#### 1. `examples/tui-demo/main.go` - Minimal TUI Demo

**Purpose:** Show simplest possible TUI usage

**Features:**
- Initialize PureTTY adapter
- Print a few lines
- Accept user input
- Clean shutdown

**Lines:** ~50-80

---

#### 2. `examples/tui-streaming/main.go` - Streaming Demo

**Purpose:** Demonstrate streaming chunks (simulated LLM response)

**Features:**
- Print static lines
- Stream chunks with delay (simulate LLM tokens)
- Show coalescing behavior
- Use context cancellation

**Lines:** ~80-120

---

#### 3. `examples/tui-blocks/main.go` - Interactive Block Demo

**Purpose:** Showcase all 9 block types

**Features:**
- Create timeline with sample blocks of each type:
  - EXECUTE (command execution)
  - PLAN (task checklist)
  - READ (file preview)
  - GREP (search results)
  - APPLY_PATCH (diff)
  - SUMMARY (changeset summary)
  - TESTING (test results)
  - NOTICE (system message)
  - ERROR (error message)
- Demonstrate navigation
- Demonstrate collapse/expand
- Demonstrate filtering

**Lines:** ~200-300

---

### 2.3 README Updates

**Add section:**
```markdown
## Terminal UI (TUI)

Spin features a native-scrollback terminal UI with block-based timeline rendering.

See [docs/tui.md](docs/tui.md) for complete documentation.

**Examples:**
- [Minimal TUI](examples/tui-demo/)
- [Streaming](examples/tui-streaming/)
- [Block Types](examples/tui-blocks/)
```

---

## 3. Acceptance Criteria

### 3.1 Documentation

- [ ] `docs/tui.md` exists and covers all required sections
- [ ] All keymap shortcuts documented
- [ ] All 9 block types explained with examples
- [ ] Troubleshooting section addresses common issues
- [ ] Links to package docs, FRDs, and examples work
- [ ] Markdown renders correctly on GitHub
- [ ] No spelling or grammar errors (use spell checker)

### 3.2 Examples

- [ ] `examples/tui-demo/main.go` compiles and runs
- [ ] `examples/tui-streaming/main.go` compiles and runs
- [ ] `examples/tui-blocks/main.go` compiles and runs
- [ ] All examples include README.md explaining usage
- [ ] All examples follow Go conventions (gofmt, golint clean)
- [ ] Examples demonstrate intended features clearly

### 3.3 Quality Gates

- [ ] All examples pass `make lint` (zero errors)
- [ ] All examples build with `go build`
- [ ] `README.md` updated with TUI section and example links
- [ ] `ROADMAP.md` updated to mark Phase 7.3 as COMPLETED
- [ ] No broken links in documentation

---

## 4. Implementation Plan

### 4.1 Phase 1: Documentation Structure (Step 3)

**Tasks:**
1. Create `docs/tui.md` skeleton with sections
2. Write Overview section
3. Write Architecture section with component diagram
4. Write Usage Guide with common workflows

**Time estimate:** 1-2 hours

---

### 4.2 Phase 2: Keymap & Block Reference (Step 4)

**Tasks:**
1. Document complete keymap table
2. Document all 9 block types with visual examples
3. Add Advanced Features section
4. Add Troubleshooting section with common issues

**Time estimate:** 1-2 hours

---

### 4.3 Phase 3: Examples (Steps 5-7)

**Tasks:**
1. Create `examples/tui-demo/` with minimal example
2. Create `examples/tui-streaming/` with streaming demo
3. Create `examples/tui-blocks/` with all block types
4. Add README.md to each example directory
5. Test all examples compile and run

**Time estimate:** 2-3 hours

---

### 4.4 Phase 4: Integration & QA (Steps 8-12)

**Tasks:**
1. Update main `README.md` with TUI section
2. Run all examples and verify output
3. Run `make lint` on examples
4. Fix any issues found
5. Update `ROADMAP.md` to mark Phase 7.3 complete
6. Review all documentation for completeness

**Time estimate:** 30 minutes

---

## 5. Testing Strategy

### 5.1 Documentation Testing

**Manual review:**
- Spell check all markdown files
- Verify all internal links resolve
- Verify all code snippets are syntactically valid
- Check formatting renders correctly on GitHub

**Tools:**
- `markdown-link-check` (optional)
- GitHub markdown preview
- Spell checker (VS Code, etc.)

---

### 5.2 Example Testing

**Compile check:**
```bash
cd examples/tui-demo && go build
cd examples/tui-streaming && go build
cd examples/tui-blocks && go build
```

**Run check:**
```bash
# Each example should run without errors
go run examples/tui-demo/main.go
go run examples/tui-streaming/main.go
go run examples/tui-blocks/main.go
```

**Lint check:**
```bash
make lint
# Examples should have zero lint errors
```

---

### 5.3 Integration Testing

**Verify:**
- README links work
- All cross-references in docs resolve
- Examples demonstrate documented features
- Troubleshooting section addresses real issues from testing

---

## 6. Design Decisions

### 6.1 Documentation Format

**Decision:** Use GitHub-flavored markdown with tables and code blocks

**Rationale:**
- Standard format for Go projects
- Renders well on GitHub
- Easy to read in terminal (with `cat`, `less`, etc.)
- Supports syntax highlighting

---

### 6.2 Example Structure

**Decision:** Each example in separate directory with `main.go` and `README.md`

**Rationale:**
- Self-contained (can `cd` and `go run`)
- Clear separation of concerns
- Easy to copy as starter template
- Follows Go project conventions

---

### 6.3 Keymap Documentation Format

**Decision:** Use markdown table with columns: Key | Mode | Action | Notes

**Rationale:**
- Scannable reference
- Easy to search (Ctrl-F)
- Clear mode context

**Example:**
```markdown
| Key | Mode | Action | Notes |
|-----|------|--------|-------|
| Enter | Input | Submit line | Send to agent |
| Ctrl-C | Any | Cancel/Exit | Cancels input or exits |
| PgUp | Timeline | Scroll up one page | Viewport-aware |
```

---

### 6.4 Block Type Examples

**Decision:** Show both code snippet and rendered output

**Rationale:**
- Code shows how to create block
- Rendered output shows visual appearance
- Both are needed for complete understanding

**Format:**
````markdown
### EXECUTE Block

**Purpose:** Shell command execution

**Code:**
```go
block := blocks.NewBlock(blocks.BlockTypeExecute)
block.Title = "Run tests"
block.Body = "=== RUN TestFoo\n--- PASS: TestFoo (0.00s)\nPASS"
meta := &blocks.ExecuteMeta{Command: "go test ./...", ExitCode: ptr.Int(0)}
blocks.SetExecuteMeta(block, meta)
```

**Rendered output:**
```
│ ▐EXECUTE▌  Run tests (cmd: "go test ./...")  [impact: medium]
│   === RUN TestFoo
│   --- PASS: TestFoo (0.00s)
│   PASS
│ ✓ [exit: 0] [dur: 0.1s]
```
````

---

## 7. Risks & Mitigations

### 7.1 Documentation Staleness

**Risk:** Documentation becomes outdated as code evolves

**Mitigation:**
- Link to package godocs for API details
- Focus on concepts over implementation
- Add "Last Updated" timestamp
- Plan documentation review in Phase 8.2

---

### 7.2 Examples Break on Code Changes

**Risk:** Refactoring breaks example code

**Mitigation:**
- Keep examples simple and focused
- Test examples as part of CI (future)
- Add examples to `make lint` targets
- Document required Go version and dependencies

---

### 7.3 Troubleshooting Section Incomplete

**Risk:** Users encounter issues not documented

**Mitigation:**
- Base troubleshooting on Phase 7.1 test findings
- Add placeholder for future issues
- Encourage users to file issues for undocumented problems
- Plan dogfooding feedback collection (Phase 8.2)

---

## 8. Dependencies

**External:**
- None (all Spin internal packages)

**Internal packages:**
- `internal/ui/adapters` (PureTTY)
- `internal/ui/blocks` (Block, Timeline, Renderer)
- `internal/ui/output` (Printer, CoordinatedWriter)
- `internal/ui/prompt` (Model, Renderer)
- `internal/ui/term` (TTY, Keyboard)

---

## 9. Open Questions

**Q1:** Should examples include MCP integration demos?

**A1:** No. MCP integration is out of scope for Phase 7.3. Focus on TUI primitives only. MCP examples can be added in Phase 7.4 (Core Integration).

---

**Q2:** Should documentation include asciinema recordings?

**A2:** Deferred to Phase 8.2 (Final QA). Asciinema requires terminal recording setup and hosting. Focus on text/code examples for now.

---

**Q3:** Should troubleshooting include performance tuning tips?

**A3:** Yes, but minimal. Reference `docs/performance.md` for detailed benchmarks. Include only user-actionable tips (e.g., terminal emulator choice, SSH compression).

---

## 10. Success Metrics

**Quantitative:**
- [ ] 3 working examples (100% compile, run without errors)
- [ ] `docs/tui.md` ≥1500 words
- [ ] Keymap documents ≥15 shortcuts
- [ ] All 9 block types documented with examples
- [ ] Zero broken links in documentation
- [ ] Zero lint errors in examples

**Qualitative:**
- [ ] Documentation is clear and scannable
- [ ] Examples demonstrate real-world usage patterns
- [ ] Troubleshooting section addresses common issues
- [ ] README effectively introduces TUI to new users

---

## 11. References

**Roadmap:**
- [ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 7.3

**Spec:**
- [tui-new.md](../tui-implementation/tui-new.md) Section 14 (Keymap)

**Related FRDs:**
- All Phase 1-6 FRDs (architecture reference)
- [FRD-20251010-e2e-tui-tests.md](./FRD-20251010-e2e-tui-tests.md) (test insights)
- [FRD-20251011-performance-virtualization-validation.md](./FRD-20251011-performance-virtualization-validation.md)

**Package Docs:**
- [docs/packages/ui-blocks.md](../../docs/packages/ui-blocks.md)
- [docs/packages/ui-output.md](../../docs/packages/ui-output.md)
- [docs/performance.md](../../docs/performance.md)

**Examples Inspiration:**
- `examples/basic-conversation/`
- `examples/custom-tool/`

---

## 12. Implementation Checklist

Following AGENTS.md 14-step workflow:

- [x] **Step 1:** Read AGENTS.md and instructions
- [x] **Step 2:** Take roadmap item (Phase 7.3)
- [x] **Step 3:** Read all docs/ (completed)
- [x] **Step 4:** Write FRD (this document)
- [ ] **Step 5:** Re-read FRD to align scope
- [ ] **Step 6:** Write tests (N/A - documentation task)
- [ ] **Step 7:** Implement (write docs and examples)
- [ ] **Step 8:** Analyze with uast/herr (examples only)
- [ ] **Step 9:** Run `make lint`
- [ ] **Step 10:** Fix analysis findings
- [ ] **Step 11:** Iterate until all examples compile
- [ ] **Step 12:** Close roadmap item
- [ ] **Step 13:** Update docs (this IS the docs task)
- [ ] **Step 14:** Update AGENTS.md if needed (unlikely)

---

## 13. Timeline Estimate

**Total time:** 4-6 hours

**Breakdown:**
- FRD writing: 30 minutes ✅
- `docs/tui.md` writing: 2-3 hours
- Examples coding: 2-3 hours
- Testing & QA: 30 minutes
- README update & roadmap: 15 minutes

**Completion target:** 2025-10-11 (same day)

---

## 14. Definition of Done

Phase 7.3 is COMPLETE when:

- [x] FRD written and approved
- [ ] `docs/tui.md` complete with all required sections
- [ ] All 3 examples compile and run successfully
- [ ] All examples have README.md
- [ ] `README.md` updated with TUI section
- [ ] `make lint` clean (zero errors)
- [ ] `ROADMAP.md` updated to mark Phase 7.3 COMPLETED
- [ ] All documentation spell-checked and reviewed
- [ ] All links verified

---

**End of FRD**
