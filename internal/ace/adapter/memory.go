package adapter

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

const (
	defaultMemMaxSize    = 1000
	defaultMemWarnSize   = 900
	defaultMemGrowthRate = 0.2
)

// MemoryConfig configures memory management.
type MemoryConfig struct {
	MaxBullets     int
	RefinementAt   int
	PruneThreshold float64
}

// DefaultMemoryConfig returns default memory configuration.
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		MaxBullets:     defaultMemMaxSize,
		RefinementAt:   defaultMemWarnSize,
		PruneThreshold: defaultMemGrowthRate,
	}
}

// MemoryManager handles playbook memory management.
type MemoryManager struct {
	config MemoryConfig
}

// NewMemoryManager creates a new memory manager.
func NewMemoryManager(config MemoryConfig) *MemoryManager {
	return &MemoryManager{config: config}
}

// ShouldRefine determines if refinement should be triggered.
func (m *MemoryManager) ShouldRefine(bulletCount int) bool {
	return bulletCount >= m.config.RefinementAt
}

// Prune removes low-utility bullets from playbook.
func (m *MemoryManager) Prune(ctx context.Context, pb *playbook.Playbook) (int, error) {
	pruned, _, err := playbook.PruneLowUtility(ctx, pb, m.config.PruneThreshold)

	return pruned, err
}
