package agentsmd

// Config configures AGENTS.md discovery and loading.
type Config struct {
	// Enabled controls whether AGENTS.md is loaded.
	// Default: true
	Enabled bool

	// Path specifies a custom path to AGENTS.md.
	// If empty, auto-discovery is used.
	Path string

	// MaxSize is the maximum file size in bytes.
	// Files larger than this are truncated with a warning.
	// Default: 100KB (100 * 1024)
	MaxSize int64
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled: true,
		MaxSize: 100 * 1024, // 100KB
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.MaxSize < 0 {
		c.MaxSize = 0
	}
	return nil
}
