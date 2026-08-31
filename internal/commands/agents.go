package commands

// Journey: specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md.

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/nav"
)

// AgentsCommand lists builtin agent specs and A2A peers from nav.
type AgentsCommand struct{}

// Name implements Command.
func (c *AgentsCommand) Name() string {
	return "/agents"
}

// Description implements Command.
func (c *AgentsCommand) Description() string {
	return "List builtin agent specs and A2A peers"
}

// Execute lists nav peer records (builtins when no remote peers).
func (c *AgentsCommand) Execute(_ context.Context, _ []string, _ CommandContext) (string, error) {
	records, err := nav.New(nav.Sources{}).Records(nav.KindPeer)
	if err != nil {
		return "", err
	}

	return formatAgents(records), nil
}

func formatAgents(records []nav.Record) string {
	if len(records) == 0 {
		return "No agents."
	}

	lines := make([]string, 0, len(records))
	for _, rec := range records {
		lines = append(lines, fmt.Sprintf("%s  %s  %s", rec.ID, rec.Title, rec.Why))
	}

	return strings.Join(lines, "\n")
}
