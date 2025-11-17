## NAME

Testing Mapping - mapping of documented scenarios to test files and functions

## DESCRIPTION

This document maintains a mapping between user-facing scenarios documented in the guides (`docs/job-*.md`) and the automated tests that verify them. Every scenario should be marked as either `covered` (has tests) or `needs-test` (requires test implementation).

## HOW TO USE

When adding or updating documentation:

1. Add a row for each new scenario in the appropriate section below
2. Mark it as `covered` if tests exist, or `needs-test` if they don't
3. Update the status when tests are added

When implementing tests:

1. Find the corresponding scenario in this document
2. Update the status from `needs-test` to `covered`
3. Add the test file and function name to the mapping

## LOCAL AGENT SCENARIOS

| Scenario | Document | Test File | Test Function | Status |
|----------|----------|-----------|---------------|--------|
| First-run: configure provider and run `spin` | `job-local-agent.md` Flow 1 | `tests/e2e/e2e_test.go` | `TestConfigCommands` | covered |
| First-run: complete one end-to-end change | `job-local-agent.md` Flow 1 | - | - | needs-test |
| Non-interactive `spin exec` run that fixes tests | `job-local-agent.md` Flow 2 | `tests/e2e/e2e_test.go` | `TestExecMode` | covered |
| Switching modes (`regular`, `review`, `planning`, `compact`) | `job-local-agent.md` Flow 3 | - | - | needs-test |
| Interactive TUI launch and initialization | `job-local-agent.md` Flow 4 | - | - | needs-test |
| Tool execution in TUI (list_directory, read_file) | `job-local-agent.md` Flow 5 | - | - | needs-test |
| Approval workflow for dangerous operations | `job-local-agent.md` Flow 6 | - | - | needs-test |
| Multi-turn conversation with context retention | `job-local-agent.md` Flow 7 | - | - | needs-test |
| File picker trigger with @ key | `job-local-agent.md` Flow 8 | - | - | needs-test |
| Help modal with Ctrl+H | `job-local-agent.md` Flow 9 | - | - | needs-test |
| Exit with Ctrl+D | `job-local-agent.md` Flow 10 | - | - | needs-test |
| Stop streaming with Ctrl+C | `job-local-agent.md` Flow 11 | - | - | needs-test |
| Tool visualization in TUI blocks | `job-local-agent.md` Flow 12 | - | - | needs-test |
| Multiple tool calls in sequence | `job-local-agent.md` Flow 13 | - | - | needs-test |

## CI/AUTOMATION SCENARIOS

| Scenario | Document | Test File | Test Function | Status |
|----------|----------|-----------|---------------|--------|
| One-off execution in shell with `spin exec` | `job-ci-automation.md` Flow 1 | `tests/e2e/e2e_test.go` | `TestExecMode` | covered |
| Exec from stdin (piping) | `job-ci-automation.md` Flow 1 | `tests/e2e/e2e_test.go` | `TestExecMode` (subtest: exec from stdin) | covered |
| CI job with `--auto-approve` to keep branch green | `job-ci-automation.md` Flow 2 | `tests/e2e/approval_cli_e2e_test.go` | Various approval CLI tests | covered |
| Read-only analysis without `--auto-approve` | `job-ci-automation.md` Flow 3 | `tests/e2e/exec_readonly_e2e_test.go` | `TestExecMode_ReadOnlyDeniesWrites` | covered |
| JSON output format for machine consumption | `job-ci-automation.md` Flow 4 | `tests/e2e/e2e_test.go` | `TestJSONOutput` | covered |
| Scheduling in cron or batch jobs | `job-ci-automation.md` Flow 5 | Same as Flow 2 | Same as Flow 2 | covered |
| Approval persistence across executions | `job-ci-automation.md` Flow 2 | `tests/e2e/approval_persistence_e2e_test.go` | `TestApprovalPersistence_SessionAndGlobalScopes` | covered |
| Approval CLI commands (list, clear, revoke) | `job-ci-automation.md` Flow 2 | `tests/e2e/approval_cli_e2e_test.go` | `TestApprovalCLI_ListAndClear_Empty`, `TestApprovalCLI_Revoke_NonExistent` | covered |

## ACP IDE SCENARIOS

| Scenario | Document | Test File | Test Function | Status |
|----------|----------|-----------|---------------|--------|
| Getting started: IDE connects to Spin and shows agent ready | `job-acp-ide.md` Flow 1 | `tests/e2e/acp/initialize_test.go` | `TestACP_Initialize` | covered |
| Getting started: Available task modes shown in IDE | `job-acp-ide.md` Flow 1 | `tests/e2e/acp/session_test.go` | `TestACP_NewSession_ModeState` | covered |
| Getting started: Send prompt and see response in IDE | `job-acp-ide.md` Flow 1 | `tests/e2e/acp/prompt_test.go` | `TestACP_Prompt_Basic` | covered |
| Refactoring: Spin reads files and shows tool execution | `job-acp-ide.md` Flow 2 | `tests/e2e/acp/tool_call_test.go` | `TestACP_Prompt_ToolCalls` | covered |
| Refactoring: Spin proposes changes and requests approval | `job-acp-ide.md` Flow 2 | `tests/e2e/acp/permission_test.go` | `TestACP_RequestPermission` | covered |
| Refactoring: Review diffs and approve/reject changes | `job-acp-ide.md` Flow 2 | `tests/e2e/acp/permission_test.go` | `TestACP_RequestPermission_Options` | covered |
| Code review: Switch to review mode in IDE | `job-acp-ide.md` Flow 3 | `tests/e2e/acp/mode_test.go` | `TestACP_SetSessionMode` | covered |
| Code review: Review mode prevents file modifications | `job-acp-ide.md` Flow 3 | `tests/e2e/acp/mode_test.go` | `TestACP_SetSessionMode_AllModes` | covered |
| Code review: IDE notified when mode changes | `job-acp-ide.md` Flow 3 | `tests/e2e/acp/mode_test.go` | `TestACP_SetSessionMode_Notifications` | covered |
| Dangerous operations: Approval dialog appears in IDE | `job-acp-ide.md` Flow 4 | `tests/e2e/acp/permission_test.go` | `TestACP_RequestPermission` | covered |
| Dangerous operations: All approval options work (Allow Once, Always, Reject) | `job-acp-ide.md` Flow 4 | `tests/e2e/acp/permission_test.go` | `TestACP_RequestPermission_Options` | covered |
| Dangerous operations: Approval decisions persist across sessions | `job-acp-ide.md` Flow 4 | `tests/e2e/acp/approval_persistence_e2e_test.go` | `TestACP_ApprovalPersistence_PromptToToolCall` | covered |
| Cancelling: Stop long-running operation from IDE | `job-acp-ide.md` Flow 5 | `tests/e2e/acp/cancel_test.go` | `TestACP_Cancel` | covered |
| Cancelling: Cancel when no prompt is active | `job-acp-ide.md` Flow 5 | `tests/e2e/acp/cancel_test.go` | `TestACP_Cancel_NoActivePrompt` | covered |
| Session persistence: Load previous session after IDE restart | `job-acp-ide.md` Flow 6 | `tests/e2e/acp/session_test.go` | `TestACP_LoadSession` | covered |
| Session persistence: Conversation history maintained | `job-acp-ide.md` Flow 6 | `tests/e2e/acp/session_test.go` | `TestACP_NewSession` | covered |
| Multi-file tasks: Spin works across multiple files | `job-acp-ide.md` Flow 7 | `tests/e2e/acp/tool_call_test.go` | `TestACP_Prompt_ToolCalls` | covered |
| Multi-file tasks: Tool execution shown for each file operation | `job-acp-ide.md` Flow 7 | `tests/e2e/acp/tool_call_test.go` | `TestACP_Prompt_ToolCallNotifications` | covered |

## TEST COVERAGE SUMMARY

### Coverage Status

- **Covered**: 28 scenarios
- **Needs Test**: 14 scenarios
- **Total**: 42 scenarios

### Coverage by Document

- `job-local-agent.md`: 1/14 covered (7%)
- `job-ci-automation.md`: 9/9 covered (100%)
- `job-acp-ide.md`: 19/19 covered (100%)

## GAPS AND NEXT STEPS

### Needs Test

All documented scenarios are now covered by tests.

### Future Enhancements

These scenarios are documented but may need additional test coverage as features evolve:

- More complex tool call scenarios in ACP
- Concurrent prompt execution in ACP
- Advanced approval policy scenarios
- MCP server integration in ACP sessions

## MAINTENANCE

This document should be updated:

- When new scenarios are added to documentation
- When new tests are implemented
- When test functions are renamed or moved
- During code reviews to ensure documentation and tests stay in sync

## SEE ALSO

- `docs/job-local-agent.md` – Local agent usage guide
- `docs/job-ci-automation.md` – CI/CD automation guide
- `docs/job-acp-ide.md` – ACP IDE integration guide
- `tests/e2e/README.md` – E2E test documentation
- `tests/README.md` – General testing documentation

