package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MaxTranscriptMessages is the maximum number of messages to keep in memory.
const MaxTranscriptMessages = 1000

// Chat represents the chat interface component.
type Chat struct {
	viewport  viewport.Model
	messages  []Message
	width     int
	height    int
	formatter *Formatter

	// Rendering state
	content      string // Cached rendered content
	contentDirty bool   // Needs re-render
	atBottom     bool   // Auto-scroll to bottom

	// Scroll tracking
	scrollPercent float64 // Current scroll position (0.0-100.0)
	userScrolled  bool    // User has manually scrolled

	// Thinking/reasoning display
	thinkingParser *ThinkingParser // Parser for <think>...</think> tags
	showThinking   bool            // Whether to show full thinking or collapse
}

// NewChat creates a new chat component.
func NewChat(width, height int) Chat {
	vp := viewport.New(width, height)
	vp.SetContent("")

	formatter, _ := NewFormatter(width)

	return Chat{
		viewport:       vp,
		messages:       make([]Message, 0),
		width:          width,
		height:         height,
		formatter:      formatter,
		content:        "",
		contentDirty:   false,
		atBottom:       true,
		scrollPercent:  100.0,
		userScrolled:   false,
		thinkingParser: NewThinkingParser(),
		showThinking:   false, // Default: collapsed (show "Thinking...")
	}
}

// SetSize updates the chat dimensions.
func (c *Chat) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.viewport.Width = width
	c.viewport.Height = height

	// Update formatter width
	if c.formatter != nil {
		_ = c.formatter.SetWidth(width) // Ignore error (non-critical)
	}

	c.contentDirty = true
}

// AddMessage adds a new message to the chat.
func (c *Chat) AddMessage(msg Message) {
	c.messages = append(c.messages, msg)

	// Trim old messages if limit exceeded
	if len(c.messages) > MaxTranscriptMessages {
		c.messages = c.messages[len(c.messages)-MaxTranscriptMessages:]
	}

	c.contentDirty = true
}

// StreamDelta appends content to the last message (for streaming responses).
func (c *Chat) StreamDelta(delta string) {
	if len(c.messages) == 0 {
		return
	}

	lastIdx := len(c.messages) - 1
	c.messages[lastIdx].Content += delta
	c.messages[lastIdx].Streaming = true
	c.contentDirty = true
}

// FinishStreaming marks the last message as complete.
func (c *Chat) FinishStreaming() {
	if len(c.messages) == 0 {
		return
	}

	lastIdx := len(c.messages) - 1
	c.messages[lastIdx].Streaming = false
	c.contentDirty = true
}

// GetMessages returns all messages.
func (c Chat) GetMessages() []Message {
	return c.messages
}

// Clear removes all messages.
func (c *Chat) Clear() {
	c.messages = make([]Message, 0)
	c.contentDirty = true
}

// Update handles Bubble Tea messages.
func (c Chat) Update(msg tea.Msg) (Chat, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil

	case tea.KeyMsg:
		// Handle scroll navigation keys
		switch msg.String() {
		case "pgup":
			c.PageUp()
			return c, nil
		case "pgdown":
			c.PageDown()
			return c, nil
		case "home":
			c.GotoTop()
			return c, nil
		case "end":
			c.GotoBottom()
			return c, nil
		}
	}

	// Re-render if needed
	if c.contentDirty {
		c.renderContent()
		c.viewport.SetContent(c.content)
		c.contentDirty = false

		// Auto-scroll to bottom (only if user hasn't manually scrolled)
		if c.atBottom && !c.userScrolled {
			c.viewport.GotoBottom()
		}
	}

	// Update viewport
	c.viewport, cmd = c.viewport.Update(msg)
	c.updateScrollState()

	return c, cmd
}

// View renders the chat with scroll indicator.
func (c Chat) View() string {
	view := c.viewport.View()

	// Add scroll position indicator if not at 100%
	if c.scrollPercent < 99.9 && c.viewport.TotalLineCount() > c.viewport.Height {
		indicator := fmt.Sprintf(" %d%% ", int(c.scrollPercent))
		indicatorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Background(lipgloss.Color("0")).
			Padding(0, 1)

		// Position indicator in bottom-right corner
		lines := strings.Split(view, "\n")
		if len(lines) > 0 {
			lastLine := lines[len(lines)-1]
			availableWidth := c.width - lipgloss.Width(lastLine) - lipgloss.Width(indicator)

			if availableWidth > 0 {
				padding := strings.Repeat(" ", availableWidth)
				lines[len(lines)-1] = lastLine + padding + indicatorStyle.Render(indicator)
				view = strings.Join(lines, "\n")
			}
		}
	}

	return view
}

// renderContent renders all messages to a string.
func (c *Chat) renderContent() {
	parts := make([]string, 0, len(c.messages))

	for i, msg := range c.messages {
		rendered := c.renderMessage(msg)
		parts = append(parts, rendered)
		// Only add spacing between messages, not after the last one
		if i < len(c.messages)-1 {
			// Add minimal spacing - single empty line only between different speakers
			if i+1 < len(c.messages) && c.messages[i].Role != c.messages[i+1].Role {
				parts = append(parts, "") // Blank line between different roles
			}
		}
	}

	c.content = strings.Join(parts, "\n")
}

// renderMessage renders a single message.
func (c *Chat) renderMessage(msg Message) string {
	var parts []string

	// Header
	header := c.renderMessageHeader(msg)
	parts = append(parts, header)

	// Thinking (if present or currently parsing)
	if msg.Thinking != "" || (msg.Streaming && c.thinkingParser.IsInThinking()) {
		if c.showThinking {
			// Show full thinking content (streaming in real-time)
			thinking := c.renderThinking(msg.Thinking, msg.Streaming)
			parts = append(parts, thinking)
		} else {
			// Show collapsed thinking indicator
			thinkingIndicator := c.renderThinkingCollapsed(msg.Streaming)
			parts = append(parts, thinkingIndicator)
		}
	}

	// Content
	content := c.renderMessageContent(msg)
	parts = append(parts, content)

	// Tool call/result
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
		parts = append(parts, "▊")
	}

	rendered := strings.Join(parts, "\n")

	// Apply highlight border if message is selected (Phase 3.8)
	if msg.Highlighted {
		rendered = c.renderHighlightBorder(rendered)
	}

	return rendered
}

// renderHighlightBorder adds a visual border around highlighted message.
// Phase 3.8: Backtrack Mode
func (c *Chat) renderHighlightBorder(content string) string {
	highlightStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")). // Blue highlight
		Padding(0, 1)

	return highlightStyle.Render(content)
}

// renderMessageHeader renders the message header (role and timestamp).
func (c *Chat) renderMessageHeader(msg Message) string {
	var roleText string
	var style lipgloss.Style

	switch msg.Role {
	case RoleUser:
		roleText = "You"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	case RoleAssistant:
		roleText = "Assistant"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	case RoleSystem:
		roleText = "System"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	case RoleTool:
		roleText = "Tool"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	default:
		roleText = string(msg.Role)
		style = lipgloss.NewStyle().Bold(true)
	}

	// Format: "You │ 14:32:01"
	timestamp := msg.Timestamp.Format("15:04:05")
	return style.Render(roleText) + " │ " + timestamp
}

// renderMessageContent renders message content with formatting.
func (c *Chat) renderMessageContent(msg Message) string {
	if c.formatter == nil {
		return msg.Content
	}

	// Use formatter for assistant messages (markdown + code)
	if msg.Role == RoleAssistant {
		rendered, err := c.formatter.RenderContent(msg.Content)
		if err != nil {
			return msg.Content
		}
		return rendered
	}

	// Plain text for other roles
	return msg.Content
}

// renderThinking renders the full thinking content (expanded).
func (c *Chat) renderThinking(thinking string, streaming bool) string {
	// Calculate max width (chat width - border - padding)
	maxWidth := c.width - 4 // 2 for border, 2 for padding

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")). // Purple
		Padding(0, 1).
		Width(maxWidth)

	content := "💭 " + thinking
	if streaming {
		content += " ▊" // Show cursor while streaming
	}

	return style.Render(content)
}

// renderThinkingCollapsed renders a collapsed thinking indicator.
func (c *Chat) renderThinkingCollapsed(streaming bool) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")). // Gray
		Padding(0, 1).
		Faint(true)

	text := "💭 Thinking..."
	if streaming {
		text += " ▊" // Show cursor while streaming
	} else {
		text += " (Press Ctrl+T to expand)"
	}

	return style.Render(text)
}

// renderToolCall renders a tool call in Claude-like style.
func (c *Chat) renderToolCall(tc *ToolCall) string {
	// Container style with subtle border
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")). // Gray border
		Padding(0, 1).
		MarginLeft(2).
		Width(c.width - 6) // Account for margins and borders

	// Header style (tool name)
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("33")) // Blue

	// Build the tool call display
	var parts []string

	// Tool name header with icon
	toolIcon := c.getToolIcon(tc.Name)
	header := fmt.Sprintf("%s %s", toolIcon, tc.Name)
	parts = append(parts, headerStyle.Render(header))

	// Format arguments in a structured way
	if len(tc.Arguments) > 0 {
		argStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")). // Light gray
			MarginLeft(2)

		for key, val := range tc.Arguments {
			// Format the value based on its type
			formattedVal := c.formatArgumentValue(val)
			argLine := fmt.Sprintf("%s: %s", key, formattedVal)
			parts = append(parts, argStyle.Render(argLine))
		}
	}

	return containerStyle.Render(strings.Join(parts, "\n"))
}

// renderToolResult renders a tool result in Claude-like style.
func (c *Chat) renderToolResult(tr *ToolResult) string {
	// Determine if this is an error or success
	isError := tr.Error != ""

	// Container style
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")). // Gray border
		Padding(0, 1).
		MarginLeft(2).
		Width(c.width - 6) // Account for margins and borders

	// Header style based on success/error
	var headerStyle lipgloss.Style
	var headerText string
	if isError {
		headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")) // Red for errors
		headerText = "❌ Tool Error"
	} else {
		headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("70")) // Green for success
		headerText = "✓ Tool Result"
	}

	// Content style
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")). // Light gray
		MarginLeft(2)

	// Build the display
	var parts []string
	parts = append(parts, headerStyle.Render(headerText))

	// Add the content (error or output)
	content := tr.Output
	if isError {
		content = tr.Error
	}

	// Truncate very long output
	if len(content) > 500 {
		content = content[:497] + "..."
	}

	// Format the content
	if content != "" {
		// Split into lines and indent each
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if line != "" {
				parts = append(parts, contentStyle.Render(line))
			}
		}
	}

	return containerStyle.Render(strings.Join(parts, "\n"))
}

// updateScrollState checks if viewport is at bottom and calculates scroll position.
func (c *Chat) updateScrollState() {
	c.atBottom = c.viewport.AtBottom()

	// Calculate scroll percentage
	if c.viewport.TotalLineCount() > 0 {
		// YOffset is the top line visible, TotalLineCount is total lines
		// We want percentage = (current_position / total_scrollable_area) * 100
		totalScrollable := c.viewport.TotalLineCount() - c.viewport.Height
		if totalScrollable > 0 {
			c.scrollPercent = (float64(c.viewport.YOffset) / float64(totalScrollable)) * 100.0
		} else {
			c.scrollPercent = 100.0 // No scrolling needed
		}
	} else {
		c.scrollPercent = 0.0
	}

	// Clamp to 0-100
	if c.scrollPercent < 0 {
		c.scrollPercent = 0
	}
	if c.scrollPercent > 100 {
		c.scrollPercent = 100
	}
}

// PageUp scrolls up by one page.
func (c *Chat) PageUp() {
	c.viewport.PageUp()
	c.userScrolled = true
	c.updateScrollState()
}

// PageDown scrolls down by one page.
func (c *Chat) PageDown() {
	c.viewport.PageDown()
	c.userScrolled = true
	c.updateScrollState()
}

// GotoTop scrolls to the top of the transcript.
func (c *Chat) GotoTop() {
	c.viewport.GotoTop()
	c.userScrolled = true
	c.updateScrollState()
}

// GotoBottom scrolls to the bottom of the transcript.
func (c *Chat) GotoBottom() {
	c.viewport.GotoBottom()
	c.userScrolled = false // Re-enable auto-scroll
	c.updateScrollState()
}

// ScrollPercent returns the current scroll position (0-100).
func (c Chat) ScrollPercent() float64 {
	return c.scrollPercent
}

// ScrollToBottom is an alias for GotoBottom (for convenience).
// Phase 3.9 implementation.
func (c *Chat) ScrollToBottom() {
	c.GotoBottom()
}

// ResetUserScroll resets the user scrolled flag (e.g., when user sends message).
func (c *Chat) ResetUserScroll() {
	c.userScrolled = false
}

// GetUserMessageIndices returns the indices of all user messages in the chat.
// Used for backtrack navigation (Phase 3.8).
func (c Chat) GetUserMessageIndices() []int {
	indices := make([]int, 0)
	for i, msg := range c.messages {
		if msg.Role == RoleUser {
			indices = append(indices, i)
		}
	}
	return indices
}

// SetHighlight highlights a specific message (for backtrack mode).
// Phase 3.8: Backtrack Mode
func (c *Chat) SetHighlight(idx int) {
	// Clear all highlights first
	for i := range c.messages {
		c.messages[i].Highlighted = false
	}

	// Set highlight on specified message
	if idx >= 0 && idx < len(c.messages) {
		c.messages[idx].Highlighted = true
		c.contentDirty = true
	}
}

// ClearHighlight removes all highlights from messages.
// Phase 3.8: Backtrack Mode
func (c *Chat) ClearHighlight() {
	for i := range c.messages {
		c.messages[i].Highlighted = false
	}
	c.contentDirty = true
}

// TruncateAfter removes all messages after the specified index.
// Phase 3.8: Conversation forking in backtrack mode
func (c *Chat) TruncateAfter(idx int) {
	if idx >= 0 && idx < len(c.messages) {
		c.messages = c.messages[:idx+1]
		c.contentDirty = true
	}
}

// CurrentMessage returns the content of the last message (for tests/debugging).
func (c Chat) CurrentMessage() string {
	if len(c.messages) == 0 {
		return ""
	}
	return c.messages[len(c.messages)-1].Content
}

// AppendDelta appends content to the last message (streaming).
// If there's no message or last message is not streaming, creates a new assistant message.
// Automatically parses <think>...</think> tags and separates them from regular content.
func (c *Chat) AppendDelta(delta string) {
	// Parse thinking tags
	regularContent, thinkingContent := c.thinkingParser.Parse(delta)

	if len(c.messages) == 0 || !c.messages[len(c.messages)-1].Streaming {
		// Start new streaming message
		c.AddMessage(Message{
			Role:      RoleAssistant,
			Content:   regularContent,
			Thinking:  thinkingContent,
			Streaming: true,
			Timestamp: time.Now(),
		})
		return
	}

	// Append to existing streaming message
	lastIdx := len(c.messages) - 1
	if regularContent != "" {
		c.messages[lastIdx].Content += regularContent
	}
	if thinkingContent != "" {
		c.messages[lastIdx].Thinking += thinkingContent
	}
	c.messages[lastIdx].Streaming = true
	c.contentDirty = true
}

// FinalizeMessage marks the last message as complete (end streaming).
func (c *Chat) FinalizeMessage() {
	c.FinishStreaming()
	c.thinkingParser.Reset() // Reset parser for next message
}

// AllMessages returns all messages (for tests).
func (c Chat) AllMessages() []Message {
	return c.messages
}

// AddUserMessage adds a user message to the chat.
func (c *Chat) AddUserMessage(content string) {
	c.AddMessage(Message{
		Role:    "user",
		Content: content,
	})
}

// AddError adds an error message to the chat (Phase 3.12).
// Errors are displayed inline in the transcript with formatting.
func (c *Chat) AddError(err ErrorDisplay) {
	msg := Message{
		Role:      RoleSystem,
		Content:   c.formatErrorMessage(err),
		IsError:   true,
		Timestamp: time.Now(),
	}
	c.AddMessage(msg)
	c.ScrollToBottom() // Auto-scroll to show error
}

// formatErrorMessage formats an error for display in the chat transcript.
// Returns formatted error with icon, operation, code, and details.
func (c *Chat) formatErrorMessage(err ErrorDisplay) string {
	var parts []string

	// Icon + Message header
	icon := c.getErrorIcon(err.Severity)
	parts = append(parts, fmt.Sprintf("%s Error: %s", icon, err.Message))

	// Operation (if present)
	if err.Operation != "" {
		parts = append(parts, fmt.Sprintf("├─ Operation: %s", err.Operation))
	}

	// Error code
	parts = append(parts, fmt.Sprintf("├─ Code: %s", err.Code))

	// Timestamp (if present)
	if err.Timestamp != "" {
		parts = append(parts, fmt.Sprintf("├─ Time: %s", err.Timestamp))
	}

	// Details (if present)
	if err.Details != "" {
		parts = append(parts, fmt.Sprintf("└─ Details: %s", err.Details))
	} else {
		// Replace last ├─ with └─ if no details
		if len(parts) > 1 {
			lastIdx := len(parts) - 1
			parts[lastIdx] = strings.Replace(parts[lastIdx], "├─", "└─", 1)
		}
	}

	return strings.Join(parts, "\n")
}

// getErrorIcon returns the emoji icon for error severity.
func (c *Chat) getErrorIcon(severity int) string {
	switch severity {
	case 0:
		return "ℹ️" // Info
	case 1:
		return "⚠️" // Warning
	case 2:
		return "❌" // Error
	case 3:
		return "🔥" // Critical
	default:
		return "❓"
	}
}

// getToolIcon returns an appropriate icon for the tool.
func (c *Chat) getToolIcon(toolName string) string {
	switch toolName {
	case "execute_command", "shell", "bash":
		return "💻"
	case "read_file", "read":
		return "📖"
	case "write_file", "write":
		return "✍️"
	case "list_directory", "ls":
		return "📁"
	case "get_context", "context":
		return "🔍"
	case "search", "grep":
		return "🔎"
	case "git":
		return "🔀"
	case "docker":
		return "🐳"
	case "python", "py":
		return "🐍"
	case "node", "npm", "yarn":
		return "📦"
	default:
		return "🔧"
	}
}

// formatArgumentValue formats a tool argument value for display.
func (c *Chat) formatArgumentValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		// Truncate long strings
		if len(v) > 100 {
			return fmt.Sprintf("%q...", v[:97])
		}
		return fmt.Sprintf("%q", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		// For numbers, arrays, objects, etc.
		str := fmt.Sprintf("%v", v)
		if len(str) > 100 {
			return str[:97] + "..."
		}
		return str
	}
}

// ToggleThinking toggles the thinking display mode (collapsed/expanded).
func (c *Chat) ToggleThinking() {
	c.showThinking = !c.showThinking
	c.contentDirty = true // Force re-render
}

// IsShowingThinking returns true if thinking is expanded.
func (c Chat) IsShowingThinking() bool {
	return c.showThinking
}
