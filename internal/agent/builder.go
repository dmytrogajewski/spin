package agent

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/security"
)

// Builder constructs Agent instances with all dependencies.
type Builder struct {
	config          *Config
	provider        llm.Provider
	workingDir      string
	emitter         *events.EventEmitter
	approvalHandler security.ApprovalHandler
}

// NewBuilder creates a new agent builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// WithConfig sets the agent configuration.
func (b *Builder) WithConfig(cfg *Config) *Builder {
	b.config = cfg
	return b
}

// WithProvider sets the LLM provider.
func (b *Builder) WithProvider(provider llm.Provider) *Builder {
	b.provider = provider
	return b
}

// WithWorkingDir sets the working directory.
func (b *Builder) WithWorkingDir(dir string) *Builder {
	b.workingDir = dir
	return b
}

// WithEmitter sets the event emitter.
func (b *Builder) WithEmitter(emitter *events.EventEmitter) *Builder {
	b.emitter = emitter
	return b
}

// WithApprovalHandler sets the approval handler.
func (b *Builder) WithApprovalHandler(handler security.ApprovalHandler) *Builder {
	b.approvalHandler = handler
	return b
}

// buildExecutor creates an Executor with appropriate options based on configuration.
func (b *Builder) buildExecutor() *Executor {
	validator := security.NewValidator()
	opts := []ExecutorOption{
		WithValidator(validator),
	}

	if b.approvalHandler != nil {
		opts = append(opts,
			WithApprovalService(security.NewApprovalService(b.approvalHandler, b.emitter, validator)),
		)
	}

	if cfg := b.config; cfg != nil {
		if cfg.Timeout > 0 {
			opts = append(opts, WithTimeout(cfg.Timeout))
		}
		if cfg.CacheCommands {
			cache := NewCommandCache(5*time.Minute, 10*1024*1024) // 5m TTL, 10MB cap
			opts = append(opts, WithCache(cache))
		}
	}

	exec, err := NewExecutor(b.workingDir, opts...)
	if err != nil {
		// In builder pattern, we panic on invalid configuration
		// This should never happen with valid builder state
		panic("failed to create executor: " + err.Error())
	}
	return exec
}
