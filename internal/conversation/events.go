package conversation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
)

// EventTransformer transforms internal events to protocol-specific formats.
// This allows protocols like ACP to intercept and transform events without
// modifying the core Conversation implementation.
//
// Usage:
//
//	transformer := &EventTransformer{conn: acpConn, sessionID: sid}
//	conversation.SetEventTransformer(transformer)
//
// The transformer receives all events emitted during conversation execution
// and can transform them to protocol-specific notifications.
type EventTransformer interface {
	// Transform processes an event and returns whether it was handled.
	// If handled is true, the transformer took responsibility for the event
	// (e.g., sent a protocol-specific notification).
	// If handled is false, the event should still be available via Stream().
	Transform(ctx context.Context, event events.Event) (handled bool)

	// Close releases any resources held by the transformer.
	Close() error
}

// attachJSONLEventLogger sets up a JSONL event logger for the given session.
// Events are written to {sessionDir}/{sessionID}/events.jsonl in append mode.
func (b *Builder) attachJSONLEventLogger(ctx context.Context, sessionID string) {
	base := b.cfg.Agent.SessionDir
	if base == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			base = filepath.Join(home, ".spin", "sessions")
		}
	} else if strings.HasPrefix(base, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			base = filepath.Join(home, base[1:])
		}
	}

	dir := filepath.Join(base, sessionID)
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		if b.logger != nil {
			b.logger.WarnContext(ctx, "event logger mkdir failed", "dir", dir, "err", err)
		}

		return
	}

	logPath := filepath.Join(dir, "events.jsonl")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		if b.logger != nil {
			b.logger.WarnContext(ctx, "event logger open failed", "path", logPath, "err", err)
		}

		return
	}

	subID, ch, subErr := b.emitter.Subscribe()
	if subErr != nil {
		_ = f.Close()

		if b.logger != nil {
			b.logger.WarnContext(ctx, "event subscribe failed", "err", subErr)
		}

		return
	}

	go func() {
		defer func() {
			b.emitter.Unsubscribe(subID)

			_ = f.Close()
		}()

		enc := json.NewEncoder(f)

		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return
				}

				record := map[string]any{
					"session_id": sessionID,
					"timestamp":  ev.Timestamp.Format(time.RFC3339Nano),
					"type":       ev.Type.String(),
					"data":       ev.Data,
				}
				_ = enc.Encode(record) // best-effort.
			case <-ctx.Done():
				return
			}
		}
	}()
}
