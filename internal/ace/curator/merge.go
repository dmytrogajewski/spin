package curator

import (
	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
)

// MergeRequest contains insights to curate into playbook.
type MergeRequest struct {
	Insights            []*reflector.Insight
	SimilarityThreshold float64
}

// MergeResult contains the result of a merge operation.
type MergeResult struct {
	Added        int               // New bullets added
	Skipped      int               // Duplicates skipped
	Updated      int               // Existing bullets updated
	Duplicates   []string          // IDs of duplicate bullets
	AddedBullets []*bullet.Bullet  // Bullets that were added
	Refined      bool              // Was refinement triggered?
	Refinement   *RefinementResult // Refinement stats (if refined)
}

// BatchMergeRequest contains multiple merge requests for parallel processing.
type BatchMergeRequest struct {
	Requests   []MergeRequest
	MaxWorkers int // 0 = runtime.NumCPU()
}

// BatchMergeResult contains results from batch processing.
type BatchMergeResult struct {
	Results []MergeResult
	Errors  []error // per-request errors (nil if successful)
}
