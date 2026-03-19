package executor

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/safety/blocklist"
)

// ErrCommandBlocklisted is returned when a command matches a blocklist pattern.
var ErrCommandBlocklisted = errors.New("command blocklisted")

// NewBlocklistStage creates a stage that checks commands against the blocklist.
// Blocked commands halt the pipeline. A nil checker skips the check.
func NewBlocklistStage(checker *blocklist.Checker) Stage {
	return func(_ context.Context, pc *PipelineContext) error {
		if checker == nil || !checker.Enabled() {
			return nil
		}

		blocked, reason := checker.Check(pc.Command.Raw)
		if blocked {
			pc.Halt(fmt.Errorf("%w: %s", ErrCommandBlocklisted, reason))
		}

		return nil
	}
}
