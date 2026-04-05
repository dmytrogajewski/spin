// Package ace provides the ACE middleware for progressive retrieval in the agent loop.
package ace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/openai/openai-go"

	acecore "github.com/dmytrogajewski/spin/internal/ace"
	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
)

const maxConceptExtract = 5

// Middleware encapsulates all ACE (Agentic Context Engineering) concerns:
// bullet retrieval, feedback processing, bullet generation, and event emission.
//
// It operates as an optional layer — if aceService is nil, all methods are no-ops.
type Middleware struct {
	aceService *acecore.Service
	aceConfig  *acecore.Config
	emitter    *events.EventEmitter
	logger     *slog.Logger
}

// New creates a new ACE Middleware. If aceService is nil,
// all methods become safe no-ops.
func New(aceService *acecore.Service, aceConfig *acecore.Config, emitter *events.EventEmitter, logger *slog.Logger) *Middleware {
	return &Middleware{
		aceService: aceService,
		aceConfig:  aceConfig,
		emitter:    emitter,
		logger:     logger,
	}
}

// ProcessFeedback parses and applies ACE feedback from an LLM response.
func (am *Middleware) ProcessFeedback(ctx context.Context, response *openai.ChatCompletion, bullets []*bullet.Bullet) {
	if am.aceService == nil || len(bullets) == 0 {
		return
	}

	responseContent := response.Choices[0].Message.Content

	fb, parseErr := am.aceService.ParseFeedback(responseContent)
	if parseErr != nil {
		if !errors.Is(parseErr, acecore.ErrDisabled) {
			am.logger.WarnContext(ctx, "ACE feedback parsing failed", "error", parseErr)
		}

		return
	}

	if fb == nil {
		return
	}

	updateErr := am.aceService.UpdateBullets(ctx, bullets, fb)
	if updateErr != nil {
		am.logger.WarnContext(ctx, "ACE bullet update failed", "error", updateErr)
	} else {
		am.logger.DebugContext(ctx, "ACE updated bullets",
			"helpful_count", len(fb.HelpfulBullets),
			"harmful_count", len(fb.HarmfulBullets))
	}
}

// AfterExecution implements agent.Middleware. Generates ACE bullets from execution trajectory.
func (am *Middleware) AfterExecution(ctx context.Context, trajCtx *trajectory.Context, resp *agent.Response) {
	if am.aceService == nil {
		am.logger.DebugContext(ctx, "Bullet generation skipped", "ace_service_nil", true)

		return
	}

	am.logger.InfoContext(ctx, "Starting bullet generation from execution", "success", resp.Success)

	traj := trajCtx.ToTrajectory()
	traj.Success = resp.Success
	am.logger.DebugContext(ctx, "Execution trajectory built", "steps", len(traj.Steps), "success", traj.Success)

	learnedBullets, err := am.generateBulletsFromTrajectory(ctx, traj)
	if err != nil {
		am.logger.WarnContext(ctx, "ACE bullet generation failed", "error", err)

		return
	}

	am.logger.InfoContext(ctx, "Successfully generated bullets from execution", "count", len(learnedBullets))

	if len(learnedBullets) == 0 {
		am.logger.DebugContext(ctx, "No bullets to display (empty result)")
	}

	am.emitLearningEvent(resp, learnedBullets)
}

// EmitRetrievalEvent emits an ACE retrieval event with calculated metrics.
func (am *Middleware) EmitRetrievalEvent(
	trajCtx *trajectory.Context,
	trigger trajectory.TriggerType,
	query string,
	bullets []*bullet.Bullet,
	turn int,
) {
	total := trajCtx.CacheHits + trajCtx.CacheMisses

	hitRate := 0.0
	if total > 0 {
		hitRate = float64(trajCtx.CacheHits) / float64(total)
	}

	bulletsNew := trajCtx.CacheMisses

	bulletData := make([]events.BulletData, len(bullets))
	for i, b := range bullets {
		category := ""
		if b.Tags != nil {
			category = b.Tags["category"]
		}

		bulletData[i] = events.BulletData{
			Content:  b.Content,
			Category: category,
		}
	}

	am.emitter.Emit(events.Event{
		Type: events.EventACERetrieval,
		Data: events.ACERetrievalData{
			Turn:             turn,
			Trigger:          string(trigger),
			Query:            query,
			BulletsRetrieved: len(bullets),
			BulletsNew:       bulletsNew,
			CacheSize:        len(trajCtx.BulletCache),
			CacheHitRate:     hitRate,
			Bullets:          bulletData,
		},
	})
}

// BeforeTurn implements agent.Middleware. Performs progressive ACE bullet retrieval.
// Retrieved bullets are stored in trajCtx and accessible via trajCtx.GetActiveBullets().
func (am *Middleware) BeforeTurn(ctx context.Context, trajCtx *trajectory.Context, turn int) {
	if !am.isProgressiveRetrievalEnabled(trajCtx) {
		return
	}

	shouldRetrieve, trigger := am.shouldRetrieve(trajCtx)
	if shouldRetrieve {
		am.retrieveAndRecordBullets(ctx, trajCtx, trigger, turn)
	}
}

func (am *Middleware) isProgressiveRetrievalEnabled(trajCtx *trajectory.Context) bool {
	return am.aceService != nil && trajCtx != nil && am.aceConfig != nil && am.aceConfig.Retrieval.ProgressiveContext.Enabled
}

func (am *Middleware) shouldRetrieve(trajCtx *trajectory.Context) (bool, trajectory.TriggerType) {
	if am.aceConfig == nil || !am.aceConfig.Retrieval.ProgressiveContext.Enabled {
		return false, ""
	}

	if trajCtx.CurrentTurn == 0 {
		return true, trajectory.TriggerInitial
	}

	cfg := am.aceConfig.Retrieval.ProgressiveContext

	if trajCtx.HasRecentError(cfg.ErrorLookback) {
		return true, trajectory.TriggerError
	}

	tools := trajCtx.GetRecentTools(cfg.ToolChangeLookback)
	if len(tools) > 1 {
		return true, trajectory.TriggerToolChange
	}

	if trajCtx.CurrentTurn-trajCtx.LastRetrievalTurn >= cfg.CacheTTL {
		return true, trajectory.TriggerInterval
	}

	return false, ""
}

func (am *Middleware) buildQueryFromContext(trajCtx *trajectory.Context, trigger trajectory.TriggerType) string {
	parts := []string{trajCtx.Query}

	if am.aceConfig == nil {
		return trajCtx.Query
	}

	switch trigger {
	case trajectory.TriggerInitial:
		return trajCtx.Query

	case trajectory.TriggerError:
		errorPatterns := trajectory.ExtractErrorPatterns(trajCtx.Steps, am.aceConfig.Retrieval.ProgressiveContext.ErrorLookback)
		parts = append(parts, errorPatterns...)

	case trajectory.TriggerToolChange:
		tools := trajCtx.GetRecentTools(am.aceConfig.Retrieval.ProgressiveContext.ToolChangeLookback)
		parts = append(parts, tools...)

	case trajectory.TriggerInterval:
		concepts := trajectory.ExtractConcepts(trajCtx.Steps, maxConceptExtract)
		parts = append(parts, concepts...)
	}

	return strings.Join(parts, " ")
}

func (am *Middleware) retrieveAndRecordBullets(ctx context.Context, trajCtx *trajectory.Context, trigger trajectory.TriggerType, turn int) {
	query := am.buildQueryFromContext(trajCtx, trigger)
	am.logger.DebugContext(ctx, "Progressive retrieval triggered", "trigger", trigger, "query", query, "turn", turn+1)

	retrievedBullets, err := am.aceService.Retrieve(ctx, query)
	if err != nil {
		am.logger.WarnContext(ctx, "ACE retrieval failed", "error", err, "turn", turn+1)

		return
	}

	event := trajectory.RetrievalEvent{
		Turn:         turn,
		Trigger:      trigger,
		Query:        query,
		BulletsAdded: extractBulletIDs(retrievedBullets),
		Timestamp:    time.Now(),
	}
	trajCtx.RecordRetrieval(event, retrievedBullets)

	am.logger.InfoContext(ctx, "Retrieved bullets",
		"count", len(retrievedBullets), "trigger", trigger,
		"cached", len(trajCtx.BulletCache), "hits", trajCtx.CacheHits, "misses", trajCtx.CacheMisses)

	if am.aceConfig.Retrieval.ProgressiveContext.EmitACEEvents {
		am.EmitRetrievalEvent(trajCtx, trigger, query, retrievedBullets, turn)
	}
}

func (am *Middleware) generateBulletsFromTrajectory(ctx context.Context, traj *generator.Trajectory) ([]*bullet.Bullet, error) {
	if am.aceService.Config().Generation.AutoReflect {
		return am.aceService.GenerateBulletsWithReflectionFromTrajectory(ctx, traj)
	}

	var sb strings.Builder
	sb.WriteString("Task: ")
	sb.WriteString(traj.Query)
	sb.WriteString("\n\nExecution Steps:\n")

	for _, step := range traj.Steps {
		fmt.Fprintf(&sb, "- [%s] %s\n", step.Type, step.Content)
	}

	sb.WriteString("\nResult: ")
	sb.WriteString(traj.Output)

	return am.aceService.GenerateBullets(ctx, sb.String(), "trajectory")
}

func (am *Middleware) emitLearningEvent(resp *agent.Response, learnedBullets []*bullet.Bullet) {
	if len(learnedBullets) == 0 || am.aceConfig == nil || !am.aceConfig.Retrieval.ProgressiveContext.EmitACEEvents {
		return
	}

	bulletData := make([]events.BulletData, len(learnedBullets))
	for i, b := range learnedBullets {
		bulletData[i] = events.BulletData{
			Content: b.Content,
		}
	}

	am.emitter.Emit(events.Event{
		Type:      events.EventACELearned,
		Timestamp: time.Now(),
		Data: events.ACELearningData{
			Success: resp.Success,
			Bullets: bulletData,
		},
	})
}

func extractBulletIDs(bullets []*bullet.Bullet) []string {
	ids := make([]string, len(bullets))
	for i, b := range bullets {
		ids[i] = b.ID
	}

	return ids
}
