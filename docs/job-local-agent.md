## NAME

Local Agent - using Spin as an autonomous coding assistant in your terminal

## WHEN TO USE

Use Spin as a local agent when you want to:

- Work inside a repository and have an agent read, edit, and refactor files
- Run tools (tests, linters, formatters, shell commands) through the agent
- Stay inside a terminal UI without losing scrollback or editor context

All flows in this document are backed by automated tests under `tests/e2e/` and
CLI/unit tests in `cmd/spin` and `internal/`.

## PREREQUISITES

- Spin binary built: `make build` (produces `bin/spin`)
- LLM provider available:
  - Local: Ollama or LM Studio
  - Remote: OpenAI, Anthropic, Google, …
- Minimal configuration in `~/.spin/config.yaml` or CLI flags

See `README.md` for provider configuration and environment variables.

## FLOW 1: FIRST INTERACTIVE SESSION IN TUI

Goal: launch the terminal UI, send a prompt, and see a response.

### Steps

1. Build the binary (from project root):

   ```bash
   make build
   ```

2. Start the TUI with a local Ollama model:

   ```bash
   ./bin/spin --provider ollama --model qwen3:0.6b
   ```

3. Wait for the UI to render. You should see the Spin logo and an input prompt.

4. Type a simple instruction:

   ```text
   Say exactly: Hello from Spin
   ```

5. Press `Enter`. The agent streams a response in the timeline area above the prompt.

6. Exit with `Ctrl+D`. The process terminates cleanly.

### What this proves

- The TUI starts correctly and renders in a real terminal.
- Spin can talk to your local provider and stream responses.
- Basic lifecycle (start, chat, exit) works end-to-end.

## FLOW 2: USING THE AGENT TO INSPECT AND EDIT FILES

Goal: ask Spin to read files in the current repository and prepare changes.

### Steps

1. Run Spin in the root of your repository:

   ```bash
   cd /path/to/your/project
   ./bin/spin --provider ollama --model qwen3:1.7b
   ```

2. After the UI initializes, request a directory listing:

   ```text
   list files in current directory
   ```

   Spin uses its `list_directory` tool to inspect the workspace and shows a block
   indicating tool execution before presenting the result.

3. Ask Spin to read a specific file:

   ```text
   read the go.mod file and summarize what this project depends on
   ```

   Spin calls `read_file`, shows a tool block (e.g., `EXECUTE` / `read_file`),
   then streams a summary in plain text.

4. Ask Spin to propose an edit:

   ```text
   Improve the README.md introduction, show a patch before applying any changes.
   ```

   Spin will typically:

   - Use `read_file` to load `README.md`
   - Propose a patch using its diff tooling
   - Show a `DIFF`/`EXECUTE` block with the proposed unified diff

5. Review the diff in the TUI. If you are satisfied, instruct Spin explicitly:

   ```text
   Apply this patch.
   ```

   Depending on the patch, this may trigger the approval flow described in Flow 3.

### What this proves

- Spin can navigate your workspace and inspect files through tools.
- File reads and diffs are visible in the timeline as structured blocks.
- The agent can prepare edits without immediately mutating your files.

## FLOW 3: CREATING FILES WITH APPROVAL

Goal: let Spin perform a write that requires user approval, and approve it from the TUI.

### Steps

1. Work in an empty temporary directory:

   ```bash
   workdir=$(mktemp -d)
   cd "$workdir"
   ./bin/spin --provider ollama --model qwen3:1.7b --sandbox workspace-write
   ```

2. After the UI starts, ask Spin to create a file:

   ```text
   Create a file called test.txt with the text 'automated test'
   ```

3. Spin classifies the operation as a write and opens an approval dialog:

   - The TUI shows an overlay with command details.
   - The rest of the interface is dimmed.

4. Press `A` to approve.

5. Wait for the overlay to disappear and for a new block to appear in the timeline
   indicating tool execution and completion.

6. In another terminal, verify the file:

   ```bash
   cat "$workdir/test.txt"
   # -> automated test
   ```

### What this proves

- Approval flows are wired into the TUI for write operations.
- Spin respects the sandbox mode and writes only inside the workspace.
- File creation can be audited by inspecting the timeline and resulting files.

- `tests/e2e/approval_persistence_e2e_test.go`
  - Verifies that approval decisions persist across runs where applicable
- `tests/e2e/acp/permission_test.go`
  - Exercises the same approval mechanisms over ACP (see ACP flows for IDE usage)

## FLOW 4: MULTI-TURN CONVERSATIONS WITH CONTEXT

Goal: keep a conversation going where Spin remembers earlier messages.

### Steps

1. Start Spin in the project root:

   ```bash
   ./bin/spin --provider ollama --model qwen3:1.7b
   ```

2. Tell Spin a fact:

   ```text
   My favorite number is 42.
   ```

3. Wait for the agent to acknowledge.

4. Ask a follow-up question:

   ```text
   What is my favorite number?
   ```

5. Spin responds with `42` (possibly inside a sentence).

### What this proves

- Conversation context is preserved across turns.
- The agent can use previous messages to answer follow-up questions without restating context.


## FLOW 5: CONTROLLING LONG RESPONSES

Goal: stop a long-running answer from the agent without killing the process.

### Steps

1. Run Spin as usual:

   ```bash
   ./bin/spin --provider ollama --model qwen3:1.7b
   ```

2. Ask for a long response:

   ```text
   Write a very long story about a robot exploring the galaxy.
   ```

3. Once the agent starts streaming, press `Ctrl+C`.

4. Streaming stops, the TUI remains responsive, and you can send another prompt.

### What this proves

- You can interrupt long LLM responses without leaving the session.
- The TUI recovers after cancellation and continues to accept input.


## FLOW 6: KEYBOARD SHORTCUTS AND MODALS

Goal: discover helper UIs (file picker, help) that speed up local workflows.

### Steps

1. Start Spin in a project:

   ```bash
   ./bin/spin --provider ollama --model qwen3:0.6b
   ```

2. Press `@` in the input area:

   - A file picker appears, letting you select files to mention or inspect.
   - Use arrow keys and `Enter` to choose a file (behavior may evolve with UI changes).
   - Press `Esc` to close the picker.

3. Press `Ctrl+H`:

   - A help modal appears with keybindings and usage tips.
   - Press `Esc` to close the modal.

### What this proves

- Spin's TUI supports quick file selection to anchor prompts in concrete files.
- Built-in help is available from inside the UI, no browser required.

## RELATED DOCUMENTS

- `docs/job-ci-automation.md` – using Spin in CI and non-interactive scripts
- `docs/job-acp-ide.md` – using Spin from an ACP-compatible IDE/editor
- `docs/testing-mapping.md` – mapping of all flows to tests


