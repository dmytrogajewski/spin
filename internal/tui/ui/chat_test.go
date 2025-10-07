package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewChat(t *testing.T) {
	c := NewChat(80, 24)

	assert.Equal(t, 80, c.width)
	assert.Equal(t, 24, c.height)
	assert.Empty(t, c.messages)
	assert.True(t, c.atBottom)
	assert.NotNil(t, c.formatter)
}

func TestChat_SetSize(t *testing.T) {
	c := NewChat(80, 24)

	c.SetSize(120, 40)

	assert.Equal(t, 120, c.width)
	assert.Equal(t, 40, c.height)
}

func TestChat_AddMessage(t *testing.T) {
	c := NewChat(80, 24)

	msg := NewUserMessage("Hello")
	c.AddMessage(msg)

	assert.Len(t, c.messages, 1)
	assert.Equal(t, "Hello", c.messages[0].Content)
	assert.True(t, c.contentDirty)
}

func TestChat_AddMessage_TrimLimit(t *testing.T) {
	c := NewChat(80, 24)

	// Add messages beyond limit
	for i := 0; i < MaxTranscriptMessages+100; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}

	// Should trim to max
	assert.Len(t, c.messages, MaxTranscriptMessages)
}

func TestChat_StreamDelta(t *testing.T) {
	c := NewChat(80, 24)

	// Add initial streaming message
	c.AddMessage(NewStreamingMessage("Hel"))

	// Stream more content
	c.StreamDelta("lo")

	assert.Equal(t, "Hello", c.messages[0].Content)
	assert.True(t, c.messages[0].Streaming)
	assert.True(t, c.contentDirty)
}

func TestChat_StreamDelta_NoMessages(t *testing.T) {
	c := NewChat(80, 24)

	// Should not panic with no messages
	c.StreamDelta("test")

	assert.Empty(t, c.messages)
}

func TestChat_FinishStreaming(t *testing.T) {
	c := NewChat(80, 24)

	c.AddMessage(NewStreamingMessage("Hello"))
	assert.True(t, c.messages[0].Streaming)

	c.FinishStreaming()

	assert.False(t, c.messages[0].Streaming)
	assert.True(t, c.contentDirty)
}

func TestChat_FinishStreaming_NoMessages(t *testing.T) {
	c := NewChat(80, 24)

	// Should not panic
	c.FinishStreaming()

	assert.Empty(t, c.messages)
}

func TestChat_View_Empty(t *testing.T) {
	c := NewChat(80, 24)

	view := c.View()

	// Should return viewport view (may be empty or contain viewport chrome)
	assert.NotNil(t, view)
}

func TestChat_View_WithMessages(t *testing.T) {
	c := NewChat(80, 24)

	c.AddMessage(NewUserMessage("Hello"))
	c.AddMessage(NewAssistantMessage("Hi there!"))

	// Trigger update to render content
	c, _ = c.Update(nil)

	view := c.View()

	// Should contain message content
	assert.Contains(t, view, "Hello")
	assert.Contains(t, view, "Hi there")
}

func TestChat_Update_Resize(t *testing.T) {
	c := NewChat(80, 24)

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updated, _ := c.Update(msg)

	assert.Equal(t, 100, updated.width)
	assert.Equal(t, 30, updated.height)
}

func TestChat_RenderMessage_User(t *testing.T) {
	c := NewChat(80, 24)

	msg := NewUserMessage("Hello world")
	rendered := c.renderMessage(msg)

	assert.Contains(t, rendered, "Hello world")
	assert.Contains(t, rendered, "You") // User header
}

func TestChat_RenderMessage_Assistant(t *testing.T) {
	c := NewChat(80, 24)

	msg := NewAssistantMessage("Hi there!")
	rendered := c.renderMessage(msg)

	assert.Contains(t, rendered, "Hi there")
	assert.Contains(t, rendered, "Assistant") // Assistant header
}

func TestChat_RenderMessage_Streaming(t *testing.T) {
	c := NewChat(80, 24)

	msg := NewStreamingMessage("Typing...")
	rendered := c.renderMessage(msg)

	assert.Contains(t, rendered, "Typing")
	assert.Contains(t, rendered, "▊") // Streaming cursor
}

func TestChat_RenderMessage_WithThinking(t *testing.T) {
	c := NewChat(80, 24)
	c.ToggleThinking() // Expand thinking content

	msg := NewAssistantMessage("I'll help")
	msg.Thinking = "User needs assistance"

	rendered := c.renderMessage(msg)

	assert.Contains(t, rendered, "User needs assistance")
	assert.Contains(t, rendered, "💭") // Reasoning emoji
}

func TestChat_RenderContent_Integration(t *testing.T) {
	c := NewChat(80, 24)

	// Add messages and render
	c.AddMessage(NewUserMessage("Test"))
	c.renderContent()

	assert.NotEmpty(t, c.content)
	assert.Contains(t, c.content, "Test")
}

func TestChat_AutoScroll(t *testing.T) {
	c := NewChat(80, 10)

	// Add many messages to trigger scrolling
	for i := 0; i < 20; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}

	// Auto-scroll should be enabled
	assert.True(t, c.atBottom)
}

func TestChat_RenderToolCall(t *testing.T) {
	c := NewChat(80, 24)

	toolCall := &ToolCall{
		Name: "shell",
		Arguments: map[string]interface{}{
			"command": "ls -la",
		},
		ID: "call_123",
	}

	msg := NewToolMessage("Executing command", toolCall, nil)
	rendered := c.renderMessage(msg)

	assert.Contains(t, rendered, "shell")
	assert.Contains(t, rendered, "ls -la")
}

func TestChat_RenderToolResult(t *testing.T) {
	c := NewChat(80, 24)

	toolResult := &ToolResult{
		ToolCallID: "call_123",
		Output:     "file1.txt\nfile2.txt",
	}

	msg := NewToolMessage("Command completed", nil, toolResult)
	rendered := c.renderMessage(msg)

	assert.Contains(t, rendered, "file1.txt")
	assert.Contains(t, rendered, "file2.txt")
}

func TestChat_GetMessages(t *testing.T) {
	c := NewChat(80, 24)

	c.AddMessage(NewUserMessage("Msg 1"))
	c.AddMessage(NewAssistantMessage("Msg 2"))

	messages := c.GetMessages()

	assert.Len(t, messages, 2)
	assert.Equal(t, "Msg 1", messages[0].Content)
	assert.Equal(t, "Msg 2", messages[1].Content)
}

func TestChat_Clear(t *testing.T) {
	c := NewChat(80, 24)

	c.AddMessage(NewUserMessage("Test"))
	assert.Len(t, c.messages, 1)

	c.Clear()

	assert.Empty(t, c.messages)
	assert.True(t, c.contentDirty)
}

// Scroll Navigation Tests

func TestChat_ScrollPercent_Initial(t *testing.T) {
	c := NewChat(80, 24)

	// Initially at 100% (bottom)
	assert.Equal(t, 100.0, c.ScrollPercent())
}

func TestChat_PageUp(t *testing.T) {
	c := NewChat(80, 10)

	// Add many messages to enable scrolling
	for i := 0; i < 50; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}
	c.renderContent()
	c.viewport.SetContent(c.content)

	c.PageUp()

	// Should mark as user scrolled
	assert.True(t, c.userScrolled)
	// Scroll position should not be 100%
	assert.Less(t, c.scrollPercent, 100.0)
}

func TestChat_PageDown(t *testing.T) {
	c := NewChat(80, 10)

	// Add many messages
	for i := 0; i < 50; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}
	c.renderContent()
	c.viewport.SetContent(c.content)

	// Go to top first
	c.GotoTop()
	assert.True(t, c.userScrolled)

	// Now page down
	c.PageDown()

	// Should still be user scrolled
	assert.True(t, c.userScrolled)
}

func TestChat_GotoTop(t *testing.T) {
	c := NewChat(80, 10)

	// Add many messages
	for i := 0; i < 50; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}
	c.renderContent()
	c.viewport.SetContent(c.content)

	c.GotoTop()

	// Should be at top (0%)
	assert.Equal(t, 0.0, c.scrollPercent)
	assert.True(t, c.userScrolled)
}

func TestChat_GotoBottom(t *testing.T) {
	c := NewChat(80, 10)

	// Add many messages
	for i := 0; i < 50; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}
	c.renderContent()
	c.viewport.SetContent(c.content)

	// Go to top first
	c.GotoTop()
	assert.True(t, c.userScrolled)

	// Go back to bottom
	c.GotoBottom()

	// Should be at bottom (100%)
	assert.Equal(t, 100.0, c.scrollPercent)
	// User scrolled should be reset (re-enable auto-scroll)
	assert.False(t, c.userScrolled)
}

func TestChat_ResetUserScroll(t *testing.T) {
	c := NewChat(80, 10)

	c.PageUp()
	assert.True(t, c.userScrolled)

	c.ResetUserScroll()
	assert.False(t, c.userScrolled)
}

func TestChat_Update_PgUpKey(t *testing.T) {
	c := NewChat(80, 10)

	// Add messages
	for i := 0; i < 50; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}
	c.renderContent()
	c.viewport.SetContent(c.content)
	c.updateScrollState()

	// Simulate PgUp key
	msg := tea.KeyMsg{Type: tea.KeyPgUp}
	updated, _ := c.Update(msg)

	// Should have scrolled
	assert.True(t, updated.userScrolled)
}

func TestChat_Update_PgDownKey(t *testing.T) {
	c := NewChat(80, 10)

	// Add messages
	for i := 0; i < 50; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}
	c.renderContent()
	c.viewport.SetContent(c.content)
	c.updateScrollState()
	c.GotoTop()

	// Simulate PgDown key
	msg := tea.KeyMsg{Type: tea.KeyPgDown}
	updated, _ := c.Update(msg)

	// Should have scrolled
	assert.True(t, updated.userScrolled)
}

func TestChat_Update_HomeKey(t *testing.T) {
	c := NewChat(80, 10)

	// Add messages
	for i := 0; i < 50; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}
	c.renderContent()
	c.viewport.SetContent(c.content)
	c.updateScrollState()

	// Simulate Home key
	msg := tea.KeyMsg{Type: tea.KeyHome}
	updated, _ := c.Update(msg)

	// Should be at top
	assert.Equal(t, 0.0, updated.scrollPercent)
}

func TestChat_Update_EndKey(t *testing.T) {
	c := NewChat(80, 10)

	// Add messages
	for i := 0; i < 50; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}
	c.renderContent()
	c.viewport.SetContent(c.content)
	c.updateScrollState()
	c.GotoTop()

	// Simulate End key
	msg := tea.KeyMsg{Type: tea.KeyEnd}
	updated, _ := c.Update(msg)

	// Should be at bottom
	assert.Equal(t, 100.0, updated.scrollPercent)
	assert.False(t, updated.userScrolled) // Auto-scroll re-enabled
}

func TestChat_AutoScroll_DisabledWhenUserScrolled(t *testing.T) {
	c := NewChat(80, 10)

	// Add initial messages
	for i := 0; i < 20; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}
	c.renderContent()
	c.viewport.SetContent(c.content)

	// User scrolls up
	c.PageUp()
	assert.True(t, c.userScrolled)

	// Add new message
	c.AddMessage(NewUserMessage("New"))
	c, _ = c.Update(nil) // Trigger update

	// Should NOT auto-scroll to bottom
	assert.True(t, c.userScrolled)
	assert.Less(t, c.scrollPercent, 99.0)
}

func TestChat_AutoScroll_EnabledWhenAtBottom(t *testing.T) {
	c := NewChat(80, 10)

	// Add initial messages
	for i := 0; i < 20; i++ {
		c.AddMessage(NewUserMessage("Message"))
	}
	c.renderContent()
	c.viewport.SetContent(c.content)

	// User is at bottom (default)
	assert.True(t, c.atBottom)
	assert.False(t, c.userScrolled)

	// Add new message
	c.AddMessage(NewUserMessage("New"))
	c, _ = c.Update(nil) // Trigger update

	// Should auto-scroll to bottom
	assert.True(t, c.atBottom || c.scrollPercent >= 99.0)
}
