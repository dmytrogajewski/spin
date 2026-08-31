package commands

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/plugins"
	"github.com/dmytrogajewski/spin/internal/skills"
)

// SkillsCommand lists the discovered skill catalog.
type SkillsCommand struct {
	options func(workDir string) skills.Options
}

// Name implements Command.
func (c *SkillsCommand) Name() string {
	return "/skills"
}

// Description implements Command.
func (c *SkillsCommand) Description() string {
	return "List discovered agent skills (name, source, description)"
}

// Execute prints one line per skill using the same Discover as Composer.
func (c *SkillsCommand) Execute(_ context.Context, _ []string, cmdCtx CommandContext) (string, error) {
	if c.options != nil {
		return skills.Format(skills.Discover(c.options(cmdCtx.GetWorkDir()))), nil
	}

	return skills.Format(plugins.DiscoverCatalog(cmdCtx.GetWorkDir(), nil)), nil
}
