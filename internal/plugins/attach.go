package plugins

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/mcp"
)

// AttachMCP maps each plugin's mcp.json into the existing MCP manager.
// A failing server does not unload that plugin's skills or stop sibling servers.
func AttachMCP(ctx context.Context, svc *mcp.Service, loaded []Plugin) []string {
	if svc == nil {
		return nil
	}

	warnings := make([]string, 0)

	for _, plugin := range loaded {
		if !plugin.MCPValid {
			continue
		}

		for _, cfg := range ServerConfigs(plugin.Root, plugin.MCP) {
			if err := svc.ConnectServer(ctx, cfg); err != nil {
				warnings = append(warnings, fmt.Sprintf(
					"%s%s/%s: %v", warnSkippedMCP, plugin.Manifest.Name, cfg.Name, err,
				))
			}
		}
	}

	return warnings
}
