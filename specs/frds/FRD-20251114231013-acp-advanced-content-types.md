# FRD: ACP Advanced Content Types Support

**Feature ID**: FRD-20251114231013  
**Feature**: Advanced Content Types (Image, Audio, Embedded Resources)  
**Roadmap Item**: Feature 9.3  
**Status**: In Progress  
**Created**: 2025-11-14

## Overview

Implement support for advanced content types in ACP protocol: image blocks, audio blocks, and enhanced embedded resource support. This feature completes the content type support that was partially implemented in Feature 4.1.

## Background

Currently, the ACP implementation only supports:
- ✅ Text blocks (fully supported)
- ✅ ResourceLink blocks (basic support)
- ✅ Resource blocks (partial support for text and blob resources)

However, the capability advertisement in `Initialize()` claims:
- `Image: true` (but not actually implemented)
- `Audio: false` (correctly advertised as not supported)
- `EmbeddedContext: true` (partially implemented)

The `convertACPContentBlocksToMessages()` function has a comment: "Image and Audio blocks not yet supported" (line 495 in `agent.go`).

## Requirements

### Functional Requirements

1. **Image Content Block Support**
   - **Inbound**: Convert ACP `ContentBlockImage` → Spin message format
   - **Outbound**: Convert Spin image content → ACP `ContentBlockImage` in notifications
   - Support base64-encoded image data
   - Support MIME type specification (image/png, image/jpeg, image/gif, etc.)
   - Use SDK helper `acp.ImageBlock(data, mimeType)`

2. **Audio Content Block Support**
   - **Inbound**: Convert ACP `ContentBlockAudio` → Spin message format
   - **Outbound**: Convert Spin audio content → ACP `ContentBlockAudio` in notifications
   - Support base64-encoded audio data
   - Support MIME type specification (audio/mpeg, audio/wav, etc.)
   - Use SDK helper `acp.AudioBlock(data, mimeType)`
   - **Note**: Spin's internal `ContentItem` doesn't support audio, so we may need to handle this as text description or skip conversion

3. **Enhanced Embedded Resource Support**
   - Improve `ContentBlockResource` handling
   - Support both `TextResourceContents` and `BlobResourceContents`
   - Properly extract and include resource name and MIME type
   - Use SDK helper `acp.ResourceBlock()` when sending

4. **Capability Advertisement**
   - Update `buildAgentCapabilities()` to accurately reflect support:
     - `Image: true` (only if fully implemented)
     - `Audio: true/false` (based on actual support)
     - `EmbeddedContext: true` (if fully implemented)

5. **Notification Support**
   - Ensure notifications can send image/audio content blocks
   - Update `convertEventToSessionUpdate()` if needed
   - Support image/audio in `agent_message_chunk` notifications

### Technical Requirements

1. **Conversion Functions**
   - Enhance `convertACPContentBlocksToMessages()` to handle Image and Audio blocks
   - Create `convertSpinContentToACPBlocks()` for outbound conversion (if needed)
   - Handle base64 encoding/decoding correctly
   - Validate MIME types

2. **Message Format**
   - Determine how to represent image/audio in Spin's `message.Message` format
   - `message.Message` only has `Content string` field
   - May need to use `protocol.ContentItem[]` format instead
   - Check if agent execution supports structured content

3. **Error Handling**
   - Handle invalid base64 data
   - Handle unsupported MIME types gracefully
   - Return appropriate errors for malformed content blocks

4. **Testing**
   - Unit tests for all content block conversions
   - Integration tests with real image/audio data
   - Test capability advertisement accuracy
   - Test notification sending with image/audio content

## Design

### Inbound Conversion (ACP → Spin)

```go
// Enhanced convertACPContentBlocksToMessages
func convertACPContentBlocksToMessages(blocks []acp.ContentBlock) ([]message.Message, error) {
    for _, block := range blocks {
        switch {
        case block.Text != nil:
            // Existing text handling
        case block.Image != nil:
            // NEW: Convert image block
            // Option 1: Include in message.Content as text description
            // Option 2: Use protocol.ContentItem format (if agent supports it)
        case block.Audio != nil:
            // NEW: Convert audio block
            // Similar to image - may need special handling
        case block.ResourceLink != nil:
            // Existing resource link handling
        case block.Resource != nil:
            // Enhanced resource handling
        }
    }
}
```

### Outbound Conversion (Spin → ACP)

For notifications, convert Spin content to ACP blocks:
- If `protocol.ContentItem` has `Image`, use `acp.ImageBlock()`
- If `protocol.ContentItem` has audio (if we add support), use `acp.AudioBlock()`
- Use SDK helpers for all conversions

### Capability Advertisement

```go
func (a *SpinACPAgent) buildAgentCapabilities() acp.AgentCapabilities {
    return acp.AgentCapabilities{
        PromptCapabilities: acp.PromptCapabilities{
            Image:           true,  // Only if fully implemented
            Audio:           false, // Or true if we implement it
            EmbeddedContext: true,  // Only if fully implemented
        },
    }
}
```

## Implementation Plan

1. **Phase 1: Image Block Support**
   - Implement inbound image block conversion
   - Implement outbound image block conversion (notifications)
   - Add unit tests
   - Update capability advertisement

2. **Phase 2: Audio Block Support**
   - Research Spin's audio support capabilities
   - Implement inbound audio block conversion (may be limited)
   - Implement outbound audio block conversion (if possible)
   - Add unit tests
   - Update capability advertisement

3. **Phase 3: Enhanced Embedded Resources**
   - Improve resource block handling
   - Support all resource content types
   - Add unit tests

4. **Phase 4: Integration Testing**
   - End-to-end tests with image/audio content
   - Test notification sending
   - Verify capability advertisement

## Acceptance Criteria

- [ ] Image blocks can be received in `PromptRequest` and converted to Spin messages
- [ ] Image blocks can be sent in `agent_message_chunk` notifications
- [ ] Audio blocks can be received in `PromptRequest` (even if converted to text description)
- [ ] Audio blocks can be sent in `agent_message_chunk` notifications (if supported)
- [ ] Embedded resources are fully supported (text and blob)
- [ ] Capability advertisement accurately reflects actual support
- [ ] All conversions use SDK helper functions (`acp.ImageBlock`, `acp.AudioBlock`, `acp.ResourceBlock`)
- [ ] Unit tests cover all content block types
- [ ] Integration tests verify end-to-end functionality
- [ ] Error handling for invalid/malformed content blocks
- [ ] Documentation updated

## Testing Strategy

### Unit Tests

1. **Image Block Conversion**
   - Test base64 image data conversion
   - Test MIME type handling
   - Test invalid base64 data
   - Test missing MIME type

2. **Audio Block Conversion**
   - Test base64 audio data conversion
   - Test MIME type handling
   - Test invalid base64 data

3. **Resource Block Conversion**
   - Test text resource contents
   - Test blob resource contents
   - Test missing resource data

4. **Capability Advertisement**
   - Test accurate capability reporting

### Integration Tests

1. **End-to-End Image Flow**
   - Send `PromptRequest` with image block
   - Verify image is processed correctly
   - Verify image appears in notifications

2. **End-to-End Audio Flow**
   - Send `PromptRequest` with audio block
   - Verify audio is processed (or converted appropriately)

3. **Mixed Content Blocks**
   - Test prompt with text + image + resource blocks
   - Verify all are converted correctly

## Dependencies

- Feature 4.1 (Prompt Method) - ✅ Completed
- SDK helpers (`acp.ImageBlock`, `acp.AudioBlock`, `acp.ResourceBlock`)
- Spin's `protocol.ContentItem` support for images

## Risks

1. **Spin Message Format Limitation**
   - `message.Message` only has `Content string` field
   - May need to use `protocol.ContentItem[]` format
   - Need to verify agent execution supports structured content

2. **Audio Support**
   - Spin may not have native audio support
   - May need to convert audio to text description
   - Or skip audio support entirely

3. **Base64 Encoding**
   - Large images/audio may cause performance issues
   - Need to handle encoding/decoding efficiently

## Open Questions

1. How should image/audio be represented in `message.Message`?
   - Use `Content string` with description?
   - Use `protocol.ContentItem[]` format?
   - Check if agent execution supports structured content

2. Should we support audio if Spin doesn't have native audio support?
   - Option A: Convert to text description
   - Option B: Skip audio support, keep `Audio: false`
   - Option C: Add audio support to Spin's content types

3. Should we validate MIME types?
   - Strict validation (only known types)?
   - Lenient validation (accept any string)?

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md) - Feature 9.3
- [ACP SDK Integration](../../docs/packages/acp-sdk-integration.md)
- [ACP Type Mapping](../../docs/packages/acp-type-mapping.md)
- [ACP Protocol Implementation](../../docs/packages/protocol-acp.md)

## Notes

- Current implementation in `convertACPContentBlocksToMessages()` (line 454-503 in `agent.go`) only handles Text, ResourceLink, and Resource blocks
- Capability advertisement claims `Image: true` but it's not implemented
- SDK provides helpers: `acp.ImageBlock()`, `acp.AudioBlock()`, `acp.ResourceBlock()`
- Spin's `protocol.ContentItem` supports `ImageContent` but not audio

