## NAME

IDE Integration - using Spin as an autonomous coding agent in your IDE

## WHEN TO USE

Use Spin with an ACP-compatible IDE when you want to:

- Get coding assistance directly in your editor without switching to a terminal
- Have the agent read, edit, and refactor files through your IDE's UI
- Review code, fix bugs, and implement features with approval dialogs in your IDE
- Maintain conversation context across IDE sessions

This mode works with IDEs that support the Agent Client Protocol (ACP), such as Cursor and other compatible editors.

## PREREQUISITES

- Spin binary built and available in `PATH` or configured in your IDE
- An ACP-compatible IDE/editor (e.g., Cursor)
- Configured LLM provider:
  - Environment variables (e.g. `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`) or
  - Local provider like Ollama/LM Studio with a model (e.g. `qwen3:0.6b`)

Your IDE should be configured to launch Spin as `spin acp` with appropriate flags.

## FLOW 1: GETTING STARTED WITH SPIN IN YOUR IDE

Goal: start using Spin in your IDE for the first time.

### Steps

1. Configure your IDE to use Spin as the ACP agent:
   - Set the agent command to: `spin acp --provider ollama --model qwen3:0.6b`
   - Or use a remote provider: `spin acp --provider openai --model gpt-4`
   - Point the workspace to your project directory

2. Open your IDE in a project directory.

3. The IDE connects to Spin automatically. You should see:
   - Agent status indicating Spin is ready
   - Available task modes (regular, review, compact, planning)

4. Start a conversation by typing a prompt in your IDE's agent panel:

   ```text
   Analyze the structure of this codebase and suggest improvements
   ```

5. Watch as Spin:
   - Reads files using its tools
   - Shows tool execution blocks in the IDE
   - Streams responses with analysis and suggestions

### What this proves

- Spin integrates with your IDE and responds to prompts
- Tool execution is visible in the IDE UI
- Basic agent functionality works in the IDE context

### Test coverage

- `tests/e2e/acp/initialize_test.go`
  - `TestACP_Initialize` – validates IDE connection and initialization
  - `TestACP_Initialize_AgentCapabilities` – verifies Spin advertises its capabilities

- `tests/e2e/acp/session_test.go`
  - `TestACP_NewSession` – validates session creation with your workspace
  - `TestACP_NewSession_ModeState` – verifies available modes are shown

## FLOW 2: REFACTORING CODE THROUGH YOUR IDE

Goal: ask Spin to refactor code and see the changes in your IDE.

### Steps

1. In your IDE, open a file you want to refactor (e.g., `auth.go`).

2. Send a prompt to Spin:

   ```text
   Refactor this file to use dependency injection instead of global variables
   ```

3. Spin will:
   - Read the current file using `read_file`
   - Analyze the code structure
   - Propose changes and show diffs in your IDE
   - Request approval before applying changes

4. Review the proposed changes in your IDE's diff view.

5. Approve or reject the changes:
   - **Allow Once**: Apply this change, ask again next time
   - **Allow Always**: Apply similar changes automatically
   - **Reject**: Skip this change

6. If approved, Spin applies the changes and you see the updated file in your IDE.

### What this proves

- Spin can read and modify files through the IDE
- Approval workflow protects against unwanted changes
- Diffs are shown clearly in the IDE interface

### Test coverage

- `tests/e2e/acp/prompt_test.go`
  - `TestACP_Prompt_Basic` – validates prompt processing and responses
  - `TestACP_Prompt_ContentBlocks` – tests handling different content types

- `tests/e2e/acp/tool_call_test.go`
  - `TestACP_Prompt_ToolCalls` – validates tool execution during prompts
  - `TestACP_Prompt_ToolCallNotifications` – verifies tool calls are shown in IDE

- `tests/e2e/acp/permission_test.go`
  - `TestACP_RequestPermission` – validates approval dialogs work correctly
  - `TestACP_RequestPermission_Options` – tests all approval options

## FLOW 3: CODE REVIEW AND SECURITY AUDIT

Goal: use Spin to review code for issues without making changes.

### Steps

1. Switch Spin to review mode in your IDE (if your IDE supports mode selection):

   ```text
   /mode review
   ```

   Review mode restricts Spin to read-only operations and focuses on analysis.

2. Ask Spin to review a specific area:

   ```text
   Review the authentication package for security vulnerabilities
   ```

3. Spin will:
   - Read relevant files
   - Analyze code patterns
   - Identify potential security issues
   - Provide recommendations without modifying files

4. Review the findings in your IDE. Spin's analysis appears as text in the agent panel.

5. If you want Spin to fix issues, switch back to regular mode and ask for fixes.

### What this proves

- Mode switching changes agent behavior appropriately
- Review mode prevents accidental modifications
- Security analysis works through the IDE interface

### Test coverage

- `tests/e2e/acp/mode_test.go`
  - `TestACP_SetSessionMode` – validates mode switching
  - `TestACP_SetSessionMode_AllModes` – tests all available modes
  - `TestACP_SetSessionMode_Notifications` – verifies IDE is notified of mode changes

## FLOW 4: HANDLING DANGEROUS OPERATIONS

Goal: understand how Spin requests approval for potentially dangerous operations.

### Steps

1. Ask Spin to perform an operation that might be dangerous:

   ```text
   Remove all TODO comments from the codebase
   ```

   or

   ```text
   Run go test ./... and fix any failing tests
   ```

2. If Spin determines the operation is dangerous (e.g., `rm -rf`, `git push --force`), your IDE shows an approval dialog with:
   - The command or operation details
   - Why it's considered dangerous
   - Options: Allow Once, Allow Always, Reject Once, Reject Always

3. Choose an option:
   - **Allow Once**: Execute this time, ask again for similar operations
   - **Allow Always**: Remember this decision and auto-approve similar operations
   - **Reject**: Skip this operation
   - **Reject Always**: Remember to always reject similar operations

4. Spin proceeds based on your choice. If you chose "Always", future similar operations won't prompt you.

### What this proves

- Approval dialogs appear in your IDE for dangerous operations
- Persistent policies remember your choices across sessions
- You maintain control over what Spin can do

### Test coverage

- `tests/e2e/acp/permission_test.go`
  - `TestACP_RequestPermission` – validates permission request flow
  - `TestACP_RequestPermission_Options` – tests all permission option kinds

- `tests/e2e/acp/approval_persistence_e2e_test.go`
  - Validates that approval decisions persist across IDE sessions

## FLOW 5: CANCELLING LONG-RUNNING OPERATIONS

Goal: stop a prompt that's taking too long.

### Steps

1. Start a prompt that might take a while:

   ```text
   Analyze the entire codebase and generate a comprehensive refactoring plan
   ```

2. If the operation is taking too long, use your IDE's cancel button or command.

3. Spin will:
   - Stop processing the current prompt
   - Cancel any in-flight tool calls
   - Return a cancellation response

4. Your IDE shows that the operation was cancelled, and you can start a new prompt.

### What this proves

- Long-running operations can be interrupted cleanly
- Cancellation works correctly through the IDE interface
- You're not stuck waiting for operations to complete

### Test coverage

- `tests/e2e/acp/cancel_test.go`
  - `TestACP_Cancel` – validates cancellation during prompt execution
  - `TestACP_Cancel_NoActivePrompt` – tests cancellation when nothing is running

## FLOW 6: CONTINUING CONVERSATIONS ACROSS IDE RESTARTS

Goal: pick up where you left off after closing and reopening your IDE.

### Steps

1. Have an active conversation with Spin in your IDE.

2. Close your IDE (or it may save the session automatically).

3. Reopen your IDE in the same project.

4. Your IDE should restore the previous session, showing:
   - Previous conversation history
   - Context from earlier prompts
   - Ability to continue the conversation

5. Continue the conversation:

   ```text
   Based on our previous discussion, implement the refactoring we discussed
   ```

6. Spin remembers the context and continues from where you left off.

### What this proves

- Sessions persist across IDE restarts
- Conversation history is maintained
- Context is preserved for seamless continuation

### Test coverage

- `tests/e2e/acp/session_test.go`
  - `TestACP_LoadSession` – validates loading an existing session
  - `TestACP_NewSession` – validates session creation and state

## FLOW 7: WORKING WITH MULTIPLE FILES AND COMPLEX TASKS

Goal: have Spin work across multiple files to complete a complex task.

### Steps

1. Ask Spin to perform a task that spans multiple files:

   ```text
   Refactor the authentication system to use JWT tokens. Update all files that use the old auth system.
   ```

2. Spin will:
   - Search for relevant files using `file_search`
   - Read multiple files to understand dependencies
   - Show tool execution blocks for each file read
   - Propose changes across multiple files
   - Request approval for each file modification

3. Review the changes in your IDE's diff view. You can see:
   - Which files will be modified
   - What changes will be made to each file
   - How files relate to each other

4. Approve or reject changes file by file, or approve all at once.

5. Spin applies the approved changes, and you see updated files in your IDE.

### What this proves

- Spin can work across multiple files in a coordinated way
- Complex refactorings are handled correctly
- Multi-file changes are presented clearly in the IDE

### Test coverage

- `tests/e2e/acp/tool_call_test.go`
  - `TestACP_Prompt_ToolCalls` – validates multiple tool calls in sequence
  - `TestACP_Prompt_ToolCallNotifications` – verifies tool execution is shown for each operation

## TASK MODES IN YOUR IDE

Spin supports different task modes that change how the agent behaves:

- **regular** (default): Full tool access, 16K token budget. Use for feature implementation and debugging.
- **review**: Read-only tools, 12K token budget. Use for code review and security audits.
- **compact**: Minimal tools, 4K token budget. Use for quick queries and documentation lookup.
- **planning**: Context tools, 4K token budget. Use for architecture planning and task breakdown.

Your IDE may allow you to switch modes, or Spin may suggest a mode based on your prompt.

## RELATED DOCUMENTS

- `docs/job-local-agent.md` – using Spin in a terminal with TUI
- `docs/job-ci-automation.md` – using Spin in CI/CD and automation scripts
- `docs/testing-mapping.md` – mapping of all flows to tests
- `tests/e2e/acp/README.md` – detailed ACP test documentation
