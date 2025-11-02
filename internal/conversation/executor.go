package conversation

import (
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/security"
)

// buildExecutor creates an agent.Executor with appropriate options based on configuration.
func (b *Builder) buildExecutor(workDir string) (*agent.Executor, error) {
	validator := security.NewValidator()
	opts := []agent.ExecutorOption{
		agent.WithValidator(validator),
	}

	if b.approvalHandler != nil {
		opts = append(opts,
			agent.WithApprovalService(security.NewApprovalService(b.approvalHandler, b.emitter, validator)),
		)
	}

	if cfg := b.cfg; cfg != nil {
		if cfg.Timeout > 0 {
			opts = append(opts, agent.WithTimeout(cfg.Timeout))
		}
		if cfg.CacheCommands {
			cache := agent.NewCommandCache(5*time.Minute, 10*1024*1024) // 5m TTL, 10MB cap
			opts = append(opts, agent.WithCache(cache))
		}
	}

	exec, err := agent.NewExecutor(workDir, opts...)
	if err != nil {
		return nil, fmt.Errorf("executor(%s): %w", workDir, err)
	}
	return exec, nil
}
