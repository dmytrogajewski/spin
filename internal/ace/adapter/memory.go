package adapter

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

// MemoryConfig configures memory management
type MemoryConfig struct {
	MaxBullets     int     // Maximum bullets before hard limit
	RefinementAt   int     // Trigger refinement at this count
	PruneThreshold float64 // Remove bullets with utility below this
}

// DefaultMemoryConfig returns default memory configuration
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		MaxBullets:     1000,
		RefinementAt:   900,
		PruneThreshold: 0.2,
	}
}

// MemoryManager handles playbook memory management
type MemoryManager struct {
	config MemoryConfig
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(config MemoryConfig) *MemoryManager {
	return &MemoryManager{
		config: config,
	}
}

// ShouldRefine determines if refinement should be triggered
func (m *MemoryManager) ShouldRefine(bulletCount int) bool {
	return bulletCount >= m.config.RefinementAt
}

// CalculateUtility computes utility score for a bullet
func (m *MemoryManager) CalculateUtility(b *bullet.Bullet) float64 {
	helpful := float64(b.HelpfulCount)
	harmful := float64(b.HarmfulCount)

	// Utility = (helpful - harmful) / (helpful + harmful + 1)
	// +1 prevents division by zero and penalizes bullets with no feedback
	return (helpful - harmful) / (helpful + harmful + 1)
}

// Prune removes low-utility bullets from playbook
func (m *MemoryManager) Prune(ctx context.Context, pb *playbook.Playbook) (int, error) {
	// Get all bullets
	allBullets := pb.List(nil)

	pruned := 0

	// Identify and remove low-utility bullets
	for _, b := range allBullets {
		utility := m.CalculateUtility(b)

		if utility < m.config.PruneThreshold {
			if err := pb.Delete(ctx, b.ID); err != nil {
				return pruned, err
			}
			pruned++
		}
	}

	return pruned, nil
}
