// Package ui provides UI components for the Spin TUI.
//
// This package contains reusable UI components built on Bubble Tea:
//   - Chat: conversation transcript display with markdown and code highlighting
//   - Input: multi-line input widget with history
//   - Message: message types and rendering logic
//   - Formatter: content formatting (markdown, code blocks, ANSI)
//   - FilePicker: fuzzy file search and selection (@-trigger)
//   - ApprovalModal: command approval dialog with approve/deny/modify actions
//
// Components support:
//   - Streaming AI responses
//   - Syntax-highlighted code blocks (chroma)
//   - Formatted markdown (glamour)
//   - ANSI color preservation
//   - Reasoning blocks
//   - Efficient viewport scrolling
//   - Interactive command approval with editing
package ui
