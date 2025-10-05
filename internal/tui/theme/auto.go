package theme

// autoTheme detects terminal background and delegates to dark or light theme.
type autoTheme struct {
	delegate Theme
}

// newAutoTheme creates a new auto theme that detects terminal background.
func newAutoTheme() *autoTheme {
	// Detect terminal background color using heuristics
	isDark := detectDarkTerminal()

	var delegate Theme
	if isDark {
		delegate = newDarkTheme()
	} else {
		delegate = newLightTheme()
	}

	return &autoTheme{delegate: delegate}
}

// Delegate all methods to the underlying theme
func (t *autoTheme) Name() string                         { return "auto" }
func (t *autoTheme) Colors() ColorScheme                  { return t.delegate.Colors() }
func (t *autoTheme) ChatStyles() ChatStyleSet             { return t.delegate.ChatStyles() }
func (t *autoTheme) StatusBarStyles() StatusBarStyleSet   { return t.delegate.StatusBarStyles() }
func (t *autoTheme) ApprovalStyles() ApprovalStyleSet     { return t.delegate.ApprovalStyles() }
func (t *autoTheme) HelpStyles() HelpStyleSet             { return t.delegate.HelpStyles() }
func (t *autoTheme) FilePickerStyles() FilePickerStyleSet { return t.delegate.FilePickerStyles() }
func (t *autoTheme) InputStyles() InputStyleSet           { return t.delegate.InputStyles() }
func (t *autoTheme) SupportsColors() bool                 { return t.delegate.SupportsColors() }
