# Tool Testing Roadmap — Fixture-Driven E2E Coverage

**Goal**: For every agent tool, run a real session that exercises it, record the fixture, analyze the trace for bugs, write tests that reproduce bugs, and fix them.

**Total tools**: 24 across 8 categories.

---

## Workflow (MANDATORY for every item)

For EACH roadmap item:

0. **Rebuild and install spin** — After any code change, run `make install` so that `spin` on `$PATH` is the latest build. Always use the installed `spin` binary (not `build/bin/spin`), so it picks up the user's default configuration from `~/.spin/spin.yaml`. **NEVER pass `--provider` or `--model` flags** — the user's config already has the correct LLM settings.
1. **Prepare workspace** — Create an isolated directory under `~/sources/testbed/<item>/` with seed files the agent needs.
2. **Run `spin exec`** with `--record-fixture` — Give the agent a task that forces it to use specific tools. Use `--auto-approve --timeout 3m`. Run from the testbed directory so spin uses it as workdir.
3. **Analyze the trace** — Read the full output. Look for:
   - Tool call failures (error responses, crashes, panics)
   - False positives (valid operations rejected — like truncation detection)
   - Silent data loss (tool succeeds but result is wrong)
   - Behavioral bugs (agent loops, wrong tool chosen, parameters ignored)
4. **Record bugs** — For each bug found, note the tool, input, expected vs actual behavior.
5. **Write reproduction test** — Use the recorded fixture or a minimal hand-crafted fixture to reproduce the bug in `tests/e2e/` or `internal/tools/`.
6. **Fix the bug** — Change production code. Verify the reproduction test now passes.
7. **Rebuild and install** — Run `make install` again after fixes.
8. **Run `go test ./... && make lint`** — Confirm no regressions.

**Fixture output directory**: `tests/e2e/fixtures/recorded/`

**IMPORTANT**: Always use the system-installed `spin` command, never a path to a dev binary. This ensures the user's `~/.spin/spin.yaml` configuration (provider, model, API keys) is used automatically.

---

## T-1: File Read + Write + Edit (Core File Trio)

> **Tools**: `read_file`, `write_file`, `edit_file`
>
> **Status**: ✅ DONE — No bugs found. All 3 tools worked correctly.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t1_file_trio.jsonl` (8 turns)
>
> **Evidence**:
> - `read_file` returned correct content across 6+ calls
> - `write_file` with Rust `'static` lifetime — no truncation false positive (fix from tetris trace verified)
> - `edit_file` exact-match replacement on `version = "0.1.0"` → `"0.2.0"` succeeded
> - FileTracker: write after read passed stale-read gate
> - Files on disk match expected content after session

### Session

```bash
mkdir -p ~/sources/testbed/t1-files && cd ~/sources/testbed/t1-files
echo 'fn greet(name: &str) { println!("Hello, {}", name); }' > lib.rs
echo '[package]\nname = "demo"\nversion = "0.1.0"' > Cargo.toml

spin exec "Read lib.rs. Then add a new function farewell(name: &str) that prints Goodbye. Then edit Cargo.toml to change version to 0.2.0. Finally read both files to confirm changes." \
  --auto-approve --timeout 3m \
  --record-fixture tests/e2e/fixtures/recorded/t1_file_trio.jsonl
```

### What to verify
- [ ] `read_file` returns correct content (not truncated, not summarized)
- [ ] `write_file` creates/overwrites file on disk — content matches
- [ ] `edit_file` applies exact replacement — old content gone, new content present
- [ ] FileTracker: `write_file` after `read_file` succeeds (stale-read gate)
- [ ] FileTracker: `edit_file` without prior `read_file` is rejected
- [ ] FileTracker: `write_file` to a file created by `shell_command` (not read) works
- [ ] OperationLog: write and edit operations are recorded for undo
- [ ] Truncation detection: no false positives on Rust/Go/Python code with lifetimes, runes, format strings

### Bugs to watch for
- Truncation false positive on single-quote languages (FIXED: T-spin trace)
- FileTracker rejects writes to files created externally (FIXED earlier)
- Edit fuzzy match ambiguity causing wrong replacement
- Large file write silently truncated by LLM max_tokens

---

## T-2: List Directory + File Search

> **Tools**: `list_directory`, `file_search`
>
> **Status**: ✅ DONE — No bugs found. Both tools worked correctly.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t2_dir_search.jsonl` (10 turns)
>
> **Evidence**:
> - `list_directory` on populated dirs returned files with types/sizes
> - `list_directory` on empty dir returned clear "Directory is empty" message
> - `list_directory` on nested paths (src/components, src/utils) worked
> - `file_search` found TypeScript files across directory tree
> - 10-turn fixture recorded: 7 list_directory calls, 4 file_search calls, 1 text summary

### Session

```bash
mkdir -p ~/sources/testbed/t2-search/src/components ~/sources/testbed/t2-search/src/utils ~/sources/testbed/t2-search/tests
echo 'export function Button() {}' > ~/sources/testbed/t2-search/src/components/Button.tsx
echo 'export function formatDate() {}' > ~/sources/testbed/t2-search/src/utils/date.ts
echo 'test("button renders", () => {})' > ~/sources/testbed/t2-search/tests/Button.test.tsx
echo 'node_modules/' > ~/sources/testbed/t2-search/.gitignore
mkdir -p ~/sources/testbed/t2-search/node_modules/react && echo '{}' > ~/sources/testbed/t2-search/node_modules/react/package.json

cd ~/sources/testbed/t2-search
spin exec "List the project structure. Then search for all TypeScript files. Then search for files containing 'Button'. Report what you found." \
  --auto-approve --timeout 2m \
  --record-fixture tests/e2e/fixtures/recorded/t2_dir_search.jsonl
```

### What to verify
- [ ] `list_directory` returns files with types and sizes
- [ ] `list_directory` on empty dir returns empty list (not error)
- [ ] `list_directory` handles nested paths correctly
- [ ] `file_search` respects `.gitignore` (no `node_modules/` results)
- [ ] `file_search` fuzzy matching finds partial names
- [ ] `file_search` with no matches returns empty result (not error)

### Bugs to watch for
- Empty directory listing confusing the agent ("No output")
- `.gitignore` not respected in file_search
- Unicode filenames causing issues

---

## T-3: Shell Command Execution

> **Tools**: `shell_command` (execute, get_environment, detect_shell)
>
> **Status**: ✅ DONE — No bugs found. All shell operations worked correctly.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t3_shell.jsonl` (7 turns)
>
> **Evidence**:
> - `node app.js` — stdout captured correctly ("hello from node")
> - Pipe chain `echo hello | tr a-z A-Z` — returned "HELLO"
> - `env | head -5` — returned 5 env vars via pipe
> - `nonexistent_command_xyz_123` — graceful "command not found" error (no crash)
> - Agent self-corrected after initial parallel tool call formatting issue

### Session

```bash
mkdir -p ~/sources/testbed/t3-shell && cd ~/sources/testbed/t3-shell
echo 'console.log("hello")' > app.js

spin exec "Run 'node app.js' and show the output. Then run 'echo \$SHELL' to detect the shell. Then run a command that lists environment variables. Then try running 'nonexistent_command_xyz' and handle the error gracefully." \
  --auto-approve --timeout 2m \
  --record-fixture tests/e2e/fixtures/recorded/t3_shell.jsonl
```

### What to verify
- [ ] Successful command returns stdout + exit code 0
- [ ] Failed command returns stderr + non-zero exit code (not crash)
- [ ] `get_environment` operation returns shell info
- [ ] `detect_shell` operation returns shell path
- [ ] Command with special characters (pipes, redirects) works
- [ ] Long-running command respects timeout
- [ ] Pipeline stages (blocklist + safety) run before execution

### Bugs to watch for
- Shell command with `&&` or `|` not parsed correctly
- Timeout not applied (command hangs forever)
- Blocklist false positive on safe commands
- exit code lost in output formatting

---

## T-4: Background Process Management

> **Tools**: `shell_command` (background), `list_processes`, `get_process_output`, `kill_process`
>
> **Status**: ✅ DONE — Design gap found and fixed: `start_process` tool added.
>
> **Journey**: [`specs/journeys/JOURNEY-T4.md`](../journeys/JOURNEY-T4.md)
>
> **Fixture**: `tests/e2e/fixtures/recorded/t4_background.jsonl` (partial — timed out before fix)
>
> **Findings & Fix**:
> - `BackgroundTaskManager.Start()` existed on the concrete type but was NOT exposed to tools.
> - **Fixed**: Added `TaskStarter` interface with `Start(ctx, command, workDir)`.
> - **Fixed**: Created `start_process` tool in `internal/tools/start_process.go`.
> - **Fixed**: `TaskManagerAdapter` now implements both `TaskManager` and `TaskStarter`.
> - **Fixed**: Tool registered in `builtin.go` (`builtinToolCount` 8 → 9).
> - 8 unit tests added, all passing. `make lint` clean.

### Session

```bash
mkdir -p ~/sources/testbed/t4-bg && cd ~/sources/testbed/t4-bg
cat > server.py << 'EOF'
import http.server, time, sys
print("Server starting on port 8765", flush=True)
httpd = http.server.HTTPServer(('', 8765), http.server.SimpleHTTPRequestHandler)
httpd.serve_forever()
EOF

spin exec "Start 'python3 server.py' as a background process. Then list all running processes. Then get the output of the server process. Then kill the server process. Finally list processes again to confirm it's gone." \
  --auto-approve --timeout 2m \
  --record-fixture tests/e2e/fixtures/recorded/t4_background.jsonl
```

### What to verify
- [ ] Background process starts and gets assigned an ID
- [ ] `list_processes` shows the running process
- [ ] `get_process_output` returns captured stdout
- [ ] `kill_process` terminates the process
- [ ] After kill, `list_processes` no longer shows it
- [ ] Multiple background processes can coexist

### Bugs to watch for
- Process output not captured (empty)
- Kill doesn't actually terminate (zombie process)
- Process ID not consistent across list/get/kill

---

## T-5: Git Context + Operations

> **Tools**: `git_context`, `git_operation`
>
> **Status**: ✅ DONE — Bug found and fixed: `get_diff` missed staged changes.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t5_git.jsonl` (9 turns)
>
> **Bug fixed**: `git_operation(get_diff)` ran `git diff` (unstaged only), returning empty after `git add`. Fixed to `git diff HEAD` which shows both staged and unstaged changes.
>
> **Evidence**:
> - `git_operation(get_status)` — returned branch, clean status correctly
> - `git_operation(stage)` — staged feature.md
> - `git_operation(get_diff)` — was empty before fix, now shows staged changes
> - `git_operation(create_branch)` — created feature/docs
> - `git_operation(commit)` — committed successfully
> - `git_operation(get_log)` — showed 2 commits
> - 4 new tests in `internal/git/diff_test.go` covering staged, unstaged, filtered, and clean diffs

### Session

```bash
mkdir -p ~/sources/testbed/t5-git && cd ~/sources/testbed/t5-git
git init && git config user.email "test@test.com" && git config user.name "Test"
echo "initial" > readme.md && git add . && git commit -m "init"

spin exec "Check git status. Create a new file called feature.md with content 'new feature'. Stage it. Show the diff. Create a new branch called 'feature/test'. Commit with message 'add feature doc'. Show the git log." \
  --auto-approve --timeout 2m \
  --record-fixture tests/e2e/fixtures/recorded/t5_git.jsonl
```

### What to verify
- [ ] `git_context` returns branch, clean/dirty status, ahead/behind
- [ ] `git_operation` stage adds files to index
- [ ] `git_operation` commit creates a commit
- [ ] `git_operation` get_diff shows staged changes
- [ ] `git_operation` create_branch creates branch
- [ ] `git_operation` get_log shows commit history
- [ ] `git_operation` get_status shows modified/untracked files

### Bugs to watch for
- Nested git repo crash (FIXED: snapshot_nestedgit)
- Branch creation without switching
- Commit with empty message
- Status in detached HEAD state

---

## T-6: Apply Patch

> **Tools**: `apply_patch`
>
> **Status**: ✅ DONE — 3 bugs found and fixed.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t6_patch.jsonl` (5 turns — patch failed before fix)
>
> **Bugs fixed**:
> 1. **Parameter name mismatch**: LLM sends `"patch"` but tool only accepted `"patch_text"` — patch silently ignored. Fixed: accept both names.
> 2. **`a/` prefix not stripped**: Unified diff `--- a/main.go` parsed filename as `a/main.go`. Fixed: `stripDiffPrefix()` removes `a/`/`b/` prefixes.
> 3. **Context matching failure**: `extractContextLines` collected ALL context lines (non-contiguous across changes), causing matcher to fail. Fixed: extract only leading contiguous context block, fall back to first block if no leading context.
>
> **Tests added**:
> - `internal/tools/apply_patch_test.go` — `TestApplyPatchTool_UnifiedDiff_ParameterNames` (alias + canonical)
> - `internal/patchapply/unifieddiff_test.go` — `TestApplier_UpdateFile_UnifiedDiffContextLines`

### Session

```bash
mkdir -p ~/sources/testbed/t6-patch && cd ~/sources/testbed/t6-patch
git init && git config user.email "test@test.com" && git config user.name "Test"
cat > main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("hello")
    fmt.Println("world")
}
EOF
git add . && git commit -m "init"

spin exec "Read main.go. Apply a patch that changes 'hello' to 'greetings' and adds a new line 'fmt.Println(\"goodbye\")' after the world line. Verify the result by reading the file again." \
  --auto-approve --timeout 2m \
  --record-fixture tests/e2e/fixtures/recorded/t6_patch.jsonl
```

### What to verify
- [ ] `apply_patch` accepts standard unified diff format
- [ ] Patch modifies correct lines
- [ ] Patch adds new lines at correct position
- [ ] Patch deletes lines when diff has `-` lines
- [ ] Invalid patch format returns clear error
- [ ] Patch on non-existent file returns error

### Bugs to watch for
- Line number offset causing wrong patch location
- Whitespace sensitivity in diff context lines
- Patch applied to wrong file
- PatchError.Error() reachable from production (wired in JOURNEY-2.2)

---

## T-7: LSP Tools (find_symbol, find_references, rename_symbol)

> **Tools**: `find_symbol`, `find_references`, `rename_symbol`
>
> **Status**: ✅ DONE — 3 bugs found and fixed in LSP integration.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t7_lsp.jsonl` (8 turns — all 3 tools failed before fix)
>
> **Bugs fixed**:
> 1. **DidOpen sends empty content**: `server.go` called `DidOpen(ctx, uri, lang, "")` — LSP server received empty document, so line numbers were out of range. Fixed: `openFileForLSP()` reads actual file content from disk.
> 2. **FindReferences missing DidOpen**: `FindReferences` never opened the file with the LSP server. Fixed: added `openFileForLSP()` call.
> 3. **find_symbol ignores name parameter**: Tool always called `FindDefinition(filePath, 0, 0)` regardless of symbol name. Fixed: added `SymbolSearcher` function type + `NewFindSymbolToolWithSearch()` that uses `SearchSymbols` (textDocument/documentSymbol).
>
> **Files changed**:
> - `internal/lsp/server.go` — `openFileForLSP()` reads file, passes to DidOpen; all 3 methods fixed
> - `internal/tools/find_symbol.go` — `SymbolSearcher` type, `NewFindSymbolToolWithSearch`, `executeSearch`
> - `internal/conversation/tools.go` — wires `SearchSymbols` into find_symbol tool

### Session

```bash
mkdir -p ~/sources/testbed/t7-lsp && cd ~/sources/testbed/t7-lsp
cat > main.go << 'EOF'
package main

import "fmt"

func greet(name string) string {
    return fmt.Sprintf("Hello, %s!", name)
}

func main() {
    msg := greet("world")
    fmt.Println(msg)
    fmt.Println(greet("universe"))
}
EOF
go mod init testmod

spin exec "Find the definition of the 'greet' function. Find all references to 'greet'. Rename 'greet' to 'sayHello' across the entire file. Read the file to confirm the rename worked." \
  --auto-approve --timeout 3m \
  --record-fixture tests/e2e/fixtures/recorded/t7_lsp.jsonl
```

### What to verify
- [ ] `find_symbol` returns file + line + column for the definition
- [ ] `find_references` returns all call sites
- [ ] `rename_symbol` changes all occurrences consistently
- [ ] LSP server starts lazily on first tool use
- [ ] LSP cache prevents redundant requests
- [ ] Graceful fallback when no LSP server available for the language

### Bugs to watch for
- LSP server not found (gopls not installed)
- Rename produces inconsistent state (some refs missed)
- LSP timeout on large projects
- File not opened (DidOpen not sent before query)

---

## T-8: Web Tools (fetch_url, web_search, open_browser)

> **Tools**: `fetch_url`, `web_search`, `open_browser`
>
> **Status**: ✅ DONE — 1 bug found and fixed.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t8_web.jsonl` (6 turns)
>
> **Evidence**:
> - `fetch_url(httpbin.org/html)` — HTML correctly converted to markdown (Moby-Dick excerpt)
> - `fetch_url(httpbin.org/status/404)` — was silently empty before fix, now reports "HTTP 404"
> - `write_file(notes.md)` — summary written successfully
>
> **Bug fixed**: `fetch_url` ignored HTTP status codes ≥ 400. A 404/500 response returned empty successful result instead of error. Fixed: check `StatusCode >= 400` and report as tool error with status code and body.
>
> **Files changed**:
> - `internal/tools/web_fetch.go` — added `httpErrorThreshold` constant and status code check
> - `internal/tools/errors.go` — added `errHTTPError` sentinel
> - `internal/tools/web_fetch_integration_test.go` — updated 500 test, added 404 test

### Session

```bash
mkdir -p ~/sources/testbed/t8-web && cd ~/sources/testbed/t8-web

spin exec "Fetch the URL https://httpbin.org/html and show the converted text content. Then try fetching a non-existent URL https://httpbin.org/status/404 and report the error. Save any useful information to notes.md." \
  --auto-approve --timeout 2m \
  --record-fixture tests/e2e/fixtures/recorded/t8_web.jsonl
```

### What to verify
- [ ] `fetch_url` retrieves HTML and converts to readable text via ConvertHTML
- [ ] `fetch_url` handles HTTP errors (404, 500) gracefully
- [ ] `fetch_url` respects timeout (30s default)
- [ ] `fetch_url` enforces max response size (5MB)
- [ ] `web_search` returns "not configured" error (stub)
- [ ] HTML conversion strips tags, preserves structure

### Bugs to watch for
- HTML conversion losing important content
- Timeout not applied (hangs on slow servers)
- Large response causing OOM
- Redirect loops not handled

---

## T-9: Memory + Scratchpad

> **Tools**: `memory`, `scratchpad`
>
> **Status**: ✅ DONE — 1 bug found and fixed: scratchpad not enabled by default.
>
> **Fixtures**:
> - `tests/e2e/fixtures/recorded/t9_memory.jsonl` (1 turn — tools missing before fix)
> - `tests/e2e/fixtures/recorded/t9_memory2.jsonl` (6 turns — all 3 ops work after fix)
>
> **Bug fixed**: `applyDefaults()` in config loader didn't apply memory defaults. When user config has no `memory:` section, scratchpad stayed disabled despite `DefaultV2()` setting `Scratchpad.Enabled: true`. Fixed: added `applyMemoryDefaults()` that applies scratchpad defaults when not explicitly configured.
>
> **Evidence after fix**:
> - `scratchpad(put, key=todo)` — stored successfully
> - `scratchpad(get, key=todo)` — retrieved correct value
> - `scratchpad(list)` — listed 1 entry
>
> **Files changed**:
> - `internal/config/loader_v2.go` — added `applyMemoryDefaults()`, called from `applyDefaults()`

### Session

```bash
mkdir -p ~/sources/testbed/t9-memory && cd ~/sources/testbed/t9-memory

spin exec "Store in memory that my preferred language is Rust and I use vim keybindings. Then store in scratchpad a temporary note: 'TODO: refactor auth module'. Read back both the memory and scratchpad to confirm they were saved." \
  --auto-approve --timeout 2m \
  --record-fixture tests/e2e/fixtures/recorded/t9_memory.jsonl
```

### What to verify
- [ ] `memory` store operation persists data
- [ ] `memory` retrieve operation returns stored data
- [ ] `scratchpad` store is session-scoped (ephemeral)
- [ ] `scratchpad` retrieve returns session data
- [ ] Memory survives across sessions (cross-session persistence)
- [ ] Scratchpad is cleared on session end

### Bugs to watch for
- Memory file corruption on concurrent writes
- Scratchpad not actually ephemeral (leaks to disk)
- Empty key/value handling

---

## T-10: Get Context + Environment

> **Tools**: `get_context`
>
> **Status**: ✅ DONE — 1 bug found and fixed: 4 tools missing from runtime registration.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t10_context.jsonl` (13 turns)
>
> **Bug fixed**: `BuiltinRuntime.RegisterTools()` only registered 9 tools (file, shell, process). Four tools from the `BuiltinTools` slice — `get_context`, `apply_patch`, `file_search`, `git_context` — were never registered in the runtime path. The LLM couldn't see or use them. Fixed: added all 4 to `RegisterTools()`, updated `builtinToolCount` 9→13.
>
> **Evidence**:
> - Agent used `shell_command` and `git_operation` as workarounds (tools worked)
> - `context_report.txt` written with full env info
> - After fix: `get_context`, `apply_patch`, `file_search`, `git_context` now available
>
> **Files changed**:
> - `internal/agent/executor/builtin.go` — registered 4 missing tools, `builtinToolCount` 9→13

### Session

```bash
mkdir -p ~/sources/testbed/t10-ctx && cd ~/sources/testbed/t10-ctx
git init && git config user.email "test@test.com" && git config user.name "Test"
echo "# My Project" > README.md && git add . && git commit -m "init"

spin exec "Get the full environment context. Report the working directory, git status, shell info, and any project type detection. Write a summary to context_report.txt." \
  --auto-approve --timeout 2m \
  --record-fixture tests/e2e/fixtures/recorded/t10_context.jsonl
```

### What to verify
- [ ] Returns working directory path
- [ ] Returns git branch and status
- [ ] Returns shell type and path
- [ ] Returns detected project type (if detectable)
- [ ] Returns OS/platform info
- [ ] Works in non-git directories (no crash)

### Bugs to watch for
- Crash in non-git directory
- Missing fields in context output
- Incorrect project type detection

---

## T-11: Multi-Tool Orchestration (Complex Task)

> **Tools**: Multiple tools in sequence — tests agent's ability to chain tools.
>
> **Status**: ✅ DONE — No bugs found. Perfect 7-turn orchestration.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t11_multi_tool.jsonl` (7 turns)
>
> **Evidence**:
> - read_file → edit_file → shell_command → git stage → git commit → git log → summary
> - Agent chained 6 different tools across 7 turns with zero errors
> - Shell output ("Result: 5\nProduct: 20") correctly influenced agent's next action (no fix needed)
> - Observation boundary maintained: tool results preserved across multi-turn conversation
> - File on disk matches expected content, git history correct

### Session

```bash
mkdir -p ~/sources/testbed/t11-multi/src && cd ~/sources/testbed/t11-multi
git init && git config user.email "test@test.com" && git config user.name "Test"
cat > src/app.py << 'EOF'
def calculate(a, b):
    return a + b

def main():
    result = calculate(2, 3)
    print(f"Result: {result}")

if __name__ == "__main__":
    main()
EOF
git add . && git commit -m "init"

spin exec "Read src/app.py. Add a multiply function. Edit the main function to also call multiply(4, 5). Run 'python3 src/app.py' to test. If there are errors, fix them. Stage and commit the changes with message 'add multiply function'. Show the git log." \
  --auto-approve --timeout 3m \
  --record-fixture tests/e2e/fixtures/recorded/t11_multi_tool.jsonl
```

### What to verify
- [ ] Agent chains read → edit → shell → git operations correctly
- [ ] Observation from shell_command (test output) influences next action
- [ ] Agent handles tool errors and self-corrects
- [ ] Multi-turn conversation maintains context
- [ ] All tool blocks visible in output
- [ ] Git commit includes all staged changes

### Bugs to watch for
- Observation summarizer destroying tool results before agent sees them
- Agent looping on the same failing command (doom loop)
- Context window overflow on long multi-tool sessions
- Tool result order confused in multi-tool turns

---

## T-12: Security & Approval Matrix

> **Tools**: All mutating tools without `--auto-approve`
>
> **Status**: ✅ DONE — Approval system works as designed. No bugs.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t12_security.jsonl` (6 turns)
>
> **Evidence**:
> - `read_file(secret.txt)` — allowed without approval (read-only) ✓
> - `list_directory(.)` — allowed without approval (read-only) ✓
> - `shell_command(echo dangerous)` — allowed (validator classifies `echo` as safe) ✓
> - `write_file(output.txt)` — **blocked** (file not created on disk) ✓
> - Approval handler: `createDenyHandler` correctly denies mutating operations
> - `echo` correctly passes: it's genuinely read-only despite the word "dangerous"

### Session

```bash
mkdir -p ~/sources/testbed/t12-security && cd ~/sources/testbed/t12-security
echo "secret data" > secret.txt

# Run WITHOUT --auto-approve to test approval gates.
spin exec "Read secret.txt. Then try to write a new file called output.txt. Then try to run 'rm secret.txt'." \
  --timeout 1m \
  --record-fixture tests/e2e/fixtures/recorded/t12_security.jsonl
```

### What to verify
- [ ] `read_file` succeeds without approval (read-only)
- [ ] `list_directory` succeeds without approval (read-only)
- [ ] `write_file` is blocked without approval
- [ ] `edit_file` is blocked without approval
- [ ] `shell_command` (mutating) is blocked without approval
- [ ] Blocklist blocks `rm -rf /`, fork bombs, etc. even with `--auto-approve`
- [ ] Blocked tool produces user-visible message

### Bugs to watch for
- Read-only tool incorrectly requiring approval
- Mutating tool bypassing approval
- Blocklist false positive on safe commands like `rm temp.txt`
- Approval message not shown in output

---

## T-13: Edge Cases & Error Handling

> **Tools**: Various — tests error paths.
>
> **Status**: ✅ DONE — No bugs found. All error paths return clear messages.
>
> **Fixture**: `tests/e2e/fixtures/recorded/t13_edge.jsonl` (8 turns)
>
> **Evidence**:
> - `read_file(/nonexistent/path.txt)` → `"failed to read file: ...no such file or directory"` ✓
> - `edit_file(exists.txt, non-matching old_content)` → `"no match found for old_content"` ✓
> - `list_directory(/no/such/dir)` → `"failed to read directory: ...no such file or directory"` ✓
> - `shell_command(exit 42)` → `"execution failed: exit status 42"` ✓
> - Agent recovered from all 4 errors and continued to final summary
> - No panics, no crashes, no hangs

### Session

```bash
mkdir -p ~/sources/testbed/t13-edge && cd ~/sources/testbed/t13-edge

spin exec "Try to read a file that doesn't exist: /nonexistent/path.txt. Try to write to a read-only path: /etc/shadow. Try to edit a file with content that doesn't match. Try to list a directory that doesn't exist. Run a shell command with invalid syntax." \
  --auto-approve --timeout 2m \
  --record-fixture tests/e2e/fixtures/recorded/t13_edge.jsonl
```

### What to verify
- [ ] `read_file` on missing file returns clear error (not crash)
- [ ] `write_file` to unwritable path returns permission error
- [ ] `edit_file` with non-matching old_content returns clear error
- [ ] `list_directory` on missing dir returns error
- [ ] `shell_command` with bad syntax returns stderr + exit code
- [ ] All errors are visible in agent output (not swallowed)
- [ ] Agent recovers from errors and continues

### Bugs to watch for
- Panic on nil path
- Permission error not propagated
- Agent stuck after error (no recovery)
- Error messages not user-friendly

---

## Summary

| Item | Tools Covered | Category | Status |
|------|--------------|----------|--------|
| T-1 | read_file, write_file, edit_file | Core Files | ⬚ TODO |
| T-2 | list_directory, file_search | Directory & Search | ⬚ TODO |
| T-3 | shell_command | Shell Execution | ⬚ TODO |
| T-4 | start_process, list_processes, get_process_output, kill_process | Background Processes | ✅ DONE |
| T-5 | git_context, git_operation | Git | ✅ DONE |
| T-6 | apply_patch | Patching | ✅ DONE |
| T-7 | find_symbol, find_references, rename_symbol | LSP | ✅ DONE |
| T-8 | fetch_url, web_search, open_browser | Web | ✅ DONE |
| T-9 | memory, scratchpad | Persistence | ✅ DONE |
| T-10 | get_context | Environment | ✅ DONE |
| T-11 | Multi-tool chain | Orchestration | ✅ DONE |
| T-12 | All mutating (no auto-approve) | Security | ✅ DONE |
| T-13 | Various error paths | Edge Cases | ✅ DONE |

**Total: 24 tools across 13 test sessions.**

**Known bugs already fixed:**
- Truncation false positive on Rust lifetimes (T-1 category, fixed in `write_file.go`)
- Snapshot failure on nested `.git` dirs (T-5 category, fixed in `snapshot.go`)
- FileTracker rejects writes to externally-created files (T-1 category, fixed earlier)
- ACP path missing hooks, Composer, Factory (fixed in `acp.go`)

**Recorded fixtures from prior sessions:**
- `tetris_create.jsonl` — 11 turns (hit truncation bug)
- `tetris_complex2.jsonl` — 13 turns (after fix, successful)
