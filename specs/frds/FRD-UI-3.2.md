# FRD-UI-3.2: Chat Interface Components

**Feature:** Chat Display and Transcript Rendering
**Phase:** 3.2
**Priority:** High
**Status:** In Progress
**Created:** 2025-10-05

---

## 1. Overview

Implement the core chat interface components for Spin's TUI, providing rich display of conversations with streaming AI responses, markdown formatting, syntax highlighting, and smooth scrolling.

**Goals:**
- Display conversation transcript with proper formatting
- Stream AI responses in real-time without flickering
- Render markdown with glamour
- Highlight code blocks with chroma
- Support ANSI color preservation
- Display reasoning blocks for compatible models
- Smooth scrolling with viewport
- Memory efficient (<30MB for large transcripts)

---

## 2. Technical Design

### 2.1 Architecture

```
┌─────────────────────────────────────────┐
│         Chat Interface (chat.go)        │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │   Viewport (Bubble Tea Component) │ │
│  │                                   │ │
│  │  ┌─────────────────────────────┐ │ │
│  │  │  Message Renderer           │ │ │
│  │  │  - User messages            │ │ │
│  │  │  - Assistant messages       │ │ │
│  │  │  - System messages          │ │ │
│  │  │  - Tool calls/results       │ │ │
│  │  └─────────────────────────────┘ │ │
│  │                                   │ │
│  │  ┌─────────────────────────────┐ │ │
│  │  │  Content Formatter          │ │ │
│  │  │  - Markdown (glamour)       │ │ │
│  │  │  - Code blocks (chroma)     │ │ │
│  │  │  - ANSI preservation        │ │ │
│  │  │  - Reasoning blocks         │ │ │
│  │  └─────────────────────────────┘ │ │
│  └───────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

### 2.2 Package Structure

```
internal/tui/ui/
├── chat.go           # Main chat component
├── chat_test.go      # Chat tests
├── message.go        # Message types and rendering
├── message_test.go   # Message tests
├── formatter.go      # Content formatting (markdown, code)
├── formatter_test.go # Formatter tests
└── styles.go         # Styling (will use from Phase 3.10)
```

### 2.3 Message Types

```go
// MessageRole represents who sent the message
type MessageRole string

const (
    RoleUser      MessageRole = "user"
    RoleAssistant MessageRole = "assistant"
    RoleSystem    MessageRole = "system"
    RoleTool      MessageRole = "tool"
)

// Message represents a single chat message
type Message struct {
    Role      MessageRole
    Content   string
    Timestamp time.Time
    Streaming bool        // True if still streaming
    ToolCall  *ToolCall   // If this is a tool call
    ToolResult *ToolResult // If this is a tool result
    Reasoning  string      // Reasoning block (if present)
}

// ToolCall represents a tool invocation
type ToolCall struct {
    Name      string
    Arguments map[string]interface{}
    ID        string
}

// ToolResult represents a tool execution result
type ToolResult struct {
    ToolCallID string
    Output     string
    Error      string
}
```

### 2.4 Chat Component

```go
// Chat represents the chat interface component
type Chat struct {
    viewport viewport.Model
    messages []Message
    width    int
    height   int

    // Rendering state
    content       string  // Cached rendered content
    contentDirty  bool    // Needs re-render
    atBottom      bool    // Auto-scroll to bottom

    // Formatters
    mdRenderer    *glamour.TermRenderer
    codeHighlight func(code, lang string) string
}

// NewChat creates a new chat component
func NewChat(width, height int) Chat

// SetSize updates the chat dimensions
func (c *Chat) SetSize(width, height int)

// AddMessage adds a new message to the chat
func (c *Chat) AddMessage(msg Message)

// UpdateLastMessage updates the last message (for streaming)
func (c *Chat) UpdateLastMessage(content string)

// View renders the chat to a string
func (c Chat) View() string

// Update handles Bubble Tea messages
func (c Chat) Update(msg tea.Msg) (Chat, tea.Cmd)
```

---

## 3. Implementation Details

### 3.1 Dependencies

**Required Packages:**
```go
// Bubble Tea ecosystem
"github.com/charmbracelet/bubbles/viewport"
"github.com/charmbracelet/lipgloss"

// Markdown rendering
"github.com/charmbracelet/glamour"

// Syntax highlighting
"github.com/alecthomas/chroma/v2"
"github.com/alecthomas/chroma/v2/formatters"
"github.com/alecthomas/chroma/v2/lexers"
"github.com/alecthomas/chroma/v2/styles"
```

### 3.2 Markdown Rendering with Glamour

```go
// createMarkdownRenderer creates a glamour renderer for terminal
func createMarkdownRenderer(width int) (*glamour.TermRenderer, error) {
    return glamour.NewTermRenderer(
        glamour.WithAutoStyle(),          // Auto light/dark
        glamour.WithWordWrap(width - 4),  // Wrap with margin
        glamour.WithPreservedNewLines(),  // Keep line breaks
    )
}

// renderMarkdown renders markdown content
func (c *Chat) renderMarkdown(content string) (string, error) {
    if c.mdRenderer == nil {
        var err error
        c.mdRenderer, err = createMarkdownRenderer(c.width)
        if err != nil {
            return content, err
        }
    }

    rendered, err := c.mdRenderer.Render(content)
    if err != nil {
        return content, err  // Fallback to raw content
    }

    return rendered, nil
}
```

### 3.3 Code Syntax Highlighting with Chroma

```go
// highlightCode applies syntax highlighting to code blocks
func highlightCode(code, language string) string {
    // Get lexer for language
    lexer := lexers.Get(language)
    if lexer == nil {
        lexer = lexers.Fallback
    }

    // Get style (monokai for dark terminals, github for light)
    style := styles.Get("monokai")
    if style == nil {
        style = styles.Fallback
    }

    // Format for terminal with 256 colors
    formatter := formatters.Get("terminal256")
    if formatter == nil {
        formatter = formatters.Fallback
    }

    // Tokenize and format
    iterator, err := lexer.Tokenise(nil, code)
    if err != nil {
        return code
    }

    var buf bytes.Buffer
    err = formatter.Format(&buf, style, iterator)
    if err != nil {
        return code
    }

    return buf.String()
}
```

### 3.4 Message Rendering

```go
// renderMessage renders a single message with appropriate styling
func (c *Chat) renderMessage(msg Message) string {
    var parts []string

    // Header with role and timestamp
    header := c.renderMessageHeader(msg)
    parts = append(parts, header)

    // Reasoning block (if present)
    if msg.Reasoning != "" {
        reasoning := c.renderReasoning(msg.Reasoning)
        parts = append(parts, reasoning)
    }

    // Main content
    content := c.renderMessageContent(msg)
    parts = append(parts, content)

    // Tool call/result (if present)
    if msg.ToolCall != nil {
        toolCall := c.renderToolCall(msg.ToolCall)
        parts = append(parts, toolCall)
    }
    if msg.ToolResult != nil {
        toolResult := c.renderToolResult(msg.ToolResult)
        parts = append(parts, toolResult)
    }

    // Streaming indicator
    if msg.Streaming {
        parts = append(parts, "▊") // Blinking cursor
    }

    return strings.Join(parts, "\n")
}

// renderMessageContent renders message content with formatting
func (c *Chat) renderMessageContent(msg Message) string {
    switch msg.Role {
    case RoleUser:
        return c.renderUserMessage(msg.Content)
    case RoleAssistant:
        return c.renderAssistantMessage(msg.Content)
    case RoleSystem:
        return c.renderSystemMessage(msg.Content)
    case RoleTool:
        return c.renderToolMessage(msg.Content)
    default:
        return msg.Content
    }
}

// renderAssistantMessage renders AI responses with markdown/code
func (c *Chat) renderAssistantMessage(content string) string {
    // Detect and extract code blocks
    codeBlocks := extractCodeBlocks(content)

    if len(codeBlocks) == 0 {
        // Pure markdown, no code blocks
        rendered, _ := c.renderMarkdown(content)
        return rendered
    }

    // Mixed content: render markdown and highlight code separately
    result := content
    for _, block := range codeBlocks {
        highlighted := highlightCode(block.Code, block.Language)
        result = strings.Replace(result, block.Raw, highlighted, 1)
    }

    return result
}
```

### 3.5 Streaming Delta Updates

```go
// StreamDelta appends content to the last message (streaming)
func (c *Chat) StreamDelta(delta string) {
    if len(c.messages) == 0 {
        return
    }

    lastIdx := len(c.messages) - 1
    c.messages[lastIdx].Content += delta
    c.messages[lastIdx].Streaming = true
    c.contentDirty = true

    // Auto-scroll to bottom if user is already at bottom
    if c.atBottom {
        c.viewport.GotoBottom()
    }
}

// FinishStreaming marks the last message as complete
func (c *Chat) FinishStreaming() {
    if len(c.messages) == 0 {
        return
    }

    lastIdx := len(c.messages) - 1
    c.messages[lastIdx].Streaming = false
    c.contentDirty = true
}
```

### 3.6 Viewport Integration

```go
// Update handles viewport scrolling
func (c Chat) Update(msg tea.Msg) (Chat, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "pgup", "pgdown", "up", "down":
            c.viewport, cmd = c.viewport.Update(msg)
            c.updateScrollState()
            return c, cmd
        }
    }

    // Check if content needs re-render
    if c.contentDirty {
        c.renderContent()
        c.viewport.SetContent(c.content)
        c.contentDirty = false
    }

    c.viewport, cmd = c.viewport.Update(msg)
    return c, cmd
}

// renderContent renders all messages to a string
func (c *Chat) renderContent() {
    var parts []string

    for _, msg := range c.messages {
        rendered := c.renderMessage(msg)
        parts = append(parts, rendered)
        parts = append(parts, "") // Blank line between messages
    }

    c.content = strings.Join(parts, "\n")
}

// updateScrollState checks if viewport is at bottom
func (c *Chat) updateScrollState() {
    c.atBottom = c.viewport.AtBottom()
}
```

### 3.7 ANSI Color Preservation

```go
// preserveANSI ensures ANSI escape codes are preserved in output
func preserveANSI(content string) string {
    // glamour and chroma already preserve ANSI
    // Just ensure we don't strip them
    return content
}
```

### 3.8 Reasoning Block Display

```go
// renderReasoning renders a reasoning block with special styling
func (c *Chat) renderReasoning(reasoning string) string {
    // Box style for reasoning
    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("63")). // Purple
        Padding(0, 1).
        Render("💭 " + reasoning)
}
```

---

## 4. Testing Strategy

### 4.1 Unit Tests

```go
func TestChat_AddMessage(t *testing.T) {
    c := NewChat(80, 24)

    msg := Message{
        Role:    RoleUser,
        Content: "Hello",
    }

    c.AddMessage(msg)

    assert.Len(t, c.messages, 1)
    assert.Equal(t, "Hello", c.messages[0].Content)
}

func TestChat_StreamDelta(t *testing.T) {
    c := NewChat(80, 24)
    c.AddMessage(Message{Role: RoleAssistant, Content: "Hel"})

    c.StreamDelta("lo")

    assert.Equal(t, "Hello", c.messages[0].Content)
    assert.True(t, c.messages[0].Streaming)
}

func TestChat_MarkdownRendering(t *testing.T) {
    c := NewChat(80, 24)

    markdown := "# Title\n\nSome **bold** text"
    rendered, err := c.renderMarkdown(markdown)

    assert.NoError(t, err)
    assert.Contains(t, rendered, "Title")
    assert.Contains(t, rendered, "bold")
}

func TestChat_CodeHighlighting(t *testing.T) {
    code := "func main() {\n\tfmt.Println(\"hello\")\n}"
    highlighted := highlightCode(code, "go")

    // Should contain ANSI codes
    assert.Contains(t, highlighted, "\x1b[")
    assert.Contains(t, highlighted, "main")
}
```

### 4.2 Snapshot Tests

```go
func TestChat_ViewSnapshot(t *testing.T) {
    c := NewChat(80, 24)
    c.AddMessage(Message{Role: RoleUser, Content: "Hello"})
    c.AddMessage(Message{Role: RoleAssistant, Content: "Hi there!"})

    view := c.View()

    // Compare with golden file
    golden.Assert(t, view, "chat_basic.golden")
}
```

### 4.3 Performance Tests

```go
func BenchmarkChat_RenderLargeTranscript(b *testing.B) {
    c := NewChat(80, 24)

    // Add 1000 messages
    for i := 0; i < 1000; i++ {
        c.AddMessage(Message{
            Role:    RoleAssistant,
            Content: fmt.Sprintf("Message %d with some content", i),
        })
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        c.renderContent()
    }
}
```

---

## 5. Performance Requirements

### 5.1 Rendering Performance

**Targets:**
- Initial render: <50ms for 100 messages
- Streaming delta: <5ms per update
- Scroll: <16ms (60 FPS)
- Memory: <30MB for 1000 messages

**Optimizations:**
- Lazy rendering (only visible viewport)
- Cached markdown rendering
- Efficient string building
- Viewport pagination

### 5.2 Memory Management

```go
// Limit transcript size to prevent memory growth
const MaxTranscriptMessages = 1000

func (c *Chat) AddMessage(msg Message) {
    c.messages = append(c.messages, msg)

    // Trim old messages if limit exceeded
    if len(c.messages) > MaxTranscriptMessages {
        // Keep last MaxTranscriptMessages
        c.messages = c.messages[len(c.messages)-MaxTranscriptMessages:]
    }

    c.contentDirty = true
}
```

---

## 6. Integration with TUI Model

### 6.1 Model Update

```go
// In internal/tui/app.go

type Model struct {
    // ... existing fields ...
    chat Chat  // Add chat component
}

func NewModel() Model {
    return Model{
        state: StateIdle,
        chat:  ui.NewChat(0, 0),  // Will be sized on first resize
    }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height

        // Size chat (leaving room for input and status bar)
        chatHeight := m.height - 3  // 1 for input, 1 for status, 1 for margin
        m.chat.SetSize(m.width, chatHeight)

        return m, nil

    // ... other cases ...
    }

    // Update chat component
    var cmd tea.Cmd
    m.chat, cmd = m.chat.Update(msg)

    return m, cmd
}
```

### 6.2 View Update

```go
// In internal/tui/view.go

func (m Model) renderPlaceholder() string {
    if m.quitting {
        return ""
    }

    // Render chat
    chatView := m.chat.View()

    // Placeholder input (Phase 3.3 will implement real input)
    inputView := "Type your message..."

    // Placeholder status bar (Phase 3.6 will implement real status)
    statusView := fmt.Sprintf("State: %s | %dx%d", m.state, m.width, m.height)

    return lipgloss.JoinVertical(
        lipgloss.Top,
        chatView,
        inputView,
        statusView,
    )
}
```

---

## 7. Quality Checklist

### 7.1 Definition of Ready (DoR)

- [x] UI component spec reviewed
- [x] Rendering pipeline understood
- [x] Markdown/code highlighting requirements clear
- [x] Dependencies identified (glamour, chroma, bubbles)

### 7.2 Definition of Done (DoD)

- [ ] Tests for rendering (≥80% coverage)
- [ ] Snapshot tests for layout
- [ ] Markdown renders correctly
- [ ] Code blocks highlighted
- [ ] Streaming is smooth (no flickering)
- [ ] Memory usage <30MB for large transcripts
- [ ] All tests passing with race detector
- [ ] Linter clean (make lint)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] ROADMAP updated

---

## 8. Risks and Mitigations

### 8.1 Risks

1. **Glamour/Chroma compatibility** - May have conflicts or rendering issues
   - **Mitigation:** Fallback to plain text if rendering fails, comprehensive tests

2. **Performance with large transcripts** - Could slow down with 1000+ messages
   - **Mitigation:** Viewport pagination, lazy rendering, message limit

3. **ANSI escape code conflicts** - Multiple formatters may interfere
   - **Mitigation:** Careful ordering, test with various content types

4. **Flickering during streaming** - Rapid updates may cause flicker
   - **Mitigation:** Debouncing, efficient re-rendering, viewport optimization

---

## 9. Success Criteria

Phase 3.2 is complete when:

1. ✅ Chat component displays messages with roles
2. ✅ Markdown renders beautifully with glamour
3. ✅ Code blocks syntax-highlighted with chroma
4. ✅ Streaming deltas work smoothly without flicker
5. ✅ Viewport scrolls efficiently
6. ✅ Reasoning blocks display correctly
7. ✅ All tests passing with ≥80% coverage
8. ✅ Performance targets met
9. ✅ Memory usage <30MB for large transcripts

---

## 10. Future Enhancements (Out of Scope)

- Message editing (Phase 3.8 - Backtrack mode)
- Search within transcript (Phase 3.7)
- Export conversation (Phase 3.7)
- Custom themes (Phase 3.10)
- Image/file attachments
- Message reactions
- Collaborative features

---

## 11. References

- [Bubble Tea Viewport](https://github.com/charmbracelet/bubbles/tree/master/viewport)
- [Glamour Documentation](https://github.com/charmbracelet/glamour)
- [Chroma Documentation](https://github.com/alecthomas/chroma)
- [specs/ui-modules/spec.md](../ui-modules/spec.md)
- [specs/ui-modules/ROADMAP.md](../ui-modules/ROADMAP.md)
- [FRD-UI-3.1.md](FRD-UI-3.1.md) - Foundation

---

**Document Version:** 1.0
**Last Updated:** 2025-10-05
**Author:** Spin Development Team
