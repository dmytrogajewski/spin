# Feature Requirements Document: ACP Diff Generation for Tool Calls

**Feature ID**: FRD-20251114082154  
**Feature**: 6.2 - Diff Generation for Tool Calls  
**Date**: 2025-11-14  
**Status**: In Progress

## Overview

Generate unified diffs for file modification tool calls (specifically `write_file`) and include them in `tool_call_update` notifications using the ACP SDK's `ToolDiffContent` helper.

## Background

When tools modify files, clients benefit from seeing the exact changes made. The ACP SDK provides `acp.ToolDiffContent(path, newText, oldText ...string)` to include diffs in tool call notifications. This feature integrates diff generation into the event-to-notification converter.

## Requirements

### Functional Requirements

1. **Diff Generation for File Write Operations**
   - Detect `write_file` tool calls
   - Read existing file content before write (if file exists)
   - Generate unified diff format between old and new content
   - Include diff in `tool_call_update` notification

2. **New File Handling**
   - Detect when file doesn't exist (new file creation)
   - Pass `oldText` as empty string to `acp.ToolDiffContent`
   - SDK handles null/empty oldText correctly

3. **Integration with Notification Converter**
   - Track old file content per tool call ID
   - Generate diff on tool completion
   - Include diff content in `UpdateToolCall` notification

### Technical Requirements

1. **Diff Library**
   - Use `github.com/pmezard/go-difflib/difflib` (already in dependencies)
   - Generate unified diff format
   - Handle file read errors gracefully (treat as new file)

2. **State Management**
   - Store old file content in a map keyed by tool ID
   - Clean up old content after tool completion
   - Handle concurrent tool calls safely

3. **SDK Integration**
   - Use `acp.ToolDiffContent(path, newText, oldText ...string)` helper
   - Include diff content in `acp.WithUpdateContent()` option
   - Maintain existing content (text output) alongside diff

## Design

### Architecture

```
EventToolCallStart (write_file)
    ↓
Read old file content (if exists)
    ↓
Store in map[toolID] = oldContent
    ↓
EventToolCallComplete (write_file)
    ↓
Get new content from start event parameters
    ↓
Generate unified diff
    ↓
Include in tool_call_update notification
```

### Implementation Details

1. **Old Content Tracking**
   - Add `oldContentMap map[string]string` to notification converter state
   - Read file on `EventToolCallStart` for `write_file` tool
   - Store with tool ID as key
   - Clean up on `EventToolCallComplete`

2. **Diff Generation**
   - Function: `generateUnifiedDiff(oldText, newText, path string) (acp.ToolCallContent, error)`
   - Use `difflib.GetUnifiedDiffString()` to generate diff
   - Format: unified diff with proper headers
   - Return `acp.ToolDiffContent(path, newText, oldText)`

3. **Notification Enhancement**
   - Modify `convertToolCallComplete()` to:
     - Check if tool is `write_file`
     - Retrieve old content from map
     - Generate diff if old content exists or file was new
     - Include diff in content array alongside text output

## Acceptance Criteria

- [ ] Diffs generated for file write operations
- [ ] Diffs included in `tool_call_update` notifications
- [ ] New file creation handled (oldText = empty)
- [ ] Existing file modification shows unified diff
- [ ] File read errors handled gracefully
- [ ] Concurrent tool calls handled safely
- [ ] Old content map cleaned up after use
- [ ] Unit tests cover diff generation
- [ ] Integration tests cover end-to-end flow
- [ ] No performance regression

## Testing Strategy

### Unit Tests

1. **Diff Generation**
   - Test unified diff format generation
   - Test new file (empty oldText)
   - Test file modification (oldText and newText)
   - Test empty files
   - Test large files

2. **Old Content Tracking**
   - Test storing old content on tool start
   - Test retrieving old content on tool complete
   - Test cleanup after tool complete
   - Test concurrent tool calls

3. **Error Handling**
   - Test file read errors (treat as new file)
   - Test missing tool ID in map
   - Test invalid file paths

### Integration Tests

1. **End-to-End Flow**
   - Test `write_file` tool call generates diff notification
   - Test diff appears in `tool_call_update` notification
   - Test diff format is correct
   - Test new file vs. existing file

## Dependencies

- `github.com/pmezard/go-difflib/difflib` - Diff generation
- `github.com/coder/acp-go-sdk` - `acp.ToolDiffContent` helper
- `internal/tools` - Tool parameter access
- `internal/events` - Event types

## Risks

1. **Performance**: Reading files on every tool start may impact performance
   - Mitigation: Only read for `write_file` tool, cache if needed

2. **Concurrency**: Multiple concurrent tool calls need safe state management
   - Mitigation: Use mutex to protect old content map

3. **File System**: File may be modified between read and write
   - Mitigation: Acceptable - diff shows what was intended vs. what existed

## Notes

- Diff generation is only for `write_file` tool initially
- Future enhancement: Support other file modification tools
- Line number extraction is deferred to future feature
- Diff format follows unified diff standard

