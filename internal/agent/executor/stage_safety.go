package executor

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/safety"
)

// ErrCommandForbidden is returned when a command is classified as forbidden.
var ErrCommandForbidden = errors.New("command forbidden")

// NewSafetyStage creates a stage that validates commands against security policy.
// Forbidden commands halt the pipeline. Nil validator skips validation.
func NewSafetyStage(validator *safety.Validator) Stage {
	return func(_ context.Context, pc *PipelineContext) error {
		if validator == nil {
			return nil
		}

		result, err := validator.Classify(pc.Command)
		if err != nil {
			return fmt.Errorf("safety classification failed: %w", err)
		}

		if result.Classification == safety.CommandForbidden {
			pc.Halt(fmt.Errorf("%w: %s", ErrCommandForbidden, result.Reason))
		}

		return nil
	}
}
