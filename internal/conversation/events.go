package conversation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// attachJSONLEventLogger sets up a JSONL event logger for the given session.
// Events are written to {sessionDir}/{sessionID}/events.jsonl in append mode.
func (b *Builder) attachJSONLEventLogger(ctx context.Context, sessionID string) {
	base := b.cfg.Agent.SessionDir
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, ".spin", "sessions")
		}
	} else if strings.HasPrefix(base, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, base[1:])
		}
	}
	dir := filepath.Join(base, sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		if b.logger != nil {
			b.logger.Warn("event logger mkdir failed", "dir", dir, "err", err)
		}
		return
	}
	logPath := filepath.Join(dir, "events.jsonl")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		if b.logger != nil {
			b.logger.Warn("event logger open failed", "path", logPath, "err", err)
		}
		return
	}

	subID, ch, subErr := b.emitter.Subscribe()
	if subErr != nil {
		_ = f.Close()
		if b.logger != nil {
			b.logger.Warn("event subscribe failed", "err", subErr)
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
				_ = enc.Encode(record) // best-effort
			case <-ctx.Done():
				return
			}
		}
	}()
}
