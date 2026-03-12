// Package delta provides delta-based change tracking and application.
package delta

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

var (
	ErrBulletNotFound = errors.New("bullet  not found")
	ErrContentFieldIsRequiredForOpupdatecontent = errors.New("content field is required for OpUpdateContent")
	ErrTagKeyAndTagValueFields = errors.New("tag_key and tag_value fields are required for OpAddTag")
	ErrTagKeyFieldIsRequiredFor = errors.New("tag_key field is required for OpRemoveTag")
	ErrEmbeddingFieldIsRequiredForOpupdateembedding = errors.New("embedding field is required for OpUpdateEmbedding")
	ErrUnknownOperation = errors.New("unknown operation")
)

// Applier applies deltas to bullets in a playbook.
type Applier struct {
	playbook *playbook.Playbook
	history  *History
	logger   *slog.Logger
}

// ApplyResult contains the result of applying a delta.
type ApplyResult struct {
	Success   bool
	DeltaID   string
	BulletID  string
	OldValue  any
	NewValue  any
	Error     error
	AppliedAt time.Time
}

// NewApplier creates a new delta applier.
func NewApplier(pb *playbook.Playbook) *Applier {
	return &Applier{
		playbook: pb,
		history:  NewHistory(),
		logger:   slog.Default(),
	}
}

// Apply applies a single delta to the playbook.
func (a *Applier) Apply(ctx context.Context, delta Delta) (*ApplyResult, error) {
	a.logger.DebugContext(ctx, "Applying delta operation",
		"delta_id", delta.ID,
		"bullet_id", delta.BulletID,
		"operation", delta.Operation,
		"source", delta.Metadata.Source,
		"reason", delta.Metadata.Reason)

	// Get bullet from playbook.
	b, exists := a.playbook.Get(delta.BulletID)
	if !exists {
err := fmt.Errorf("bullet %s not found: %w", delta.BulletID, ErrBulletNotFound)
		a.logger.WarnContext(ctx, "Delta apply failed: bullet not found",
			"delta_id", delta.ID,
			"bullet_id", delta.BulletID)

		return &ApplyResult{
			Success:   false,
			DeltaID:   delta.ID,
			BulletID:  delta.BulletID,
			Error:     err,
			AppliedAt: time.Now(),
		}, err
	}

	// Create clone for modification (copy-on-write).
	modified := b.Clone()

	// Apply delta based on operation.
	oldValue, newValue, err := applyOperation(modified, delta)
	if err != nil {
		a.logger.WarnContext(ctx, "Delta operation failed",
			"delta_id", delta.ID,
			"operation", delta.Operation,
			"error", err)

		return &ApplyResult{
			Success:   false,
			DeltaID:   delta.ID,
			BulletID:  delta.BulletID,
			Error:     err,
			AppliedAt: time.Now(),
		}, err
	}

	a.logger.DebugContext(ctx, "Delta operation applied",
		"operation", delta.Operation,
		"old_value", oldValue,
		"new_value", newValue)

	// Update bullet in playbook.
	err = a.playbook.Update(ctx, modified)
	if err != nil {
		a.logger.WarnContext(ctx, "Failed to update bullet in playbook",
			"delta_id", delta.ID,
			"bullet_id", delta.BulletID,
			"error", err)

		return &ApplyResult{
			Success:   false,
			DeltaID:   delta.ID,
			BulletID:  delta.BulletID,
			Error:     err,
			AppliedAt: time.Now(),
		}, err
	}

	// Record delta in history.
	a.history.Record(delta)

	a.logger.DebugContext(ctx, "Delta applied successfully",
		"delta_id", delta.ID,
		"bullet_id", delta.BulletID,
		"operation", delta.Operation)

	return &ApplyResult{
		Success:   true,
		DeltaID:   delta.ID,
		BulletID:  delta.BulletID,
		OldValue:  oldValue,
		NewValue:  newValue,
		AppliedAt: time.Now(),
	}, nil
}

// GetHistory returns the delta history.
func (a *Applier) GetHistory() *History {
	return a.history
}

// applyOperation applies a delta operation to a bullet (copy-on-write).
func applyOperation(b *bullet.Bullet, delta Delta) (oldValue, newValue any, err error) {
	switch delta.Operation {
	case OpUpdateContent:
		if delta.Fields.Content == nil {
			return nil, nil, ErrContentFieldIsRequiredForOpupdatecontent
		}

		oldValue = b.Content
		b.Content = *delta.Fields.Content
		newValue = b.Content

	case OpIncrementHelpful:
		oldValue = b.HelpfulCount
		b.IncrementHelpful()
		newValue = b.HelpfulCount

	case OpIncrementHarmful:
		oldValue = b.HarmfulCount
		b.IncrementHarmful()
		newValue = b.HarmfulCount

	case OpAddTag:
		if delta.Fields.TagKey == nil || delta.Fields.TagValue == nil {
			return nil, nil, ErrTagKeyAndTagValueFields
		}

		if b.Tags == nil {
			b.Tags = make(map[string]string)
		}

		oldValue = b.Tags[*delta.Fields.TagKey]
		b.Tags[*delta.Fields.TagKey] = *delta.Fields.TagValue
		newValue = *delta.Fields.TagValue

	case OpRemoveTag:
		if delta.Fields.TagKey == nil {
			return nil, nil, ErrTagKeyFieldIsRequiredFor
		}

		if b.Tags == nil {
			oldValue = nil
		} else {
			oldValue = b.Tags[*delta.Fields.TagKey]
			delete(b.Tags, *delta.Fields.TagKey)
		}

		newValue = nil

	case OpUpdateEmbedding:
		if delta.Fields.Embedding == nil {
			return nil, nil, ErrEmbeddingFieldIsRequiredForOpupdateembedding
		}

		oldValue = b.Embedding
		b.Embedding = delta.Fields.Embedding
		newValue = b.Embedding

	default:
return nil, nil, fmt.Errorf("unknown operation: %s: %w", delta.Operation, ErrUnknownOperation)
	}

	// Update timestamp.
	b.UpdatedAt = time.Now()

	return oldValue, newValue, nil
}
