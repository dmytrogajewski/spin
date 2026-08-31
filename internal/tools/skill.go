package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/skills"
)

const (
	skillToolName     = "skill"
	loadSkillToolName = "load_skill"
	skillParamName    = "name"
	skillParamPath    = "path"
)

// SkillTool loads one skill body into the current turn.
// Optional path reads one file under the skill root (contained; one hop).
type SkillTool struct {
	name    string
	catalog skills.Catalog
}

// NewSkillTool creates the model-facing skill tool.
func NewSkillTool(catalog skills.Catalog) *SkillTool {
	return makeSkillTool(skillToolName, catalog)
}

// NewLoadSkillTool creates the load_skill alias of SkillTool.
func NewLoadSkillTool(catalog skills.Catalog) *SkillTool {
	return makeSkillTool(loadSkillToolName, catalog)
}

func makeSkillTool(name string, catalog skills.Catalog) *SkillTool {
	return &SkillTool{name: name, catalog: catalog}
}

// Name implements Tool.
func (t *SkillTool) Name() string {
	return t.name
}

// Description implements Tool.
func (t *SkillTool) Description() string {
	return "Load a skill from the catalog by name. Returns the SKILL.md body and skill root. " +
		"Optional path reads one file under the skill root (scripts/, references/, assets/). " +
		"Do not use this to load every reference at once."
}

// Schema implements Tool.
func (t *SkillTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					skillParamName: {
						Type:        "string",
						Description: "Catalog skill name to activate",
					},
					skillParamPath: {
						Type:        "string",
						Description: "Optional relative path under the skill root to read (one file)",
					},
				},
				Required: []string{skillParamName},
			},
		},
	}
}

// Execute implements Tool.
func (t *SkillTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return NewToolError(err), nil
	}

	name := strings.TrimSpace(params.GetStringOr(skillParamName, ""))
	if name == "" {
		return NewToolError(skills.ErrEmptyName), nil
	}

	activation, err := skills.Activate(t.catalog, name)
	if err != nil {
		return NewToolError(err), nil
	}

	rel := strings.TrimSpace(params.GetStringOr(skillParamPath, ""))
	if rel != "" {
		return t.readSkillPath(activation, rel)
	}

	return formatActivationResult(activation), nil
}

func (t *SkillTool) readSkillPath(activation skills.Activation, rel string) (ToolResult, error) {
	data, err := skills.ReadResource(activation.Root, rel)
	if err != nil {
		return NewToolError(err), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", activation.Name)
	fmt.Fprintf(&b, "source: %s\n", activation.Source)
	fmt.Fprintf(&b, "root: %s\n", activation.Root)
	fmt.Fprintf(&b, "path: %s\n\n", rel)
	b.Write(data)

	return NewToolResult(b.String()).WithMetadata(activationMetadata(activation, rel)), nil
}

func formatActivationResult(activation skills.Activation) ToolResult {
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", activation.Name)
	fmt.Fprintf(&b, "source: %s\n", activation.Source)
	fmt.Fprintf(&b, "root: %s\n", activation.Root)

	if activation.AllowedTools != "" {
		fmt.Fprintf(&b, "allowed-tools: %s\n", activation.AllowedTools)
	}

	b.WriteString("\n")
	b.WriteString(activation.Body)

	return NewToolResult(b.String()).WithMetadata(activationMetadata(activation, ""))
}

func activationMetadata(activation skills.Activation, rel string) map[string]any {
	meta := map[string]any{
		"name":   activation.Name,
		"source": string(activation.Source),
		"root":   activation.Root,
	}
	if activation.AllowedTools != "" {
		meta["allowed-tools"] = activation.AllowedTools
	}

	if rel != "" {
		meta["path"] = rel
	}

	return meta
}

// RegisterSkillTools registers skill and load_skill on the registry.
func RegisterSkillTools(registry *Registry, catalog skills.Catalog) {
	if registry == nil {
		return
	}

	_ = registry.RegisterOrReplace(NewSkillTool(catalog))
	_ = registry.RegisterOrReplace(NewLoadSkillTool(catalog))
}
