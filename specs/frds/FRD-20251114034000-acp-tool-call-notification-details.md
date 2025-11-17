# Feature Requirements Document: ACP Tool Call Notification Details

**Feature ID**: FRD-20251114034000  
**Feature Name**: Tool Call Notification Details  
**Roadmap Feature**: 6.1  
**Status**: In Progress  
**Created**: 2025-11-14  
**Author**: Spin Agent

## Overview

Enhance tool call notifications with proper content, locations, and status tracking to provide clients with rich information about tool execution.

## Background

Currently, tool call notifications are basic - they only include tool ID and title. This feature enhances them with:
- Tool kind mapping (read, edit, execute, search, etc.)
- Tool status tracking (pending, in_progress, completed, failed)
- Tool call content (text output, diffs)
- File locations (if available)

## Requirements

### Functional Requirements

1. **Tool Kind Mapping**
   - Map Spin tool names to ACP tool kinds:
     - `read_file` → `read`
     - `write_file` → `edit`
     - `shell_command` → `execute`
     - `file_search` → `search`
     - `list_directory` → `read` (directory listing)
     - Other tools → default to empty (no kind specified)

2. **File Location Extraction**
   - Extract file paths from tool parameters:
     - `read_file`: extract `path` parameter
     - `write_file`: extract `path` parameter
     - `file_search`: extract `workspace_root` parameter (if available)
     - `list_directory`: extract `path` parameter
   - Create `ToolCallLocation` objects with file paths
   - Include line numbers if available (future enhancement)

3. **Tool Call Start Enhancement**
   - Include tool kind in `StartToolCall` using `WithStartKind()`
   - Include file locations using `WithStartLocations()`
   - Include raw input using `WithStartRawInput()` (tool parameters)

4. **Tool Call Update Enhancement**
   - Include tool output content using `WithUpdateContent()`:
     - Text output → `ToolCallContent` with text
     - File diffs → `ToolCallContent` with diff (deferred to Feature 6.2)
   - Include raw output using `WithUpdateRawOutput()` (tool result)
   - Track status correctly (pending → in_progress → completed/failed)

5. **Tool Status Tracking**
   - `EventToolCallStart` → status: `pending`
   - `EventToolCallProgress` → status: `in_progress`
   - `EventToolCallComplete` (success) → status: `completed`
   - `EventToolCallComplete` (failure) → status: `failed`

## Technical Design

### Tool Kind Mapping Function

```go
func mapToolNameToKind(toolName string) *acp.ToolKind {
    switch toolName {
    case "read_file":
        return acp.Ptr(acp.ToolKindRead)
    case "write_file":
        return acp.Ptr(acp.ToolKindEdit)
    case "shell_command":
        return acp.Ptr(acp.ToolKindExecute)
    case "file_search":
        return acp.Ptr(acp.ToolKindSearch)
    case "list_directory":
        return acp.Ptr(acp.ToolKindRead)
    default:
        return nil // No kind specified
    }
}
```

### File Location Extraction

```go
func extractFileLocations(toolName string, params tools.ToolParameters) []acp.ToolCallLocation {
    var locations []acp.ToolCallLocation
    
    switch toolName {
    case "read_file", "write_file", "list_directory":
        if path, err := params.GetString("path"); err == nil && path != "" {
            locations = append(locations, acp.ToolCallLocation{
                Path: path,
            })
        }
    case "file_search":
        if root, err := params.GetString("workspace_root"); err == nil && root != "" {
            locations = append(locations, acp.ToolCallLocation{
                Path: root,
            })
        }
    }
    
    return locations
}
```

### Enhanced Tool Call Start

```go
func convertToolCallStart(event events.Event) (acp.SessionUpdate, bool) {
    data, ok := event.ToolCallStartData()
    if !ok {
        return acp.SessionUpdate{}, false
    }
    
    toolCallID := acp.ToolCallId(data.ToolID)
    title := data.ToolName
    
    // Map tool kind
    kind := mapToolNameToKind(data.ToolName)
    
    // Extract file locations
    locations := extractFileLocations(data.ToolName, data.Parameters)
    
    // Extract raw input (parameters)
    rawInput := extractRawInput(data.Parameters)
    
    // Build options
    opts := []acp.ToolCallStartOpt{}
    if kind != nil {
        opts = append(opts, acp.WithStartKind(*kind))
    }
    if len(locations) > 0 {
        opts = append(opts, acp.WithStartLocations(locations))
    }
    if rawInput != nil {
        opts = append(opts, acp.WithStartRawInput(rawInput))
    }
    
    update := acp.StartToolCall(toolCallID, title, opts...)
    return update, true
}
```

### Enhanced Tool Call Update

```go
func convertToolCallComplete(event events.Event) (acp.SessionUpdate, bool) {
    data, ok := event.ToolCallCompleteData()
    if !ok {
        return acp.SessionUpdate{}, false
    }
    
    toolCallID := acp.ToolCallId(data.ToolID)
    
    // Determine status
    var status acp.ToolCallStatus
    if data.Success {
        status = acp.ToolCallStatusCompleted
    } else {
        status = acp.ToolCallStatusFailed
    }
    
    // Build options
    opts := []acp.ToolCallUpdateOpt{
        acp.WithUpdateStatus(status),
    }
    
    // Add content if output is available
    if data.Output != "" {
        content := []acp.ToolCallContent{
            {
                Text: acp.Ptr(data.Output),
            },
        }
        opts = append(opts, acp.WithUpdateContent(content))
    }
    
    // Add raw output
    rawOutput := map[string]interface{}{
        "output": data.Output,
        "success": data.Success,
    }
    if data.Error != "" {
        rawOutput["error"] = data.Error
    }
    opts = append(opts, acp.WithUpdateRawOutput(rawOutput))
    
    update := acp.UpdateToolCall(toolCallID, opts...)
    return update, true
}
```

## Implementation Tasks

1. Create `mapToolNameToKind()` function
2. Create `extractFileLocations()` function
3. Create `extractRawInput()` helper function
4. Enhance `convertToolCallStart()` with kind, locations, and raw input
5. Enhance `convertToolCallProgress()` with status tracking
6. Enhance `convertToolCallComplete()` with content and raw output
7. Write unit tests for tool kind mapping
8. Write unit tests for file location extraction
9. Write integration tests for enhanced notifications
10. Update documentation

## Testing Requirements

### Unit Tests

- Test `mapToolNameToKind()` for all known tools
- Test `mapToolNameToKind()` for unknown tools (returns nil)
- Test `extractFileLocations()` for each tool type
- Test `extractFileLocations()` with missing parameters
- Test `extractRawInput()` with various parameter types

### Integration Tests

- Test `convertToolCallStart()` includes kind, locations, and raw input
- Test `convertToolCallProgress()` includes correct status
- Test `convertToolCallComplete()` includes content and raw output
- Test tool call lifecycle (start → progress → complete) with all enhancements

## Acceptance Criteria

- [x] Tool kind mapping implemented for all known tools
- [ ] File locations extracted from tool parameters
- [ ] Tool call start notifications include kind, locations, and raw input
- [ ] Tool call update notifications include content and raw output
- [ ] Tool status tracking is correct throughout lifecycle
- [ ] Unit tests with ≥90% coverage
- [ ] Integration tests passing
- [ ] No lint errors
- [ ] Documentation updated

## Dependencies

- Feature 4.2 completed (Event to ACP Notification Converter)
- ACP SDK v0.6.3 with tool kind and location support

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md#feature-61-tool-call-notification-details)
- [ACP SDK Integration](../../docs/packages/acp-sdk-integration.md)
- [ACP Protocol Specification](https://agentclientprotocol.com/protocol/tool-calls)

## Notes

- File diff generation is deferred to Feature 6.2
- Line number extraction is a future enhancement
- Tool kind mapping can be extended as new tools are added

