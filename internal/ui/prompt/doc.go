// Package prompt provides pure state management and rendering for command-line prompt editing.
//
// This package implements a readline-like editing experience with:
//   - Buffer: Editable text buffer with cursor management
//   - History: Command history with navigation and draft preservation
//   - Model: Combined buffer + history state machine
//   - Renderer: ANSI-based single-line prompt rendering with Unicode support
//
// All components are designed for testing and composition with terminal layers.
//
// # Basic Usage
//
//	m := prompt.NewModel(1000) // 1000 history entries
//
// Insert text
//
//	m.Insert('h')
//	m.Insert('i')
//
// Edit
//
//	m.Backspace()
//	m.Delete()
//	m.MoveLeft()
//	m.MoveRight()
//
// Kill-line operations
//
//	m.ClearLineLeft()  // Ctrl-U
//	m.ClearLineRight() // Ctrl-K
//	m.DeleteWord()     // Ctrl-W
//
// History navigation
//
//	m.PrevHistory() // Up arrow
//	m.NextHistory() // Down arrow
//
// Submit
//
//	text := m.Submit() // Returns text, adds to history, clears buffer
//
// # Architecture
//
// The package follows a layered design:
//
//   - Buffer: Low-level rune buffer with cursor (buffer.go)
//   - History: Ring buffer with navigation state (history.go)
//   - Model: High-level API combining buffer + history (model.go)
//   - Renderer: Single-line ANSI rendering with Unicode width calculation (renderer.go)
//
// This separation allows testing each layer independently and provides
// flexibility for different UI implementations.
//
// # Rendering
//
// The Renderer provides single-line prompt rendering with:
//   - Accurate cursor positioning using rivo/uniseg for width calculation
//   - Optional right-aligned status text
//   - Horizontal scrolling for long lines
//   - Support for wide characters (emoji, CJK) and combining marks
//
// Example:
//
//	r := prompt.NewRenderer(os.Stdout, 80, "> ")
//	r.Redraw(model, "typing")  // Renders: "> hello█                typing"
//
// # Unicode Handling
//
// The buffer correctly handles:
//   - Emoji (multi-byte runes)
//   - CJK characters
//   - Combining marks (future: use rivo/uniseg for grapheme clusters)
//
// Cursor navigation works on rune boundaries. For production use with
// complex Unicode, integrate rivo/uniseg for proper grapheme cluster handling.
//
// # History Behavior
//
// History navigation preserves uncommitted drafts:
//
//  1. User types "draft text"
//  2. Press Up → sees previous command, draft saved
//  3. Press Up again → sees older command
//  4. Press Down → sees newer command
//  5. Press Down again → draft restored
//
// Submitting during navigation resets to normal (not navigating) state.
package prompt
