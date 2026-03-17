package prompt

// Section names for the Regular mode prompt.
const (
	SectionIdentity      = "identity"
	SectionToolPrinciple = "tool-principle"
	SectionToolGuidance  = "tool-guidance"
	SectionResponseStyle = "response-style"
	SectionProjectInstr  = "project-instructions"
)

// Priority values for standard sections.
// Project instructions come first (matching legacy builder ordering).
const (
	priorityProjectInstr  = 5
	priorityIdentity      = 10
	priorityToolPrinciple = 100
	priorityToolGuidance  = 110
	priorityResponseStyle = 200
)

// templateIdentity is the core identity section.
const templateIdentity = `You are an expert software engineer assistant ` +
	`with access to tools for reading, writing, and editing files, ` +
	`executing commands, and searching code.`

// templateToolPrinciple is the core tool-use principle.
const templateToolPrinciple = `# Core Principle: Always Use Tools

When you need to write or modify code, you MUST use the appropriate tools ` +
	`(write_file, apply_patch, etc.). NEVER output code blocks in chat ` +
	`as a substitute for actually writing the code.

Bad behavior (DO NOT DO THIS):
- Showing code in a markdown code block and saying "here's the code you can use"
- Outputting a full file and asking the user to copy it
- Describing what code should look like instead of writing it

Good behavior:
- Use write_file to create new files
- Use apply_patch or edit tools to modify existing files
- Use execute_command to run builds, tests, or other commands
- Only show code snippets in chat when explaining concepts or ` +
	`when the user explicitly asks to see code without writing it`

// templateToolGuidance is the tool usage guidelines section.
const templateToolGuidance = `# Tool Usage Guidelines

1. ALWAYS read files before modifying them to understand existing code structure
2. When asked to implement something, write the code using tools - don't just describe it
3. After writing code, verify it works by running appropriate commands (build, test, lint)
4. If a tool call fails, analyze the error and try again with corrections
5. Use file_search and list_directory to explore the codebase before making changes`

// templateResponseStyle is the response style section.
const templateResponseStyle = `# Response Style

- Be direct and concise
- Focus on actions, not explanations of what you "would" do
- When you complete a task, briefly summarize what was done
- If you encounter errors, fix them rather than just reporting them`

// DefaultRegularSections returns the standard sections for Regular mode.
// These sections compose to produce output equivalent to RegularSystemPrompt
// when joined with double-newline separators.
func DefaultRegularSections() []Section {
	return []Section{
		{
			Name:      SectionIdentity,
			Priority:  priorityIdentity,
			Cacheable: true,
			Template:  templateIdentity,
		},
		{
			Name:      SectionToolPrinciple,
			Priority:  priorityToolPrinciple,
			Cacheable: true,
			Template:  templateToolPrinciple,
		},
		{
			Name:      SectionToolGuidance,
			Priority:  priorityToolGuidance,
			Cacheable: true,
			Template:  templateToolGuidance,
		},
		{
			Name:      SectionResponseStyle,
			Priority:  priorityResponseStyle,
			Cacheable: true,
			Template:  templateResponseStyle,
		},
	}
}

// ProjectInstructionsSection creates a section for AGENTS.md content.
// The content is passed at construction time; the section is always included.
func ProjectInstructionsSection(content string) Section {
	return Section{
		Name:      SectionProjectInstr,
		Priority:  priorityProjectInstr,
		Cacheable: false,
		Template:  "# Project Instructions\n\n" + content + "\n\n---",
	}
}
