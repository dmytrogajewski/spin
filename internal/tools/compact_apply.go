package tools

import "github.com/dmytrogajewski/spin/internal/contexteng/compact"

type compactControl struct {
	enabled *bool
}

// SetCompactEnabled overrides default-on compact for this tool.
func (c *compactControl) SetCompactEnabled(enabled bool) {
	c.enabled = &enabled
}

func (c *compactControl) compactOn() bool {
	if c.enabled != nil {
		return *c.enabled
	}

	return true
}

func applyBuiltinCompact(on bool, cmd, stdout string) ToolResult {
	if !compact.ShouldApply(on, cmd) {
		return NewToolResult(stdout)
	}

	applied := compact.Default().Apply(cmd, []byte(stdout), nil, 0)

	return NewToolResult(string(applied.Stdout)).WithMetadata(withCompactLedger(nil, applied.Ledger))
}

// ApplyCompactSettings sets the shared escape hatch and read level on compact-aware tools.
func ApplyCompactSettings(registry *Registry, enabled bool, readLevel string) {
	if registry == nil {
		return
	}

	for _, tool := range registry.List() {
		if setter, ok := tool.(interface{ SetCompactEnabled(bool) }); ok {
			setter.SetCompactEnabled(enabled)
		}

		if setter, ok := tool.(interface{ SetReadLevel(string) }); ok && readLevel != "" {
			setter.SetReadLevel(readLevel)
		}
	}
}
