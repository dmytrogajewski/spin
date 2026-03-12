package adapter

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
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

// CalculateUtility computes utility score for a bullet.
func (m *MemoryManager) CalculateUtility(b *bullet.Bullet) float64 {
	helpful := float64(b.HelpfulCount)
	harmful := float64(b.HarmfulCount)

	return (helpful - harmful) / (helpful + harmful + 1)
}

// Prune removes low-utility bullets from playbook.
func (m *MemoryManager) Prune(ctx context.Context, pb *playbook.Playbook) (int, error) {
	toPrune := m.findLowUtilityBullets(pb.List(nil))

	return m.deleteBullets(ctx, pb, toPrune)
}

// findLowUtilityBullets returns IDs of bullets below the utility threshold.
func (m *MemoryManager) findLowUtilityBullets(bullets []*bullet.Bullet) []string {
	var ids []string

	for _, b := range bullets {
		if m.CalculateUtility(b) < m.config.PruneThreshold {
			ids = append(ids, b.ID)
		}
	}

	return ids
}

// deleteBullets removes bullets by ID from the playbook.
func (m *MemoryManager) deleteBullets(ctx context.Context, pb *playbook.Playbook, ids []string) (int, error) {
	deleted := 0

	for _, id := range ids {
		err := pb.Delete(ctx, id)
		if err != nil {
			return deleted, err
		}

		deleted++
	}

	return deleted, nil
}
