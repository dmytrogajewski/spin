package tools

import "github.com/dmytrogajewski/spin/pkg/classify"

// ToolCategory represents a functional category for a tool.
// Both TUI block rendering and ACP approval use this shared classification
// to avoid maintaining parallel tool-name-to-category mappings.
type ToolCategory int

const (
	// CategoryUnknown is the default for unrecognized tool names.
	CategoryUnknown ToolCategory = iota
	// CategoryRead covers tools that read files or context.
	CategoryRead
	// CategoryEdit covers tools that write or patch files.
	CategoryEdit
	// CategoryExecute covers tools that run commands or processes.
	CategoryExecute
	// CategorySearch covers tools that search files or symbols.
	CategorySearch
	// CategoryMove covers tools that rename or move symbols.
	CategoryMove
	// CategoryFetch covers tools that access network/web resources.
	CategoryFetch
	// CategoryThink covers tools that aid reasoning (memory, scratchpad).
	CategoryThink
	// CategoryNotice covers tools that return informational context.
	CategoryNotice
)

// toolClassifier is the rule-based classifier for tool names.
var toolClassifier = buildToolClassifier()

func buildToolClassifier() *classify.Classifier[string, ToolCategory] {
	c := classify.New[string, ToolCategory](CategoryUnknown)

	isOneOf := func(names ...string) func(string) bool {
		set := make(map[string]bool, len(names))
		for _, n := range names {
			set[n] = true
		}

		return func(s string) bool { return set[s] }
	}

	c.AddRule("read_files", isOneOf("read_file", "list_directory", "get_process_output", "list_processes"), CategoryRead)
	c.AddRule("notice", isOneOf(gitContextName, getContextName), CategoryNotice)
	c.AddRule("edit_files", isOneOf("write_file", "edit_file", applyPatchName), CategoryEdit)
	c.AddRule("execute", isOneOf("execute_command", shellCommandName, "start_process", "git_operation", "kill_process"), CategoryExecute)
	c.AddRule("search", isOneOf(fileSearchName, "find_symbol", "find_references"), CategorySearch)
	c.AddRule("move", isOneOf("rename_symbol"), CategoryMove)
	c.AddRule("fetch", isOneOf("fetch_url", "web_search", "capture_web_screenshot", "open_browser"), CategoryFetch)
	c.AddRule("think", isOneOf("memory", "scratchpad"), CategoryThink)

	return c
}

// ClassifyTool maps a tool name to its functional category.
func ClassifyTool(toolName string) ToolCategory {
	return toolClassifier.Classify(toolName)
}
