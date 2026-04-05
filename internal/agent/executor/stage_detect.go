package executor

import (
	"context"
	"strings"
)

// NewDetectStage creates a stage that detects server/watcher commands.
// Matching commands set IsServer=true on the PipelineContext.
func NewDetectStage() Stage {
	return func(_ context.Context, pc *PipelineContext) error {
		cmd := pc.Command
		fullCmd := buildFullCommand(cmd.Program, cmd.Args)

		for _, pattern := range ServerPatterns {
			if pattern.MatchString(fullCmd) {
				pc.IsServer = true

				return nil
			}
		}

		return nil
	}
}

// buildFullCommand reconstructs the command string from program and args.
func buildFullCommand(program string, args []string) string {
	if len(args) == 0 {
		return program
	}

	return program + " " + strings.Join(args, " ")
}
