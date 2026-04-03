package tools

// resolveWorkspaceRoot returns the workspace_root param if set, falling back to defaultRoot.
func resolveWorkspaceRoot(defaultRoot string, params ToolParameters) string {
	root, err := params.GetString("workspace_root")
	if err == nil && root != "" {
		return root
	}

	return defaultRoot
}
