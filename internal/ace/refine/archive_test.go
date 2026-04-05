package refine

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
)

func TestArchiveReason_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		reason ArchiveReason
		want   string
	}{
		{"low utility", ReasonLowUtility, "low_utility"},
		{"merged", ReasonMerged, "merged"},
		{"manual", ReasonManual, "manual"},
		{"superseded", ReasonSuperseded, "superseded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if string(tt.reason) != tt.want {
				t.Errorf("expected %s, got %s", tt.want, string(tt.reason))
			}
		})
	}
}

func TestArchive_Archive(t *testing.T) {
	t.Parallel()

	archive := NewArchive()

	b, _ := bullet.New("Test content")
	metadata := map[string]string{"key": "value"}

	archive.Archive(b, ReasonLowUtility, metadata)

	if archive.Len() != 1 {
		t.Errorf("expected len 1, got %d", archive.Len())
	}

	archived, exists := archive.Get(b.ID)
	if !exists {
		t.Fatal("expected archived bullet to exist")
	}

	if archived.Bullet.Content != "Test content" {
		t.Errorf("expected content 'Test content', got '%s'", archived.Bullet.Content)
	}

	if archived.Reason != ReasonLowUtility {
		t.Errorf("expected reason %s, got %s", ReasonLowUtility, archived.Reason)
	}

	if archived.Metadata["key"] != "value" {
		t.Errorf("expected metadata key=value, got %s", archived.Metadata["key"])
	}

	if archived.RemovedAt.IsZero() {
		t.Error("expected non-zero RemovedAt timestamp")
	}
}

func TestArchive_Get(t *testing.T) {
	t.Parallel()

	archive := NewArchive()

	b, _ := bullet.New("Test content")
	archive.Archive(b, ReasonMerged, nil)

	// Get existing.
	archived, exists := archive.Get(b.ID)
	if !exists {
		t.Fatal("expected archived bullet to exist")
	}

	if archived.Bullet.ID != b.ID {
		t.Errorf("expected ID %s, got %s", b.ID, archived.Bullet.ID)
	}

	// Get non-existent.
	_, exists = archive.Get("non-existent")
	if exists {
		t.Error("expected non-existent bullet to not exist")
	}
}

func TestArchive_List(t *testing.T) {
	t.Parallel()

	archive := NewArchive()

	b1, _ := bullet.New("Content 1")
	b2, _ := bullet.New("Content 2")
	b3, _ := bullet.New("Content 3")

	archive.Archive(b1, ReasonLowUtility, nil)
	archive.Archive(b2, ReasonMerged, nil)
	archive.Archive(b3, ReasonLowUtility, nil)

	// List all.
	all := archive.List(nil)
	if len(all) != 3 {
		t.Errorf("expected 3 archived bullets, got %d", len(all))
	}

	// List with filter (only low utility).
	lowUtility := archive.List(func(ab *ArchivedBullet) bool {
		return ab.Reason == ReasonLowUtility
	})
	if len(lowUtility) != 2 {
		t.Errorf("expected 2 low utility bullets, got %d", len(lowUtility))
	}

	// List with filter (only merged).
	merged := archive.List(func(ab *ArchivedBullet) bool {
		return ab.Reason == ReasonMerged
	})
	if len(merged) != 1 {
		t.Errorf("expected 1 merged bullet, got %d", len(merged))
	}
}

func TestArchive_Stats(t *testing.T) {
	t.Parallel()

	archive := NewArchive()

	// Empty archive.
	stats := archive.Stats()
	if stats.TotalBullets != 0 {
		t.Errorf("expected 0 total bullets, got %d", stats.TotalBullets)
	}

	// Add bullets.
	b1, _ := bullet.New("Content 1")
	b2, _ := bullet.New("Content 2")
	b3, _ := bullet.New("Content 3")

	archive.Archive(b1, ReasonLowUtility, nil)
	time.Sleep(1 * time.Millisecond)
	archive.Archive(b2, ReasonMerged, nil)
	time.Sleep(1 * time.Millisecond)
	archive.Archive(b3, ReasonLowUtility, nil)

	stats = archive.Stats()
	if stats.TotalBullets != 3 {
		t.Errorf("expected 3 total bullets, got %d", stats.TotalBullets)
	}

	if stats.ByReason[ReasonLowUtility] != 2 {
		t.Errorf("expected 2 low utility, got %d", stats.ByReason[ReasonLowUtility])
	}

	if stats.ByReason[ReasonMerged] != 1 {
		t.Errorf("expected 1 merged, got %d", stats.ByReason[ReasonMerged])
	}

	if stats.OldestArchive.IsZero() {
		t.Error("expected non-zero oldest archive timestamp")
	}

	if stats.NewestArchive.IsZero() {
		t.Error("expected non-zero newest archive timestamp")
	}

	if !stats.NewestArchive.After(stats.OldestArchive) && !stats.NewestArchive.Equal(stats.OldestArchive) {
		t.Error("expected newest >= oldest")
	}
}

func TestArchive_Clear(t *testing.T) {
	t.Parallel()

	archive := NewArchive()

	b1, _ := bullet.New("Content 1")
	b2, _ := bullet.New("Content 2")

	archive.Archive(b1, ReasonLowUtility, nil)
	archive.Archive(b2, ReasonMerged, nil)

	if archive.Len() != 2 {
		t.Errorf("expected len 2 before clear, got %d", archive.Len())
	}

	archive.Clear()

	if archive.Len() != 0 {
		t.Errorf("expected len 0 after clear, got %d", archive.Len())
	}

	_, exists := archive.Get(b1.ID)
	if exists {
		t.Error("expected bullet to not exist after clear")
	}
}

func TestArchive_Clone(t *testing.T) {
	t.Parallel()

	archive := NewArchive()

	original, _ := bullet.New("Original content")
	original.IncrementHelpful()

	archive.Archive(original, ReasonLowUtility, nil)

	// Modify original after archiving.
	original.IncrementHelpful()
	original.Content = "Modified content"

	// Archived bullet should be unchanged.
	archived, _ := archive.Get(original.ID)
	if archived.Bullet.Content != "Original content" {
		t.Errorf("expected archived content 'Original content', got '%s'", archived.Bullet.Content)
	}

	if archived.Bullet.HelpfulCount != 1 {
		t.Errorf("expected archived helpful count 1, got %d", archived.Bullet.HelpfulCount)
	}
}

func TestArchive_MetadataPreservation(t *testing.T) {
	t.Parallel()

	archive := NewArchive()

	b, _ := bullet.New("Test content")
	metadata := map[string]string{
		"merged_into":   "bullet-123",
		"reason_detail": "too similar",
	}

	archive.Archive(b, ReasonMerged, metadata)

	archived, _ := archive.Get(b.ID)
	if len(archived.Metadata) != 2 {
		t.Errorf("expected 2 metadata entries, got %d", len(archived.Metadata))
	}

	if archived.Metadata["merged_into"] != "bullet-123" {
		t.Errorf("expected merged_into=bullet-123, got %s", archived.Metadata["merged_into"])
	}
}

func TestArchive_NilMetadata(t *testing.T) {
	t.Parallel()

	archive := NewArchive()

	b, _ := bullet.New("Test content")
	archive.Archive(b, ReasonManual, nil)

	archived, _ := archive.Get(b.ID)
	if archived.Metadata == nil {
		t.Error("expected metadata map to be initialized, got nil")
	}

	if len(archived.Metadata) != 0 {
		t.Errorf("expected empty metadata map, got %d entries", len(archived.Metadata))
	}
}

func TestArchive_Concurrency(t *testing.T) {
	t.Parallel()

	archive := NewArchive()

	const (
		goroutines          = 10
		bulletsPerGoroutine = 10
	)

	done := make(chan bool, goroutines)

	// Concurrent writes.
	for g := range goroutines {
		go func(_ int) {
			for range bulletsPerGoroutine {
				b, _ := bullet.New("Test content")
				archive.Archive(b, ReasonLowUtility, nil)
			}

			done <- true
		}(g)
	}

	// Wait for all goroutines.
	for range goroutines {
		<-done
	}

	expected := goroutines * bulletsPerGoroutine
	if archive.Len() != expected {
		t.Errorf("expected %d archived bullets, got %d", expected, archive.Len())
	}

	// Concurrent reads.
	stop := make(chan bool)
	errors := make(chan error, goroutines)

	for range goroutines {
		go func() {
			for {
				select {
				case <-stop:
					return
				default:
					_ = archive.List(nil)
					_ = archive.Stats()
				}
			}
		}()
	}

	// Let readers run briefly.
	time.Sleep(10 * time.Millisecond)
	close(stop)

	// Check for errors.
	select {
	case err := <-errors:
		t.Errorf("concurrent read error: %v", err)
	default:
		// No errors.
	}
}
