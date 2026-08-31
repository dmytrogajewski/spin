package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/dmytrogajewski/spin/internal/nav"
)

const (
	navigateToolName  = "navigate"
	navigateKindParam = "kind"
	navigateIDParam   = "id"
	navigatePathParam = "path"
	navigateNameParam = "name"
)

// NavigateTool returns structured index records instead of raw trees or file bodies.
type NavigateTool struct {
	index *nav.Index
}

// NewNavigateTool creates the model-facing navigate tool.
func NewNavigateTool(index *nav.Index) *NavigateTool {
	return &NavigateTool{index: index}
}

// Name implements Tool.
func (t *NavigateTool) Name() string {
	return navigateToolName
}

// Description implements Tool.
func (t *NavigateTool) Description() string {
	return "Structured navigation index. Returns records (kind, id, title, why, open) " +
		"for skill, plugin, session, peer, path, or symbol. open is a path or card URL, never a file body. " +
		"Path listings are R10 tree-compressed."
}

// Schema implements Tool.
func (t *NavigateTool) Schema() ToolSchema {
	return ToolSchema{Type: "function", Function: FunctionSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters: ParameterSchema{
			Type: "object",
			Properties: map[string]PropertyDefinition{
				navigateKindParam: {
					Type:        "string",
					Description: "Index kind: " + nav.ValidKinds,
					Enum:        strings.Split(nav.ValidKinds, "|"),
				},
				navigateIDParam: {
					Type:        "string",
					Description: "Optional record id filter",
				},
				navigatePathParam: {
					Type:        "string",
					Description: "Directory to list when kind=path",
				},
				navigateNameParam: {
					Type:        "string",
					Description: "Symbol name when kind=symbol",
				},
			},
			Required: []string{navigateKindParam},
		},
	}}
}

// Execute implements Tool.
func (t *NavigateTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return NewToolError(err), nil
	}

	index := t.index
	if index == nil {
		index = nav.New(nav.Sources{})
	}

	result, err := index.Lookup(nav.Query{
		Kind: nav.Kind(strings.TrimSpace(params.GetStringOr(navigateKindParam, ""))),
		ID:   strings.TrimSpace(params.GetStringOr(navigateIDParam, "")),
		Path: strings.TrimSpace(params.GetStringOr(navigatePathParam, "")),
		Name: strings.TrimSpace(params.GetStringOr(navigateNameParam, "")),
	})
	if err != nil {
		return NewToolError(err), nil
	}

	payload, encErr := json.Marshal(result)
	if encErr != nil {
		return NewToolError(encErr), nil
	}

	return NewToolResult(string(payload)), nil
}

// RegisterNavigateTool registers navigate on the registry.
func RegisterNavigateTool(registry *Registry, index *nav.Index) {
	if registry == nil {
		return
	}

	_ = registry.RegisterOrReplace(NewNavigateTool(index))
}
