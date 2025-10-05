// Package tui provides an interactive terminal user interface for Spin
// using the Bubble Tea framework.
//
// The TUI implements The Elm Architecture pattern with a Model-Update-View cycle:
//   - Model: Application state
//   - Update: Message processing and state transitions
//   - View: Rendering to terminal
//
// The TUI supports multiple states (idle, waiting for response, tool approval, etc.)
// and handles keyboard input, terminal resizing, and integration with the core agent.
package tui
