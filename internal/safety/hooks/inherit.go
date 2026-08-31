package hooks

// CopyScripts returns a new slice of extra hook scripts (plugin, skill, frontmatter).
// A missing copy is a test failure at the call site — this helper does not log.
func CopyScripts(src []PluginScript) []PluginScript {
	return append([]PluginScript{}, src...)
}

// PluginScripts returns a copy of extra scripts registered on this runner.
func (r *Runner) PluginScripts() []PluginScript {
	if r == nil {
		return nil
	}

	return CopyScripts(r.pluginScripts)
}
