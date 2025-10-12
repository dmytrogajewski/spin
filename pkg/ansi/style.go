package ansi

import "strings"

// Style represents a text styling builder that provides a fluent API
// for applying ANSI colors and text styles.
//
// Styles can be chained together and are applied in the order they are called.
// The final string includes an automatic reset code at the end.
//
// Example:
//
//	styled := ansi.New("Error").Red().Bold().String()
//	// Output: "\x1b[31m\x1b[1mError\x1b[0m"
type Style struct {
	text  string
	codes []string
}

// New creates a new Style builder for the given text.
//
// Example:
//
//	s := ansi.New("Important message")
//	styled := s.Red().Bold().String()
func New(text string) *Style {
	return &Style{
		text:  text,
		codes: make([]string, 0, 4), // Pre-allocate for common case (1-2 styles)
	}
}

// Red applies red foreground color.
func (s *Style) Red() *Style {
	s.codes = append(s.codes, Red)
	return s
}

// Green applies green foreground color.
func (s *Style) Green() *Style {
	s.codes = append(s.codes, Green)
	return s
}

// Yellow applies yellow foreground color.
func (s *Style) Yellow() *Style {
	s.codes = append(s.codes, Yellow)
	return s
}

// Blue applies blue foreground color.
func (s *Style) Blue() *Style {
	s.codes = append(s.codes, Blue)
	return s
}

// Magenta applies magenta foreground color.
func (s *Style) Magenta() *Style {
	s.codes = append(s.codes, Magenta)
	return s
}

// Cyan applies cyan foreground color.
func (s *Style) Cyan() *Style {
	s.codes = append(s.codes, Cyan)
	return s
}

// White applies white foreground color.
func (s *Style) White() *Style {
	s.codes = append(s.codes, White)
	return s
}

// Black applies black foreground color.
func (s *Style) Black() *Style {
	s.codes = append(s.codes, Black)
	return s
}

// Bold applies bold/bright styling.
func (s *Style) Bold() *Style {
	s.codes = append(s.codes, Bold)
	return s
}

// Dim applies dim/faint styling.
func (s *Style) Dim() *Style {
	s.codes = append(s.codes, Dim)
	return s
}

// Italic applies italic styling.
// Note: Not all terminals support italic text.
func (s *Style) Italic() *Style {
	s.codes = append(s.codes, Italic)
	return s
}

// Underline applies underline styling.
func (s *Style) Underline() *Style {
	s.codes = append(s.codes, Underline)
	return s
}

// String returns the styled text with all applied ANSI codes and a reset code at the end.
//
// If no styles have been applied, the original text is returned unchanged.
// For empty text with styles, the codes and reset are still applied.
//
// Example:
//
//	ansi.New("text").Red().Bold().String()
//	// Returns: "\x1b[31m\x1b[1mtext\x1b[0m"
//
//	ansi.New("text").String()
//	// Returns: "text"
func (s *Style) String() string {
	// No styles applied, return text as-is
	if len(s.codes) == 0 {
		return s.text
	}

	// Estimate final size to minimize allocations
	// Each ANSI code is ~4-5 bytes, plus text, plus reset (4 bytes)
	estimatedSize := len(s.text) + len(s.codes)*5 + 4
	var b strings.Builder
	b.Grow(estimatedSize)

	// Apply all style codes
	for _, code := range s.codes {
		b.WriteString(code)
	}

	// Add text
	b.WriteString(s.text)

	// Add reset
	b.WriteString(Reset)

	return b.String()
}
