//go:build windows

package sandbox

// NewSandbox creates a Windows sandbox (currently not implemented).
// Returns a NoopSandbox as Windows sandboxing support is planned for future releases.
func NewSandbox(mode Mode) (Sandbox, error) {
	return &NoopSandbox{}, nil
}
