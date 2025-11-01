package refine

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

func TestDefaultGrowthThresholds(t *testing.T) {
	thresholds := DefaultGrowthThresholds()

	if thresholds.MaxBullets <= 0 {
		t.Error("expected positive MaxBullets")
	}
	if thresholds.MaxTokens <= 0 {
		t.Error("expected positive MaxTokens")
	}
	if thresholds.MinUtility <= 0 {
		t.Error("expected positive MinUtility")
	}
	if thresholds.CheckInterval <= 0 {
		t.Error("expected positive CheckInterval")
	}
}

func TestNewGrowthMonitor(t *testing.T) {
	pb := playbook.New(nil, nil)
	thresholds := DefaultGrowthThresholds()

	monitor := NewGrowthMonitor(pb, thresholds)

	if monitor.playbook != pb {
		t.Error("expected playbook to be set")
	}
	if monitor.thresholds.MaxBullets != thresholds.MaxBullets {
		t.Error("expected thresholds to be set")
	}
}

func TestGrowthMonitor_CheckGrowth(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	thresholds := GrowthThresholds{
		MaxBullets: 10,
		MaxTokens:  500,
		MinUtility: 0.5,
	}

	monitor := NewGrowthMonitor(pb, thresholds)

	// Initial check (empty playbook)
	metrics, needsRefine := monitor.CheckGrowth(ctx)

	if metrics.BulletCount != 0 {
		t.Errorf("expected 0 bullets, got %d", metrics.BulletCount)
	}
	if needsRefine {
		t.Error("expected no refinement needed for empty playbook")
	}

	// Add some bullets
	for i := 0; i < 5; i++ {
		b, _ := bullet.New("Test content")
		b.IncrementHelpful()
		pb.Add(ctx, b)
	}

	metrics, needsRefine = monitor.CheckGrowth(ctx)

	if metrics.BulletCount != 5 {
		t.Errorf("expected 5 bullets, got %d", metrics.BulletCount)
	}
	if metrics.EstimatedTokens <= 0 {
		t.Error("expected positive token estimate")
	}
	if needsRefine {
		t.Error("expected no refinement needed (below threshold)")
	}
}

func TestGrowthMonitor_ThresholdTriggers(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		thresholds    GrowthThresholds
		bulletCount   int
		shouldTrigger bool
	}{
		{
			name: "below all thresholds",
			thresholds: GrowthThresholds{
				MaxBullets: 100,
				MaxTokens:  5000,
				MinUtility: 0.1,
			},
			bulletCount:   50,
			shouldTrigger: false,
		},
		{
			name: "exceeds bullet threshold",
			thresholds: GrowthThresholds{
				MaxBullets: 10,
				MaxTokens:  10000,
				MinUtility: 0.1,
			},
			bulletCount:   15,
			shouldTrigger: true,
		},
		{
			name: "exceeds token threshold",
			thresholds: GrowthThresholds{
				MaxBullets: 1000,
				MaxTokens:  100, // 100 tokens = ~2 bullets at 50 tokens each
				MinUtility: 0.1,
			},
			bulletCount:   5,
			shouldTrigger: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh playbook for each test
			testPb := playbook.New(nil, nil)
			monitor := NewGrowthMonitor(testPb, tt.thresholds)

			// Add bullets
			for i := 0; i < tt.bulletCount; i++ {
				b, _ := bullet.New("Test content")
				b.IncrementHelpful()
				testPb.Add(ctx, b)
			}

			_, needsRefine := monitor.CheckGrowth(ctx)

			if needsRefine != tt.shouldTrigger {
				t.Errorf("expected shouldTrigger=%v, got %v", tt.shouldTrigger, needsRefine)
			}
		})
	}
}

func TestGrowthMonitor_GetMetrics(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	monitor := NewGrowthMonitor(pb, DefaultGrowthThresholds())

	// Add bullets
	for i := 0; i < 3; i++ {
		b, _ := bullet.New("Test content")
		pb.Add(ctx, b)
	}

	// First check to populate metrics
	monitor.CheckGrowth(ctx)

	// Get metrics without checking
	metrics := monitor.GetMetrics()

	if metrics.BulletCount != 3 {
		t.Errorf("expected 3 bullets, got %d", metrics.BulletCount)
	}
}

func TestGrowthMonitor_MarkRefinement(t *testing.T) {
	pb := playbook.New(nil, nil)
	monitor := NewGrowthMonitor(pb, DefaultGrowthThresholds())

	// Initially zero
	metrics := monitor.GetMetrics()
	if !metrics.LastRefinement.IsZero() {
		t.Error("expected zero LastRefinement initially")
	}

	// Mark refinement
	before := time.Now()
	time.Sleep(1 * time.Millisecond)
	monitor.MarkRefinement()
	time.Sleep(1 * time.Millisecond)
	after := time.Now()

	metrics = monitor.GetMetrics()
	if metrics.LastRefinement.IsZero() {
		t.Error("expected non-zero LastRefinement after marking")
	}
	if metrics.LastRefinement.Before(before) || metrics.LastRefinement.After(after) {
		t.Error("expected LastRefinement to be between before and after times")
	}
}

func TestGrowthMonitor_GrowthRate(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	monitor := NewGrowthMonitor(pb, DefaultGrowthThresholds())

	// Add bullets over time
	for i := 0; i < 5; i++ {
		b, _ := bullet.New("Test content")
		pb.Add(ctx, b)
		monitor.CheckGrowth(ctx)
		time.Sleep(1 * time.Millisecond)
	}

	metrics := monitor.GetMetrics()

	// Growth rate should be calculated (may be 0 for short time periods)
	if metrics.GrowthRate < 0 {
		t.Errorf("expected non-negative growth rate, got %f", metrics.GrowthRate)
	}
}

func TestGrowthMonitor_HistoryLimit(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	monitor := NewGrowthMonitor(pb, DefaultGrowthThresholds())

	// Add 150 data points (more than the 100 limit)
	for i := 0; i < 150; i++ {
		b, _ := bullet.New("Test content")
		pb.Add(ctx, b)
		monitor.CheckGrowth(ctx)
	}

	monitor.mu.RLock()
	histLen := len(monitor.bulletHistory)
	monitor.mu.RUnlock()

	if histLen > 100 {
		t.Errorf("expected history limited to 100, got %d", histLen)
	}
}

func TestGrowthMonitor_ShouldRefine(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	thresholds := GrowthThresholds{
		MaxBullets: 5,
	}

	monitor := NewGrowthMonitor(pb, thresholds)

	// Initially should not refine
	if monitor.ShouldRefine() {
		t.Error("expected ShouldRefine=false initially")
	}

	// Add bullets to exceed threshold
	for i := 0; i < 6; i++ {
		b, _ := bullet.New("Test content")
		pb.Add(ctx, b)
	}

	// Check to update metrics
	monitor.CheckGrowth(ctx)

	// Now should refine
	if !monitor.ShouldRefine() {
		t.Error("expected ShouldRefine=true after exceeding threshold")
	}
}

func TestGrowthMonitor_Concurrency(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	monitor := NewGrowthMonitor(pb, DefaultGrowthThresholds())

	// Add initial bullets
	for i := 0; i < 10; i++ {
		b, _ := bullet.New("Test content")
		pb.Add(ctx, b)
	}

	const goroutines = 10
	done := make(chan bool, goroutines)

	// Concurrent reads
	for g := 0; g < goroutines; g++ {
		go func() {
			for i := 0; i < 10; i++ {
				monitor.CheckGrowth(ctx)
				monitor.GetMetrics()
				monitor.ShouldRefine()
			}
			done <- true
		}()
	}

	// Wait for completion
	for g := 0; g < goroutines; g++ {
		<-done
	}

	// Should not panic or cause race conditions
}
